package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zc12120/atomhub/internal/types"
)

var (
	ErrKeyNotFound = errors.New("key not found")

	sqliteTimeLayouts = []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02T15:04:05",
	}
)

type KeyStore struct {
	db *sql.DB
}

func NewKeyStore(db *sql.DB) *KeyStore {
	return &KeyStore{db: db}
}

func (s *KeyStore) Create(ctx context.Context, key types.UpstreamKey) (types.UpstreamKey, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(
		ctx,
		`insert into upstream_keys (name, provider, base_url, api_key, enabled, created_at, updated_at)
		 values (?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(key.Name),
		string(key.Provider),
		strings.TrimSpace(key.BaseURL),
		key.APIKey,
		boolToInt(key.Enabled),
		now,
		now,
	)
	if err != nil {
		return types.UpstreamKey{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return types.UpstreamKey{}, err
	}
	return s.Get(ctx, id)
}

func (s *KeyStore) Update(ctx context.Context, key types.UpstreamKey) (types.UpstreamKey, error) {
	if key.ID == 0 {
		return types.UpstreamKey{}, ErrKeyNotFound
	}
	res, err := s.db.ExecContext(
		ctx,
		`update upstream_keys
		 set name = ?, provider = ?, base_url = ?, api_key = ?, enabled = ?, updated_at = ?
		 where id = ?`,
		strings.TrimSpace(key.Name),
		string(key.Provider),
		strings.TrimSpace(key.BaseURL),
		key.APIKey,
		boolToInt(key.Enabled),
		time.Now().UTC().Format(time.RFC3339Nano),
		key.ID,
	)
	if err != nil {
		return types.UpstreamKey{}, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return types.UpstreamKey{}, err
	}
	if rows == 0 {
		return types.UpstreamKey{}, ErrKeyNotFound
	}
	return s.Get(ctx, key.ID)
}

func (s *KeyStore) SetEnabled(ctx context.Context, keyID int64, enabled bool) error {
	res, err := s.db.ExecContext(
		ctx,
		`update upstream_keys set enabled = ?, updated_at = ? where id = ?`,
		boolToInt(enabled),
		time.Now().UTC().Format(time.RFC3339Nano),
		keyID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrKeyNotFound
	}
	return nil
}

func (s *KeyStore) Delete(ctx context.Context, keyID int64) error {
	res, err := s.db.ExecContext(ctx, `delete from upstream_keys where id = ?`, keyID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrKeyNotFound
	}
	return nil
}

func (s *KeyStore) Get(ctx context.Context, keyID int64) (types.UpstreamKey, error) {
	row := s.db.QueryRowContext(
		ctx,
		`select id, name, provider, base_url, api_key, enabled, created_at, updated_at
		 from upstream_keys where id = ?`,
		keyID,
	)
	key, err := scanUpstreamKey(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.UpstreamKey{}, ErrKeyNotFound
		}
		return types.UpstreamKey{}, err
	}
	return key, nil
}

func (s *KeyStore) List(ctx context.Context) ([]types.UpstreamKey, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`select id, name, provider, base_url, api_key, enabled, created_at, updated_at
		 from upstream_keys order by id asc`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]types.UpstreamKey, 0)
	for rows.Next() {
		key, scanErr := scanUpstreamKey(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *KeyStore) ListEnabled(ctx context.Context) ([]types.UpstreamKey, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`select id, name, provider, base_url, api_key, enabled, created_at, updated_at
		 from upstream_keys where enabled = 1 order by id asc`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]types.UpstreamKey, 0)
	for rows.Next() {
		key, scanErr := scanUpstreamKey(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUpstreamKey(scanner rowScanner) (types.UpstreamKey, error) {
	var (
		key        types.UpstreamKey
		provider   string
		enabledInt int
		createdRaw string
		updatedRaw string
	)
	if err := scanner.Scan(
		&key.ID,
		&key.Name,
		&provider,
		&key.BaseURL,
		&key.APIKey,
		&enabledInt,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return types.UpstreamKey{}, err
	}
	createdAt, err := parseSQLiteTime(createdRaw)
	if err != nil {
		return types.UpstreamKey{}, fmt.Errorf("parse key created_at: %w", err)
	}
	updatedAt, err := parseSQLiteTime(updatedRaw)
	if err != nil {
		return types.UpstreamKey{}, fmt.Errorf("parse key updated_at: %w", err)
	}
	key.Provider = types.Provider(provider)
	key.Enabled = enabledInt == 1
	key.CreatedAt = createdAt
	key.UpdatedAt = updatedAt
	return key, nil
}

func parseSQLiteTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	for _, layout := range sqliteTimeLayouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", raw)
}

func parseNullableSQLiteTime(raw sql.NullString) (*time.Time, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil, nil
	}
	parsed, err := parseSQLiteTime(raw.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullableTimeValue(ts *time.Time) any {
	if ts == nil || ts.IsZero() {
		return nil
	}
	return ts.UTC().Format(time.RFC3339Nano)
}
