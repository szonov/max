// Package protocol содержит базовые типы MAX протокола.
//
// Пакет описывает формат сообщений, версии протокола,
// команды и другие сущности, используемые при обмене
// данными с сервером MAX.
package protocol

import "encoding/json"

// Version определяет версию MAX протокола.
type Version uint16

const (
	// VersionCurrent текущая поддерживаемая версия MAX протокола.
	VersionCurrent Version = 11
)

// Command определяет тип сообщения MAX протокола.
type Command uint8

const (
	// CommandRequest обозначает запрос клиента к серверу.
	CommandRequest Command = 0

	// CommandResponse обозначает успешный ответ сервера.
	CommandResponse Command = 1

	// CommandError обозначает ошибку выполнения запроса.
	CommandError Command = 3
)

// Opcode определяет тип операции MAX протокола.
//
// Конкретные значения opcode зависят от используемого API.
type Opcode int32

// Message представляет сообщение MAX протокола.
//
// Сообщение используется для передачи запросов,
// ответов и ошибок между клиентом и сервером.
type Message struct {
	// Cmd определяет тип сообщения.
	Cmd Command `json:"cmd"`

	// Opcode определяет выполняемую операцию.
	Opcode Opcode `json:"opcode"`

	// Payload содержит данные сообщения в JSON формате.
	Payload json.RawMessage `json:"payload"`

	// Seq содержит идентификатор запроса,
	// используемый для сопоставления запросов и ответов.
	Seq uint32 `json:"seq"`

	// Ver содержит версию протокола.
	Ver Version `json:"ver"`
}

// Decode распаковывает Payload сообщения в переданную структуру.
//
// Обычно используется для преобразования данных
// запроса, ответа или события в конкретный тип.
//
// Пример:
//
//	var resp MyResponse
//	if err := msg.Decode(&resp); err != nil {
//	    ...
//	}
func (m Message) Decode(payload any) error {
	return json.Unmarshal(m.Payload, payload)
}
