package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/szonov/max/protocol"
	"github.com/szonov/max/transport"
)

// Client реализует RPC слой поверх транспорта.
//
// Client отвечает за:
//   - генерацию sequence id;
//   - ожидание ответов;
//   - маршрутизацию response к ожидающему вызову;
//   - обработку ошибок протокола.
//
// Client не знает конкретные MAX opcode.
type Client struct {
	transport *transport.Websocket

	ver protocol.Version
	seq atomic.Uint32

	waitersMu sync.Mutex
	waiters   map[waiterKey]*waiter // Активные RPC запросы, ожидающие ответ от сервера.

	// events
	Events Events
}

func New(transport *transport.Websocket, ver protocol.Version) *Client {
	c := &Client{
		transport: transport,
		waiters:   make(map[waiterKey]*waiter),
		ver:       ver,
	}

	transport.Events.Message.Subscribe(c.onMessage)
	transport.Events.Disconnect.Subscribe(c.failAllWaiters)

	return c
}

func (c *Client) nextSeq() uint32 {
	return c.seq.Add(1)
}

// send отправляет RPC запрос.
//
// Формирует protocol.Message и передаёт его в транспорт.
// Не ожидает ответа.
func (c *Client) send(ctx context.Context, opcode protocol.Opcode, seq uint32, payload any) error {
	if c.transport == nil {
		return wrapError(opcode, seq, ErrTransportInvalid)
	}

	var raw json.RawMessage

	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return wrapError(opcode, seq, fmt.Errorf("marshal payload: %w", err))
		}
		raw = b
	}

	return c.transport.Send(ctx, protocol.Message{
		Cmd:     protocol.CommandRequest,
		Opcode:  opcode,
		Payload: raw,
		Seq:     seq,
		Ver:     c.ver,
	})
}

// onMessage обрабатывает входящие сообщения от транспорта.
//
// Если сообщение является ответом на ожидающий RPC вызов,
// оно передаётся соответствующему waiter.
//
// Остальные сообщения публикуются через Events.Message.
func (c *Client) onMessage(ctx context.Context, msg json.RawMessage) {

	var m protocol.Message

	if err := json.Unmarshal(msg, &m); err != nil {
		c.Events.Error.Emit(ctx, fmt.Errorf("decode message: %w", err))
		return
	}

	w := c.takeWaiter(m.Opcode, m.Seq)

	if w != nil {
		w.ch <- waiterResult{
			msg: m,
		}
		return
	}

	c.Events.Message.Emit(ctx, m)
}
