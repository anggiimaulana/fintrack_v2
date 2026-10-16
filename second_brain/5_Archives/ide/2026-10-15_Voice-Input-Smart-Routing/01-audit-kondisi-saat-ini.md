# 01 — Audit Kondisi Saat Ini

| Field | Isi |
|---|---|
| **Dokumen** | Audit Voice: kondisi terkini, akar masalah, inventaris dead code |
| **Tanggal** | 2026-10-15 |
| **Metode** | Pembacaan kode statis (`grep`/`read`) + penelusuran alur |
| **Status** | Selesai |

---

## 1. Peta Alur Voice Saat Ini

```
┌─────────────────────────────────────────────────────────────────────────┐
│ FRONTEND                                                                 │
│                                                                          │
│  components/ai/VoiceInput.tsx   ──► SpeechRecognition (Web Speech API)   │
│         ▲                                  │                             │
│         │ onTranscript(text)               │ audio → vendor browser      │
│         │                                  ▼                             │
│  ChatInput.tsx / InlineAiAssistant.tsx   transcript (teks)               │
│         │                                  │                             │
│         └──────────────► POST /ai/chat (teks)                            │
│                                                                          │
│  app/(dashboard)/user/voice/page.tsx ──► SpeechRecognition (idem)        │
│         │                                  │                             │
│         └──────────────► POST /ai/chat (teks)                            │
│                                                                          │
│  ❌ TIDAK ADA pemanggil POST /ai/voice                                   │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│ BACKEND (siap, tetapi tidak terpakai)                                    │
│                                                                          │
│  POST /api/v1/ai/voice                                                   │
│    ├─ featureGate("voice_input")      (router.go:345)                    │
│    ├─ limitGate("ai_requests_per_cycle")                                 │
│    ├─ limitGate("ai_tokens_per_cycle")                                   │
│    ├─ ledger.Reserve(ModalityVoice)   (ai_handler.go:490)                │
│    ├─ aiUC.ProcessVoice → gateway.TranscribeAudio                        │
│    ├─ settleVoice → Commit / Release  (ai_handler.go:520)                │
│    └─ emitUsageChanged("ai_voice")    (ai_handler.go:509)                │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Bukti Kode

### 2.1 `VoiceInput.tsx` — Mode Browser (tidak menyentuh backend)

```ts
// frontend/components/ai/VoiceInput.tsx
const recognition = new SpeechRecognition();   // ← Web Speech API browser
recognition.lang = "id-ID";
recognition.continuous = false;
recognition.interimResults = true;
recognition.maxAlternatives = 1;
...
recognition.onresult = (event) => {
  ...
  if (finalTranscript) {
    onTranscript(finalTranscript);             // ← hanya teks yang keluar
  }
};
```

**Fakta:**
- Tidak ada `fetch`/`api.post` di file ini → audio tidak pernah ke server.
- `onTranscript(text)` mengirim **teks** ke parent.
- Dukungan: hanya `SpeechRecognition`/`webkitSpeechRecognition` (Chrome, Edge, Safari).

### 2.2 `/ai/voice` — Mode Server (siap, tidak dipanggil)

```go
// backend/internal/delivery/http/handler/ai_handler.go:473
func (h *AIHandler) ProcessVoice(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    var req struct{ Audio string `json:"audio" binding:"required"` }
    ...
    if len(req.Audio) > 5*1024*1024 {           // batas 5MB (base64)
        response.BadRequest(c, "Audio too large. Max 5MB.", "file_too_large")
        return
    }
    // reserve → ProcessVoice → settle(commit/release)
}
```

```go
// backend/internal/delivery/http/router.go:344
ai.POST("/voice", gated(aiH.ProcessVoice,
    featureGate(config, "voice_input"),
    limitGate(config, "ai_requests_per_cycle"),
    limitGate(config, "ai_tokens_per_cycle"))...)
```

**Fakta:**
- Gate fitur + limit **sudah** terpasang.
- Ledger reserve/commit/release **sudah** benar.
- `emitUsageChanged` **sudah** dipanggil → kuota real-time sudah siap.
- **Yang hilang: pemanggil di frontend.**

### 2.3 Grep pembuktian dead code

| Pencarian | Hasil |
|---|---|
| `grep -rn "ai/voice" frontend/` | **0 hasil** |
| `grep -rn "MediaRecorder\|getUserMedia" frontend/` | **0 hasil** |
| `<VoiceInput` di `transactions/page.tsx` | **0 pemakaian** (hanya import) |
| Pemakai `VoiceInput` | `ChatInput.tsx:40`, `InlineAiAssistant.tsx:507` |

### 2.4 Halaman `/user/voice`

```tsx
// frontend/app/(dashboard)/user/voice/page.tsx:75
const SpeechRecognition = (window as WindowWithSpeech).SpeechRecognition
  || (window as WindowWithSpeech).webkitSpeechRecognition;
...
// lalu hasil transcript dikirim ke /ai/chat (TEKS), bukan /ai/voice
const chatMutation = useMutation({
  mutationFn: async (text: string) => {
    const res = await api.post("/ai/chat", { message: text });   // ← teks
    return res.data;
  },
});
```

**Fakta:** halaman "Input Suara" ini pun **tidak** memakai endpoint voice server.

---

## 3. Akar Masalah

1. **Dua konsep "voice" yang tertukar.** Yang diimplementasikan adalah *speech-to-text di
   browser* (aksesibilitas input), sedangkan yang dijual di paket (`voice_input`) adalah
   *transkripsi server* (AI Gateway, berbayar token). Keduanya tidak dihubungkan.
2. **Tidak ada perekam audio.** Untuk mengirim ke `/ai/voice` dibutuhkan `MediaRecorder`
   + `getUserMedia` (merekam audio nyata → base64). Ini belum ada sama sekali.
3. **Tidak ada pemilih mode.** Tidak ada logika yang memeriksa entitlement untuk memutuskan
   apakah memakai browser atau server.
4. **Tidak ada fallback.** Bila nanti dipaksa ke server, plan/kuota/browser bisa membuat
   fitur mati total (dead-end).

---

## 4. Inventaris Komponen yang Relevan

| Berkas | Peran | Status |
|---|---|---|
| `frontend/components/ai/VoiceInput.tsx` | Komponen mic (mode browser) | ✅ jalan, ❌ tidak sadar entitlement |
| `frontend/app/(dashboard)/user/voice/page.tsx` | Halaman input suara | ✅ jalan, ❌ pakai browser saja |
| `frontend/components/ai/ChatInput.tsx` | Pemakai `VoiceInput` (chat) | ✅ |
| `frontend/components/ai/InlineAiAssistant.tsx` | Pemakai `VoiceInput` (inline) | ✅ |
| `frontend/hooks/use-entitlements.ts` | Sumber `features`/`usage` real-time | ✅ siap dipakai |
| `frontend/lib/realtime-dispatcher.ts` | Invalidate `entitlements.me` saat `usage_changed` | ✅ (2026-10-15) |
| `backend/.../handler/ai_handler.go` | `ProcessVoice` | ✅ siap, ❌ tidak dipanggil |
| `backend/.../usecase/ai_usecase.go` | `ProcessVoice` → gateway | ✅ siap |
| `backend/.../aigateway/gateway/service.go` | `TranscribeAudio` | ✅ siap |
| `backend/internal/entitlement/catalog.go` | Feature key `voice_input` | ✅ terdaftar |
| `backend/internal/admin/settings.go` | Allowlist `writableSettings` (dinamis) | ✅ ada, ❌ belum ada key `voice_*` |
| `backend/internal/runtimeconfig/resolver.go` | Resolver DB-over-env (dinamis) | ✅ siap dipakai |
| `backend/internal/featureflag/*` | Feature flag (kill switch) | ✅ siap dipakai |

---

## 5. Temuan Gap (akibat keputusan produk)

| ID | Gap | Bukti | Dampak |
|---|---|---|---|
| **VOI-07** | Tidak ada konfigurasi dinamis voice (durasi maks, bahasa) | `writableSettings` tidak memuat `voice_*` | Super Admin tidak bisa mengatur (Q-01, Q-06) |
| **VOI-08** | Tidak ada limit **harian** voice; `plan_limits` hanya per-siklus | `plan_limits` seed: hanya `*_per_cycle` | Mode browser tak terbatas (Q-03) |
| **VOI-09** | Tidak ada riwayat transkripsi di server | tidak ada tabel/endpoint voice history | Riwayat hilang saat ganti perangkat (Q-04) |
| **VOI-10** | Tidak ada VU meter | grep `AnalyserNode` = 0 | UX rekaman buruk (Q-05) |

---

## 6. Kesimpulan Audit

> Backend voice **lengkap untuk jalur server** (gate, ledger, realtime), tetapi **belum punya**:
> (a) konfigurasi dinamis (durasi/bahasa) — **VOI-07**,
> (b) limit harian per paket — **VOI-08**,
> (c) riwayat transkripsi server — **VOI-09**.
>
> Di sisi klien, yang hilang: (d) perekam audio (`MediaRecorder`), (e) pemilih mode
> berbasis entitlement, (f) fallback berlapis, (g) indikator mode, (h) VU meter — **VOI-10**.
>
> **Karena keputusan produk (Q-01/Q-03/Q-06), desain ini kini menyentuh backend** (bukan
> lagi frontend-only). Semua penambahan bersifat **aditif** dan memanfaatkan infrastruktur
> dinamis yang sudah ada (`writableSettings`, `runtimeconfig`, `plan_limits`).
> Rincian di dokumen 03–06.
