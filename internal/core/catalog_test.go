package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type memDBStore struct{ items map[string]Database }

func (m *memDBStore) List() ([]Database, error) {
	out := []Database{}
	for _, item := range m.items {
		out = append(out, item)
	}
	return out, nil
}
func (m *memDBStore) Save(item Database) error {
	if m.items == nil {
		m.items = map[string]Database{}
	}
	m.items[item.ID] = item
	return nil
}
func (m *memDBStore) Delete(id string) error {
	delete(m.items, id)
	return nil
}

type memSecrets struct{ items map[string]ConnectionRequest }

func (m *memSecrets) SaveConnectionSecret(id string, value ConnectionRequest) error {
	if m.items == nil {
		m.items = map[string]ConnectionRequest{}
	}
	value.ID = id
	m.items[id] = value
	return nil
}
func (m *memSecrets) GetConnectionSecret(id string) (ConnectionRequest, error) {
	item, ok := m.items[id]
	if !ok {
		return ConnectionRequest{}, ErrNotFound
	}
	return item, nil
}
func (m *memSecrets) ListConnectionSecrets() (map[string]ConnectionRequest, error) {
	return m.items, nil
}
func (m *memSecrets) DeleteConnectionSecret(id string) error {
	delete(m.items, id)
	return nil
}

type fakeConnector struct {
	connected map[string]bool
	tables    []Table
	connects  int
}

func (f *fakeConnector) Connect(_ context.Context, input ConnectionRequest) (Database, error) {
	if f.connected == nil {
		f.connected = map[string]bool{}
	}
	id := input.ID
	if id == "" {
		id = "pg_tmp"
	}
	f.connected[id] = true
	f.connects++
	return Database{ID: id, Name: input.Name, Engine: "postgres", Host: input.Host, Status: "connected"}, nil
}
func (f *fakeConnector) Tables(context.Context, string) ([]Table, error) { return f.tables, nil }
func (f *fakeConnector) Connected(id string) bool                        { return f.connected[id] }
func (f *fakeConnector) Close(id string) error {
	delete(f.connected, id)
	return nil
}

func TestSyncReconnectsFromSavedSecret(t *testing.T) {
	conn := &fakeConnector{tables: []Table{{Schema: "public", Name: "orders", Columns: []Column{{Name: "id", DataType: "integer"}}}}}
	secrets := &memSecrets{items: map[string]ConnectionRequest{
		"pg_1": {Name: "biz", Engine: "postgres", Host: "db.local", Database: "app", Username: "u", Password: "secret"},
	}}
	svc := CatalogService{
		Store:     &memDBStore{items: map[string]Database{"pg_1": {ID: "pg_1", Name: "biz"}}},
		Connector: conn,
		Secrets:   secrets,
		Snapshots: nil,
	}
	if conn.Connected("pg_1") {
		t.Fatal("should start disconnected")
	}
	snap, err := svc.Sync(context.Background(), "pg_1")
	if err != nil {
		t.Fatal(err)
	}
	if !conn.Connected("pg_1") {
		t.Fatal("sync should reconnect")
	}
	if len(snap.Tables) != 1 || snap.Tables[0].Name != "orders" {
		t.Fatalf("%+v", snap.Tables)
	}
}

func TestUpdateKeepsPasswordWhenBlank(t *testing.T) {
	conn := &fakeConnector{connected: map[string]bool{"pg_1": true}}
	secrets := &memSecrets{items: map[string]ConnectionRequest{
		"pg_1": {Name: "old", Engine: "postgres", Host: "db.local", Port: 5432, Database: "app", Username: "u", Password: "secret"},
	}}
	svc := CatalogService{
		Store:     &memDBStore{items: map[string]Database{"pg_1": {ID: "pg_1", Name: "old"}}},
		Connector: conn,
		Secrets:   secrets,
	}
	saved, err := svc.Update(context.Background(), "pg_1", ConnectionRequest{Name: "new", Engine: "postgres", Host: "db.local", Port: 5432, Database: "app", Username: "u"})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Name != "new" {
		t.Fatalf("%+v", saved)
	}
	if secrets.items["pg_1"].Password != "secret" {
		t.Fatalf("password %+v", secrets.items["pg_1"])
	}
}

func TestLiveStatusDoesNotTreatSavedSecretAsLiveConnection(t *testing.T) {
	svc := CatalogService{
		Connector: &fakeConnector{},
		Secrets:   &memSecrets{items: map[string]ConnectionRequest{"pg_1": {Name: "biz", Host: "db.local"}}},
	}
	item := svc.LiveStatus(Database{ID: "pg_1", Name: "biz", Status: "disconnected"})
	if item.Connected || item.Status != "disconnected" {
		t.Fatalf("%+v", item)
	}
}

func TestEnsureConnectedRequiresSavedSecret(t *testing.T) {
	svc := CatalogService{
		Store:     &memDBStore{items: map[string]Database{"pg_1": {ID: "pg_1", Name: "biz"}}},
		Connector: &fakeConnector{},
		Secrets:   &memSecrets{},
	}
	err := svc.EnsureConnected(context.Background(), "pg_1")
	if err == nil || err.Error() == "" {
		t.Fatal("expected reconnect error")
	}
}

func TestConnectionSettingsReturnsSavedSecrets(t *testing.T) {
	svc := CatalogService{Secrets: &memSecrets{items: map[string]ConnectionRequest{
		"pg_1": {Name: "biz", Username: "reader", Password: "secret", Host: "db.local", Database: "app", SSH: &SSHTunnelRequest{Host: "bastion", PrivateKey: "KEY", Password: "ssh"}},
	}}}
	item, err := svc.ConnectionSettings("pg_1")
	if err != nil {
		t.Fatal(err)
	}
	if item.Username != "reader" || item.Password != "secret" || item.Host != "db.local" || item.Database != "app" {
		t.Fatalf("edit form missing saved connection %+v", item)
	}
	if item.SSH.Host != "bastion" || item.SSH.PrivateKey != "KEY" {
		t.Fatalf("%+v", item.SSH)
	}
}

func TestCatalogMetadataJSONIncludesSourceDescriptionsAndRelationships(t *testing.T) {
	table := Table{
		Schema:      "public",
		Name:        "orders",
		Description: "订单事实表",
		Columns: []Column{{
			Name:         "customer_id",
			DataType:     "bigint",
			Description:  "客户编号",
			DefaultValue: "nextval('orders_customer_id_seq'::regclass)",
			PrimaryKey:   true,
			ForeignKey:   &FieldRef{Schema: "public", Table: "customers", Name: "id"},
		}},
	}
	raw, err := json.Marshal(table)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"订单事实表", "客户编号", "default_value", "primary_key", "foreign_key", "customers"} {
		if !strings.Contains(string(raw), fragment) {
			t.Fatalf("catalog metadata JSON is missing %q: %s", fragment, raw)
		}
	}
}
