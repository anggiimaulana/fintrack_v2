package usecase

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/infrastructure/crypto"
	"github.com/fintrack/backend/internal/infrastructure/embedding"
	"github.com/fintrack/backend/pkg/logger"
)

type geminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

type sessionRateLimit struct {
	count     int
	cooldownUntil time.Time
}

type AIUsecase struct {
	chatSessionRepo domain.ChatSessionRepository
	chatMessageRepo domain.ChatMessageRepository
	keyRepo         domain.GeminiKeyRepository
	txRepo          domain.TransactionRepository
	incomeCatRepo   domain.IncomeCategoryRepository
	expenseCatRepo  domain.ExpenseCategoryRepository
	savingsCatRepo  domain.SavingsCategoryRepository
	profileRepo     domain.ProfileRepository
	cryptoKey       []byte
	httpClient      *http.Client
	rateLimits      map[string]*sessionRateLimit
	rateLimitMu     sync.Mutex
	embedder        *embedding.GeminiEmbedder
	embedRepo       domain.EmbeddingRepository
	ragWarned       bool
}

func NewAIUsecase(
	chatSessionRepo domain.ChatSessionRepository,
	chatMessageRepo domain.ChatMessageRepository,
	keyRepo domain.GeminiKeyRepository,
	txRepo domain.TransactionRepository,
	incomeCatRepo domain.IncomeCategoryRepository,
	expenseCatRepo domain.ExpenseCategoryRepository,
	savingsCatRepo domain.SavingsCategoryRepository,
	profileRepo domain.ProfileRepository,
	cryptoKey []byte,
	embedder *embedding.GeminiEmbedder,
	embedRepo domain.EmbeddingRepository,
) *AIUsecase {
	return &AIUsecase{
		chatSessionRepo: chatSessionRepo,
		chatMessageRepo: chatMessageRepo,
		keyRepo:         keyRepo,
		txRepo:          txRepo,
		incomeCatRepo:   incomeCatRepo,
		expenseCatRepo:  expenseCatRepo,
		savingsCatRepo:  savingsCatRepo,
		profileRepo:     profileRepo,
		cryptoKey:       cryptoKey,
		httpClient:      &http.Client{Timeout: 120 * time.Second},
		rateLimits:      make(map[string]*sessionRateLimit),
		embedder:        embedder,
		embedRepo:       embedRepo,
	}
}

// ─── Content Filter ────────────────────────────────────────────

var allowedTopics = []string{
	"keuangan", "finansial", "uang", "budget", "anggaran",
	"pemasukan", "pendapatan", "gaji", "bonus",
	"pengeluaran", "belanja", "biaya", "tagihan",
	"tabungan", "investasi", "saham", "reksadana", "obligasi",
	"hutang", "pinjaman", "kredit", "cicilan",
	"transaksi", "kategori", "laporan", "grafik",
	"analisis", "insight", "rekomendasi", "saran",
	"spk", "skor", "status", "kritis", "boros",
	"gemini", "apikey", "token", "api",
	"pajak", "tax", "asuransi", "dana darurat",
	"pensiun", "retirement", "portofolio",
	"neraca", "cashflow", "arus kas", "neraca",
	"resesi", "inflasi", "suku bunga", "dividen",
	"crypto", "aset", "liabilitas", "ekuitas",
}

var blockedPatterns = []string{
	"hack", "curi", "bobol", "exploit", "crack",
	"buatkan kode", "tulis kode", "code",
	"politik", "presiden", "partai", "pemilu",
	"narkoba", "judi", "slot", "togel",
	"senjata", "teroris", "radikal",
	"seks", "porno", "dewasa",
	"rasisme", "sara",
}

func (u *AIUsecase) isContentAllowed(message string) (bool, string) {
	msg := strings.ToLower(message)

	for _, pattern := range blockedPatterns {
		if strings.Contains(msg, pattern) {
			return false, "Maaf, saya hanya dapat membantu pertanyaan seputar keuangan dan pengelolaan finansial. Silakan tanyakan hal lain yang berkaitan dengan keuangan Anda."
		}
	}

	greetings := []string{"hai", "hi", "halo", "helo", "hey", "hay", "hy", "tes", "test", "coba", "123", "yo", "oi", "eh"}
	for _, g := range greetings {
		if msg == g || strings.Contains(msg, g) {
			return true, ""
		}
	}

	if len([]rune(message)) < 3 {
		return true, ""
	}

	isFinanceRelated := false
	for _, topic := range allowedTopics {
		if strings.Contains(msg, topic) {
			isFinanceRelated = true
			break
		}
	}

	if !isFinanceRelated && len(msg) > 5 {
		questionWords := []string{"siapa", "apa", "bagaimana", "kenapa", "mengapa", "kapan", "dimana"}
		isQuestion := false
		for _, q := range questionWords {
			if strings.Contains(msg, q) {
				isQuestion = true
				break
			}
		}
		if isQuestion {
			financeWords := []string{"uang", "finansial", "keuangan", "budget", "tabungan", "investasi", "pajak"}
			isFinanceQ := false
			for _, f := range financeWords {
				if strings.Contains(msg, f) {
					isFinanceQ = true
					break
				}
			}
			if !isFinanceQ {
				return false, "Maaf, saya adalah asisten keuangan FinTrack. Saya hanya dapat membantu pertanyaan seputar keuangan pribadi, analisis transaksi, dan perencanaan finansial. Silakan tanyakan sesuatu yang berkaitan dengan data keuangan Anda."
			}
		}
	}

	return true, ""
}

// ─── RAG: Get User Context ─────────────────────────────────────

func (u *AIUsecase) buildUserContext(ctx context.Context, userID uuid.UUID) (string, error) {
	var sb strings.Builder

	profile, _ := u.profileRepo.GetByID(ctx, userID)
	if profile != nil {
		sb.WriteString(fmt.Sprintf("Nama Pengguna: %s\n", profile.FullName))
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	prevMonthStart := monthStart.AddDate(0, -1, 0)

	stats, _ := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: monthStart, EndDate: now})
	if stats != nil {
		sb.WriteString(fmt.Sprintf("Bulan Ini - Pemasukan: Rp%.0f, Pengeluaran: Rp%.0f, Tabungan: Rp%.0f, Saldo Bersih: Rp%.0f\n",
			stats.TotalIncome, stats.TotalExpense, stats.TotalSavings, stats.NetBalance))
	}

	prevStats, _ := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: prevMonthStart, EndDate: monthStart.Add(-time.Second)})
	if prevStats != nil && prevStats.TotalIncome > 0 {
		incomeChange := ((stats.TotalIncome - prevStats.TotalIncome) / prevStats.TotalIncome) * 100
		expenseChange := ((stats.TotalExpense - prevStats.TotalExpense) / prevStats.TotalExpense) * 100
		sb.WriteString(fmt.Sprintf("Perubahan dari bulan lalu - Pemasukan: %.1f%%, Pengeluaran: %.1f%%\n", incomeChange, expenseChange))
	}

	recentTx, _ := u.txRepo.GetRecentByUser(ctx, userID, 5)
	if len(recentTx) > 0 {
		sb.WriteString("\n5 Transaksi Terbaru:\n")
		for _, t := range recentTx {
			desc := ""
			if t.Description != nil {
				desc = *t.Description
			}
			sb.WriteString(fmt.Sprintf("- %s: Rp%.0f (%s)\n", desc, t.Amount, t.TransactionDate.Format("2006-01-02")))
		}
	}

	categories, _ := u.expenseCatRepo.List(ctx, false)
	if len(categories) > 0 {
		sb.WriteString("\nKategori Pengeluaran:\n")
		for _, c := range categories {
			sb.WriteString(fmt.Sprintf("- %s\n", c.Name))
		}
	}

	savingsCats, _ := u.savingsCatRepo.List(ctx, false)
	if len(savingsCats) > 0 {
		sb.WriteString("\nKategori Tabungan:\n")
		for _, c := range savingsCats {
			sb.WriteString(fmt.Sprintf("- %s\n", c.Name))
		}
	}

	return sb.String(), nil
}

// retrieveRelevantTransactions performs a vector search over the user's
// transaction embeddings and formats the results for prompt injection.
// Best-effort: returns "" if RAG is disabled or search fails.
func (u *AIUsecase) retrieveRelevantTransactions(ctx context.Context, userID uuid.UUID, message string) string {
	if u.embedder == nil || u.embedRepo == nil {
		return ""
	}
	if !u.embedder.Enabled() {
		if !u.ragWarned {
			logger.Warn("RAG disabled: GEMINI_EMBEDDING_API_KEY not set")
			u.ragWarned = true
		}
		return ""
	}
	if len([]rune(strings.TrimSpace(message))) <= 10 {
		return ""
	}

	vec, err := u.embedder.Embed(ctx, message)
	if err != nil {
		logger.Warn("RAG embed failed", zap.Error(err))
		return ""
	}

	results, err := u.embedRepo.Search(ctx, userID, vec, 8)
	if err != nil {
		logger.Warn("RAG search failed", zap.Error(err))
		return ""
	}

	var sb strings.Builder
	for _, r := range results {
		if r.Similarity < 0.35 {
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s (similarity %.2f)\n", r.Content, r.Similarity))
	}
	return sb.String()
}

// ─── Build System Prompt ───────────────────────────────────────

func (u *AIUsecase) buildSystemPrompt(userContext string) string {
	return fmt.Sprintf(`Hari ini: %s

Kamu adalah asisten keuangan FinTrack yang santai dan helpful.

IDENTITAS:
- Nama: FinTrack AI Assistant
- Bahasa: Indonesia gaul santai, kayak ngobrol sama temen

GAYA JAWABAN:
- Untuk **perintah langsung** (catat, buat, tambah, input): jawab SANGAT SINGKAT, cukup "Done!" atau "Berhasil" + detailnya. Jangan nanya-nanya lagi, langsung eksekusi.
- Untuk **analisis/investasi/pertanyaan kompleks**: jawab agak panjang boleh, jelasin detail, pake paragraf, tapi tetep to the point dan gak bertele-tele.
- Pake **bold** buat nominal (Rp 50.000), persentase, poin penting
- Kalo lebih dari 1 item, pake format numbered list
- Inget konteks chat sebelumnya, jangan lupa apa yg udah dibahas

ATURAN:
1. Jawab seputar keuangan, analisis finansial, investasi, dan pengelolaan uang
2. Kalo ditanya di luar konteks, tolak dengan sopan
3. Pake data pengguna biar jawabannya personal
4. Kalo data kosong atau gak ada, JANGAN ngasih rekomendasi atau analisis palsu. Bilang aja datanya belum ada.
5. JANGAN pernah membuat rekomendasi investasi palsu (contoh saham, reksadana, dll) dari data yang gak ada atau contoh hipotetis
6. Untuk perintah CRUD → langsung panggil tool, jangan tanya konfirmasi lagi
7. Kalo kategori yg diminta gak ada, pake aja yg paling mirip atau auto-create

DATA PENGGUNA SAAT INI:
%s

INSTRUKSI:
- Perintah catat/buat/tambah → KAMU HARUS panggil tool <TOOL_CALL> dulu baru jawab "Done!". Kalo cuma jawab "Done!" doang, transaksi GAK AKAN tersimpan.
- Pertanyaan analisis → jawab detail, pake paragraf, berikan insight
- Pake **bold** buat nominal, tanggal, atau poin penting
- Inget konteks chat sebelumnya biar gak nanya ulang`, time.Now().Format("2006-01-02"), userContext)
}

// ─── Multi-Provider Key Management (Round-Robin Callback) ──────

type providerEndpoint struct {
	baseURL string
	model   string
	path    string
}

var providerEndpoints = map[string]map[string]providerEndpoint{
	"fast": {
		"gemini":     {baseURL: "https://generativelanguage.googleapis.com/v1beta", model: "gemini-2.0-flash-lite", path: "/models/gemini-2.0-flash-lite:generateContent"},
		"openai":     {baseURL: "https://api.openai.com/v1", model: "gpt-4o-mini", path: "/chat/completions"},
		"groq":       {baseURL: "https://api.groq.com/openai/v1", model: "meta-llama/llama-4-scout-17b-16e-instruct", path: "/chat/completions"},
		"openrouter": {baseURL: "https://openrouter.ai/api/v1", model: "meta-llama/llama-3.1-8b-instruct:free", path: "/chat/completions"},
		"qwen":       {baseURL: "https://api.qwen.ai/v1", model: "qwen-turbo", path: "/chat/completions"},
		"deepseek":   {baseURL: "https://api.deepseek.com/v1", model: "deepseek-chat", path: "/chat/completions"},
		"mistral":    {baseURL: "https://api.mistral.ai/v1", model: "mistral-small-latest", path: "/chat/completions"},
		"cohere":     {baseURL: "https://api.cohere.ai/v1", model: "command-r", path: "/generate"},
		"perplexity": {baseURL: "https://api.perplexity.ai", model: "sonar-pro", path: "/chat/completions"},
		"claude":     {baseURL: "https://api.anthropic.com/v1", model: "claude-3-haiku-20240307", path: "/messages"},
	},
	"reasoning": {
		"gemini":     {baseURL: "https://generativelanguage.googleapis.com/v1beta", model: "gemini-2.0-pro-exp-02-05", path: "/models/gemini-2.0-pro-exp-02-05:generateContent"},
		"openai":     {baseURL: "https://api.openai.com/v1", model: "o3-mini", path: "/chat/completions"},
		"openrouter": {baseURL: "https://openrouter.ai/api/v1", model: "deepseek/deepseek-r1:free", path: "/chat/completions"},
		"deepseek":   {baseURL: "https://api.deepseek.com/v1", model: "deepseek-reasoner", path: "/chat/completions"},
		"claude":     {baseURL: "https://api.anthropic.com/v1", model: "claude-3-opus-20240229", path: "/messages"},
	},
}

func (u *AIUsecase) getAvailableKeys(ctx context.Context, userID uuid.UUID) ([]domain.GeminiAPIKey, error) {
	return u.keyRepo.ListByUser(ctx, userID)
}

func (u *AIUsecase) findWorkingKey(ctx context.Context, keys []domain.GeminiAPIKey) *domain.GeminiAPIKey {
	for i := range keys {
		if keys[i].Status == domain.KeyActive {
			key := keys[i]
			return &key
		}
	}
	return nil
}

func (u *AIUsecase) findWorkingKeyByModality(ctx context.Context, keys []domain.GeminiAPIKey, modality string) *domain.GeminiAPIKey {
	for i := range keys {
		if keys[i].Status != domain.KeyActive {
			continue
		}
		switch modality {
		case domain.ModalityAudio:
			if keys[i].SupportsAudio {
				key := keys[i]
				return &key
			}
		case domain.ModalityVision:
			if keys[i].SupportsVision {
				key := keys[i]
				return &key
			}
		default:
			if keys[i].SupportsText {
				key := keys[i]
				return &key
			}
		}
	}
	return nil
}

func (u *AIUsecase) decryptAPIKey(encrypted string) (string, error) {
	plaintext, err := crypto.Decrypt(encrypted, u.cryptoKey)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// ─── Template Responses ────────────────────────────────────────

func getTemplateResponse(msg string) (string, bool) {
	greetingPatterns := []string{
		"hai", "hi", "halo", "helo", "hey", "hay", "hy",
		"siang", "pagi", "malam", "sore",
		"selamat pagi", "selamat siang", "selamat sore", "selamat malam",
		"good morning", "good afternoon", "good evening",
		"tes", "test", "coba", "123", "testing",
	}

	identityPatterns := []string{
		"siapa kamu", "kamu siapa", "who are you", "nama kamu",
		"kenalan", "perkenalkan diri",
	}

	creatorPatterns := []string{
		"siapa pembuatmu", "dibuat oleh siapa", "siapa yang membuat",
		"who created you", "siapa develop", "pengembang",
		"kembangan kamu", "siapa buat kamu",
	}

	capabilityPatterns := []string{
		"bisa apa", "what can you do", "kemampuan",
		"fitur", "fungsi kamu", "kamu bisa apa",
	}

	for _, p := range greetingPatterns {
		if msg == p || strings.HasPrefix(msg, p) {
			return "Halo! Ada yang bisa saya bantu terkait keuangan Anda?\n\nSaya bisa membantu:\n**1.** Mengecek saldo dan pengeluaran bulan ini\n**2.** Menganalisis kebiasaan belanja\n**3.** Mencatat transaksi baru\n**4.** Memberikan rekomendasi anggaran\n**5.** Menganalisis struk belanja (foto)\n**6.** Memproses perintah suara\n\nSilakan tanya apa pun tentang keuangan Anda!", true
		}
	}

	for _, p := range identityPatterns {
		if strings.Contains(msg, p) {
			return "Saya **FinTrack AI Assistant**! Asisten keuangan pintar yang siap membantu Anda mengelola:\n- **Pemasukan & Pengeluaran**\n- **Anggaran & Tabungan**\n- **Analisis keuangan pribadi**\n- **Visualisasi data transaksi**\n\nSaya dikembangkan khusus untuk membantu Anda mencapai kebebasan finansial!", true
		}
	}

	for _, p := range creatorPatterns {
		if strings.Contains(msg, p) {
			return "Saya dikembangkan oleh tim **FinTrack** sebagai asisten keuangan berbasis AI.\n\nTeknologi yang digunakan:\n- **Frontend:** Next.js + TypeScript\n- **Backend:** Go (Gin Framework)\n- **Database:** PostgreSQL (Supabase)\n- **AI:** Multi-provider (Gemini, OpenAI, Groq, dll)\n- **Arsitektur:** Clean Architecture + RAG", true
		}
	}

	for _, p := range capabilityPatterns {
		if strings.Contains(msg, p) {
			return "Saya bisa melakukan banyak hal untuk keuangan Anda!\n\n**\u2705 Analisis Keuangan**\n- Cek total pemasukan/pengeluaran bulan ini\n- Bandingkan dengan bulan sebelumnya\n- Lihat tren pengeluaran per kategori\n\n**\u2705 Catat Transaksi**\n- Tambah pemasukan atau pengeluaran\n- Kategorisasi otomatis\n\n**\u2705 Rekomendasi**\n- Saran anggaran bulanan\n- Tips hemat berdasarkan pola belanja\n- Peringatan jika pengeluaran membengkak\n\n**\u2705 Fitur Khusus**\n- Upload foto struk untuk analisis otomatis\n- Input suara untuk catat transaksi\n- Export laporan keuangan", true
		}
	}

	return "", false
}

// ─── Core Chat with Multi-Provider ─────────────────────────────

func (u *AIUsecase) Chat(ctx context.Context, userID uuid.UUID, req domain.ChatRequest, sessionID *uuid.UUID, source string, msgChan chan<- domain.ChatStreamEvent) error {
	defer close(msgChan)

	msgLower := strings.ToLower(strings.TrimSpace(req.Message))

	if templateResp, ok := getTemplateResponse(msgLower); ok {
		// Template responses (greetings, identity, capabilities) skip content filter
		if sessionID == nil {
			sid := uuid.New()
			sessionID = &sid
			session := &domain.ChatSession{ID: *sessionID, UserID: userID, Title: "Percakapan baru", Source: source}
			if err := u.chatSessionRepo.Create(ctx, session); err != nil {
				return fmt.Errorf("failed to create session: %w", err)
			}
			msgChan <- domain.ChatStreamEvent{Type: "session_created", Content: session.ID.String()}
		}
		msgChan <- domain.ChatStreamEvent{Type: "content", Content: templateResp}
		msgChan <- domain.ChatStreamEvent{Type: "done", Done: true}
		return nil
	}

	allowed, rejection := u.isContentAllowed(req.Message)
	if !allowed {
		msgChan <- domain.ChatStreamEvent{Type: "content", Content: rejection}
		msgChan <- domain.ChatStreamEvent{Type: "done", Done: true}
		return nil
	}

	if sessionID == nil {
		sid := uuid.New()
		sessionID = &sid
		session := &domain.ChatSession{ID: *sessionID, UserID: userID, Title: "Percakapan baru", Source: source}
		if err := u.chatSessionRepo.Create(ctx, session); err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
		msgChan <- domain.ChatStreamEvent{Type: "session_created", Content: session.ID.String()}
	}

	u.rateLimitMu.Lock()
	rl, exists := u.rateLimits[sessionID.String()]
	if !exists {
		rl = &sessionRateLimit{}
		u.rateLimits[sessionID.String()] = rl
	}
	u.rateLimitMu.Unlock()

	if time.Now().Before(rl.cooldownUntil) {
		waitSecs := int(time.Until(rl.cooldownUntil).Seconds())
		msgChan <- domain.ChatStreamEvent{Type: "cooldown", Content: fmt.Sprintf("%d", waitSecs)}
		msgChan <- domain.ChatStreamEvent{Type: "done", Done: true}
		return nil
	}

	userContext, err := u.buildUserContext(ctx, userID)
	if err != nil {
		logger.Warn("failed to build user context", zap.Error(err))
	}

	// RAG: augment context with vector search over this user's transactions.
	if rag := u.retrieveRelevantTransactions(ctx, userID, req.Message); rag != "" {
		userContext += "\nTRANSAKSI RELEVAN (hasil pencarian vektor):\n" + rag
	}

	systemPrompt := u.buildSystemPrompt(userContext) + u.buildToolsSystemPrompt()

	keys, err := u.getAvailableKeys(ctx, userID)
	if err != nil || len(keys) == 0 {
		msgChan <- domain.ChatStreamEvent{Type: "content", Content: "Anda belum menambahkan API key. Silakan tambahkan API key terlebih dahulu di menu API Keys."}
		msgChan <- domain.ChatStreamEvent{Type: "done", Done: true}
		return nil
	}

	history, _ := u.chatMessageRepo.ListBySession(ctx, *sessionID, userID)
	// Trim history to last 20 messages to prevent token overflow on small-context models
	if len(history) > 20 {
		history = history[len(history)-20:]
	}

	mode := req.Mode
	if mode == "" {
		mode = domain.LLMFast
	}

	key := u.findWorkingKeyByModality(ctx, keys, domain.ModalityText)
	if key == nil {
		key = u.findWorkingKey(ctx, keys)
	}
	if key == nil {
		msgChan <- domain.ChatStreamEvent{Type: "content", Content: "Semua API key Anda saat ini tidak aktif atau mencapai limit. Silakan periksa dan tambahkan key baru di menu API Keys."}
		msgChan <- domain.ChatStreamEvent{Type: "done", Done: true}
		return nil
	}

	decryptedKey, err := u.decryptAPIKey(key.EncryptedKey)
	if err != nil {
		msgChan <- domain.ChatStreamEvent{Type: "content", Content: "Gagal mendekripsi API key. Silakan hapus dan tambahkan kembali."}
		msgChan <- domain.ChatStreamEvent{Type: "done", Done: true}
		return nil
	}

	endpoints, ok := providerEndpoints[string(mode)][key.Provider]
	if !ok {
		endpoints = providerEndpoints[string(mode)]["gemini"]
	}
	if key.Model != "" {
		endpoints.model = key.Model
	}

	msgChan <- domain.ChatStreamEvent{
		Type:       "agent_step",
		StepLabel:  "Menganalisis perintah...",
		StepStatus: "running",
	}

	// Streaming content filter: strips <TOOL_CALL> and <TOOL_RESULT> blocks
	// so raw tool JSON is never exposed to the client
	var streamBuf string
	var insideToolTag bool
	var curTagName string

	onContent := func(chunk string) {
		streamBuf += chunk

		for {
			if insideToolTag {
				endTag := "</" + curTagName + ">"
				endIdx := strings.Index(streamBuf, endTag)
				if endIdx == -1 {
					break // still inside tag, buffer more
				}
				streamBuf = streamBuf[endIdx+len(endTag):]
				insideToolTag = false
				continue
			}

			callIdx := strings.Index(streamBuf, "<TOOL_CALL>")
			resultIdx := strings.Index(streamBuf, "<TOOL_RESULT>")

			tagIdx := -1
			tagLen := 0
			tagName := ""

			if callIdx >= 0 && (resultIdx < 0 || callIdx < resultIdx) {
				tagIdx = callIdx
				tagLen = len("<TOOL_CALL>")
				tagName = "TOOL_CALL"
			} else if resultIdx >= 0 {
				tagIdx = resultIdx
				tagLen = len("<TOOL_RESULT>")
				tagName = "TOOL_RESULT"
			}

			if tagIdx == -1 {
				// check for partial tag start ("<TOOL_") near the end
				ltIdx := strings.LastIndex(streamBuf, "<TOOL_")
				if ltIdx >= 0 && ltIdx > len(streamBuf)-20 {
					safe := streamBuf[:ltIdx]
					if safe != "" {
						msgChan <- domain.ChatStreamEvent{Type: "content", Content: safe}
					}
					streamBuf = streamBuf[ltIdx:]
				} else {
					ltIdx2 := strings.LastIndex(streamBuf, "<")
					if ltIdx2 >= 0 && ltIdx2 > len(streamBuf)-12 {
						safe := streamBuf[:ltIdx2]
						if safe != "" {
							msgChan <- domain.ChatStreamEvent{Type: "content", Content: safe}
						}
						streamBuf = streamBuf[ltIdx2:]
					} else if streamBuf != "" {
						msgChan <- domain.ChatStreamEvent{Type: "content", Content: streamBuf}
						streamBuf = ""
					}
				}
				break
			}

			if tagIdx > 0 {
				msgChan <- domain.ChatStreamEvent{Type: "content", Content: streamBuf[:tagIdx]}
			}
			streamBuf = streamBuf[tagIdx+tagLen:]
			insideToolTag = true
			curTagName = tagName
		}
	}

	flushStreamBuf := func() {
		if !insideToolTag && streamBuf != "" {
			msgChan <- domain.ChatStreamEvent{Type: "content", Content: streamBuf}
			streamBuf = ""
		}
	}

	msgChan <- domain.ChatStreamEvent{
		Type:       "agent_step",
		StepLabel:  "Menganalisis perintah...",
		StepStatus: "success",
	}

	var responseText string
	var callErr error
	currentKey := key
	currentDecrypted := decryptedKey
	currentEndpoints := endpoints

	for attempt := 0; attempt < len(keys); attempt++ {
		if currentKey == nil {
			break
		}

		currentDecrypted, callErr = u.decryptAPIKey(currentKey.EncryptedKey)
		if callErr != nil {
			currentKey = u.findNextActiveKey(keys, currentKey.ID)
			continue
		}

		var epOk bool
		currentEndpoints, epOk = providerEndpoints[string(mode)][currentKey.Provider]
		if !epOk {
			currentEndpoints = providerEndpoints[string(mode)]["gemini"]
		}
		if currentKey.Model != "" {
			currentEndpoints.model = currentKey.Model
		}

		responseText, callErr = u.callProvider(ctx, currentEndpoints, currentDecrypted, systemPrompt, history, req.Message, onContent)
		if callErr == nil {
			break
		}

		currentKey.Status = domain.KeyError
		errStr := callErr.Error()
		currentKey.LastError = &errStr
		u.keyRepo.Update(ctx, currentKey)

		msgChan <- domain.ChatStreamEvent{Type: "step", Step: fmt.Sprintf("Key %s error, coba key lain...", currentKey.Alias)}
		currentKey = u.findNextActiveKey(keys, currentKey.ID)
	}

	flushStreamBuf()

	if callErr != nil {
		msgChan <- domain.ChatStreamEvent{Type: "content", Content: fmt.Sprintf("Maaf, terjadi kesalahan saat menghubungi layanan AI: %s", callErr.Error())}
		msgChan <- domain.ChatStreamEvent{Type: "done", Done: true}
		return nil
	}

	// ─── Agent Loop ────────────────────────────────────────────────

	var fullResponse strings.Builder
	if currentEndpoints.model == "" {
		currentEndpoints.model = endpoints.model
	}
	if currentKey != nil && currentKey.Model != "" {
		currentEndpoints.model = currentKey.Model
	}

	agentHistory := make([]domain.ChatMessage, 0, len(history)+4)
	agentHistory = append(agentHistory, history...)
	agentHistory = append(agentHistory, domain.ChatMessage{
		SessionID: *sessionID,
		UserID:    userID,
		Role:      "user",
		Content:   req.Message,
	})

	for iteration := 0; iteration < 2; iteration++ {
		toolCalls := parseToolCalls(ctx, responseText)
		if len(toolCalls) == 0 {
			fullResponse.WriteString(responseText)
			break
		}

		cleanText := stripToolCalls(responseText)

		var results []toolExecResult
		for _, call := range toolCalls {
			msgChan <- domain.ChatStreamEvent{
				Type: "agent_step",
				StepLabel: u.getToolLabel(ctx, call.Tool),
				StepStatus: "running",
			}
			// Only send tool name to client — never expose raw parameters
			msgChan <- domain.ChatStreamEvent{
				Type: "tool_call",
				ToolCall: &domain.ToolCallInfo{
					Tool:       call.Tool,
					Parameters: nil,
				},
			}

			result := u.executeTool(ctx, userID, call)
			results = append(results, result)

			// Only expose chart data to client — hide transaction IDs & internal data
			sanitizedData := interface{}(nil)
			if result.Tool == "generate_chart" && result.Data != nil {
				sanitizedData = result.Data
			}
			msgChan <- domain.ChatStreamEvent{
				Type: "tool_result",
				ToolResult: &domain.ToolResultInfo{
					Tool:  result.Tool,
					Data:  sanitizedData,
					Error: result.Error,
				},
			}
			status := "success"
			if result.Error != "" {
				status = "error"
			}
			msgChan <- domain.ChatStreamEvent{
				Type: "agent_step",
				StepLabel: u.getToolLabel(ctx, call.Tool),
				StepStatus: status,
			}
		}

		// Skip re-invoking LLM if all tools are simple CRUD that don't need analysis
		if allToolsSimple(results) {
			// The LLM may have already generated text alongside tool calls (streamed via onContent).
			// Use that text if present; otherwise generate a fallback response.
			// IMPORTANT: Only write ONE of them to prevent duplication.
			cleanText = strings.TrimSpace(cleanText)
			if cleanText != "" {
				fullResponse.WriteString(cleanText)
			} else {
				fullResponse.WriteString(u.generateToolResponse(results))
			}
			break
		}

		agentHistory = append(agentHistory, domain.ChatMessage{
			SessionID: *sessionID,
			UserID:    userID,
			Role:      "assistant",
			Content:   responseText,
		})
		// Cap agent history to prevent token overflow
		if len(agentHistory) > 12 {
			agentHistory = agentHistory[len(agentHistory)-12:]
		}

		toolMessage := u.buildToolResultPrompt(results)
		responseText, err = u.callProvider(ctx, currentEndpoints, currentDecrypted, systemPrompt, agentHistory, toolMessage, onContent)
		flushStreamBuf()
		if err != nil {
			logger.Error("agent loop provider call failed", zap.Error(err))
			fullResponse.WriteString(fmt.Sprintf("\n\nMaaf, terjadi kesalahan saat memproses tool: %s", err.Error()))
			break
		}
	}

	userMsg := &domain.ChatMessage{
		SessionID: *sessionID,
		UserID:    userID,
		Role:      "user",
		Content:   req.Message,
	}
	u.chatMessageRepo.Create(ctx, userMsg)

	assistantContent := fullResponse.String()
	assistantMsg := &domain.ChatMessage{
		SessionID: *sessionID,
		UserID:    userID,
		Role:      "assistant",
		Content:   assistantContent,
	}
	u.chatMessageRepo.Create(ctx, assistantMsg)

	if _, ok := getTemplateResponse(msgLower); !ok {
		session, _ := u.chatSessionRepo.GetByID(ctx, *sessionID, userID)
		if session != nil && session.Title == "Percakapan baru" {
			title := u.generateTitle(ctx, req.Message)
			if title != "" && title != "Percakapan baru" {
				if err := u.chatSessionRepo.Update(ctx, &domain.ChatSession{
					ID:     *sessionID,
					UserID: userID,
					Title:  title,
				}); err == nil {
					msgChan <- domain.ChatStreamEvent{Type: "title_update", Content: title}
				}
			}
		}
	}

	u.rateLimitMu.Lock()
	rl, exists = u.rateLimits[sessionID.String()]
	if exists {
		rl.count++
	if rl.count >= 20 {
		rl.cooldownUntil = time.Now().Add(5 * time.Second)
		rl.count = 0
	}
	}
	u.rateLimitMu.Unlock()

	msgChan <- domain.ChatStreamEvent{Type: "done", Done: true}

	return nil
}

func (u *AIUsecase) findNextActiveKey(keys []domain.GeminiAPIKey, currentID uuid.UUID) *domain.GeminiAPIKey {
	found := false
	for i := range keys {
		if keys[i].ID == currentID {
			found = true
			continue
		}
		if found && keys[i].Status == domain.KeyActive {
			key := keys[i]
			return &key
		}
	}
	for i := range keys {
		if keys[i].ID != currentID && keys[i].Status == domain.KeyActive {
			key := keys[i]
			return &key
		}
	}
	return nil
}

// ─── Receipt Response Helpers ──────────────────────────────────

func stripThinkTags(text string) string {
	result := text
	for {
		start := strings.Index(result, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "</think>")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+len("</think>"):]
	}
	return strings.TrimSpace(result)
}

func extractJSONBlock(text string) string {
	// Try to find ```json ... ``` block
	start := strings.Index(text, "```json")
	if start == -1 {
		// Try ``` without language tag
		start = strings.Index(text, "```")
		if start >= 0 {
			// Skip the ``` marker
			start += 3
		}
	} else {
		start += 7 // len("```json")
	}

	if start >= 3 {
		end := strings.Index(text[start:], "```")
		if end >= 0 {
			return strings.TrimSpace(text[start : start+end])
		}
		// No closing ```, try to return everything after the marker
		return strings.TrimSpace(text[start:])
	}

	return text
}

// ─── Provider Call ─────────────────────────────────────────────

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	Stream    bool            `json:"stream"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type openAIVisionContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

type openAIVisionMessage struct {
	Role    string                `json:"role"`
	Content []openAIVisionContent `json:"content"`
}

type openAIVisionRequest struct {
	Model     string               `json:"model"`
	Messages  []openAIVisionMessage `json:"messages"`
	MaxTokens int                  `json:"max_tokens,omitempty"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inline_data,omitempty"`
}

type anthropicRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
	MaxTokens int             `json:"max_tokens"`
}

func (u *AIUsecase) callProvider(ctx context.Context, ep providerEndpoint, apiKey, systemPrompt string, history []domain.ChatMessage, userMessage string, onContent func(string)) (string, error) {
	switch {
	case strings.Contains(ep.baseURL, "generativelanguage"):
		return u.callGemini(ctx, ep, apiKey, systemPrompt, history, userMessage, onContent)
	case strings.Contains(ep.baseURL, "anthropic"):
		return u.callAnthropic(ctx, ep, apiKey, systemPrompt, history, userMessage, onContent)
	default:
		return u.callOpenAICompatible(ctx, ep, apiKey, systemPrompt, history, userMessage, onContent)
	}
}

func (u *AIUsecase) callOpenAICompatible(ctx context.Context, ep providerEndpoint, apiKey, systemPrompt string, history []domain.ChatMessage, userMessage string, onContent func(string)) (string, error) {
	messages := []openAIMessage{
		{Role: "system", Content: systemPrompt},
	}
	for _, h := range history {
		role := "user"
		if h.Role == "assistant" {
			role = "assistant"
		}
		messages = append(messages, openAIMessage{Role: role, Content: h.Content})
	}
	messages = append(messages, openAIMessage{Role: "user", Content: userMessage})

	body := openAIRequest{
		Model:     ep.model,
		Messages:  messages,
		Stream:    true,
		MaxTokens: 4096,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", ep.baseURL+ep.path, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "FinTrack/1.0")

	resp, err := u.httpClient.Do( req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var fullText strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				fullText.WriteString(choice.Delta.Content)
				if onContent != nil {
					onContent(choice.Delta.Content)
				}
			}
		}
	}

	if fullText.Len() == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return fullText.String(), nil
}

func (u *AIUsecase) callGemini(ctx context.Context, ep providerEndpoint, apiKey, systemPrompt string, history []domain.ChatMessage, userMessage string, onContent func(string)) (string, error) {
	var contents []geminiContent

	for _, h := range history {
		role := "user"
		if h.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: h.Content}},
		})
	}

	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: []geminiPart{{Text: systemPrompt + "\n\n" + userMessage}},
	})

	body := geminiRequest{Contents: contents}
	jsonBody, _ := json.Marshal(body)

	streamPath := strings.Replace(ep.path, ":generateContent", ":streamGenerateContent", 1)
	if streamPath == ep.path {
		streamPath = ep.path + "?alt=sse"
	}
	url := ep.baseURL + streamPath + "?key=" + apiKey + "&alt=sse"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.httpClient.Do( req)
	if err != nil {
		return "", fmt.Errorf("Gemini call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var fullText strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var chunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		for _, c := range chunk.Candidates {
			for _, p := range c.Content.Parts {
				if p.Text != "" {
					fullText.WriteString(p.Text)
					if onContent != nil {
						onContent(p.Text)
					}
				}
			}
		}
	}

	if fullText.Len() == 0 {
		return "", fmt.Errorf("empty Gemini response")
	}

	return fullText.String(), nil
}

func (u *AIUsecase) callAnthropic(ctx context.Context, ep providerEndpoint, apiKey, systemPrompt string, history []domain.ChatMessage, userMessage string, onContent func(string)) (string, error) {
	messages := []openAIMessage{}

	for _, h := range history {
		role := "user"
		if h.Role == "assistant" {
			role = "assistant"
		}
		messages = append(messages, openAIMessage{Role: role, Content: h.Content})
	}
	messages = append(messages, openAIMessage{Role: "user", Content: userMessage})

	body := anthropicRequest{
		Model:     ep.model,
		Messages:  messages,
		System:    systemPrompt,
		MaxTokens: 4096,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", ep.baseURL+ep.path, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := u.httpClient.Do( req)
	if err != nil {
		return "", fmt.Errorf("Anthropic call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Anthropic returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	fullText := ""
	if len(result.Content) > 0 {
		fullText = result.Content[0].Text
	}

	if fullText == "" {
		return "", fmt.Errorf("empty Anthropic response")
	}

	if onContent != nil {
		onContent(fullText)
	}

	return fullText, nil
}

// ─── Session CRUD ──────────────────────────────────────────────

func (u *AIUsecase) CreateSession(ctx context.Context, userID uuid.UUID, title, source string) (*domain.ChatSession, error) {
	if source == "" {
		source = "chatbot"
	}
	s := &domain.ChatSession{UserID: userID, Title: title, Source: source}
	if err := u.chatSessionRepo.Create(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (u *AIUsecase) ListSessions(ctx context.Context, userID uuid.UUID, source string) ([]domain.ChatSession, error) {
	if source != "" {
		return u.chatSessionRepo.ListByUserAndSource(ctx, userID, source)
	}
	return u.chatSessionRepo.ListByUser(ctx, userID)
}

func (u *AIUsecase) UpdateSession(ctx context.Context, userID, id uuid.UUID, title string) (*domain.ChatSession, error) {
	s, err := u.chatSessionRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	s.Title = title
	if err := u.chatSessionRepo.Update(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (u *AIUsecase) DeleteSession(ctx context.Context, userID, id uuid.UUID) error {
	return u.chatSessionRepo.Delete(ctx, id, userID)
}

func (u *AIUsecase) ListMessages(ctx context.Context, userID, sessionID uuid.UUID) ([]domain.ChatMessage, error) {
	return u.chatMessageRepo.ListBySession(ctx, sessionID, userID)
}

// ─── Agent Helpers ─────────────────────────────────────────────

func stripToolCalls(text string) string {
	result := text
	for {
		start := strings.Index(result, "<TOOL_CALL>")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "</TOOL_CALL>")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+len("</TOOL_CALL>"):]
	}
	return strings.TrimSpace(result)
}

// Simple tools that don't need LLM follow-up to generate a response.
var simpleTools = map[string]bool{
	"create_transaction": true,
}

func allToolsSimple(results []toolExecResult) bool {
	for _, r := range results {
		if !simpleTools[r.Tool] || r.Error != "" {
			return false
		}
	}
	return len(results) > 0
}

func (u *AIUsecase) generateToolResponse(results []toolExecResult) string {
	var sb strings.Builder
	for _, r := range results {
		switch r.Tool {
		case "create_transaction":
			data, ok := r.Data.(map[string]interface{})
			if !ok {
				sb.WriteString("Done! Transaksi berhasil dicatat.")
				continue
			}
			amount, _ := data["amount"].(float64)
			desc, _ := data["description"].(string)
			txType, _ := data["type"].(string)
			typeLabel := "Pengeluaran"
			switch txType {
			case "income":
				typeLabel = "Pemasukan"
			case "savings":
				typeLabel = "Tabungan"
			}
			formatted := fmt.Sprintf("Rp %.0f", amount)
			if desc != "" {
				sb.WriteString(fmt.Sprintf("Done! **%s** %s sebesar %s berhasil dicatat.", desc, typeLabel, formatted))
			} else {
				sb.WriteString(fmt.Sprintf("Done! %s sebesar %s berhasil dicatat.", typeLabel, formatted))
			}
		default:
			sb.WriteString("Done! Perintah berhasil dijalankan.")
		}
	}
	return sb.String()
}

func (u *AIUsecase) buildToolResultPrompt(results []toolExecResult) string {
	var sb strings.Builder
	sb.WriteString("Berikut adalah hasil eksekusi tool yang diminta:\n\n")
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("Tool %d: %s\n", i+1, r.Tool))
		if r.Error != "" {
			sb.WriteString(fmt.Sprintf("Error: %s\n", r.Error))
		} else {
			data, _ := json.MarshalIndent(r.Data, "", "  ")
			sb.WriteString(fmt.Sprintf("Hasil: %s\n", string(data)))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("Berdasarkan hasil di atas, berikan respons yang informatif dan bermanfaat kepada pengguna dalam bahasa Indonesia. Jangan memanggil tool lagi kecuali benar-benar diperlukan.")
	return sb.String()
}

func (u *AIUsecase) getToolLabel(ctx context.Context, tool string) string {
	switch tool {
	case "create_transaction":
		return "Mencatat transaksi..."
	case "get_spending_summary":
		return "Mengambil ringkasan pengeluaran..."
	case "get_category_expenses":
		return "Menganalisis pengeluaran per kategori..."
	case "get_transactions":
		return "Mencari transaksi..."
	case "get_categories":
		return "Memuat daftar kategori..."
	case "get_finance_insight":
		return "Menganalisis data keuangan..."
	case "generate_chart":
		return "Membuat visualisasi grafik..."
	default:
		return fmt.Sprintf("Menjalankan %s...", tool)
	}
}

func (u *AIUsecase) truncateTitle(ctx context.Context, msg string) string {
	runes := []rune(msg)
	if len(runes) > 50 {
		return string(runes[:50]) + "..."
	}
	return msg
}

var fillerWords = []string{
	"hai", "halo", "hi", "helo", "hey",
	"saya", "aku", "gue", "gw", "kami",
	"tolong", "dong", "ya", "yuk", "ayo", "mari",
	"bisa", "ingin", "mau", "hendak", "akan",
	"untuk", "saja", "aja", "kak", "bang", "mas", "pak", "bu",
	"permisi", "maaf", "minta", "tanya",
	"itu", "ini", "dia", "mereka", "kita",
}

func (u *AIUsecase) generateTitle(ctx context.Context, message string) string {
	msg := strings.TrimSpace(message)

	// Remove leading commands like "catat", "tolong catat", "buat", etc
	cleaned := msg
	lower := strings.ToLower(cleaned)
	prefixes := []string{"tolong catat", "catatkan", "catat", "buatkan", "buat", "tambahkan", "tambah", "input", "masukan"}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			cleaned = strings.TrimSpace(cleaned[len(p):])
			break
		}
	}

	words := strings.Fields(cleaned)
	var result []string
	for _, w := range words {
		wLower := strings.ToLower(w)
		// Skip filler words
		isFiller := false
		for _, f := range fillerWords {
			if wLower == f {
				isFiller = true
				break
			}
		}
		// Skip numbers and currency
		if !isFiller {
			_, err := strconv.ParseFloat(w, 64)
			hasRp := strings.HasPrefix(wLower, "rp")
			hasPercent := strings.HasSuffix(wLower, "%")
			if err != nil || hasRp || hasPercent {
				continue
			}
			result = append(result, w)
		}
	}

	title := strings.Join(result, " ")
	if title == "" {
		title = msg
	}
	if len(title) > 0 {
		title = strings.ToUpper(title[:1]) + title[1:]
	}

	runes := []rune(title)
	if len(runes) > 50 {
		return string(runes[:50]) + "..."
	}
	return title
}

// ─── Receipt Analysis ──────────────────────────────────────────

func (u *AIUsecase) AnalyzeReceipt(ctx context.Context, userID uuid.UUID, imageData []byte, contentType string) (map[string]interface{}, error) {
	if len(imageData) == 0 || len(imageData) > 10*1024*1024 {
		return nil, fmt.Errorf("invalid image data")
	}

	keys, err := u.getAvailableKeys(ctx, userID)
	if err != nil || len(keys) == 0 {
		return map[string]interface{}{
			"success": false,
			"message": "Silakan tambahkan API key terlebih dahulu untuk menggunakan fitur ini.",
			"items":   []interface{}{},
		}, nil
	}

	base64Image := base64Encode(imageData)
	prompt := fmt.Sprintf(`Anda adalah asisten analisis struk keuangan. Ekstrak informasi dari gambar struk berikut.

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
}`)

	key := u.findWorkingKeyByModality(ctx, keys, domain.ModalityVision)
	if key == nil {
		key = u.findWorkingKey(ctx, keys)
	}
	if key == nil {
		return map[string]interface{}{
			"success": false,
			"message": "Semua API key tidak aktif. Periksa di menu API Keys.",
			"items":   []interface{}{},
		}, nil
	}

	var text string
	var lastErr error
	currentKey := key

	for attempt := 0; attempt < len(keys); attempt++ {
		if currentKey == nil {
			break
		}

		decryptedKey, decryptErr := u.decryptAPIKey(currentKey.EncryptedKey)
		if decryptErr != nil {
			lastErr = decryptErr
			currentKey = u.findNextActiveKey(keys, currentKey.ID)
			continue
		}

		endpoints, ok := providerEndpoints["fast"][currentKey.Provider]
		if !ok {
			endpoints = providerEndpoints["fast"]["gemini"]
		}

		model := endpoints.model
		if currentKey.Model != "" {
			model = currentKey.Model
		}

		switch {
		case strings.Contains(endpoints.baseURL, "generativelanguage"):
			text, err = u.analyzeReceiptGemini(ctx, endpoints, decryptedKey, prompt, base64Image, contentType)
		case strings.Contains(endpoints.baseURL, "anthropic"):
			text, err = u.analyzeReceiptAnthropic(ctx, endpoints, model, decryptedKey, prompt, base64Image, contentType)
		default:
			text, err = u.analyzeReceiptOpenAI(ctx, endpoints, model, decryptedKey, prompt, base64Image, contentType)
		}

		if err == nil {
			break
		}

		lastErr = err
		currentKey.Status = domain.KeyError
		errStr := err.Error()
		currentKey.LastError = &errStr
		u.keyRepo.Update(ctx, currentKey)

		currentKey = u.findNextActiveKey(keys, currentKey.ID)
	}

	if err != nil {
		return u.receiptErrorResult(lastErr), nil
	}

	if text == "" {
		return map[string]interface{}{
			"success": false,
			"message": "Gambar tidak dikenali sebagai struk. Pastikan foto jelas dan terbaca.",
			"items":   []interface{}{},
		}, nil
	}

	// Clean AI response: strip <think> tags, extract JSON from ```json blocks
	cleaned := text
	cleaned = stripThinkTags(cleaned)
	cleaned = extractJSONBlock(cleaned)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		// Try fallback: parse the original raw text directly
		if err2 := json.Unmarshal([]byte(text), &result); err2 != nil {
			result = map[string]interface{}{
				"success":     true,
				"raw_text":    text,
				"items":       []interface{}{},
				"total":       0,
				"merchant":    "",
				"category":    "Lainnya",
				"type":        "expense",
			}
		}
	}

	if _, ok := result["type"]; !ok {
		result["type"] = "expense"
	}

	result["success"] = true
	return result, nil
}

func (u *AIUsecase) receiptErrorResult(err error) map[string]interface{} {
	msg := err.Error()
	msgLower := strings.ToLower(msg)

	apiMsg := extractAPIErrorMessage(msg)

	friendly := "Gagal menganalisis struk. Silakan coba lagi dengan file yang lebih jelas."

	switch {
	case strings.Contains(msg, "429") || strings.Contains(msg, "429 ") ||
		strings.Contains(msgLower, "rate limit") || strings.Contains(msgLower, "rate_limit") ||
		strings.Contains(msgLower, "quota") || strings.Contains(msgLower, "too many requests") ||
		strings.Contains(msgLower, "insufficient_quota") || strings.Contains(msgLower, "insufficient quota") ||
		strings.Contains(msgLower, "exceeded"):
		friendly = "Kuota API telah habis atau batas permintaan terlampaui. Silakan ganti API Key atau tunggu beberapa saat."

	case strings.Contains(msg, "401") || strings.Contains(msg, "403") ||
		strings.Contains(msgLower, "unauthorized") || strings.Contains(msgLower, "forbidden") ||
		strings.Contains(msgLower, "api key") || strings.Contains(msgLower, "invalid key") ||
		strings.Contains(msgLower, "not valid") || strings.Contains(msgLower, "permission") ||
		strings.Contains(msgLower, "auth"):
		friendly = "API Key tidak valid. Periksa kembali API Key di menu API Keys."

	case strings.Contains(msgLower, "model not found") || strings.Contains(msgLower, "model_not_found") ||
		strings.Contains(msgLower, "does not exist"):
		friendly = "Model tidak ditemukan. Periksa kembali nama model di pengaturan API Key."

	case strings.Contains(msgLower, "does not support") || strings.Contains(msgLower, "image_url") ||
		strings.Contains(msgLower, "vision") || strings.Contains(msgLower, "image input") ||
		strings.Contains(msgLower, "not support") || strings.Contains(msgLower, "image is not supported") ||
		strings.Contains(msgLower, "cannot parse image") || strings.Contains(msgLower, "invalid image") ||
		strings.Contains(msgLower, "unable to process image"):
		friendly = "Model ini tidak mendukung analisis gambar. Silakan ganti ke model vision di menu API Keys (contoh: meta-llama/llama-4-scout-17b-16e-instruct, qwen/qwen3.6-27b, gpt-4o-mini)."

	case strings.Contains(msgLower, "decommissioned") || strings.Contains(msgLower, "deprecated") ||
		strings.Contains(msgLower, "no longer supported") || strings.Contains(msgLower, "has been discontinued"):
		friendly = "Model sudah tidak tersedia (decommissioned/deprecated). Silakan ganti model di menu API Keys."

	case strings.Contains(msgLower, "not a valid model") || strings.Contains(msgLower, "invalid model"):
		friendly = "Model ID tidak valid. Periksa kembali nama model di pengaturan API Key."

	case strings.Contains(msg, "400") && strings.Contains(msgLower, "current quota"):
		friendly = "Kuota model saat ini tidak mencukupi untuk memproses gambar. Silakan ganti model atau API Key."

	case strings.Contains(msgLower, "timeout") || strings.Contains(msgLower, "timed out") ||
		strings.Contains(msgLower, "deadline"):
		friendly = "Waktu analisis habis. Pastikan file tidak terlalu besar dan coba lagi."
	}

	if apiMsg != "" && !strings.Contains(msgLower, "html") {
		message := friendly
		return map[string]interface{}{
			"success": false,
			"message": message,
			"detail":  apiMsg,
			"items":   []interface{}{},
		}
	}

	return map[string]interface{}{
		"success": false,
		"message": friendly,
		"items":   []interface{}{},
	}
}

// extractAPIErrorMessage tries to extract the provider's error message from the API response body.
// The error format from helpers: "API returned 400: <json body>"
func extractAPIErrorMessage(msg string) string {
	idx := strings.Index(msg, ": ")
	if idx < 0 {
		return ""
	}
	body := msg[idx+2:]
	if body == "" {
		return ""
	}

	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err == nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}

	var geminiErr struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &geminiErr); err == nil && geminiErr.Error.Message != "" {
		return geminiErr.Error.Message
	}

	return ""
}

func (u *AIUsecase) analyzeReceiptGemini(ctx context.Context, ep providerEndpoint, apiKey, prompt, base64Image, contentType string) (string, error) {
	body := geminiRequest{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: prompt},
					{InlineData: &geminiInlineData{MimeType: contentType, Data: base64Image}},
				},
			},
		},
	}

	jsonBody, _ := json.Marshal(body)
	url := ep.baseURL + ep.path + "?key=" + apiKey
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.httpClient.Do( req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", nil
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (u *AIUsecase) analyzeReceiptOpenAI(ctx context.Context, ep providerEndpoint, model, apiKey, prompt, base64Image, contentType string) (string, error) {
	body := openAIVisionRequest{
		Model: model,
		Messages: []openAIVisionMessage{
			{
				Role: "user",
				Content: []openAIVisionContent{
					{Type: "text", Text: prompt},
					{Type: "image_url", ImageURL: &openAIImageURL{URL: "data:" + contentType + ";base64," + base64Image}},
				},
			},
		},
		MaxTokens: 4096,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", ep.baseURL+ep.path, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := u.httpClient.Do( req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", nil
	}

	return openAIResp.Choices[0].Message.Content, nil
}

func (u *AIUsecase) analyzeReceiptAnthropic(ctx context.Context, ep providerEndpoint, model, apiKey, prompt, base64Image, contentType string) (string, error) {
	mediaType := contentType
	if mediaType == "" {
		mediaType = "image/jpeg"
	}

	body := map[string]interface{}{
		"model":      model,
		"max_tokens": 4096,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": prompt},
					{
						"type": "image",
						"source": map[string]interface{}{
							"type":       "base64",
							"media_type": mediaType,
							"data":       base64Image,
						},
					},
				},
			},
		},
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", ep.baseURL+ep.path, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := u.httpClient.Do( req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", nil
	}

	return anthropicResp.Content[0].Text, nil
}

func (u *AIUsecase) CreateTransaction(ctx context.Context, tx *domain.Transaction) error {
	return u.txRepo.Create(ctx, tx)
}

// ─── Voice Processing ──────────────────────────────────────────

func (u *AIUsecase) ProcessVoice(ctx context.Context, userID uuid.UUID, audioBase64 string) (map[string]interface{}, error) {
	if len(audioBase64) > 5*1024*1024 {
		return nil, fmt.Errorf("audio too large")
	}

	keys, err := u.getAvailableKeys(ctx, userID)
	if err != nil || len(keys) == 0 {
		return map[string]interface{}{
			"success":    false,
			"message":    "Silakan tambahkan API key terlebih dahulu.",
			"transcript": "",
		}, nil
	}

	currentKey := u.findWorkingKeyByModality(ctx, keys, domain.ModalityAudio)
	if currentKey == nil {
		currentKey = u.findWorkingKey(ctx, keys)
	}
	if currentKey == nil {
		return map[string]interface{}{
			"success":    false,
			"message":    "Semua API key tidak aktif.",
			"transcript": "",
		}, nil
	}

	prompt := "Transkripsikan audio berikut dengan akurat. Hanya keluarkan teks hasil transkripsi, tanpa penjelasan tambahan."

	var transcript string
	var lastErr error

	for attempt := 0; attempt < len(keys); attempt++ {
		if currentKey == nil {
			break
		}

		decryptedKey, decryptErr := u.decryptAPIKey(currentKey.EncryptedKey)
		if decryptErr != nil {
			lastErr = decryptErr
			currentKey = u.findNextActiveKey(keys, currentKey.ID)
			continue
		}

		endpoints, ok := providerEndpoints["fast"][currentKey.Provider]
		if !ok {
			endpoints = providerEndpoints["fast"]["gemini"]
		}

		model := endpoints.model
		if currentKey.Model != "" {
			model = currentKey.Model
		}

		switch {
		case strings.Contains(endpoints.baseURL, "generativelanguage"):
			transcript, lastErr = u.voiceProcessGemini(ctx, endpoints, decryptedKey, prompt, audioBase64)
		default:
			if strings.Contains(endpoints.baseURL, "openai") || strings.Contains(endpoints.baseURL, "groq") || strings.Contains(endpoints.baseURL, "openrouter") {
				transcript, lastErr = u.voiceProcessOpenAI(ctx, endpoints, model, decryptedKey, prompt, audioBase64)
			} else {
				lastErr = fmt.Errorf("provider %s does not support voice processing", currentKey.Provider)
			}
		}

		if lastErr == nil {
			return map[string]interface{}{
				"success":    true,
				"transcript": transcript,
			}, nil
		}

		currentKey.Status = domain.KeyError
		errStr := lastErr.Error()
		currentKey.LastError = &errStr
		u.keyRepo.Update(ctx, currentKey)

		currentKey = u.findNextActiveKey(keys, currentKey.ID)
	}

	return map[string]interface{}{
		"success":    false,
		"message":    "Semua API key gagal. Periksa key atau coba lagi nanti.",
		"transcript": "",
	}, nil
}

func (u *AIUsecase) voiceProcessGemini(ctx context.Context, ep providerEndpoint, apiKey, prompt, audioBase64 string) (string, error) {
	// Detect MIME type from base64 header
	mimeType := "audio/webm"
	if len(audioBase64) > 20 {
		header := audioBase64[:20]
		if strings.Contains(header, "UklGR") {
			mimeType = "audio/wav"
		} else if strings.Contains(header, "Ogg") || strings.Contains(header, "T2dn") {
			mimeType = "audio/ogg"
		} else if strings.Contains(header, "AAAA") || strings.Contains(header, "fLCG") {
			mimeType = "audio/mp4"
		}
	}

	body := geminiRequest{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: prompt},
					{InlineData: &geminiInlineData{MimeType: mimeType, Data: audioBase64}},
				},
			},
		},
	}

	jsonBody, _ := json.Marshal(body)
	url := ep.baseURL + ep.path + "?key=" + apiKey
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.httpClient.Do( req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", nil
}

func (u *AIUsecase) voiceProcessOpenAI(ctx context.Context, ep providerEndpoint, model, apiKey, prompt, audioBase64 string) (string, error) {
	// Use OpenAI-compatible audio input format with vision-style multimodal messages
	type audioContentPart struct {
		Type     string `json:"type"`
		Text     string `json:"text,omitempty"`
		InputAudio *struct {
			Data   string `json:"data"`
			Format string `json:"format"`
		} `json:"input_audio,omitempty"`
	}

	type audioMessage struct {
		Role    string             `json:"role"`
		Content []audioContentPart `json:"content"`
	}

	type audioRequest struct {
		Model     string          `json:"model"`
		Messages  []audioMessage  `json:"messages"`
		MaxTokens int             `json:"max_tokens,omitempty"`
	}

	// Detect audio format from the base64 data header
	audioFormat := "wav"
	if len(audioBase64) > 20 {
		header := audioBase64[:20]
		if strings.Contains(header, "UklGR") { // RIFF/WAV base64 starts with UklGR
			audioFormat = "wav"
		} else if strings.Contains(header, "Ogg") || strings.Contains(header, "T2dn") { // OGG/Opus
			audioFormat = "opus"
		} else if strings.Contains(header, "AAAA") || strings.Contains(header, "fLCG") { // MP4/M4A
			audioFormat = "mp4"
		} else {
			audioFormat = "webm"
		}
	}

	body := audioRequest{
		Model: model,
		Messages: []audioMessage{
			{
				Role: "user",
				Content: []audioContentPart{
					{Type: "text", Text: prompt},
					{Type: "input_audio", InputAudio: &struct {
						Data   string `json:"data"`
						Format string `json:"format"`
					}{Data: audioBase64, Format: audioFormat}},
				},
			},
		},
		MaxTokens: 4096,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", ep.baseURL+ep.path, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := u.httpClient.Do( req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openAIResp.Choices) > 0 {
		return openAIResp.Choices[0].Message.Content, nil
	}
	return "", nil
}
