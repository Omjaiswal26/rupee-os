package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rupee-os/rupee-os/internal/classification"
	"github.com/rupee-os/rupee-os/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

type DashboardMonthBar struct {
	Month            string `json:"month"`
	Label            string `json:"label"`
	HasStatement     bool   `json:"has_statement"`
	Status           string `json:"status"`
	TransactionCount int64  `json:"transaction_count"`
	IncomePaise      int64  `json:"income_paise"`
	ExpensePaise     int64  `json:"expense_paise"`
	UncategorizedPaise int64 `json:"uncategorized_paise"`
	NeedsReviewCount int64  `json:"needs_review_count"`
	Score            *int   `json:"score,omitempty"`
	Verdict          string `json:"verdict,omitempty"`
}

type YearHealthDTO struct {
	Year        int                          `json:"year"`
	Score       int                          `json:"score"`
	Verdict     string                       `json:"verdict"`
	Tip         string                       `json:"tip"`
	Signals     classification.HealthSignals `json:"signals"`
	ModelName   string                       `json:"model_name"`
	GeneratedAt time.Time                    `json:"generated_at"`
}

type YearDashboard struct {
	Year               int                 `json:"year"`
	Years              []int               `json:"years"`
	IncomePaise        int64               `json:"income_paise"`
	ExpensePaise       int64               `json:"expense_paise"`
	UncategorizedPaise int64               `json:"uncategorized_paise"`
	MonthsWithData     int                 `json:"months_with_data"`
	NeedsReviewMonths  int                 `json:"needs_review_months"`
	Months             []DashboardMonthBar `json:"months"`
	Health             *YearHealthDTO      `json:"health"`
}

func (s *HealthService) YearDashboard(ctx context.Context, year int) (*YearDashboard, error) {
	if year < 1990 || year > 2100 {
		return nil, fmt.Errorf("invalid year")
	}
	cal, err := s.Txns.Calendar(ctx, year)
	if err != nil {
		return nil, err
	}

	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	type yearAgg struct {
		Income        int64
		Expense       int64
		Uncategorized int64
	}
	var ya yearAgg
	_ = s.DB.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(CASE
				WHEN transaction_type = 'income'
				 AND COALESCE(category, 'Other') NOT IN ('Other', 'Transfers')
				THEN amount_paise ELSE 0 END), 0) AS income,
			COALESCE(SUM(CASE
				WHEN transaction_type IN ('expense', 'investment', 'fee')
				 AND COALESCE(category, 'Other') <> 'Other'
				THEN amount_paise ELSE 0 END), 0) AS expense,
			COALESCE(SUM(CASE
				WHEN category IS NULL OR category = 'Other'
				THEN amount_paise ELSE 0 END), 0) AS uncategorized
		FROM transactions
		WHERE transaction_date >= ? AND transaction_date < ?
	`, start, end).Scan(&ya).Error

	// Month health scores for bars.
	var healthRows []models.MonthHealthReport
	_ = s.DB.WithContext(ctx).
		Where("month >= ? AND month < ?", start, end).
		Find(&healthRows).Error
	healthByMonth := map[string]models.MonthHealthReport{}
	for _, h := range healthRows {
		key := h.Month.UTC().Format("2006-01")
		healthByMonth[key] = h
	}

	bars := make([]DashboardMonthBar, 0, 12)
	monthsWithData := 0
	reviewMonths := 0
	for _, m := range cal.Months {
		bar := DashboardMonthBar{
			Month:            m.Month,
			Label:            m.Label[:3], // Jan, Feb…
			HasStatement:     m.HasStatement,
			Status:           m.Status,
			TransactionCount: m.TransactionCount,
			NeedsReviewCount: m.NeedsReviewCount,
		}
		if m.HasStatement {
			monthsWithData++
			if sum, err := s.Txns.Summary(ctx, m.Month); err == nil {
				bar.IncomePaise = sum.IncomePaise
				bar.ExpensePaise = sum.ExpensePaise
				bar.UncategorizedPaise = sum.UncategorizedPaise
			}
		}
		if m.Status == MonthStatusNeedsReview || m.NeedsReviewCount > 0 {
			reviewMonths++
		}
		if h, ok := healthByMonth[m.Month]; ok {
			sc := h.Score
			bar.Score = &sc
			bar.Verdict = h.Verdict
		}
		bars = append(bars, bar)
	}

	out := &YearDashboard{
		Year:               year,
		Years:              cal.Years,
		IncomePaise:        ya.Income,
		ExpensePaise:       ya.Expense,
		UncategorizedPaise: ya.Uncategorized,
		MonthsWithData:     monthsWithData,
		NeedsReviewMonths:  reviewMonths,
		Months:             bars,
	}

	var yr models.YearHealthReport
	if err := s.DB.WithContext(ctx).First(&yr, "year = ?", year).Error; err == nil {
		dto, _ := s.yearToDTO(&yr)
		out.Health = dto
	}

	return out, nil
}

func (s *HealthService) GetYearHealth(ctx context.Context, year int) (*YearHealthDTO, error) {
	var yr models.YearHealthReport
	if err := s.DB.WithContext(ctx).First(&yr, "year = ?", year).Error; err != nil {
		return nil, err
	}
	return s.yearToDTO(&yr)
}

func (s *HealthService) GenerateYearHealth(ctx context.Context, year int) (*YearHealthDTO, error) {
	if year < 1990 || year > 2100 {
		return nil, fmt.Errorf("invalid year")
	}
	dash, err := s.YearDashboard(ctx, year)
	if err != nil {
		return nil, err
	}
	if dash.MonthsWithData == 0 {
		return nil, fmt.Errorf("no statements for %d", year)
	}

	signals, err := s.buildYearSignals(ctx, year, dash)
	if err != nil {
		return nil, err
	}
	scored := classification.ScoreHealth(signals)

	tip := classification.FallbackTip(signals, scored)
	modelName := "deterministic"
	if s.Client != nil && s.Client.Model != "" {
		modelName = s.Client.Model
		if t, err := s.Client.GenerateTip(ctx, fmt.Sprintf("%d", year), signals, scored); err != nil {
			s.Log.Warn("year health tip failed; using fallback", "year", year, "err", err)
		} else {
			tip = t
		}
	}

	sigJSON, err := json.Marshal(signals)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row := models.YearHealthReport{
		Year:        year,
		Score:       scored.Score,
		Verdict:     scored.Verdict,
		Tip:         tip,
		Signals:     datatypes.JSON(sigJSON),
		ModelName:   modelName,
		GeneratedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "year"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"score", "verdict", "tip", "signals", "model_name", "generated_at", "updated_at",
		}),
	}).Create(&row).Error; err != nil {
		return nil, err
	}
	s.Log.Info("year health saved", "year", year, "score", row.Score, "model", row.ModelName)
	return s.yearToDTO(&row)
}

func (s *HealthService) buildYearSignals(ctx context.Context, year int, dash *YearDashboard) (classification.HealthSignals, error) {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	var investPaise, txnCount, pending, review int64
	_ = s.DB.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(amount_paise), 0)
		FROM transactions
		WHERE transaction_date >= ? AND transaction_date < ?
		  AND (transaction_type = 'investment' OR category = 'Investments')
	`, start, end).Scan(&investPaise).Error
	_ = s.DB.WithContext(ctx).Model(&models.Transaction{}).
		Where("transaction_date >= ? AND transaction_date < ?", start, end).
		Count(&txnCount).Error
	_ = s.DB.WithContext(ctx).Model(&models.Transaction{}).
		Where("transaction_date >= ? AND transaction_date < ? AND classification_status = ?",
			start, end, models.ClassificationPending).
		Count(&pending).Error
	_ = s.DB.WithContext(ctx).Model(&models.Transaction{}).
		Where("transaction_date >= ? AND transaction_date < ? AND classification_status = ?",
			start, end, models.ClassificationNeedsReview).
		Count(&review).Error

	income := dash.IncomePaise
	expense := dash.ExpensePaise
	uncat := dash.UncategorizedPaise

	var savings, investShare, uncatRatio float64
	if income > 0 {
		savings = float64(income-expense) / float64(income)
	}
	if expense > 0 {
		investShare = float64(investPaise) / float64(expense)
	}
	denom := income + expense + uncat
	if denom > 0 {
		uncatRatio = float64(uncat) / float64(denom)
	}

	return classification.HealthSignals{
		IncomePaise:        income,
		ExpensePaise:       expense,
		UncategorizedPaise: uncat,
		InvestmentPaise:    investPaise,
		TransactionCount:   txnCount,
		PendingCount:       pending,
		NeedsReviewCount:   review,
		SavingsRate:        savings,
		InvestShare:        investShare,
		UncategorizedRatio: uncatRatio,
		TopCategories:      nil,
	}, nil
}

func (s *HealthService) yearToDTO(row *models.YearHealthReport) (*YearHealthDTO, error) {
	var signals classification.HealthSignals
	if len(row.Signals) > 0 {
		_ = json.Unmarshal(row.Signals, &signals)
	}
	return &YearHealthDTO{
		Year:        row.Year,
		Score:       row.Score,
		Verdict:     row.Verdict,
		Tip:         row.Tip,
		Signals:     signals,
		ModelName:   row.ModelName,
		GeneratedAt: row.GeneratedAt,
	}, nil
}
