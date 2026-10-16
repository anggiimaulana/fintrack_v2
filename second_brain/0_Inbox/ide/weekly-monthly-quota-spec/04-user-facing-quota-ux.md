# User-Facing Quota Display & UX

Dokumen ini mendefinisikan antarmuka pengguna untuk menampilkan status kuota dwi-siklus, termasuk progress bars, countdown, badge, toast, dan halaman usage.

---

## 1. Komponen Utama

### 1.1. `DualQuotaCard` (BARU)

Menggantikan/memperluas `QuotaUsageCard` yang sudah ada. Menampilkan kuota per metrik dengan dual progress bar.

**Lokasi**: `frontend/components/billing/DualQuotaCard.tsx`

```
┌─────────────────────────────────────────────────────────────┐
│  Kuota Anda                                                 │
│  Siklus: 1 Okt 14:30 — 31 Okt 14:30                       │
│                                                             │
│  ┌─ Kuota AI ─────────────────────────────────────────────┐ │
│  │  ████████████░░░░░░░░░░░░░░░░░░  42.000 / 150.000     │ │
│  │  Bulan ini                        (28% terpakai)       │ │
│  │                                                        │ │
│  │  ██████████████████████████░░░░  12.000 / 50.000       │ │
│  │  Minggu ini                       (24% terpakai)       │ │
│  │                                                        │ │
│  │  Sisa efektif: 38.000 kuota                            │ │
│  │  ⏱ Reset mingguan: 3h 12m  │  Reset bulanan: 12d 5h   │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─ Transaksi ────────────────────────────────────────────┐ │
│  │  ██████████████████████████████████████░░  720 / 1.000 │ │
│  │  Bulan ini                        (72% terpakai)       │ │
│  │                                                        │ │
│  │  █████████████████████░░░░░░░░░  140 / 200             │ │
│  │  Minggu ini                       (70% terpakai)       │ │
│  │                                                        │ │
│  │  ⚠ Sisa efektif: 60 transaksi                         │ │
│  │  ⏱ Reset mingguan: 3h 12m  │  Reset bulanan: 12d 5h   │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─ Permintaan AI ────────────────────────────────────────┐ │
│  │  ███░░░░░░░░░░░░░░░░░░░░░░░░░░  150 / 2.000           │ │
│  │  Bulan ini                        (8% terpakai)        │ │
│  │  (Hanya siklus bulanan)                                │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 1.2. Visual Design Rules

| Elemen | Warna/Style | Kondisi |
|--------|------------|---------|
| Progress bar monthly | `bg-primary` (biru) | Normal |
| Progress bar weekly | `bg-chart-2` (hijau/teal) | Normal |
| Progress bar | `bg-yellow-500` | ≥ threshold (warning) |
| Progress bar | `bg-destructive` (merah) | ≥ 95% atau blocked |
| Badge "Normal" | `bg-green-100 text-green-800` | Status normal |
| Badge "Peringatan" | `bg-yellow-100 text-yellow-800` | ≥ threshold |
| Badge "Degraded" | `bg-orange-100 text-orange-800` | Weekly habis, degraded aktif |
| Badge "Terblokir" | `bg-red-100 text-red-800` | Monthly habis atau degraded cap |
| Countdown timer | `text-muted-foreground text-xs` | Normal |
| Countdown timer | `text-destructive font-bold` | < 1 jam |
| Sisa efektif | `text-lg font-bold` | Selalu |

### 1.3. Responsivitas

- **Desktop**: progress bar side-by-side (monthly di atas, weekly di bawah).
- **Mobile (<768px)**: stack vertikal, progress bar full-width.
- Countdown timer: di desktop tampil inline sejajar; di mobile di bawah progress bar.

---

## 2. Status Badge per Metrik

### 2.1. Komponen `QuotaStatusBadge`

**Lokasi**: `frontend/components/billing/QuotaStatusBadge.tsx`

```typescript
type QuotaStatus = "normal" | "warning" | "degraded_throttle" | "degraded_downgrade" | "blocked";

interface QuotaStatusBadgeProps {
  status: QuotaStatus;
}
```

### 2.2. Label Badge

| Status | Label | Ikon SVG |
|--------|-------|----------|
| `normal` | Normal | `CheckCircle2` |
| `warning` | Peringatan | `AlertTriangle` |
| `degraded_throttle` | Dibatasi | `Gauge` |
| `degraded_downgrade` | Model Diturunkan | `ArrowDownCircle` |
| `blocked` | Terblokir | `XCircle` |

### 2.3. Tooltip Detail

Saat badge di-hover, tampilkan tooltip:

```
Status: Dibatasi (throttle)
Kuota mingguan habis. Dibatasi ke 5 permintaan per 60 detik.
Penggunaan degraded: 3 / 10.000 maks.
Reset mingguan: Kamis 14:30 WIB.
```

---

## 3. Countdown Timer

### 3.1. Komponen `QuotaCountdown`

**Lokasi**: `frontend/components/billing/QuotaCountdown.tsx`

Menggunakan pattern `useSyncExternalStore` yang sudah ada di `LifecycleCountdown.tsx` — clock store yang sama, hanya formatting berbeda.

```typescript
interface QuotaCountdownProps {
  weeklyResetAt?: string;   // RFC3339, null jika monthly_only
  monthlyResetAt: string;   // RFC3339
  className?: string;
}
```

### 3.2. Format Tampilan

| Kondisi | Format |
|---------|--------|
| > 1 hari | "3 hari 5 jam" |
| 1-24 jam | "5 jam 32 menit" |
| < 1 jam | "42 menit" (warna merah, bold) |
| Sudah lewat | "Mereset..." (sementara lazy reset belum terjadi) |

### 3.3. Dual Countdown

```
⏱ Mingguan: 3h 12m  •  Bulanan: 12d 5h
```

Jika hanya monthly_only:
```
⏱ Reset: 12d 5h
```

---

## 4. Toast Notifications

### 4.1. Trigger Notifikasi

| Event | Toast Type | Pesan |
|-------|-----------|-------|
| Usage ≥ threshold% (weekly) | Warning | "Kuota mingguan {metrik} tinggal {remaining}. Reset {countdown}." |
| Usage ≥ threshold% (monthly) | Warning | "Kuota bulanan {metrik} tinggal {remaining}. Reset {countdown}." |
| Weekly habis → degraded aktif | Info | "Kuota mingguan {metrik} habis. Mode: {degraded_mode}." |
| Weekly habis → hard block | Error | "Kuota mingguan {metrik} habis. Fitur diblokir sampai {reset_at}." |
| Monthly habis | Error | "Kuota bulanan {metrik} habis. Tingkatkan paket untuk melanjutkan." |
| Quota reset (weekly) | Success | "Kuota mingguan telah direset." |
| Quota reset (monthly) | Success | "Kuota bulanan telah direset. Selamat menikmati siklus baru!" |

### 4.2. Deduplikasi Toast

- Toast per metrik per event per window: tidak menampilkan toast yang sama dua kali untuk metrik+event+window yang sama.
- Gunakan `sessionStorage` key: `quota_toast_{limitKey}_{eventType}_{cycleStart}`.

### 4.3. Implementasi

Toast dipicu oleh SSE event handler di `realtime-dispatcher.ts`. Saat event `usage_changed` atau `quota_reset` diterima:

1. Refetch `GET /api/v1/me/quota`.
2. Bandingkan status baru dengan status lama (cached di React Query).
3. Jika status berubah → tampilkan toast sesuai tabel §4.1.

---

## 5. Halaman `/user/usage` (DIPERLUAS)

### 5.1. Section Baru: Quota Overview

Di bagian atas halaman (sebelum chart), tambahkan `DualQuotaCard`:

```
┌─────────────────────────────────────────────────────┐
│  [DualQuotaCard - lihat §1.1]                       │
└─────────────────────────────────────────────────────┘

[UsageChart - existing]

[DataTable - existing]
```

### 5.2. Filter "Siklus" di Tabel History

Tambahkan filter chip:

```
[Semua]  [Mingguan]  [Bulanan]
```

Filter ini mem-filter `usage_counters` berdasarkan `cycle_type`.

---

## 6. Halaman `/user/subscription` (DIPERLUAS)

### 6.1. Info Kuota di Card Plan

Di card subscription yang sudah ada, tambahkan ringkasan:

```
┌─────────────────────────────────────────────┐
│  Pro Plan — Rp39.000/bulan                  │
│  Status: Aktif                              │
│                                             │
│  Siklus bulanan: 1 Okt — 31 Okt            │
│  Siklus mingguan: 16 Okt — 23 Okt          │  ← BARU
│                                             │
│  Fitur mingguan aktif:                      │  ← BARU
│  • Kuota AI: 50.000/minggu                  │
│  • Transaksi: 200/minggu                    │
│  • OCR: 30/minggu                           │
│                                             │
│  [Lihat Detail Penggunaan]                  │
│  [Tingkatkan Paket]                         │
└─────────────────────────────────────────────┘
```

---

## 7. Komponen React: Struktur File

```
frontend/components/billing/
├── DualQuotaCard.tsx           — Card utama dual progress bars
├── DualQuotaBar.tsx            — Satu metrik: monthly bar + weekly bar
├── QuotaStatusBadge.tsx        — Badge status per metrik
├── QuotaCountdown.tsx          — Countdown timer (weekly + monthly)
├── QuotaUsageCard.tsx          — EXISTING, tetap sebagai fallback
└── LifecycleCountdown.tsx      — EXISTING, tidak diubah

frontend/hooks/
├── use-entitlements.ts         — EXISTING, types diperluas
├── use-usage.ts                — EXISTING
└── use-quota.ts                — BARU: hook untuk GET /api/v1/me/quota
```

### 7.1. `useQuota` Hook (BARU)

```typescript
interface QuotaSnapshot {
  plan_code: string;
  weekly_anchor: string | null;
  monthly_period: { start: string; end: string };
  quotas: Record<string, MetricQuota>;
  warning_threshold: number;
}

interface MetricQuota {
  label: string;
  unit: string;
  icon: string;
  monthly: { used: number; limit: number; remaining: number; unlimited: boolean };
  weekly: { used: number; limit: number; remaining: number; unlimited: boolean } | null;
  effective_remaining: number;
  status: QuotaStatus;
  degraded_mode: string | null;
  degraded_config: DegradedConfig | null;
  weekly_resets_at: string | null;
  monthly_resets_at: string;
}

export function useQuota() {
  return useQuery({
    queryKey: queryKeys.quota.snapshot,
    queryFn: () => api.get("/me/quota").then(r => r.data.data),
    staleTime: staleTimeConfig.quota,    // 30s
    refetchInterval: 60_000,             // 60s polling sebagai fallback SSE
  });
}
```

---

## 8. Aksesibilitas (A11y)

### 8.1. Progress Bars

- `role="progressbar"` dengan `aria-valuenow`, `aria-valuemin=0`, `aria-valuemax`.
- `aria-label`: "Kuota AI bulanan: 42.000 dari 150.000 terpakai (28%)".
- Screen reader: angka exact, bukan visual bar.

### 8.2. Status Badges

- `aria-live="polite"` pada container badge — perubahan status diumumkan.
- Ikon SVG: `aria-hidden="true"`, label teks menyampaikan makna.

### 8.3. Countdown Timers

- `aria-live="off"` — countdown yang terus berubah tidak perlu diumumkan setiap menit.
- `aria-label` statis: "Reset mingguan dalam 3 jam 12 menit".
- Saat < 1 jam: `aria-live="polite"` aktif, perubahan diumumkan.

### 8.4. Toast Notifications

- `role="alert"` (existing pattern dari `sonner`).
- Pesan teks lengkap, tidak bergantung pada warna.

---

## 9. Dark Mode & Neobrutalist Theme

Seluruh komponen baru mengikuti design system FinTrack v2:
- Border: `border-2 border-foreground rounded-md`.
- Shadow: `shadow-[4px_4px_0px_0px] shadow-foreground`.
- Progress bars: rounded corners, border-2.
- Warna adaptive: `text-foreground`, `bg-background`, semantic colors dari CSS variables.
- Dark mode: semua warna dari variabel CSS, zero hardcode `#hex`.
- Ikon: SVG dari `lucide-react`, zero emoji.
