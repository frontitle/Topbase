package core

import (
	"context"
	"fmt"
	"time"
)

type ConnectionRequest struct {
	// ID is the catalog id. Add ignores a client-supplied value; Update/Test use it to merge secrets.
	ID       string            `json:"id,omitempty"`
	Name     string            `json:"name"`
	Engine   string            `json:"engine"`
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Database string            `json:"database"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	SSLMode  string            `json:"ssl_mode"`
	DSN      string            `json:"dsn"`
	SSH      *SSHTunnelRequest `json:"ssh,omitempty"`
	ClearSSH bool              `json:"clear_ssh,omitempty"`
}

// SSHTunnelRequest describes the bastion host, never the target database.
// HostKeyFingerprint is optional. When supplied, it must be the SHA256
// fingerprint shown by the SSH server and will be verified.
type SSHTunnelRequest struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Username           string `json:"username"`
	AuthenticationType string `json:"authentication_type,omitempty"`
	Password           string `json:"password,omitempty"`
	PrivateKey         string `json:"private_key,omitempty"`
	PrivateKeyPassword string `json:"private_key_password,omitempty"`
	HostKeyFingerprint string `json:"host_key_fingerprint"`
}

type Column struct {
	Name         string    `json:"name"`
	DataType     string    `json:"data_type"`
	Nullable     bool      `json:"nullable"`
	Description  string    `json:"description,omitempty"`
	DefaultValue string    `json:"default_value,omitempty"`
	PrimaryKey   bool      `json:"primary_key,omitempty"`
	ForeignKey   *FieldRef `json:"foreign_key,omitempty"`
}

type Table struct {
	Schema      string   `json:"schema"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Columns     []Column `json:"columns"`
}

type DatabaseConnector interface {
	Connect(context.Context, ConnectionRequest) (Database, error)
	Tables(context.Context, string) ([]Table, error)
	Connected(id string) bool
	Close(string) error
}

type DatabaseStore interface {
	List() ([]Database, error)
	Save(Database) error
	Delete(string) error
}

// ConnectionSecretStore stores the connection settings required to reopen a
// catalog entry. Production implementations should use a managed secret store.
type ConnectionSecretStore interface {
	SaveConnectionSecret(string, ConnectionRequest) error
	GetConnectionSecret(id string) (ConnectionRequest, error)
	ListConnectionSecrets() (map[string]ConnectionRequest, error)
	DeleteConnectionSecret(string) error
}

type SchemaSnapshot struct {
	DatabaseID string    `json:"database_id"`
	Tables     []Table   `json:"tables"`
	SyncedAt   time.Time `json:"synced_at"`
}

type SchemaSnapshotStore interface {
	SaveSnapshot(SchemaSnapshot) error
	GetSnapshot(databaseID string) (SchemaSnapshot, error)
	DeleteSnapshot(databaseID string) error
}

type CatalogService struct {
	Store     DatabaseStore
	Connector DatabaseConnector
	Secrets   ConnectionSecretStore
	Snapshots SchemaSnapshotStore
}

func (s CatalogService) Add(ctx context.Context, input ConnectionRequest) (Database, error) {
	input.ID = ""
	database, err := s.Connector.Connect(ctx, input)
	if err != nil {
		return Database{}, err
	}
	if database.CreatedAt.IsZero() {
		database.CreatedAt = time.Now().UTC()
	}
	if err := s.Store.Save(database); err != nil {
		_ = s.Connector.Close(database.ID)
		return Database{}, err
	}
	if s.Secrets != nil {
		input.ID = database.ID
		if err := s.Secrets.SaveConnectionSecret(database.ID, input); err != nil {
			_ = s.Store.Delete(database.ID)
			_ = s.Connector.Close(database.ID)
			return Database{}, err
		}
	}
	if snap, err := s.Sync(ctx, database.ID); err == nil {
		database.LastSyncedAt = &snap.SyncedAt
		database.TableCount = len(snap.Tables)
	}
	database.Connected = s.Connector != nil && s.Connector.Connected(database.ID)
	return database, nil
}

// Test validates every network hop without writing catalog metadata.
func (s CatalogService) Test(ctx context.Context, input ConnectionRequest) error {
	id := input.ID
	if id != "" {
		merged, err := s.mergeSecret(id, input)
		if err != nil {
			return err
		}
		input = merged
	}
	input.ID = ""
	database, err := s.Connector.Connect(ctx, input)
	if err != nil {
		return err
	}
	return s.Connector.Close(database.ID)
}

func (s CatalogService) Remove(id string) error {
	if err := s.Connector.Close(id); err != nil {
		return err
	}
	if s.Secrets != nil {
		if err := s.Secrets.DeleteConnectionSecret(id); err != nil {
			return err
		}
	}
	if s.Snapshots != nil {
		_ = s.Snapshots.DeleteSnapshot(id)
	}
	return s.Store.Delete(id)
}

func (s CatalogService) Snapshot(databaseID string) (SchemaSnapshot, error) {
	if s.Snapshots == nil {
		return SchemaSnapshot{}, fmt.Errorf("schema snapshot store is not configured")
	}
	return s.Snapshots.GetSnapshot(databaseID)
}

func (s CatalogService) Sync(ctx context.Context, databaseID string) (SchemaSnapshot, error) {
	if s.Connector == nil {
		return SchemaSnapshot{}, fmt.Errorf("database connector is not configured")
	}
	if err := s.EnsureConnected(ctx, databaseID); err != nil {
		return SchemaSnapshot{}, err
	}
	tables, err := s.Connector.Tables(ctx, databaseID)
	if err != nil {
		return SchemaSnapshot{}, err
	}
	snap := SchemaSnapshot{DatabaseID: databaseID, Tables: tables, SyncedAt: time.Now().UTC()}
	if s.Snapshots != nil {
		if err := s.Snapshots.SaveSnapshot(snap); err != nil {
			return SchemaSnapshot{}, err
		}
	}
	return snap, nil
}

func (s CatalogService) EnsureConnected(ctx context.Context, databaseID string) error {
	if s.Connector == nil {
		return fmt.Errorf("database connector is not configured")
	}
	if s.Connector.Connected(databaseID) {
		return nil
	}
	if s.Secrets == nil {
		return fmt.Errorf("database %q is not connected; edit the saved connection and try again", databaseID)
	}
	input, err := s.Secrets.GetConnectionSecret(databaseID)
	if err != nil {
		return fmt.Errorf("no saved connection for %q; edit the connection settings and save again", databaseID)
	}
	input.ID = databaseID
	if _, err := s.Connector.Connect(ctx, input); err != nil {
		return fmt.Errorf("reconnect failed: %w", err)
	}
	return nil
}

func (s CatalogService) Update(ctx context.Context, id string, input ConnectionRequest) (Database, error) {
	existing, err := s.get(id)
	if err != nil {
		return Database{}, err
	}
	merged, err := s.mergeSecret(id, input)
	if err != nil {
		return Database{}, err
	}
	merged.ID = id
	database, err := s.Connector.Connect(ctx, merged)
	if err != nil {
		return Database{}, err
	}
	database.ID = id
	database.CreatedAt = existing.CreatedAt
	if err := s.Store.Save(database); err != nil {
		return Database{}, err
	}
	if s.Secrets != nil {
		if err := s.Secrets.SaveConnectionSecret(id, merged); err != nil {
			return Database{}, err
		}
	}
	if snap, err := s.Sync(ctx, id); err == nil {
		database.LastSyncedAt = &snap.SyncedAt
		database.TableCount = len(snap.Tables)
	}
	database.Connected = s.Connector.Connected(id)
	return database, nil
}

func (s CatalogService) ConnectionSettings(id string) (ConnectionRequest, error) {
	if s.Secrets == nil {
		return ConnectionRequest{}, fmt.Errorf("connection secrets are not configured")
	}
	item, err := s.Secrets.GetConnectionSecret(id)
	if err != nil {
		return ConnectionRequest{}, err
	}
	item.ID = id
	item.ClearSSH = false
	return item, nil
}

func (s CatalogService) LiveStatus(item Database) Database {
	inPool := s.Connector != nil && s.Connector.Connected(item.ID)
	item.Connected = inPool
	if item.Connected {
		item.Status = "connected"
	} else {
		item.Status = "disconnected"
	}
	return item
}

func (s CatalogService) get(id string) (Database, error) {
	items, err := s.Store.List()
	if err != nil {
		return Database{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return s.LiveStatus(item), nil
		}
	}
	return Database{}, ErrNotFound
}

func (s CatalogService) mergeSecret(id string, input ConnectionRequest) (ConnectionRequest, error) {
	if s.Secrets == nil {
		return input, nil
	}
	existing, err := s.Secrets.GetConnectionSecret(id)
	if err != nil {
		return input, nil
	}
	if input.Name == "" {
		input.Name = existing.Name
	}
	if input.Engine == "" {
		input.Engine = existing.Engine
	}
	if input.Password == "" {
		input.Password = existing.Password
	}
	if input.DSN == "" && input.Host == "" {
		input.DSN = existing.DSN
	}
	if input.ClearSSH {
		input.SSH = nil
	} else if input.SSH == nil {
		input.SSH = existing.SSH
	} else if existing.SSH != nil {
		if input.SSH.Password == "" {
			input.SSH.Password = existing.SSH.Password
		}
		if input.SSH.PrivateKey == "" {
			input.SSH.PrivateKey = existing.SSH.PrivateKey
		}
		if input.SSH.PrivateKeyPassword == "" {
			input.SSH.PrivateKeyPassword = existing.SSH.PrivateKeyPassword
		}
		if input.SSH.HostKeyFingerprint == "" {
			input.SSH.HostKeyFingerprint = existing.SSH.HostKeyFingerprint
		}
	}
	return input, nil
}

func (s CatalogService) RescanTable(ctx context.Context, databaseID, schema, table string) (SchemaSnapshot, error) {
	snap, err := s.Sync(ctx, databaseID)
	if err != nil {
		return SchemaSnapshot{}, err
	}
	found := false
	for _, item := range snap.Tables {
		if item.Schema == schema && item.Name == table {
			found = true
			break
		}
	}
	if !found {
		return SchemaSnapshot{}, fmt.Errorf("table %s.%s was not found after rescan", schema, table)
	}
	return snap, nil
}
