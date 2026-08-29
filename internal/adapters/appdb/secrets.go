package appdb

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
)

const masterKeyBytes = 32

func LoadMasterKey(dataDir string, engine Engine) ([]byte, error) {
	if raw := strings.TrimSpace(os.Getenv("TOPBASE_MASTER_KEY")); raw != "" {
		return parseMasterKey(raw)
	}
	path := strings.TrimSpace(os.Getenv("TOPBASE_MASTER_KEY_FILE"))
	if path == "" {
		path = filepath.Join(dataDir, "master.key")
	}
	raw, err := os.ReadFile(path)
	if err == nil {
		return parseMasterKey(strings.TrimSpace(string(raw)))
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read Topbase master key: %w", err)
	}
	if engine != EngineSQLite || os.Getenv("TOPBASE_MASTER_KEY_FILE") != "" {
		return nil, fmt.Errorf("Topbase master key is required for a shared application database; set TOPBASE_MASTER_KEY or TOPBASE_MASTER_KEY_FILE")
	}
	key := make([]byte, masterKeyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate Topbase master key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	encoded := base64.RawStdEncoding.EncodeToString(key) + "\n"
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return nil, fmt.Errorf("persist Topbase master key: %w", err)
	}
	return key, nil
}

func parseMasterKey(raw string) ([]byte, error) {
	decoders := []func(string) ([]byte, error){
		base64.RawStdEncoding.DecodeString,
		base64.StdEncoding.DecodeString,
		hex.DecodeString,
	}
	for _, decode := range decoders {
		key, err := decode(raw)
		if err == nil && len(key) == masterKeyBytes {
			return key, nil
		}
	}
	return nil, fmt.Errorf("Topbase master key must be 32 bytes encoded as base64 or hexadecimal")
}

type connectionSecretStore struct {
	store *Store
	aead  cipher.AEAD
}

func (s *Store) ConnectionSecrets(key []byte) (core.ConnectionSecretStore, error) {
	if len(key) != masterKeyBytes {
		return nil, fmt.Errorf("Topbase master key must contain %d bytes", masterKeyBytes)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &connectionSecretStore{store: s, aead: aead}, nil
}

func (s *connectionSecretStore) SaveConnectionSecret(id string, value core.ConnectionRequest) error {
	value.ID = id
	plaintext, err := json.Marshal(value)
	if err != nil {
		return err
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	ciphertext := s.aead.Seal(nil, nonce, plaintext, []byte(id))
	_, err = s.store.db.Exec(`INSERT INTO connection_secrets(database_id, ciphertext, nonce, key_version, updated_at) VALUES(?,?,?,?,?)
		ON CONFLICT(database_id) DO UPDATE SET ciphertext=excluded.ciphertext, nonce=excluded.nonce, key_version=excluded.key_version, updated_at=excluded.updated_at`,
		id, base64.RawStdEncoding.EncodeToString(ciphertext), base64.RawStdEncoding.EncodeToString(nonce), 1, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *connectionSecretStore) GetConnectionSecret(id string) (core.ConnectionRequest, error) {
	var ciphertext, nonce string
	err := s.store.db.QueryRow(`SELECT ciphertext, nonce FROM connection_secrets WHERE database_id=?`, id).Scan(&ciphertext, &nonce)
	if errors.Is(err, sql.ErrNoRows) {
		return core.ConnectionRequest{}, core.ErrNotFound
	}
	if err != nil {
		return core.ConnectionRequest{}, err
	}
	return s.decrypt(id, ciphertext, nonce)
}

func (s *connectionSecretStore) ListConnectionSecrets() (map[string]core.ConnectionRequest, error) {
	rows, err := s.store.db.Query(`SELECT database_id, ciphertext, nonce FROM connection_secrets ORDER BY database_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]core.ConnectionRequest{}
	for rows.Next() {
		var id, ciphertext, nonce string
		if err := rows.Scan(&id, &ciphertext, &nonce); err != nil {
			return nil, err
		}
		item, err := s.decrypt(id, ciphertext, nonce)
		if err != nil {
			return nil, err
		}
		items[id] = item
	}
	return items, rows.Err()
}

func (s *connectionSecretStore) DeleteConnectionSecret(id string) error {
	_, err := s.store.db.Exec(`DELETE FROM connection_secrets WHERE database_id=?`, id)
	return err
}

func (s *connectionSecretStore) decrypt(id, ciphertextRaw, nonceRaw string) (core.ConnectionRequest, error) {
	ciphertext, err := base64.RawStdEncoding.DecodeString(ciphertextRaw)
	if err != nil {
		return core.ConnectionRequest{}, fmt.Errorf("decode connection secret %q: %w", id, err)
	}
	nonce, err := base64.RawStdEncoding.DecodeString(nonceRaw)
	if err != nil {
		return core.ConnectionRequest{}, fmt.Errorf("decode connection secret nonce %q: %w", id, err)
	}
	plaintext, err := s.aead.Open(nil, nonce, ciphertext, []byte(id))
	if err != nil {
		return core.ConnectionRequest{}, fmt.Errorf("decrypt connection secret %q: check TOPBASE_MASTER_KEY: %w", id, err)
	}
	var item core.ConnectionRequest
	if err := json.Unmarshal(plaintext, &item); err != nil {
		return core.ConnectionRequest{}, err
	}
	item.ID = id
	return item, nil
}
