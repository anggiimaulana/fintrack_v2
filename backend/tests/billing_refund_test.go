package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fintrack/backend/internal/billing"
	deliveryHTTP "github.com/fintrack/backend/internal/delivery/http"
	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/entitlement"
	"github.com/fintrack/backend/internal/infrastructure/auth"
	"github.com/fintrack/backend/internal/repository/postgres"
	"github.com/fintrack/backend/internal/usecase"
)

// noChargeGateway always fails to produce content: it returns an error before
// emitting anything, which the chat usecase turns into a NoCharge terminal event
// (no billable work happened).
type noChargeGateway struct{}

func (noChargeGateway) ChatStream(context.Context, *domain.AIChatRequest) (<-chan domain.AIChatStreamChunk, error) {
	return nil, domain.ErrAllEndpointsExhausted
}
func (noChargeGateway) ChatSync(context.Context, *domain.AIChatRequest) (string, domain.TokenUsage, error) {
	return "", domain.TokenUsage{}, domain.ErrAllEndpointsExhausted
}
func (noChargeGateway) CreateEmbedding(context.Context, *domain.AIEmbeddingRequest) ([]float32, domain.TokenUsage, error) {
	return nil, domain.TokenUsage{}, nil
}
func (noChargeGateway) TranscribeAudio(context.Context, *domain.AIAudioTranscriptionRequest) (string, domain.TokenUsage, error) {
	return "", domain.TokenUsage{}, nil
}
func (noChargeGateway) AnalyzeVision(context.Context, *domain.AIVisionRequest) (string, domain.TokenUsage, error) {
	return "", domain.TokenUsage{}, nil
}

// oneTokenEstimateSettings pins every reserve estimate to 1 token. The
// overhead keys return 0 and chars-per-token a very large value, because
// ForChat adds a configurable fixed overhead PLUS a prompt-length component.
type oneTokenEstimateSettings struct{}

func (oneTokenEstimateSettings) GetInt(context.Context, string) (int64, error) { return 1, nil }

func (oneTokenEstimateSettings) GetString(_ context.Context, key string) (string, error) {
	switch key {
	case "ai_estimate_overhead_tokens", "ai_estimate_output_headroom":
		return "0", nil
	case "ai_estimate_chars_per_token":
		return "1000000", nil
	}
	return "", nil
}

// TestBilling_UnservedChatTurnIsNotCharged proves the refund-on-failure contract:
// when the AI gateway cannot serve a turn (no content produced), the reserved
// quota is RELEASED, so the user's used_tokens does not increase and the balance
// returns to its starting value.
func TestBilling_UnservedChatTurnIsNotCharged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
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

	quotaResolver := billing.QuotaResolverFunc(func(ctx context.Context, userID uuid.UUID, at time.Time) (time.Time, time.Time, int64, int64, error) {
		res, err := entResolver.Resolve(ctx, userID)
		if err != nil {
			return time.Time{}, time.Time{}, 0, 0, err
		}
		quota := res.Limit("ai_tokens_per_cycle")
		if quota < 0 {
			quota = 1_000_000_000
		}
		ratio := res.QuotaRatioValue()
		sub, _ := billingRepo.GetByUserID(ctx, userID)
		if sub != nil && sub.CurrentPeriodEnd != nil {
			return sub.CurrentPeriodStart, *sub.CurrentPeriodEnd, quota, ratio, nil
		}
		start := at.Truncate(24 * time.Hour)
		return start, start.AddDate(0, 0, 30), quota, ratio, nil
	})
	ledgerService := billing.NewLedgerService(billingRepo, txManager, quotaResolver, nil)

	// A gateway that cannot serve the turn -> NoCharge path.
	aiUC := usecase.NewAIUsecase(
		postgres.NewChatSessionRepo(pool),
		postgres.NewChatMessageRepo(pool),
		txRepo,
		postgres.NewIncomeCategoryRepo(pool),
		postgres.NewExpenseCategoryRepo(pool),
		postgres.NewSavingsCategoryRepo(pool),
		postgres.NewUserRepo(pool),
		nil, nil,
	).WithGateway(noChargeGateway{})

	engine := deliveryHTTP.SetupRouter(
		deliveryHTTP.RouterConfig{
			JWTSecret:   entTestSecret,
			Entitlement: entMw,
			Usage:       usage,
			Ledger:      ledgerService,
			Estimates:   billing.NewEstimateStore(oneTokenEstimateSettings{}),
		},
		usecase.NewTransactionUsecase(txRepo, nil, nil, nil, nil),
		nil, nil, nil, nil, nil, nil, nil, aiUC, nil,
		usecase.NewReportUsecase(txRepo),
		nil, nil, nil, nil, nil, nil,
	)
	srv := httptest.NewServer(engine)
	defer srv.Close()

	userID := uuid.New()
	email := fmt.Sprintf("refund-%s@fintrack.test", userID.String()[:8])
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, role) VALUES ($1,$2,'x','Refund User','user')`,
		userID, email); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	defer pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)

	now := time.Now().UTC()
	periodStart := now.Add(-time.Hour)
	periodEnd := now.Add(30 * 24 * time.Hour)
	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, plan_id, status, current_period_start, current_period_end)
		 VALUES ($1,$2,$3,'active',$4,$5)`,
		uuid.New(), userID, proPlanID, periodStart, periodEnd); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
	periodID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO token_periods (id, user_id, period_start, period_end, quota_tokens, used_tokens, reserved_tokens)
		 VALUES ($1,$2,$3,$4,1000,100,0)`,
		periodID, userID, periodStart, periodEnd); err != nil {
		t.Fatalf("insert token_periods: %v", err)
	}

	tokenStr, err := auth.IssueAccessToken(userID, "user", email, []byte(entTestSecret), time.Hour)
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	reqBody, _ := json.Marshal(map[string]string{"message": "berapa total pengeluaran saya bulan ini"})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/ai/chat", strings.NewReader(string(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	req.Header.Set("X-Request-Id", "refund-"+userID.String())
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("chat request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (SSE stream), got %d", resp.StatusCode)
	}
	// Drain the SSE stream so the handler's deferred settle (release) completes
	// before we assert the ledger state.
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("read stream: %v", err)
	}

	// After the unserved turn: no reservation left, used_tokens unchanged.
	var used, reserved int64
	if err := pool.QueryRow(ctx,
		`SELECT used_tokens, reserved_tokens FROM token_periods WHERE id=$1`, periodID).Scan(&used, &reserved); err != nil {
		t.Fatalf("query period: %v", err)
	}
	if reserved != 0 {
		t.Errorf("expected reserved_tokens=0 after release, got %d", reserved)
	}
	if used != 100 {
		t.Errorf("expected used_tokens unchanged at 100, got %d (an unserved turn must not charge)", used)
	}

	// Exactly one reserve and one release; zero commits.
	var reserves, releases, commits int
	pool.QueryRow(ctx, `SELECT count(*) FROM token_ledger WHERE user_id=$1 AND entry_type='reserve'`, userID).Scan(&reserves)
	pool.QueryRow(ctx, `SELECT count(*) FROM token_ledger WHERE user_id=$1 AND entry_type='release'`, userID).Scan(&releases)
	pool.QueryRow(ctx, `SELECT count(*) FROM token_ledger WHERE user_id=$1 AND entry_type='commit'`, userID).Scan(&commits)
	if reserves != 1 || releases != 1 || commits != 0 {
		t.Errorf("ledger entries: reserve=%d release=%d commit=%d, want 1/1/0", reserves, releases, commits)
	}
}
