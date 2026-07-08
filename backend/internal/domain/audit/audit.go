// Package audit defines the append-only audit trail contract.
package audit

import (
	"context"
	"time"
)

type Log struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id,omitempty"`
	UserID     string         `json:"user_id,omitempty"`
	UserName   string         `json:"user_name,omitempty"` // joined for the viewer
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Before     map[string]any `json:"before,omitempty"`
	After      map[string]any `json:"after,omitempty"`
	IP         string         `json:"ip"`
	UserAgent  string         `json:"user_agent"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Repository interface {
	Insert(ctx context.Context, l *Log) error
	List(ctx context.Context, tenantID string, limit, offset int) ([]Log, int64, error)
}
