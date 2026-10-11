# Tabel Routing AI Dinamis & Pemetaan Provider

Dokumen ini mendefinisikan struktur tabel `ai_routes`, strategi pemetaan rute per use case, transisi dari map hardcoded lama, dan penanganan timeout.

---

## 1. Skema Database `ai_routes`

Tabel `ai_routes` menghubungkan use-case fitur finansial dengan model dan endpoint target.

```sql
CREATE TABLE IF NOT EXISTS ai_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    use_case VARCHAR(50) NOT NULL,
    endpoint_id UUID NOT NULL REFERENCES ai_endpoints(id) ON DELETE CASCADE,
    model VARCHAR(100) NOT NULL,
    combo VARCHAR(50) NOT NULL DEFAULT 'default',
    priority INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    timeout_ms INT NOT NULL DEFAULT 30000,
    max_retries INT NOT NULL DEFAULT 2,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_use_case CHECK (use_case IN ('chat', 'vision', 'voice', 'embedding', 'reasoning'))
);

CREATE INDEX idx_ai_routes_lookup ON ai_routes(use_case, is_active, priority);
```

### Penjelasan Kolom:
- `use_case`: Kategori fitur (`chat`, `vision`, `voice`, `embedding`, `reasoning`).
- `endpoint_id`: Relasi ke endpoint penyedia koneksi di tabel `ai_endpoints`.
- `model`: Nama model yang dikirimkan ke payload OpenAI-compatible (contoh: `gpt-4o-mini`, `deepseek-r1`, `text-embedding-004`).
- `combo`: Profil performa yang dipilih pengguna/sistem (`default`, `fast`, `precise`, `reasoning`).
- `priority`: Urutan fallback chain dalam use-case yang sama (1 = utama, 2 = sekunder, dst).
- `is_active`: Flag aktif/nonaktif rute secara instan tanpa menghapus baris.
- `timeout_ms`: Batas waktu maksimum eksekusi HTTP request (default 30.000 ms = 30 detik).
- `max_retries`: Jumlah percobaan ulang sebelum dialihkan ke rute prioritas berikutnya.

---

## 2. Transisi dari Hardcoded Map Lama

Sebelum Fase P03, modul `backend/internal/usecase/ai_usecase.go` memuat pemetaan hardcoded seperti berikut:

```go
// [POLA LAMA DI AI_USECASE.GO - DIHAPUS PADA P03]
var providerEndpoints = map[string]struct{ baseURL, model, path string }{
    "gemini":     {baseURL: "https://generativelanguage.googleapis.com/v1beta", model: "gemini-1.5-flash"},
    "openai":     {baseURL: "https://api.openai.com/v1", model: "gpt-4o-mini"},
    "groq":       {baseURL: "https://api.groq.com/openai/v1", model: "meta-llama/llama-4-scout-17b-16e-instruct"},
    "openrouter": {baseURL: "https://openrouter.ai/api/v1", model: "meta-llama/llama-3.1-8b-instruct:free"},
    "qwen":       {baseURL: "https://api.qwen.ai/v1", model: "qwen-turbo"},
    "deepseek":   {baseURL: "https://api.deepseek.com/v1", model: "deepseek-chat"},
    "mistral":    {baseURL: "https://api.mistral.ai/v1", model: "mistral-small-latest"},
    "cohere":     {baseURL: "https://api.cohere.ai/v1", model: "command-r"},
    "perplexity": {baseURL: "https://api.perplexity.ai", model: "sonar-pro"},
}
```

Peta hardcoded di atas ditransformasikan menjadi entri dinamis di tabel `ai_routes` yang diarahkan ke 9router lokal atau provider fallback:

### Seed Pemetaan Default (`0007_ai_gateway.sql`):

| Use Case | Endpoint Target | Model | Priority | Timeout | Keterangan |
|---|---|---|---|---|---|
| `chat` | `9router-local-primary` | `gpt-4o-mini` | 1 | 30.000 ms | Rute utama obrolan keuangan |
| `chat` | `openai-cloud-fallback` | `gpt-4o-mini` | 2 | 30.000 ms | Cadangan jika 9router mati |
| `vision` | `9router-local-primary` | `gpt-4o-mini` | 1 | 45.000 ms | Analisa gambar bukti struk / nota |
| `voice` | `9router-local-primary` | `whisper-1` | 1 | 30.000 ms | Transkripsi perintah suara |
| `embedding` | `9router-local-primary` | `text-embedding-004` | 1 | 15.000 ms | Vektor transaksi (wajib 768d) |
| `reasoning` | `9router-local-primary` | `deepseek-r1` | 1 | 60.000 ms | Analisa keuangan mendalam |

---

## 3. Resolusi Rute Saat Runtime (Resolution Logic)

Alur resolusi rute di dalam `internal/aigateway/router`:

```go
func (r *Router) ResolveRoutes(ctx context.Context, useCase string, combo string) ([]RoutePlan, error) {
    // 1. Ambil seluruh rute aktif dari in-memory cache
    routes := r.cache.GetActiveRoutes(useCase, combo)
    if len(routes) == 0 {
        return nil, ErrNoActiveRouteForUseCase
    }

    var validPlans []RoutePlan
    for _, rt := range routes {
        endpoint := r.cache.GetEndpoint(rt.EndpointID)
        // 2. Evaluasi status endpoint & Circuit Breaker
        if endpoint == nil || endpoint.Status != "active" {
            continue
        }
        if r.circuitBreaker.IsOpen(endpoint.ID) {
            continue
        }

        validPlans = append(validPlans, RoutePlan{
            RouteID:    rt.ID,
            Endpoint:   endpoint,
            Model:      rt.Model,
            TimeoutMS:  rt.TimeoutMS,
            MaxRetries: rt.MaxRetries,
        })
    }

    if len(validPlans) == 0 {
        return nil, ErrAllRoutesUnavailable
    }

    return validPlans, nil
}
```

---

## 4. Keuntungan Pemisahan Rute Dinamis
1. **Zero-Code Model Upgrade**: Jika vendor merilis model baru yang lebih murah dan cepat (misal `gpt-4o-mini-2026`), admin cukup mengubah nilai kolom `model` pada rute tanpa compile ulang Go backend.
2. **Perlindungan Terhadap Lonjakan Biaya**: Fitur `reasoning` (DeepSeek-R1) atau `vision` dapat diatur batas timeout dan kuotanya tanpa mengganggu stabilitas chat harian.
3. **Pemberlakuan Kill Switch Per-Fitur**: Admin dapat menonaktifkan rute `voice` hanya dengan `UPDATE ai_routes SET is_active = false WHERE use_case = 'voice'`, menghentikan pengeluaran tanpa mematikan seluruh asisten.
