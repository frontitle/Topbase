package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/topbase/topbase/internal/adapters/appdb"
	"github.com/topbase/topbase/internal/buildinfo"
)

func (s *server) applicationDatabaseStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	completed, completedAt, err := s.store.ProductionMigrationCompleted()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	engine := s.store.Engine()
	writeJSON(w, http.StatusOK, map[string]any{
		"engine": engine, "mode": applicationDatabaseMode(engine), "schema_version": s.store.SchemaVersion(),
		"development_risk":    engine == appdb.EngineSQLite,
		"migration_available": engine == appdb.EngineSQLite && !completed,
		"migration_completed": completed, "migration_completed_at": completedAt,
	})
}

func (s *server) migrateApplicationDatabase(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		Engine    appdb.Engine `json:"engine"`
		DSN       string       `json:"dsn"`
		Host      string       `json:"host"`
		Port      int          `json:"port"`
		Database  string       `json:"database"`
		Schema    string       `json:"schema"`
		Username  string       `json:"username"`
		Password  string       `json:"password"`
		TLSMode   string       `json:"tls_mode"`
		CAFile    string       `json:"ca_file"`
		Confirmed bool         `json:"confirmed"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if !input.Confirmed {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请确认目标为空库并已先下载备份"})
		return
	}
	if input.Engine == appdb.EnginePostgres {
		if input.Port == 0 {
			input.Port = 5432
		}
		if input.Schema == "" {
			input.Schema = "public"
		}
	}
	if input.Engine == appdb.EngineMySQL && input.Port == 0 {
		input.Port = 3306
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()
	report, err := s.store.MigrateSQLiteToProduction(ctx, appdb.Config{
		Engine: input.Engine, DSN: input.DSN, Host: input.Host, Port: input.Port, Database: input.Database,
		Schema: input.Schema, Username: input.Username, Password: input.Password, TLSMode: input.TLSMode,
		CAFile: input.CAFile, AppVersion: buildinfo.Version, ConnectTimeout: 15 * time.Second,
		MaxOpenConns: 4, MaxIdleConns: 1,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"report":    report,
		"next_step": "将相同连接参数写入部署环境变量，保持 TOPBASE_MASTER_KEY 不变，然后重启所有 Topbase 节点。当前进程仍使用 SQLite。",
	})
}

func (s *server) exportApplicationDatabase(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	temporary, err := os.CreateTemp("", "topbase-logical-backup-*.zip")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	name := temporary.Name()
	defer os.Remove(name)
	defer temporary.Close()
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	if _, err := s.store.ExportLogical(ctx, temporary, buildinfo.Version); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "export application database: " + err.Error()})
		return
	}
	if _, err := temporary.Seek(0, 0); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	info, err := temporary.Stat()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	filename := "topbase-backup-" + time.Now().UTC().Format("20060102T150405Z") + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, filename, info.ModTime(), temporary)
}

func applicationDatabaseMode(engine appdb.Engine) string {
	if engine == appdb.EngineSQLite {
		return "development"
	}
	return "production"
}
