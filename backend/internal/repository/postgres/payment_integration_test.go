package postgres

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/payment"
)

// countingActivator counts activations across concurrent webhook processing.
type countingActivator struct {
	mu    sync.Mutex
	count int
}

func (c *countingActivator) ActivateForUser(context.Context, uuid.UUID, string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	return nil
}

// fixedGateway returns a scripted settled webhook + status.
type fixedGateway struct {
	name string
	ref  string
}

func (f *fixedGateway) Name() string { return f.name }
func (f *fixedGateway) CreateTransaction(context.Context, payment.CreateTxRequest) (*payment.CreateTxResponse, error) {
	return &payment.CreateTxResponse{Status: payment.PaymentPending}, nil
}
func (f *fixedGateway) VerifyWebhook(context.Context, []byte, map[string]string) (*payment.WebhookResult, error) {
	return &payment.WebhookResult{OrderID: f.ref, GatewayRef: f.ref, Status: payment.PaymentSettled, AmountIDR: 39000, PaymentMethod: "qris"}, nil
}
func (f *fixedGateway) CheckStatus(context.Context, string) (*payment.StatusResult, error) {
	return &payment.StatusResult{OrderID: f.ref, Status: payment.PaymentSettled, AmountIDR: 39000}, nil
}
func (f *fixedGateway) CancelTransaction(context.Context, string) error { return nil }

// TestPayment_ConcurrentWebhookIdempotent is the core anti-double-credit proof
// against a live database: many concurrent webhooks for the same order must
// settle the payment and activate the subscription exactly once.
func TestPayment_ConcurrentWebhookIdempotent(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPaymentRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, full_name, role) VALUES ($1,$2,$3,'user')`,
		userID, userID.String()+"@itest.local", "PayITest"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	// Create invoice + payment with a unique gateway_ref.
	ref := "INV-ITEST-" + userID.String()[:8]
	note := "pro"
	inv := &payment.Invoice{
		InvoiceNumber: "INV/ITEST/" + userID.String()[:8],
		UserID:        userID, SubtotalIDR: 39000, TotalIDR: 39000, Status: payment.InvoiceIssued, PlanCode: &note,
	}
	if err := repo.CreateInvoice(ctx, inv); err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	pay := &payment.Payment{
		InvoiceID: inv.ID, UserID: userID, Gateway: "midtrans", GatewayRef: &ref,
		AmountIDR: 39000, Status: payment.PaymentPending, IdempotencyKey: "itest-" + userID.String(),
	}
	if err := repo.CreatePayment(ctx, pay); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	reg := payment.NewRegistry()
	reg.Register(&fixedGateway{name: "midtrans", ref: ref})
	act := &countingActivator{}
	proc := payment.NewWebhookProcessor(repo, reg, act, tx)

	// Fire 20 concurrent webhooks for the same order.
	const n = 20
	var processed, idempotent int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res, err := proc.Process(ctx, "midtrans", []byte("{}"), nil)
			if err != nil {
				t.Errorf("process: %v", err)
				return
			}
			if res.AlreadyProcessed {
				atomic.AddInt32(&idempotent, 1)
			} else {
				atomic.AddInt32(&processed, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if processed != 1 {
		t.Fatalf("exactly 1 webhook should apply the transition, got %d", processed)
	}
	if int(processed+idempotent) != n {
		t.Fatalf("accounting mismatch: processed=%d idempotent=%d", processed, idempotent)
	}
	if act.count != 1 {
		t.Fatalf("subscription must be activated exactly once, got %d", act.count)
	}

	// Verify DB state.
	stored, err := repo.GetPaymentByID(ctx, pay.ID)
	if err != nil || stored == nil {
		t.Fatalf("get payment: %v", err)
	}
	if stored.Status != payment.PaymentSettled {
		t.Fatalf("payment should be settled, got %s", stored.Status)
	}
	invAfter, err := repo.GetInvoiceByID(ctx, inv.ID)
	if err != nil || invAfter == nil {
		t.Fatalf("get invoice: %v", err)
	}
	if invAfter.Status != payment.InvoicePaid {
		t.Fatalf("invoice should be paid, got %s", invAfter.Status)
	}
}

// TestPayment_InvoiceNumberSequence verifies atomic, gapless numbering.
func TestPayment_InvoiceNumberSequence(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPaymentRepo(pool)
	ctx := context.Background()

	day := "20990101" // far-future day to avoid clashing with real data
	seen := map[string]bool{}
	for range 20 {
		num, err := repo.NextInvoiceNumber(ctx, day)
		if err != nil {
			t.Fatalf("next number: %v", err)
		}
		if seen[num] {
			t.Fatalf("duplicate invoice number %s", num)
		}
		seen[num] = true
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM invoice_counters WHERE day = $1::date`, day)
	})
	if len(seen) != 20 {
		t.Fatalf("expected 20 unique numbers, got %d", len(seen))
	}
}

// TestListRenewalCandidates covers the renewal worker's query: a due, active,
// non-cancelled paid subscription is a candidate; a cancelled one is not.
func TestListRenewalCandidates(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPaymentRepo(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, password_hash, full_name, role) VALUES ($1,$2,'x','Renew','user')`,
		userID, userID.String()+"@test.local"); err != nil {
		t.Fatalf("user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	var proID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM plans WHERE code='pro'`).Scan(&proID); err != nil {
		t.Fatalf("pro plan: %v", err)
	}
	subID := uuid.New()
	pastEnd := time.Now().UTC().Add(-time.Hour)
	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, plan_id, status, current_period_start, current_period_end)
		 VALUES ($1,$2,$3,'active',$4,$5)`, subID, userID, proID, pastEnd.Add(-30*24*time.Hour), pastEnd); err != nil {
		t.Fatalf("sub: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM subscriptions WHERE id=$1`, subID) })

	cands, err := repo.ListRenewalCandidates(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, c := range cands {
		if c.UserID == userID {
			found = true
			if c.PriceIDR <= 0 {
				t.Fatalf("candidate must carry a positive price: %d", c.PriceIDR)
			}
		}
	}
	if !found {
		t.Fatal("due subscription must be a renewal candidate")
	}

	// Cancelled -> excluded.
	if _, err := pool.Exec(ctx, `UPDATE subscriptions SET cancel_at_period_end=true WHERE id=$1`, subID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	cands2, _ := repo.ListRenewalCandidates(ctx, time.Now().UTC())
	for _, c := range cands2 {
		if c.UserID == userID {
			t.Fatal("cancelled subscription must NOT be a renewal candidate")
		}
	}
}

// TestRenewalMarker_GoMatchesSQL is the regression guard for a double-billing
// bug: the renewal worker's Go marker uses Unix() (truncation), while the
// candidate query's SQL used `::bigint` (rounding). A period end with a
// sub-second remainder therefore produced two DIFFERENT markers, the exclusion
// check never matched, and the same cycle was invoiced again on every sweep.
func TestRenewalMarker_GoMatchesSQL(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPaymentRepo(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,email,password_hash,full_name,role) VALUES ($1,$2,'x','M','user')`,
		userID, userID.String()+"@test.local"); err != nil {
		t.Fatalf("user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })
	var proID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM plans WHERE code='pro'`).Scan(&proID); err != nil {
		t.Fatalf("pro plan: %v", err)
	}

	subID := uuid.New()
	// A sub-second remainder is exactly what exposed the truncate/round mismatch.
	pastEnd := time.Now().UTC().Add(-90 * time.Minute).Truncate(time.Millisecond)
	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriptions (id,user_id,plan_id,status,current_period_start,current_period_end)
		 VALUES ($1,$2,$3,'active',$4,$5)`, subID, userID, proID, pastEnd.Add(-30*24*time.Hour), pastEnd); err != nil {
		t.Fatalf("sub: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM subscriptions WHERE id=$1`, subID) })

	cands, err := repo.ListRenewalCandidates(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var end time.Time
	found := false
	for _, c := range cands {
		if c.SubscriptionID == subID {
			end, found = c.PeriodEnd, true
		}
	}
	if !found {
		t.Fatal("candidate not found")
	}

	var sqlMarker string
	if err := pool.QueryRow(ctx,
		`SELECT 'renewal:' || $1::uuid::text || ':' || floor(extract(epoch FROM $2::timestamptz))::bigint::text`,
		subID, end).Scan(&sqlMarker); err != nil {
		t.Fatalf("sql marker: %v", err)
	}
	goMarker := payment.RenewalMarker(subID, end)
	if goMarker != sqlMarker {
		t.Fatalf("marker mismatch (double-billing risk): go=%q sql=%q", goMarker, sqlMarker)
	}
}

// TestScheduledDowngrade_SyncsQuota is the regression guard for a quota leak: a
// scheduled downgrade changed the plan but left the running token period with
// the OLD (higher) quota, so the downgraded user kept the paid quota until the
// cycle lapsed. The manager must sync the running period to the new plan.
func TestScheduledDowngrade_SyncsQuota(t *testing.T) {
	pool := newTestPool(t)
	repo := NewBillingRepo(pool)
	tx := NewTxManager(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,email,password_hash,full_name,role) VALUES ($1,$2,'x','DG','user')`,
		userID, userID.String()+"@test.local"); err != nil {
		t.Fatalf("user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	proPlan, _ := repo.GetPlanByCode(ctx, "pro")
	freePlan, _ := repo.GetPlanByCode(ctx, "free")
	freeLimits, _ := repo.GetLimits(ctx, freePlan.ID)

	// Running Pro period.
	start := time.Now().UTC().Add(-time.Hour)
	p, err := repo.GetOrCreatePeriod(ctx, userID, start.Unix(), start.Add(30*24*time.Hour).Unix(), 150000, 1)
	if err != nil {
		t.Fatalf("period: %v", err)
	}

	// Subscription on Pro with a scheduled downgrade to Free, due now.
	pastEnd := time.Now().UTC().Add(-time.Minute)
	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriptions (user_id, plan_id, status, current_period_start, current_period_end,
		    cancel_at_period_end, metadata)
		 VALUES ($1,$2,'active',$3,$4,true,$5)`,
		userID, proPlan.ID, pastEnd.Add(-30*24*time.Hour), pastEnd,
		[]byte(`{"scheduled_plan_code":"free"}`)); err != nil {
		t.Fatalf("sub: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM subscriptions WHERE user_id=$1`, userID) })

	// Apply the downgrade through the manager (the path the worker uses).
	mgr := billing.NewSubscriptionManager(repo, repo, nil, tx).
		WithQuotaSyncer(billing.NewQuotaSyncer(repo, repo))
	if err := mgr.ApplyScheduledDowngrade(ctx, userID); err != nil {
		t.Fatalf("apply: %v", err)
	}

	after, err := repo.GetPeriodByIDForUpdate(ctx, p.ID)
	if err != nil || after == nil {
		t.Fatalf("period lookup: %v", err)
	}
	if after.QuotaTokens != freeLimits["ai_tokens_per_cycle"] {
		t.Fatalf("downgrade must sync quota: got %d want %d", after.QuotaTokens, freeLimits["ai_tokens_per_cycle"])
	}
}

// TestCardToken_CRUD tests card token persistence, default toggling, and deletion.
func TestCardToken_CRUD(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPaymentRepo(pool)
	ctx := context.Background()

	userID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,email,password_hash,full_name,role) VALUES ($1,$2,'x','CardUser','user')`,
		userID, userID.String()+"@test.local"); err != nil {
		t.Fatalf("user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })

	// 1) First card: should be auto-set as default.
	card1 := &payment.CardToken{
		UserID:     userID,
		Gateway:    "midtrans",
		TokenID:    "tok_card_1",
		MaskedCard: "481111-1111",
		CardType:   "credit",
		Bank:       "bca",
	}
	if err := repo.SaveCardToken(ctx, card1); err != nil {
		t.Fatalf("save card 1: %v", err)
	}

	def, err := repo.GetDefaultCardToken(ctx, userID)
	if err != nil || def == nil {
		t.Fatalf("expected default card: %v", err)
	}
	if def.MaskedCard != "481111-1111" || !def.IsDefault {
		t.Fatalf("card 1 should be default: %+v", def)
	}

	// 2) Second card: saved with is_default = true -> should become the new default.
	card2 := &payment.CardToken{
		UserID:     userID,
		Gateway:    "midtrans",
		TokenID:    "tok_card_2",
		MaskedCard: "520000-2222",
		CardType:   "credit",
		Bank:       "mandiri",
		IsDefault:  true,
	}
	if err := repo.SaveCardToken(ctx, card2); err != nil {
		t.Fatalf("save card 2: %v", err)
	}

	def2, err := repo.GetDefaultCardToken(ctx, userID)
	if err != nil || def2 == nil {
		t.Fatalf("expected default card: %v", err)
	}
	if def2.MaskedCard != "520000-2222" {
		t.Fatalf("card 2 should now be default, got: %s", def2.MaskedCard)
	}

	// List should return 2 cards
	list, err := repo.ListCardTokensByUser(ctx, userID)
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 cards in list, got %d: %v", len(list), err)
	}

	// 3) Delete card 2: card 1 should be promoted to default
	if err := repo.DeleteCardToken(ctx, userID, card2.ID); err != nil {
		t.Fatalf("delete card 2: %v", err)
	}
	defAfterDel, err := repo.GetDefaultCardToken(ctx, userID)
	if err != nil || defAfterDel == nil {
		t.Fatalf("expected card 1 to be promoted to default: %v", err)
	}
	if defAfterDel.MaskedCard != "481111-1111" {
		t.Fatalf("expected card 1 as default after del, got: %s", defAfterDel.MaskedCard)
	}
}
