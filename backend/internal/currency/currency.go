// Package currency provides multi-currency conversion against a database-backed
// catalog and append-only rate history.
//
// Policy (which currencies exist, which rate applies) lives in DATA
// (currencies / currency_rates); this package only provides the CAPABILITY.
// IDR is the base accounting currency for this release.
package currency

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

// BaseCode is the base accounting currency. ponytail: single constant, not read
// from currencies.is_base — upgrade path is to resolve it from the catalog if a
// deployment ever needs a different base.
const BaseCode = "IDR"

// Errors surfaced to the transport layer.
var (
	ErrUnknown       = errors.New("currency unknown")
	ErrInactive      = errors.New("currency inactive")
	ErrNoRate        = errors.New("currency has no rate")
	ErrRateDuplicate = errors.New("a rate already exists for that effective time")
)

// Meta is the catalog entry for a currency (formatting + policy metadata).
type Meta struct {
	Code          string `json:"code" db:"code"`
	Name          string `json:"name" db:"name"`
	Symbol        string `json:"symbol" db:"symbol"`
	Locale        string `json:"locale" db:"locale"`
	DecimalDigits int    `json:"decimal_digits" db:"decimal_digits"`
	IsActive      bool   `json:"is_active" db:"is_active"`
	IsBase        bool   `json:"is_base" db:"is_base"`
	SortOrder     int    `json:"sort_order" db:"sort_order"`
	RateSource    string `json:"rate_source" db:"rate_source"`
}

// Rate is one append-only history row: 1 unit CurrencyCode = RateToBase base.
type Rate struct {
	ID           uuid.UUID  `json:"id,omitempty" db:"id"`
	CurrencyCode string     `json:"currency_code" db:"currency_code"`
	RateToBase   float64    `json:"rate_to_base" db:"rate_to_base"`
	EffectiveAt  time.Time  `json:"effective_at" db:"effective_at"`
	Source       string     `json:"source" db:"source"`
	Note         *string    `json:"note,omitempty" db:"note"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt    time.Time  `json:"created_at,omitempty" db:"created_at"`
}

// Repo is the persistence port.
type Repo interface {
	// List returns the catalog. When includeInactive is false, only active rows.
	List(ctx context.Context, includeInactive bool) ([]Meta, error)
	Get(ctx context.Context, code string) (*Meta, error)
	// RateAsOf returns the newest rate whose effective_at <= at, or nil.
	RateAsOf(ctx context.Context, code string, at time.Time) (*Rate, error)
	// InsertRate appends a rate row. Appending is the only mutation allowed.
	InsertRate(ctx context.Context, r Rate) error
	// ListRates returns the rate history for a currency, newest first.
	ListRates(ctx context.Context, code string, limit int) ([]Rate, error)
}

// AdminRepo extends Repo with catalog mutations used by the Super Admin surface.
type AdminRepo interface {
	Repo
	// CountRates reports how many rate rows exist for a currency.
	CountRates(ctx context.Context, code string) (int, error)
	// UpsertCurrency inserts or updates catalog metadata.
	UpsertCurrency(ctx context.Context, m Meta) error
}

// Service implements conversion on top of the catalog repository.
type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service { return &Service{repo: repo} }

// Catalog returns the selectable (active) currencies, ordered.
func (s *Service) Catalog(ctx context.Context) ([]Meta, error) {
	return s.repo.List(ctx, false)
}

// FullCatalog returns every currency, including inactive (Super Admin view).
func (s *Service) FullCatalog(ctx context.Context) ([]Meta, error) {
	return s.repo.List(ctx, true)
}

// Meta looks up one currency.
func (s *Service) Meta(ctx context.Context, code string) (*Meta, error) {
	return s.repo.Get(ctx, code)
}

// RateAsOf exposes the raw rate lookup (used for legends and audit).
func (s *Service) RateAsOf(ctx context.Context, code string, at time.Time) (*Rate, error) {
	if code == BaseCode {
		return &Rate{CurrencyCode: BaseCode, RateToBase: 1, EffectiveAt: at, Source: "base"}, nil
	}
	return s.repo.RateAsOf(ctx, code, at)
}

// ValidateSelectable reports whether a user may switch to this currency: it must
// exist, be active, and (unless base) have at least one rate available.
func (s *Service) ValidateSelectable(ctx context.Context, code string, at time.Time) error {
	if code == "" {
		return ErrUnknown
	}
	m, err := s.repo.Get(ctx, code)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrUnknown
	}
	if !m.IsActive {
		return ErrInactive
	}
	if code == BaseCode {
		return nil
	}
	r, err := s.repo.RateAsOf(ctx, code, at)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrNoRate
	}
	return nil
}

// RateForWrite resolves the rate to snapshot for a write at `at`. It validates
// the currency is active and, when `at` predates the rate history (a backdated
// entry), falls back to the newest known rate so the write is not rejected for a
// reason the user cannot act on.
func (s *Service) RateForWrite(ctx context.Context, code string, at time.Time) (*Rate, error) {
	if code == BaseCode {
		return &Rate{CurrencyCode: BaseCode, RateToBase: 1, EffectiveAt: at, Source: "base"}, nil
	}
	m, err := s.repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrUnknown
	}
	if !m.IsActive {
		return nil, ErrInactive
	}
	r, err := s.repo.RateAsOf(ctx, code, at)
	if err != nil {
		return nil, err
	}
	if r == nil {
		r, err = s.repo.RateAsOf(ctx, code, time.Now())
		if err != nil {
			return nil, err
		}
	}
	if r == nil {
		return nil, ErrNoRate
	}
	return r, nil
}

// ToBase converts an amount in `code` to base (IDR) using the rate as of `at`.
// It returns the base amount, the rate row used, and an error. For the base
// currency the rate is 1 and no query is issued.
func (s *Service) ToBase(ctx context.Context, code string, amount float64, at time.Time) (float64, *Rate, error) {
	if code == "" || code == BaseCode {
		return Round2(amount), &Rate{CurrencyCode: BaseCode, RateToBase: 1, EffectiveAt: at, Source: "base"}, nil
	}
	m, err := s.repo.Get(ctx, code)
	if err != nil {
		return 0, nil, err
	}
	if m == nil {
		return 0, nil, ErrUnknown
	}
	if !m.IsActive {
		return 0, nil, ErrInactive
	}
	r, err := s.repo.RateAsOf(ctx, code, at)
	if err != nil {
		return 0, nil, err
	}
	if r == nil {
		return 0, nil, ErrNoRate
	}
	return Round2(amount * r.RateToBase), r, nil
}

// FromBase converts an aggregate base (IDR) value into `code` using the rate as
// of `at`, rounded to the target currency's decimal digits.
func (s *Service) FromBase(ctx context.Context, code string, base float64, at time.Time) (float64, *Rate, error) {
	if code == "" || code == BaseCode {
		return Round2(base), &Rate{CurrencyCode: BaseCode, RateToBase: 1, EffectiveAt: at, Source: "base"}, nil
	}
	m, err := s.repo.Get(ctx, code)
	if err != nil {
		return 0, nil, err
	}
	if m == nil {
		return 0, nil, ErrUnknown
	}
	if !m.IsActive {
		return 0, nil, ErrInactive
	}
	r, err := s.repo.RateAsOf(ctx, code, at)
	if err != nil {
		return 0, nil, err
	}
	if r == nil || r.RateToBase == 0 {
		return 0, nil, ErrNoRate
	}
	return RoundTo(base/r.RateToBase, m.DecimalDigits), r, nil
}

// Round2 rounds to 2 decimals (base IDR precision). ponytail: float64 matches
// the rest of the codebase; amounts stay far below 2^53 so this is exact enough.
func Round2(v float64) float64 { return RoundTo(v, 2) }

// RoundTo rounds v to `digits` decimal places. Digits are clamped to the
// catalog's supported range (0..4) so a bad catalog value can never yield NaN
// (math.Pow overflow) or a silent no-round.
func RoundTo(v float64, digits int) float64 {
	if digits < 0 {
		digits = 0
	}
	if digits > 4 {
		digits = 4
	}
	p := math.Pow(10, float64(digits))
	if math.IsInf(p, 0) {
		return v
	}
	out := math.Round(v*p) / p
	if math.IsNaN(out) || math.IsInf(out, 0) {
		return v
	}
	return out
}
