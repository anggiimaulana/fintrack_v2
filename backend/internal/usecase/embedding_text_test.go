package usecase

import (
	"testing"
	"time"

	"github.com/fintrack/backend/internal/domain"
)

func TestBuildEmbeddingText(t *testing.T) {
	desc := "makan siang sate padang"
	date := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)

	tx := &domain.Transaction{
		Type:            domain.TransactionExpense,
		Amount:          35000,
		Description:     &desc,
		TransactionDate: date,
	}

	got := buildEmbeddingText(tx, "Makanan & Minuman")
	want := "2026-04-15 expense Makanan & Minuman makan siang sate padang Rp35000"
	if got != want {
		t.Fatalf("buildEmbeddingText = %q, want %q", got, want)
	}
}

func TestBuildEmbeddingTextNilDescription(t *testing.T) {
	tx := &domain.Transaction{
		Type:            domain.TransactionIncome,
		Amount:          7000000,
		TransactionDate: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
	}
	got := buildEmbeddingText(tx, "Gaji Tetap")
	want := "2026-05-25 income Gaji Tetap  Rp7000000"
	if got != want {
		t.Fatalf("buildEmbeddingText = %q, want %q", got, want)
	}
}
