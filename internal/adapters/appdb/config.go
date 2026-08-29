package appdb

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

type Engine string

const (
	EngineSQLite   Engine = "sqlite"
	EnginePostgres Engine = "postgres"
	EngineMySQL    Engine = "mysql"
)

type Config struct {
	Engine          Engine
	DSN             string
	Path            string
	Host            string
	Port            int
	Database        string
	Schema          string
	Username        string
	Password        string
	TLSMode         string
	CAFile          string
	AppVersion      string
	ConnectTimeout  time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func ConfigFromEnv(dataDir, appVersion string) (Config, error) {
	engine, err := normalizeEngine(os.Getenv("TOPBASE_APP_DB_ENGINE"))
	if err != nil {
		return Config{}, err
	}
	if engine == "" {
		engine = EngineSQLite
	}
	cfg := Config{
		Engine:          engine,
		DSN:             strings.TrimSpace(os.Getenv("TOPBASE_APP_DB_DSN")),
		Path:            dataDir + string(os.PathSeparator) + "app.db",
		Host:            strings.TrimSpace(os.Getenv("TOPBASE_APP_DB_HOST")),
		Database:        strings.TrimSpace(os.Getenv("TOPBASE_APP_DB_NAME")),
		Schema:          strings.TrimSpace(os.Getenv("TOPBASE_APP_DB_SCHEMA")),
		Username:        strings.TrimSpace(os.Getenv("TOPBASE_APP_DB_USER")),
		Password:        os.Getenv("TOPBASE_APP_DB_PASSWORD"),
		TLSMode:         strings.ToLower(strings.TrimSpace(os.Getenv("TOPBASE_APP_DB_TLS_MODE"))),
		CAFile:          strings.TrimSpace(os.Getenv("TOPBASE_APP_DB_CA_FILE")),
		AppVersion:      appVersion,
		ConnectTimeout:  envDuration("TOPBASE_APP_DB_CONNECT_TIMEOUT", 10*time.Second),
		MaxOpenConns:    envInt("TOPBASE_APP_DB_MAX_OPEN_CONNS", 20),
		MaxIdleConns:    envInt("TOPBASE_APP_DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: envDuration("TOPBASE_APP_DB_CONN_MAX_LIFETIME", 30*time.Minute),
	}
	if raw := strings.TrimSpace(os.Getenv("TOPBASE_APP_DB_PORT")); raw != "" {
		cfg.Port, err = strconv.Atoi(raw)
		if err != nil || cfg.Port <= 0 || cfg.Port > 65535 {
			return Config{}, fmt.Errorf("TOPBASE_APP_DB_PORT must be a valid TCP port")
		}
	}
	if cfg.Engine == EnginePostgres {
		if cfg.Port == 0 {
			cfg.Port = 5432
		}
		if cfg.Schema == "" {
			cfg.Schema = "public"
		}
	}
	if cfg.Engine == EngineMySQL && cfg.Port == 0 {
		cfg.Port = 3306
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func normalizeEngine(raw string) (Engine, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return "", nil
	case "sqlite", "sqlite3":
		return EngineSQLite, nil
	case "postgres", "postgresql", "pgsql":
		return EnginePostgres, nil
	case "mysql":
		return EngineMySQL, nil
	default:
		return "", fmt.Errorf("unsupported TOPBASE_APP_DB_ENGINE %q; use postgres or mysql", raw)
	}
}

func (c Config) Validate() error {
	if c.Engine != EngineSQLite && c.Engine != EnginePostgres && c.Engine != EngineMySQL {
		return fmt.Errorf("unsupported application database engine %q", c.Engine)
	}
	if c.Engine == EngineSQLite {
		if c.Path == "" && c.DSN == "" {
			return errors.New("sqlite application database path is required")
		}
		return nil
	}
	if c.DSN == "" {
		missing := make([]string, 0, 4)
		if c.Host == "" {
			missing = append(missing, "TOPBASE_APP_DB_HOST")
		}
		if c.Database == "" {
			missing = append(missing, "TOPBASE_APP_DB_NAME")
		}
		if c.Username == "" {
			missing = append(missing, "TOPBASE_APP_DB_USER")
		}
		if len(missing) > 0 {
			return fmt.Errorf("application database configuration is incomplete: %s", strings.Join(missing, ", "))
		}
	}
	if c.Engine == EnginePostgres && !validIdentifier(c.Schema) {
		return fmt.Errorf("invalid PostgreSQL schema %q", c.Schema)
	}
	return nil
}

func (c Config) driverAndDSN() (string, string, error) {
	if c.Engine == EngineSQLite {
		if c.DSN != "" {
			return "sqlite", c.DSN, nil
		}
		return "sqlite", "file:" + strings.ReplaceAll(c.Path, "\\", "/") + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", nil
	}
	if c.DSN != "" {
		if c.Engine == EnginePostgres {
			return "pgx", c.DSN, nil
		}
		return "mysql", c.DSN, nil
	}
	if c.Engine == EnginePostgres {
		u := &url.URL{Scheme: "postgres", Host: net.JoinHostPort(c.Host, strconv.Itoa(c.Port)), Path: "/" + c.Database}
		u.User = url.UserPassword(c.Username, c.Password)
		q := u.Query()
		mode := c.TLSMode
		if mode == "" {
			mode = "prefer"
		}
		if mode == "verify-identity" {
			mode = "verify-full"
		}
		q.Set("sslmode", mode)
		if c.CAFile != "" {
			q.Set("sslrootcert", c.CAFile)
		}
		q.Set("connect_timeout", strconv.Itoa(max(1, int(c.ConnectTimeout.Seconds()))))
		q.Set("search_path", c.Schema)
		u.RawQuery = q.Encode()
		return "pgx", u.String(), nil
	}
	mc := mysqlDriver.NewConfig()
	mc.User = c.Username
	mc.Passwd = c.Password
	mc.Net = "tcp"
	mc.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	mc.DBName = c.Database
	mc.Timeout = c.ConnectTimeout
	mc.ReadTimeout = 30 * time.Second
	mc.WriteTimeout = 30 * time.Second
	mc.Collation = "utf8mb4_unicode_ci"
	mc.Params = map[string]string{"charset": "utf8mb4"}
	tlsName, err := mysqlTLSName(c)
	if err != nil {
		return "", "", err
	}
	mc.TLSConfig = tlsName
	return "mysql", mc.FormatDSN(), nil
}

var mysqlTLSConfigs sync.Map

func mysqlTLSName(c Config) (string, error) {
	mode := strings.ToLower(c.TLSMode)
	if mode == "" {
		mode = "preferred"
	}
	switch mode {
	case "disable", "disabled", "false":
		return "false", nil
	case "preferred":
		return "preferred", nil
	case "require", "required", "true":
		return "true", nil
	case "verify-ca", "verify-full", "verify-identity":
	default:
		return "", fmt.Errorf("unsupported MySQL TLS mode %q", c.TLSMode)
	}
	if c.CAFile == "" {
		return "", fmt.Errorf("TOPBASE_APP_DB_CA_FILE is required for MySQL TLS mode %s", mode)
	}
	pem, err := os.ReadFile(c.CAFile)
	if err != nil {
		return "", fmt.Errorf("read application database CA: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return "", errors.New("application database CA file does not contain a PEM certificate")
	}
	hash := sha256.Sum256(append([]byte(c.Host+"\x00"+mode+"\x00"), pem...))
	name := "topbase-" + hex.EncodeToString(hash[:8])
	if _, loaded := mysqlTLSConfigs.LoadOrStore(name, struct{}{}); loaded {
		return name, nil
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}
	if mode == "verify-full" || mode == "verify-identity" {
		tlsConfig.ServerName = c.Host
	} else {
		tlsConfig.InsecureSkipVerify = true // Verification is performed below without a DNS name.
		tlsConfig.VerifyConnection = func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return errors.New("database server did not provide a TLS certificate")
			}
			_, err := state.PeerCertificates[0].Verify(x509.VerifyOptions{
				Roots: roots, Intermediates: intermediatePool(state.PeerCertificates[1:]),
			})
			return err
		}
	}
	if err := mysqlDriver.RegisterTLSConfig(name, tlsConfig); err != nil {
		mysqlTLSConfigs.Delete(name)
		return "", fmt.Errorf("register MySQL TLS configuration: %w", err)
	}
	return name, nil
}

func intermediatePool(certs []*x509.Certificate) *x509.CertPool {
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}
	return pool
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func envDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
