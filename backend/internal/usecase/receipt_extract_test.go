package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/domain"
)

// mustParseDate parses a YYYY-MM-DD date for tests.
func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad test date %q: %v", s, err)
	}
	return d
}

// ─── Fake category repositories ────────────────────────────────
//
// The AI usecase only ever calls List for receipt categorization, but the
// interfaces require the full CRUD surface, so the unused methods are stubs.

type fakeIncomeCatRepo struct{ names []string }

func (f *fakeIncomeCatRepo) Create(context.Context, *domain.IncomeCategory) error { return nil }
func (f *fakeIncomeCatRepo) GetByID(context.Context, uuid.UUID) (*domain.IncomeCategory, error) {
	return nil, nil
}
func (f *fakeIncomeCatRepo) List(context.Context, bool) ([]domain.IncomeCategory, error) {
	out := make([]domain.IncomeCategory, 0, len(f.names))
	for _, n := range f.names {
		out = append(out, domain.IncomeCategory{Name: n, IsActive: true})
	}
	return out, nil
}
func (f *fakeIncomeCatRepo) Update(context.Context, *domain.IncomeCategory) error { return nil }
func (f *fakeIncomeCatRepo) SoftDelete(context.Context, uuid.UUID) error          { return nil }

type fakeExpenseCatRepo struct{ names []string }

func (f *fakeExpenseCatRepo) Create(context.Context, *domain.ExpenseCategory) error { return nil }
func (f *fakeExpenseCatRepo) GetByID(context.Context, uuid.UUID) (*domain.ExpenseCategory, error) {
	return nil, nil
}
func (f *fakeExpenseCatRepo) List(context.Context, bool) ([]domain.ExpenseCategory, error) {
	out := make([]domain.ExpenseCategory, 0, len(f.names))
	for _, n := range f.names {
		out = append(out, domain.ExpenseCategory{Name: n, IsActive: true})
	}
	return out, nil
}
func (f *fakeExpenseCatRepo) Update(context.Context, *domain.ExpenseCategory) error { return nil }
func (f *fakeExpenseCatRepo) SoftDelete(context.Context, uuid.UUID) error           { return nil }

type fakeSavingsCatRepo struct{ names []string }

func (f *fakeSavingsCatRepo) Create(context.Context, *domain.SavingsCategory) error { return nil }
func (f *fakeSavingsCatRepo) GetByID(context.Context, uuid.UUID) (*domain.SavingsCategory, error) {
	return nil, nil
}
func (f *fakeSavingsCatRepo) List(context.Context, bool) ([]domain.SavingsCategory, error) {
	out := make([]domain.SavingsCategory, 0, len(f.names))
	for _, n := range f.names {
		out = append(out, domain.SavingsCategory{Name: n, IsActive: true})
	}
	return out, nil
}
func (f *fakeSavingsCatRepo) Update(context.Context, *domain.SavingsCategory) error { return nil }
func (f *fakeSavingsCatRepo) SoftDelete(context.Context, uuid.UUID) error           { return nil }

func seededCats() receiptCategories {
	return receiptCategories{
		Expense: []string{"Makanan & Minuman", "Transportasi", "Pendidikan", "Sosial & Donasi", "Lain-lain"},
		Income:  []string{"Gaji", "Bonus", "Investasi", "Lainnya"},
		Savings: []string{"Tabungan Darurat", "Pendidikan", "Pensiun", "Lainnya"},
	}
}

// TestSnapCategory_ExactAndSubstring verifies the model's near-miss names are
// reconciled onto real database category names.
func TestSnapCategory_ExactAndSubstring(t *testing.T) {
	cats := seededCats()
	cases := []struct {
		name   string
		raw    string
		txType string
		want   string
	}{
		{"exact", "Transportasi", "expense", "Transportasi"},
		{"case-insensitive", "transportasi", "expense", "Transportasi"},
		{"substring into db name", "Makanan", "expense", "Makanan & Minuman"},
		{"db name contains raw", "Makanan & Minuman", "expense", "Makanan & Minuman"},
		{"savings type", "Pensiun", "savings", "Pensiun"},
		{"income type", "Gaji", "income", "Gaji"},
		{"generic fallback expense", "Lainnya", "expense", "Lain-lain"},
		{"generic fallback income", "Lain-lain", "income", "Lainnya"},
		{"unknown snaps to generic bucket", "Zzz", "expense", "Lain-lain"},
		{"empty stays empty", "", "expense", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := snapCategory(tc.raw, tc.txType, cats); got != tc.want {
				t.Fatalf("snapCategory(%q, %q) = %q, want %q", tc.raw, tc.txType, got, tc.want)
			}
		})
	}
}

// TestNormalizeReceiptResult verifies the single- and multi-transaction shapes
// are both reconciled with the database.
func TestNormalizeReceiptResult(t *testing.T) {
	cats := seededCats()

	t.Run("single fills description from merchant", func(t *testing.T) {
		res := map[string]interface{}{
			"type":     "expense",
			"category": "Makanan",
			"merchant": "Warung Bu Tini",
			"total":    float64(25000),
		}
		normalizeReceiptResult(res, cats)
		if res["category"] != "Makanan & Minuman" {
			t.Fatalf("expected category snapped, got %v", res["category"])
		}
		if res["description"] != "Warung Bu Tini" {
			t.Fatalf("expected description from merchant, got %v", res["description"])
		}
	})

	t.Run("multi transaction each normalized", func(t *testing.T) {
		res := map[string]interface{}{
			"transactions": []interface{}{
				map[string]interface{}{"type": "expense", "category": "Transportasi", "total": float64(239000)},
				map[string]interface{}{"type": "savings", "category": "Pendidikan", "description": "tabungan anak", "total": float64(851000)},
			},
		}
		normalizeReceiptResult(res, cats)
		txs := res["transactions"].([]interface{})
		first := txs[0].(map[string]interface{})
		if first["category"] != "Transportasi" {
			t.Fatalf("expected first category kept, got %v", first["category"])
		}
		second := txs[1].(map[string]interface{})
		if second["category"] != "Pendidikan" {
			t.Fatalf("expected savings category kept, got %v", second["category"])
		}
		if second["description"] != "tabungan anak" {
			t.Fatalf("expected description preserved, got %v", second["description"])
		}
	})

	t.Run("nil-safe", func(t *testing.T) {
		normalizeReceiptResult(nil, cats) // must not panic
	})
}

// TestLoadReceiptCategories verifies the real category names are loaded from
// the repositories and the loader is nil-safe.
func TestLoadReceiptCategories(t *testing.T) {
	u := &AIUsecase{
		expenseCatRepo: &fakeExpenseCatRepo{names: []string{"Transportasi", "Makanan & Minuman"}},
		incomeCatRepo:  &fakeIncomeCatRepo{names: []string{"Gaji"}},
		savingsCatRepo: &fakeSavingsCatRepo{names: []string{"Pensiun"}},
	}
	cats := u.loadReceiptCategories(context.Background())
	if len(cats.Expense) != 2 || cats.Expense[0] != "Transportasi" {
		t.Fatalf("unexpected expense categories: %+v", cats.Expense)
	}
	if len(cats.Income) != 1 || cats.Income[0] != "Gaji" {
		t.Fatalf("unexpected income categories: %+v", cats.Income)
	}
	if len(cats.Savings) != 1 || cats.Savings[0] != "Pensiun" {
		t.Fatalf("unexpected savings categories: %+v", cats.Savings)
	}

	// Nil repos must not panic and must yield empty lists.
	empty := (&AIUsecase{}).loadReceiptCategories(context.Background())
	if len(empty.Expense) != 0 || len(empty.Income) != 0 || len(empty.Savings) != 0 {
		t.Fatalf("expected empty categories without repos, got %+v", empty)
	}
}

// TestReceiptPromptInjectsCategoriesAndDate verifies the prompt carries the
// live category names and today's date so the model can pick an existing name
// and resolve relative dates.
func TestReceiptPromptInjectsCategoriesAndDate(t *testing.T) {
	cats := seededCats()
	prompt := receiptPrompt(cats, mustParseDate(t, "2026-10-15"))
	for _, want := range []string{"Makanan & Minuman", "Sosial & Donasi", "Pensiun", "2026-10-15", "Kamis"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\n%s", want, prompt)
		}
	}
}

// TestParseReceiptResponseRespectsRefusal verifies a model "not a receipt"
// answer is not force-flipped to success.
func TestParseReceiptResponseRespectsRefusal(t *testing.T) {
	res := parseReceiptResponse(`{"success": false, "message": "Bukan struk"}`)
	if res["success"] != false {
		t.Fatalf("expected success=false preserved, got %v", res["success"])
	}

	// A plain JSON answer without a success flag defaults to true.
	ok := parseReceiptResponse(`{"merchant":"Toko","total":1000,"category":"Lainnya","type":"expense"}`)
	if ok["success"] != true {
		t.Fatalf("expected success defaulted to true, got %v", ok["success"])
	}
}

// TestParseReceiptResponse_JSONNullDoesNotPanic guards the nil-map panic: a bare
// `null` unmarshals into a nil map with no error, and writing to it would crash
// the chat worker goroutine.
func TestParseReceiptResponse_JSONNullDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("parseReceiptResponse panicked on JSON null: %v", r)
		}
	}()
	res := parseReceiptResponse(`null`)
	if res == nil {
		t.Fatal("expected a non-nil result map")
	}
	if _, ok := res["type"]; !ok {
		t.Fatal("expected a defaulted type field")
	}
	// A scalar/array must be handled the same way.
	if got := parseReceiptResponse(`"just a string"`); got == nil {
		t.Fatal("expected a non-nil result map for a scalar")
	}
	if got := parseReceiptResponse(`[1,2,3]`); got == nil {
		t.Fatal("expected a non-nil result map for an array")
	}
}
