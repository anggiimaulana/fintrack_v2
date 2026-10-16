# Desain Komponen Chart Interaktif — FinTrack v2

Dokumen ini menguraikan spesifikasi teknis komponen grafik analitik interaktif berbasis library Recharts di dalam antarmuka Super Admin Dashboard.

---

## 1. Prinsip Visualisasi & Interaktivitas

Semua grafik visualisasi di portal Super Admin mematuhi pedoman UX berikut:
1. **Responsif Sepenuhnya**: Dibungkus di dalam `<ResponsiveContainer width="100%" height={320}>` sehingga beradaptasi mulus pada berbagai lebar layar.
2. **Kustom Tooltip Cerdas**: Tooltip diformat rapi menggunakan standar lokal Indonesia (misalnya pemformatan mata uang IDR via `Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR' })`).
3. **Drill-Down Interaktif**: Klik pada titik data grafik (batang bar atau area chart) secara otomatis mengarahkan super admin ke halaman detail terkait dengan query params terpasang (misal klik pada batang tanggal tertentu mengarahkan ke `/super-admin/payments?date=2026-10-10`).
4. **Legenda Interaktif (Toggle Series)**: Admin dapat mengklik item di legenda untuk menampilkan atau menyembunyikan seri data spesifik secara dinamis.

---

## 2. Rincian Komponen Visualisasi

### 2.1 AreaChart: Tren Pendapatan & MRR Harian
- **Komponen**: `<RevenueAreaChart data={timeseriesData} />`
- **Tipe Visual**: Gradient Area Chart bergaris ganda (Pendapatan Kotor Transaksi vs Garis Rata-rata MRR).
- **Warna Palet**: Emerald/Green (`#10B981`) dengan gradien memudar ke transparan.
- **Sumbu X**: Tanggal format harian (`DD MMM`).
- **Sumbu Y**: Nominal Rupiah dalam skala jutaan/ratusan ribu.
- **Interaksi Tooltip**: Menampilkan rincian: Total Pendapatan, Jumlah Transaksi Berhasil, dan Nilai MRR Proyeksi.

### 2.2 BarChart: Konsumsi Token Berdasarkan Modality
- **Komponen**: `<TokenConsumptionBarChart data={tokenData} />`
- **Tipe Visual**: Stacked Bar Chart (Batang Bertumpuk) yang memisahkan pemakaian token AI ke dalam 4 modalitas:
  - Chat (`#3B82F6` - Biru)
  - Vision (`#8B5CF6` - Ungu)
  - Voice (`#F59E0B` - Amber/Oranye)
  - Embedding (`#10B981` - Hijau)
- **Filter Rentang Waktu**: Tab tombol di header chart untuk beralih antara rentang `7 Hari Terakhir`, `30 Hari Terakhir`, dan `90 Hari Terakhir`.
- **Aksi Klik (Drill-Down)**: Mengarahkan pengguna ke `/super-admin/tokens?date=YYYY-MM-DD` dengan tabel ledger langsung terfilter pada tanggal tersebut.

### 2.3 Donut / Pie Chart: Rasio Metode Pembayaran & Gateway
- **Komponen**: `<PaymentMethodDonutChart data={gatewaySplitData} />`
- **Tipe Visual**: Donut Chart berlubang tengah (inner radius 60%, outer radius 80%) menampilkan pangsa gateway:
  - Midtrans (Virtual Account & QRIS)
  - Xendit
  - Manual Transfer (Bank Bukti Bayar)
- **Nilai Tengah**: Menampilkan total persentase transaksi sukses global (`96.8%`).

---

## 3. Contoh Implementasi Komponen Recharts

```tsx
import React from 'react';
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from 'recharts';

interface RevenueDataPoint {
  date: string;
  revenue_idr: number;
}

export function RevenueAreaChart({ data }: { data: RevenueDataPoint[] }) {
  const formatIDR = (value: number) =>
    new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      maximumFractionDigits: 0,
    }).format(value);

  return (
    <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-base font-semibold text-foreground">Tren Pendapatan Harian</h3>
      </div>
      <ResponsiveContainer width="100%" height={320}>
        <AreaChart data={data} margin={{ top: 10, right: 30, left: 20, bottom: 0 }}>
          <defs>
            <linearGradient id="colorRevenue" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#10b981" stopOpacity={0.4} />
              <stop offset="95%" stopColor="#10b981" stopOpacity={0.0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
          <XAxis dataKey="date" className="text-xs" />
          <YAxis tickFormatter={(val) => `Rp ${(val / 1_000_000).toFixed(1)}jt`} className="text-xs" />
          <Tooltip formatter={(value: number) => [formatIDR(value), 'Pendapatan']} />
          <Area
            type="monotone"
            dataKey="revenue_idr"
            stroke="#10b981"
            strokeWidth={2}
            fillOpacity={1}
            fill="url(#colorRevenue)"
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
```
Semua komponen grafik dioptimalkan agar tidak melakukan re-render yang tidak perlu saat interaksi scroll halaman terjadi.
