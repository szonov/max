package statestore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// FileStore реализует интерфейс Store,
// сохраняя состояние в JSON файле.
//
// FileStore является одной из возможных реализаций Store
// и может использоваться для хранения конфигурации,
// сессий и других небольших объемов данных.
type FileStore[T any] struct {
	path string
}

// NewFileStore создаёт файловое хранилище состояния.
//
// Значение будет сохраняться в указанный JSON файл.
func NewFileStore[T any](path string) *FileStore[T] {
	return &FileStore[T]{
		path: path,
	}
}

// Load загружает состояние из файла.
//
// Если файл отсутствует, возвращает (nil, nil).
//
// Возвращает ошибку при невозможности прочитать файл
// или декодировать его содержимое.
func (s *FileStore[T]) Load(ctx context.Context) (*T, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	var value T

	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}

	return &value, nil
}

// Save сохраняет состояние в файл.
//
// Перед сохранением автоматически создаются отсутствующие
// директории.
//
// Запись выполняется через временный файл с последующим
// атомарным переименованием, что уменьшает риск повреждения
// данных при аварийном завершении процесса.
func (s *FileStore[T]) Save(ctx context.Context, value T) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	dir := filepath.Dir(s.path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"

	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tmp, s.path)
}

// Destroy удаляет файл состояния.
//
// Отсутствие файла не считается ошибкой.
func (s *FileStore[T]) Destroy(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}
