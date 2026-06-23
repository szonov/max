package statestore

import (
	"context"
	"errors"
	"sync"
)

// ErrNilValue возвращается при попытке сохранить
// nil значение через State.Set.
var ErrNilValue = errors.New("value is nil")

// State предоставляет потоко-безопасное хранение состояния в памяти
// с возможностью сохранения через Store.
//
// State хранит локальную копию значения и при наличии Store
// синхронизирует изменения с постоянным хранилищем.
//
// Возвращаемые значения всегда копируются, чтобы исключить
// изменение внутреннего состояния извне.
type State[T any] struct {
	mu    sync.RWMutex
	value *T
	store Store[T]
}

// NewState создаёт новое состояние.
//
// Если store равен nil, состояние будет храниться
// только в памяти процесса.
func NewState[T any](store Store[T]) *State[T] {
	return &State[T]{
		store: store,
	}
}

// NewFileState создаёт State с файловым хранилищем.
//
// Эквивалентно:
//
//	store := NewFileStore[T](path)
//	state := NewState(store)
func NewFileState[T any](path string) *State[T] {
	return NewState(
		NewFileStore[T](path),
	)
}

// set заменяет текущее значение состояния.
//
// Внутри сохраняется копия переданного значения.
func (s *State[T]) set(value *T) {
	if value == nil {
		s.value = nil
		return
	}
	// set A COPY of value
	s.value = new(*value)
}

// Get возвращает текущее значение состояния.
//
// Если состояние отсутствует, возвращается nil.
//
// Возвращаемое значение является копией внутреннего состояния,
// поэтому его изменение не влияет на содержимое State.
func (s *State[T]) Get() *T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.value == nil {
		return nil
	}

	// returns A COPY of value
	return new(*s.value)
}

// Set обновляет текущее состояние.
//
// Значение копируется во внутреннее состояние.
//
// Если настроен Store, новое значение дополнительно
// сохраняется в постоянное хранилище.
//
// Важно: State сначала обновляет локальное значение, а затем выполняет сохранение через Store.
//
// Возвращает ErrNilValue, если value == nil.
// Для удаления состояния используйте Clear.
func (s *State[T]) Set(ctx context.Context, value *T) error {
	if value == nil {
		return ErrNilValue
	}

	s.mu.Lock()
	s.set(value)
	s.mu.Unlock()

	if s.store != nil {
		return s.store.Save(ctx, *value)
	}

	return nil
}

// Clear удаляет текущее состояние.
//
// Если настроен Store, состояние также удаляется
// из постоянного хранилища.
func (s *State[T]) Clear(ctx context.Context) error {
	s.mu.Lock()
	s.value = nil
	s.mu.Unlock()

	if s.store != nil {
		return s.store.Destroy(ctx)
	}
	return nil
}

// Has возвращает true, если состояние установлено.
func (s *State[T]) Has() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.value != nil
}

// Load синхронизирует внутреннее состояние с данными из Store.
//
// Если Store не настроен, метод ничего не делает.
//
// Если хранилище возвращает nil значение,
// состояние считается отсутствующим.
//
// Возвращает ошибку только в случае ошибки чтения из хранилища.
func (s *State[T]) Load(ctx context.Context) error {
	if s.store == nil {
		return nil
	}

	value, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.set(value)
	s.mu.Unlock()

	return nil
}
