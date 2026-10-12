package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// Client wraps a go-redis client with a graceful-degradation flag. When Redis
// is disabled or unreachable, Enabled() returns false and callers must skip
// caching (the application keeps working, only without cache).
type Client struct {
	rdb     *goredis.Client
	enabled bool
}

// New creates and verifies a Redis client from cfg.
//
// Behaviour:
//   - cfg.URL empty            => returns a disabled client (nil error).
//   - connect/ping fails       => returns a disabled client + the error so the
//     caller can log a warning and continue (graceful degradation).
//
// The returned client is never nil so callers can defer Close() safely.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.URL == "" {
		logger.Warn("REDIS_URL not set — cache/pubsub disabled (graceful degradation)")
		return &Client{enabled: false}, nil
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 3 * time.Second
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 3 * time.Second
	}
	opt, err := goredis.ParseURL(cfg.URL)
	if err != nil {
		return &Client{enabled: false}, fmt.Errorf("invalid REDIS_URL: %w", err)
	}
	if cfg.PoolSize > 0 {
		opt.PoolSize = cfg.PoolSize
	}
	opt.DialTimeout = cfg.DialTimeout
	opt.ReadTimeout = cfg.ReadTimeout
	opt.WriteTimeout = cfg.WriteTimeout

	rdb := goredis.NewClient(opt)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		// Close the half-open client and degrade gracefully.
		_ = rdb.Close()
		return &Client{enabled: false}, fmt.Errorf("redis ping failed: %w", err)
	}

	logger.Info("Redis client established", zap.Int("pool_size", opt.PoolSize))
	return &Client{rdb: rdb, enabled: true}, nil
}

// Enabled reports whether Redis is connected and usable.
func (c *Client) Enabled() bool {
	return c != nil && c.enabled && c.rdb != nil
}

// Raw returns the underlying go-redis client, or nil when disabled.
func (c *Client) Raw() *goredis.Client {
	if !c.Enabled() {
		return nil
	}
	return c.rdb
}

// Ping verifies connectivity; returns an error when disabled or unreachable.
func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return ErrDisabled
	}
	return c.rdb.Ping(ctx).Err()
}

// Close releases the connection pool. Safe to call on a disabled client.
func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}
