package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	internalauth "github.com/zc12120/atomhub/internal/auth"
	"github.com/zc12120/atomhub/internal/types"
)

var ErrDownstreamKeyNotFound = errors.New("downstream key not found")

type DownstreamKeyStore struct {
	db     *sql.DB
	secret string
}

func NewDownstreamKeyStore(db *sql.DB, secret string) *DownstreamKeyStore {
	return &DownstreamKeyStore{db: db, secret: secret}
}

func (s *DownstreamKeyStore) Create(ctx context.Context, key types.DownstreamKey) (types.DownstreamKey, string, error) {
	name := strings.TrimSpace(key.Name)
	if name == "" {
		return types.DownstreamKey{}, "", errors.New("name is required")
	}

	token, tokenPrefix, tokenHash, err := internalauth.GenerateDownstreamToken()
	if err != nil {
		return types.DownstreamKey{}, "", err
	}
	encryptedToken, err := internalauth.EncryptDownstreamToken(s.secret, token)
	if err != nil {
		return types.DownstreamKey{}, "", err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(
		ctx,
		`insert into downstream_keys (
			name, token_prefix, token_hash, encrypted_token, enabled, request_count,
			prompt_tokens, completion_tokens, total_tokens, created_at, updated_at
		) values (?, ?, ?, ?, ?, 0, 0, 0, 0, ?, ?)`,
		name,
		tokenPrefix,
		tokenHash,
		encryptedToken,
		boolToInt(key.Enabled),
		now,
		now,
	)
	if err != nil {
		return types.DownstreamKey{}, "", err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return types.DownstreamKey{}, "", err
	}

	created, err := s.Get(ctx, id)
	if err != nil {
		return types.DownstreamKey{}, "", err
	}
	return created, token, nil
}

func (s *DownstreamKeyStore) Get(ctx context.Context, id int64) (types.DownstreamKey, error) {
	row := s.db.QueryRowContext(
		ctx,
		`select id, name, token_prefix, token_hash, encrypted_token, enabled, last_used_at,
		        request_count, prompt_tokens, completion_tokens, total_tokens,
		        created_at, updated_at
		 from downstream_keys where id = ?`,
		id,
	)
	key, err := scanDownstreamKey(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.DownstreamKey{}, ErrDownstreamKeyNotFound
		}
		return types.DownstreamKey{}, err
	}
	return key, nil
}

func (s *DownstreamKeyStore) FindByToken(ctx context.Context, token string) (types.DownstreamKey, error) {
	row := s.db.QueryRowContext(
		ctx,
		`select id, name, token_prefix, token_hash, encrypted_token, enabled, last_used_at,
		        request_count, prompt_tokens, completion_tokens, total_tokens,
		        created_at, updated_at
		 from downstream_keys
		 where token_hash = ? and enabled = 1`,
		internalauth.HashDownstreamToken(strings.TrimSpace(token)),
	)
	key, err := scanDownstreamKey(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.DownstreamKey{}, ErrDownstreamKeyNotFound
		}
		return types.DownstreamKey{}, err
	}
	return key, nil
}

func (s *DownstreamKeyStore) List(ctx context.Context) ([]types.DownstreamKey, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`select id, name, token_prefix, token_hash, encrypted_token, enabled, last_used_at,
		        request_count, prompt_tokens, completion_tokens, total_tokens,
		        created_at, updated_at
		 from downstream_keys
		 order by id asc`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]types.DownstreamKey, 0)
	for rows.Next() {
		key, scanErr := scanDownstreamKey(rows)
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

func (s *DownstreamKeyStore) Update(ctx context.Context, key types.DownstreamKey) (types.DownstreamKey, error) {
	if key.ID == 0 {
		return types.DownstreamKey{}, ErrDownstreamKeyNotFound
	}
	name := strings.TrimSpace(key.Name)
	if name == "" {
		return types.DownstreamKey{}, errors.New("name is required")
	}

	res, err := s.db.ExecContext(
		ctx,
		`update downstream_keys
		 set name = ?, enabled = ?, updated_at = ?
		 where id = ?`,
		name,
		boolToInt(key.Enabled),
		time.Now().UTC().Format(time.RFC3339Nano),
		key.ID,
	)
	if err != nil {
		return types.DownstreamKey{}, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return types.DownstreamKey{}, err
	}
	if rows == 0 {
		return types.DownstreamKey{}, ErrDownstreamKeyNotFound
	}
	return s.Get(ctx, key.ID)
}

func (s *DownstreamKeyStore) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `delete from downstream_keys where id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDownstreamKeyNotFound
	}
	return nil
}

func (s *DownstreamKeyStore) Reveal(ctx context.Context, id int64) (string, error) {
	key, err := s.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(key.EncryptedToken) == "" {
		return "", errors.New("downstream key was created before encrypted storage; please regenerate it")
	}
	return internalauth.DecryptDownstreamToken(s.secret, key.EncryptedToken)
}

func (s *DownstreamKeyStore) Regenerate(ctx context.Context, id int64) (types.DownstreamKey, string, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return types.DownstreamKey{}, "", err
	}
	token, tokenPrefix, tokenHash, err := internalauth.GenerateDownstreamToken()
	if err != nil {
		return types.DownstreamKey{}, "", err
	}
	encryptedToken, err := internalauth.EncryptDownstreamToken(s.secret, token)
	if err != nil {
		return types.DownstreamKey{}, "", err
	}
	res, err := s.db.ExecContext(
		ctx,
		`update downstream_keys
		 set token_prefix = ?, token_hash = ?, encrypted_token = ?, updated_at = ?
		 where id = ?`,
		tokenPrefix,
		tokenHash,
		encryptedToken,
		time.Now().UTC().Format(time.RFC3339Nano),
		id,
	)
	if err != nil {
		return types.DownstreamKey{}, "", err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return types.DownstreamKey{}, "", err
	}
	if rows == 0 {
		return types.DownstreamKey{}, "", ErrDownstreamKeyNotFound
	}
	refreshed, err := s.Get(ctx, existing.ID)
	if err != nil {
		return types.DownstreamKey{}, "", err
	}
	return refreshed, token, nil
}

func (s *DownstreamKeyStore) RecordUsage(ctx context.Context, id int64, usage types.UsageTokens, usedAt time.Time) error {
	if id == 0 {
		return nil
	}
	_, err := s.db.ExecContext(
		ctx,
		`update downstream_keys
		 set request_count = request_count + 1,
		     prompt_tokens = prompt_tokens + ?,
		     completion_tokens = completion_tokens + ?,
		     total_tokens = total_tokens + ?,
		     last_used_at = ?,
		     updated_at = ?
		 where id = ?`,
		maxInt64(usage.PromptTokens, 0),
		maxInt64(usage.CompletionTokens, 0),
		maxInt64(usage.TotalTokens, 0),
		usedAt.UTC().Format(time.RFC3339Nano),
		usedAt.UTC().Format(time.RFC3339Nano),
		id,
	)
	return err
}

func scanDownstreamKey(scanner rowScanner) (types.DownstreamKey, error) {
	var (
		key            types.DownstreamKey
		enabledInt     int
		encryptedToken sql.NullString
		lastUsedRaw    sql.NullString
		createdRaw     string
		updatedRaw     string
	)
	if err := scanner.Scan(
		&key.ID,
		&key.Name,
		&key.TokenPrefix,
		&key.TokenHash,
		&encryptedToken,
		&enabledInt,
		&lastUsedRaw,
		&key.RequestCount,
		&key.PromptTokens,
		&key.CompletionTokens,
		&key.TotalTokens,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return types.DownstreamKey{}, err
	}

	lastUsedAt, err := parseNullableSQLiteTime(lastUsedRaw)
	if err != nil {
		return types.DownstreamKey{}, fmt.Errorf("parse downstream key last_used_at: %w", err)
	}
	createdAt, err := parseSQLiteTime(createdRaw)
	if err != nil {
		return types.DownstreamKey{}, fmt.Errorf("parse downstream key created_at: %w", err)
	}
	updatedAt, err := parseSQLiteTime(updatedRaw)
	if err != nil {
		return types.DownstreamKey{}, fmt.Errorf("parse downstream key updated_at: %w", err)
	}

	key.Enabled = enabledInt == 1
	key.EncryptedToken = strings.TrimSpace(encryptedToken.String)
	key.LastUsedAt = lastUsedAt
	key.CreatedAt = createdAt
	key.UpdatedAt = updatedAt
	return key, nil
}
