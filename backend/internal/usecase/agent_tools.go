package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/pkg/logger"
)

// ─── Tool Definitions ──────────────────────────────────────────

type toolParam struct {
	Type        string                  `json:"type"`
	Description string                  `json:"description,omitempty"`
	Enum        []string                `json:"enum,omitempty"`
	Properties  map[string]toolParam    `json:"properties,omitempty"`
	Required    []string                `json:"required,omitempty"`
	Items       *map[string]interface{} `json:"items,omitempty"`
}

type toolDef struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Parameters  toolParam `json:"parameters"`
}

var agentTools = []toolDef{
	{
		Name:        "get_categories",
		Description: "Get all available income, expense, or savings categories",
		Parameters: toolParam{
			Type: "object",
			Properties: map[string]toolParam{
				"type": {Type: "string", Description: "Category type: 'income', 'expense', or 'savings'"},
			},
			Required: []string{},
		},
	},
	{
		Name:        "get_spending_summary",
		Description: "Get spending summary for a specific period (monthly, yearly)",
		Parameters: toolParam{
			Type: "object",
			Properties: map[string]toolParam{
				"period": {Type: "string", Description: "Period type: 'this_month', 'last_month', 'this_year', 'last_3_months'"},
			},
			Required: []string{"period"},
		},
	},
	{
		Name:        "get_category_expenses",
		Description: "Get expense breakdown by category for a period",
		Parameters: toolParam{
			Type: "object",
			Properties: map[string]toolParam{
				"period": {Type: "string", Description: "Period: 'this_month', 'last_month', 'this_year'"},
			},
			Required: []string{"period"},
		},
	},
	{
		Name:        "get_transactions",
		Description: "Search or list recent transactions with optional filters",
		Parameters: toolParam{
			Type: "object",
			Properties: map[string]toolParam{
				"type":     {Type: "string", Description: "Filter by type: 'income', 'expense', or 'savings'", Enum: []string{"income", "expense", "savings"}},
				"category": {Type: "string", Description: "Filter by category name"},
				"search":   {Type: "string", Description: "Search keyword in description"},
				"limit":    {Type: "string", Description: "Max results (default 10)"},
			},
			Required: []string{},
		},
	},
	{
		Name:        "create_transaction",
		Description: "Create a new transaction (income, expense, or savings) after user confirmation",
		Parameters: toolParam{
			Type: "object",
			Properties: map[string]toolParam{
				"type":        {Type: "string", Description: "Transaction type", Enum: []string{"income", "expense", "savings"}},
				"amount":      {Type: "string", Description: "Amount in Rupiah (numbers only, e.g. 50000)"},
				"category":    {Type: "string", Description: "Category name (must match existing category)"},
				"description": {Type: "string", Description: "Optional description"},
				"date":        {Type: "string", Description: "Date in YYYY-MM-DD format (default today)"},
			},
			Required: []string{"type", "amount", "category"},
		},
	},
	{
		Name:        "get_finance_insight",
		Description: "Get personalized finance analysis, investment tips, and recommendations based on user data",
		Parameters: toolParam{
			Type: "object",
			Properties: map[string]toolParam{
				"topic": {Type: "string", Description: "Topic: 'saving', 'investment', 'budget', 'debt', 'general'"},
			},
			Required: []string{"topic"},
		},
	},
	{
		Name:        "generate_chart",
		Description: "Generate chart data for visualization of income, expense, and savings over a date range. Call this when user asks for charts, graphs, or visual comparisons of their financial data.",
		Parameters: toolParam{
			Type: "object",
			Properties: map[string]toolParam{
				"start_date":    {Type: "string", Description: "Start date in YYYY-MM-DD format"},
				"end_date":      {Type: "string", Description: "End date in YYYY-MM-DD format"},
				"compare_start": {Type: "string", Description: "Optional: comparison start date in YYYY-MM-DD format"},
				"compare_end":   {Type: "string", Description: "Optional: comparison end date in YYYY-MM-DD format"},
			},
			Required: []string{"start_date", "end_date"},
		},
	},
}

func (u *AIUsecase) buildToolsSystemPrompt() string {
	var sb strings.Builder
	sb.WriteString("\n\n=== TOOLS YANG TERSEDIA ===\n")
	sb.WriteString("Anda dapat menggunakan tools berikut untuk membantu pengguna. Keluarkan panggilan tool dalam format JSON:\n")
	sb.WriteString("<TOOL_CALL>\n{\"tool\": \"nama_tool\", \"parameters\": {...}}\n</TOOL_CALL>\n\n")
	sb.WriteString("Setelah tool dieksekusi, hasilnya akan diberikan kepada Anda dalam format:\n")
	sb.WriteString("<TOOL_RESULT>\n{\"tool\": \"nama_tool\", \"data\": ...}\n</TOOL_RESULT>\n\n")
	sb.WriteString("Tools:\n")
	for i, t := range agentTools {
		sb.WriteString(fmt.Sprintf("\n%d. **%s**: %s\n", i+1, t.Name, t.Description))
		params := t.Parameters.Properties
		if len(params) > 0 {
			sb.WriteString("   Parameters:\n")
			for name, p := range params {
				sb.WriteString(fmt.Sprintf("   - `%s` (%s): %s\n", name, p.Type, p.Description))
			}
		}
	}
	sb.WriteString("\n⚠️ PENTING - BACA INI DULU:\n")
	sb.WriteString("- Kalo user nyuruh **catat/buat/tambah** transaksi, KAMU HARUS panggil tool <TOOL_CALL>. Kalo cuma bilang \"Done!\" doang tanpa <TOOL_CALL>, transaksi GAK AKAN tersimpan dan user bakal komplen!\n")
	sb.WriteString("- Kalo user minta **grafik/chart/visualisasi/diagram** data keuangan, KAMU HARUS panggil tool `generate_chart` dengan <TOOL_CALL>. JANGAN cuma jawab pake teks doang, panggil dulu tool generate_chart biar grafisnya muncul!\n")
	sb.WriteString("- Contoh yang BENAR:\n")
	sb.WriteString("  User: \"catat pengeluaran kopi 25rb\"\n")
	sb.WriteString("  Kamu: <TOOL_CALL>\n")
	sb.WriteString("  {\"tool\": \"create_transaction\", \"parameters\": {\"type\": \"expense\", \"amount\": \"25000\", \"category\": \"Kopi\", \"description\": \"kopi\", \"date\": \"2026-06-23\"}}\n")
	sb.WriteString("  </TOOL_CALL>\n")
	sb.WriteString("  (tunggu TOOL_RESULT, lalu jawab \"Done! Transaksi...\")\n")
	sb.WriteString("- Contoh tabungan:\n")
	sb.WriteString("  User: \"catat tabungan 500rb\"\n")
	sb.WriteString("  Kamu: <TOOL_CALL>\n")
	sb.WriteString("  {\"tool\": \"create_transaction\", \"parameters\": {\"type\": \"savings\", \"amount\": \"500000\", \"category\": \"Tabungan Harian\", \"description\": \"tabungan\", \"date\": \"2026-06-23\"}}\n")
	sb.WriteString("  </TOOL_CALL>\n")
	sb.WriteString("  (tunggu TOOL_RESULT, lalu jawab \"Done! Transaksi tabungan tersimpan\")\n")
	sb.WriteString("- Contoh beberapa transaksi sekaligus (natural language / curhatan transaksi):\n")
	sb.WriteString("  User: \"saya abis bayar kontrakan 625k, listrik 25k, galon 5k, dan tadi dapat dari kantor 200k\"\n")
	sb.WriteString("  Kamu panggil tool create_transaction untuk SETIAP item:\n")
	sb.WriteString("  <TOOL_CALL>\n")
	sb.WriteString("  {\"tool\": \"create_transaction\", \"parameters\": {\"type\": \"expense\", \"amount\": \"625000\", \"category\": \"Tempat Tinggal\", \"description\": \"Bayar kontrakan\"}}\n")
	sb.WriteString("  </TOOL_CALL>\n")
	sb.WriteString("  <TOOL_CALL>\n")
	sb.WriteString("  {\"tool\": \"create_transaction\", \"parameters\": {\"type\": \"expense\", \"amount\": \"25000\", \"category\": \"Tagihan & Utilitas\", \"description\": \"Listrik\"}}\n")
	sb.WriteString("  </TOOL_CALL>\n")
	sb.WriteString("  <TOOL_CALL>\n")
	sb.WriteString("  {\"tool\": \"create_transaction\", \"parameters\": {\"type\": \"expense\", \"amount\": \"5000\", \"category\": \"Kebutuhan Pokok\", \"description\": \"Galon\"}}\n")
	sb.WriteString("  </TOOL_CALL>\n")
	sb.WriteString("  <TOOL_CALL>\n")
	sb.WriteString("  {\"tool\": \"create_transaction\", \"parameters\": {\"type\": \"income\", \"amount\": \"200000\", \"category\": \"Gaji\", \"description\": \"Dapat dari kantor\"}}\n")
	sb.WriteString("  </TOOL_CALL>\n")
	sb.WriteString("  (tunggu semua TOOL_RESULT, lalu jawab dengan rekap ramah rincian pengeluaran & pemasukan)\n")
	sb.WriteString("- Contoh visualisasi:\n")
	sb.WriteString("  User: \"tampilkan grafik pengeluaran bulan ini\"\n")
	sb.WriteString("  Kamu: <TOOL_CALL>\n")
	sb.WriteString("  {\"tool\": \"generate_chart\", \"parameters\": {\"type\": \"expense\", \"start_date\": \"2026-06-01\", \"end_date\": \"2026-06-24\", \"granularity\": \"week\"}}\n")
	sb.WriteString("  </TOOL_CALL>\n")
	sb.WriteString("  (tunggu TOOL_RESULT, lalu tampilkan grafik dan jelaskan insight-nya secara singkat)\n")
	sb.WriteString("- Contoh pertanyaan analisis / rekomendasi investasi / siklus keuangan:\n")
	sb.WriteString("  User: \"berikan saya rekomendasi investasi berdasarkan siklus keuangan 1 tahun terakhir\"\n")
	sb.WriteString("  Kamu bisa langsung analisis dari 'Siklus Keuangan 1 Tahun Terakhir' di DATA PENGGUNA SAAT INI atau panggil:\n")
	sb.WriteString("  <TOOL_CALL>\n")
	sb.WriteString("  {\"tool\": \"get_finance_insight\", \"parameters\": {\"topic\": \"investment\"}}\n")
	sb.WriteString("  </TOOL_CALL>\n")
	sb.WriteString("  (lalu berikan analisis mendalam dan rekomendasi alokasi investasi bertingkat berbasis angka riil pengguna)\n")
	sb.WriteString("- Kalo kategori gak ditemukan, auto-create aja kategori barunya\n")
	sb.WriteString("- Kalo parameter kurang, pake default yg masuk akal (misal date: hari ini, description: nama kategori)\n")
	sb.WriteString("- Jangan panggil tool yang gak perlu. Tapi kalo user minta catat transaksi, WAJIB panggil tool.\n")
	sb.WriteString("- Setelah dapet TOOL_RESULT, langsung kasi tau user hasilnya dengan bahasa alami\n")
	sb.WriteString("- JANGAN PERNAH menyebutkan nama tool internal seperti create_transaction, get_finance_insight, get_spending_summary, dll ke user. Bicaralah secara alami.\n")
	return sb.String()
}

// ─── Tool Call Parsing ─────────────────────────────────────────

type toolCallRequest struct {
	Tool       string                 `json:"tool"`
	Parameters map[string]interface{} `json:"parameters"`
}

func parseToolCalls(ctx context.Context, text string) []toolCallRequest {
	var calls []toolCallRequest
	remaining := text
	for {
		start := strings.Index(remaining, "<TOOL_CALL>")
		if start == -1 {
			break
		}
		start += len("<TOOL_CALL>")
		end := strings.Index(remaining[start:], "</TOOL_CALL>")
		if end == -1 {
			break
		}
		jsonStr := strings.TrimSpace(remaining[start : start+end])
		var call toolCallRequest
		if err := json.Unmarshal([]byte(jsonStr), &call); err == nil {
			calls = append(calls, call)
		}
		remaining = remaining[start+end+len("</TOOL_CALL>"):]
	}
	return calls
}

// ─── Tool Execution ────────────────────────────────────────────

type toolExecResult struct {
	Tool  string      `json:"tool"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func (u *AIUsecase) executeTool(ctx context.Context, userID uuid.UUID, call toolCallRequest) toolExecResult {
	switch call.Tool {
	case "get_categories":
		return u.execGetCategories(ctx, userID, call.Parameters)
	case "get_spending_summary":
		return u.execSpendingSummary(ctx, userID, call.Parameters)
	case "get_category_expenses":
		return u.execCategoryExpenses(ctx, userID, call.Parameters)
	case "get_transactions":
		return u.execGetTransactions(ctx, userID, call.Parameters)
	case "create_transaction":
		return u.execCreateTransaction(ctx, userID, call.Parameters)
	case "get_finance_insight":
		return u.execFinanceInsight(ctx, userID, call.Parameters)
	case "generate_chart":
		return u.execGenerateChart(ctx, userID, call.Parameters)
	default:
		return toolExecResult{Tool: call.Tool, Error: fmt.Sprintf("Tool '%s' tidak dikenal", call.Tool)}
	}
}

// ─── Tool Implementations ──────────────────────────────────────

func (u *AIUsecase) execGetCategories(ctx context.Context, userID uuid.UUID, params map[string]interface{}) toolExecResult {
	catType, _ := params["type"].(string)

	incomeCats, _ := u.incomeCatRepo.List(ctx, true)
	expenseCats, _ := u.expenseCatRepo.List(ctx, true)
	savingsCats, _ := u.savingsCatRepo.List(ctx, true)

	result := map[string]interface{}{}
	if catType == "" || catType == "income" {
		var list []map[string]interface{}
		for _, c := range incomeCats {
			list = append(list, map[string]interface{}{
				"id": c.ID.String(), "name": c.Name, "icon": c.Icon,
			})
		}
		result["income_categories"] = list
	}
	if catType == "" || catType == "expense" {
		var list []map[string]interface{}
		for _, c := range expenseCats {
			list = append(list, map[string]interface{}{
				"id": c.ID.String(), "name": c.Name, "icon": c.Icon,
			})
		}
		result["expense_categories"] = list
	}
	if catType == "" || catType == "savings" {
		var list []map[string]interface{}
		for _, c := range savingsCats {
			list = append(list, map[string]interface{}{
				"id": c.ID.String(), "name": c.Name, "icon": c.Icon,
			})
		}
		result["savings_categories"] = list
	}
	return toolExecResult{Tool: "get_categories", Data: result}
}

func (u *AIUsecase) execSpendingSummary(ctx context.Context, userID uuid.UUID, params map[string]interface{}) toolExecResult {
	period, _ := params["period"].(string)
	now := time.Now()
	var start, end time.Time

	switch strings.ToLower(strings.TrimSpace(period)) {
	case "this_month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now
	case "last_month":
		start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)
	case "this_year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end = now
	case "last_3_months", "3_months":
		start = time.Date(now.Year(), now.Month()-3, 1, 0, 0, 0, 0, now.Location())
		end = now
	case "last_12_months", "1_year", "one_year", "1_tahun", "1_tahun_terakhir":
		start = now.AddDate(-1, 0, 0)
		end = now
	case "last_year":
		start = time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)
	default:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now
	}

	stats, err := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: start, EndDate: end})
	if err != nil {
		return toolExecResult{Tool: "get_spending_summary", Error: err.Error()}
	}

	data := map[string]interface{}{
		"period":        period,
		"total_income":  stats.TotalIncome,
		"total_expense": stats.TotalExpense,
		"total_savings": stats.TotalSavings,
		"net_balance":   stats.NetBalance,
		"income_count":  stats.IncomeCount,
		"expense_count": stats.ExpenseCount,
		"savings_count": stats.SavingsCount,
		"start_date":    start.Format("2006-01-02"),
		"end_date":      end.Format("2006-01-02"),
	}

	prevStart := start.AddDate(0, -1, 0)
	prevEnd := start.Add(-time.Second)
	prevStats, _ := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: prevStart, EndDate: prevEnd})
	if prevStats != nil && prevStats.TotalExpense > 0 {
		change := ((stats.TotalExpense - prevStats.TotalExpense) / prevStats.TotalExpense) * 100
		data["expense_change_percent"] = fmt.Sprintf("%.1f", change)
	}

	return toolExecResult{Tool: "get_spending_summary", Data: data}
}

func (u *AIUsecase) execCategoryExpenses(ctx context.Context, userID uuid.UUID, params map[string]interface{}) toolExecResult {
	period, _ := params["period"].(string)
	now := time.Now()
	var start, end time.Time

	switch strings.ToLower(strings.TrimSpace(period)) {
	case "this_month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now
	case "last_month":
		start = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)
	case "this_year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end = now
	case "last_3_months", "3_months":
		start = time.Date(now.Year(), now.Month()-3, 1, 0, 0, 0, 0, now.Location())
		end = now
	case "last_12_months", "1_year", "one_year", "1_tahun", "1_tahun_terakhir":
		start = now.AddDate(-1, 0, 0)
		end = now
	case "last_year":
		start = time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)
	default:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = now
	}

	categories, err := u.txRepo.GetCategoryExpensesByPeriod(ctx, userID, start, end)
	if err != nil {
		return toolExecResult{Tool: "get_category_expenses", Error: err.Error()}
	}

	data := map[string]interface{}{
		"period":     period,
		"start_date": start.Format("2006-01-02"),
		"end_date":   end.Format("2006-01-02"),
	}
	var list []map[string]interface{}
	var total float64
	for _, c := range categories {
		list = append(list, map[string]interface{}{
			"category": c.CategoryName, "total": c.Total, "count": c.Count,
		})
		total += c.Total
	}
	data["categories"] = list
	data["total_expenses"] = total
	return toolExecResult{Tool: "get_category_expenses", Data: data}
}

func (u *AIUsecase) execGetTransactions(ctx context.Context, userID uuid.UUID, params map[string]interface{}) toolExecResult {
	limit := 10
	if l, ok := params["limit"].(string); ok {
		fmt.Sscanf(l, "%d", &limit)
	}

	filter := domain.TransactionFilter{Page: 1, PerPage: limit}

	if t, ok := params["type"].(string); ok {
		tType := domain.TransactionType(t)
		filter.Type = &tType
	}

	if search, ok := params["search"].(string); ok {
		filter.Search = search
	}

	txns, _, err := u.txRepo.List(ctx, userID, filter)
	if err != nil {
		return toolExecResult{Tool: "get_transactions", Error: err.Error()}
	}

	var list []map[string]interface{}
	for _, t := range txns {
		desc := ""
		if t.Description != nil {
			desc = *t.Description
		}
		catName := ""
		if t.ExpenseCategoryID != nil {
			if cat, _ := u.expenseCatRepo.GetByID(ctx, *t.ExpenseCategoryID); cat != nil {
				catName = cat.Name
			}
		}
		if t.IncomeCategoryID != nil {
			if cat, _ := u.incomeCatRepo.GetByID(ctx, *t.IncomeCategoryID); cat != nil {
				catName = cat.Name
			}
		}
		if t.SavingsCategoryID != nil {
			if cat, _ := u.savingsCatRepo.GetByID(ctx, *t.SavingsCategoryID); cat != nil {
				catName = cat.Name
			}
		}
		list = append(list, map[string]interface{}{
			"id": t.ID.String(), "type": t.Type, "amount": t.Amount,
			"category": catName, "description": desc,
			"date": t.TransactionDate.Format("2006-01-02"),
		})
	}
	return toolExecResult{Tool: "get_transactions", Data: map[string]interface{}{
		"count": len(list), "transactions": list,
	}}
}

func (u *AIUsecase) execCreateTransaction(ctx context.Context, userID uuid.UUID, params map[string]interface{}) toolExecResult {
	txType, _ := params["type"].(string)
	amountStr, _ := params["amount"].(string)
	categoryName, _ := params["category"].(string)
	desc, _ := params["description"].(string)

	var amount float64
	fmt.Sscanf(amountStr, "%f", &amount)
	if amount <= 0 {
		return toolExecResult{Tool: "create_transaction", Error: "Jumlah tidak valid"}
	}

	txDate := time.Now()

	cType := domain.TransactionType(txType)
	var incomeCatID, expenseCatID, savingsCatID *uuid.UUID

	switch cType {
	case domain.TransactionIncome:
		cats, err := u.incomeCatRepo.List(ctx, false)
		if err == nil {
			catLower := strings.ToLower(categoryName)
			for _, c := range cats {
				if strings.EqualFold(c.Name, categoryName) {
					incomeCatID = &c.ID
					break
				}
			}
			if incomeCatID == nil {
				for _, c := range cats {
					cNameLower := strings.ToLower(c.Name)
					if strings.Contains(cNameLower, catLower) || strings.Contains(catLower, cNameLower) {
						incomeCatID = &c.ID
						break
					}
				}
			}
			if incomeCatID == nil {
				newCat := &domain.IncomeCategory{Name: categoryName, Icon: "circle", Color: "#94a3b8", SortOrder: 99}
				if err := u.incomeCatRepo.Create(ctx, newCat); err != nil {
					logger.Warn("gagal auto-create income category", zap.Error(err))
					for _, c := range cats {
						if strings.EqualFold(c.Name, "Lainnya") {
							incomeCatID = &c.ID
							break
						}
					}
				} else {
					incomeCatID = &newCat.ID
				}
			}
		}
	case domain.TransactionExpense:
		cats, err := u.expenseCatRepo.List(ctx, false)
		if err == nil {
			catLower := strings.ToLower(categoryName)
			for _, c := range cats {
				if strings.EqualFold(c.Name, categoryName) {
					expenseCatID = &c.ID
					break
				}
			}
			if expenseCatID == nil {
				for _, c := range cats {
					cNameLower := strings.ToLower(c.Name)
					if strings.Contains(cNameLower, catLower) || strings.Contains(catLower, cNameLower) {
						expenseCatID = &c.ID
						break
					}
				}
			}
			if expenseCatID == nil {
				newCat := &domain.ExpenseCategory{Name: categoryName, Icon: "circle", Color: "#94a3b8", SortOrder: 99}
				if err := u.expenseCatRepo.Create(ctx, newCat); err != nil {
					logger.Warn("gagal auto-create expense category", zap.Error(err))
					for _, c := range cats {
						if strings.EqualFold(c.Name, "Lain-lain") {
							expenseCatID = &c.ID
							break
						}
					}
				} else {
					expenseCatID = &newCat.ID
				}
			}
		}
	case domain.TransactionSavings:
		cats, err := u.savingsCatRepo.List(ctx, false)
		if err == nil {
			catLower := strings.ToLower(categoryName)
			for _, c := range cats {
				if strings.EqualFold(c.Name, categoryName) {
					savingsCatID = &c.ID
					break
				}
			}
			if savingsCatID == nil {
				for _, c := range cats {
					cNameLower := strings.ToLower(c.Name)
					if strings.Contains(cNameLower, catLower) || strings.Contains(catLower, cNameLower) {
						savingsCatID = &c.ID
						break
					}
				}
			}
			if savingsCatID == nil {
				newCat := &domain.SavingsCategory{Name: categoryName, Icon: "circle", Color: "#94a3b8", SortOrder: 99}
				if err := u.savingsCatRepo.Create(ctx, newCat); err != nil {
					logger.Warn("gagal auto-create savings category", zap.Error(err))
					for _, c := range cats {
						if strings.EqualFold(c.Name, "Lainnya") {
							savingsCatID = &c.ID
							break
						}
					}
				} else {
					savingsCatID = &newCat.ID
				}
			}
		}
	}

	var descPtr *string
	if desc != "" {
		descPtr = &desc
	}

	tx := &domain.Transaction{
		UserID:            userID,
		Type:              cType,
		Amount:            amount,
		IncomeCategoryID:  incomeCatID,
		ExpenseCategoryID: expenseCatID,
		SavingsCategoryID: savingsCatID,
		Description:       descPtr,
		TransactionDate:   txDate,
		Source:            domain.SourceAI,
	}

	if err := u.txRepo.Create(ctx, tx); err != nil {
		logger.Error("failed to create transaction from AI", zap.Error(err))
		return toolExecResult{Tool: "create_transaction", Error: "Gagal menyimpan transaksi"}
	}

	return toolExecResult{Tool: "create_transaction", Data: map[string]interface{}{
		"id": tx.ID.String(), "type": string(cType), "amount": amount,
		"category": categoryName, "description": desc,
		"date": txDate.Format("2006-01-02"),
	}}
}

func (u *AIUsecase) execGenerateChart(ctx context.Context, userID uuid.UUID, params map[string]interface{}) toolExecResult {
	startStr, _ := params["start_date"].(string)
	endStr, _ := params["end_date"].(string)
	compareStartStr, _ := params["compare_start"].(string)
	compareEndStr, _ := params["compare_end"].(string)

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return toolExecResult{Tool: "generate_chart", Error: "Format start_date tidak valid. Gunakan YYYY-MM-DD"}
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return toolExecResult{Tool: "generate_chart", Error: "Format end_date tidak valid. Gunakan YYYY-MM-DD"}
	}

	// Determine granularity based on date range
	days := end.Sub(start).Hours() / 24
	granularity := "month"
	switch {
	case days <= 35:
		granularity = "day"
	case days <= 95:
		granularity = "week"
	}

	points, err := u.txRepo.GetTimeSeries(ctx, userID, start, end, granularity)
	if err != nil {
		return toolExecResult{Tool: "generate_chart", Error: "Gagal mengambil data: " + err.Error()}
	}

	var labels []string
	var incomeData []float64
	var expenseData []float64
	var savingsData []float64
	for _, p := range points {
		labels = append(labels, p.Period)
		incomeData = append(incomeData, p.Income)
		expenseData = append(expenseData, p.Expense)
		savingsData = append(savingsData, p.Savings)
	}

	result := map[string]interface{}{
		"chart_type": "comparison",
		"labels":     labels,
		"income":     incomeData,
		"expense":    expenseData,
		"savings":    savingsData,
	}

	// Add comparison period if provided
	if compareStartStr != "" && compareEndStr != "" {
		cs, err1 := time.Parse("2006-01-02", compareStartStr)
		ce, err2 := time.Parse("2006-01-02", compareEndStr)
		if err1 == nil && err2 == nil {
			cmpPoints, _ := u.txRepo.GetTimeSeries(ctx, userID, cs, ce, granularity)
			var cmpLabels []string
			var cmpIncome, cmpExpense, cmpSavings []float64
			for _, p := range cmpPoints {
				cmpLabels = append(cmpLabels, p.Period)
				cmpIncome = append(cmpIncome, p.Income)
				cmpExpense = append(cmpExpense, p.Expense)
				cmpSavings = append(cmpSavings, p.Savings)
			}
			result["compare_labels"] = cmpLabels
			result["compare_income"] = cmpIncome
			result["compare_expense"] = cmpExpense
			result["compare_savings"] = cmpSavings
		}
	}

	return toolExecResult{Tool: "generate_chart", Data: result}
}

func (u *AIUsecase) execFinanceInsight(ctx context.Context, userID uuid.UUID, params map[string]interface{}) toolExecResult {
	topic, _ := params["topic"].(string)

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	stats, _ := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: monthStart, EndDate: now})

	oneYearAgo := now.AddDate(-1, 0, 0)
	yearStats, _ := u.txRepo.GetStats(ctx, userID, domain.StatsPeriod{StartDate: oneYearAgo, EndDate: now})

	categories, _ := u.txRepo.GetCategoryExpensesByPeriod(ctx, userID, monthStart, now)

	profile, _ := u.profileRepo.GetByID(ctx, userID)

	hasData := (stats != nil && stats.TotalIncome+stats.TotalExpense+stats.TotalSavings > 0) ||
		(yearStats != nil && yearStats.TotalIncome+yearStats.TotalExpense+yearStats.TotalSavings > 0)

	data := map[string]interface{}{
		"topic":    topic,
		"has_data": hasData,
	}
	if stats != nil {
		data["monthly_income"] = stats.TotalIncome
		data["monthly_expense"] = stats.TotalExpense
		data["monthly_savings"] = stats.TotalSavings
		data["net_balance"] = stats.NetBalance
	}
	if yearStats != nil {
		data["yearly_income"] = yearStats.TotalIncome
		data["yearly_expense"] = yearStats.TotalExpense
		data["yearly_savings"] = yearStats.TotalSavings
		data["yearly_net_balance"] = yearStats.NetBalance
		data["avg_monthly_income"] = yearStats.TotalIncome / 12
		data["avg_monthly_expense"] = yearStats.TotalExpense / 12
		data["avg_monthly_savings"] = yearStats.TotalSavings / 12
		if yearStats.TotalIncome > 0 {
			data["savings_rate_percent"] = fmt.Sprintf("%.1f", ((yearStats.TotalSavings+yearStats.NetBalance)/yearStats.TotalIncome)*100)
		}
	}
	if len(categories) > 0 {
		data["top_category"] = categories[0].CategoryName
		data["top_category_total"] = categories[0].Total
	}
	if profile != nil {
		data["user_name"] = profile.FullName
	}

	return toolExecResult{Tool: "get_finance_insight", Data: data}
}
