package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
)

type MockStorage struct {
	mu    sync.Mutex
	files map[string][]byte
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		files: make(map[string][]byte),
	}
}

func (m *MockStorage) Upload(ctx context.Context, key string, file io.Reader) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if file == nil {
		m.files[key] = []byte{}
		return nil
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	m.files[key] = data
	return nil
}

func (m *MockStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, ok := m.files[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.files, key)
	return nil
}
