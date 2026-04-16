package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/zc12120/atomhub/internal/auth"
)

// ErrInvalidCredentials indicates username/password mismatch.
var ErrInvalidCredentials = errors.New("invalid credentials")

// AdminUser represents an admin account.
type AdminUser struct {
	ID           int64
	Username     string
	PasswordHash string
}

// AdminRepository provides admin-user persistence and authentication.
type AdminRepository struct {
	db *sql.DB
}

// NewAdminRepository constructs an admin repository.
func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// EnsureDefaultAdmin creates the default admin user when absent.
func (r *AdminRepository) EnsureDefaultAdmin(ctx context.Context, username, passwordHash string) error {
	username = strings.TrimSpace(username)
	passwordHash = strings.TrimSpace(passwordHash)
	if username == "" || passwordHash == "" {
		return errors.New("username and password hash are required")
	}

	_, err := r.db.ExecContext(
		ctx,
		`insert into admin_users (username, password_hash) values (?, ?) on conflict(username) do nothing`,
		username,
		passwordHash,
	)
	return err
}

// GetByUsername returns the admin user for a username.
func (r *AdminRepository) GetByUsername(ctx context.Context, username string) (AdminUser, error) {
	var user AdminUser
	err := r.db.QueryRowContext(
		ctx,
		`select id, username, password_hash from admin_users where username = ?`,
		strings.TrimSpace(username),
	).Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		return AdminUser{}, err
	}
	return user, nil
}

// Authenticate validates admin credentials.
func (r *AdminRepository) Authenticate(ctx context.Context, username, password string) (AdminUser, error) {
	user, err := r.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AdminUser{}, ErrInvalidCredentials
		}
		return AdminUser{}, err
	}

	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		return AdminUser{}, ErrInvalidCredentials
	}

	return user, nil
}
