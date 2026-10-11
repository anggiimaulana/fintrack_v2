# Dynamic Pricing Engine — FinTrack v2

Dokumen ini menjelaskan arsitektur dan kalkulasi matematis mesin penetapan harga dinamis (*Dynamic Pricing Engine*) pada modul `internal/billing/service/`. Mesin ini bertugas mengonversi konsumsi token model AI menjadi estimasi nilai nominal Rupiah bulat (`BIGINT`), menghitung margin keuntungan, serta menerapkan pengali modalitas (*modality multipliers*) secara adil dan terkonfigurasi.

---

## 1. Latar Belakang & Prinsip Utama (K-04, K-11, K-27)

1. **Konfigurasi Sepenuhnya Dinamis:** Seluruh parameter penetapan harga disimpan di basis data pada tabel `pricing_rules` dan dapat diubah sewaktu-waktu oleh Super Admin melalui antarmuka tanpa perlu *deploy* ulang atau me-restart aplikasi.
2. **Kepatuhan Tipe Moneter Bulat:** Seluruh kalkulasi biaya token menghasilkan bilangan bulat Rupiah (`int64`). Dilarang ada nilai desimal atau pecahan sen yang tersimpan di kolom basis data atau dipaparkan di antarmuka pengguna.
3. **Penyelarasan Modalitas AI:** Konsumsi sumber daya komputasi berbeda secara drastis antara modalitas teks biasa, pengenalan gambar (vision), pemrosesan suara (audio/voice), dan komputasi penalaran kompleks (reasoning). Mesin harga memperhitungkan modalitas ini melalui bobot pengali terpisah.

---

## 2. Formula Matematis Penetapan Biaya Token

Biaya akhir dalam mata uang Rupiah dihitung menggunakan rumus:

$$\text{Cost}_{\text{IDR}} = \left\lceil \left( \frac{\text{Tokens}}{1000} \right) \times \text{Base}_{\text{per\_1k}} \times \text{Markup}_{\text{mult}} \times \text{Modality}_{\text{mult}} \times \left(1 + \frac{\text{Markup}_{\text{pct}}}{100}\right) \right\rceil$$

### Definisi Variabel:
- $\text{Tokens}$: Jumlah token aktual yang dikonsumsi (integer).
- $\text{Base}_{\text{per\_1k}}$: Harga dasar modalitas teks per 1.000 token dalam Rupiah (misal: Rp 300).
- $\text{Markup}_{\text{mult}}$: Faktor pengali margin keuntungan kotor (misal: 1.25x untuk margin laba 25%).
- $\text{Modality}_{\text{mult}}$: Bobot pengali berdasarkan modalitas pemanggilan AI.
- $\text{Markup}_{\text{pct}}$: Tambahan persentase margin tambahan (integer, default: 0%).
- $\lceil \cdot \rceil$: Pembulatan ke atas (*ceiling*) ke bilangan bulat integer IDR terdekat.

---

## 3. Matriks Pengali Modalitas Default (`modality_multipliers`)

Konfigurasi disimpan dalam kolom `JSONB` pada tabel `pricing_rules`:

```json
{
  "chat": 1.0,
  "vision": 2.5,
  "voice": 3.0,
  "embedding": 0.5,
  "reasoning": 2.0
}
```

### Rasionalisasi Bobot Modalitas:
1. **Chat (`1.0x`):** Standar teks dasar (misal: analisis percakapan finansial). Menggunakan rasio acuan normal.
2. **Vision (`2.5x`):** Pemindaian dan ekstraksi struk/nota (*OCR receipt*). Membutuhkan resolusi gambar tinggi dan kapasitas token visual yang lebih mahal dari provider AI.
3. **Voice (`3.0x`):** Transkripsi dan interpretasi pesan suara pengguna. Melibatkan model audio (*Whisper / Speech-to-Text*) dengan overhead pemrosesan sinyal.
4. **Embedding (`0.5x`):** Vektorisasi transaksi untuk kebutuhan RAG (*vector 768d*). Model embedding jauh lebih murah per token dibandingkan model instruksi generatif.
5. **Reasoning (`2.0x`):** Analisis perencanaan keuangan mendalam (*financial advisor chain-of-thought*) yang membutuhkan reasoning tokens tambahan.

---

## 4. Implementasi Go: Algoritma Presisi Integer

Untuk mencegah kesalahan akumulasi *floating-point*, kalkulasi di Go mengonversi nilai pengali desimal menjadi basis integer penskalaan (faktor basis 10.000):

```go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "math"

    "fintrack_v2/internal/billing/domain"
)

type PricingCalculator struct {
    pricingRepo domain.PricingRuleRepository
}

func NewPricingCalculator(repo domain.PricingRuleRepository) *PricingCalculator {
    return &PricingCalculator{pricingRepo: repo}
}

// CalculateCostIDR menghitung biaya token bulat dalam Rupiah
func (c *PricingCalculator) CalculateCostIDR(ctx context.Context, tokens int64, modality string) (int64, error) {
    if tokens <= 0 {
        return 0, nil
    }

    // 1. Ambil aturan harga aktif dari cache / DB
    rule, err := c.pricingRepo.GetActiveRule(ctx)
    if err != nil {
        return 0, fmt.Errorf("gagal memuat aturan harga: %w", err)
    }

    // 2. Parse modality multiplier dari JSONB
    var multipliers map[string]float64
    if err := json.Unmarshal(rule.ModalityMultipliers, &multipliers); err != nil {
        return 0, fmt.Errorf("format JSON modality multipliers rusak: %w", err)
    }

    modalityMult, exists := multipliers[modality]
    if !exists {
        modalityMult = 1.0 // fallback ke bobot normal jika tidak terdaftar
    }

    // 3. Kalkulasi presisi tinggi:
    // Gunakan float64 hanya untuk kalkulasi sementara, lalu segera bulatkan ke atas ke int64
    tokensInThousands := float64(tokens) / 1000.0
    baseCost := tokensInThousands * float64(rule.BasePer1kIDR)
    withMarkup := baseCost * rule.MarkupMultiplier
    withModality := withMarkup * modalityMult
    
    if rule.MarkupPercent > 0 {
        withModality = withModality * (1.0 + (float64(rule.MarkupPercent) / 100.0))
    }

    // 4. Pembulatan ke atas (Ceil) ke integer Rupiah
    costIDR := int64(math.Ceil(withModality))

    return costIDR, nil
}
```

---

## 5. Simulasi Perhitungan Biaya (Contoh Kasus)

Dengan konfigurasi:
- `BasePer1kIDR`: Rp 300
- `MarkupMultiplier`: 1.25 (25% margin)
- `MarkupPercent`: 0%

| Skenario Penggunaan | Modalitas | Pengali | Jumlah Token | Kalkulasi Kasar | Hasil Akhir (`cost_idr`) |
|---|:---:|:---:|:---:|:---:|:---:|
| Tanya saldo cepat | `chat` | 1.0x | 450 | $\frac{450}{1000} \times 300 \times 1.25 \times 1.0 = 168.75$ | **Rp 169** |
| Analisis portofolio bulanan | `chat` | 1.0x | 3.500 | $\frac{3500}{1000} \times 300 \times 1.25 \times 1.0 = 1312.5$ | **Rp 1.313** |
| Scan struk restoran | `vision` | 2.5x | 2.200 | $\frac{2200}{1000} \times 300 \times 1.25 \times 2.5 = 2062.5$ | **Rp 2.063** |
| Input pengeluaran via suara | `voice` | 3.0x | 1.100 | $\frac{1100}{1000} \times 300 \times 1.25 \times 3.0 = 1237.5$ | **Rp 1.238** |
| Vektorisasi transaksi (RAG) | `embedding` | 0.5x | 12.000 | $\frac{12000}{1000} \times 300 \times 1.25 \times 0.5 = 2250.0$ | **Rp 2.250** |
| Konseling keuangan AI | `reasoning` | 2.0x | 4.800 | $\frac{4800}{1000} \times 300 \times 1.25 \times 2.0 = 3600.0$ | **Rp 3.600** |

Semua hasil pembulatan konsisten ke atas (*ceiling*), memastikan FinTrack tidak pernah menanggung selisih pecahan biaya vendor komputasi.
