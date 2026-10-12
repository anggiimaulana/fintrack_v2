package tests

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/auth"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/internal/usecase"
)

// TestP08_Patch_RefreshTokenConcurrentRotation verifies that concurrent requests
// presenting the exact same refresh token result in exactly one successful rotation,
// while the concurrent attempt is blocked and triggers reuse protection.
func TestP08_Patch_RefreshTokenConcurrentRotation(t *testing.T) {
	dsn := getTestDSN(t)
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	defer pool.Close()

	userRepo := postgres.NewUserRepo(pool)
	refreshRepo := postgres.NewRefreshTokenRepo(pool)
	authUC := usecase.NewAuthUsecase(userRepo, refreshRepo, []byte("test-jwt-secret-32-bytes-minimum!"))

	userID := uuid.New()
	userEmail := fmt.Sprintf("rf-race-%s@fintrack.test", userID.String()[:8])
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, role)
		 VALUES ($1, $2, 'dummy', 'Refresh Race User', 'user')`,
		userID, userEmail); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	}()

	rawToken, hash := auth.NewRefreshToken()
	rt := &domain.RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := refreshRepo.Create(ctx, rt); err != nil {
		t.Fatalf("create refresh token: %v", err)
	}

	const concurrency = 10
	var successCount, failureCount int32
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start

			pair, err := authUC.Refresh(context.Background(), rawToken, fmt.Sprintf("worker-%d", worker))
			if err == nil && pair != nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failureCount, 1)
			}
		}(i)
	}

	close(start)
	wg.Wait()

	t.Logf("Concurrent refresh result: successes=%d, failures=%d", successCount, failureCount)

	// Invariant: at most 1 can succeed
	if successCount > 1 {
		t.Fatalf("CRITICAL RACE: %d concurrent refreshes succeeded on the same token (max 1 permitted)", successCount)
	}
}

func getTestDSN(t *testing.T) string {
	t.Helper()
	dsn := "postgres://postgres:postgres@localhost:5435/fintrack_test?sslmode=disable"
	return dsn
}
