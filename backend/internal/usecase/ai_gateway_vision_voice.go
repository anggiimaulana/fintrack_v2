package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fintrack/backend/internal/aigateway/metering"
	"github.com/fintrack/backend/internal/domain"
)

// receiptCategories is the set of category names the model may choose from,
// loaded from the database. Injecting the real names is what makes the model
// "smart": instead of inventing a near-miss (e.g. "Makanan" when the seeded
// category is "Makanan & Minuman"), it picks a name that exists, so the client
// can match it to a category id and stop falling back to "Lain-lain".
type receiptCategories struct {
	Expense []string
	Income  []string
	Savings []string
}

// loadReceiptCategories reads the active category names. It is nil-safe: a
// usecase without repos (tests, partial wiring) yields empty lists, and the
// prompt falls back to generic guidance.
func (u *AIUsecase) loadReceiptCategories(ctx context.Context) receiptCategories {
	var cats receiptCategories
	if u.expenseCatRepo != nil {
		if list, err := u.expenseCatRepo.List(ctx, false); err == nil {
			for _, c := range list {
				if c.Name != "" {
					cats.Expense = append(cats.Expense, c.Name)
				}
			}
		}
	}
	if u.incomeCatRepo != nil {
		if list, err := u.incomeCatRepo.List(ctx, false); err == nil {
			for _, c := range list {
				if c.Name != "" {
					cats.Income = append(cats.Income, c.Name)
				}
			}
		}
	}
	if u.savingsCatRepo != nil {
		if list, err := u.savingsCatRepo.List(ctx, false); err == nil {
			for _, c := range list {
				if c.Name != "" {
					cats.Savings = append(cats.Savings, c.Name)
				}
			}
		}
	}
	return cats
}

// joinOrNone renders a category list for the prompt, or a fallback instruction
// when the list is empty (repos unavailable).
func joinOrNone(names []string) string {
	if len(names) == 0 {
		return "(pilih nama kategori umum yang paling mendekati)"
	}
	return strings.Join(names, ", ")
}

// receiptPrompt builds the vision prompt for receipt extraction. The category
// lists come from the live database so the model returns a name that exists,
// and the current date lets it resolve relative dates ("hari ini", "kemarin").
func receiptPrompt(cats receiptCategories, now time.Time) string {
	today := now.Format("2006-01-02")
	weekday := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}[int(now.Weekday())]

	return fmt.Sprintf(`Anda adalah asisten analisis struk/bukti transaksi keuangan Indonesia. Baca gambar dan ekstrak datanya dengan teliti.

Hari ini: %s, %s (WIB). Gunakan ini untuk menafsirkan tanggal relatif seperti "hari ini", "kemarin", atau tanggal tanpa tahun.

Aturan:
1. Tentukan TYPE transaksi:
   - "expense"  = pengeluaran (belanja, tagihan, bayar, transfer keluar, MRT, makan, dll)
   - "income"   = pemasukan (gaji, bonus, transfer masuk, penjualan, komisi)
   - "savings"  = tabungan/investasi (setoran tabungan, deposito, reksadana, dana pensiun)
2. Pilih CATEGORY **persis salah satu** dari daftar di bawah sesuai type. Jangan mengarang nama baru, jangan menyingkat, dan gunakan ejaan persis (termasuk "&").
3. Ambil NOMINAL total akhir yang benar-benar dibayar/diterima (setelah diskon/pajak bila ada). Hilangkan pemisah ribuan dan simbol mata uang.
4. Tulis DESKRIPSI singkat dan jelas: nama transaksi atau nama merchant (mis. "Tabungan pendidikan anak", "MRT", "Kopi Kenangan"). Maksimal 60 karakter.
5. Ambil TANGGAL (YYYY-MM-DD) dan WAKTU (HH:MM, 24 jam) bila terbaca. Jika waktu tidak ada, kosongkan.
6. Ambil MERCHANT/nama toko bila ada.
7. Rincian ITEM bila terlihat (nama, jumlah, harga satuan, total).
8. Hanya ekstrak yang jelas terbaca. Jangan mengarang data. Jika nominal tidak terbaca, isi 0.
9. Jika gambar BUKAN struk/tidak jelas, kembalikan {"success": false, "message": "..."}.

Kategori EXPENSE yang tersedia: %s
Kategori INCOME yang tersedia: %s
Kategori SAVINGS yang tersedia: %s

Jika struk memuat lebih dari satu transaksi (mis. pengeluaran + tabungan), kembalikan array "transactions".

Format JSON (satu transaksi):
{
  "success": true,
  "type": "expense",
  "category": "nama kategori persis dari daftar",
  "total": 239000,
  "description": "MRT",
  "merchant": "MRT Jakarta",
  "date": "%s",
  "time": "16:33",
  "items": [{"name": "Item", "quantity": 1, "unit_price": 239000, "total": 239000}],
  "raw_text": "teks yang terbaca"
}

Format JSON (beberapa transaksi dari satu struk):
{
  "success": true,
  "merchant": "Nama Toko",
  "date": "%s",
  "transactions": [
    {"type": "expense", "total": 50000, "category": "Pulsa & Data", "description": "pulsa 50k"},
    {"type": "savings", "total": 100000, "category": "Pendidikan", "description": "tabungan anak"}
  ],
  "items": [],
  "raw_text": "teks yang terbaca"
}`, weekday, today, joinOrNone(cats.Expense), joinOrNone(cats.Income), joinOrNone(cats.Savings), today, today)
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
			result = nil
		}
	}
	// A JSON `null` (or a scalar) unmarshals into a NIL map WITHOUT an error, and
	// writing to a nil map panics. Fall back to the same shape as an unparseable
	// response so the worker never crashes on odd model output.
	if result == nil {
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

	if _, ok := result["type"]; !ok {
		result["type"] = "expense"
	}
	// Respect an explicit refusal (e.g. the image is not a receipt) instead of
	// forcing success: the client uses this flag to show the message and skip
	// saving. Only default to true when the model did not say otherwise.
	if _, ok := result["success"]; !ok {
		result["success"] = true
	}
	return result
}

// analyzeReceiptViaGateway analyzes a receipt image through the centralized
// gateway (vision use case).
func (u *AIUsecase) analyzeReceiptViaGateway(ctx context.Context, imageData []byte, contentType string) (map[string]interface{}, domain.TokenUsage, error) {
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Load the live category names so the prompt asks for a name that exists,
	// then normalize whatever the model returns back onto those names.
	cats := u.loadReceiptCategories(ctx)
	prompt := receiptPrompt(cats, time.Now())

	text, usage, err := u.gateway.AnalyzeVision(ctx, &domain.AIVisionRequest{
		Prompt:      prompt,
		ImageBase64: base64Image,
		ImageMIME:   contentType,
	})
	if err != nil {
		return u.receiptErrorResult(err), usage, nil
	}
	// Provider-reported usage wins; otherwise estimate (text + flat per-image).
	if usage.TotalTokens == 0 {
		usage = metering.EstimateVision(prompt, text)
	}
	result := parseReceiptResponse(text)
	normalizeReceiptResult(result, cats)
	return result, usage, nil
}

// normalizeReceiptResult reconciles the model's answer with the database:
//   - the returned category is snapped onto the closest real category name for
//     its type, so the client can resolve an id instead of falling back;
//   - the description is filled from the merchant when the model omitted it;
//   - a missing/invalid type defaults to "expense".
//
// It is best-effort: when no category is close enough the original value is
// left in place (the client still has its own fallback).
func normalizeReceiptResult(result map[string]interface{}, cats receiptCategories) {
	if result == nil {
		return
	}

	if _, ok := result["type"]; !ok {
		result["type"] = "expense"
	}
	txType, _ := result["type"].(string)

	// Normalize the single-transaction shape.
	if _, ok := result["category"]; ok {
		raw, _ := result["category"].(string)
		result["category"] = snapCategory(raw, txType, cats)
	}
	if desc, _ := result["description"].(string); strings.TrimSpace(desc) == "" {
		if merchant, _ := result["merchant"].(string); strings.TrimSpace(merchant) != "" {
			result["description"] = merchant
		}
	}

	// Normalize each element of the multi-transaction shape.
	if txs, ok := result["transactions"].([]interface{}); ok {
		for _, item := range txs {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if _, ok := m["type"]; !ok {
				m["type"] = "expense"
			}
			itemType, _ := m["type"].(string)
			raw, _ := m["category"].(string)
			m["category"] = snapCategory(raw, itemType, cats)
			if desc, _ := m["description"].(string); strings.TrimSpace(desc) == "" {
				if merchant, _ := m["merchant"].(string); strings.TrimSpace(merchant) != "" {
					m["description"] = merchant
				}
			}
		}
	}
}

// snapCategory maps a model-provided category name onto the closest real
// category for the given type: exact match (case-insensitive) first, then a
// substring match in either direction, then the generic "lain" bucket. The
// generic fallback is deliberate: returning a real name (rather than the
// model's invented one) is what lets the client resolve a category id and
// auto-save, instead of failing with "Kategori tidak ditemukan".
func snapCategory(raw, txType string, cats receiptCategories) string {
	want := strings.TrimSpace(raw)
	if want == "" {
		return want
	}
	list := cats.Expense
	switch txType {
	case "income":
		list = cats.Income
	case "savings":
		list = cats.Savings
	}
	if len(list) == 0 {
		return want
	}

	lower := strings.ToLower(want)
	for _, name := range list {
		if strings.ToLower(name) == lower {
			return name
		}
	}
	// Substring match either way ("Makanan" -> "Makanan & Minuman").
	for _, name := range list {
		n := strings.ToLower(name)
		if strings.Contains(n, lower) || strings.Contains(lower, n) {
			return name
		}
	}
	// Generic bucket, tolerating "Lainnya" vs "Lain-lain".
	for _, name := range list {
		n := strings.ToLower(name)
		if strings.Contains(n, "lain") {
			return name
		}
	}
	return want
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
		usage = metering.EstimateAudioFromBytes(len(audio), transcript)
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
