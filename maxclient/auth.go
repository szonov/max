package maxclient

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"time"

	"github.com/szonov/max/rpc"
)

// AuthFailureReason описывает причину невозможности завершить авторизацию.
type AuthFailureReason string

const (
	// AuthRequired требуется выполнить авторизацию.
	AuthRequired AuthFailureReason = "auth_required"

	// AuthQrCodeRequired приложение должно показать QR код пользователю.
	AuthQrCodeRequired AuthFailureReason = "auth_qrcode_required"

	// AuthQrCodeExpired срок действия QR кода истёк.
	AuthQrCodeExpired AuthFailureReason = "qrcode_expired"

	// AuthPasswordRequired требуется пароль для аккаунта с 2FA.
	AuthPasswordRequired AuthFailureReason = "auth_password_required"

	// AuthNoTokenReceived сервер не вернул токен авторизации.
	AuthNoTokenReceived AuthFailureReason = "no_token_received"

	// AuthInvalidToken сохранённый токен недействителен.
	AuthInvalidToken AuthFailureReason = "invalid_token"
)

// AuthError представляет ожидаемую ошибку процесса
// авторизации и может быть обработана приложением
// через errors.As.
type AuthError struct {
	Reason AuthFailureReason
}

// NewAuthError создаёт ошибку авторизации.
func NewAuthError(reason AuthFailureReason) *AuthError {
	return &AuthError{Reason: reason}
}

// Error реализует интерфейс error.
func (e *AuthError) Error() string {
	return string(e.Reason)
}

// PasswordFunc запрашивает у приложения пароль для завершения авторизации.
//
// hint содержит подсказку пароля, если она задана пользователем.
type PasswordFunc func(ctx context.Context, hint string) (string, error)

// QrCode содержит данные для отображения QR кода авторизации.
type QrCode struct {
	// Link ссылка, которую необходимо закодировать в QR код.
	Link string

	// ExpiresAt время окончания действия QR кода.
	ExpiresAt time.Time
}

// LoginViaToken выполняет авторизацию через сохранённый token.
//
// При успешной авторизации клиент считается готовым к работе,
// а необработанный ответ сервера передаётся в событие Ready.
//
// Ответ может содержать профиль, контакты, чаты и другие данные.
// maxclient не интерпретирует эти данные; приложение может
// самостоятельно распаковать json.RawMessage.
func (c *Client) LoginViaToken(ctx context.Context) (json.RawMessage, error) {
	token := c.Token()
	if token == "" {
		return nil, NewAuthError(AuthInvalidToken)
	}

	req := maps.Clone(c.loginViaTokenPayload)
	if req == nil {
		req = Map{}
	}

	req["token"] = token

	var resp json.RawMessage

	if err := c.Call(ctx, OpcodeLoginByToken, req, &resp); err != nil {

		if maxError, ok := errors.AsType[*rpc.MaxError](err); ok {
			if maxError.Code == "login.token" {
				return nil, NewAuthError(AuthInvalidToken)
			}
		}

		return nil, err
	}

	resp = append(json.RawMessage(nil), resp...)

	c.Events.Ready.Emit(ctx, resp)

	return resp, nil
}

// LoginViaQr требует запущенного клиента.
// Метод должен вызываться после Start().
func (c *Client) LoginViaQr(ctx context.Context) (json.RawMessage, error) {
	var resp struct {
		PollingInterval int    `json:"pollingInterval"`
		Link            string `json:"qrLink"`
		Ttl             int    `json:"ttl"`
		TrackId         string `json:"trackId"`
		ExpiresAt       int64  `json:"expiresAt"`
	}

	if err := c.Call(ctx, OpcodeQrStart, nil, &resp); err != nil {
		return nil, err
	}

	expiresAt := time.UnixMilli(resp.ExpiresAt)

	if !c.Events.QrCode.HasSubscribers() {
		return nil, NewAuthError(AuthQrCodeRequired)
	}

	waitCtx, cancel := context.WithTimeout(ctx, time.Until(expiresAt))

	// Отправляем событие выше
	// пусть показывают каким-то образом конечному пользователю
	// отдаем ему контекст - показывает пока контекст не отменится или протухнет
	c.Events.QrCode.Emit(waitCtx, QrCode{
		Link:      resp.Link,
		ExpiresAt: expiresAt,
	})

	// А сами начинаем слушать ответ из websocket
	err := c.waitQrAuth(
		waitCtx,
		resp.TrackId,
		time.Duration(resp.PollingInterval)*time.Millisecond,
	)
	cancel()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, NewAuthError(AuthQrCodeExpired)
		}
		return nil, err
	}

	trackId := resp.TrackId

	type qrLoginResponse struct {
		// Empty `{}` when 2FA cloud password is enabled and `passwordChallenge` is set instead.
		TokenAttrs struct {
			Login struct {
				Token string `json:"token"`
			} `json:"LOGIN"`
		} `json:"tokenAttrs"`
		// Present when account has 2FA cloud password — caller must finish via op 115.
		PasswordChallenge *struct {
			TrackId string `json:"trackId"`
			// Подсказка к паролю, заданная пользователем
			Hint string `json:"hint"`
		} `json:"passwordChallenge"`
	}

	// Пытаемся залогиниться с этим trackId
	var loginResp qrLoginResponse
	err = c.Call(ctx, OpcodeQrLogin, Map{"trackId": trackId}, &loginResp)
	if err != nil {
		return nil, err
	}

	token := loginResp.TokenAttrs.Login.Token

	// Аккаунт оказался с 2FA облачным паролем: tokenAttrs = `{}`, passwordChallenge установлен.
	if token == "" && loginResp.PasswordChallenge != nil && loginResp.PasswordChallenge.TrackId == trackId {
		pass, err := c.requestPassword(ctx, loginResp.PasswordChallenge.Hint)
		if err != nil {
			return nil, err
		}

		loginResp = qrLoginResponse{}
		err = c.Call(ctx, OpcodeQrPasswordLogin, Map{"trackId": trackId, "password": pass}, &loginResp)
		if err != nil {
			return nil, err
		}
		token = loginResp.TokenAttrs.Login.Token
	}

	if token != "" {
		session := c.mustSession()
		session.Token = token

		if err = c.SetSession(ctx, session); err != nil {
			return nil, err
		}

		return c.LoginViaToken(ctx)
	}

	return nil, NewAuthError(AuthNoTokenReceived)
}

// waitQrAuth ожидает подтверждения авторизации по QR коду.
//
// Метод периодически опрашивает сервер до тех пор,
// пока пользователь не подтвердит вход,
// не истечёт срок действия QR кода
// или не будет отменён context.
func (c *Client) waitQrAuth(ctx context.Context, trackId string, pollingInterval time.Duration) error {

	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	req := Map{"trackId": trackId}

	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			var resp struct {
				Status struct {
					LoginAvailable bool `json:"loginAvailable"`
				} `json:"status"`
			}
			if err := c.Call(ctx, OpcodeQrPoll, req, &resp); err != nil {
				return err
			}

			if resp.Status.LoginAvailable {
				return nil
			}
		}
	}
}

// Logout завершает текущую авторизационную сессию.
//
// После успешного выхода очищается локальная сессия
// и сохранённый результат Hello.
func (c *Client) Logout(ctx context.Context) error {
	var resp json.RawMessage
	err := c.Call(ctx, OpcodeLogout, nil, &resp)
	if err != nil {
		return err
	}
	c.setHelloResponse(nil)
	return c.ClearSession(ctx)
}

// requestPassword запрашивает у приложения пароль
// для аккаунта с включённой двухфакторной авторизацией.
//
// Если функция получения пароля не задана,
// возвращается AuthPasswordRequired.
func (c *Client) requestPassword(ctx context.Context, hint string) (string, error) {
	if c.passwordFunc == nil {
		return "", NewAuthError(AuthPasswordRequired)
	}

	return c.passwordFunc(ctx, hint)
}
