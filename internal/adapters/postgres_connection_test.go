package adapters

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestPostgresConnectedRequiresLiveTunnel(t *testing.T) {
	connector := NewPostgresConnector()
	done := make(chan struct{})
	connector.databases["pg_live"] = databaseConnection{db: &sql.DB{}, tunnel: &sshTunnel{done: done}}
	if !connector.Connected("pg_live") {
		t.Fatal("open tunnel should be connected")
	}
	close(done)
	if connector.Connected("pg_live") {
		t.Fatal("closed tunnel must not be reported as connected")
	}
}

func TestRecoverableConnectionError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "reset", err: errors.New("tls error: connection reset by peer"), want: true},
		{name: "eof", err: io.EOF, want: true},
		{name: "syntax", err: errors.New(`ERROR: syntax error at or near "FROM"`), want: false},
		{name: "cancel", err: context.Canceled, want: false},
		{name: "deadline", err: context.DeadlineExceeded, want: false},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := recoverableConnectionError(test.err); got != test.want {
				t.Fatalf("recoverableConnectionError(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

func TestConnectionLockIsStablePerDatabase(t *testing.T) {
	t.Parallel()
	connector := NewPostgresConnector()
	first := connector.connectionLock("pg_one")
	second := connector.connectionLock("pg_one")
	other := connector.connectionLock("pg_two")
	if first != second {
		t.Fatal("same database should share a reconnect lock")
	}
	if first == other {
		t.Fatal("different databases should not share a reconnect lock")
	}
}

func TestPostgresCatalogQueryReadsDatabaseMetadata(t *testing.T) {
	t.Parallel()
	for _, fragment := range []string{
		"obj_description",
		"col_description",
		"pg_get_expr",
		"indisprimary",
		"pg_constraint",
		"contype = 'f'",
	} {
		if !strings.Contains(postgresCatalogQuery, fragment) {
			t.Fatalf("catalog query does not read %q metadata", fragment)
		}
	}
}
