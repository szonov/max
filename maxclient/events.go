package maxclient

import (
	"encoding/json"

	"github.com/szonov/max/events"
	"github.com/szonov/max/protocol"
)

// Events содержит события MAX клиента.
type Events struct {
	// Ready вызывается после успешной авторизации.
	//
	// В событие передаётся необработанный ответ сервера
	// на запрос LoginViaToken. Ответ может содержать
	// профиль пользователя, контакты, чаты и другие данные.
	Ready events.EventOf[json.RawMessage]

	// AuthRequired вызывается, когда клиенту требуется
	// выполнить авторизацию.
	//
	// Обычно это происходит при отсутствии сохранённого токена
	// или после получения ошибки AuthInvalidToken.
	AuthRequired events.Event

	// QrCode вызывается при начале авторизации через QR код.
	//
	// Обработчик должен показать пользователю QR код
	// и скрыть его после отмены переданного context.
	QrCode events.EventOf[QrCode]

	// Error вызывается при возникновении ошибок,
	// которые не могут быть обработаны автоматически.
	Error events.EventOf[error]

	// Message вызывается для входящих сообщений MAX протокола,
	// не связанных с RPC запросами.
	//
	// Содержимое сообщения не интерпретируется библиотекой
	// и передаётся приложению в виде protocol.Message.
	Message events.EventOf[protocol.Message]
}
