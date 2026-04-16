package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/zc12120/atomhub/internal/types"
)

var ErrStateNotFound = errors.New("key state not found")

type StateStore struct {
	db *sql.DB

	mu       sync.Mutex
	inflight map[int64]int

	cooldownBase time.Duration
	cooldownCap  time.Duration
}

func NewStateStore(db *sql.DB) *StateStore {
	return &StateStore{
		db:           db,
		inflight:     make(map[int64]int),
		cooldownBase: 30 * time.Second,
		cooldownCap:  10 * time.Minute,
	}
}

func (s *StateStore) Ensure(ctx context.Context, keyID int64) error {
	_, err := s.db.ExecContext(
		ctx,
		`insert into key_state (key_id, status, consecutive_failures)
		 values (?, ?, 0)
		 on conflict(key_id) do nothing`,
		keyID,
		string(types.KeyStatusHealthy),
	)
	return err
}

func (s *StateStore) Get(ctx context.Context, keyID int64) (types.KeyState, error) {
	if err := s.Ensure(ctx, keyID); err != nil {
		return types.KeyState{}, err
	}
	row := s.db.QueryRowContext(
		ctx,
		`select key_id, status, cooldown_until, consecutive_failures, last_error, last_success_at, last_probe_at
		 from key_state where key_id = ?`,
		keyID,
	)
	state, err := scanKeyState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.KeyState{}, ErrStateNotFound
		}
		return types.KeyState{}, err
	}
	return state, nil
}

func (s *StateStore) Candidates(ctx context.Context, keyIDs []int64) ([]types.KeyCandidate, error) {
	if len(keyIDs) == 0 {
		return nil, nil
	}

	for _, keyID := range keyIDs {
		if err := s.Ensure(ctx, keyID); err != nil {
			return nil, err
		}
	}

	query, args := inClauseQuery(
		`select key_id, status, cooldown_until, consecutive_failures, last_error, last_success_at, last_probe_at
		 from key_state where key_id in (%s)`,
		keyIDs,
	)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make(map[int64]types.KeyState, len(keyIDs))
	for rows.Next() {
		state, scanErr := scanKeyState(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		states[state.KeyID] = state
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	candidates := make([]types.KeyCandidate, 0, len(keyIDs))
	for _, keyID := range keyIDs {
		state, ok := states[keyID]
		if !ok {
			state = types.KeyState{KeyID: keyID, Status: types.KeyStatusHealthy}
		}
		coolingDown := state.CooldownUntil != nil && now.Before(*state.CooldownUntil)
		if coolingDown {
			state.Status = types.KeyStatusCoolingDown
		}
		candidates = append(candidates, types.KeyCandidate{
			KeyID:         keyID,
			Status:        state.Status,
			CoolingDown:   coolingDown,
			CooldownUntil: state.CooldownUntil,
			Inflight:      s.Inflight(keyID),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].KeyID < candidates[j].KeyID
	})
	return candidates, nil
}

func (s *StateStore) MarkProbeSuccess(ctx context.Context, keyID int64) error {
	if err := s.Ensure(ctx, keyID); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(
		ctx,
		`update key_state
		 set status = ?, cooldown_until = null, consecutive_failures = 0,
		     last_error = '', last_success_at = ?, last_probe_at = ?
		 where key_id = ?`,
		string(types.KeyStatusHealthy),
		now,
		now,
		keyID,
	)
	return err
}

func (s *StateStore) MarkProbeFailure(ctx context.Context, keyID int64, probeErr error) error {
	return s.markFailure(ctx, keyID, probeErr, true)
}

func (s *StateStore) MarkSuccess(ctx context.Context, keyID int64) error {
	if err := s.Ensure(ctx, keyID); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(
		ctx,
		`update key_state
		 set status = ?, cooldown_until = null, consecutive_failures = 0,
		     last_error = '', last_success_at = ?
		 where key_id = ?`,
		string(types.KeyStatusHealthy),
		now,
		keyID,
	)
	return err
}

func (s *StateStore) MarkFailure(ctx context.Context, keyID int64, callErr error) error {
	return s.markFailure(ctx, keyID, callErr, false)
}

func (s *StateStore) markFailure(ctx context.Context, keyID int64, inErr error, isProbe bool) error {
	if err := s.Ensure(ctx, keyID); err != nil {
		return err
	}
	state, err := s.Get(ctx, keyID)
	if err != nil {
		return err
	}

	failures := state.ConsecutiveFailures + 1
	cooldownFor := s.failureCooldown(failures)
	cooldownUntil := time.Now().UTC().Add(cooldownFor)
	status := types.KeyStatusDegraded
	if cooldownFor > 0 {
		status = types.KeyStatusCoolingDown
	}
	lastErr := ""
	if inErr != nil {
		lastErr = inErr.Error()
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	query := `update key_state
		set status = ?, cooldown_until = ?, consecutive_failures = ?, last_error = ?`
	args := []any{string(status), cooldownUntil.Format(time.RFC3339Nano), failures, lastErr}
	if isProbe {
		query += `, last_probe_at = ?`
		args = append(args, now)
	}
	query += ` where key_id = ?`
	args = append(args, keyID)

	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *StateStore) Inflight(keyID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inflight[keyID]
}

func (s *StateStore) IncrementInflight(keyID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inflight[keyID]++
	return s.inflight[keyID]
}

func (s *StateStore) DecrementInflight(keyID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current := s.inflight[keyID]; current <= 1 {
		delete(s.inflight, keyID)
		return 0
	}
	s.inflight[keyID]--
	return s.inflight[keyID]
}

func (s *StateStore) Overview(ctx context.Context) (types.HealthOverview, error) {
	rows, err := s.db.QueryContext(ctx, `select status, cooldown_until from key_state`)
	if err != nil {
		return types.HealthOverview{}, err
	}
	defer rows.Close()

	now := time.Now().UTC()
	overview := types.HealthOverview{}
	for rows.Next() {
		var (
			statusRaw string
			cooldown  sql.NullString
		)
		if err := rows.Scan(&statusRaw, &cooldown); err != nil {
			return types.HealthOverview{}, err
		}
		overview.Total++
		status := types.KeyStatus(strings.TrimSpace(statusRaw))
		cooldownUntil, err := parseNullableSQLiteTime(cooldown)
		if err != nil {
			return types.HealthOverview{}, fmt.Errorf("parse cooldown_until: %w", err)
		}
		if cooldownUntil != nil && now.Before(*cooldownUntil) {
			overview.CoolingDown++
			continue
		}
		switch status {
		case types.KeyStatusHealthy:
			overview.Healthy++
		case types.KeyStatusDisabled:
			overview.Disabled++
		case types.KeyStatusCoolingDown:
			overview.CoolingDown++
		default:
			overview.Degraded++
		}
	}
	if err := rows.Err(); err != nil {
		return types.HealthOverview{}, err
	}
	return overview, nil
}

func scanKeyState(scanner rowScanner) (types.KeyState, error) {
	var (
		state         types.KeyState
		statusRaw     string
		cooldownRaw   sql.NullString
		lastErrorRaw  sql.NullString
		lastSuccessAt sql.NullString
		lastProbeAt   sql.NullString
	)
	if err := scanner.Scan(
		&state.KeyID,
		&statusRaw,
		&cooldownRaw,
		&state.ConsecutiveFailures,
		&lastErrorRaw,
		&lastSuccessAt,
		&lastProbeAt,
	); err != nil {
		return types.KeyState{}, err
	}
	state.Status = types.KeyStatus(strings.TrimSpace(statusRaw))
	state.LastError = strings.TrimSpace(lastErrorRaw.String)

	cooldownUntil, err := parseNullableSQLiteTime(cooldownRaw)
	if err != nil {
		return types.KeyState{}, fmt.Errorf("parse cooldown_until: %w", err)
	}
	state.CooldownUntil = cooldownUntil

	lastSuccess, err := parseNullableSQLiteTime(lastSuccessAt)
	if err != nil {
		return types.KeyState{}, fmt.Errorf("parse last_success_at: %w", err)
	}
	state.LastSuccessAt = lastSuccess

	lastProbe, err := parseNullableSQLiteTime(lastProbeAt)
	if err != nil {
		return types.KeyState{}, fmt.Errorf("parse last_probe_at: %w", err)
	}
	state.LastProbeAt = lastProbe
	return state, nil
}

func (s *StateStore) failureCooldown(failures int) time.Duration {
	if failures <= 0 {
		return 0
	}
	exponent := math.Min(float64(failures-1), 10)
	cooldown := float64(s.cooldownBase) * math.Pow(2, exponent)
	if cooldown > float64(s.cooldownCap) {
		return s.cooldownCap
	}
	return time.Duration(cooldown)
}

func inClauseQuery(base string, ids []int64) (string, []any) {
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	return fmt.Sprintf(base, strings.Join(placeholders, ",")), args
}
