package adapters

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/topbase/topbase/internal/core"
)

type fakeSSHClient struct {
	keepaliveErr error
	wait         chan struct{}
	closed       chan struct{}
	closeOnce    sync.Once
}

func (f *fakeSSHClient) Dial(string, string) (net.Conn, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeSSHClient) SendRequest(string, bool, []byte) (bool, []byte, error) {
	return true, nil, f.keepaliveErr
}

func (f *fakeSSHClient) Wait() error {
	select {
	case <-f.wait:
	case <-f.closed:
	}
	return net.ErrClosed
}

func (f *fakeSSHClient) Close() error {
	f.closeOnce.Do(func() { close(f.closed) })
	return nil
}

func TestSSHTunnelRequiresHostAndUsername(t *testing.T) {
	_, _, err := openSSHTunnel(context.Background(), core.SSHTunnelRequest{
		Host: "bastion.example.com", Password: "secret",
	}, "database.internal", 5432)
	if err == nil || !strings.Contains(err.Error(), "host and username") {
		t.Fatalf("expected host and username validation error, got %v", err)
	}
}

func TestSSHAuthRequiresCredential(t *testing.T) {
	_, err := sshAuth(core.SSHTunnelRequest{})
	if err == nil || !strings.Contains(err.Error(), "password or private key") {
		t.Fatalf("expected credential validation error, got %v", err)
	}
}

func TestSSHTunnelKeepaliveMarksDeadConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	client := &fakeSSHClient{keepaliveErr: errors.New("connection lost"), wait: make(chan struct{}), closed: make(chan struct{})}
	tunnel := newSSHTunnel(listener, client, "database.internal", 5432, time.Millisecond, 100*time.Millisecond)
	t.Cleanup(func() { _ = tunnel.Close() })

	deadline := time.Now().Add(time.Second)
	for tunnel.Alive() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if tunnel.Alive() {
		t.Fatal("tunnel should be marked dead after keepalive failure")
	}
}

func TestSSHTunnelMonitorMarksClosedConnectionDead(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	wait := make(chan struct{})
	client := &fakeSSHClient{wait: wait, closed: make(chan struct{})}
	tunnel := newSSHTunnel(listener, client, "database.internal", 5432, time.Hour, time.Second)
	close(wait)

	deadline := time.Now().Add(time.Second)
	for tunnel.Alive() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if tunnel.Alive() {
		t.Fatal("tunnel should be marked dead when SSH Wait returns")
	}
}
