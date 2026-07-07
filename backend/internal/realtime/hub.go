// Package realtime fans kitchen events out to SSE subscribers. Events
// travel through Redis pub/sub so every backend replica sees them.
package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	goredis "github.com/redis/go-redis/v9"
)

const channelPrefix = "kitchen:"

// Event is the payload pushed to kitchen displays. Clients treat events
// as refresh hints and refetch the queue.
type Event struct {
	Type        string `json:"type"` // order_fired | kitchen_status | item_status | priority
	OrderID     string `json:"order_id"`
	OrderNumber int64  `json:"order_number,omitempty"`
	Value       string `json:"value,omitempty"`
}

type subscriber struct {
	tenantID string
	ch       chan Event
}

// Hub bridges Redis pub/sub to in-process SSE subscribers.
type Hub struct {
	redis  *goredis.Client
	logger *slog.Logger

	mu   sync.RWMutex
	subs map[*subscriber]struct{}
}

func NewHub(redis *goredis.Client, logger *slog.Logger) *Hub {
	return &Hub{redis: redis, logger: logger, subs: map[*subscriber]struct{}{}}
}

// Run consumes Redis pub/sub until ctx is cancelled. Call in a goroutine.
func (h *Hub) Run(ctx context.Context) {
	pubsub := h.redis.PSubscribe(ctx, channelPrefix+"*")
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-pubsub.Channel():
			if !ok {
				return
			}
			tenantID := strings.TrimPrefix(msg.Channel, channelPrefix)
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				h.logger.Warn("failed to decode kitchen event", "error", err)
				continue
			}
			h.dispatch(tenantID, event)
		}
	}
}

func (h *Hub) dispatch(tenantID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for sub := range h.subs {
		if sub.tenantID != tenantID {
			continue
		}
		select {
		case sub.ch <- event:
		default:
			// Slow consumer: drop rather than block the hub. The client
			// refetches on the next event or poll.
		}
	}
}

// Subscribe registers an SSE consumer. Call the returned func to leave.
func (h *Hub) Subscribe(tenantID string) (<-chan Event, func()) {
	sub := &subscriber{tenantID: tenantID, ch: make(chan Event, 16)}
	h.mu.Lock()
	h.subs[sub] = struct{}{}
	h.mu.Unlock()

	return sub.ch, func() {
		h.mu.Lock()
		delete(h.subs, sub)
		h.mu.Unlock()
		close(sub.ch)
	}
}

// Publish sends an event to every replica via Redis. Failures are
// logged, never fatal — realtime is best-effort on top of polling.
func (h *Hub) Publish(ctx context.Context, tenantID string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("failed to encode kitchen event", "error", err)
		return
	}
	if err := h.redis.Publish(ctx, channelPrefix+tenantID, payload).Err(); err != nil {
		h.logger.Warn("failed to publish kitchen event", "error", err)
	}
}
