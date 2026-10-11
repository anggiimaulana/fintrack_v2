// Package audit records an append-only trail of administrative and billing
// actions. Writes are asynchronous (buffered channel + batch worker) so request
// latency is not affected, and sensitive fields are masked before persistence.
package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Event describes a single auditable action.
type Event struct {
	ActorID   *uuid.UUID     `json:"actor_id,omitempty"`
	Action    string         `json:"action"`
	Entity    string         `json:"entity"`
	EntityID  string         `json:"entity_id,omitempty"`
	Before    map[string]any `json:"before,omitempty"`
	After     map[string]any `json:"after,omitempty"`
	IP        string         `json:"ip,omitempty"`
	UserAgent string         `json:"user_agent,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// Repository persists audit events. Implementations must be append-only.
type Repository interface {
	InsertBatch(ctx context.Context, events []Event) error
}

// Service enqueues audit events for asynchronous persistence.
type Service struct {
	repo    Repository
	ch      chan Event
	done    chan struct{}
	stopped chan struct{}
}

// Config tunes the asynchronous worker.
type Config struct {
	// Buffer is the size of the in-memory queue (default 2048).
	Buffer int
	// BatchSize is the max events per DB insert (default 100).
	BatchSize int
	// FlushInterval is the max time before a partial batch is flushed
	// (default 500ms).
	FlushInterval time.Duration
}

// DefaultConfig returns the documented defaults.
func DefaultConfig() Config {
	return Config{Buffer: 2048, BatchSize: 100, FlushInterval: 500 * time.Millisecond}
}

// NewService starts the background worker. Call Close() to flush and stop it.
func NewService(repo Repository, cfg Config) *Service {
	if cfg.Buffer <= 0 {
		cfg.Buffer = 2048
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 500 * time.Millisecond
	}
	s := &Service{
		repo:    repo,
		ch:      make(chan Event, cfg.Buffer),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go s.run(cfg)
	return s
}

// Log enqueues an event. It never blocks the caller: if the queue is full the
// event is dropped (a warning is emitted by the worker's caller policy).
// Masking is applied here so sensitive values are not retained.
func (s *Service) Log(_ context.Context, e Event) {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	e.Before = Mask(e.Before)
	e.After = Mask(e.After)

	select {
	case s.ch <- e:
	default:
		// Queue full: drop rather than block request handling. A metric/alert
		// on this condition is added in later phases.
	}
}

// Close flushes remaining events and stops the worker.
func (s *Service) Close() {
	close(s.done)
	<-s.stopped
}

func (s *Service) run(cfg Config) {
	defer close(s.stopped)

	ticker := time.NewTicker(cfg.FlushInterval)
	defer ticker.Stop()

	batch := make([]Event, 0, cfg.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.repo.InsertBatch(ctx, batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-s.done:
			// Drain remaining queued events before exiting.
			for {
				select {
				case e := <-s.ch:
					batch = append(batch, e)
					if len(batch) >= cfg.BatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		case e := <-s.ch:
			batch = append(batch, e)
			if len(batch) >= cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}
