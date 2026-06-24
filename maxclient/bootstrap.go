package maxclient

import (
	"context"
	"errors"
)

// bootstrap выполняет начальную инициализацию соединения.
//
// Вызывается после установки WebSocket соединения.
//
// Последовательность:
//
//   - Hello;
//   - проверка наличия токена;
//   - восстановление авторизации через токен;
//   - запрос авторизации, если токен отсутствует или недействителен.
func (c *Client) bootstrap(ctx context.Context) {

	if err := c.Hello(ctx); err != nil {
		c.Events.Error.Emit(ctx, err)
		return
	}

	if c.Token() == "" {
		c.requireAuth(ctx)
		return
	}

	_, err := c.LoginViaToken(ctx)

	if err != nil {

		// токен протух, либо вышли со всех устройств
		if authErr, ok := errors.AsType[*AuthError](err); ok && authErr.Reason == AuthInvalidToken {
			if err = c.ClearSession(ctx); err != nil {
				c.Events.Error.Emit(ctx, err)
			}

			c.requireAuth(ctx)

			return
		}

		// другие ошибки даем залогировать
		c.Events.Error.Emit(ctx, err)

		return
	}
}

// requireAuth сообщает приложению, что требуется авторизация.
//
// Если на событие AuthRequired нет подписчиков,
// клиент останавливается с ошибкой AuthRequired.
func (c *Client) requireAuth(ctx context.Context) {

	if !c.Events.AuthRequired.HasSubscribers() {
		c.Stop(NewAuthError(AuthRequired))
		return
	}

	c.Events.AuthRequired.Emit(ctx)
}
