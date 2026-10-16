# Arsitektur AI Gateway Terpusat

Dokumen ini menjelaskan arsitektur tingkat tinggi dan desain komponen untuk modul AI Gateway Terpusat FinTrack v2 (`internal/aigateway/`).

---

## 1. Latar Belakang & Masalah
Pada FinTrack v2 fase sebelumnya (P01), pemanggilan AI bergantung pada pola BYOK (Bring Your Own Key) di mana pengguna memasukkan API key pribadi yang disimpan terenkripsi di tabel `gemini_api_keys`. Selain itu, modul `ai_usecase.go` memuat hardcoded provider map ke berbagai vendor AI eksternal. Pola ini memiliki kelemahan kritis:
1. **Ketidakmampuan Monetisasi**: Penggunaan token tidak dapat diatur atau dijual dalam model SaaS langganan.
2. **Kerapuhan Ketersediaan (Single Point of Failure)**: Jika kunci user habis limit atau vendor eksternal mengalami gangguan, sistem langsung mengembalikan error tanpa mekanisme failover.
3. **Fragmentasi Format & Modality**: Kode harus memelihara integrasi terpisah untuk format Gemini REST, OpenAI completions, Anthropic messages, dan Whisper audio.
4. **Resiko Inkompatibilitas Embedding**: Model embedding yang tidak seragam berpotensi merusak indeks pgvector 768d.

---

## 2. Arsitektur Komponen

AI Gateway Terpusat memisahkan layer bisnis perbankan/keuangan dari layer komunikasi kecerdasan buatan melalui perantara gateway yang transparan:

```
[ Frontend: Chat, Vision, Voice, Analytics ]
                     │
                     ▼
[ Backend HTTP Handler (Session, Auth, JWT) ]
                     │
                     ▼
[ AIUsecase (Financial Context Injection, RAG, Sanitasi) ]
                     │ (Adapter Tipis via Interface)
                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     INTERNAL AI GATEWAY MODULE                          │
│                                                                         │
│  ┌────────────────┐     ┌────────────────┐     ┌─────────────────────┐  │
│  │ Routing Engine │────►│ Circuit Breaker│────►│ Failover Controller │  │
│  └───────┬────────┘     └────────────────┘     └──────────┬──────────┘  │
│          │                                                │             │
│          ▼                                                ▼             │
│  ┌────────────────┐                            ┌─────────────────────┐  │
│  │ In-Memory Route│                            │ OpenAI-Compatible   │  │
│  │ & Config Cache │                            │ Client Engine       │  │
│  └────────────────┘                            └──────────┬──────────┘  │
│                                                           │             │
│  ┌────────────────┐     ┌────────────────┐                │             │
│  │ Health Checker │     │ Metering Token │◄───────────────┘             │
│  │ Background Job │     │ & Estimator    │                              │
│  └────────────────┘     └────────────────┘                              │
└───────────────────────────────────┬─────────────────────────────────────┘
                                    │
                                    ├──► [ 1. Primary: 9router Lokal (Port 20128) ]
                                    ├──► [ 2. Secondary: Direct Cloud Fallback A  ]
                                    └──► [ 3. Tertiary: Direct Cloud Fallback B   ]
```

---

## 3. Komponen Utama

### 3.1. `AIGatewayService` (Facade)
Facade utama yang diakses oleh `AIUsecase`. Menyediakan metode:
- `ChatStream(ctx context.Context, req *domain.AIChatRequest) (<-chan domain.AIChatStreamChunk, error)`
- `ChatSync(ctx context.Context, req *domain.AIChatRequest) (*domain.AIChatResponse, error)`
- `CreateEmbedding(ctx context.Context, text string) ([]float32, *domain.TokenUsage, error)`
- `AnalyzeVision(ctx context.Context, req *domain.AIVisionRequest) (string, *domain.TokenUsage, error)`
- `TranscribeAudio(ctx context.Context, req *domain.AIAudioRequest) (string, *domain.TokenUsage, error)`

### 3.2. Dynamic Routing Engine
- Menerima `use_case` dan preferensi profil model (contoh: `use_case = "chat"`).
- Mencari rute aktif di cache in-memory yang dipetakan dari tabel `ai_routes`.
- Mengembalikan daftar rute yang telah diurutkan berdasarkan `priority` terendah ke tertinggi.
- Memeriksa status Kill Switch di level use case maupun endpoint.

### 3.3. Circuit Breaker In-Memory
- Melindungi upstream endpoint dari cascade failure.
- Menjaga state per endpoint:
  - `Closed`: Normal, semua trafik diizinkan.
  - `Open`: Upstream sedang bermasalah; trafik ditolak langsung untuk failover instan.
  - `Half-Open`: Periode uji coba setelah masa cooldown 30 detik.

### 3.4. Failover Controller
- Mengeksekusi pemanggilan ke endpoint urutan pertama.
- Jika terjadi kegagalan (HTTP 5xx, network drop, timeout, rate limit 429):
  1. Melaporkan kegagalan ke Circuit Breaker.
  2. Beralih secara transparan ke endpoint urutan kedua.
  3. Mengulangi proses hingga sukses atau batas retry tercapai.

### 3.5. OpenAI-Compatible Client
- Menstandarkan seluruh komunikasi upstream ke spesifikasi OpenAI REST API.
- 9router lokal bertindak sebagai aggregator yang menerjemahkan model non-OpenAI ke format OpenAI.
- Mengelola HTTP transport connection pool, keep-alive, streaming chunk reader via SSE (`bufio.Reader`).

### 3.6. Token Metering & Heuristic Estimator
- Menghitung konsumsi token input/output.
- Mengambil data eksak dari field `usage` pada respon model.
- Mengaktifkan heuristic tokenizer jika upstream tidak mengembalikan data usage (terutama pada streaming chunk terputus).

---

## 4. Aliran Eksekusi (Sequence Flow)

```mermaid
sequenceDiagram
    participant User as Frontend Client
    participant UC as AIUsecase
    participant GW as AIGateway
    participant CB as CircuitBreaker
    participant Prim as 9router (Primary)
    participant Sec as Cloud (Secondary)

    User->>UC: Kirim Pesan Chat
    UC->>UC: Kumpulkan Konteks Finansial (RAG)
    UC->>GW: ChatStream(req, use_case="chat")
    GW->>CB: Cek Status Endpoint 9router
    CB-->>GW: Status CLOSED (Sehat)
    GW->>Prim: POST /v1/chat/completions (Stream)
    alt Primary Berhasil
        Prim-->>GW: SSE Stream Chunks
        GW-->>UC-->>User: Stream Delta Text
    else Primary Gagal (503 / Timeout)
        Prim-->>GW: HTTP Error / Timeout
        GW->>CB: Laporkan Kegagalan (Record Failure)
        GW->>Sec: Failover: POST /v1/chat/completions (Stream)
        Sec-->>GW: SSE Stream Chunks
        GW-->>UC-->>User: Stream Delta Text
    end
```

---

## 5. Standar Kesalahan & Resiliensi
1. **Graceful Degradation**: Jika seluruh rute mengalami kegagalan, gateway mengembalikan respons terformat yang informatif tanpa memutus koneksi aplikasi:
   *"Maaf, asisten FinTrack sedang mengalami antrian pemrosesan yang tinggi. Silakan ulangi beberapa saat lagi."*
2. **Isolasi Kegagalan**: Kegagalan pada modality `voice` atau `vision` tidak boleh mempengaruhi ketersediaan modality `chat` teks atau kalkulasi `embedding` transaksi.
