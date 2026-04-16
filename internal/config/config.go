package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultHTTPAddr      = ":8080"
	defaultDBPath        = "./data/atomhub.db"
	defaultSessionSecret = "atomhub-dev-secret"
	defaultSessionTTL    = 24 * time.Hour
	defaultAdminUsername = "admin"
	defaultAdminPassword = "admin"
)

// Config contains runtime configuration for the AtomHub backend foundation.
type Config struct {
	HTTPAddr      string
	DBPath        string
	SessionSecret string
	SessionTTL    time.Duration
	AdminUsername string
	AdminPassword string
}

// Load parses configuration from environment variables.
func Load() (Config, error) {
	ttl := defaultSessionTTL
	if rawTTL := strings.TrimSpace(os.Getenv("ATOMHUB_SESSION_TTL")); rawTTL != "" {
		parsed, err := time.ParseDuration(rawTTL)
		if err != nil {
			return Config{}, fmt.Errorf("parse ATOMHUB_SESSION_TTL: %w", err)
		}
		ttl = parsed
	}

	cfg := Config{
		HTTPAddr:      defaultIfEmpty(os.Getenv("ATOMHUB_HTTP_ADDR"), defaultHTTPAddr),
		DBPath:        defaultIfEmpty(os.Getenv("ATOMHUB_DB_PATH"), defaultDBPath),
		SessionSecret: defaultIfEmpty(os.Getenv("ATOMHUB_SESSION_SECRET"), defaultSessionSecret),
		SessionTTL:    ttl,
		AdminUsername: defaultIfEmpty(os.Getenv("ATOMHUB_ADMIN_USERNAME"), defaultAdminUsername),
		AdminPassword: defaultIfEmpty(os.Getenv("ATOMHUB_ADMIN_PASSWORD"), defaultAdminPassword),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks required configuration values.
func (c Config) Validate() error {
	switch {
	case strings.TrimSpace(c.HTTPAddr) == "":
		return errors.New("http addr is required")
	case strings.TrimSpace(c.DBPath) == "":
		return errors.New("db path is required")
	case strings.TrimSpace(c.SessionSecret) == "":
		return errors.New("session secret is required")
	case c.SessionTTL <= 0:
		return errors.New("session ttl must be positive")
	case strings.TrimSpace(c.AdminUsername) == "":
		return errors.New("admin username is required")
	case strings.TrimSpace(c.AdminPassword) == "":
		return errors.New("admin password is required")
	}

	if dir := filepath.Dir(c.DBPath); dir == "" {
		return errors.New("db path directory is invalid")
	}
	return nil
}

func defaultIfEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}
