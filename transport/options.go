package transport

import (
	"log/slog"
	"net/http"
	"time"
)

type Option func(*Websocket)

// HeadersFunc устанавливает HTTP заголовки для соединения с websocket сервером
type HeadersFunc func() http.Header

// WithHeaderFunc установка функции получения HTTP заголовков для соединения с websocket сервером
func WithHeaderFunc(headerFunc HeadersFunc) Option {
	return func(ws *Websocket) {
		ws.headersFunc = headerFunc
	}
}

// WithHeartbeatInterval задаёт периодичность запуска heartbeat.
//
// Используется для регулярной проверки активности соединения.
func WithHeartbeatInterval(interval time.Duration) Option {
	return func(ws *Websocket) {
		ws.heartbeatInterval = interval
	}
}

// WithHeartbeatTimeout задаёт максимальное время ожидания ответа heartbeat.
//
// Если за указанное время проверка не завершилась успешно,
// соединение считается потерянным.
func WithHeartbeatTimeout(timeout time.Duration) Option {
	return func(ws *Websocket) {
		ws.heartbeatTimeout = timeout
	}
}

// WithReadLimit устанавливает максимальный размер одного входящего сообщения.
//
// По умолчанию используется значение 10 MB.
func WithReadLimit(readLimit int64) Option {
	return func(ws *Websocket) {
		ws.readLimit = readLimit
	}
}

// WithLogger устанавливает logger для транспорта.
//
// При включенном Debug уровне логирования транспорт может выводить
// дополнительные диагностические сообщения.
func WithLogger(logger *slog.Logger) Option {
	return func(ws *Websocket) {
		ws.logger = logger
		ws.debugEnabled = ws.logger != nil && ws.logger.Enabled(nil, slog.LevelDebug)
	}
}
