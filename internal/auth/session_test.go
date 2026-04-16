package auth_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zc12120/atomhub/internal/app"
	"github.com/zc12120/atomhub/internal/auth"
	"github.com/zc12120/atomhub/internal/config"
	"github.com/zc12120/atomhub/internal/store"
)

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := auth.HashPassword("changeme")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "changeme" {
		t.Fatalf("hash should not match plaintext")
	}

	if err := auth.VerifyPassword(hash, "changeme"); err != nil {
		t.Fatalf("verify password: %v", err)
	}

	if err := auth.VerifyPassword(hash, "wrong"); err == nil {
		t.Fatalf("expected invalid password error")
	}
}

func TestSessionManagerCookieRoundTripAndClear(t *testing.T) {
	manager := auth.NewSessionManager("test-secret", 24*time.Hour)
	recorder := httptest.NewRecorder()

	if err := manager.Set(recorder, "admin"); err != nil {
		t.Fatalf("set session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/me", nil)
	for _, cookie := range recorder.Result().Cookies() {
		req.AddCookie(cookie)
	}

	username, ok := manager.Get(req)
	if !ok || username != "admin" {
		t.Fatalf("expected authenticated admin, got %q ok=%v", username, ok)
	}

	clearRecorder := httptest.NewRecorder()
	manager.Clear(clearRecorder)
	cleared := clearRecorder.Result().Cookies()
	if len(cleared) == 0 || cleared[0].MaxAge >= 0 {
		t.Fatalf("expected expired session cookie")
	}
}

func TestSessionManagerRejectsTamperedCookie(t *testing.T) {
	manager := auth.NewSessionManager("test-secret", time.Hour)
	recorder := httptest.NewRecorder()

	if err := manager.Set(recorder, "admin"); err != nil {
		t.Fatalf("set session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/me", nil)
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie")
	}
	tampered := *cookies[0]
	tampered.Value = tampered.Value + "tamper"
	req.AddCookie(&tampered)

	if _, ok := manager.Get(req); ok {
		t.Fatalf("expected tampered cookie to be rejected")
	}
}

func TestRequireAdminMiddleware(t *testing.T) {
	manager := auth.NewSessionManager("test-secret", time.Hour)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, ok := auth.UsernameFromContext(r.Context())
		if !ok || username != "admin" {
			t.Fatalf("expected admin username in context")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	protected := auth.RequireAdmin(manager, next)

	unauthReq := httptest.NewRequest(http.MethodGet, "/admin/me", nil)
	unauthRecorder := httptest.NewRecorder()
	protected.ServeHTTP(unauthRecorder, unauthReq)
	if unauthRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing session, got %d", unauthRecorder.Code)
	}

	loginRecorder := httptest.NewRecorder()
	if err := manager.Set(loginRecorder, "admin"); err != nil {
		t.Fatalf("set session: %v", err)
	}
	authReq := httptest.NewRequest(http.MethodGet, "/admin/me", nil)
	for _, cookie := range loginRecorder.Result().Cookies() {
		authReq.AddCookie(cookie)
	}
	authRecorder := httptest.NewRecorder()
	protected.ServeHTTP(authRecorder, authReq)
	if authRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for valid session, got %d", authRecorder.Code)
	}
}

func TestStoreMigrateAndAdminAuthentication(t *testing.T) {
	db := openTestSQLite(t)

	if err := store.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := store.NewAdminRepository(db)
	hash, err := auth.HashPassword("secret-pass")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if err := repo.EnsureDefaultAdmin(context.Background(), "admin", hash); err != nil {
		t.Fatalf("ensure admin: %v", err)
	}
	if err := repo.EnsureDefaultAdmin(context.Background(), "admin", hash); err != nil {
		t.Fatalf("ensure admin idempotent: %v", err)
	}

	if _, err := repo.Authenticate(context.Background(), "admin", "secret-pass"); err != nil {
		t.Fatalf("authenticate success: %v", err)
	}

	if _, err := repo.Authenticate(context.Background(), "admin", "bad-pass"); err == nil {
		t.Fatalf("expected bad password failure")
	}
}

func TestConfigLoadAndAppBootstrap(t *testing.T) {
	t.Setenv("ATOMHUB_HTTP_ADDR", "127.0.0.1:9090")
	t.Setenv("ATOMHUB_DB_PATH", filepath.Join(t.TempDir(), "atomhub.db"))
	t.Setenv("ATOMHUB_SESSION_SECRET", "bootstrap-secret")
	t.Setenv("ATOMHUB_SESSION_TTL", "2h")
	t.Setenv("ATOMHUB_ADMIN_USERNAME", "root")
	t.Setenv("ATOMHUB_ADMIN_PASSWORD", "root-pass")
	t.Setenv("ATOMHUB_GATEWAY_TOKEN", "bootstrap-token")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Fatalf("expected custom http addr, got %q", cfg.HTTPAddr)
	}

	application, err := app.New(cfg)
	if err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}
	t.Cleanup(func() {
		_ = application.Close()
	})

	if !hasTable(t, application.DB, "admin_users") || !hasTable(t, application.DB, "request_logs") {
		t.Fatalf("expected migrations to create base tables")
	}

	loginBody := []byte(`{"username":"root","password":"root-pass"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/admin/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	application.Handler.ServeHTTP(loginRecorder, loginReq)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d body=%s", loginRecorder.Code, loginRecorder.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/admin/me", nil)
	for _, cookie := range loginRecorder.Result().Cookies() {
		meReq.AddCookie(cookie)
	}
	meRecorder := httptest.NewRecorder()
	application.Handler.ServeHTTP(meRecorder, meReq)
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("expected /admin/me success, got %d", meRecorder.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(meRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode me payload: %v", err)
	}
	if payload["username"] != "root" {
		t.Fatalf("expected username root, got %#v", payload)
	}
}

func openTestSQLite(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(dbPath)
	})
	return db
}

func hasTable(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()

	const query = `select count(1) from sqlite_master where type='table' and name=?`
	var count int
	if err := db.QueryRow(query, name).Scan(&count); err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return count == 1
}
