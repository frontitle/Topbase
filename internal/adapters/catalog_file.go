package adapters

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/topbase/topbase/internal/core"
)

// FileCatalog is a development catalog. It persists non-secret connection
// metadata only; source DSNs remain in the running process and should be
// supplied by a secret manager in production.
type FileCatalog struct {
	path string
	mu   sync.Mutex
}

func NewFileCatalog(path string) *FileCatalog { return &FileCatalog{path: path} }

func (c *FileCatalog) List() ([]core.Database, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.read()
}

func (c *FileCatalog) Save(database core.Database) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	items, err := c.read()
	if err != nil {
		return err
	}
	for i, existing := range items {
		if existing.ID == database.ID {
			items[i] = database
			return c.write(items)
		}
	}
	return c.write(append(items, database))
}

func (c *FileCatalog) Delete(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	items, err := c.read()
	if err != nil {
		return err
	}
	filtered := items[:0]
	for _, item := range items {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	return c.write(filtered)
}

func (c *FileCatalog) read() ([]core.Database, error) {
	data, err := os.ReadFile(c.path)
	if errors.Is(err, os.ErrNotExist) {
		return []core.Database{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []core.Database
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *FileCatalog) write(items []core.Database) error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0600)
}
