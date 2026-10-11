package redis

import (
	"context"
	"sync"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// Channel names used for cross-instance coordination.
const (
	// ChannelCacheInvalidate broadcasts cache invalidation events so each
	// instance can drop its local (L1) lookups.
	ChannelCacheInvalidate = "cache:invalidate"
)

// PubSub is a thin wrapper around Redis pub/sub. When Redis is disabled, Publish
// is a no-op and Subscribe returns a nil channel, so callers can degrade
// gracefully (single-instance behaviour).
type PubSub struct {
	client *Client
}

// NewPubSub builds a PubSub bound to the given client.
func NewPubSub(client *Client) *PubSub {
	return &PubSub{client: client}
}

// Enabled reports whether pub/sub is active.
func (p *PubSub) Enabled() bool {
	return p != nil && p.client != nil && p.client.Enabled()
}

// Publish sends msg to channel. Returns nil when pub/sub is disabled.
func (p *PubSub) Publish(ctx context.Context, channel, msg string) error {
	if !p.Enabled() {
		return ErrDisabled
	}
	return p.client.Raw().Publish(ctx, channel, msg).Err()
}

// Subscribe listens on channel and invokes handler for each message until ctx is
// cancelled. It returns immediately with a wait function; the returned stop
// function unsubscribes and waits for the handler goroutine to finish.
//
// When pub/sub is disabled, it returns a no-op stop function.
func (p *PubSub) Subscribe(ctx context.Context, channel string, handler func(msg string)) (stop func()) {
	if !p.Enabled() {
		return func() {}
	}

	sub := p.client.Raw().Subscribe(ctx, channel)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer sub.Close()
		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.Error("pubsub handler panic", zap.Any("recover", r))
						}
					}()
					handler(msg.Payload)
				}()
			}
		}
	}()

	return func() {
		_ = sub.Close()
		wg.Wait()
	}
}

// ensure the underlying client type is referenced (compile-time guard).
var _ = (*goredis.Client)(nil)

// SubscribePattern listens on a glob pattern (e.g. "realtime:u:*") and invokes
// handler with the concrete channel and payload for each message. Used where the
// set of channels is unbounded — subscribing per user would mean churning
// subscriptions as clients connect and disconnect.
func (p *PubSub) SubscribePattern(ctx context.Context, pattern string, handler func(channel, msg string)) (stop func()) {
	if !p.Enabled() {
		return func() {}
	}

	sub := p.client.Raw().PSubscribe(ctx, pattern)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer sub.Close()
		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.Error("pubsub pattern handler panic", zap.Any("recover", r))
						}
					}()
					handler(msg.Channel, msg.Payload)
				}()
			}
		}
	}()

	return func() {
		_ = sub.Close()
		wg.Wait()
	}
}
