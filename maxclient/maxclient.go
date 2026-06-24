package maxclient

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/szonov/max/protocol"
	"github.com/szonov/max/rpc"
	"github.com/szonov/max/statestore"
	"github.com/szonov/max/transport"
)

// Map используется для сокращения записи payload в RPC вызовах.
type Map map[string]any

// Client представляет минимальную реализацию MAX клиента.
//
// Клиент отвечает за:
//
//   - установку и поддержание WebSocket соединения;
//   - выполнение Hello;
//   - авторизацию;
//   - хранение сессии;
//   - выполнение RPC вызовов;
//   - получение событий MAX протокола.
//
// Клиент не интерпретирует содержимое сообщений,
// контактов, чатов и других объектов MAX.
type Client struct {

	// Транспортный и RPC слои.
	ws  *transport.Websocket
	rpc *rpc.Client

	// Состояние текущей сессии.
	session *statestore.State[Session]

	// Последний успешный ответ Hello.
	//
	// Значение доступно через HelloResponse().
	helloMu sync.RWMutex
	hello   json.RawMessage

	// Параметры авторизации через токен.
	//
	// Значения используются как payload запроса
	// LoginViaToken. Поле token добавляется автоматически.
	loginViaTokenPayload Map

	// helloUserAgent содержит объект userAgent,
	// который будет отправлен серверу в запросе Hello.
	helloUserAgent Map

	// Функция получения пароля для аккаунтов
	// с включённой двухфакторной авторизацией.
	passwordFunc PasswordFunc

	// События клиента.
	Events Events
}

// New создаёт новый MAX клиент.
//
// Если транспорт не задан через WithTransport,
// автоматически создаётся транспорт по умолчанию.
//
// Клиент также создаёт RPC слой поверх транспорта
// и инициализирует хранение сессии.
//
// После создания экземпляра методы Websocket()
// и RPC() всегда возвращают корректные объекты.
//
// Для запуска клиента используйте Start.
func New(opts ...Option) *Client {

	c := &Client{
		session:              statestore.NewState[Session](nil),
		loginViaTokenPayload: DefaultLoginViaTokenPayload(),
		helloUserAgent:       DefaultHelloUserAgent(),
	}

	for _, o := range opts {
		o(c)
	}

	if c.ws == nil {
		WithTransport(DefaultTransport())(c)
	}

	c.ws.Events.Connect.Subscribe(func(ctx context.Context) {
		go c.bootstrap(ctx)
	})

	c.rpc.Events.Error.Subscribe(c.Events.Error.Emit)
	c.rpc.Events.Message.Subscribe(c.Events.Message.Emit)

	return c
}

// Start запускает MAX клиент.
//
// Метод загружает сохранённую сессию,
// при необходимости автоматически создаёт DeviceID
// и запускает WebSocket транспорт.
//
// Метод блокируется до остановки клиента или отмены context.
func (c *Client) Start(ctx context.Context) error {
	if err := c.LoadSession(ctx); err != nil {
		return err
	}

	if err := c.ensureDeviceID(ctx); err != nil {
		return err
	}

	return c.ws.Run(ctx)
}

// Stop останавливает WebSocket транспорт.
//
// Переданная ошибка будет возвращена из Start.
func (c *Client) Stop(err error) {
	c.ws.Stop(err)
}

// Websocket возвращает используемый WebSocket транспорт.
func (c *Client) Websocket() *transport.Websocket {
	return c.ws
}

// RPC возвращает используемый RPC клиент.
func (c *Client) RPC() *rpc.Client {
	return c.rpc
}

// Call выполняет RPC вызов через MAX клиент.
func (c *Client) Call(ctx context.Context, opcode protocol.Opcode, payload any, resp any) error {
	return c.rpc.Call(ctx, opcode, payload, resp)
}

// heartbeat выполняет heartbeat запрос MAX протокола.
//
// Метод используется транспортом для проверки доступности
// соединения и отличается от стандартного WebSocket Ping тем,
// что использует специальный opcode MAX сервера.
//
// Параметр WebSocket соединения не используется, поскольку
// heartbeat выполняется через RPC слой.
func (c *Client) heartbeat(ctx context.Context, _ *transport.Websocket) error {
	var resp json.RawMessage
	return c.Call(ctx, OpcodePing, Map{"interactive": false}, &resp)
}

// ensureDeviceID гарантирует наличие DeviceID в сессии.
//
// Если сессия отсутствует, создаётся новая.
// Если DeviceID отсутствует, генерируется новый и сохраняется.
func (c *Client) ensureDeviceID(ctx context.Context) error {
	session := c.Session()
	if session == nil {
		session = &Session{}
	}

	if session.DeviceID != "" {
		return nil
	}

	deviceID, err := NewDeviceID()
	if err != nil {
		return err
	}

	session.DeviceID = deviceID

	return c.SetSession(ctx, session)
}
