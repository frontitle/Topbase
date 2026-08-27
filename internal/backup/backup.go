package backup

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var sidecarFiles = []string{
	"connection-secrets.json",
	"catalog.json",
	"table-metadata.json",
}

type Manifest struct {
	CreatedAt string   `json:"created_at"`
	Version   string   `json:"version"`
	Files     []string `json:"files"`
}

// Create produces a consistent SQLite snapshot plus Topbase's file-backed
// secrets and compatibility metadata. It writes into a temporary sibling and
// only renames it to destination after every file has been completed.
func Create(dataDir, destination, version string, now time.Time) (Manifest, error) {
	if dataDir == "" || destination == "" {
		return Manifest{}, errors.New("data directory and destination are required")
	}
	sourceDB := filepath.Join(dataDir, "app.db")
	if info, err := os.Stat(sourceDB); err != nil {
		return Manifest{}, fmt.Errorf("open source app database: %w", err)
	} else if !info.Mode().IsRegular() {
		return Manifest{}, fmt.Errorf("source app database is not a regular file: %s", sourceDB)
	}

	destination, err := filepath.Abs(destination)
	if err != nil {
		return Manifest{}, err
	}
	if _, err := os.Stat(destination); err == nil {
		return Manifest{}, fmt.Errorf("backup destination already exists: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return Manifest{}, err
	}
	temporary, err := os.MkdirTemp(parent, "."+filepath.Base(destination)+".tmp-")
	if err != nil {
		return Manifest{}, err
	}
	defer os.RemoveAll(temporary)
	if err := os.Chmod(temporary, 0700); err != nil {
		return Manifest{}, err
	}

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(sourceDB)+"?_pragma=busy_timeout(10000)")
	if err != nil {
		return Manifest{}, err
	}
	defer db.Close()
	backupDB := filepath.Join(temporary, "app.db")
	if _, err := db.Exec(`VACUUM INTO ?`, backupDB); err != nil {
		return Manifest{}, fmt.Errorf("snapshot application database: %w", err)
	}
	if err := os.Chmod(backupDB, 0600); err != nil {
		return Manifest{}, err
	}

	manifest := Manifest{
		CreatedAt: now.UTC().Format(time.RFC3339),
		Version:   version,
		Files:     []string{"app.db"},
	}
	for _, name := range sidecarFiles {
		source := filepath.Join(dataDir, name)
		if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return Manifest{}, err
		}
		if err := copyFile(source, filepath.Join(temporary, name)); err != nil {
			return Manifest{}, fmt.Errorf("copy %s: %w", name, err)
		}
		manifest.Files = append(manifest.Files, name)
	}
	manifest.Files = append(manifest.Files, "manifest.json")
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Manifest{}, err
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := os.WriteFile(filepath.Join(temporary, "manifest.json"), manifestBytes, 0600); err != nil {
		return Manifest{}, err
	}
	if err := os.Rename(temporary, destination); err != nil {
		return Manifest{}, fmt.Errorf("publish backup: %w", err)
	}
	return manifest, nil
}

func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	return errors.Join(copyErr, closeErr)
}
