# statestore

Пакет `statestore` предоставляет простой механизм хранения состояния приложения.

Основная идея пакета заключается в разделении:

* состояния в памяти (`State`);
* механизма постоянного хранения (`Store`).

## Возможности

* потоко-безопасное хранение состояния в памяти;
* автоматическая синхронизация с хранилищем;
* поддержка любых типов через generics;
* абстракция хранения через интерфейс `Store`;
* готовая файловая реализация `FileStore`.

## Архитектура

```text
State[T]
    ↓
Store[T]
    |
    +-- FileStore[T]
    +-- DatabaseStore[T]
    +-- RedisStore[T]
    +-- ...
```

`State` отвечает за работу с состоянием в памяти.

`Store` отвечает за постоянное хранение данных.

Пакет специально разделяет эти обязанности.

## Установка

```bash
go get github.com/szonov/max/statestore
```

## State

Создание состояния без постоянного хранилища:

```go
state := statestore.NewState[Config](nil)
```

Создание состояния с файловым хранилищем:

```go
store := statestore.NewFileStore[Config]("config.json")

state := statestore.NewState(store)
```

## Чтение состояния

```go
cfg := state.Get()

if cfg != nil {
	fmt.Println(cfg.Token)
}
```

Метод `Get` всегда возвращает копию значения.

Изменение возвращенного объекта не влияет на внутреннее состояние.

## Запись состояния

```go
cfg := Config{
	Token: "secret",
}

err := state.Set(ctx, &cfg)
```

Если настроен `Store`, новое значение будет автоматически сохранено.

## Удаление состояния

```go
err := state.Clear(ctx)
```

Если настроен `Store`, сохраненные данные также будут удалены.

## Проверка наличия состояния

```go
if state.Has() {
	...
}
```

## Загрузка состояния

```go
err := state.Load(ctx)
```

После вызова `Load` состояние будет загружено из хранилища в память.

Если сохраненного состояния нет, ошибка не возвращается.

## Store

Интерфейс `Store` определяет способ хранения данных:

```go
type Store[T any] interface {
	Load(ctx context.Context) (*T, error)
	Save(ctx context.Context, value T) error
	Destroy(ctx context.Context) error
}
```

Отсутствие данных не считается ошибкой.

В этом случае `Load` должен вернуть:

```go
nil, nil
```

## FileStore

`FileStore` реализует интерфейс `Store`, сохраняя данные в JSON файле.

Пример:

```go
store := statestore.NewFileStore[Session](
	"session.json",
)
```

Файлы сохраняются в формате JSON с автоматическим созданием директорий.

Запись выполняется через временный файл с последующим переименованием для уменьшения риска повреждения данных.

## Собственные реализации Store

Пользователь может реализовать собственное хранилище:

```go
type DatabaseStore struct {
	...
}

func (s *DatabaseStore) Load(ctx context.Context) (*Config, error) {
	...
}

func (s *DatabaseStore) Save(ctx context.Context, value Config) error {
	...
}

func (s *DatabaseStore) Destroy(ctx context.Context) error {
	...
}
```

После этого хранилище можно использовать вместе со `State`:

```go
state := statestore.NewState(
	&DatabaseStore{},
)
```

## Использование в проекте

Пакет используется в проекте `github.com/szonov/max` для хранения:

* сессий авторизации;
* конфигурации приложений;
* других небольших состояний.

При этом пакет не зависит от MAX и может использоваться отдельно.
