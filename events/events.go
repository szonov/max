package events

import (
	"context"
	"sync"
)

// Event представляет событие без дополнительных данных.
//
// Подписчики получают только контекст вызова.
type Event struct {
	mu       sync.RWMutex
	handlers []func(context.Context)
}

// Subscribe добавляет обработчик события.
//
// Добавленный обработчик будет вызван при следующем и последующих вызовах Emit.
// Метод безопасен для вызова из нескольких goroutine.
func (e *Event) Subscribe(h func(context.Context)) {
	e.mu.Lock()
	e.handlers = append(e.handlers, h)
	e.mu.Unlock()
}

// HasSubscribers возвращает true, если у события есть хотя бы один подписчик.
//
// Метод используется для проверки наличия обработчиков перед выполнением
// операций, результат которых требует обязательного потребителя.
//
// Метод безопасен для вызова из нескольких goroutine.
func (e *Event) HasSubscribers() bool {
	e.mu.Lock()
	res := len(e.handlers) > 0
	e.mu.Unlock()

	return res
}

// Emit вызывает зарегистрированные обработчики последовательно.
//
// Подписчики вызываются в порядке подписки.
// Потокобезопасность обеспечивается внутри события.
// Способ вызова обработчиков определяется владельцем события.
//
// Обработчики не должны изменять список подписчиков во время выполнения.
func (e *Event) Emit(ctx context.Context) {
	e.mu.RLock()
	handlers := append([]func(context.Context){}, e.handlers...)
	e.mu.RUnlock()

	for _, h := range handlers {
		h(ctx)
	}
}

// EventOf представляет событие с одним параметром.
//
// Параметр события задаётся через тип T.
// EventOf используется для передачи данных вместе с уведомлением.
//
// Например:
//
//	var MessageEvent Event1[Message]
type EventOf[T any] struct {
	mu       sync.RWMutex
	handlers []func(context.Context, T)
}

// Subscribe добавляет обработчик события.
//
// Обработчик получает context и значение события.
// Добавленный обработчик будет вызван при следующих вызовах Emit.
//
// Метод безопасен для вызова из нескольких goroutine.
func (e *EventOf[T]) Subscribe(h func(context.Context, T)) {
	e.mu.Lock()
	e.handlers = append(e.handlers, h)
	e.mu.Unlock()
}

// HasSubscribers возвращает true, если есть зарегистрированные обработчики.
//
// Может использоваться для определения необходимости генерации события.
//
// Метод безопасен для вызова из нескольких goroutine.
func (e *EventOf[T]) HasSubscribers() bool {
	e.mu.Lock()
	res := len(e.handlers) > 0
	e.mu.Unlock()

	return res
}

// Emit вызывает зарегистрированные обработчики последовательно.
//
// Всем обработчикам передаётся одинаковый context и значение события.
//
// Подписчики вызываются в порядке подписки.
// Потокобезопасность обеспечивается внутри события.
// Способ вызова обработчиков определяется владельцем события.
//
// Обработчики не должны изменять список подписчиков во время выполнения.
func (e *EventOf[T]) Emit(ctx context.Context, v T) {
	e.mu.RLock()
	handlers := append([]func(context.Context, T){}, e.handlers...)
	e.mu.RUnlock()

	for _, h := range handlers {
		h(ctx, v)
	}
}
