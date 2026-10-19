# Spesifikasi Teknis: Analitik, Grafik Interaktif & Smart Insight

Dokumen ini mendefinisikan integrasi komponen grafik berbasis Recharts, sinkronisasi status visualisasi ke URL search params, kartu *Smart Insight*, dan pemetaan grafik interaktif pada portal `admin` dan `super_admin`.

---

## 1. Komponen Grafik Terpadu

Visualisasi data FinTrack v2 menggunakan komponen Recharts yang telah dibungkus secara modular di `@/components/charts/`:

1. **`BarChartView`**: Perbandingan diskrit (misal: volume transaksi harian, breakdown pengeluaran per kategori, konsumsi token per modul).
2. **`LineChartView` & `AreaChartView`**: Tren kontinyu time-series (misal: pertumbuhan MRR, tren pendaftaran user, latensi p95 API gateway).
3. **`PieChartView` / Donut Chart**: Distribusi proporsi (misal: rasio status pembayaran, sebaran paket langganan Pro vs Enterprise vs Free).
4. **`MultiSeriesChart`**: Visualisasi perbandingan multi-metrik dalam satu sumbu waktu (misal: total request AI vs request sukses vs kuota diblokir).

---

## 2. Sinkronisasi Status Grafik ke URL (`chart-params.ts`)

Setiap interaksi pada filter grafik tersimpan langsung di URL, memungkinkan berbagi link (*shareable URL*) dan bookmark halaman dengan visualisasi yang persis sama.

### 2.1. Skema Parameter URL Grafik

| Parameter | Tipe Nilai | Default | Deskripsi |
|---|---|---|---|
| `chart` | `bar` \| `line` \| `area` \| `pie` | Sesuai konfigurasi halaman | Jenis tampilan visual grafik. |
| `period` | `week` \| `month` \| `3months` \| `6months` \| `year` | `month` | Rentang waktu agregasi data. |
| `group` | `summary` \| `category` \| `provider` \| `status` | `summary` | Dimensi pengelompokan data. |
| `tab` | string | Halaman default | Tab sub-analitik yang aktif. |

### 2.2. Helper Sinkronisasi Dua Arah
```typescript
import { useSearchParams, useRouter, usePathname } from "next/navigation";
import { parseChartFromParams, chartToParams, type ChartState } from "@/lib/chart-params";

export function useChartUrlState(defaults?: Partial<ChartState>) {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();

  const chartState = parseChartFromParams(searchParams, defaults);

  const setChartState = (next: Partial<ChartState>) => {
    const sp = new URLSearchParams(searchParams.toString());
    const params = chartToParams({ ...chartState, ...next });
    Object.entries(params).forEach(([k, v]) => sp.set(k, v));
    router.replace(`${pathname}?${sp.toString()}`, { scroll: false });
  };

  return { chartState, setChartState };
}
```

---

## 3. Matriks Halaman Target & Visualisasi Analitik

Sesuai prinsip efisiensi, grafik ditempatkan pada modul-modul strategis berorientasi volume dan metrik:

### 3.1. Halaman Portal Admin

| Rute Halaman | Jenis Grafik | Metrik yang Ditampilkan | Filter URL yang Didukung |
|---|---|---|---|
| `/admin` | `AreaChartView` + `BarChartView` | Total Transaksi (Volume & Nilai Rupiah) harian/mingguan + Top Kategori Pengeluaran. | `?chart=area&period=month&tab=overview` |
| `/admin/users` | `LineChartView` | Pertumbuhan Pengguna Baru vs Pengguna Aktif per Periode. | `?period=3months&group=status` |
| `/admin/providers` | `MultiSeriesChart` | Latensi Rata-rata (ms) & Success Rate (%) per Gateway Pembayaran / AI Provider. | `?period=week&group=provider` |

### 3.2. Halaman Portal Super Admin

| Rute Halaman | Jenis Grafik | Metrik yang Ditampilkan | Filter URL yang Didukung |
|---|---|---|---|
| `/super-admin` | `AreaChartView` + Donut | MRR (Monthly Recurring Revenue) Tren 6 Bulan + Distribusi Metode Pembayaran. | `?period=6months&chart=area` |
| `/super-admin/tokens` | `BarChartView` | Konsumsi Token AI per Model (GPT-4o, Claude 3.5, Gemini 1.5) & Pengeluaran Harian. | `?period=month&group=model` |
| `/super-admin/subscriptions` | `PieChartView` (Donut) | Distribusi Paket Langganan Aktif (Free, Starter, Pro, Enterprise). | `?group=plan` |
| `/super-admin/payments` | `BarChartView` | Funnel Pembayaran: Berhasil vs Pending Review Manual vs Gagal/Kedaluwarsa. | `?period=month&tab=funnel` |
| `/super-admin/kuota` | `BarChartView` Horizontal | Top 10 Pengguna Pemakan Kuota Tertinggi & Alokasi Token Terpakai. | `?period=month&tab=top-users` |
| `/super-admin/ai-gateway` | `LineChartView` | Latensi p95/p99 Gateway AI & Status Insiden Circuit Breaker per Route. | `?period=week&group=route` |

---

## 4. Komponen *Smart Insight Card*

Setiap grafik didampingi oleh kartu *Smart Insight* untuk menyajikan interpretasi data bisnis yang langsung dapat dipahami tanpa kebingungan membaca angka mentah:

```
+---------------------------------------------------------------------------------------------------------+
| [Ikon SVG: Sparkles]  INSIGHT ANALITIK OPERASIONAL                                                       |
|                                                                                                         |
| "Konsumsi token AI meningkat 18.4% minggu ini, didominasi oleh fitur Receipt Scanner Vision."            |
|                                                                                                         |
| [Badge Kenaikan: +18.4% ^]   [Rata-rata Harian: 4.250 Token]   [Total Biaya: Rp 2.450.000]              |
|                                                                                                         |
| Tindakan Rekomendasi:                                                                                   |
| -> Terdapat 3 akun mendekati batas kuota bulanan. Buka menu Kuota untuk mitigasi sebelum hard limit.    |
+---------------------------------------------------------------------------------------------------------+
```

### 4.1. Struktur Data Insight (`SmartInsightProps`)
```typescript
export interface SmartInsightProps {
  title?: string;
  summaryText: string;
  deltaPercent?: number; // Nilai persentase perbandingan periode sebelumnya
  deltaDirection?: "up" | "down" | "flat";
  isPositiveDelta?: boolean; // Hijau jika kenaikan bersifat positif, merah jika negatif (e.g. latensi naik = merah)
  metricsSummary?: { label: string; value: string }[];
  actionRecommendation?: {
    text: string;
    actionLabel?: string;
    href?: string;
  };
}
```

### 4.2. Aturan Format Bahasa & Presentasi
1. **Bahasa Manusiawi**: Ringkasan ditulis padat, lugas, dan bebas jargon statistik yang membingungkan.
2. **Ikon Resmi SVG**:
   - Tren naik: `TrendingUp` (hijau jika menguntungkan, merah jika biaya/error naik).
   - Tren turun: `TrendingDown` (merah jika revenue turun, hijau jika latensi/error turun).
   - Stabil: `Minus` (netral abu-abu).
   - Ikon wawasan: `Sparkles` atau `Lightbulb` (lucide-react).
3. **Penyajian Rupiah & Angka**: Semua nominal diformat ke Rupiah (`formatRupiah`) dan angka besar diformat dengan pemisah ribuan standar Indonesia (`Intl.NumberFormat('id-ID')`).

---

## 5. Optimalisasi Performa & Zero Layout Shift

1. **Responsivitas Mulus**:
   - Seluruh grafik dibungkus `<ResponsiveContainer width="100%" height={320}>`.
   - Menggunakan debounce pada event resize kontainer ketika sidebar dibuka atau diciutkan.
2. **Pembersihan Siklus Hidup**:
   - Animasi Recharts diatur durasi pendek (`isAnimationActive={true} animationDuration={300}`) untuk menjaga kelancaran interaksi (*smooth 60fps*).
   - Tooltip Recharts menggunakan komponen kustom yang di-memoize untuk mencegah lag kursor saat hover data padat.
