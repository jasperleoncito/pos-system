// Package auth defines identity entities and the persistence contracts
// used by the authentication service.
package auth

import (
	"context"
	"time"
)

type User struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	FullName        string     `json:"full_name"`
	Phone           string     `json:"phone"`
	AvatarKey       string     `json:"avatar_key"`
	IsSuperAdmin    bool       `json:"is_super_admin"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type DeviceSession struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	RefreshTokenHash string     `json:"-"`
	DeviceName       string     `json:"device_name"`
	UserAgent        string     `json:"user_agent"`
	IP               string     `json:"ip"`
	LastUsedAt       time.Time  `json:"last_used_at"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, u *User) error
	MarkEmailVerified(ctx context.Context, id string, at time.Time) error
	UpdatePassword(ctx context.Context, id, passwordHash string) error
}

type SessionRepository interface {
	Create(ctx context.Context, s *DeviceSession) error
	GetByTokenHash(ctx context.Context, hash string) (*DeviceSession, error)
	ListActiveByUser(ctx context.Context, userID string) ([]DeviceSession, error)
	// Rotate atomically replaces the refresh hash on an existing session.
	Rotate(ctx context.Context, sessionID, newHash string, expiresAt time.Time) error
	Revoke(ctx context.Context, sessionID string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}
