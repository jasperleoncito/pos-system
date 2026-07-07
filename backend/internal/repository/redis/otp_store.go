// Package redis contains Redis-backed stores for ephemeral auth state.
package redis

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// TokenStore keeps one-time tokens (password reset, email verification)
// in Redis with a TTL, keyed by purpose.
type TokenStore struct {
	client *goredis.Client
}

func NewTokenStore(client *goredis.Client) *TokenStore {
	return &TokenStore{client: client}
}

func key(purpose, token string) string {
	return fmt.Sprintf("otp:%s:%s", purpose, token)
}

// Issue creates a random one-time token bound to a user ID.
func (s *TokenStore) Issue(ctx context.Context, purpose, userID string, ttl time.Duration) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	if err := s.client.Set(ctx, key(purpose, token), userID, ttl).Err(); err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}
	return token, nil
}

// Consume validates and deletes a one-time token, returning the user ID
// it was issued for. Returns empty string when invalid or expired.
func (s *TokenStore) Consume(ctx context.Context, purpose, token string) (string, error) {
	userID, err := s.client.GetDel(ctx, key(purpose, token)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to consume token: %w", err)
	}
	return userID, nil
}
