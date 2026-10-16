# Kalkulasi Prorata Upgrade & Kebijakan Downgrade (Proration)

Dokumen ini memaparkan model matematis perhitungan penyesuaian biaya prorata saat pengguna melakukan peningkatan paket (*upgrade*) serta aturan penjadwalan saat penurunan paket (*downgrade*) di FinTrack v2.

---

## 1. Prinsip Utama Prorata

Sesuai keputusan **K-08**, **K-09**, dan **K-P05-07**:
1. **Upgrade Bersifat Instan & Prorated**: Pengguna langsung menikmati limit dan fitur paket baru seketika setelah pembayaran selisih lunas.
2. **Pembulatan ke Bawah (Floor)**: Selisih biaya harian selalu dibulatkan ke bawah (*floor*) ke nilai rupiah penuh. Tidak ada biaya desimal atau sen.
3. **Downgrade Berlaku di Akhir Siklus (Tanpa Refund)**: Pengguna tidak menerima pengembalian dana saat berpindah ke paket lebih rendah atau gratis; hak akses paket tinggi tetap dinikmati hingga periode aktif berakhir.

---

## 2. Formula Matematis Prorata Upgrade

Misalkan pengguna saat ini berada pada **Paket A** (harga $P_A$, durasi $D$ hari) dan ingin beralih ke **Paket B** (harga $P_B$, durasi $D$ hari). Periode aktif paket berakhir pada $T_{\text{end}}$ dan waktu transaksi upgrade adalah $T_{\text{now}}$.

### Langkah 1: Menghitung Sisa Hari Aktif
Perhitungan menggunakan zona waktu resmi sistem `Asia/Jakarta`:
$$S = \max\left(0, \left\lceil \frac{T_{\text{end}} - T_{\text{now}}}{86400} \right\rceil\right)$$
Di mana $S$ adalah jumlah hari tersisa dalam siklus berjalan.

### Langkah 2: Menghitung Nilai Kredit Paket Lama
Nilai kredit yang belum terpakai dari paket lama dihitung secara proporsional dan dibulatkan ke bawah:
$$C_A = \left\lfloor \frac{S \times P_A}{D} \right\rfloor$$

### Langkah 3: Menghitung Beban Biaya Paket Baru
Beban pemakaian paket baru untuk sisa hari yang sama:
$$C_B = \left\lfloor \frac{S \times P_B}{D} \right\rfloor$$

### Langkah 4: Menghitung Nominal Tagihan Upgrade
Tagihan yang harus dibayar pengguna adalah selisih beban dikurangi kredit:
$$P_{\text{upgrade}} = \max(0, C_B - C_A)$$

---

## 3. Contoh Kasus Riil

- **Kondisi**:
  - Paket Lama (Starter): $P_A = \text{Rp } 50.000$ (Durasi $D = 30$ hari).
  - Paket Baru (Pro): $P_B = \text{Rp } 150.000$ (Durasi $D = 30$ hari).
  - Waktu berjalan: Hari ke-10 (Sisa hari $S = 20$ hari).
- **Perhitungan**:
  - Kredit Paket Lama ($C_A$):
    $$C_A = \left\lfloor \frac{20 \times 50.000}{30} \right\rfloor = \lfloor 33.333,33 \rfloor = \text{Rp } 33.333$$
  - Beban Paket Baru ($C_B$):
    $$C_B = \left\lfloor \frac{20 \times 150.000}{30} \right\rfloor = \lfloor 100.000 \rfloor = \text{Rp } 100.000$$
  - Tagihan Upgrade ($P_{\text{upgrade}}$):
    $$P_{\text{upgrade}} = 100.000 - 33.333 = \text{Rp } 66.667$$

Invoice penyesuaian baru diterbitkan sebesar **Rp 66.667**. Begitu pembayaran selesai terverifikasi, tanggal kedaluwarsa $T_{\text{end}}$ tetap sama, namun entitlement paket user langsung ditingkatkan ke paket Pro.

---

## 4. Implementasi Go di `internal/billing/proration.go`

```go
package billing

import (
	"math"
	"time"
)

type ProrationResult struct {
	RemainingDays    int64 `json:"remaining_days"`
	OldPlanCreditIDR int64 `json:"old_plan_credit_idr"`
	NewPlanChargeIDR int64 `json:"new_plan_charge_idr"`
	AmountToPayIDR   int64 `json:"amount_to_pay_idr"`
}

func CalculateUpgradeProration(
	oldPriceIDR int64,
	newPriceIDR int64,
	cycleDurationDays int64,
	periodEnd time.Time,
	now time.Time,
) ProrationResult {
	if newPriceIDR <= oldPriceIDR || cycleDurationDays <= 0 {
		return ProrationResult{}
	}

	diffSeconds := periodEnd.Sub(now).Seconds()
	if diffSeconds <= 0 {
		return ProrationResult{}
	}

	// Pembulatan ke atas untuk sisa hari kalender
	remainingDays := int64(math.Ceil(diffSeconds / 86400.0))
	if remainingDays > cycleDurationDays {
		remainingDays = cycleDurationDays
	}

	// Pembulatan ke bawah (floor) untuk kalkulasi uang integer rupiah
	oldCredit := (remainingDays * oldPriceIDR) / cycleDurationDays
	newCharge := (remainingDays * newPriceIDR) / cycleDurationDays

	amountToPay := newCharge - oldCredit
	if amountToPay < 0 {
		amountToPay = 0
	}

	return ProrationResult{
		RemainingDays:    remainingDays,
		OldPlanCreditIDR: oldCredit,
		NewPlanChargeIDR: newCharge,
		AmountToPayIDR:   amountToPay,
	}
}
```

---

## 5. Kebijakan Downgrade (End-of-Cycle)

Ketika user memilih paket yang lebih rendah dari paket aktif saat ini:
1. **Tidak Ada Invoice Pengembalian Dana**: Sistem tidak mengeluarkan refund dalam bentuk uang tunai maupun kredit saldo.
2. **Pencatatan Penjadwalan**:
   - Kolom `subscriptions.scheduled_plan_id` diisi dengan ID paket baru yang dituju.
   - Kolom `subscriptions.scheduled_at` mencatat waktu permintaan.
3. **Penerapan Otomatis Saat Siklus Berakhir**:
   - Saat worker renewal mendeteksi `current_period_end` tercapai, worker memeriksa apakah `scheduled_plan_id` terisi.
   - Jika terisi, paket langganan diturunkan ke `scheduled_plan_id`, entitas limit diperbarui, dan kolom jadwal dikosongkan.
   - Tidak ada data transaksi pengguna yang dihapus (*soft-limit policy*).
