# Strategi Embedding 768d & Integritas pgvector

Dokumen ini mendefinisikan persyaratan teknis, validasi dimensi, dan integrasi modul embedding pada FinTrack v2 untuk menjamin konsistensi pencarian semantik (RAG) pada tabel PostgreSQL `transaction_embeddings`.

---

## 1. Latar Belakang & Batasan Teknis

Pada arsitektur FinTrack v2 (P01), fitur RAG (Retrieval-Augmented Generation) digunakan untuk mencari riwayat transaksi keuangan pengguna berdasarkan kemiripan semantik (contoh: query *"beli kopi minggu lalu"* mencocokkan transaksi *"Starbucks Reserve"* atau *"Kopi Kenangan"*).

Indeks vektor di database PostgreSQL didefinisikan sebagai:
```sql
CREATE TABLE IF NOT EXISTS transaction_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    embedding vector(768) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tx_embedding_cosine 
ON transaction_embeddings USING ivfflat (embedding vector_cosine_ops) 
WITH (lists = 100);
```

### Konsekuensi Kritis:
- Tipe `vector(768)` di pgvector **mengunci panjang array tepat pada 768 float**.
- Memasukkan array vektor dengan panjang selain 768 (misal 1536 float dari OpenAI `text-embedding-3-small`, atau 384 float dari `all-MiniLM-L6-v2`) akan memicu error tingkat database:
  `ERROR: different vector dimensions 1536 and 768 (SQLSTATE 22000)`.
- Oleh karena itu, sesuai **K-18** dan **K-P03-10**, seluruh model embedding di AI Gateway **wajib dikunci pada dimensi 768**.

---

## 2. Model Acuan & Konfigurasi Route

Model default yang digunakan adalah **`text-embedding-004`** (dari Google/Gemini ecosystem atau endpoint OpenAI-compatible yang menyediakan embedding 768d, misalnya via 9router).

Konfigurasi rute embedding di tabel `ai_routes`:
```sql
INSERT INTO ai_routes (use_case, endpoint_id, model, combo, priority, is_active, timeout_ms)
VALUES (
    'embedding',
    (SELECT id FROM ai_endpoints WHERE name = '9router-local-primary'),
    'text-embedding-004',
    'default',
    1,
    TRUE,
    15000
);
```

---

## 3. Validasi Runtime Ketat di Adapter Embedder

Package `backend/internal/infrastructure/embedding/embedder.go` tidak lagi membuat HTTP request mandiri ke Google SDK, melainkan memanggil gateway service dengan guard verifikasi dimensi:

```go
package embedding

import (
	"context"
	"errors"
	"fmt"

	"github.com/fintrack/backend/internal/domain"
)

var (
	ErrInvalidDimension = errors.New("embedding dimension mismatch; expected 768")
)

type GatewayEmbedder struct {
	gateway domain.AIGatewayService
}

func NewGatewayEmbedder(gw domain.AIGatewayService) *GatewayEmbedder {
	return &GatewayEmbedder{gateway: gw}
}

// GenerateEmbedding menghasilkan vektor 768 float dan memvalidasi dimensinya
func (e *GatewayEmbedder) GenerateEmbedding(ctx context.Context, text string) ([]float32, *domain.TokenUsage, error) {
	if text == "" {
		return nil, nil, errors.New("cannot embed empty text")
	}

	vector, usage, err := e.gateway.CreateEmbedding(ctx, text)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// [CRITICAL GUARD] Validasi panjang dimensi vektor
	if len(vector) != 768 {
		return nil, nil, fmt.Errorf("%w: upstream returned %d dimensions", ErrInvalidDimension, len(vector))
	}

	return vector, usage, nil
}
```

---

## 4. Query RAG & Cosine Similarity

Dalam pencarian transaksi di `AIUsecase`, vektor query pengguna dicari terhadap riwayat transaksi menggunakan operator cosine distance (`<=>`):

```sql
SELECT 
    t.id, 
    t.amount, 
    t.description, 
    t.transaction_date, 
    c.name AS category_name,
    1 - (te.embedding <=> $1) AS similarity_score
FROM transaction_embeddings te
JOIN transactions t ON t.id = te.transaction_id
LEFT JOIN expense_categories c ON c.id = t.expense_category_id
WHERE t.user_id = $2
ORDER BY te.embedding <=> $1 ASC
LIMIT 10;
```

---

## 5. Prosedur Uji Integritas Embedding (Test Plan)

Setiap implementasi atau pengujian embedding di Fase P03 wajib memverifikasi:
1. **Pengecekan Panjang Array**: `assert.Len(t, embeddingVector, 768)`.
2. **Pengecekan Non-Zero Float**: Memastikan array bukan merupakan vektor kosong `[0.0, 0.0, ...]`.
3. **Pengecekan Insertion ke PostgreSQL**: Uji coba insert langsung ke tabel `transaction_embeddings` dummy untuk membuktikan tidak terjadi error mismatch pgvector.
4. **Pengecekan Score Similarity**: Query dengan teks identik menghasilkan similarity score mendekati `1.0`.
