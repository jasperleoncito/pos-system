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

type SessionRepo struct {
	db *pgxpool.Pool
}

func NewSessionRepo(db *pgxpool.Pool) *SessionRepo { return &SessionRepo{db: db} }

const sessionColumns = `id, user_id, refresh_token_hash, device_name, user_agent, ip,
	last_used_at, expires_at, revoked_at, created_at`

func scanSession(row pgx.Row) (*auth.DeviceSession, error) {
	var s auth.DeviceSession
	err := row.Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.DeviceName, &s.UserAgent, &s.IP,
		&s.LastUsedAt, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("session")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}
	return &s, nil
}

func (r *SessionRepo) Create(ctx context.Context, s *auth.DeviceSession) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO device_sessions (user_id, refresh_token_hash, device_name, user_agent, ip, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, last_used_at, created_at`,
		s.UserID, s.RefreshTokenHash, s.DeviceName, s.UserAgent, s.IP, s.ExpiresAt,
	).Scan(&s.ID, &s.LastUsedAt, &s.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *SessionRepo) GetByTokenHash(ctx context.Context, hash string) (*auth.DeviceSession, error) {
	return scanSession(r.db.QueryRow(ctx, `
		SELECT `+sessionColumns+` FROM device_sessions
		WHERE refresh_token_hash = $1 AND revoked_at IS NULL AND deleted_at IS NULL`, hash))
}

func (r *SessionRepo) ListActiveByUser(ctx context.Context, userID string) ([]auth.DeviceSession, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+sessionColumns+` FROM device_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now() AND deleted_at IS NULL
		ORDER BY last_used_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []auth.DeviceSession
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *s)
	}
	return sessions, rows.Err()
}

func (r *SessionRepo) Rotate(ctx context.Context, sessionID, newHash string, expiresAt time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE device_sessions
		SET refresh_token_hash = $2, expires_at = $3, last_used_at = now(), updated_at = now()
		WHERE id = $1 AND revoked_at IS NULL AND deleted_at IS NULL`,
		sessionID, newHash, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to rotate session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.Unauthorized("session is no longer active")
	}
	return nil
}

func (r *SessionRepo) Revoke(ctx context.Context, sessionID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE device_sessions SET revoked_at = now(), updated_at = now() WHERE id = $1 AND revoked_at IS NULL`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}
	return nil
}

func (r *SessionRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE device_sessions SET revoked_at = now(), updated_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}
	return nil
}
