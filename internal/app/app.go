package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/zc12120/atomhub/internal/auth"
	"github.com/zc12120/atomhub/internal/catalog"
	"github.com/zc12120/atomhub/internal/config"
	"github.com/zc12120/atomhub/internal/probe"
	anthropicprovider "github.com/zc12120/atomhub/internal/providers/anthropic"
	"github.com/zc12120/atomhub/internal/providers/common"
	geminiprovider "github.com/zc12120/atomhub/internal/providers/gemini"
	openaiprovider "github.com/zc12120/atomhub/internal/providers/openai"
	"github.com/zc12120/atomhub/internal/selector"
	"github.com/zc12120/atomhub/internal/store"
)

// App contains bootstrapped backend dependencies.
type App struct {
	Config  config.Config
	DB      *sql.DB
	Handler http.Handler

	server             *http.Server
	adminRepo          *store.AdminRepository
	sessionManager     *auth.SessionManager
	keyStore           *store.KeyStore
	downstreamKeyStore *store.DownstreamKeyStore
	modelStore         *store.ModelStore
	stateStore         *store.StateStore
	logStore           *store.LogStore
	statsStore         *store.StatsStore
	catalog            *catalog.Catalog
	probeService       *probe.Service
	selector           *selector.Selector
	upstreamClient     *http.Client
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
	keyStore := store.NewKeyStore(db)
	downstreamKeyStore := store.NewDownstreamKeyStore(db)
	modelStore := store.NewModelStore(db)
	stateStore := store.NewStateStore(db)
	logStore := store.NewLogStore(db)
	statsStore := store.NewStatsStore(db)
	catalogStore := catalog.New(keyStore, modelStore)
	probeService := probe.NewService(
		keyStore,
		modelStore,
		stateStore,
		catalogStore,
		openaiprovider.New(common.NewClient(30*time.Second)),
		anthropicprovider.New(common.NewClient(30*time.Second)),
		geminiprovider.New(common.NewClient(30*time.Second)),
	)
	_ = catalogStore.Rebuild(context.Background())

	application := &App{
		Config:             cfg,
		DB:                 db,
		adminRepo:          adminRepo,
		sessionManager:     sessionManager,
		keyStore:           keyStore,
		downstreamKeyStore: downstreamKeyStore,
		modelStore:         modelStore,
		stateStore:         stateStore,
		logStore:           logStore,
		statsStore:         statsStore,
		catalog:            catalogStore,
		probeService:       probeService,
		selector:           selector.New(),
		upstreamClient:     &http.Client{Timeout: 60 * time.Second},
	}

	application.Handler = application.routes()
	return application, nil
}

// Run starts the HTTP server and blocks until it stops.
func (a *App) Run(ctx context.Context) error {
	go func() {
		results := a.probeService.ProbeAll(context.Background())
		for keyID, err := range results {
			if err != nil {
				log.Printf("initial probe failed for key %d: %v", keyID, err)
			}
		}
	}()

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
