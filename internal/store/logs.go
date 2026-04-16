package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zc12120/atomhub/internal/types"
)

type LogStore struct {
	db *sql.DB
}

func NewLogStore(db *sql.DB) *LogStore {
	return &LogStore{db: db}
}

func (s *LogStore) Insert(
	ctx context.Context,
	keyID int64,
	model string,
	usage types.UsageTokens,
	latency time.Duration,
	callErr error,
) (int64, error) {
	status := "ok"
	errMsg := ""
	if callErr != nil {
		status = "error"
		errMsg = callErr.Error()
	}
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(
		ctx,
		`insert into request_logs (
			key_id, model, prompt_tokens, completion_tokens, total_tokens,
			latency_ms, status, error_message, created_at
		) values (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		keyID,
		model,
		maxInt64(usage.PromptTokens, 0),
		maxInt64(usage.CompletionTokens, 0),
		maxInt64(usage.TotalTokens, 0),
		latency.Milliseconds(),
		status,
		errMsg,
		createdAt,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *LogStore) ListRecent(ctx context.Context, limit int) ([]types.RequestLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(
		ctx,
		`select id, key_id, model, prompt_tokens, completion_tokens, total_tokens,
		        latency_ms, status, error_message, created_at
		 from request_logs
		 order by id desc
		 limit ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]types.RequestLog, 0, limit)
	for rows.Next() {
		entry, scanErr := scanRequestLog(rows)
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

func scanRequestLog(scanner rowScanner) (types.RequestLog, error) {
	var (
		entry      types.RequestLog
		errMessage sql.NullString
		createdRaw string
	)
	if err := scanner.Scan(
		&entry.ID,
		&entry.KeyID,
		&entry.Model,
		&entry.PromptTokens,
		&entry.CompletionTokens,
		&entry.TotalTokens,
		&entry.LatencyMS,
		&entry.Status,
		&errMessage,
		&createdRaw,
	); err != nil {
		return types.RequestLog{}, err
	}
	entry.ErrorMessage = errMessage.String
	createdAt, err := parseSQLiteTime(createdRaw)
	if err != nil {
		return types.RequestLog{}, fmt.Errorf("parse request log created_at: %w", err)
	}
	entry.CreatedAt = createdAt
	return entry, nil
}

func maxInt64(v int64, min int64) int64 {
	if v < min {
		return min
	}
	return v
}
