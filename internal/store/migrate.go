package store

import "database/sql"

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
		`create table if not exists request_logs (
            id integer primary key autoincrement,
            key_id integer not null,
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
	return nil
}
