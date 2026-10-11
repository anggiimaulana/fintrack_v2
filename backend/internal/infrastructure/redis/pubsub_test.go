package redis

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestPubSub_Disabled verifies graceful behaviour without Redis.
func TestPubSub_Disabled(t *testing.T) {
	ps := NewPubSub(nil)
	if ps.Enabled() {
		t.Fatal("expected disabled pubsub for nil client")
	}
	if err := ps.Publish(context.Background(), ChannelCacheInvalidate, "x"); err != ErrDisabled {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
	stop := ps.Subscribe(context.Background(), ChannelCacheInvalidate, func(string) {})
	stop() // must not panic
}

// TestPubSub_PublishSubscribe is env-gated: it verifies cross-subscriber
// delivery against a live Redis.
func TestPubSub_PublishSubscribe(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set — skipping live pub/sub test")
	}
	c, err := New(context.Background(), Config{URL: url, DialTimeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	ps := NewPubSub(c)
	if !ps.Enabled() {
		t.Fatal("expected enabled pubsub")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received atomic.Int32
	got := make(chan string, 1)
	stop := ps.Subscribe(ctx, ChannelCacheInvalidate, func(msg string) {
		received.Add(1)
		select {
		case got <- msg:
		default:
		}
	})
	defer stop()

	// Give the subscriber a moment to attach, then publish.
	time.Sleep(100 * time.Millisecond)
	if err := ps.Publish(context.Background(), ChannelCacheInvalidate, "u:abc"); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case msg := <-got:
		if msg != "u:abc" {
			t.Fatalf("expected u:abc, got %q", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pub/sub message")
	}
}
