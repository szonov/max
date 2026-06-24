package maxclient

import (
	"context"
	"encoding/json"
	"maps"
)

// Hello выполняет начальный запрос к MAX WebSocket серверу.
//
// Метод отправляет сведения об устройстве и клиенте,
// получает ответ сервера и сохраняет его в необработанном виде.
//
// Перед выполнением запроса старый ответ Hello очищается.
// Если запрос завершится ошибкой, HelloResponse вернёт nil.
func (c *Client) Hello(ctx context.Context) error {
	c.setHelloResponse(nil)

	req := Map{
		"deviceId":  c.DeviceID(),
		"userAgent": maps.Clone(c.helloUserAgent),
	}

	var resp json.RawMessage

	if err := c.Call(ctx, OpcodeHello, req, &resp); err != nil {
		return err
	}

	c.setHelloResponse(resp)

	return nil
}

// HelloResponse возвращает необработанный ответ
// последнего успешного запроса Hello.
//
// Если Hello ещё не выполнялся, возвращается nil.
//
// Возвращаемое значение является копией.
// Его можно безопасно изменять без влияния на состояние Client.
func (c *Client) HelloResponse() json.RawMessage {
	c.helloMu.RLock()
	defer c.helloMu.RUnlock()

	if c.hello == nil {
		return nil
	}

	return append(json.RawMessage(nil), c.hello...)
}

// setHelloResponse сохраняет копию ответа Hello.
func (c *Client) setHelloResponse(resp json.RawMessage) {
	c.helloMu.Lock()
	defer c.helloMu.Unlock()

	if resp == nil {
		c.hello = nil
		return
	}

	c.hello = append(json.RawMessage(nil), resp...)
}
