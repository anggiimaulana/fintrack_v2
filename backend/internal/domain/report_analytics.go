package domain

// ─── Report analytics (deep-dive tabs) ────────────────────────
//
// These types back GET /reports/analytics, which supplies the data the
// "Laporan" deep-dive tabs need beyond the base report: period-over-period
// deltas, biggest category movers, a daily series for the calendar heatmap,
// statistical anomalies, financial ratios, and the best/worst month.

// PeriodDelta is one metric compared across two periods.
type PeriodDelta struct {
	Current   float64 `json:"current"`
	Previous  float64 `json:"previous"`
	Change    float64 `json:"change"`     // current - previous
	ChangePct float64 `json:"change_pct"` // percentage change (0 when previous is 0)
}

// PeriodComparisonSet is one comparison window (MoM, QoQ, or YoY).
type PeriodComparisonSet struct {
	Label   string      `json:"label"`
	Start   string      `json:"start"`
	End     string      `json:"end"`
	Income  PeriodDelta `json:"income"`
	Expense PeriodDelta `json:"expense"`
	Savings PeriodDelta `json:"savings"`
	Surplus PeriodDelta `json:"surplus"`
}

// PeriodComparison bundles the three standard comparison windows.
type PeriodComparison struct {
	MoM PeriodComparisonSet `json:"mom"`
	QoQ PeriodComparisonSet `json:"qoq"`
	YoY PeriodComparisonSet `json:"yoy"`
}

// CategoryMover is a category whose spend changed most versus the prior period.
type CategoryMover struct {
	CategoryName string  `json:"category_name"`
	Current      float64 `json:"current"`
	Previous     float64 `json:"previous"`
	Change       float64 `json:"change"`
	ChangePct    float64 `json:"change_pct"`
}

// TopMovers splits the largest increases and decreases.
type TopMovers struct {
	Up   []CategoryMover `json:"up"`
	Down []CategoryMover `json:"down"`
}

// DailyPoint is one day's totals, used to draw the calendar heatmap.
type DailyPoint struct {
	Date    string  `json:"date"` // YYYY-MM-DD
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Savings float64 `json:"savings"`
	Net     float64 `json:"net"`
}

// CategoryAnomaly flags a category whose current value deviates from its own
// recent history by more than a threshold number of standard deviations.
type CategoryAnomaly struct {
	CategoryName string  `json:"category_name"`
	Current      float64 `json:"current"`
	Mean         float64 `json:"mean"`
	StdDev       float64 `json:"std_dev"`
	ZScore       float64 `json:"z_score"`
	Direction    string  `json:"direction"` // "spike" | "drop"
}

// FinancialRatios are derived, human-meaningful ratios.
type FinancialRatios struct {
	// SavingsRate = (income - expense) / income.
	SavingsRate float64 `json:"savings_rate"`
	// ExpenseRatio = expense / income.
	ExpenseRatio float64 `json:"expense_ratio"`
	// RunwayMonths = accumulated savings / average monthly expense; how many
	// months the savings would cover current spending. Capped for display.
	RunwayMonths float64 `json:"runway_months"`
	// MonthsSpanned is the number of months in the reported period.
	MonthsSpanned int `json:"months_spanned"`
}

// BestWorstPeriod names the best and worst months by surplus.
type BestWorstPeriod struct {
	BestMonth  string  `json:"best_month"`
	BestValue  float64 `json:"best_value"`
	WorstMonth string  `json:"worst_month"`
	WorstValue float64 `json:"worst_value"`
}

// ReportAnalytics is the full payload for the analytics tabs.
type ReportAnalytics struct {
	Comparison PeriodComparison `json:"comparison"`
	TopMovers  TopMovers        `json:"top_movers"`
	Daily      []DailyPoint     `json:"daily"`
	Anomalies  []CategoryAnomaly `json:"anomalies"`
	Ratios     FinancialRatios  `json:"ratios"`
	BestWorst  BestWorstPeriod  `json:"best_worst"`
}
