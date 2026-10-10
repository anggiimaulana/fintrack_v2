package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/pkg/logger"
)

// coalesceWindow is how long a worker waits before recalculating, so a burst of
// mutations for the same user collapses into a single recalculation.
const coalesceWindow = 250 * time.Millisecond

// SPKQueue deduplicates and coalesces SPK recalculation jobs. Each user has at
// most one job in flight; enqueues that arrive while a job is queued or running
// are folded into a single follow-up run. A fixed worker pool drains the queue,
// so mutation handlers never spawn unbounded goroutines.
type SPKQueue struct {
	mu      sync.Mutex
	pending map[uuid.UUID]bool // queued or running
	dirty   map[uuid.UUID]bool // enqueue arrived while pending

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
		dirty:     make(map[uuid.UUID]bool),
		jobs:      make(chan uuid.UUID, 256),
		workers:   workers,
		calculate: calculate,
	}
}

// Enqueue schedules a recalculation for userID. If one is already pending or
// running, the request is coalesced (marked dirty) instead of queued again.
func (q *SPKQueue) Enqueue(userID uuid.UUID) {
	q.mu.Lock()
	if q.pending[userID] {
		q.dirty[userID] = true
		q.mu.Unlock()
		return
	}
	q.pending[userID] = true
	q.mu.Unlock()

	select {
	case q.jobs <- userID:
	default:
		// Queue full: clear the marker so a later mutation can retry.
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
				for {
					// Wait until no new mutation arrives for a full window
					// (debounce), then recalculate exactly once.
					q.mu.Lock()
					q.dirty[uid] = false
					q.mu.Unlock()

					for {
						select {
						case <-ctx.Done():
						case <-time.After(coalesceWindow):
						}
						q.mu.Lock()
						d := q.dirty[uid]
						q.dirty[uid] = false
						q.mu.Unlock()
						if !d {
							break
						}
					}

					q.calculate(ctx, uid)

					// If mutations landed during the calculation, loop again.
					q.mu.Lock()
					d := q.dirty[uid]
					if !d {
						delete(q.pending, uid)
						q.mu.Unlock()
						break
					}
					q.mu.Unlock()
				}
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
