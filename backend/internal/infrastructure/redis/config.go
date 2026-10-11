// Package redis provides a shared Redis client used for versioned caching,
// pub/sub invalidation, and cross-instance coordination.
//
// The client is designed for graceful degradation: if REDIS_URL is unset or
// Redis is unreachable at boot, the client reports Enabled() == false and the
// application continues without caching rather than failing to start.
package redis

import "time"

// Config holds Redis connection settings.
type Config struct {
	// URL is the Redis connection string (e.g. redis://localhost:6380/0).
	// Empty URL disables Redis entirely (graceful degradation).
	URL string
	// PoolSize is the maximum number of socket connections.
	PoolSize int
	// DialTimeout is the timeout for establishing a new connection.
	DialTimeout time.Duration
	// ReadTimeout is the timeout for socket reads.
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for socket writes.
	WriteTimeout time.Duration
}

// ConfigFromEnv builds a Config from environment variables.
//
//	REDIS_URL          connection string (empty => disabled)
//	REDIS_POOL_SIZE    max connections (default 20)
func ConfigFromEnv(getenv func(string) string, atoi func(string, int) int) Config {
	return Config{
		URL:          getenv("REDIS_URL"),
		PoolSize:     atoi(getenv("REDIS_POOL_SIZE"), 20),
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}
