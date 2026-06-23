package transport

import (
	"errors"
	"fmt"
)

// Operation описывает этап работы транспорта, на котором произошла ошибка.
type Operation string

const (
	// OpDial ошибка при установке WebSocket соединения.
	OpDial Operation = "dial"

	// OpRead ошибка при чтении данных из WebSocket соединения.
	OpRead Operation = "read"

	// OpWrite ошибка при отправке данных в WebSocket соединение.
	OpWrite Operation = "write"

	// OpHeartbeat ошибка при проверке состояния соединения.
	OpHeartbeat Operation = "heartbeat"
)

// ErrHeartbeatTimeout возвращается при отсутствии ответа на heartbeat.
var ErrHeartbeatTimeout = errors.New("heartbeat timeout")

// ErrClosed возвращается при попытке выполнить операцию
// с закрытым или неактивным соединением.
var ErrClosed = errors.New("connection closed")

// Error представляет ошибку транспорта.
//
// Поле Op позволяет определить этап, на котором произошла ошибка,
// а Err содержит исходную ошибку.
//
// Error поддерживает errors.Is и errors.As через Unwrap.
type Error struct {
	Op  Operation
	Err error
}

func (e *Error) Error() string {
	return fmt.Sprintf("transport %s: %v", e.Op, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// wrapError добавляет информацию об операции транспорта к исходной ошибке.
//
// Если err == nil, возвращается nil.
// Исходная ошибка доступна через errors.Is/errors.As.
func wrapError(op Operation, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Err: err}
}
