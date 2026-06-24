package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"golang.org/x/sync/errgroup"
)

// state описывает текущее состояние Websocket транспорта.
type state uint8

const (
	// stateDisconnected означает отсутствие активного соединения.
	stateDisconnected state = iota

	// stateConnected означает успешное установление WebSocket соединения.
	stateConnected
)

// Websocket реализует WebSocket транспорт.
//
// Transport отвечает только за установление соединения,
// передачу сообщений и контроль жизненного цикла соединения.
//
// Пакет не знает о формате сообщений и не содержит логики протокола.
// Полученные сообщения передаются через Events.Message.
//
// Websocket поддерживает:
//   - автоматическое переподключение;
//   - heartbeat-проверку соединения;
//   - отправку сообщений;
//   - события жизненного цикла.
//
// Один экземпляр Websocket предназначен для одного соединения.
type Websocket struct {
	// configs
	url         string
	headersFunc func() http.Header // opts: WithHeaderFunc
	readLimit   int64              // opts: WithReadLimit

	// heartbeat config
	heartbeatFunc     HeartbeatFunc // SetHeartbeatFunc
	heartbeatInterval time.Duration // opts: WithHeartbeatInterval
	heartbeatTimeout  time.Duration // opts: WithHeartbeatTimeout

	// transport
	conn   *websocket.Conn
	mu     sync.RWMutex
	stopCh chan error

	// lifecycle
	state atomic.Value

	// events
	Events Events

	// logging
	logger       *slog.Logger // opts: WithLogger
	debugEnabled bool
}

// New создаёт новый Websocket транспорт.
//
// Соединение устанавливается только после вызова Run.
//
// Опции позволяют настроить заголовки,
// heartbeat, лимиты чтения и логирование.
func New(url string, opts ...Option) *Websocket {
	ws := &Websocket{
		url:               url,
		readLimit:         10 * 1024 * 1024, // 10MB
		heartbeatInterval: 30 * time.Second,
		heartbeatTimeout:  10 * time.Second,
	}

	for _, o := range opts {
		o(ws)
	}
	return ws
}

// HeartbeatFunc выполняет проверку активности WebSocket соединения.
type HeartbeatFunc func(ctx context.Context, ws *Websocket) error

// SetHeartbeatFunc установка функции для поддержания соединения с websocket сервером
func (ws *Websocket) SetHeartbeatFunc(fn HeartbeatFunc) {
	ws.heartbeatFunc = fn
}

// IsConnected возвращает true, если WebSocket соединение активно.
//
// Метод безопасен для вызова из нескольких goroutine.
func (ws *Websocket) IsConnected() bool {
	v := ws.state.Load()
	if v == nil {
		return false
	}
	return v.(state) == stateConnected
}

// Run запускает жизненный цикл WebSocket соединения.
//
// Метод блокируется до остановки транспорта или отмены context.
//
// При потере соединения выполняется автоматическое переподключение
// с увеличением интервала ожидания.
//
// Возвращает:
//   - ошибку остановки через Stop;
//   - ошибку context при завершении;
//
// Ошибки соединения передаются через Events.Disconnect,
// а критические ошибки остановки возвращаются из Run.
func (ws *Websocket) Run(ctx context.Context) error {

	ws.mu.Lock()
	ws.stopCh = make(chan error, 1)
	ws.mu.Unlock()

	defer func() {
		ws.mu.Lock()
		ws.stopCh = nil
		ws.mu.Unlock()
	}()

	ws.state.Store(stateDisconnected)

	backoff := time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		started := time.Now()

		if ws.debugEnabled {
			ws.logger.Debug(ws.t("connecting"), "url", ws.url)
		}

		err := ws.runConnection(ctx)

		if ws.debugEnabled {
			if err != nil && !errors.Is(err, context.Canceled) {
				ws.logger.Debug(ws.t("disconnected"), "err", err)
			} else {
				ws.logger.Debug(ws.t("disconnected"))
			}
		}

		ws.state.Store(stateDisconnected)

		if errors.Is(err, context.Canceled) {
			return nil
		}

		ws.Events.Disconnect.Emit(ctx, err)

		if time.Since(started) > time.Minute {
			backoff = time.Second
		}

		select {

		case err := <-ws.stopCh:
			return err

		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return ctx.Err()

		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

// Stop останавливает работу транспорта.
//
// Текущее WebSocket соединение будет закрыто,
// после чего Run завершится с переданной ошибкой.
//
// Если транспорт уже остановлен, вызов не имеет эффекта.
func (ws *Websocket) Stop(err error) {
	ws.mu.Lock()
	stopCh := ws.stopCh
	conn := ws.conn
	ws.mu.Unlock()

	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}

	select {
	case stopCh <- err:
	default:
	}
}

// runConnection устанавливает одно WebSocket соединение
// и запускает связанные с ним goroutine.
//
// Внутри запускаются:
//   - цикл чтения сообщений;
//   - heartbeat.
//
// При завершении любого из циклов соединение считается потерянным.
func (ws *Websocket) runConnection(ctx context.Context) error {
	headers := http.Header{}
	if ws.headersFunc != nil {
		headers = ws.headersFunc()
	}

	conn, _, err := websocket.Dial(ctx, ws.url, &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		return wrapError(OpDial, err)
	}

	conn.SetReadLimit(ws.readLimit)

	ws.mu.Lock()
	ws.conn = conn
	ws.mu.Unlock()

	defer func() {

		ws.mu.Lock()
		ws.conn = nil
		ws.mu.Unlock()

		_ = conn.Close(websocket.StatusNormalClosure, "reconnect")
	}()

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, connCtx := errgroup.WithContext(connCtx)

	g.Go(func() error {
		return ws.readLoop(connCtx)
	})

	g.Go(func() error {
		return ws.heartbeatLoop(connCtx)
	})

	ws.state.Store(stateConnected)

	if ws.debugEnabled {
		ws.logger.Debug(ws.t("connected"), "url", ws.url)
	}

	ws.Events.Connect.Emit(ctx)

	return g.Wait()
}

// readLoop постоянно читает сообщения из WebSocket.
//
// Полученные сообщения передаются через Events.Message.
//
// Обработчики Message запускаются асинхронно,
// чтобы не блокировать единственный цикл чтения WebSocket.
//
// Ошибка чтения приводит к завершению текущего соединения.
func (ws *Websocket) readLoop(ctx context.Context) error {

	for {

		var msg json.RawMessage

		err := wsjson.Read(ctx, ws.conn, &msg)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return wrapError(OpRead, err)
		}

		if ws.debugEnabled {
			ws.logger.Debug(ws.t("read"), "msg", string(msg))
		}

		// отправляем в горутину, что бы не блокировать чтение
		// внутри обработчика события могут быть другие вызовы которые требуют чтения
		go ws.Events.Message.Emit(ctx, msg)
	}
}

// heartbeatLoop выполняет периодическую проверку соединения.
//
// Использует пользовательскую HeartbeatFunc,
// если она задана.
//
// При отсутствии ответа в течение heartbeatTimeout
// соединение считается потерянным.
func (ws *Websocket) heartbeatLoop(ctx context.Context) error {

	ticker := time.NewTicker(ws.heartbeatInterval)
	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:

			pingCtx, cancel := context.WithTimeout(ctx, ws.heartbeatTimeout)
			var err error
			if ws.debugEnabled {
				ws.logger.Debug("heartbeat")
			}
			if ws.heartbeatFunc != nil {
				err = ws.heartbeatFunc(pingCtx, ws)
			} else {
				err = ws.conn.Ping(pingCtx)
			}
			cancel()

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return wrapError(OpHeartbeat, ErrHeartbeatTimeout)
				}
				return wrapError(OpHeartbeat, err)
			}
		}
	}
}

// Send отправляет сообщение через WebSocket.
//
// Возвращает transport.Error с OpWrite при ошибке отправки.
//
// Если соединение отсутствует,
// возвращается ErrClosed.
//
// Метод безопасен для вызова из нескольких goroutine.
func (ws *Websocket) Send(ctx context.Context, v any) error {

	if !ws.IsConnected() {
		return wrapError(OpWrite, ErrClosed)
	}

	ws.mu.RLock()
	conn := ws.conn
	ws.mu.RUnlock()

	if conn == nil {
		return wrapError(OpWrite, ErrClosed)
	}

	if ws.debugEnabled {
		b, _ := json.Marshal(v)
		ws.logger.Debug(ws.t("send"), "msg", string(b))
	}

	if err := wsjson.Write(ctx, conn, v); err != nil {
		return wrapError(OpWrite, err)
	}

	return nil
}

// t возвращает цветной префикс для debug логов.
func (ws *Websocket) t(key string) string {
	return fmt.Sprintf("%swebsocket:%s%s", "\033[32m", key, "\033[0m")
}
