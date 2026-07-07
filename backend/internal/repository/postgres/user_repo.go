package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/auth"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo { return &UserRepo{db: db} }

const userColumns = `id, email, password_hash, full_name, phone, avatar_key,
	is_super_admin, email_verified_at, status, created_at, updated_at`

func scanUser(row pgx.Row) (*auth.User, error) {
	var u auth.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Phone, &u.AvatarKey,
		&u.IsSuperAdmin, &u.EmailVerifiedAt, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("user")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, u *auth.User) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, phone, is_super_admin, email_verified_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		u.Email, u.PasswordHash, u.FullName, u.Phone, u.IsSuperAdmin, u.EmailVerifiedAt, u.Status,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("email is already registered")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*auth.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE lower(email) = lower($1) AND deleted_at IS NULL`, email))
}

func (r *UserRepo) Update(ctx context.Context, u *auth.User) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET full_name = $2, phone = $3, avatar_key = $4, status = $5, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`,
		u.ID, u.FullName, u.Phone, u.AvatarKey, u.Status)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *UserRepo) MarkEmailVerified(ctx context.Context, id string, at time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET email_verified_at = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, at)
	if err != nil {
		return fmt.Errorf("failed to mark email verified: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}
