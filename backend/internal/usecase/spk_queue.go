package usecase

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// SPKQueue deduplicates and serializes SPK recalculation jobs. Each user has at
// most one pending job; a fixed worker pool drains the queue, so mutation
// handlers never spawn unbounded goroutines.
type SPKQueue struct {
	pending map[uuid.UUID]bool
	mu      sync.Mutex
	jobs    chan uuid.UUID
	wg      sync.WaitGroup
	workers int
	started bool

	calculate func(ctx context.Context, userID uuid.UUID)
}

// NewSPKQueue builds a queue with the given worker count.
func NewSPKQueue(workers int, calculate func(ctx context.Context, userID uuid.UUID)) *SPKQueue {
	if workers < 1 {
		workers = 4
	}
	return &SPKQueue{
		pending:   make(map[uuid.UUID]bool),
		jobs:      make(chan uuid.UUID, 256),
		workers:   workers,
		calculate: calculate,
	}
}

// Enqueue schedules a recalculation for userID, skipping if one is already
// pending for that user.
func (q *SPKQueue) Enqueue(userID uuid.UUID) {
	q.mu.Lock()
	if q.pending[userID] {
		q.mu.Unlock()
		return
	}
	q.pending[userID] = true
	q.mu.Unlock()

	select {
	case q.jobs <- userID:
	default:
		// Queue full: drop the marker so a later mutation can retry.
		q.mu.Lock()
		delete(q.pending, userID)
		q.mu.Unlock()
		logger.Warn("SPK queue full, dropping job", zap.String("user_id", userID.String()))
	}
}

// Start launches the worker pool.
func (q *SPKQueue) Start(ctx context.Context) {
	q.mu.Lock()
	if q.started {
		q.mu.Unlock()
		return
	}
	q.started = true
	q.mu.Unlock()

	for range q.workers {
		q.wg.Add(1)
		go func() {
			defer q.wg.Done()
			for uid := range q.jobs {
				q.mu.Lock()
				delete(q.pending, uid)
				q.mu.Unlock()

				q.calculate(ctx, uid)
			}
		}()
	}
	logger.Info("SPK queue started", zap.Int("workers", q.workers))
}

// Stop drains the queue and waits for in-flight jobs to finish.
func (q *SPKQueue) Stop() {
	close(q.jobs)
	q.wg.Wait()
	logger.Info("SPK queue stopped")
}
