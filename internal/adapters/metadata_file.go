package adapters

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/topbase/topbase/internal/core"
)

type FileMetadataStore struct {
	path string
	mu   sync.Mutex
}

func NewFileMetadataStore(path string) *FileMetadataStore { return &FileMetadataStore{path: path} }

func (s *FileMetadataStore) All() (map[string]core.TableAnnotation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

func (s *FileMetadataStore) GetTableAnnotation(databaseID, schema, table string) (core.TableAnnotation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return core.TableAnnotation{}, err
	}
	return items[metadataKey(databaseID, schema, table)], nil
}

func (s *FileMetadataStore) SaveTableAnnotation(databaseID, schema, table string, value core.TableAnnotation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	items[metadataKey(databaseID, schema, table)] = value
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *FileMetadataStore) read() (map[string]core.TableAnnotation, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]core.TableAnnotation{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := map[string]core.TableAnnotation{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func metadataKey(databaseID, schema, table string) string {
	return databaseID + "/" + schema + "/" + table
}
