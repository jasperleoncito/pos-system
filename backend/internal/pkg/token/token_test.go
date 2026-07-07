package token

import (
	"testing"
	"time"
)

func newTestManager(accessTTL time.Duration) *Manager {
	return NewManager("test-secret-at-least-32-characters!!", accessTTL, 24*time.Hour)
}

func TestAccessTokenRoundTrip(t *testing.T) {
	// Arrange
	m := newTestManager(time.Minute)

	// Act
	raw, err := m.NewAccessToken("user-1", "tenant-1", "manager", "sess-1", false)
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}
	claims, err := m.ParseAccessToken(raw)

	// Assert
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.UserID != "user-1" || claims.TenantID != "tenant-1" ||
		claims.Role != "manager" || claims.SessionID != "sess-1" || claims.IsSuperAdmin {
		t.Errorf("claims mismatch: %+v", claims)
	}
}

func TestParseAccessTokenRejectsExpired(t *testing.T) {
	m := newTestManager(-time.Minute) // already expired

	raw, err := m.NewAccessToken("user-1", "", "", "sess-1", false)
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}

	if _, err := m.ParseAccessToken(raw); err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestParseAccessTokenRejectsWrongSecret(t *testing.T) {
	m := newTestManager(time.Minute)
	other := NewManager("a-completely-different-secret-value!", time.Minute, 24*time.Hour)

	raw, err := m.NewAccessToken("user-1", "", "", "sess-1", false)
	if err != nil {
		t.Fatalf("NewAccessToken: %v", err)
	}

	if _, err := other.ParseAccessToken(raw); err == nil {
		t.Error("expected error for wrong secret, got nil")
	}
}

func TestParseAccessTokenRejectsGarbage(t *testing.T) {
	m := newTestManager(time.Minute)
	if _, err := m.ParseAccessToken("not-a-jwt"); err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

func TestNewRefreshTokenHashMatches(t *testing.T) {
	raw, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	if raw == "" || hash == "" {
		t.Fatal("expected non-empty token and hash")
	}
	if HashToken(raw) != hash {
		t.Error("HashToken(raw) does not match returned hash")
	}
}

func TestNewRefreshTokenIsUnique(t *testing.T) {
	a, _, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	b, _, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	if a == b {
		t.Error("expected unique refresh tokens")
	}
}
