package rpc

import (
	"errors"
	"fmt"

	"github.com/szonov/max/protocol"
)

// Ошибки RPC слоя.

// ErrTimeout означает, что RPC запрос не получил ответ
// за отведенное время ожидания.
var ErrTimeout = errors.New("request timeout")

// ErrClosed означает, что транспортное соединение закрыто
// и выполнение RPC невозможно.
var ErrClosed = errors.New("connection closed")

// ErrTransportInvalid означает, что RPC клиент создан
// без настроенного транспорта.
var ErrTransportInvalid = errors.New("transport is not set")

// Error содержит ошибку выполнения RPC запроса.
//
// Включает opcode и sequence number, что позволяет
// сопоставить ошибку с конкретным запросом.
//
// Используется для ошибок локального выполнения:
// - ошибка транспорта;
// - таймаут ожидания;
// - ошибка сериализации.
//
// Ошибки, пришедшие от MAX сервера,
// представлены типом MaxError.
type Error struct {
	Opcode protocol.Opcode
	Seq    uint32
	Err    error
}

func (e *Error) Error() string {
	return fmt.Sprintf("rpc opcode=%d seq=%d: %v", e.Opcode, e.Seq, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func wrapError(opcode protocol.Opcode, seq uint32, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Opcode: opcode, Seq: seq, Err: err}
}

// MaxError представляет ошибку,
// возвращенную сервером MAX в ответе CommandError.
//
// Code содержит идентификатор ошибки сервера,
// например "track.not.found".
type MaxError struct {
	Code    string
	Message string
}

func (e *MaxError) Error() string {
	return fmt.Sprintf("max error %s: %s", e.Code, e.Message)
}
