package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"github.com/fintrack/backend/internal/domain"
	infraDB "github.com/fintrack/backend/internal/infrastructure/db"
	"github.com/fintrack/backend/internal/infrastructure/embedding"
	"github.com/fintrack/backend/internal/repository/postgres"
)

// cmd/backfill generates embeddings for transactions that lack one.
//
// Usage:
//
//	go run ./cmd/backfill [--limit N] [--user <uuid>]
func main() {
	_ = godotenv.Load()

	limit := flag.Int("limit", 0, "max number of transactions to backfill (0 = all)")
	userFilter := flag.String("user", "", "restrict to a single user id")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}
	apiKey := os.Getenv("GEMINI_EMBEDDING_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "GEMINI_EMBEDDING_API_KEY is required for backfill")
		os.Exit(1)
	}

	ctx := context.Background()

	poolCfg := infraDB.PoolConfigFromEnv(os.Getenv, func(key string, def int) int {
		if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
			return v
		}
		return def
	})
	pool, err := infraDB.NewPool(ctx, databaseURL, poolCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	embedRepo := postgres.NewEmbeddingRepo(pool)
	incomeCatRepo := postgres.NewIncomeCategoryRepo(pool)
	expenseCatRepo := postgres.NewExpenseCategoryRepo(pool)
	savingsCatRepo := postgres.NewSavingsCategoryRepo(pool)
	embedder := embedding.NewGeminiEmbedder(apiKey)

	var userUUID *uuid.UUID
	if *userFilter != "" {
		id, err := uuid.Parse(*userFilter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --user: %v\n", err)
			os.Exit(1)
		}
		userUUID = &id
	}

	const batch = 20
	processed := 0

	for {
		if *limit > 0 && processed >= *limit {
			break
		}
		txs, err := embedRepo.ListMissingTransactions(ctx, batch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to list missing: %v\n", err)
			os.Exit(1)
		}
		if len(txs) == 0 {
			break
		}

		progressed := false
		for i := range txs {
			if *limit > 0 && processed >= *limit {
				break
			}
			t := &txs[i]
			if userUUID != nil && t.UserID != *userUUID {
				continue
			}

			catName := resolveCategoryName(ctx, t, incomeCatRepo, expenseCatRepo, savingsCatRepo)
			desc := ""
			if t.Description != nil {
				desc = *t.Description
			}
			content := fmt.Sprintf("%s %s %s %s Rp%.0f",
				t.TransactionDate.Format("2006-01-02"), t.Type, catName, desc, t.Amount)

			vec, err := embedder.Embed(ctx, content)
			if err != nil {
				fmt.Fprintf(os.Stderr, "embed failed for %s: %v\n", t.ID, err)
				continue
			}
			metadata := map[string]interface{}{
				"transaction_id": t.ID.String(),
				"date":           t.TransactionDate.Format("2006-01-02"),
				"type":           string(t.Type),
				"category":       catName,
				"amount":         t.Amount,
			}
			if err := embedRepo.Upsert(ctx, t.UserID, t.ID, content, vec, metadata); err != nil {
				fmt.Fprintf(os.Stderr, "upsert failed for %s: %v\n", t.ID, err)
				continue
			}
			processed++
			progressed = true
			fmt.Printf("%d embedded (%s)\n", processed, t.ID)
		}

		// Avoid an infinite loop if every row was filtered out.
		if !progressed {
			break
		}

		time.Sleep(200 * time.Millisecond)
	}

	fmt.Printf("backfill complete: %d transactions embedded\n", processed)
}

func resolveCategoryName(
	ctx context.Context,
	t *domain.Transaction,
	incomeRepo *postgres.IncomeCategoryRepo,
	expenseRepo *postgres.ExpenseCategoryRepo,
	savingsRepo *postgres.SavingsCategoryRepo,
) string {
	if t.IncomeCategoryID != nil {
		if c, err := incomeRepo.GetByID(ctx, *t.IncomeCategoryID); err == nil && c != nil {
			return c.Name
		}
	}
	if t.ExpenseCategoryID != nil {
		if c, err := expenseRepo.GetByID(ctx, *t.ExpenseCategoryID); err == nil && c != nil {
			return c.Name
		}
	}
	if t.SavingsCategoryID != nil {
		if c, err := savingsRepo.GetByID(ctx, *t.SavingsCategoryID); err == nil && c != nil {
			return c.Name
		}
	}
	return ""
}
