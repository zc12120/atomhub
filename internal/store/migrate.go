package store

import (
	"database/sql"
	"fmt"
)

// Migrate creates foundation schema needed by backend services.
func Migrate(db *sql.DB) error {
	stmts := []string{
		`create table if not exists admin_users (
            id integer primary key autoincrement,
            username text not null unique,
            password_hash text not null,
            created_at text not null default current_timestamp
        );`,
		`create table if not exists upstream_keys (
            id integer primary key autoincrement,
            name text not null,
            provider text not null,
            base_url text not null,
            api_key text not null,
            enabled integer not null default 1,
            created_at text not null default current_timestamp,
            updated_at text not null default current_timestamp
        );`,
		`create table if not exists key_models (
            id integer primary key autoincrement,
            key_id integer not null,
            model text not null,
            created_at text not null default current_timestamp,
            unique(key_id, model)
        );`,
		`create table if not exists key_state (
            key_id integer primary key,
            status text not null default 'healthy',
            cooldown_until text,
            consecutive_failures integer not null default 0,
            last_error text,
            last_success_at text,
            last_probe_at text
        );`,
		`create table if not exists downstream_keys (
            id integer primary key autoincrement,
            name text not null,
            token_prefix text not null,
            token_hash text not null unique,
            encrypted_token text,
            enabled integer not null default 1,
            last_used_at text,
            request_count integer not null default 0,
            prompt_tokens integer not null default 0,
            completion_tokens integer not null default 0,
            total_tokens integer not null default 0,
            created_at text not null default current_timestamp,
            updated_at text not null default current_timestamp
        );`,
		`create table if not exists request_logs (
            id integer primary key autoincrement,
            key_id integer not null,
            downstream_key_id integer,
            model text not null,
            prompt_tokens integer not null default 0,
            completion_tokens integer not null default 0,
            total_tokens integer not null default 0,
            latency_ms integer not null default 0,
            status text not null,
            error_message text,
            created_at text not null default current_timestamp
        );`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := ensureColumnExists(db, "request_logs", "downstream_key_id", "integer"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "downstream_keys", "encrypted_token", "text"); err != nil {
		return err
	}
	return nil
}

func ensureColumnExists(db *sql.DB, tableName string, columnName string, columnDDL string) error {
	rows, err := db.Query(fmt.Sprintf("pragma table_info(%s);", tableName))
	if err != nil {
		return fmt.Errorf("read table info for %s: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return fmt.Errorf("scan table info for %s: %w", tableName, err)
		}
		if name == columnName {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table info for %s: %w", tableName, err)
	}

	if _, err := db.Exec(fmt.Sprintf("alter table %s add column %s %s;", tableName, columnName, columnDDL)); err != nil {
		return fmt.Errorf("add %s.%s column: %w", tableName, columnName, err)
	}
	return nil
}
