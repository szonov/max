# transport

Универсальный WebSocket транспорт для JSON-сообщений.

Пакет предоставляет низкоуровневый слой работы с WebSocket соединением:
установка соединения, переподключение, heartbeat, отправка и получение сообщений.

**Transport специально не содержит логики конкретного протокола и работает только с транспортным уровнем WebSocket..**

Он не знает о формате сообщений, методах API или правилах взаимодействия
с сервером. Пакет может использоваться с любым JSON WebSocket API.

## Возможности

- подключение к WebSocket серверу;
- автоматическое переподключение при разрыве соединения;
- heartbeat-проверка соединения;
- пользовательская реализация heartbeat;
- отправка JSON сообщений;
- получение сообщений через события;
- контроль жизненного цикла соединения;
- потокобезопасная отправка сообщений.

## Установка

Пакет можно использовать как отдельный WebSocket транспорт:

```bash
go get github.com/szonov/max/transport
```

## Пример использования

Пример простого клиента для любого JSON WebSocket API:

```go
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/szonov/max/transport"
)

func main() {
	ctx := context.Background()

	ws := transport.New(
		"wss://example.com/socket",
	)

	ws.Events.Connect.Subscribe(
		func(ctx context.Context) {
			log.Println("connected")
		},
	)

	ws.Events.Message.Subscribe(
		func(ctx context.Context, msg json.RawMessage) {
			log.Println("message:", string(msg))
		},
	)

	ws.Events.Error.Subscribe(
		func(ctx context.Context, err error) {
			log.Println("transport error:", err)
		},
	)

	if err := ws.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
```

### Отправка сообщений

После установки соединения сообщения можно отправлять напрямую:

```go
err := ws.Send(
    ctx,
    map[string]any{
    "type": "ping",
    },
)

if err != nil {
    log.Fatal(err)
}
```

## События

Transport предоставляет события:

```go
type Events struct {
    Connect
    Disconnect
    Error
    Message
}
```

`Message` содержит исходное JSON-сообщение без преобразования.

Разбор сообщения должен выполняться уровнем выше.

Например:

```
WebSocket
    ↓
transport
    ↓
rpc / protocol
    ↓
application
```

## Heartbeat

По умолчанию используется стандартный WebSocket Ping.

При необходимости можно заменить проверку соединения:

```go
transport.WithHeartbeatFunc(
    func(ctx context.Context, ws *Websocket) error {
        return ws.Send(ctx, heartbeatMessage{})
    },
)
```

## Не входит в ответственность пакета

Transport не реализует:

- протокол приложения;
- авторизацию;
- RPC;
- маршрутизацию сообщений;
- хранение состояния.

Эти задачи должны решаться отдельными слоями.

## Использование

Пакет используется как транспортный слой в проекте
`github.com/szonov/max`, но может применяться независимо от него.
```