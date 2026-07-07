package redis

import (
	"context"
	"encoding/json"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Cache is a small JSON cache with prefix invalidation, used by the
// analytics dashboard (2–5 minute TTLs).
type Cache struct {
	client *goredis.Client
}

func NewCache(client *goredis.Client) *Cache { return &Cache{client: client} }

// GetJSON loads a cached value into target; ok is false on miss.
func (c *Cache) GetJSON(ctx context.Context, key string, target any) bool {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, target) == nil
}

// SetJSON stores a value with a TTL; failures are ignored (cache only).
func (c *Cache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	c.client.Set(ctx, key, raw, ttl)
}

// DeletePrefix removes every key under prefix (SCAN + DEL batches).
func (c *Cache) DeletePrefix(ctx context.Context, prefix string) {
	iter := c.client.Scan(ctx, 0, prefix+"*", 200).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 200 {
			c.client.Del(ctx, keys...)
			keys = keys[:0]
		}
	}
	if len(keys) > 0 {
		c.client.Del(ctx, keys...)
	}
}
