# Admin Configuration UI — Super Admin Plan Matrix

Dokumen ini mendefinisikan antarmuka konfigurasi kuota mingguan + bulanan untuk Super Admin, termasuk layout, interaksi, validasi form, dan komponen.

---

## 1. Lokasi & Navigasi

### 1.1. Route

Konfigurasi cycle terintegrasi ke halaman plan yang sudah ada:

```
/super-admin/plans          → daftar plan (existing)
/super-admin/plans          → PlanFormDialog (existing, DIPERLUAS)
```

Tidak ada route baru. Konfigurasi cycle menjadi tab/section baru di dalam `PlanFormDialog`.

### 1.2. Sidebar Entry

Tidak ada perubahan sidebar. Entry "Paket Langganan" sudah ada di `SuperAdminSidebar.tsx`.

---

## 2. Perluasan `PlanFormDialog`

### 2.1. Tab Baru: "Kuota & Siklus"

`PlanFormDialog` saat ini menampilkan form untuk nama, harga, entitlements, dan limits. Ditambah tab baru:

```
[Informasi Umum]  [Fitur & Limit]  [Kuota & Siklus]  ← BARU
```

### 2.2. Layout Tab "Kuota & Siklus"

```
┌─────────────────────────────────────────────────────────────────┐
│  Kuota & Siklus                                                 │
│                                                                 │
│  Konfigurasi siklus reset dan perilaku degraded per metrik.     │
│  Metrik yang tidak dikonfigurasi akan menggunakan siklus        │
│  bulanan saja (default).                                        │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ Metrik              │ Siklus         │ Limit Mingguan │ ... ││
│  ├─────────────────────┼────────────────┼────────────────┼─────┤│
│  │ Kuota AI            │ [▼ Mingguan+B] │ [50.000      ] │ ... ││
│  │ Permintaan AI       │ [▼ Bulanan   ] │ —              │     ││
│  │ Transaksi           │ [▼ Mingguan+B] │ [200         ] │ ... ││
│  │ Pindai struk (OCR)  │ [▼ Mingguan+B] │ [30          ] │ ... ││
│  │ Input suara / hari  │ [▼ Bulanan   ] │ —              │     ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  Threshold Notifikasi:  [80] %  (default: 80%)                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.3. Kolom Tabel Matrix

| Kolom | Tipe Input | Keterangan |
|-------|-----------|------------|
| **Metrik** | Label (read-only) | Diambil dari `metric_definitions.label`. Dinamis — metrik baru otomatis muncul. |
| **Limit Bulanan** | Number input | Diambil dari `plan_limits.limit_value` (existing). -1 = unlimited. |
| **Siklus** | Select dropdown | `Bulanan` / `Mingguan + Bulanan`. Default: `Bulanan`. |
| **Limit Mingguan** | Number input | Muncul hanya jika Siklus = `Mingguan + Bulanan`. Wajib > 0 dan ≤ Limit Bulanan. |
| **Saat Mingguan Habis** | Select dropdown | Muncul hanya jika Siklus = `Mingguan + Bulanan`. Opsi: `Blokir` / `Batasi Kecepatan` / `Turunkan Model AI`. |
| **Parameter Degraded** | Conditional inputs | Muncul sesuai pilihan degraded mode (lihat §2.4). |

### 2.4. Parameter Degraded (Conditional)

#### Saat "Batasi Kecepatan" (throttle) dipilih:

```
┌──────────────────────────────────────────────────┐
│ Batasan:  [5] request per [60] detik             │
│ Maks penggunaan degraded: [___] (kosong=tanpa batas) │
└──────────────────────────────────────────────────┘
```

- `throttle_rate_amount`: integer > 0 (wajib).
- `throttle_rate_window_secs`: integer > 0 (wajib).
- `max_degraded_usage`: integer ≥ 0 (opsional, kosong = tanpa batas).

#### Saat "Turunkan Model AI" (model_downgrade) dipilih:

```
┌──────────────────────────────────────────────────┐
│ Model fallback: [gpt-4o-mini        ▼]          │
│ Maks penggunaan degraded: [10000] (kosong=tanpa batas) │
└──────────────────────────────────────────────────┘
```

- `fallback_model`: select dari daftar model yang tersedia di `ai_routes` (dinamis dari DB). Wajib non-empty.
- `max_degraded_usage`: integer ≥ 0 (opsional).

#### Saat "Blokir" (hard_block) dipilih:

Tidak ada parameter tambahan.

---

## 3. Komponen Frontend

### 3.1. File Baru

```
frontend/components/super-admin/plans/CycleConfigMatrix.tsx    — Tabel matrix utama
frontend/components/super-admin/plans/CycleConfigRow.tsx       — Satu baris metrik
frontend/components/super-admin/plans/DegradedParamsForm.tsx   — Parameter degraded conditional
frontend/hooks/usePlanCycles.ts                                 — Hook fetch/mutate cycle config
```

### 3.2. `CycleConfigMatrix`

Props:
```typescript
interface CycleConfigMatrixProps {
  planId: string;
  monthlyLimits: Record<string, number>;  // dari plan_limits
  metrics: MetricDefinition[];             // dari metric_definitions
}
```

Behavior:
- Fetch `GET /api/v1/admin/plans/:planId/cycles` saat mount.
- Merge dengan `metric_definitions` untuk menampilkan semua metrik — termasuk yang belum dikonfigurasi.
- Metrik tanpa config tampil sebagai `Bulanan` (default).
- Save: `PUT /api/v1/admin/plans/:planId/cycles` — batch upsert semua rows yang berubah.

### 3.3. `usePlanCycles` Hook

```typescript
export function usePlanCycles(planId: string) {
  const query = useQuery({
    queryKey: queryKeys.superAdmin.planCycles(planId),
    queryFn: () => api.get(`/admin/plans/${planId}/cycles`).then(r => r.data),
    enabled: !!planId,
  });

  const mutation = useMutation({
    mutationFn: (cycles: CycleConfig[]) =>
      api.put(`/admin/plans/${planId}/cycles`, { cycles }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.superAdmin.planCycles(planId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.entitlements.me });
    },
  });

  return { cycles: query.data, isLoading: query.isLoading, saveCycles: mutation.mutate };
}
```

---

## 4. Validasi Form (Client-Side)

### 4.1. Aturan Validasi

| Field | Aturan | Pesan Error |
|-------|--------|-------------|
| `weekly_limit` | > 0 jika cycle = weekly_monthly | "Limit mingguan wajib diisi dan > 0." |
| `weekly_limit` | ≤ monthly limit | "Limit mingguan tidak boleh melebihi limit bulanan ({monthly})." |
| `degraded_mode` | wajib jika cycle = weekly_monthly | "Pilih perilaku saat kuota mingguan habis." |
| `throttle_rate_amount` | > 0 jika mode = throttle | "Jumlah request per window wajib > 0." |
| `throttle_rate_window_secs` | > 0 jika mode = throttle | "Window waktu wajib > 0 detik." |
| `fallback_model` | non-empty jika mode = model_downgrade | "Pilih model fallback." |
| `max_degraded_usage` | ≥ 0 jika diisi | "Batas degraded harus ≥ 0." |
| `warning_threshold` | 1–100 | "Threshold harus antara 1% dan 100%." |

### 4.2. Validasi Cross-Field

- Jika `cycle_type = 'monthly_only'`, semua field weekly harus di-clear/null.
- Jika limit bulanan = -1 (unlimited), `cycle_type` otomatis `monthly_only` (tidak masuk akal punya weekly pada unlimited monthly).

---

## 5. Quota Notification Config

### 5.1. Section Terpisah di Halaman Plans

Di bawah tabel plans, tambahkan card kecil:

```
┌─────────────────────────────────────────────────────┐
│  Pengaturan Notifikasi Kuota                        │
│                                                     │
│  Threshold peringatan global: [80] %                │
│                                                     │
│  Override per plan:                                  │
│  ┌────────────┬──────────────┐                      │
│  │ Plan       │ Threshold    │                      │
│  ├────────────┼──────────────┤                      │
│  │ Pro        │ [75] %       │                      │
│  │ Business   │ [90] %       │                      │
│  └────────────┴──────────────┘                      │
│                                                     │
│  [Simpan]                                           │
└─────────────────────────────────────────────────────┘
```

---

## 6. Quota Monitor Enhancement

### 6.1. Halaman `/super-admin/quota-monitor` (DIPERLUAS)

Tabel quota monitor yang sudah ada diperluas dengan kolom:

| Kolom Baru | Keterangan |
|-----------|------------|
| **Siklus** | `Mingguan` / `Bulanan` — menunjukkan cycle mana yang aktif. |
| **Weekly Used / Limit** | `12.000 / 50.000` — muncul jika plan user punya weekly. |
| **Status** | Badge: `Normal` / `Peringatan` / `Degraded` / `Terblokir`. |
| **Reset Berikutnya** | Countdown: "3h 12m" (weekly) atau "12d 5h" (monthly). |

### 6.2. Filter Baru

- **Filter Siklus**: `Semua` / `Mingguan + Bulanan` / `Bulanan Saja` — filter user berdasarkan plan cycle config.
- **Filter Status**: `Semua` / `Normal` / `Peringatan` / `Degraded` / `Terblokir`.

---

## 7. Metric Definitions Management

### 7.1. Halaman Existing yang Dimodifikasi

Super admin sudah bisa mengelola `metric_definitions` dari UI catalog. Perlu ditambahkan:

- **Field `icon`**: dropdown ikon lucide-react (atau text input untuk nama ikon).
- **Field `display_order`**: number input untuk urutan tampil di UI user.

Kedua field ini ditambahkan ke form edit metrik yang sudah ada di halaman catalog management.

### 7.2. Penambahan Limit Type Baru

Saat super admin membuat `metric_definition` baru:
1. Metrik otomatis muncul di matrix `CycleConfigMatrix` semua plan.
2. Default: `monthly_only`, limit = 0 (deny).
3. Super admin set limit dan cycle per plan sesuai kebutuhan.
4. Backend menggunakan `resolver_key` untuk mengambil usage — resolver harus sudah ada di kode (deploy diperlukan untuk *kemampuan baru*, bukan untuk *kebijakan baru*).
