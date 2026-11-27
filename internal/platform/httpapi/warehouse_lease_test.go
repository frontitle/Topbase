package httpapi

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/frontitle/Topbase/internal/adapters/appdb"
)

func TestOverlappingWarehouseRunsKeepTheFirstLease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	open := func(id string) *server {
		store, err := appdb.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { store.Close() })
		return &server{store: store, instanceID: id}
	}
	first, other := open("first"), open("other")
	reserve := func(s *server) func() {
		release, err := s.reserveWarehouseRun(context.Background(), "orders")
		if err != nil {
			t.Fatal(err)
		}
		return release
	}
	active := reserve(first)
	if active == nil {
		t.Fatal("first run could not reserve lease")
	}
	if reserve(first) != nil || reserve(other) != nil {
		t.Fatal("overlapping attempt obtained the active schedule lease")
	}
	active()
	retry := reserve(other)
	if retry == nil {
		t.Fatal("retry on a second node could not obtain the released lease")
	}
	retry()
}
