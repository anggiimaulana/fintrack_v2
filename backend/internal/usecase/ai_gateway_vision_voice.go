package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fintrack/backend/internal/aigateway/metering"
	"github.com/fintrack/backend/internal/domain"
)

// receiptPrompt is the vision prompt for receipt extraction. Shared by the
// centralized gateway adapter.
func receiptPrompt() string {
	return `Anda adalah asisten analisis struk keuangan. Ekstrak informasi dari gambar struk berikut.

Aturan:
1. Identifikasi setiap item: **nama**, **jumlah**, **harga satuan**, **total**
2. Total keseluruhan struk
3. Tanggal transaksi (jika terbaca)
4. Nama toko/merchant (jika terbaca)
5. Kategori yang sesuai — untuk expense: Makanan, Transportasi, Belanja, Hiburan, Kesehatan, Pendidikan, Lain-lain. Untuk income: Gaji, Bonus, Investasi, Lainnya. Untuk savings: Tabungan Harian, Deposito, Lainnya.
6. Hanya ekstrak data yang jelas terbaca, jangan membuat data palsu
7. Jika gambar bukan struk/tidak jelas, beri tahu pengguna
8. Penting: Tentukan type transaksi: "income" untuk pemasukan (gaji, transfer masuk), "expense" untuk pengeluaran (belanja, tagihan), "savings" untuk tabungan/investasi (setoran tabungan, deposito, beli reksadana)
9. Penting: Jika struk mengandung lebih dari 1 jenis transaksi (misalnya pengeluaran + tabungan), kembalikan array "transactions" dengan setiap item memiliki type, total, category, description sendiri. Jika hanya 1 transaksi, kembalikan format single object seperti biasa.

Format respons JSON (single transaction):
{
  "merchant": "Nama Toko",
  "date": "2024-01-01",
  "total": 50000,
  "category": "Makanan",
  "type": "expense",
  "items": [{"name": "Item", "quantity": 1, "unit_price": 25000, "total": 25000}],
  "raw_text": "teks yang terbaca"
}

Format respons JSON (multiple transactions dari 1 struk):
{
  "merchant": "Nama Toko",
  "date": "2024-01-01",
  "transactions": [
    {"type": "expense", "total": 50000, "category": "Pulsa", "description": "pulsa 50k"},
    {"type": "expense", "total": 10000, "category": "Belanja", "description": "sampo 10k"}
  ],
  "items": [{"name": "Item", "quantity": 1, "unit_price": 25000, "total": 25000}],
  "raw_text": "teks yang terbaca"
}`
}

// parseReceiptResponse turns the model's raw text into the receipt result map.
// Shared by the vision adapter and the receipt error path.
func parseReceiptResponse(text string) map[string]interface{} {
	if text == "" {
		return map[string]interface{}{
			"success": false,
			"message": "Gambar tidak dikenali sebagai struk. Pastikan foto jelas dan terbaca.",
			"items":   []interface{}{},
		}
	}

	cleaned := text
	cleaned = stripThinkTags(cleaned)
	cleaned = extractJSONBlock(cleaned)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		if err2 := json.Unmarshal([]byte(text), &result); err2 != nil {
			result = map[string]interface{}{
				"success":  true,
				"raw_text": text,
				"items":    []interface{}{},
				"total":    0,
				"merchant": "",
				"category": "Lainnya",
				"type":     "expense",
			}
		}
	}

	if _, ok := result["type"]; !ok {
		result["type"] = "expense"
	}
	result["success"] = true
	return result
}

// analyzeReceiptViaGateway analyzes a receipt image through the centralized
// gateway (vision use case).
func (u *AIUsecase) analyzeReceiptViaGateway(ctx context.Context, imageData []byte, contentType string) (map[string]interface{}, domain.TokenUsage, error) {
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	text, usage, err := u.gateway.AnalyzeVision(ctx, &domain.AIVisionRequest{
		Prompt:      receiptPrompt(),
		ImageBase64: base64Image,
		ImageMIME:   contentType,
	})
	if err != nil {
		return u.receiptErrorResult(err), usage, nil
	}
	// Provider-reported usage wins; otherwise estimate (text + flat per-image).
	if usage.TotalTokens == 0 {
		usage = metering.EstimateVision(receiptPrompt(), text)
	}
	return parseReceiptResponse(text), usage, nil
}

// voicePrompt is the transcription prompt shared by both paths.
const voicePrompt = "Transkripsikan audio berikut dengan akurat. Hanya keluarkan teks hasil transkripsi, tanpa penjelasan tambahan."

// processVoiceViaGateway transcribes audio through the centralized gateway.
func (u *AIUsecase) processVoiceViaGateway(ctx context.Context, audioBase64 string) (map[string]interface{}, domain.TokenUsage, error) {
	audio, err := base64.StdEncoding.DecodeString(audioBase64)
	if err != nil {
		return map[string]interface{}{"success": false, "message": "Data audio tidak valid.", "transcript": ""}, domain.TokenUsage{}, nil
	}

	transcript, usage, err := u.gateway.TranscribeAudio(ctx, &domain.AIAudioTranscriptionRequest{
		Filename: "audio.webm",
		Audio:    audio,
		MIME:     "audio/webm",
	})
	if err != nil {
		return map[string]interface{}{
			"success":    false,
			"message":    "Gagal memproses audio. Silakan coba lagi.",
			"transcript": "",
		}, usage, nil
	}

	// Provider-reported usage wins; otherwise estimate from audio length.
	if usage.TotalTokens == 0 {
		usage = metering.EstimateAudio(estimateAudioSeconds(audio), transcript)
	}

	// Some providers return a JSON object as text; keep the existing behaviour
	// of trimming quotes/whitespace.
	return map[string]interface{}{
		"success":    true,
		"transcript": strings.TrimSpace(transcript),
	}, usage, nil
}

// estimateAudioSeconds approximates audio duration from the encoded size. WebM
// opus at ~32 kbps is roughly 4 KB/s; this is only used when the provider omits
// usage, so a coarse estimate is acceptable.
func estimateAudioSeconds(audio []byte) float64 {
	const bytesPerSecond = 4000.0
	if len(audio) == 0 {
		return 0
	}
	return float64(len(audio)) / bytesPerSecond
}

var _ = fmt.Sprintf
