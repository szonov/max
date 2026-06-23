package rpc

import (
	"context"

	"github.com/szonov/max/protocol"
)

// waiterKey используется для поиска ожидающего RPC запроса.
//
// Ответ от сервера сопоставляется с запросом по комбинации
// opcode и sequence number.
type waiterKey struct {
	Opcode protocol.Opcode
	Seq    uint32
}

// waiterResult содержит результат ожидаемого RPC запроса.
//
// В случае успешного выполнения msg содержит ответ сервера.
//
// Если выполнение запроса невозможно (например, соединение
// разорвано до получения ответа), err содержит причину ошибки.
type waiterResult struct {
	msg protocol.Message
	err error
}

// waiter представляет ожидающий RPC запрос.
//
// После отправки запроса Client регистрирует waiter и ожидает
// поступления результата через канал ch.
//
// Результат может содержать как ответ сервера, так и ошибку,
// возникшую до получения ответа.
type waiter struct {
	ch chan waiterResult
}

// registerWaiter регистрирует ожидание ответа для RPC запроса.
//
// Созданный waiter будет использоваться для передачи ответа,
// соответствующего указанным opcode и seq.
func (c *Client) registerWaiter(opcode protocol.Opcode, seq uint32) *waiter {

	w := &waiter{
		ch: make(chan waiterResult, 1),
	}

	key := waiterKey{Opcode: opcode, Seq: seq}

	c.waitersMu.Lock()
	c.waiters[key] = w
	c.waitersMu.Unlock()

	return w
}

// removeWaiter удаляет ожидание ответа для RPC запроса.
//
// Обычно вызывается после получения ответа либо завершения
// ожидания по таймауту или отмене контекста.
func (c *Client) removeWaiter(opcode protocol.Opcode, seq uint32) {

	key := waiterKey{Opcode: opcode, Seq: seq}

	c.waitersMu.Lock()
	delete(c.waiters, key)
	c.waitersMu.Unlock()
}

// takeWaiter извлекает зарегистрированный waiter для указанного
// RPC запроса.
//
// После успешного извлечения waiter удаляется из списка активных
// ожиданий и больше не может быть получен повторно.
//
// Возвращает nil, если ожидание для указанной пары opcode и seq
// не зарегистрировано.
func (c *Client) takeWaiter(opcode protocol.Opcode, seq uint32) *waiter {

	key := waiterKey{Opcode: opcode, Seq: seq}

	c.waitersMu.Lock()
	defer c.waitersMu.Unlock()

	w := c.waiters[key]

	if w != nil {
		delete(c.waiters, key)
	}

	return w
}

// failAllWaiters завершает все активные ожидания RPC запросов.
//
// Используется при потере соединения или остановке транспорта,
// когда получение ответов от сервера больше невозможно.
//
// Каждому ожидающему запросу передается ошибка, после чего
// все waiter удаляются из списка активных ожиданий.
func (c *Client) failAllWaiters(ctx context.Context, err error) {
	c.waitersMu.Lock()
	defer c.waitersMu.Unlock()

	for k, w := range c.waiters {
		select {
		case w.ch <- waiterResult{err: err}:
		default:
		}
		delete(c.waiters, k)
	}
}
