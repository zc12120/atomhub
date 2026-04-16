package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/zc12120/atomhub/internal/types"
)

type ModelStore struct {
	db *sql.DB
}

func NewModelStore(db *sql.DB) *ModelStore {
	return &ModelStore{db: db}
}

func (s *ModelStore) ReplaceForKey(ctx context.Context, keyID int64, models []string) error {
	cleaned := uniqueNonEmpty(models)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `delete from key_models where key_id = ?`, keyID); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, model := range cleaned {
		if _, err := tx.ExecContext(
			ctx,
			`insert into key_models (key_id, model, created_at) values (?, ?, ?)`,
			keyID,
			model,
			now,
		); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *ModelStore) ListByKey(ctx context.Context, keyID int64) ([]types.KeyModel, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`select id, key_id, model, created_at from key_models where key_id = ? order by model asc`,
		keyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.KeyModel, 0)
	for rows.Next() {
		entry, scanErr := scanKeyModel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ModelStore) ListAll(ctx context.Context) ([]types.KeyModel, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`select id, key_id, model, created_at from key_models order by model asc, key_id asc`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.KeyModel, 0)
	for rows.Next() {
		entry, scanErr := scanKeyModel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanKeyModel(scanner rowScanner) (types.KeyModel, error) {
	var (
		entry      types.KeyModel
		createdRaw string
	)
	if err := scanner.Scan(&entry.ID, &entry.KeyID, &entry.Model, &createdRaw); err != nil {
		return types.KeyModel{}, err
	}
	createdAt, err := parseSQLiteTime(createdRaw)
	if err != nil {
		return types.KeyModel{}, fmt.Errorf("parse key model created_at: %w", err)
	}
	entry.CreatedAt = createdAt
	return entry, nil
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
