package maxclient

import (
	"context"
)

// Session содержит данные авторизационной сессии клиента.
//
// DeviceID используется для идентификации устройства.
//
// Token содержит токен авторизации и может отсутствовать,
// если клиент ещё не прошёл авторизацию.
type Session struct {
	DeviceID string `json:"device_id"`
	Token    string `json:"token,omitempty"`
}

// LoadSession загружает данные сессии из настроенного хранилища.
func (c *Client) LoadSession(ctx context.Context) error {
	return c.session.Load(ctx)
}

// SetSession сохраняет данные текущей сессии.
//
// При наличии хранилища сессия будет сохранена
// также в постоянное хранилище.
func (c *Client) SetSession(ctx context.Context, session *Session) error {
	return c.session.Set(ctx, session)
}

// Session возвращает копию текущей сессии.
//
// Если сессия отсутствует, возвращается nil.
func (c *Client) Session() *Session {
	return c.session.Get()
}

// mustSession возвращает текущую сессию.
//
// Метод используется во внутренних местах клиента,
// где по инвариантам приложения сессия обязана существовать.
//
// Если сессия отсутствует, вызывается panic, поскольку
// это означает нарушение внутренней логики клиента.
func (c *Client) mustSession() *Session {
	session := c.Session()

	if session == nil {
		panic("maxclient: session is nil")
	}

	return session
}

// HasSession возвращает true, если данные сессии загружены.
func (c *Client) HasSession() bool {
	return c.session.Has()
}

// ClearSession удаляет текущую сессию.
//
// При наличии хранилища данные также будут удалены
// из постоянного хранилища.
func (c *Client) ClearSession(ctx context.Context) error {
	return c.session.Clear(ctx)
}

// DeviceID возвращает идентификатор устройства из текущей сессии.
//
// Если сессия отсутствует, возвращается пустая строка.
func (c *Client) DeviceID() string {
	session := c.Session()
	if session == nil {
		return ""
	}
	return session.DeviceID
}

// Token возвращает токен авторизации из текущей сессии.
//
// Если сессия отсутствует или клиент не авторизован,
// возвращается пустая строка.
func (c *Client) Token() string {
	session := c.Session()
	if session == nil {
		return ""
	}
	return session.Token
}
