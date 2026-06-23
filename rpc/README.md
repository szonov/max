# rpc

Пакет `rpc` реализует механизм запросов и ответов поверх транспортного слоя.

Пакет использует `transport.Websocket` для передачи сообщений и предоставляет удобный API для выполнения RPC вызовов с ожиданием ответа.

## Назначение

`rpc` отвечает за:

* генерацию уникальных sequence number;
* отправку запросов;
* ожидание ответов;
* сопоставление ответов с запросами;
* обработку ошибок протокола;
* обработку таймаутов.

Пакет не содержит бизнес-логики приложения и не знает назначение конкретных opcode.

## Архитектура

```text
application
    ↓
rpc
    ↓
transport
    ↓
websocket
```

`transport` отвечает за доставку сообщений.

`rpc` отвечает за механизм request/response.

Интерпретация конкретных opcode выполняется уровнем выше.

## Установка

```bash
go get github.com/szonov/max/rpc
```

## Создание клиента

```go
ws := transport.New("wss://example.com/ws")

client := rpc.New(
    ws,
    protocol.VersionCurrent,
)
```

## Выполнение запроса

```go
var resp GetProfileResponse

err := client.Call(
    ctx,
    OpcodeGetProfile,
    GetProfileRequest{
        UserID: 123,
    },
    &resp,
)
if err != nil {
    return err
}
```

Если запрос выполнен успешно, ответ будет декодирован в `resp`.

## Запрос без ожидания ответа

Если ответ не требуется, можно передать `nil`:

```go
err := client.Call(
    ctx,
    OpcodeTyping,
    TypingRequest{
        ChatID: 123,
    },
    nil,
)
```

В этом случае запрос будет отправлен без ожидания ответа.

## Серверные события

Сообщения, не связанные с ожидающими RPC вызовами, публикуются через событие:

```go
client.Events.Message.Subscribe(
    func(ctx context.Context, msg protocol.Message) {
        // обработка серверного события
    },
)
```

Например, это могут быть уведомления или другие сообщения, инициированные сервером.

## Ошибки

### Ошибки RPC слоя

```go
errors.Is(err, rpc.ErrTimeout)
errors.Is(err, rpc.ErrClosed)
```

### Ошибки транспорта

```go
var transportErr *transport.Error

if errors.As(err, &transportErr) {
    // обработка ошибки транспорта
}
```

### Ошибки сервера

```go
var maxErr *rpc.MaxError

if errors.As(err, &maxErr) {
    fmt.Println(maxErr.Code)
}
```

## Что не входит в ответственность пакета

Пакет не реализует:

* авторизацию;
* хранение сессий;
* обработку контактов;
* обработку чатов;
* бизнес-логику приложения.

Эти задачи должны решаться отдельными слоями поверх RPC клиента.
