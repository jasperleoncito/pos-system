package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AccessClaims are the JWT claims embedded in every access token.
// TenantID and Role are empty for super admins operating outside a tenant.
type AccessClaims struct {
	UserID       string `json:"sub"`
	TenantID     string `json:"tid,omitempty"`
	Role         string `json:"role,omitempty"`
	SessionID    string `json:"sid"`
	IsSuperAdmin bool   `json:"is_super,omitempty"`
	TokenType    string `json:"typ"`
	jwt.RegisteredClaims
}

// Manager signs and verifies access tokens and mints refresh tokens.
type Manager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// NewAccessToken signs a JWT for the given identity.
func (m *Manager) NewAccessToken(userID, tenantID, role, sessionID string, isSuperAdmin bool) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		UserID:       userID,
		TenantID:     tenantID,
		Role:         role,
		SessionID:    sessionID,
		IsSuperAdmin: isSuperAdmin,
		TokenType:    "access",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// ParseAccessToken verifies signature, expiry, and token type.
func (m *Manager) ParseAccessToken(raw string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}
	if !tok.Valid || claims.TokenType != "access" {
		return nil, fmt.Errorf("invalid access token")
	}
	return claims, nil
}

// RefreshTTL exposes the configured refresh window for session expiry rows.
func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

// NewRefreshToken returns an opaque random token and its SHA-256 hash.
// Only the hash is persisted; the raw value goes to the client once.
func NewRefreshToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, HashToken(raw), nil
}

// HashToken hashes an opaque token for storage/lookup.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
