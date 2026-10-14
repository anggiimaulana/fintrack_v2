package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/billing"
	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/internal/infrastructure/auth"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/internal/usecase"
)

// TestP08_001_TokenQuotaConcurrency simulates 20 concurrent requests hitting
// /api/v1/ai/chat with a remaining balance of exactly 5 tokens.
// Invariant: exactly 5 succeed (HTTP 200), exactly 15 are blocked with HTTP 429
// (insufficient_quota), zero 5xx errors, token balance never drops below zero,
// and exactly 5 ledger records are written.
func TestP08_001_TokenQuotaConcurrency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping quota concurrency test")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	seedPlanCatalogue(t, pool)

	billingRepo := postgres.NewBillingRepo(pool)
	txRepo := postgres.NewTransactionRepo(pool)
	txManager := postgres.NewTxManager(pool)
	entResolver := entitlement.NewResolver(postgres.NewEntitlementSource(billingRepo), nil)
	entMw := middleware.NewEntitlementMiddleware(entResolver)
	usage := &middleware.UsageProviders{
		CountTransactions: txRepo.CountActiveByUser,
		CurrentTokens:     middleware.TokenPeriodUsage(billingRepo),
		CountVisionScans: func(ctx context.Context, userID uuid.UUID) (int64, error) {
			return billingRepo.CountVisionScansInPeriod(ctx, userID, time.Now().UTC().Unix())
		},
	}

	quotaResolver := billing.QuotaResolverFunc(func(ctx context.Context, userID uuid.UUID, at time.Time) (time.Time, time.Time, int64, error) {
		res, err := entResolver.Resolve(ctx, userID)
		if err != nil {
			return time.Time{}, time.Time{}, 0, err
		}
		quota := res.Limit("ai_tokens_per_cycle")
		if quota < 0 {
			quota = 1000000000
		}
		sub, _ := billingRepo.GetByUserID(ctx, userID)
		if sub != nil && sub.CurrentPeriodEnd != nil {
			return sub.CurrentPeriodStart, *sub.CurrentPeriodEnd, quota, nil
		}
		start := at.Truncate(24 * time.Hour)
		return start, start.AddDate(0, 0, 30), quota, nil
	})
	ledgerService := billing.NewLedgerService(billingRepo, txManager, quotaResolver, nil)

	aiUC := usecase.NewAIUsecase(
		postgres.NewChatSessionRepo(pool),
		postgres.NewChatMessageRepo(pool),
		txRepo,
		postgres.NewIncomeCategoryRepo(pool),
		postgres.NewExpenseCategoryRepo(pool),
		postgres.NewSavingsCategoryRepo(pool),
		postgres.NewUserRepo(pool),
		nil, nil,
	)

	engine := deliveryHTTP.SetupRouter(
		deliveryHTTP.RouterConfig{
			JWTSecret:   entTestSecret,
			Entitlement: entMw,
			Usage:       usage,
			Ledger:      ledgerService,
		},
		usecase.NewTransactionUsecase(txRepo, nil, nil, nil, nil),
		nil, nil, nil, nil, aiUC, nil,
		usecase.NewReportUsecase(txRepo),
		nil, nil, nil, nil, nil, nil,
	)
	srv := httptest.NewServer(engine)
	defer srv.Close()

	// Provision test user
	userID := uuid.New()
	userEmail := fmt.Sprintf("race-%s@fintrack.test", userID.String()[:8])
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, role)
		 VALUES ($1, $2, 'dummy', 'Race Test User', 'user')`,
		userID, userEmail); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	}()

	// Provision active subscription on pro plan (to enable ai_chat feature)
	now := time.Now().UTC()
	subID := uuid.New()
	periodStart := now.Add(-time.Hour)
	periodEnd := now.Add(30 * 24 * time.Hour)
	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, plan_id, status, current_period_start, current_period_end)
		 VALUES ($1, $2, $3, 'active', $4, $5)`,
		subID, userID, proPlanID, periodStart, periodEnd); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	// Seed exact token quota = 5 (0 used, 0 reserved => exactly 5 available)
	periodID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO token_periods (id, user_id, period_start, period_end, quota_tokens, used_tokens, reserved_tokens)
		 VALUES ($1, $2, $3, $4, 5, 0, 0)`,
		periodID, userID, periodStart, periodEnd); err != nil {
		t.Fatalf("insert token_periods: %v", err)
	}

	tokenStr, err := auth.IssueAccessToken(userID, "user", userEmail, []byte(entTestSecret), time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	const concurrency = 20
	var successCount, rejectedCount, otherCount int32
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(vu int) {
			defer wg.Done()
			<-startBarrier

			reqBody, _ := json.Marshal(map[string]string{
				"message": "Halo FinTrack, ringkasan pengeluaran saya",
			})
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/api/v1/ai/chat", bytes.NewReader(reqBody))
			if err != nil {
				atomic.AddInt32(&otherCount, 1)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tokenStr)
			req.Header.Set("X-Request-Id", fmt.Sprintf("vu-%d-%d", vu, time.Now().UnixNano()))

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				atomic.AddInt32(&otherCount, 1)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			switch resp.StatusCode {
			case http.StatusOK:
				atomic.AddInt32(&successCount, 1)
			case http.StatusTooManyRequests:
				atomic.AddInt32(&rejectedCount, 1)
				var errResp struct {
					Error string `json:"error"`
				}
				_ = json.Unmarshal(body, &errResp)
				if errResp.Error != "insufficient_quota" {
					t.Errorf("expected insufficient_quota, got body: %s", string(body))
				}
			default:
				t.Logf("unexpected status %d: %s", resp.StatusCode, string(body))
				atomic.AddInt32(&otherCount, 1)
			}
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	t.Logf("Results: success(200)=%d, rejected(429)=%d, other=%d", successCount, rejectedCount, otherCount)

	if otherCount != 0 {
		t.Fatalf("expected 0 other/error responses, got %d", otherCount)
	}
	if successCount != 5 {
		t.Fatalf("expected exactly 5 successes (200), got %d", successCount)
	}
	if rejectedCount != 15 {
		t.Fatalf("expected exactly 15 rejected (429), got %d", rejectedCount)
	}

	// Verify database invariant: available quota == 0, never negative
	var quotaTokens, usedTokens, reservedTokens int64
	err = pool.QueryRow(ctx,
		`SELECT quota_tokens, used_tokens, reserved_tokens FROM token_periods WHERE id = $1`, periodID).
		Scan(&quotaTokens, &usedTokens, &reservedTokens)
	if err != nil {
		t.Fatalf("query token_periods: %v", err)
	}

	available := quotaTokens - usedTokens - reservedTokens
	if available != 0 {
		t.Errorf("expected available 0, got %d (quota=%d, used=%d, reserved=%d)",
			available, quotaTokens, usedTokens, reservedTokens)
	}
	if usedTokens+reservedTokens != 5 {
		t.Errorf("expected total consumed 5, got used=%d reserved=%d", usedTokens, reservedTokens)
	}

	// Verify token_ledger entries: exactly 5 reserves and 5 commits
	var reserveCount, commitCount int
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM token_ledger WHERE user_id = $1 AND entry_type = 'reserve'`, userID).Scan(&reserveCount)
	if err != nil {
		t.Fatalf("query token_ledger reserve count: %v", err)
	}
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM token_ledger WHERE user_id = $1 AND entry_type = 'commit'`, userID).Scan(&commitCount)
	if err != nil {
		t.Fatalf("query token_ledger commit count: %v", err)
	}
	if reserveCount != 5 {
		t.Errorf("expected exactly 5 reserve rows, got %d", reserveCount)
	}
	if commitCount != 5 {
		t.Errorf("expected exactly 5 commit rows, got %d", commitCount)
	}
}
