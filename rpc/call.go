package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/szonov/max/protocol"
)

// Call выполняет RPC запрос.
//
// payload сериализуется в JSON и отправляется серверу.
//
// Для получения ответа resp должен содержать указатель
// на структуру результата.
//
// Если resp != nil, Call ожидает ответ сервера и декодирует
// полученный payload в переданную структуру.
//
// Если resp == nil, запрос отправляется без ожидания ответа.
//
// Возможные ошибки:
//   - ErrClosed — транспорт не подключен;
//   - ErrTimeout — истекло время ожидания ответа;
//   - MaxError — сервер вернул CommandError;
//   - Error — ошибка RPC слоя (транспорт, сериализация и др.).
func (c *Client) Call(ctx context.Context, opcode protocol.Opcode, payload any, resp any) error {

	seq := c.nextSeq()

	if !c.transport.IsConnected() {
		return wrapError(opcode, seq, ErrClosed)
	}

	if resp == nil {
		// не нужно ждать ответа
		return c.send(ctx, opcode, seq, payload)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	w := c.registerWaiter(opcode, seq)
	defer c.removeWaiter(opcode, seq)

	if err := c.send(waitCtx, opcode, seq, payload); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return wrapError(opcode, seq, ErrTimeout)
		}
		return wrapError(opcode, seq, err)
	}

	select {

	case result := <-w.ch:

		if result.err != nil {
			return wrapError(opcode, seq, result.err)
		}

		env := result.msg

		switch env.Cmd {

		case protocol.CommandResponse:

			if err := json.Unmarshal(env.Payload, &resp); err != nil {
				return wrapError(opcode, seq, fmt.Errorf("decode response: %w, %+v", err, env.Payload))
			}
			return nil

		case protocol.CommandError:

			var p struct {
				Error   string `json:"error"`
				Message string `json:"message"`
			}

			if err := json.Unmarshal(env.Payload, &p); err != nil {
				return wrapError(opcode, seq, fmt.Errorf("decode error response: %w, %+v", err, env.Payload))
			}

			return wrapError(opcode, seq, &MaxError{
				Code:    p.Error,
				Message: p.Message,
			})

		default:
			return wrapError(opcode, seq, fmt.Errorf("unexpected cmd=%d", env.Cmd))
		}

	case <-waitCtx.Done():
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return wrapError(opcode, seq, ErrTimeout)
		}
		return waitCtx.Err()
	}
}
