package adapters

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/topbase/topbase/internal/core"
	"golang.org/x/crypto/ssh"
)

type sshTunnel struct {
	listener net.Listener
	client   sshTunnelClient
	done     chan struct{}
	once     sync.Once
}

type sshTunnelClient interface {
	Dial(string, string) (net.Conn, error)
	SendRequest(string, bool, []byte) (bool, []byte, error)
	Wait() error
	Close() error
}

const (
	sshKeepaliveInterval = 20 * time.Second
	sshKeepaliveTimeout  = 8 * time.Second
)

func openSSHTunnel(ctx context.Context, config core.SSHTunnelRequest, targetHost string, targetPort int) (*sshTunnel, string, error) {
	if strings.TrimSpace(config.Host) == "" || strings.TrimSpace(config.Username) == "" {
		return nil, "", fmt.Errorf("SSH host and username are required")
	}
	if targetHost == "" || targetPort == 0 {
		return nil, "", fmt.Errorf("database host and port are required when SSH tunneling is enabled")
	}
	auth, err := sshAuth(config)
	if err != nil {
		return nil, "", err
	}
	port := config.Port
	if port == 0 {
		port = 22
	}
	clientConfig := &ssh.ClientConfig{
		User: config.Username, Auth: auth,
		HostKeyCallback: verifyFingerprint(config.HostKeyFingerprint),
		Timeout:         8 * time.Second,
		AuthCallback:    retryAuthWhenServerOmitsMethods(config.AuthenticationType, auth),
	}
	dialer := net.Dialer{}
	sshConn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(config.Host, fmt.Sprint(port)))
	if err != nil {
		return nil, "", fmt.Errorf("connect SSH bastion: %w", err)
	}
	conn, channels, requests, err := ssh.NewClientConn(sshConn, net.JoinHostPort(config.Host, fmt.Sprint(port)), clientConfig)
	if err != nil {
		if config.AuthenticationType == "password" && strings.Contains(err.Error(), "attempted methods [none]") {
			return nil, "", fmt.Errorf("SSH server does not accept password authentication; it requires a public key. Select “私钥” and provide the private key authorized for this bastion")
		}
		return nil, "", fmt.Errorf("authenticate SSH bastion: %w", err)
	}
	client := ssh.NewClient(conn, channels, requests)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return nil, "", fmt.Errorf("open local SSH tunnel: %w", err)
	}
	tunnel := newSSHTunnel(listener, client, targetHost, targetPort, sshKeepaliveInterval, sshKeepaliveTimeout)
	return tunnel, listener.Addr().String(), nil
}

func newSSHTunnel(listener net.Listener, client sshTunnelClient, targetHost string, targetPort int, keepaliveInterval, keepaliveTimeout time.Duration) *sshTunnel {
	tunnel := &sshTunnel{listener: listener, client: client, done: make(chan struct{})}
	go tunnel.accept(targetHost, targetPort)
	go tunnel.monitor()
	go tunnel.keepalive(keepaliveInterval, keepaliveTimeout)
	return tunnel
}

// A few managed SSH gateways incorrectly return an empty allowed-method list
// after the initial "none" request. AuthCallback lets us still offer the
// credentials selected by the user once, instead of failing with "[none]".
func retryAuthWhenServerOmitsMethods(authenticationType string, auth []ssh.AuthMethod) ssh.ClientAuthCallback {
	names := []string{"publickey"}
	if authenticationType == "password" || len(auth) > 1 {
		names = []string{"password", "keyboard-interactive"}
	}
	return func(context *ssh.ClientAuthContext) (ssh.AuthMethod, error) {
		if len(context.AllowedMethods) != 0 {
			return nil, nil
		}
		for index, method := range auth {
			name := names[index]
			if !contains(context.TriedMethods, name) {
				return method, nil
			}
		}
		return nil, nil
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func sshAuth(config core.SSHTunnelRequest) ([]ssh.AuthMethod, error) {
	if config.AuthenticationType != "password" && strings.TrimSpace(config.PrivateKey) != "" {
		var signer ssh.Signer
		var err error
		if config.PrivateKeyPassword != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(config.PrivateKey), []byte(config.PrivateKeyPassword))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(config.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("read SSH private key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	if config.AuthenticationType == "key" {
		return nil, fmt.Errorf("provide an SSH private key")
	}
	if config.Password == "" {
		return nil, fmt.Errorf("provide an SSH password or private key")
	}
	keyboardInteractive := ssh.KeyboardInteractive(func(_ string, _ string, questions []string, _ []bool) ([]string, error) {
		answers := make([]string, len(questions))
		for i := range answers {
			answers[i] = config.Password
		}
		return answers, nil
	})
	return []ssh.AuthMethod{ssh.Password(config.Password), keyboardInteractive}, nil
}

func verifyFingerprint(expected string) ssh.HostKeyCallback {
	if strings.TrimSpace(expected) == "" {
		return ssh.InsecureIgnoreHostKey()
	}
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		actual := ssh.FingerprintSHA256(key)
		if actual != strings.TrimSpace(expected) {
			return fmt.Errorf("SSH host-key fingerprint mismatch: expected %s, got %s", expected, actual)
		}
		return nil
	}
}

func (t *sshTunnel) accept(targetHost string, targetPort int) {
	for {
		local, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.done:
				return
			default:
				return
			}
		}
		go func() {
			defer local.Close()
			remote, err := t.client.Dial("tcp", net.JoinHostPort(targetHost, fmt.Sprint(targetPort)))
			if err != nil {
				return
			}
			defer remote.Close()
			copyDone := make(chan struct{}, 2)
			go func() { _, _ = io.Copy(remote, local); copyDone <- struct{}{} }()
			go func() { _, _ = io.Copy(local, remote); copyDone <- struct{}{} }()
			<-copyDone
		}()
	}
}

func (t *sshTunnel) monitor() {
	_ = t.client.Wait()
	_ = t.Close()
}

func (t *sshTunnel) keepalive(interval, timeout time.Duration) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-t.done:
			return
		case <-ticker.C:
			result := make(chan error, 1)
			go func() {
				_, _, err := t.client.SendRequest("keepalive@openssh.com", true, nil)
				result <- err
			}()
			select {
			case <-t.done:
				return
			case err := <-result:
				if err != nil {
					_ = t.Close()
					return
				}
			case <-time.After(timeout):
				_ = t.Close()
				return
			}
		}
	}
}

func (t *sshTunnel) Alive() bool {
	if t == nil {
		return false
	}
	select {
	case <-t.done:
		return false
	default:
		return true
	}
}

func (t *sshTunnel) Close() error {
	var err error
	t.once.Do(func() {
		close(t.done)
		_ = t.listener.Close()
		err = t.client.Close()
	})
	return err
}
