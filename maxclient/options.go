package maxclient

import (
	"maps"

	"github.com/szonov/max/protocol"
	"github.com/szonov/max/rpc"
	"github.com/szonov/max/statestore"
	"github.com/szonov/max/transport"
)

// Option настраивает Client при создании.
type Option func(client *Client)

// WithTransport устанавливает WebSocket транспорт.
//
// При установке транспорта автоматически создаётся
// RPC клиент и настраивается heartbeat MAX протокола.
func WithTransport(ws *transport.Websocket) Option {
	return func(c *Client) {
		if ws == nil {
			return
		}
		c.ws = ws
		c.rpc = rpc.New(ws, protocol.VersionCurrent)
		ws.SetHeartbeatFunc(c.heartbeat)
	}
}

// WithSessionStore устанавливает хранилище сессии.
//
// По умолчанию сессия хранится только в памяти.
func WithSessionStore(store statestore.Store[Session]) Option {
	return func(c *Client) {
		c.session = statestore.NewState(store)
	}
}

// WithSessionFileStore сохраняет сессию в JSON файле.
func WithSessionFileStore(path string) Option {
	return func(c *Client) {
		c.session = statestore.NewFileState[Session](path)
	}
}

// WithLoginViaTokenPayload устанавливает payload,
// используемый при авторизации через токен.
//
// Поле token добавляется автоматически и переопределяет
// значение token из payload, если оно было задано.
func WithLoginViaTokenPayload(payload Map) Option {
	return func(c *Client) {
		c.loginViaTokenPayload = maps.Clone(payload)
	}
}

// WithHelloUserAgent устанавливает объект userAgent,
// передаваемый серверу в запросе Hello.
//
// Значение полностью заменяет настройки по умолчанию.
//
// Поле deviceId формируется клиентом автоматически
// и не зависит от данного параметра.
func WithHelloUserAgent(userAgent Map) Option {
	return func(c *Client) {
		c.helloUserAgent = maps.Clone(userAgent)
	}
}

// WithPasswordFunc устанавливает функцию запроса пароля
// для аккаунтов с включённым облачным паролем.
func WithPasswordFunc(fn PasswordFunc) Option {
	return func(c *Client) {
		c.passwordFunc = fn
	}
}
