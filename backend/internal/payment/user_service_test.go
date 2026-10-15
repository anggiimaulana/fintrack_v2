package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/pkg/pagination"
)

// ─── stubs ────────────────────────────────────────────────────

// fakeSettings is a minimal SettingsRepository.
type fakeSettings struct{ vals map[string]string }

func (f *fakeSettings) GetString(_ context.Context, key string) (string, error) {
	return f.vals[key], nil
}
func (f *fakeSettings) GetInt(context.Context, string) (int64, error) { return 0, nil }
func (f *fakeSettings) Set(context.Context, string, any) error        { return nil }

// fakePlanRepo is a billing.PlanRepository backed by a slice.
type fakePlanRepo struct{ plans []*billing.Plan }

func (f *fakePlanRepo) GetPlanByCode(_ context.Context, code string) (*billing.Plan, error) {
	for _, p := range f.plans {
		if p.Code == code {
			return p, nil
		}
	}
	return nil, billing.ErrPlanNotFound
}
func (f *fakePlanRepo) GetPlanByID(_ context.Context, id uuid.UUID) (*billing.Plan, error) {
	for _, p := range f.plans {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, billing.ErrPlanNotFound
}
func (f *fakePlanRepo) ListActivePlans(context.Context) ([]*billing.Plan, error) {
	var out []*billing.Plan
	for _, p := range f.plans {
		if p.IsActive {
			out = append(out, p)
		}
	}
	return out, nil
}
func (f *fakePlanRepo) GetEntitlements(context.Context, uuid.UUID) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (f *fakePlanRepo) GetLimits(context.Context, uuid.UUID) (map[string]int64, error) {
	return map[string]int64{}, nil
}

// fakeEntSource returns canned features/limits.
type fakeEntSource struct {
	features map[string]bool
	limits   map[string]int64
}

func (f *fakeEntSource) GetEntitlements(context.Context, uuid.UUID) (map[string]bool, error) {
	return f.features, nil
}
func (f *fakeEntSource) GetLimits(context.Context, uuid.UUID) (map[string]int64, error) {
	return f.limits, nil
}

// fakeSubRepo is a billing.SubscriptionRepository backed by one row.
type fakeSubRepo struct {
	sub     *billing.Subscription
	events  []*billing.SubscriptionEvent
	updated int
}

func (f *fakeSubRepo) GetByUserID(context.Context, uuid.UUID) (*billing.Subscription, error) {
	if f.sub == nil {
		return nil, billing.ErrSubscriptionNotFound
	}
	return f.sub, nil
}
func (f *fakeSubRepo) Create(context.Context, *billing.Subscription) error { return nil }
func (f *fakeSubRepo) Update(_ context.Context, s *billing.Subscription) error {
	f.sub = s
	f.updated++
	return nil
}
func (f *fakeSubRepo) RecordEvent(_ context.Context, ev *billing.SubscriptionEvent) error {
	f.events = append(f.events, ev)
	return nil
}
func (f *fakeSubRepo) ListEvents(context.Context, uuid.UUID, int) ([]*billing.SubscriptionEvent, error) {
	return f.events, nil
}
func (f *fakeSubRepo) ListEventsPaged(_ context.Context, _ uuid.UUID, _ *pagination.Query) ([]*billing.SubscriptionEvent, int, error) {
	return f.events, len(f.events), nil
}

// ─── tests ────────────────────────────────────────────────────

func TestUserService_ListAndGetScopedByOwner(t *testing.T) {
	repo := newMemRepo()
	ctx := context.Background()
	userID := uuid.New()
	other := uuid.New()

	invID := uuid.New()
	note := "pro"
	_ = repo.CreateInvoice(ctx, &Invoice{ID: invID, InvoiceNumber: "INV/1", UserID: userID, Status: InvoiceIssued, Notes: &note})
	pay := &Payment{ID: uuid.New(), InvoiceID: invID, UserID: userID, Gateway: "manual", AmountIDR: 39000, Status: PaymentPending, IdempotencyKey: "k1"}
	_ = repo.CreatePayment(ctx, pay)

	svc := NewUserService(repo, &fakeSettings{}, nil, nil)

	list, _, err := svc.ListPayments(ctx, userID, nil)
	if err != nil {
		t.Fatalf("ListPayments: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(list))
	}

	// Owner can read it.
	got, err := svc.GetPayment(ctx, userID, pay.ID)
	if err != nil || got == nil {
		t.Fatalf("GetPayment owner: %v %v", got, err)
	}
	// Another user must not.
	other2, err := svc.GetPayment(ctx, other, pay.ID)
	if err != nil {
		t.Fatalf("GetPayment other err: %v", err)
	}
	if other2 != nil {
		t.Fatal("expected nil for non-owner")
	}
}

func TestUserService_UploadProofRequiresManualConfig(t *testing.T) {
	svc := NewUserService(newMemRepo(), &fakeSettings{}, nil, nil)
	_, err := svc.UploadProof(context.Background(), uuid.New(), uuid.New(), []byte("x"), "p.png")
	if !errors.Is(err, ErrGatewayNotConfigured) {
		t.Fatalf("expected ErrGatewayNotConfigured, got %v", err)
	}
}

func TestUserService_UploadProofDelegatesAndMovesToReview(t *testing.T) {
	repo := newMemRepo()
	ctx := context.Background()
	userID := uuid.New()
	invID := uuid.New()
	note := "pro"
	_ = repo.CreateInvoice(ctx, &Invoice{ID: invID, UserID: userID, Status: InvoiceIssued, Notes: &note})
	pay := &Payment{ID: uuid.New(), InvoiceID: invID, UserID: userID, Gateway: "manual", AmountIDR: 39000, Status: PaymentPending, IdempotencyKey: "k2"}
	_ = repo.CreatePayment(ctx, pay)

	manual := NewManualService(repo, nil, noopTx{}, &memProofStore{})
	svc := NewUserService(repo, &fakeSettings{}, nil, manual)

	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	updated, err := svc.UploadProof(ctx, userID, pay.ID, png, "p.png")
	if err != nil {
		t.Fatalf("UploadProof: %v", err)
	}
	if updated.Status != PaymentPendingReview {
		t.Fatalf("status = %s, want pending_review", updated.Status)
	}
}

func TestUserService_PaymentInstructionsFromSnapshot(t *testing.T) {
	repo := newMemRepo()
	ctx := context.Background()
	userID := uuid.New()
	invID := uuid.New()
	_ = repo.CreateInvoice(ctx, &Invoice{ID: invID, UserID: userID, Status: InvoiceIssued})
	pay := &Payment{
		ID: uuid.New(), InvoiceID: invID, UserID: userID, Gateway: "manual", AmountIDR: 1000,
		Status: PaymentPending, IdempotencyKey: "k3",
		PayloadSnapshot: map[string]any{"instructions": map[string]any{"bank_name": "BCA", "account_number": "123"}},
	}
	_ = repo.CreatePayment(ctx, pay)

	svc := NewUserService(repo, &fakeSettings{}, nil, nil)
	instr, err := svc.PaymentInstructions(ctx, userID, pay.ID)
	if err != nil {
		t.Fatalf("PaymentInstructions: %v", err)
	}
	if instr["bank_name"] != "BCA" {
		t.Fatalf("bank_name = %v, want BCA", instr["bank_name"])
	}
}

func TestCatalog_ListPublicPlansDynamic(t *testing.T) {
	free := &billing.Plan{ID: uuid.New(), Code: "free", Name: "Free", IsActive: true, IsPublic: true, SortOrder: 1}
	pro := &billing.Plan{ID: uuid.New(), Code: "pro", Name: "Pro", PriceMonthlyIDR: 39000, IsActive: true, IsPublic: true, SortOrder: 2}
	hidden := &billing.Plan{ID: uuid.New(), Code: "legacy", Name: "Legacy", IsActive: true, IsPublic: false, SortOrder: 3}
	repo := &fakePlanRepo{plans: []*billing.Plan{free, pro, hidden}}
	ent := &fakeEntSource{features: map[string]bool{"ai_chat": true}, limits: map[string]int64{"max_transactions": 100}}

	svc := NewCatalogService(repo, ent, []LimitMeta{
		{Key: "max_transactions", Label: "Transaksi", Unit: "transaksi", Enforced: true},
		{Key: "max_categories", Label: "Kategori", Unit: "kategori", Enforced: false},
	}, nil)
	plans, err := svc.ListPublicPlans(context.Background())
	if err != nil {
		t.Fatalf("ListPublicPlans: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 public plans, got %d", len(plans))
	}
	if plans[1].PriceMonthlyIDR != 39000 {
		t.Fatalf("pro price = %d, want 39000", plans[1].PriceMonthlyIDR)
	}
	if !plans[0].Features["ai_chat"] {
		t.Fatal("expected features attached")
	}
	if plans[0].Limits["max_transactions"] != 100 {
		t.Fatal("expected limits attached")
	}
	// F4: the catalog carries the enforced flag so the UI can hide limits the
	// backend does not enforce.
	if !plans[0].LimitMeta["max_transactions"].Enforced {
		t.Fatal("max_transactions should be marked enforced")
	}
}

// TestCatalog_UnenforcedLimitsCarryFlag verifies the public catalog exposes the
// `enforced` flag so the pricing UI never advertises a limit that is not applied
// (F4: max_categories/max_wallets).
func TestCatalog_UnenforcedLimitsCarryFlag(t *testing.T) {
	pro := &billing.Plan{ID: uuid.New(), Code: "pro", Name: "Pro", IsActive: true, IsPublic: true, SortOrder: 2}
	repo := &fakePlanRepo{plans: []*billing.Plan{pro}}
	ent := &fakeEntSource{
		features: map[string]bool{},
		limits:   map[string]int64{"max_transactions": 5000, "max_categories": 30, "max_wallets": 5},
	}
	svc := NewCatalogService(repo, ent, []LimitMeta{
		{Key: "max_transactions", Label: "Transaksi", Unit: "transaksi", Enforced: true},
		{Key: "max_categories", Label: "Kategori", Unit: "kategori", Enforced: false},
		{Key: "max_wallets", Label: "Dompet", Unit: "dompet", Enforced: false},
	}, nil)

	plans, err := svc.ListPublicPlans(context.Background())
	if err != nil {
		t.Fatalf("ListPublicPlans: %v", err)
	}
	meta := plans[0].LimitMeta
	if meta["max_categories"].Enforced || meta["max_wallets"].Enforced {
		t.Fatalf("categories/wallets must be marked unenforced: %+v", meta)
	}
	if meta["max_categories"].Label != "Kategori" || meta["max_wallets"].Label != "Dompet" {
		t.Fatalf("labels not carried: %+v", meta)
	}
	if !meta["max_transactions"].Enforced {
		t.Fatal("max_transactions should stay enforced")
	}
}

// TestCatalog_ExposesTaxInclusiveTotal verifies the catalog reports the same
// tax-inclusive total the checkout will charge, so the pricing card can show the
// real price before an invoice is issued.
func TestCatalog_ExposesTaxInclusiveTotal(t *testing.T) {
	pro := &billing.Plan{ID: uuid.New(), Code: "pro", Name: "Pro", PriceMonthlyIDR: 39000, IsActive: true, IsPublic: true, SortOrder: 2}
	repo := &fakePlanRepo{plans: []*billing.Plan{pro}}
	ent := &fakeEntSource{features: map[string]bool{}, limits: map[string]int64{}}

	settings := newMemSettings()
	settings.m["tax_enabled"] = "true"
	settings.m["tax_rate_percent"] = int64(11)
	svc := NewCatalogService(repo, ent, nil, NewTaxCalculator(settings))

	plans, err := svc.ListPublicPlans(context.Background())
	if err != nil {
		t.Fatalf("ListPublicPlans: %v", err)
	}
	if plans[0].TaxRatePercent != 11 {
		t.Fatalf("tax rate = %d, want 11", plans[0].TaxRatePercent)
	}
	if plans[0].TaxIDR != 4290 {
		t.Fatalf("tax = %d, want 4290", plans[0].TaxIDR)
	}
	if plans[0].TotalIDR != 43290 {
		t.Fatalf("total = %d, want 43290", plans[0].TotalIDR)
	}
}

// TestCatalog_NoTaxCalculatorFallsBackToSubtotal verifies a nil tax calculator
// yields total == subtotal rather than a zero total.
func TestCatalog_NoTaxCalculatorFallsBackToSubtotal(t *testing.T) {
	pro := &billing.Plan{ID: uuid.New(), Code: "pro", Name: "Pro", PriceMonthlyIDR: 39000, IsActive: true, IsPublic: true, SortOrder: 2}
	repo := &fakePlanRepo{plans: []*billing.Plan{pro}}
	ent := &fakeEntSource{features: map[string]bool{}, limits: map[string]int64{}}
	svc := NewCatalogService(repo, ent, nil, nil)

	plans, err := svc.ListPublicPlans(context.Background())
	if err != nil {
		t.Fatalf("ListPublicPlans: %v", err)
	}
	if plans[0].TotalIDR != 39000 || plans[0].TaxIDR != 0 {
		t.Fatalf("expected total 39000 / tax 0, got %d / %d", plans[0].TotalIDR, plans[0].TaxIDR)
	}
}

// TestCatalog_NoLimitMetaWhenNotConfigured verifies the response degrades
// gracefully when no catalog is supplied (older deployments).
func TestCatalog_NoLimitMetaWhenNotConfigured(t *testing.T) {
	pro := &billing.Plan{ID: uuid.New(), Code: "pro", Name: "Pro", IsActive: true, IsPublic: true, SortOrder: 2}
	repo := &fakePlanRepo{plans: []*billing.Plan{pro}}
	ent := &fakeEntSource{features: map[string]bool{}, limits: map[string]int64{"max_transactions": 100}}
	svc := NewCatalogService(repo, ent, nil, nil)

	plans, err := svc.ListPublicPlans(context.Background())
	if err != nil {
		t.Fatalf("ListPublicPlans: %v", err)
	}
	if len(plans[0].LimitMeta) != 0 {
		t.Fatalf("expected empty limit_meta, got %+v", plans[0].LimitMeta)
	}
	if plans[0].Limits["max_transactions"] != 100 {
		t.Fatal("limits must still be returned")
	}
}

func TestSubscriptionService_ScheduleDowngradeSoftLimit(t *testing.T) {
	proID := uuid.New()
	freeID := uuid.New()
	repo := &fakePlanRepo{plans: []*billing.Plan{
		{ID: freeID, Code: "free", Name: "Free", IsActive: true, IsPublic: true},
		{ID: proID, Code: "pro", Name: "Pro", PriceMonthlyIDR: 39000, IsActive: true, IsPublic: true},
	}}
	subs := &fakeSubRepo{sub: &billing.Subscription{ID: uuid.New(), UserID: uuid.New(), PlanID: proID, Status: billing.StatusActive}}

	var invalidated int
	svc := NewSubscriptionService(subs, repo, nil, func(context.Context, uuid.UUID) { invalidated++ })

	if err := svc.ScheduleDowngrade(context.Background(), subs.sub.UserID, "free"); err != nil {
		t.Fatalf("ScheduleDowngrade: %v", err)
	}
	// Soft-limit: plan NOT changed now, only scheduled.
	if subs.sub.PlanID != proID {
		t.Fatal("plan must not change immediately (soft-limit)")
	}
	if !subs.sub.CancelAtPeriodEnd {
		t.Fatal("cancel_at_period_end must be set")
	}
	if subs.sub.Metadata["scheduled_plan_code"] != "free" {
		t.Fatalf("scheduled_plan_code = %v, want free", subs.sub.Metadata["scheduled_plan_code"])
	}
	if invalidated != 1 {
		t.Fatalf("expected 1 invalidation, got %d", invalidated)
	}
	if len(subs.events) != 1 || subs.events[0].EventType != "downgrade_scheduled" {
		t.Fatalf("expected downgrade_scheduled event, got %+v", subs.events)
	}
}

func TestSubscriptionService_ScheduleDowngradeSamePlanRejected(t *testing.T) {
	proID := uuid.New()
	repo := &fakePlanRepo{plans: []*billing.Plan{{ID: proID, Code: "pro", IsActive: true}}}
	subs := &fakeSubRepo{sub: &billing.Subscription{ID: uuid.New(), UserID: uuid.New(), PlanID: proID, Status: billing.StatusActive}}
	svc := NewSubscriptionService(subs, repo, nil, nil)
	if err := svc.ScheduleDowngrade(context.Background(), subs.sub.UserID, "pro"); err == nil {
		t.Fatal("downgrade to the current plan must be rejected")
	}
}

func TestSubscriptionService_CancelDowngrade(t *testing.T) {
	proID := uuid.New()
	repo := &fakePlanRepo{plans: []*billing.Plan{{ID: proID, Code: "pro", IsActive: true}}}
	subs := &fakeSubRepo{sub: &billing.Subscription{
		ID: uuid.New(), UserID: uuid.New(), PlanID: proID, Status: billing.StatusActive,
		CancelAtPeriodEnd: true,
		Metadata:          map[string]any{"scheduled_plan_code": "free"},
	}}
	svc := NewSubscriptionService(subs, repo, nil, nil)
	if err := svc.CancelDowngrade(context.Background(), subs.sub.UserID); err != nil {
		t.Fatalf("CancelDowngrade: %v", err)
	}
	if subs.sub.CancelAtPeriodEnd {
		t.Fatal("cancel_at_period_end must be cleared")
	}
	if _, ok := subs.sub.Metadata["scheduled_plan_code"]; ok {
		t.Fatal("scheduled_plan_code must be removed")
	}
}

func TestManualGateway_InstructionsOnlyIncludeConfiguredFields(t *testing.T) {
	// Fully configured.
	gw := NewManualGateway("BCA", "123456", "PT FinTrack")
	res, err := gw.CreateTransaction(context.Background(), CreateTxRequest{OrderID: "INV/1", AmountIDR: 43290})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	ins, ok := res.Raw["instructions"].(map[string]any)
	if !ok {
		t.Fatal("instructions missing")
	}
	if ins["bank_name"] != "BCA" || ins["account_number"] != "123456" || ins["amount_idr"] != int64(43290) {
		t.Fatalf("unexpected instructions: %+v", ins)
	}

	// Unconfigured: must NOT advertise empty bank fields.
	gw2 := NewManualGateway("", "", "")
	res2, _ := gw2.CreateTransaction(context.Background(), CreateTxRequest{OrderID: "INV/2", AmountIDR: 1000})
	ins2 := res2.Raw["instructions"].(map[string]any)
	if _, exists := ins2["bank_name"]; exists {
		t.Fatal("empty bank_name must be omitted")
	}
	if ins2["amount_idr"] != int64(1000) {
		t.Fatalf("amount must always be present, got %+v", ins2)
	}
}
