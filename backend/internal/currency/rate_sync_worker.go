package currency

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/runtimeconfig"
	"github.com/fintrack/backend/pkg/logger"
)

// RateSyncWorker periodically syncs exchange rates for currencies configured
// with rate_source = 'provider'. All knobs (interval, enabled, dedup window)
// are dynamically resolved via runtimeconfig.Resolver.
type RateSyncWorker struct {
	repo     Repo
	factory  *ProviderFactory
	resolver *runtimeconfig.Resolver
	stop     chan struct{}
	done     chan struct{}
}

// NewRateSyncWorker constructs a worker with the given repo and provider factory.
func NewRateSyncWorker(repo Repo, factory *ProviderFactory, resolver *runtimeconfig.Resolver) *RateSyncWorker {
	return &RateSyncWorker{
		repo:     repo,
		factory:  factory,
		resolver: resolver,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start launches the background synchronization loop.
func (w *RateSyncWorker) Start(ctx context.Context) {
	go func() {
		defer close(w.done)

		// Brief initial delay before first sweep so startup is not delayed
		select {
		case <-time.After(5 * time.Second):
			w.SyncOnce(ctx)
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		}

		for {
			intervalHours := 6
			if w.resolver != nil {
				intervalHours = w.resolver.Int(ctx, "currency_sync_interval_hours", "CURRENCY_SYNC_INTERVAL_HOURS", 6)
			}
			if intervalHours <= 0 {
				intervalHours = 6
			}
			interval := time.Duration(intervalHours) * time.Hour

			select {
			case <-time.After(interval):
				w.SyncOnce(ctx)
			case <-ctx.Done():
				return
			case <-w.stop:
				return
			}
		}
	}()
}

// Stop signals the background worker to stop and waits for it to exit.
func (w *RateSyncWorker) Stop() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	<-w.done
}

// SyncOnce performs a single synchronization pass across all active provider-backed currencies.
func (w *RateSyncWorker) SyncOnce(ctx context.Context) (int, error) {
	enabled := true
	if w.resolver != nil {
		enabled = w.resolver.Bool(ctx, "currency_auto_sync_enabled", "CURRENCY_AUTO_SYNC_ENABLED", true)
	}
	if !enabled {
		return 0, nil
	}

	dedupHours := 24
	if w.resolver != nil {
		dedupHours = w.resolver.Int(ctx, "currency_dedup_hours", "CURRENCY_DEDUP_HOURS", 24)
	}
	if dedupHours <= 0 {
		dedupHours = 24
	}

	provider, err := w.factory.ResolveProvider(ctx)
	if err != nil {
		logger.Warn("rate sync worker: failed to resolve provider", zap.Error(err))
		return 0, err
	}

	currencies, err := w.repo.List(ctx, false)
	if err != nil {
		logger.Warn("rate sync worker: failed to list currencies", zap.Error(err))
		return 0, err
	}

	now := time.Now()
	updatedCount := 0

	for _, c := range currencies {
		if c.IsBase || c.RateSource != "provider" {
			continue
		}

		fetchedRate, ferr := provider.FetchRate(ctx, c.Code, BaseCode)
		if ferr != nil {
			logger.Warn("rate sync worker: fetch rate failed, using fallback",
				zap.String("currency", c.Code),
				zap.String("provider", provider.Name()),
				zap.Error(ferr))
			continue
		}
		if fetchedRate <= 0 {
			logger.Warn("rate sync worker: provider returned non-positive rate",
				zap.String("currency", c.Code),
				zap.Float64("rate", fetchedRate))
			continue
		}

		// Deduplication check: compare with newest rate
		existing, rerr := w.repo.RateAsOf(ctx, c.Code, now)
		if rerr == nil && existing != nil {
			isWithinDedupWindow := now.Sub(existing.EffectiveAt) < time.Duration(dedupHours)*time.Hour
			isIdenticalRate := math.Abs(fetchedRate-existing.RateToBase) < 1e-4
			if isWithinDedupWindow && isIdenticalRate {
				// Rate unchanged within deduplication window, skip insert
				continue
			}
		}

		note := fmt.Sprintf("scheduled-sync from %s", provider.Name())
		newRate := Rate{
			CurrencyCode: c.Code,
			RateToBase:   fetchedRate,
			EffectiveAt:  now,
			Source:       "provider",
			Note:         &note,
		}

		if err := w.repo.InsertRate(ctx, newRate); err != nil {
			logger.Warn("rate sync worker: failed to insert new rate",
				zap.String("currency", c.Code),
				zap.Error(err))
			continue
		}

		updatedCount++
		logger.Info("rate sync worker: updated currency rate",
			zap.String("currency", c.Code),
			zap.Float64("rate", fetchedRate),
			zap.String("provider", provider.Name()))
	}

	return updatedCount, nil
}
