package adapters

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/topbase/topbase/internal/core"
)

// FileConnectionSecretStore is the local-development secret store. Its file
// and parent directory are owner-only; deployments should replace it with a
// KMS-backed implementation.
type FileConnectionSecretStore struct {
	path string
	mu   sync.Mutex
}

func NewFileConnectionSecretStore(path string) *FileConnectionSecretStore {
	return &FileConnectionSecretStore{path: path}
}

func (s *FileConnectionSecretStore) SaveConnectionSecret(id string, value core.ConnectionRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	value.ID = id
	items[id] = value
	return s.write(items)
}

func (s *FileConnectionSecretStore) GetConnectionSecret(id string) (core.ConnectionRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return core.ConnectionRequest{}, err
	}
	item, ok := items[id]
	if !ok {
		return core.ConnectionRequest{}, core.ErrNotFound
	}
	item.ID = id
	return item, nil
}

func (s *FileConnectionSecretStore) ListConnectionSecrets() (map[string]core.ConnectionRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

func (s *FileConnectionSecretStore) DeleteConnectionSecret(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	delete(items, id)
	return s.write(items)
}

func (s *FileConnectionSecretStore) read() (map[string]core.ConnectionRequest, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]core.ConnectionRequest{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := map[string]core.ConnectionRequest{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *FileConnectionSecretStore) write(items map[string]core.ConnectionRequest) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}
