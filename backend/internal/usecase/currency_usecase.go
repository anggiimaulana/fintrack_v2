package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/currency"
)

// ErrCurrencyActivation means a currency cannot be activated without a rate.
var ErrCurrencyActivation = errors.New("currency needs a rate before activation")

// ErrCurrencyInvalid is returned for any malformed catalog metadata.
var ErrCurrencyInvalid = errors.New("currency metadata invalid")

var (
	codeRe   = regexp.MustCompile(`^[A-Z]{3}$`)
	localeRe = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
)

// CurrencyInput is the presence-aware write shape for the catalog. Optional
// fields are pointers so an update can explicitly clear a value (e.g. deactivate
// with is_active=false) without accidentally wiping fields the caller omitted.
type CurrencyInput struct {
	Code          string
	Name          string
	Symbol        *string
	Locale        *string
	DecimalDigits *int
	IsActive      *bool
	SortOrder     *int
	RateSource    *string
}

// CurrencyUsecase exposes the dynamic currency catalog/rates to the transport
// layer and enforces the activation invariant (no active currency without a
// currently-effective rate).
type CurrencyUsecase struct {
	svc     *currency.Service
	repo    currency.AdminRepo
	factory *currency.ProviderFactory
}

func NewCurrencyUsecase(svc *currency.Service, repo currency.AdminRepo, factory ...*currency.ProviderFactory) *CurrencyUsecase {
	uc := &CurrencyUsecase{svc: svc, repo: repo}
	if len(factory) > 0 {
		uc.factory = factory[0]
	}
	return uc
}

func (u *CurrencyUsecase) WithProviderFactory(f *currency.ProviderFactory) *CurrencyUsecase {
	u.factory = f
	return u
}

// Catalog returns the selectable (active) currencies.
func (u *CurrencyUsecase) Catalog(ctx context.Context) ([]currency.Meta, error) {
	return u.svc.Catalog(ctx)
}

// FullCatalog returns every currency including inactive (Super Admin view).
func (u *CurrencyUsecase) FullCatalog(ctx context.Context) ([]currency.Meta, error) {
	return u.svc.FullCatalog(ctx)
}

// Rates returns the append-only rate history for a currency.
func (u *CurrencyUsecase) Rates(ctx context.Context, code string, limit int) ([]currency.Rate, error) {
	return u.repo.ListRates(ctx, strings.ToUpper(strings.TrimSpace(code)), limit)
}

// SyncRates triggers an on-demand synchronization pass for all provider-backed currencies.
func (u *CurrencyUsecase) SyncRates(ctx context.Context) (int, error) {
	if u.factory == nil {
		return 0, errors.New("provider factory belum terkonfigurasi")
	}
	worker := currency.NewRateSyncWorker(u.repo, u.factory, u.factory.Resolver())
	return worker.SyncOnce(ctx)
}

// RateAsOf returns the rate applicable at a moment (for the frontend catalog).
func (u *CurrencyUsecase) RateAsOf(ctx context.Context, code string, at time.Time) (*currency.Rate, error) {
	return u.svc.RateAsOf(ctx, strings.ToUpper(strings.TrimSpace(code)), at)
}

// UpsertCurrency creates or updates catalog metadata. Activation requires a rate
// that is effective now; the base currency cannot be modified.
func (u *CurrencyUsecase) UpsertCurrency(ctx context.Context, in CurrencyInput) (*currency.Meta, error) {
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	if !codeRe.MatchString(code) {
		return nil, fmt.Errorf("%w: kode mata uang harus 3 huruf (mis. JPY)", ErrCurrencyInvalid)
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 64 {
		return nil, fmt.Errorf("%w: nama wajib diisi (maks 64 karakter)", ErrCurrencyInvalid)
	}

	existing, err := u.svc.Meta(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.IsBase {
		return nil, fmt.Errorf("%w: mata uang dasar tidak dapat diubah", ErrCurrencyInvalid)
	}

	// Merge with the existing row so an update that omits a field keeps it.
	m := currency.Meta{Code: code, Name: name}
	if existing != nil {
		m = *existing
		m.Code = code
		m.Name = name
	}
	if in.Symbol != nil {
		m.Symbol = strings.TrimSpace(*in.Symbol)
	}
	if in.Locale != nil {
		m.Locale = strings.TrimSpace(*in.Locale)
	}
	if in.DecimalDigits != nil {
		m.DecimalDigits = *in.DecimalDigits
	}
	if in.IsActive != nil {
		m.IsActive = *in.IsActive
	}
	if in.SortOrder != nil {
		m.SortOrder = *in.SortOrder
	}
	if in.RateSource != nil {
		m.RateSource = strings.TrimSpace(*in.RateSource)
	}
	if existing == nil {
		// Sensible defaults for a brand-new currency.
		if m.Symbol == "" {
			m.Symbol = code
		}
		if m.Locale == "" {
			m.Locale = "id-ID"
		}
	}

	if m.Symbol == "" || len(m.Symbol) > 8 {
		return nil, fmt.Errorf("%w: simbol wajib diisi (maks 8 karakter)", ErrCurrencyInvalid)
	}
	if m.Locale == "" || len(m.Locale) > 16 || !localeRe.MatchString(m.Locale) {
		return nil, fmt.Errorf("%w: locale tidak valid (mis. id-ID, ja-JP)", ErrCurrencyInvalid)
	}
	if m.DecimalDigits < 0 || m.DecimalDigits > 4 {
		return nil, fmt.Errorf("%w: jumlah desimal harus 0..4", ErrCurrencyInvalid)
	}
	if m.RateSource == "" {
		m.RateSource = "manual"
	}
	if m.RateSource != "manual" && m.RateSource != "provider" {
		return nil, fmt.Errorf("%w: sumber kurs harus manual atau provider", ErrCurrencyInvalid)
	}

	// Ensure provider-backed currencies have an initial rate row even if initially inactive.
	if m.RateSource == "provider" && u.factory != nil {
		r, _ := u.svc.RateAsOf(ctx, code, time.Now())
		if r == nil {
			provider, perr := u.factory.ResolveProvider(ctx)
			if perr != nil {
				return nil, fmt.Errorf("%w: gagal menyiapkan provider kurs: %v", ErrCurrencyInvalid, perr)
			}
			fetchedRate, ferr := provider.FetchRate(ctx, code, currency.BaseCode)
			if ferr != nil {
				return nil, fmt.Errorf("%w: gagal mengambil kurs otomatis dari provider (%s): %v", ErrCurrencyInvalid, provider.Name(), ferr)
			}
			if fetchedRate <= 0 || math.IsNaN(fetchedRate) || math.IsInf(fetchedRate, 0) {
				return nil, fmt.Errorf("%w: nilai kurs dari provider tidak valid (<= 0)", ErrCurrencyInvalid)
			}
			note := "auto-fetch initial"
			initialRate := currency.Rate{
				CurrencyCode: code,
				RateToBase:   fetchedRate,
				EffectiveAt:  time.Now(),
				Source:       "provider",
				Note:         &note,
			}
			if err := u.repo.InsertRate(ctx, initialRate); err != nil {
				return nil, fmt.Errorf("%w: gagal menyimpan kurs awal: %v", ErrCurrencyInvalid, err)
			}
		}
	}

	if m.IsActive {
		r, rerr := u.svc.RateAsOf(ctx, code, time.Now())
		if rerr != nil {
			return nil, rerr
		}
		if r == nil {
			return nil, ErrCurrencyActivation
		}
	}
	if err := u.repo.UpsertCurrency(ctx, m); err != nil {
		return nil, err
	}
	return u.svc.Meta(ctx, code)
}

// AddRate appends a rate (append-only; corrections are new rows).
func (u *CurrencyUsecase) AddRate(ctx context.Context, rt currency.Rate, actorID *uuid.UUID) error {
	code := strings.ToUpper(strings.TrimSpace(rt.CurrencyCode))
	if !codeRe.MatchString(code) {
		return fmt.Errorf("%w: kode mata uang tidak valid", ErrCurrencyInvalid)
	}
	if math.IsNaN(rt.RateToBase) || math.IsInf(rt.RateToBase, 0) || rt.RateToBase <= 0 {
		return errors.New("rate must be a positive finite number")
	}
	// rate_to_base is numeric(20,10): 10 integer digits. A larger value would be
	// rejected by Postgres with a 500, so fail fast with a clear message here.
	if rt.RateToBase >= 1e10 {
		return errors.New("rate is out of range")
	}
	m, err := u.svc.Meta(ctx, code)
	if err != nil {
		return err
	}
	if m == nil {
		return currency.ErrUnknown
	}
	if rt.Source == "" {
		rt.Source = "manual"
	}
	if rt.Source != "manual" && rt.Source != "provider" {
		return fmt.Errorf("%w: sumber kurs harus manual atau provider", ErrCurrencyInvalid)
	}
	if rt.EffectiveAt.IsZero() {
		rt.EffectiveAt = time.Now()
	}
	rt.CurrencyCode = code
	rt.CreatedBy = actorID
	return u.repo.InsertRate(ctx, rt)
}
