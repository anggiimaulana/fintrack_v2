# Runbook — Dunning & Grace Period

## 1. Tujuan

Menagih perpanjangan langganan secara bertahap tanpa langsung memutus layanan, memaksimalkan
tingkat perpanjangan (renewal rate) sambil melindungi pendapatan.

## 2. Jadwal

| Waktu | Aksi | Status |
|---|---|---|
| H-3 | Reminder perpanjangan (in-app/email) | `active` |
| H-1 | Reminder + tautan bayar | `active` |
| H-0 | Reminder terakhir; coba auto-charge (bila metode mendukung) | `active` |
| H+0 → H+3 | **Grace period** — fitur tetap jalan | `grace` |
| H+4 dst | Dunning: reminder berulang, akses terbatas | `past_due` |
| Lewat kebijakan | Turun ke plan free (soft-limit) | `expired` |

> Grace period default **3 hari** (dapat diubah super admin di `settings`).

## 3. Langkah Operasional

1. Sistem otomatis mengirim reminder sesuai jadwal.
2. Auto-charge dicoba bila metode mendukung (mis. kartu).
3. Bila gagal → masuk `grace`; kirim notifikasi + tautan bayar.
4. Bila tetap gagal → `past_due`; batasi fitur (tetap tidak hapus data).
5. Bila melewati kebijakan → `expired` → turun ke free.

## 4. Prinsip

- **Jangan pernah** menghapus data user saat dunning/expired.
- Komunikasi jelas: alasan, jumlah, cara bayar, tenggat.
- Setiap perubahan status → `subscription_events` + audit.

## 5. Metrik

- Renewal rate, dunning recovery rate, waktu rata-rata pemulihan.

## 6. Eskalasi

- Pembayaran valid tapi status belum berubah → cek webhook (lihat `RUNBOOK_PEMBAYARAN_MANUAL.md`).

## Referensi

- `KEBIJAKAN_BILLING.md`
- `../../2_Projects/P05_Payments-Billing/teknis/dunning-grace.md`
