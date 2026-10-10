package postgres

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
)

// newTestPool connects to TEST_DATABASE_URL or skips.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration tests")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newAuthUC(pool *pgxpool.Pool) *usecase.AuthUsecase {
	userRepo := NewUserRepo(pool)
	refreshRepo := NewRefreshTokenRepo(pool)
	return usecase.NewAuthUsecase(userRepo, refreshRepo, []byte("test-secret-at-least-32-characters-long"))
}

func uniqueEmail() string {
	return "it-" + uuid.NewString() + "@example.com"
}

// TestIntegrationAuthFlow covers register → login → refresh → reuse detection.
func TestIntegrationAuthFlow(t *testing.T) {
	pool := newTestPool(t)
	uc := newAuthUC(pool)
	ctx := t.Context()

	email := uniqueEmail()
	const password = "User@123456"

	if _, err := uc.Register(ctx, email, password, "IT User"); err != nil {
		t.Fatalf("register: %v", err)
	}

	pair, err := uc.Login(ctx, email, password, "test-agent")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("login returned empty tokens")
	}

	// Refresh rotates: new pair differs, old raw token is now revoked.
	pair2, err := uc.Refresh(ctx, pair.RefreshToken, "test-agent")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if pair2.RefreshToken == pair.RefreshToken {
		t.Fatal("refresh did not rotate the token")
	}

	// Reusing the old refresh token must fail AND revoke everything.
	if _, err := uc.Refresh(ctx, pair.RefreshToken, "test-agent"); err == nil {
		t.Fatal("expected reuse of old refresh token to fail")
	}
	// The successor token must also now be revoked (reuse response).
	if _, err := uc.Refresh(ctx, pair2.RefreshToken, "test-agent"); err == nil {
		t.Fatal("expected all tokens to be revoked after reuse detection")
	}
}

// TestIntegrationUserIsolation asserts a user cannot read another user's tx.
func TestIntegrationUserIsolation(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	txRepo := NewTransactionRepo(pool)
	userRepo := NewUserRepo(pool)

	// Create two users.
	uA, err := userRepo.Create(ctx, &domain.Profile{Email: uniqueEmail(), FullName: "A", Role: domain.RoleUser}, "x")
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	uB, err := userRepo.Create(ctx, &domain.Profile{Email: uniqueEmail(), FullName: "B", Role: domain.RoleUser}, "x")
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, uA.ID, uB.ID)
	})

	// Create an income category for A's transaction.
	catRepo := NewIncomeCategoryRepo(pool)
	cat := &domain.IncomeCategory{Name: "IT Cat " + uuid.NewString(), Icon: "x", Color: "#000000"}
	if err := catRepo.Create(ctx, cat); err != nil {
		t.Fatalf("create category: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM income_categories WHERE id = $1`, cat.ID) })

	desc := "A's transaction"
	tx := &domain.Transaction{
		UserID:           uA.ID,
		Type:             domain.TransactionIncome,
		Amount:           100000,
		IncomeCategoryID: &cat.ID,
		Description:      &desc,
		TransactionDate:  time.Now(),
	}
	if err := txRepo.Create(ctx, tx); err != nil {
		t.Fatalf("create tx: %v", err)
	}

	// B must not see A's transaction.
	got, err := txRepo.GetByID(ctx, tx.ID, uB.ID)
	if err != nil {
		t.Fatalf("GetByID(B): %v", err)
	}
	if got != nil {
		t.Fatal("user B was able to read user A's transaction (isolation breach)")
	}

	// A can see it.
	gotA, err := txRepo.GetByID(ctx, tx.ID, uA.ID)
	if err != nil || gotA == nil {
		t.Fatalf("user A should read own transaction: %v", err)
	}
}

// TestIntegrationOptimisticLock asserts a stale update returns ErrConflict.
func TestIntegrationOptimisticLock(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	txRepo := NewTransactionRepo(pool)
	userRepo := NewUserRepo(pool)

	u, err := userRepo.Create(ctx, &domain.Profile{Email: uniqueEmail(), FullName: "L", Role: domain.RoleUser}, "x")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, u.ID) })

	catRepo := NewIncomeCategoryRepo(pool)
	cat := &domain.IncomeCategory{Name: "IT Lock " + uuid.NewString(), Icon: "x", Color: "#000000"}
	_ = catRepo.Create(ctx, cat)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM income_categories WHERE id = $1`, cat.ID) })

	tx := &domain.Transaction{
		UserID:           u.ID,
		Type:             domain.TransactionIncome,
		Amount:           1000,
		IncomeCategoryID: &cat.ID,
		TransactionDate:  time.Now(),
	}
	if err := txRepo.Create(ctx, tx); err != nil {
		t.Fatalf("create tx: %v", err)
	}

	stale := tx.UpdatedAt

	// First update succeeds with the current timestamp.
	amount := 2000.0
	tx.Amount = amount
	if err := txRepo.Update(ctx, tx, &stale); err != nil {
		t.Fatalf("first update should succeed: %v", err)
	}

	// Second update with the same (now stale) timestamp must conflict.
	tx.Amount = 3000
	err = txRepo.Update(ctx, tx, &stale)
	if err != domain.ErrConflict {
		t.Fatalf("expected ErrConflict on stale update, got %v", err)
	}
}

// TestIntegrationSPKRace fires 20 concurrent transaction creations and asserts
// SPK recalculation stays bounded (queue dedupe + advisory lock).
func TestIntegrationSPKRace(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	txRepo := NewTransactionRepo(pool)
	spkRepo := NewSPKRepo(pool)
	userRepo := NewUserRepo(pool)
	txManager := NewTxManager(pool)

	u, err := userRepo.Create(ctx, &domain.Profile{Email: uniqueEmail(), FullName: "R", Role: domain.RoleUser}, "x")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, u.ID) })

	catRepo := NewIncomeCategoryRepo(pool)
	cat := &domain.IncomeCategory{Name: "IT Race " + uuid.NewString(), Icon: "x", Color: "#000000"}
	_ = catRepo.Create(ctx, cat)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM income_categories WHERE id = $1`, cat.ID) })

	spkUC := usecase.NewSPKUsecase(txRepo, spkRepo, txManager)
	embedRepo := NewEmbeddingRepo(pool)
	txUC := usecase.NewTransactionUsecase(txRepo, usecase.NewSPKQueue(4, func(c context.Context, uid uuid.UUID) {
		_, _ = spkUC.Calculate(c, uid)
	}), nil, embedRepo, nil)

	before, _ := spkRepo.ListByUser(ctx, u.ID, 100)
	beforeCount := len(before)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := txUC.Create(ctx, u.ID, domain.CreateTransactionRequest{
				Type:             domain.TransactionIncome,
				Amount:           float64(10000 + n),
				IncomeCategoryID: &cat.ID,
				TransactionDate:  new(time.Now()),
			})
			if err != nil {
				t.Errorf("create tx %d: %v", n, err)
			}
		}(i)
	}
	wg.Wait()

	// Directly recalc once to settle, then assert growth is bounded.
	if _, err := spkUC.Calculate(ctx, u.ID); err != nil {
		t.Fatalf("calculate: %v", err)
	}

	after, _ := spkRepo.ListByUser(ctx, u.ID, 100)
	growth := len(after) - beforeCount
	if growth > 2 {
		t.Fatalf("SPK rows grew by %d (want <=2)", growth)
	}
}

