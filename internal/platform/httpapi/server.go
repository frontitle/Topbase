package httpapi

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/topbase/topbase/internal/adapters"
	"github.com/topbase/topbase/internal/adapters/appdb"
	"github.com/topbase/topbase/internal/adapters/feishu"
	"github.com/topbase/topbase/internal/app/content"
	"github.com/topbase/topbase/internal/app/identity"
	"github.com/topbase/topbase/internal/app/notify"
	appquery "github.com/topbase/topbase/internal/app/query"
	appwarehouse "github.com/topbase/topbase/internal/app/warehouse"
	"github.com/topbase/topbase/internal/buildinfo"
	"github.com/topbase/topbase/internal/core"
)

//go:embed web/*
var web embed.FS

const sessionCookie = "topbase_session"

type server struct {
	queries   core.QueryService
	ai        core.AIProvider
	catalog   core.CatalogService
	metadata  core.MetadataStore
	identity  identity.Service
	content   content.Service
	dataset   appquery.DatasetService
	warehouse *appwarehouse.Service
	notify    notify.Service
	static    fs.FS
	store     *appdb.Store
	connector *adapters.SQLConnector
	cancel    context.CancelFunc
	workers   sync.WaitGroup
}

type runtimeHandler struct {
	handler http.Handler
	close   func() error
}

func (h *runtimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.handler.ServeHTTP(w, r) }
func (h *runtimeHandler) Close() error                                     { return h.close() }

func dataDir() string {
	if dir := os.Getenv("TOPBASE_DATA_DIR"); dir != "" {
		return dir
	}
	return "data"
}

func NewServer() http.Handler {
	dir := dataDir()
	appPath := filepath.Join(dir, "app.db")
	store, err := appdb.OpenWithVersion(appPath, buildinfo.Version)
	if err != nil {
		panic("open application database: " + err.Error())
	}

	fileCatalog := adapters.NewFileCatalog(filepath.Join(dir, "catalog.json"))
	if items, err := fileCatalog.List(); err == nil {
		if err := store.ImportCatalog(items); err != nil {
			log.Printf("topbase: import catalog.json: %v", err)
		}
	}
	fileMeta := adapters.NewFileMetadataStore(filepath.Join(dir, "table-metadata.json"))
	if notes, err := fileMeta.All(); err == nil {
		for key, note := range notes {
			parts := strings.SplitN(key, "/", 3)
			if len(parts) != 3 {
				continue
			}
			_ = store.SaveTableAnnotation(parts[0], parts[1], parts[2], note)
		}
	}

	connector := adapters.NewSQLConnector()
	secrets := adapters.NewFileConnectionSecretStore(filepath.Join(dir, "connection-secrets.json"))
	queries := core.QueryService{Executor: connector}
	dataset := appquery.DatasetService{
		Queries: queries, Compile: connector.Compile,
		Expand: &appquery.Expander{
			Fields: store.Fields(), Models: store.Models(), Metrics: store.Metrics(),
			Segments: store.Segments(), Questions: store.Questions(),
		},
	}
	contentSvc := content.Service{
		Collections: store.Collections(), Questions: store.Questions(), Dashboards: store.Dashboards(),
		Bookmarks: store.Bookmarks(), Revisions: store.Revisions(), Alerts: store.Alerts(),
		Notifications: store.Notifications(), SearchStore: store,
		Fields: store.Fields(), Models: store.Models(), Metrics: store.Metrics(),
		Segments: store.Segments(), Glossary: store.Glossary(),
	}
	ident := identity.Service{
		Users: store.Users(), Groups: store.Groups(), Sessions: store.Sessions(),
		Settings: store, APIKeys: store.APIKeys(),
	}
	wh := &appwarehouse.Service{
		Schedules: store.Schedules(), Runs: store.Runs(), Tables: store.Materialized(),
		Edges: store.Lineage(), Questions: store.Questions(), Models: store.Models(), Writer: connector,
		Compile: adapters.CompilePostgresWarehouse,
	}
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	s := &server{
		queries:   queries,
		ai:        adapters.DemoAI{},
		catalog:   core.CatalogService{Store: store, Connector: connector, Secrets: secrets, Snapshots: store},
		metadata:  store,
		identity:  ident,
		content:   contentSvc,
		dataset:   dataset,
		warehouse: wh,
		notify: notify.Service{
			Content: contentSvc, Dataset: dataset, Subscriptions: store.Subscriptions(),
			Deliver: func(title, body, channel string) {
				if channel == "feishu" {
					if err := feishu.NotifyCard(title, body); err != nil {
						log.Printf("topbase: feishu notify: %v", err)
					}
				}
			},
		},
		store: store, connector: connector, cancel: cancelWorkers,
	}
	s.notify.Deliver = s.deliverNotification
	wh.Notify = func(title, body string) {
		_ = contentSvc.RecordNotification(core.Notification{Title: title, Body: body})
		if err := feishu.NotifyCard(title, body); err != nil {
			log.Printf("topbase: feishu notify: %v", err)
		}
	}
	s.restoreConnections()
	if os.Getenv("TOPBASE_CRON") != "off" {
		s.workers.Add(1)
		go func() {
			defer s.workers.Done()
			s.tickWarehouse(workerCtx)
		}()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/ready", s.readiness)
	mux.HandleFunc("GET /api/version", s.version)
	mux.HandleFunc("GET /api/setup/status", s.setupStatus)
	mux.HandleFunc("POST /api/setup", s.completeSetup)
	mux.HandleFunc("POST /api/session", s.createSession)
	mux.HandleFunc("GET /api/auth/options", s.authOptions)
	mux.HandleFunc("GET /api/admin/auth-settings", s.getAuthSettings)
	mux.HandleFunc("PUT /api/admin/auth-settings", s.saveAuthSettings)
	mux.HandleFunc("DELETE /api/session", s.deleteSession)
	mux.HandleFunc("GET /api/user/current", s.currentUser)
	mux.HandleFunc("GET /api/user/profile", s.getUserProfile)
	mux.HandleFunc("PUT /api/user/profile", s.updateUserProfile)
	mux.HandleFunc("PUT /api/user/password", s.changeUserPassword)
	mux.HandleFunc("DELETE /api/user/external-identities/{provider}", s.unbindUserExternalIdentity)
	mux.HandleFunc("GET /api/databases", s.databases)
	mux.HandleFunc("GET /api/database-engines", s.databaseEngines)
	mux.HandleFunc("POST /api/databases", s.connectDatabase)
	mux.HandleFunc("POST /api/databases/test", s.testDatabase)
	mux.HandleFunc("GET /api/databases/{id}", s.getDatabase)
	mux.HandleFunc("PUT /api/databases/{id}", s.updateDatabase)
	mux.HandleFunc("DELETE /api/databases/{id}", s.deleteDatabase)
	mux.HandleFunc("GET /api/databases/{id}/connection", s.getDatabaseConnection)
	mux.HandleFunc("POST /api/databases/{id}/test", s.testSavedDatabase)
	mux.HandleFunc("POST /api/databases/{id}/sync", s.syncDatabase)
	mux.HandleFunc("POST /api/databases/{id}/tables/{schema}/{table}/rescan", s.rescanTable)
	mux.HandleFunc("GET /api/databases/{id}/tables", s.tables)
	mux.HandleFunc("GET /api/databases/{id}/tables/{schema}/{table}/annotation", s.getAnnotation)
	mux.HandleFunc("PUT /api/databases/{id}/tables/{schema}/{table}/annotation", s.saveAnnotation)
	mux.HandleFunc("POST /api/databases/{id}/visual-query", s.visualQuery)
	mux.HandleFunc("POST /api/dataset", s.runDataset)
	mux.HandleFunc("POST /api/dataset/drill", s.drillDataset)
	mux.HandleFunc("GET /api/semantic-types", s.listSemanticTypes)
	mux.HandleFunc("GET /api/databases/{id}/tables/{schema}/{table}/fields", s.listFields)
	mux.HandleFunc("PUT /api/databases/{id}/tables/{schema}/{table}/fields", s.saveField)
	mux.HandleFunc("GET /api/models", s.listModels)
	mux.HandleFunc("POST /api/models", s.createModel)
	mux.HandleFunc("GET /api/models/{id}", s.getModel)
	mux.HandleFunc("GET /api/metrics", s.listMetrics)
	mux.HandleFunc("POST /api/metrics", s.createMetric)
	mux.HandleFunc("GET /api/segments", s.listSegments)
	mux.HandleFunc("POST /api/segments", s.createSegment)
	mux.HandleFunc("GET /api/glossary", s.listGlossary)
	mux.HandleFunc("POST /api/glossary", s.createGlossary)
	mux.HandleFunc("POST /api/queries/run", s.runQuery)
	mux.HandleFunc("POST /api/ai/chat", s.chat)
	mux.HandleFunc("GET /api/questions", s.listQuestions)
	mux.HandleFunc("POST /api/questions", s.createQuestion)
	mux.HandleFunc("GET /api/questions/{id}", s.getQuestion)
	mux.HandleFunc("PUT /api/questions/{id}", s.updateQuestion)
	mux.HandleFunc("DELETE /api/questions/{id}", s.archiveQuestion)
	mux.HandleFunc("POST /api/questions/{id}/export", s.exportQuestion)
	mux.HandleFunc("GET /api/collections", s.listCollections)
	mux.HandleFunc("POST /api/collections", s.createCollection)
	mux.HandleFunc("GET /api/collections/{id}", s.getCollection)
	mux.HandleFunc("PUT /api/collections/{id}", s.updateCollection)
	mux.HandleFunc("DELETE /api/collections/{id}", s.deleteCollection)
	mux.HandleFunc("GET /api/collections/{id}/shares", s.getCollectionShares)
	mux.HandleFunc("PUT /api/collections/{id}/shares", s.putCollectionShares)
	mux.HandleFunc("GET /api/dashboards", s.listDashboards)
	mux.HandleFunc("POST /api/dashboards", s.createDashboard)
	mux.HandleFunc("GET /api/dashboards/{id}", s.getDashboard)
	mux.HandleFunc("PUT /api/dashboards/{id}", s.updateDashboard)
	mux.HandleFunc("DELETE /api/dashboards/{id}", s.archiveDashboard)
	mux.HandleFunc("POST /api/dashboards/{id}/cards/{cardId}/dataset", s.runDashboardCard)
	mux.HandleFunc("POST /api/embed/validate", s.validateEmbedURL)
	mux.HandleFunc("POST /api/dashboards/{id}/public-link", s.enableDashboardPublicLink)
	mux.HandleFunc("DELETE /api/dashboards/{id}/public-link", s.disableDashboardPublicLink)
	mux.HandleFunc("POST /api/dashboards/{id}/copy", s.copyDashboard)
	mux.HandleFunc("GET /api/public/dashboard/{uuid}", s.getPublicDashboard)
	mux.HandleFunc("POST /api/public/dashboard/{uuid}/cards/{cardId}/dataset", s.runPublicDashboardCard)
	mux.HandleFunc("POST /api/dataset/export", s.exportDataset)
	mux.HandleFunc("GET /api/search", s.search)
	mux.HandleFunc("GET /api/bookmarks", s.listBookmarks)
	mux.HandleFunc("POST /api/bookmarks", s.createBookmark)
	mux.HandleFunc("DELETE /api/bookmarks/{id}", s.deleteBookmark)
	mux.HandleFunc("GET /api/revisions", s.listRevisions)
	mux.HandleFunc("GET /api/trash", s.listTrash)
	mux.HandleFunc("POST /api/trash/{type}/{id}/restore", s.restoreTrash)
	mux.HandleFunc("GET /api/alerts", s.listAlerts)
	mux.HandleFunc("POST /api/alerts", s.createAlert)
	mux.HandleFunc("DELETE /api/alerts/{id}", s.deleteAlert)
	mux.HandleFunc("POST /api/alerts/{id}/run", s.runAlert)
	mux.HandleFunc("GET /api/notifications", s.listNotifications)
	mux.HandleFunc("GET /api/users", s.listUsers)
	mux.HandleFunc("GET /api/shareable-users", s.listShareableUsers)
	mux.HandleFunc("POST /api/users", s.inviteUser)
	mux.HandleFunc("PATCH /api/users/{id}", s.setUserActive)
	mux.HandleFunc("POST /api/users/{id}/password", s.resetUserPassword)
	mux.HandleFunc("PUT /api/users/{id}/groups", s.replaceUserGroups)
	mux.HandleFunc("POST /api/users/{id}/external-identities", s.bindUserExternalIdentity)
	mux.HandleFunc("GET /api/settings", s.getSettings)
	mux.HandleFunc("PUT /api/settings", s.putSettings)
	mux.HandleFunc("GET /api/admin/settings", s.getAdminSettings)
	mux.HandleFunc("PUT /api/admin/settings", s.putAdminSettings)
	mux.HandleFunc("GET /api/admin/monitor", s.adminMonitor)
	mux.HandleFunc("GET /api/api-keys", s.listAPIKeys)
	mux.HandleFunc("POST /api/api-keys", s.createAPIKey)
	mux.HandleFunc("DELETE /api/api-keys/{id}", s.deleteAPIKey)
	mux.HandleFunc("GET /api/permissions/graph", s.getPermissionGraph)
	mux.HandleFunc("PUT /api/permissions/graph", s.putPermissionGraph)
	mux.HandleFunc("GET /api/schedules", s.listSchedules)
	mux.HandleFunc("POST /api/schedules", s.createSchedule)
	mux.HandleFunc("POST /api/schedules/{id}/run", s.runSchedule)
	mux.HandleFunc("GET /api/runs", s.listRuns)
	mux.HandleFunc("GET /api/warehouse/tables", s.listWarehouseTables)
	mux.HandleFunc("GET /api/lineage/{type}/{id}", s.listLineage)
	mux.HandleFunc("POST /api/ai/propose-schedule", s.proposeSchedule)
	mux.HandleFunc("GET /api/groups", s.listGroups)
	mux.HandleFunc("POST /api/groups", s.createGroup)
	mux.HandleFunc("PUT /api/groups/{id}/members", s.replaceGroupMembers)
	mux.HandleFunc("GET /api/projects/{id}/access", s.getProjectAccess)
	mux.HandleFunc("PUT /api/projects/{id}/access", s.putProjectAccess)
	mux.HandleFunc("GET /api/identity/providers", s.listIdentityProviders)
	mux.HandleFunc("PUT /api/identity/providers", s.saveIdentityProviders)
	mux.HandleFunc("GET /api/webhooks", s.listWebhooks)
	mux.HandleFunc("PUT /api/webhooks", s.saveWebhooks)
	mux.HandleFunc("GET /api/subscriptions", s.listAllSubscriptions)
	mux.HandleFunc("POST /api/feishu/departments/sync", s.syncFeishuDepartments)
	mux.HandleFunc("GET /api/dashboards/{id}/subscriptions", s.listSubscriptions)
	mux.HandleFunc("POST /api/dashboards/{id}/subscriptions", s.createSubscription)
	mux.HandleFunc("POST /api/subscriptions/{id}/run", s.runSubscription)
	mux.HandleFunc("PUT /api/subscriptions/{id}", s.updateSubscription)
	mux.HandleFunc("DELETE /api/subscriptions/{id}", s.deleteSubscription)
	mux.HandleFunc("GET /auth/feishu/login", s.feishuLogin)
	mux.HandleFunc("GET /auth/oauth/{provider}/login", s.oauthLogin)
	mux.HandleFunc("GET /auth/oauth/{provider}/callback", s.oauthCallback)
	static, err := fs.Sub(web, "web")
	if err != nil {
		panic(err)
	}
	s.static = static
	mux.HandleFunc("GET /admin/{path...}", s.serveAdminStatic)
	mux.HandleFunc("GET /dashboard/{id}/{$}", s.serveDashboardView)
	mux.HandleFunc("GET /public/dashboard/{uuid}/{$}", s.servePublicDashboard)
	mux.HandleFunc("GET /embed/dashboard/{uuid}/{$}", s.servePublicDashboard)
	mux.HandleFunc("GET /questions/new/{$}", s.serveNewAnalysis)
	mux.HandleFunc("GET /questions/{id}/{$}", s.serveQuestionView)
	mux.HandleFunc("GET /collections/{id}/{$}", s.serveCollectionView)
	mux.Handle("GET /", http.FileServer(http.FS(static)))
	handler := s.accessControl(s.csrfProtection(securityHeaders(mux)))
	return &runtimeHandler{handler: handler, close: s.close}
}

func (s *server) serveAdminStatic(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentSessionUser(r)
	if !ok {
		http.Redirect(w, r, "/auth/login/", http.StatusFound)
		return
	}
	if !s.identity.IsAdmin(user.ID) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.FileServer(http.FS(s.static)).ServeHTTP(w, r)
}

func (s *server) restoreConnections() {
	secrets, err := s.catalog.Secrets.ListConnectionSecrets()
	if err != nil {
		log.Printf("topbase: read saved connection secrets: %v", err)
		return
	}
	for id, input := range secrets {
		input.ID = id
		if _, err := s.catalog.Connector.Connect(context.Background(), input); err != nil {
			log.Printf("topbase: restore database %q: %v", id, err)
		}
	}
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	info := buildinfo.Current(s.store.SchemaVersion())
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "version": info.Version, "commit": info.Commit})
}

func (s *server) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "error": "application database unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "schema_version": s.store.SchemaVersion()})
}

func (s *server) version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, buildinfo.Current(s.store.SchemaVersion()))
}

func (s *server) close() error {
	if s.cancel != nil {
		s.cancel()
	}
	s.workers.Wait()
	return errors.Join(s.connector.CloseAll(), s.store.Close())
}

func (s *server) currentSessionUser(r *http.Request) (core.User, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return core.User{}, false
	}
	user, err := s.identity.UserForSession(cookie.Value)
	if err != nil {
		return core.User{}, false
	}
	return user, true
}

func setSessionCookie(w http.ResponseWriter, sessionID string, expires time.Time) {
	secure := strings.EqualFold(os.Getenv("TOPBASE_SECURE_COOKIES"), "true")
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: sessionID, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: expires,
	})
	setCSRFCookie(w, newCSRFToken(), expires)
}

func setCSRFCookie(w http.ResponseWriter, token string, expires time.Time) {
	secure := strings.EqualFold(os.Getenv("TOPBASE_SECURE_COOKIES"), "true")
	http.SetCookie(w, &http.Cookie{Name: csrfCookie, Value: token, Path: "/", Secure: secure, SameSite: http.SameSiteLaxMode, Expires: expires})
}

func clearSessionCookie(w http.ResponseWriter) {
	secure := strings.EqualFold(os.Getenv("TOPBASE_SECURE_COOKIES"), "true")
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: csrfCookie, Value: "", Path: "/", Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}
