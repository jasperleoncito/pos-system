// Package queue defines asynq task types and a thin enqueue client.
// The API enqueues; cmd/worker consumes.
package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Task type names.
const (
	TaskEmailSend       = "email:send"
	TaskLowStock        = "notify:low_stock"
	TaskAttendanceAlert = "notify:attendance"
	TaskDailySummary    = "notify:daily_summary"
)

type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

type LowStockPayload struct {
	TenantID  string  `json:"tenant_id"`
	ItemName  string  `json:"item_name"`
	Stock     float64 `json:"stock"`
	AlertType string  `json:"alert_type"` // low_stock | out_of_stock
}

type AttendanceAlertPayload struct {
	TenantID     string `json:"tenant_id"`
	EmployeeName string `json:"employee_name"`
	LateMinutes  int    `json:"late_minutes"`
	ClockInLocal string `json:"clock_in_local"`
}

type DailySummaryPayload struct {
	TenantID string `json:"tenant_id"`
}

// Client wraps asynq for enqueueing from the API process.
type Client struct {
	inner *asynq.Client
}

func NewClient(redisAddr, redisPassword string) *Client {
	return &Client{inner: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword})}
}

func (c *Client) Close() error { return c.inner.Close() }

func (c *Client) enqueue(taskType string, payload any, opts ...asynq.Option) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal %s payload: %w", taskType, err)
	}
	defaults := []asynq.Option{asynq.MaxRetry(5), asynq.Timeout(30 * time.Second)}
	if _, err := c.inner.Enqueue(asynq.NewTask(taskType, raw), append(defaults, opts...)...); err != nil {
		return fmt.Errorf("failed to enqueue %s: %w", taskType, err)
	}
	return nil
}

// Send implements the transactional-mail contract by queueing the
// email; the worker delivers it via SMTP.
func (c *Client) Send(to, subject, htmlBody string) error {
	return c.enqueue(TaskEmailSend, EmailPayload{To: to, Subject: subject, HTML: htmlBody})
}

func (c *Client) EnqueueLowStock(p LowStockPayload) error {
	return c.enqueue(TaskLowStock, p)
}

func (c *Client) EnqueueAttendanceAlert(p AttendanceAlertPayload) error {
	return c.enqueue(TaskAttendanceAlert, p)
}
