# 00 — AI Chat: Kuota Token, Kebocoran Payload, Sesi Ganda, & Layout

| Field | Isi |
|---|---|
| **Dokumen** | Audit & perbaikan AI Chat (kuota token, render, sesi, layout) |
| **Tanggal** | 2026-10-15 |
| **Status** | Temuan + perbaikan (kode) |
| **Referensi** | `2026-10-14_Audit-Kuota-Biaya-Dinamis/00-audit-kuota-biaya.md` |

---

## Ringkasan Eksekutif

Laporan user: "chat sekali di akun Free langsung habis 5k token", "1 chat jadi 2 sesi",
"response menampilkan JSON/toolcalls", "limit error bocor kode mentah", "sidebar chat terpotong".

| ID | Temuan | Severity | Status |
|---|---|---|---|
| **AI-TOK-01** | Reserve token **flat 2000** padahal 1 chat nyata ±5.000–10.000 token → 1 chat menguras seluruh kuota Free (5.000) | 🔴 Kritis | ✅ **Diperbaiki** |
| **AI-TOK-02** | Commit ter-clamp ke sisa kuota **tanpa sinyal apa pun** ke user | 🟠 Tinggi | ✅ **Diperbaiki** (indikator UI) |
| **AI-LEAK-01** | Model kadang meng-echo payload internal (```json `{"chart":…}`) → tampil mentah ke user | 🟠 Tinggi | ✅ **Diperbaiki** |
| **AI-ERR-01** | `limit_exceeded` (402) dari middleware tidak ditangani UI → bubble "…limit_exceeded" | 🟠 Tinggi | ✅ **Diperbaiki** |
| **AI-SES-01** | Dua jalur pembuatan sesi (handleSend + efek `?q=` dgn `setTimeout 300ms`) tanpa guard → 2 sesi | 🟠 Tinggi | ✅ **Diperbaiki** |
| **AI-UI-01** | Sidebar sesi terpotong + scroll horizontal (`min-width:auto` + `truncate`) | 🟡 Sedang | ✅ **Diperbaiki** |
| **AI-ERR-02** | `InlineAiAssistant` menelan error (`catch {}`) tanpa feedback | 🟡 Sedang | ✅ **Diperbaiki** |

---

## 1. AI-TOK-01 — 1 chat menguras seluruh kuota Free (KRITIS)

### Bukti (laporan scout + audit live)

| Faktor | Nilai |
|---|---|
| Kuota Free (`ai_tokens_per_cycle`) | **5.000 token/siklus** (`seeds/0002_subscription_token_seeds.sql:69`) |
| Reserve chat (flat) | **2.000** (`billing/estimate.go:26`) |
| Pemakaian nyata 1 chat | **±5.000–10.000 token** (audit live: 1 chat = **9.945** token) |
| Commit | charge token asli, **di-clamp** ke `quota - used` (`ledger.go:166-177`) |

### Akar masalah

Reserve memakai konstanta flat per modalitas, **tidak memperhitungkan panjang prompt**:

```go
// billing/estimate.go — SEBELUM
func DefaultQuotaEstimate() QuotaEstimate {
    return QuotaEstimate{Chat: 2000, Reasoning: 4000, Vision: 5000, ...}
}
```

Padahal setiap chat membawa beban tetap besar: system prompt + tools prompt +
konteks pengguna + RAG + **riwayat 20 pesan terakhir** (`ai_gateway_adapter.go:150`),
ditambah agent loop maksimum 2 iterasi.

Karena commit meng-clamp ke sisa kuota, satu chat sukses = `used_tokens = 5000`,
`available = 0`. Permintaan berikutnya ditolak **402 `limit_exceeded`** — user tidak
pernah diberi tahu pada saat kejadian.

### Perbaikan

Estimasi kini **proporsional terhadap request** (`billing/estimate.go`):

```go
const (
    chatOverheadTokens       = 3000  // system prompt + tools + RAG + 20 riwayat
    chatOutputHeadroomTokens = 800
    charsPerToken            = 4     // sengaja konservatif → reserve errs high
)

func (s *EstimateStore) ForChat(ctx context.Context, modality Modality, promptChars int64) int64 {
    base := s.For(ctx, modality)          // ai_estimate_chat tetap jadi LANTAI
    estimated := chatOverheadTokens + (promptChars/charsPerToken) + chatOutputHeadroomTokens
    if estimated < base { return base }
    return estimated
}
```

Handler memanggil `chatEstimate(ctx, modality, req.Message)` (`ai_handler.go:118`).
Nilai `ai_estimate_chat` tetap berfungsi sebagai **lantai**, jadi Super Admin masih
bisa menaikkan hold minimum dari `settings`.

**Efek:** reserve sekarang mendekati biaya nyata → tidak ada lagi kejutan clamp yang
menguras siklus dalam satu chat.

> Catatan: kuota Free 5.000 token memang kecil (satu chat tipikal ~4.000–6.000 token).
> Menaikkan `ai_tokens_per_cycle` Free adalah keputusan produk, bukan bug — lihat
> rekomendasi di bagian 7.

---

## 2. AI-TOK-02 — Commit ter-clamp senyap

`Commit` memotong tagihan ke sisa kuota dan hanya menulis log server
(`ledger.go:167-175`). User tidak melihat apa pun sampai request berikutnya gagal.

**Perbaikan (UI):** `ChatWindow` kini membaca `usage["ai_tokens_per_cycle"]` dari
`GET /me/entitlements` dan menampilkan:
- badge sisa token di header (kuning saat ≤20%, merah saat habis),
- banner "Kuota token AI siklus ini sudah habis + tautan tingkatkan paket" di atas input.

---

## 3. AI-LEAK-01 — Payload internal tampil mentah

### Bukti (screenshot user)

Bubble assistant menampilkan blok JSON (`{"chart": {"type":"bar", …}}`) lalu baris
`→ skipped: advanced chart styling, add when UI framework requires theme integration.`

Filter lama hanya membersihkan tag `<TOOL_CALL>`/`<TOOL_RESULT>`
(`ChatMessage.tsx:39-45`, `ai_gateway_adapter.go:69-99`). Model kadang meng-echo
payload tanpa tag — dan `buildToolResultPrompt` (`ai_usecase.go:734-750`)
memang mengirim JSON hasil tool kembali ke LLM, sehingga besar peluang disalin.

### Perbaikan

Modul baru `frontend/lib/chat-sanitize.ts` (`sanitizeAssistantContent`) menghapus:
- tag `<TOOL_CALL>`/`<TOOL_RESULT>` (lengkap & parsial),
- blok ` ```json … ``` `,
- blok fenced tanpa tag yang isinya murni JSON,
- objek JSON mandiri ber-key internal (`chart`, `chart_type`, `datasets`, `tool`, …),
- komentar internal (`→ skipped: …`).

Dipakai di `ChatMessage` **hanya untuk pesan assistant** (pesan user tidak diubah).
Jika hasil akhir kosong, tampil kalimat ramah alih-alih bubble kosong.
8 unit test (`lib/chat-sanitize.test.ts`).

---

## 4. AI-ERR-01 — Kode error mentah bocor ke user

Screenshot user: bubble **"Maaf, terjadi kesalahan. Silakan coba lagi. limit_exceeded"**.

`ChatWindow` hanya menangani `insufficient_quota`/`quota_exceeded`; gate middleware
mengembalikan **402 `limit_exceeded`** / `limit_not_in_plan` / `feature_not_in_plan`
(`middleware/entitlement.go:133-168`) sehingga jatuh ke cabang generik.

### Perbaikan

- `ChatWindow.tsx`: **semua** kode gate diteruskan ke `notifyEntitlementError` →
  toast upsell; kode mentah tidak lagi ditempel ke pesan.
- `lib/entitlement-errors.ts`: pesan kini **spesifik per limit** dan menyebut angka
  pemakaian, mis. *"Kuota token AI Anda untuk siklus ini sudah habis — Terpakai 5.000
  dari 5.000"*.
- `InlineAiAssistant.tsx`: `catch {}` → toast (gate → upsell, lainnya → pesan singkat).

---

## 5. AI-SES-01 — Satu pesan menjadi dua sesi

### Bukti (screenshot user)

Sidebar menampilkan **dua sesi dengan judul identik** ("Buat grafik pengeluaran per kate…").

### Akar masalah

Dua jalur membuat sesi tanpa saling tahu:

```tsx
// ChatWindow.tsx — SEBELUM (efek auto-send)
createSession.mutateAsync(msg.substring(0, 50)).then((newSession) => {
  setTimeout(() => handleSend(msg, newSession.id), 300);  // ← jendela race 300ms
});
```

Selama jendela 300 ms itu, `handleSend` juga bisa menempuh cabang pembuatan sesinya
sendiri (`ChatWindow.tsx:194`), dan `handleNewSession` tidak punya guard in-flight —
`disabled={createSession.isPending}` baru aktif setelah re-render. Backend juga
melakukan `INSERT` polos tanpa idempotency (`chat_session_repo.go:34`).

### Perbaikan

- Satu helper `ensureSession()` dengan **in-flight guard** (`sessionCreateRef`):
  pemanggil bersamaan menunggu promise yang sama → tepat satu `POST /ai/sessions`.
- `handleSend` memakai `ensureSession`; efek `?q=` memanggil `handleSend` langsung
  (**`setTimeout` 300 ms dihapus**).
- `handleNewSession` early-return saat `createSession.isPending` (anti double-click).

---

## 6. AI-UI-01 — Sidebar terpotong & scroll horizontal

`<button className="flex-1 … truncate">` adalah flex item dengan `min-width:auto`,
sehingga tidak boleh menyusut di bawah lebar kontennya. Akibatnya baris melebihi
`w-60`, muncul scrollbar horizontal, dan judul tampak terpotong.

**Perbaikan:** `min-w-0` pada tombol baris, `overflow-x-hidden` pada kontainer daftar,
`min-w-0` pada wrapper baris, rail desktop dilebarkan `w-60 → w-64`, dan rail diberi
`min-w-0`. Diterapkan pada rail desktop **dan** overlay mobile.

---

## 7. Rekomendasi lanjutan (belum dikerjakan)

1. **Kuota Free 5.000 token terlalu kecil** untuk satu chat tipikal (±4.000–6.000).
   Pertimbangkan menaikkan `ai_tokens_per_cycle` Free atau memberi kuota chat terpisah.
2. **Event `quota_clamped`** dari backend saat commit ter-potong, agar UI bisa
   memperingatkan tepat pada momennya (kini UI hanya mencerminkan `usage` setelahnya).
3. **Idempotency key pada `POST /ai/sessions`** sebagai sabuk pengaman server-side
   (kini dedupe hanya di klien).
4. **Satukan kode "token habis"** (402 `limit_exceeded` vs 429 `insufficient_quota`)
   agar klien tidak perlu menangani dua bentuk.

---

## 8. Verifikasi

| Perintah | Hasil |
|---|---|
| `go vet ./...` | ✅ bersih |
| `go build ./...` | ✅ bersih |
| `go test ./internal/billing/... ./internal/delivery/... ./internal/usecase/...` | ✅ ok |
| `pnpm lint` | ✅ 46 problems (1 error pre-existing, tanpa regresi) |
| `pnpm test` | ✅ **289 passed (38 files)** — naik dari 271 |
| `pnpm build` | ✅ sukses |
