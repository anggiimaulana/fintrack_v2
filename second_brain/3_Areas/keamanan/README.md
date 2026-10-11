# 🔐 3_Areas — Keamanan

Area tanggung jawab berkelanjutan untuk **keamanan** FinTrack v2: kebijakan, standar, dan prosedur yang
berlaku sepanjang siklus hidup produk.

## Scope

- Manajemen secret & kunci (JWT, `ENCRYPTION_KEY`, kunci AI gateway, Midtrans server key).
- Keamanan autentikasi & sesi (bcrypt cost 12, JWT HS256 15 menit, refresh token rotasi, cookie httpOnly).
- Otorisasi: role (`super_admin`, `user`) + entitlement (backend otoritatif).
- Keamanan pembayaran (verifikasi webhook, idempotensi, PCI-DSS SAQ-A).
- Kepatuhan UU PDP No. 27/2022.
- Respons insiden keamanan & rotasi kredensial.

## Dokumen

| Dokumen | Isi |
|---|---|
| `KEAMANAN_DAN_PRIVASI.md` | Kebijakan keamanan & privasi menyeluruh |
| `SECRET_MANAGEMENT.md` | Standar pengelolaan secret & rotasi |
| `AUTENTIKASI_OTORISASI.md` | Model auth & RBAC/entitlement |
| `KEAMANAN_PEMBAYARAN.md` | Standar keamanan transaksi & webhook |
| `RESPONS_INSIDEN.md` | Prosedur penanganan insiden keamanan |

## Prinsip

1. **Backend adalah satu-satunya otoritas.** Klien tidak pernah dipercaya.
2. **Least privilege** — role DB `fintrack_app` tanpa DDL; akses minimum yang diperlukan.
3. **Secret tidak pernah di-commit.** Hanya `.env.example` yang boleh ada di git.
4. **Enkripsi at-rest** untuk kunci AI & data sensitif; **TLS** untuk trafik.
5. **Audit append-only** untuk semua aksi sensitif.

## Referensi

- Aturan workspace: `../../CLAUDE.md`, `../../.agents/rules/workspace_rules.md`
- BRD kepatuhan: `../4_Resources/Dokumen-Spesifikasi/BRD/BRD - FinTrack v2.md`
- Area billing: `../billing-pembayaran/`
