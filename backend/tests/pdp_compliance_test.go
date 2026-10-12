package tests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/fintrack/backend/internal/user"
)

// TestP08_009_And_010_PDPComplianceAndTaxRetention tests UU PDP No. 27/2022 right to erasure
// alongside 10-year statutory tax retention under UU KUP No. 28/2007.
// Invariant: PII completely shredded, chat & vector memory wiped, financial invoices
// and payments retained with exact Rupiah values, audit log entry recorded.
func TestP08_009_And_010_PDPComplianceAndTaxRetention(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PDP compliance test")
	}
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	defer pool.Close()

	anonymizeSvc := user.NewAnonymizeService(pool)

	// 1. Provision target user
	userID := uuid.New()
	origEmail := fmt.Sprintf("pdp-victim-%s@example.com", userID.String()[:8])
	origName := "Budi Santoso Rahasia"
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, role)
		 VALUES ($1, $2, 'hashed_secret_password_123', $3, 'user')`,
		userID, origEmail, origName); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	}()

	// 2. Provision PII / Non-financial data:
	// a. Chat session & messages
	sessionID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO chat_sessions (id, user_id, title) VALUES ($1, $2, 'Konsultasi Hutang')`,
		sessionID, userID); err != nil {
		t.Fatalf("insert chat session: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO chat_messages (id, session_id, user_id, role, content)
		 VALUES (gen_random_uuid(), $1, $2, 'user', 'Saya punya hutang 50 juta dan rahasia pribadi')`,
		sessionID, userID); err != nil {
		t.Fatalf("insert chat message: %v", err)
	}

	// b. Vector embedding
	var expCatID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM expense_categories LIMIT 1`).Scan(&expCatID)
	if err != nil {
		expCatID = uuid.New()
		_, _ = pool.Exec(ctx, `INSERT INTO expense_categories (id, name) VALUES ($1, 'Test Cat')`, expCatID)
	}

	txID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO transactions (id, user_id, type, amount, expense_category_id, transaction_date, description)
		 VALUES ($1, $2, 'expense', 500000, $3, CURRENT_DATE, 'Belanja obat pribadi')`,
		txID, userID, expCatID); err != nil {
		t.Fatalf("insert transaction: %v", err)
	}
	var vecStr strings.Builder
	vecStr.WriteString("[0.5")
	for i := 1; i < 768; i++ {
		vecStr.WriteString(",0")
	}
	vecStr.WriteString("]")
	if _, err := pool.Exec(ctx,
		`INSERT INTO transaction_embeddings (id, transaction_id, user_id, content, embedding)
		 VALUES (gen_random_uuid(), $1, $2, 'obat pribadi', $3::vector)`,
		txID, userID, vecStr.String()); err != nil {
		t.Fatalf("insert embedding: %v", err)
	}

	// c. Refresh token & Gemini key
	if _, err := pool.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		 VALUES (gen_random_uuid(), $1, 'hash123', NOW() + interval '7 days')`, userID); err != nil {
		t.Fatalf("insert refresh token: %v", err)
	}


	// 3. Provision Financial / Tax Data:
	// Invoice & Payment (statutory tax records)
	invID := uuid.New()
	invoiceNum := fmt.Sprintf("INV-PDP-%s", userID.String()[:8])
	const subtotal = int64(150000)
	const taxAmount = int64(16500) // PPN 11%
	const total = int64(166500)

	if _, err := pool.Exec(ctx,
		`INSERT INTO invoices (id, invoice_number, user_id, subtotal_idr, tax_idr, total_idr, status, issued_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'paid', NOW() - interval '3 months')`,
		invID, invoiceNum, userID, subtotal, taxAmount, total); err != nil {
		t.Fatalf("insert invoice: %v", err)
	}
	payID := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO payments (id, invoice_id, user_id, gateway, amount_idr, status, idempotency_key)
		 VALUES ($1, $2, $3, 'midtrans', $4, 'settled', $5)`,
		payID, invID, userID, total, "idem-"+payID.String()); err != nil {
		t.Fatalf("insert payment: %v", err)
	}

	// 4. EXECUTE TASK-P08-009: Anonymize User Data (Right to Erasure)
	result, err := anonymizeSvc.AnonymizeUserData(ctx, userID, nil, "Permintaan Pengguna Sesuai UU PDP Pasal 43")
	if err != nil {
		t.Fatalf("AnonymizeUserData failed: %v", err)
	}

	t.Logf("Anonymize result: chatDeleted=%d, embeddingsDeleted=%d, invoicesRetained=%d, paymentsRetained=%d",
		result.ChatMessagesDeleted, result.EmbeddingsDeleted, result.InvoicesRetained, result.PaymentsRetained)

	// Verify TASK-P08-009 (Data Wiped):
	// a. Chat messages & sessions completely deleted
	var chatCount, sessionCount, embedCount, tokenCount int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM chat_messages WHERE user_id = $1`, userID).Scan(&chatCount)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM chat_sessions WHERE user_id = $1`, userID).Scan(&sessionCount)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM transaction_embeddings WHERE user_id = $1`, userID).Scan(&embedCount)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM refresh_tokens WHERE user_id = $1`, userID).Scan(&tokenCount)

	if chatCount != 0 || sessionCount != 0 || embedCount != 0 || tokenCount != 0 {
		t.Errorf("PII data leak! chat=%d, sessions=%d, embeddings=%d, tokens=%d",
			chatCount, sessionCount, embedCount, tokenCount)
	}

	// b. User record pseudonymized and shredded
	var userEmail, userName, userPass string
	var isActive bool
	err = pool.QueryRow(ctx,
		`SELECT email, full_name, password_hash, is_active FROM users WHERE id = $1`, userID).
		Scan(&userEmail, &userName, &userPass, &isActive)
	if err != nil {
		t.Fatalf("query user: %v", err)
	}

	expectedEmail := fmt.Sprintf("anonymized-%s@deleted.fintrack.internal", userID.String())
	if userEmail != expectedEmail {
		t.Errorf("expected email %s, got %s", expectedEmail, userEmail)
	}
	if userName != "Eks Pengguna (Teranonimkan)" {
		t.Errorf("expected pseudonymized name, got %s", userName)
	}
	if userPass != "DELETED_CRYPTOGRAPHIC_SHREDDING" {
		t.Errorf("expected password shredded, got %s", userPass)
	}
	if isActive != false {
		t.Errorf("expected user is_active false, got %v", isActive)
	}

	// c. Audit log written
	var auditCount int
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE action = 'user.pdp_anonymize' AND entity_id = $1`,
		userID.String()).Scan(&auditCount)
	if auditCount == 0 {
		t.Errorf("expected audit_logs entry for PDP anonymization")
	}

	// Verify TASK-P08-010 (10-Year Statutory Tax Retention Integrity):
	// Invoices and payments must remain intact with exact monetary amounts
	var invTotal, invTax, invSubtotal int64
	var invStatus string
	err = pool.QueryRow(ctx,
		`SELECT subtotal_idr, tax_idr, total_idr, status FROM invoices WHERE id = $1`, invID).
		Scan(&invSubtotal, &invTax, &invTotal, &invStatus)
	if err != nil {
		t.Fatalf("query invoice: %v", err)
	}

	if invStatus != "paid" {
		t.Errorf("expected invoice status paid, got %s", invStatus)
	}
	if invSubtotal != subtotal || invTax != taxAmount || invTotal != total {
		t.Errorf("Tax integrity violated! got (subtotal=%d, tax=%d, total=%d), want (%d, %d, %d)",
			invSubtotal, invTax, invTotal, subtotal, taxAmount, total)
	}

	var payAmount int64
	var payStatus string
	err = pool.QueryRow(ctx,
		`SELECT amount_idr, status FROM payments WHERE id = $1`, payID).
		Scan(&payAmount, &payStatus)
	if err != nil {
		t.Fatalf("query payment: %v", err)
	}
	if payStatus != "settled" || payAmount != total {
		t.Errorf("Payment integrity violated! got amount=%d status=%s, want %d settled",
			payAmount, payStatus, total)
	}

	t.Log("TASK-P08-009 & TASK-P08-010 PASSED: PII completely shredded, tax records preserved with 100% financial balance.")
}
