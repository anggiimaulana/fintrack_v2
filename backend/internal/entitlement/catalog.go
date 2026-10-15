// Package entitlement resolves a user's effective access rights from their
// subscription plan (features + limits). Results are cached with a version
// stamp so a plan change propagates instantly without re-login.
package entitlement

// FeatureKey describes one boolean feature switch a plan can grant.
type FeatureKey struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// LimitKey describes one numeric plan limit.
type LimitKey struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Unit is the display unit ("token", "transaksi", "scan", ...).
	Unit string `json:"unit"`
	// Enforced reports whether the backend actually computes usage for this key.
	// A limit that is not enforced is informational only.
	Enforced bool `json:"enforced"`
}

// FeatureCatalog is the authoritative list of feature keys the backend
// understands. The admin UI reads this so operators pick a key from a dropdown
// instead of typing one — a typo like `ai_chatt` can no longer silently disable
// a feature. The key set is code-defined because the code is what interprets it.
var FeatureCatalog = []FeatureKey{
	{Key: "ai_chat", Label: "Asisten AI (chat)", Description: "Chat keuangan dengan AI."},
	{Key: "ocr_receipt", Label: "Pindai struk (OCR)", Description: "Ekstraksi data dari foto struk."},
	{Key: "voice_input", Label: "Input suara", Description: "Transkripsi perintah suara."},
	{Key: "rag_search", Label: "Pencarian cerdas (RAG)", Description: "Pencarian semantik atas transaksi."},
	{Key: "reports_export", Label: "Ekspor laporan", Description: "Unduh laporan keuangan."},
	{Key: "advanced_budgeting", Label: "Anggaran lanjutan", Description: "Anggaran & analisis lanjutan."},
	{Key: "api_access", Label: "Akses API", Description: "Akses API publik."},
	{Key: "bank_sync", Label: "Sinkronisasi bank", Description: "Sinkronisasi rekening bank (belum tersedia)."},
}

// LimitCatalog is the authoritative list of numeric limit keys.
var LimitCatalog = []LimitKey{
	{Key: "ai_tokens_per_cycle", Label: "Token AI / siklus", Unit: "token", Enforced: true},
	{Key: "ai_requests_per_cycle", Label: "Permintaan AI / siklus", Unit: "permintaan", Enforced: true},
	{Key: "max_transactions", Label: "Transaksi", Unit: "transaksi", Enforced: true},
	{Key: "ocr_scans_per_cycle", Label: "Pindai struk / siklus", Unit: "scan", Enforced: true},
	{Key: "voice_requests_per_day", Label: "Permintaan suara / hari", Unit: "permintaan", Enforced: true},
	// Not enforced: categories are admin-managed and there is no per-user wallet
	// entity yet, so these are informational until those features exist.
	{Key: "max_categories", Label: "Kategori", Unit: "kategori", Enforced: false},
	{Key: "max_wallets", Label: "Dompet", Unit: "dompet", Enforced: false},
}

// IsKnownFeature reports whether key is a catalogued feature key.
func IsKnownFeature(key string) bool {
	for _, f := range FeatureCatalog {
		if f.Key == key {
			return true
		}
	}
	return false
}

// IsKnownLimit reports whether key is a catalogued limit key.
func IsKnownLimit(key string) bool {
	for _, l := range LimitCatalog {
		if l.Key == key {
			return true
		}
	}
	return false
}
