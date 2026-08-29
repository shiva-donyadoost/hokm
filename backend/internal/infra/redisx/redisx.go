// Package redisx provides Redis-backed coordination utilities: rate
// limiting and presence. Failures degrade gracefully (fail open) so a Redis
// outage never blocks gameplay.
package redisx

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewClient connects and pings; returns nil-safe client on failure with err.
func NewClient(ctx context.Context, addr string) (*redis.Client, error) {
	c := redis.NewClient(&redis.Options{Addr: addr})
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return c, nil
}

// RateLimiter is a fixed-window counter per key.
type RateLimiter struct {
	client *redis.Client
	window time.Duration
	limit  int64
}

func NewRateLimiter(client *redis.Client, window time.Duration, limit int64) *RateLimiter {
	return &RateLimiter{client: client, window: window, limit: limit}
}

// Allow records one hit for key and reports whether the limit permits it.
// A Redis failure fails OPEN (allows the request) — availability over
// strictness for a game server.
func (r *RateLimiter) Allow(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	windowKey := fmt.Sprintf("ratelimit:%s:%d", key, time.Now().Unix()/int64(r.window.Seconds()))
	n, err := r.client.Incr(ctx, windowKey).Result()
	if err != nil {
		return true
	}
	if n == 1 {
		_ = r.client.Expire(ctx, windowKey, r.window+time.Second).Err()
	}
	return n <= r.limit
}

// Presence marks a user active with a TTL and reports prior state.
type Presence struct {
	client *redis.Client
	ttl    time.Duration
}

func NewPresence(client *redis.Client, ttl time.Duration) *Presence {
	return &Presence{client: client, ttl: ttl}
}

// Touch records activity for the user; returns whether they were already
// marked present.
func (p *Presence) Touch(userID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	key := "presence:" + userID
	existed, _ := p.client.Exists(ctx, key).Result()
	_ = p.client.Set(ctx, key, time.Now().UnixMilli(), p.ttl).Err()
	return existed > 0
}

// IsPresent reports whether the user is currently marked present.
func (p *Presence) IsPresent(userID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	n, _ := p.client.Exists(ctx, "presence:"+userID).Result()
	return n > 0
}
