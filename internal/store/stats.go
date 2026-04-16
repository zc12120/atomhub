package store

import (
	"context"
	"database/sql"

	"github.com/zc12120/atomhub/internal/types"
)

type StatsStore struct {
	db *sql.DB
}

func NewStatsStore(db *sql.DB) *StatsStore {
	return &StatsStore{db: db}
}

func (s *StatsStore) TokenStats(ctx context.Context) ([]types.ModelTokenStat, types.TokenSummary, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`select model,
		        coalesce(sum(prompt_tokens), 0) as prompt_tokens,
		        coalesce(sum(completion_tokens), 0) as completion_tokens,
		        coalesce(sum(total_tokens), 0) as total_tokens,
		        count(*) as request_count
		 from request_logs
		 group by model
		 order by total_tokens desc, model asc`,
	)
	if err != nil {
		return nil, types.TokenSummary{}, err
	}
	defer rows.Close()

	items := make([]types.ModelTokenStat, 0)
	for rows.Next() {
		var item types.ModelTokenStat
		if err := rows.Scan(
			&item.Model,
			&item.PromptTokens,
			&item.CompletionTokens,
			&item.TotalTokens,
			&item.RequestCount,
		); err != nil {
			return nil, types.TokenSummary{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, types.TokenSummary{}, err
	}

	var summary types.TokenSummary
	if err := s.db.QueryRowContext(
		ctx,
		`select
		    coalesce(sum(prompt_tokens), 0),
		    coalesce(sum(completion_tokens), 0),
		    coalesce(sum(total_tokens), 0),
		    count(*)
		 from request_logs`,
	).Scan(
		&summary.PromptTokens,
		&summary.CompletionTokens,
		&summary.TotalTokens,
		&summary.RequestCount,
	); err != nil {
		return nil, types.TokenSummary{}, err
	}

	return items, summary, nil
}
