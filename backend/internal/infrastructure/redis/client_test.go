package redis

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestNew_EmptyURL verifies that an empty URL yields a disabled client without
// error, so the application can degrade gracefully.
func TestNew_EmptyURL(t *testing.T) {
	c, err := New(context.Background(), Config{})
	if err != nil {
		t.Fatalf("expected no error for empty URL, got %v", err)
	}
	if c.Enabled() {
		t.Fatal("expected disabled client for empty URL")
	}
	if c.Raw() != nil {
		t.Fatal("expected nil raw client when disabled")
	}
	if err := c.Ping(context.Background()); err != ErrDisabled {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close on disabled client should be nil, got %v", err)
	}
}

// TestNew_InvalidURL verifies that a malformed URL is reported and the client
// stays disabled.
func TestNew_InvalidURL(t *testing.T) {
	c, err := New(context.Background(), Config{URL: "::not-a-url::"})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if c.Enabled() {
		t.Fatal("expected disabled client for invalid URL")
	}
}

// TestNew_Unreachable verifies graceful degradation when Redis cannot be reached.
func TestNew_Unreachable(t *testing.T) {
	// Reserved TEST-NET-1 address; connection will fail fast.
	cfg := Config{
		URL:          "redis://192.0.2.1:6379/0",
		DialTimeout:  300 * time.Millisecond,
		ReadTimeout:  300 * time.Millisecond,
		WriteTimeout: 300 * time.Millisecond,
	}
	c, err := New(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for unreachable Redis")
	}
	if c.Enabled() {
		t.Fatal("expected disabled client when unreachable")
	}
}

// TestNew_Enabled is env-gated: it runs only when REDIS_URL is set (e.g. in CI
// or a local stack with Redis running on :6380).
func TestNew_Enabled(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set — skipping live Redis test")
	}
	c, err := New(context.Background(), Config{
		URL:          url,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("expected successful connection to %s, got %v", url, err)
	}
	defer c.Close()

	if !c.Enabled() {
		t.Fatal("expected enabled client")
	}
	if c.Raw() == nil {
		t.Fatal("expected non-nil raw client")
	}
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}
