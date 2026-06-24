package maxclient

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/szonov/max/transport"
)

// WebsocketURL содержит адрес WebSocket endpoint MAX.
const WebsocketURL = "wss://ws-api.oneme.ru/websocket"

// AppVersion содержит версию web-клиента MAX,
// передаваемую серверу в запросе Hello.
const AppVersion = "26.6.10"

// UserAgent содержит User-Agent web-клиента MAX.
//
// Используется при установке WebSocket соединения
// и передается серверу в запросе Hello.
const UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:151.0) Gecko/20100101 Firefox/151.0"

// Origin содержит Origin web-клиента MAX.
const Origin = "https://web.max.ru"

// DefaultTransport создаёт WebSocket транспорт
// с настройками по умолчанию для MAX.
//
// Устанавливает стандартные WebSocket URL,
// User-Agent и Origin.
//
// Переданные opts применяются после настроек по умолчанию
// и могут переопределять параметры транспорта.
func DefaultTransport(opts ...transport.Option) *transport.Websocket {

	// Создаем список опций
	options := []transport.Option{
		transport.WithHeaderFunc(func() http.Header {
			return http.Header{
				"User-Agent": []string{UserAgent},
				"Origin":     []string{Origin},
			}
		}),
	}

	// Добавляем переданные опции
	options = append(options, opts...)

	return transport.New(WebsocketURL, options...)
}

// NewDeviceID создаёт новый случайный идентификатор устройства.
//
// Идентификатор имеет формат UUID v4 и используется
// клиентом MAX как device_id.
func NewDeviceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// DefaultHelloUserAgent возвращает объект userAgent,
// используемый web-клиентом MAX в запросе Hello.
//
// Возвращаемое значение является копией и может
// свободно изменяться вызывающим кодом.
func DefaultHelloUserAgent() Map {
	return Map{
		"deviceType":      "WEB",
		"pushDeviceType":  "WEBPUSH",
		"locale":          "ru",
		"deviceLocale":    "ru",
		"osVersion":       "Linux",
		"deviceName":      "Firefox",
		"headerUserAgent": UserAgent,
		"appVersion":      AppVersion,
		"screen":          "1080x1920 1.0x",
		"timezone":        "Asia/Novosibirsk",
	}
}

// DefaultLoginViaTokenPayload возвращает payload,
// используемый при авторизации через токен по умолчанию.
//
// Поле token добавляется автоматически в LoginViaToken.
func DefaultLoginViaTokenPayload() Map {
	return Map{
		"chatsCount":   40,
		"interactive":  true,
		"chatsSync":    0,
		"contactsSync": 0,
		"presenceSync": -1,
		"draftsSync":   0,
	}
}
