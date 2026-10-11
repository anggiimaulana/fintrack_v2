# 💳 3_Areas — Billing & Pembayaran

Area tanggung jawab berkelanjutan untuk **langganan, kuota token, dan pembayaran** FinTrack v2.
Ini area paling sensitif (uang nyata) — standar di sini **mengikat**.

## Scope

- Siklus hidup langganan (aktivasi, upgrade, downgrade, renewal, cancel, grace).
- Kuota token & ledger pemakaian.
- Harga dinamis (base + markup).
- Pembayaran (Midtrans + manual), invoice, prorata, dunning.
- Kebijakan uang & akurasi.

## Dokumen

| Dokumen | Isi |
|---|---|
| `README.md` | File ini |
| `KEBIJAKAN_BILLING.md` | Aturan bisnis billing yang mengikat |
| `RUNBOOK_DUNNING.md` | Prosedur penagihan & grace period |
| `RUNBOOK_PEMBAYARAN_MANUAL.md` | Prosedur review bukti transfer |
| `RUNBOOK_REFUND.md` | Prosedur refund |

## Prinsip

1. **Uang = integer rupiah.** Tidak pernah float.
2. **Ledger append-only.** Saldo hanya bertambah lewat entri baru.
3. **Idempoten.** Semua mutasi uang/token aman terhadap retry.
4. **Tidak pernah menghapus data user** karena alasan billing (soft-limit).
5. **Auditable.** Semua aksi billing tercatat.

## Referensi

- BRD/PRD: `../4_Resources/Dokumen-Spesifikasi/`
- Fase terkait: `../2_Projects/P04_Subscription-Token-Core/`, `../2_Projects/P05_Payments-Billing/`
- Keamanan pembayaran: `../keamanan/KEAMANAN_PEMBAYARAN.md`
