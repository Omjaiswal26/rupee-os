package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/rupee-os/rupee-os/internal/classification"
	"github.com/rupee-os/rupee-os/internal/models"
	"github.com/rupee-os/rupee-os/internal/validation"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HealthService struct {
	DB     *gorm.DB
	Txns   *TransactionService
	Client *classification.HealthClient
	Log    *slog.Logger
}

type MonthHealthDTO struct {
	Month            string                       `json:"month"`
	Score            int                          `json:"score"`
	Verdict          string                       `json:"verdict"`
	Tip              string                       `json:"tip"`
	Signals          classification.HealthSignals `json:"signals"`
	ModelName        string                       `json:"model_name"`
	GeneratedAt      time.Time                    `json:"generated_at"`
	Classifying      bool                         `json:"classifying"`
	Stale            bool                         `json:"stale"`
}

func (s *HealthService) Get(ctx context.Context, monthKey string) (*MonthHealthDTO, error) {
	start, err := validation.ParseMonthKey(monthKey)
	if err != nil {
		return nil, err
	}
	var row models.MonthHealthReport
	err = s.DB.WithContext(ctx).First(&row, "month = ?", start).Error
	if err != nil {
		return nil, err
	}
	return s.toDTO(monthKey, &row, false)
}

// Generate builds signals, scores, asks the health model for a tip, and upserts the row.
func (s *HealthService) Generate(ctx context.Context, monthKey string) (*MonthHealthDTO, error) {
	start, err := validation.ParseMonthKey(monthKey)
	if err != nil {
		return nil, err
	}

	summary, err := s.Txns.Summary(ctx, monthKey)
	if err != nil {
		return nil, err
	}
	if summary.TransactionCount == 0 {
		return nil, fmt.Errorf("no transactions for month %s", monthKey)
	}
	if summary.Classifying {
		return nil, fmt.Errorf("classification still in progress")
	}

	signals, err := s.buildSignals(ctx, monthKey, summary)
	if err != nil {
		return nil, err
	}
	scored := classification.ScoreHealth(signals)

	tip := classification.FallbackTip(signals, scored)
	modelName := "deterministic"
	if s.Client != nil && s.Client.Model != "" {
		modelName = s.Client.Model
		if t, err := s.Client.GenerateTip(ctx, monthKey, signals, scored); err != nil {
			s.Log.Warn("health tip generation failed; using fallback", "month", monthKey, "err", err)
		} else {
			tip = t
		}
	}

	sigJSON, err := json.Marshal(signals)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row := models.MonthHealthReport{
		Month:       start,
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
		Columns: []clause.Column{{Name: "month"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"score", "verdict", "tip", "signals", "model_name", "generated_at", "updated_at",
		}),
	}).Create(&row).Error; err != nil {
		return nil, err
	}

	s.Log.Info("month health saved", "month", monthKey, "score", row.Score, "model", row.ModelName)
	return s.toDTO(monthKey, &row, false)
}

func (s *HealthService) DeleteMonth(ctx context.Context, monthKey string) error {
	start, err := validation.ParseMonthKey(monthKey)
	if err != nil {
		return err
	}
	return s.DB.WithContext(ctx).Where("month = ?", start).Delete(&models.MonthHealthReport{}).Error
}

func (s *HealthService) buildSignals(ctx context.Context, monthKey string, summary *MonthSummary) (classification.HealthSignals, error) {
	start, err := validation.ParseMonthKey(monthKey)
	if err != nil {
		return classification.HealthSignals{}, err
	}
	end := start.AddDate(0, 1, 0)

	var investPaise int64
	_ = s.DB.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(amount_paise), 0)
		FROM transactions
		WHERE transaction_date >= ? AND transaction_date < ?
		  AND (transaction_type = 'investment' OR category = 'Investments')
	`, start, end).Scan(&investPaise).Error

	income := summary.IncomePaise
	expense := summary.ExpensePaise
	uncat := summary.UncategorizedPaise

	var savings float64
	if income > 0 {
		savings = float64(income-expense) / float64(income)
	}
	var investShare float64
	if expense > 0 {
		investShare = float64(investPaise) / float64(expense)
	}
	denom := income + expense + uncat
	var uncatRatio float64
	if denom > 0 {
		uncatRatio = float64(uncat) / float64(denom)
	}

	top := make([]classification.CategoryShare, 0, 5)
	totalBreakdown := int64(0)
	for _, c := range summary.CategoryBreakdown {
		totalBreakdown += c.AmountPaise
	}
	for i, c := range summary.CategoryBreakdown {
		if i >= 5 {
			break
		}
		share := 0.0
		if totalBreakdown > 0 {
			share = float64(c.AmountPaise) / float64(totalBreakdown)
		}
		top = append(top, classification.CategoryShare{
			Category:    c.Category,
			AmountPaise: c.AmountPaise,
			Share:       share,
		})
	}

	return classification.HealthSignals{
		IncomePaise:        income,
		ExpensePaise:       expense,
		UncategorizedPaise: uncat,
		InvestmentPaise:    investPaise,
		TransactionCount:   summary.TransactionCount,
		PendingCount:       summary.PendingCount,
		NeedsReviewCount:   summary.NeedsReviewCount,
		SavingsRate:        savings,
		InvestShare:        investShare,
		UncategorizedRatio: uncatRatio,
		TopCategories:      top,
	}, nil
}

func (s *HealthService) toDTO(monthKey string, row *models.MonthHealthReport, stale bool) (*MonthHealthDTO, error) {
	var signals classification.HealthSignals
	if len(row.Signals) > 0 {
		_ = json.Unmarshal(row.Signals, &signals)
	}
	summary, _ := s.Txns.Summary(context.Background(), monthKey)
	classifying := summary != nil && summary.Classifying
	return &MonthHealthDTO{
		Month:       monthKey,
		Score:       row.Score,
		Verdict:     row.Verdict,
		Tip:         row.Tip,
		Signals:     signals,
		ModelName:   row.ModelName,
		GeneratedAt: row.GeneratedAt,
		Classifying: classifying,
		Stale:       stale,
	}, nil
}
