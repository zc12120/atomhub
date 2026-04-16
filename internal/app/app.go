package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/zc12120/atomhub/internal/auth"
	"github.com/zc12120/atomhub/internal/config"
	"github.com/zc12120/atomhub/internal/store"
)

// App contains bootstrapped backend dependencies.
type App struct {
	Config  config.Config
	DB      *sql.DB
	Handler http.Handler

	server         *http.Server
	adminRepo      *store.AdminRepository
	sessionManager *auth.SessionManager
}

// New initializes application configuration, database, migrations, auth, and HTTP routes.
func New(cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	db, err := store.OpenSQLite(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	if err := store.Migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	adminRepo := store.NewAdminRepository(db)
	passwordHash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("hash default admin password: %w", err)
	}

	if err := adminRepo.EnsureDefaultAdmin(context.Background(), cfg.AdminUsername, passwordHash); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ensure default admin: %w", err)
	}

	sessionManager := auth.NewSessionManager(cfg.SessionSecret, cfg.SessionTTL)

	application := &App{
		Config:         cfg,
		DB:             db,
		adminRepo:      adminRepo,
		sessionManager: sessionManager,
	}

	application.Handler = application.routes()

	return application, nil
}

// Run starts the HTTP server and blocks until it stops.
func (a *App) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:    a.Config.HTTPAddr,
		Handler: a.Handler,
	}
	a.server = server

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err := server.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Close releases open resources.
func (a *App) Close() error {
	if a.server != nil {
		_ = a.server.Close()
	}
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /admin/login", a.handleAdminLogin)
	mux.HandleFunc("POST /admin/logout", a.handleAdminLogout)
	mux.Handle("GET /admin/me", auth.RequireAdmin(a.sessionManager, http.HandlerFunc(a.handleAdminMe)))

	return mux
}

type adminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *App) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req adminLoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, err := a.adminRepo.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, store.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authentication failed"})
		return
	}

	if err := a.sessionManager.Set(w, user.Username); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleAdminLogout(w http.ResponseWriter, _ *http.Request) {
	a.sessionManager.Clear(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleAdminMe(w http.ResponseWriter, r *http.Request) {
	username, ok := auth.UsernameFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"username": username})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
