package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rupee-os/rupee-os/internal/classification"
	"github.com/rupee-os/rupee-os/internal/models"
	"github.com/rupee-os/rupee-os/internal/validation"
	"gorm.io/gorm"
)

type TransactionService struct {
	DB *gorm.DB
}

type TransactionFilter struct {
	Month    string // YYYY-MM optional
	Query    string
	Category string
	Status   string
	Sort     string // date | amount | description | category
	Order    string // asc | desc
	Limit    int
	Offset   int
}

type MonthSummary struct {
	Month                 string           `json:"month"`
	IncomePaise           int64            `json:"income_paise"`
	ExpensePaise          int64            `json:"expense_paise"`
	UncategorizedPaise    int64            `json:"uncategorized_paise"`
	NetPaise              int64            `json:"net_paise"` // kept for API compat; UI no longer shows it
	TransactionCount      int64            `json:"transaction_count"`
	NeedsReviewCount      int64            `json:"needs_review_count"`
	CategoryBreakdown     []CategoryAmount `json:"category_breakdown"`
}

type CategoryAmount struct {
	Category   string `json:"category"`
	AmountPaise int64 `json:"amount_paise"`
}

func (s *TransactionService) ListMonths(ctx context.Context) ([]string, error) {
	var months []time.Time
	err := s.DB.WithContext(ctx).
		Model(&models.Import{}).
		Where("status = ? AND validated_month IS NOT NULL", models.ImportStatusCompleted).
		Distinct("validated_month").
		Order("validated_month DESC").
		Pluck("validated_month", &months).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(months))
	for _, m := range months {
		out = append(out, m.Format("2006-01"))
	}
	return out, nil
}

func (s *TransactionService) Summary(ctx context.Context, monthKey string) (*MonthSummary, error) {
	start, err := validation.ParseMonthKey(monthKey)
	if err != nil {
		return nil, err
	}
	end := start.AddDate(0, 1, 0)

	type agg struct {
		Income         int64
		Expense        int64
		Uncategorized  int64
		Count          int64
	}
	var a agg
	// Income/expense exclude category Other (shown as Uncategorized).
	// transfer excluded from income/expense; investment + fee count as expense.
	err = s.DB.WithContext(ctx).Raw(`
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
				THEN amount_paise ELSE 0 END), 0) AS uncategorized,
			COUNT(*) AS count
		FROM transactions
		WHERE transaction_date >= ? AND transaction_date < ?
	`, start, end).Scan(&a).Error
	if err != nil {
		return nil, err
	}

	var reviewCount int64
	_ = s.DB.WithContext(ctx).Model(&models.Transaction{}).
		Where("transaction_date >= ? AND transaction_date < ? AND classification_status = ?",
			start, end, models.ClassificationNeedsReview).
		Count(&reviewCount).Error

	// Pie includes all categories (Transfers, Other/Uncategorized, spend, income).
	var breakdown []CategoryAmount
	err = s.DB.WithContext(ctx).Raw(`
		SELECT CASE
		         WHEN category IS NULL OR category = 'Other' THEN 'Uncategorized'
		         ELSE category
		       END AS category,
		       COALESCE(SUM(amount_paise), 0) AS amount_paise
		FROM transactions
		WHERE transaction_date >= ? AND transaction_date < ?
		GROUP BY CASE
		           WHEN category IS NULL OR category = 'Other' THEN 'Uncategorized'
		           ELSE category
		         END
		ORDER BY amount_paise DESC
	`, start, end).Scan(&breakdown).Error
	if err != nil {
		return nil, err
	}

	return &MonthSummary{
		Month:              monthKey,
		IncomePaise:        a.Income,
		ExpensePaise:       a.Expense,
		UncategorizedPaise: a.Uncategorized,
		NetPaise:           a.Income - a.Expense,
		TransactionCount:   a.Count,
		NeedsReviewCount:   reviewCount,
		CategoryBreakdown:  breakdown,
	}, nil
}

func (s *TransactionService) List(ctx context.Context, f TransactionFilter) ([]models.Transaction, int64, error) {
	q := s.DB.WithContext(ctx).Model(&models.Transaction{})

	if f.Month != "" {
		start, err := validation.ParseMonthKey(f.Month)
		if err != nil {
			return nil, 0, err
		}
		end := start.AddDate(0, 1, 0)
		q = q.Where("transaction_date >= ? AND transaction_date < ?", start, end)
	}
	if f.Query != "" {
		like := "%" + strings.TrimSpace(f.Query) + "%"
		q = q.Where("original_description ILIKE ?", like)
	}
	if f.Category != "" {
		if strings.EqualFold(f.Category, "Uncategorized") || f.Category == "Other" {
			q = q.Where("category IS NULL OR category = '' OR category = ?", "Other")
		} else {
			q = q.Where("category = ?", f.Category)
		}
	}
	switch strings.ToLower(strings.TrimSpace(f.Status)) {
	case "":
		// no status filter
	case "attention":
		// Ambiguous / incomplete: pending, needs review, or Other/uncategorized.
		q = q.Where(`
			classification_status IN (?, ?)
			OR category = ?
			OR category IS NULL
			OR category = ''
		`, models.ClassificationPending, models.ClassificationNeedsReview, "Other")
	default:
		q = q.Where("classification_status = ?", f.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	orderClause := listOrderSQL(f.Sort, f.Order)

	var rows []models.Transaction
	err := q.Order(orderClause).
		Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func listOrderSQL(sort, order string) string {
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(order), "asc") {
		dir = "ASC"
	}
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "amount":
		// Signed amount: credits positive, debits negative.
		return fmt.Sprintf(
			`(CASE WHEN direction = 'credit' THEN amount_paise ELSE -amount_paise END) %s, transaction_date DESC, created_at DESC`,
			dir,
		)
	case "description":
		return fmt.Sprintf("original_description %s, transaction_date DESC", dir)
	case "category":
		return fmt.Sprintf("COALESCE(category, '') %s, transaction_date DESC", dir)
	default: // date
		return fmt.Sprintf("transaction_date %s, created_at %s", dir, dir)
	}
}

func (s *TransactionService) ListNeedsReview(ctx context.Context, limit int) ([]models.Transaction, error) {
	return s.ListAttention(ctx, "", limit)
}

// ListAttention returns pending, needs_review, Other, and uncategorized rows.
func (s *TransactionService) ListAttention(ctx context.Context, month string, limit int) ([]models.Transaction, error) {
	rows, _, err := s.List(ctx, TransactionFilter{
		Month:  month,
		Status: "attention",
		Limit:  limit,
	})
	return rows, err
}

type CorrectClassificationInput struct {
	Category         string  `json:"category" binding:"required"`
	Subcategory      *string `json:"subcategory"`
	Type             *string `json:"type"`
	Note             *string `json:"note"`
	CreateRule       bool    `json:"create_rule"`
	ApplyToMatching  bool    `json:"apply_to_matching"`
}

type CorrectResult struct {
	Transaction  *models.Transaction `json:"transaction"`
	AppliedCount int                 `json:"applied_count"`
	Pattern      string              `json:"pattern,omitempty"`
}

func (s *TransactionService) Correct(
	ctx context.Context,
	txnID uuid.UUID,
	in CorrectClassificationInput,
) (*CorrectResult, error) {
	if !classification.IsAllowedCategory(in.Category) {
		return nil, fmt.Errorf("invalid category %q", in.Category)
	}
	if in.Type != nil && *in.Type != "" && !classification.IsAllowedType(*in.Type) {
		return nil, fmt.Errorf("invalid type %q", *in.Type)
	}

	var txn models.Transaction
	if err := s.DB.WithContext(ctx).First(&txn, "id = ?", txnID).Error; err != nil {
		return nil, err
	}

	pattern := classification.RulePatternFromDescription(txn.OriginalDescription)
	createRule := in.CreateRule || in.ApplyToMatching
	applied := 1

	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fb := models.UserFeedback{
			ID:                   uuid.New(),
			TransactionID:        txnID,
			PreviousCategory:     txn.Category,
			PreviousSubcategory:  txn.Subcategory,
			CorrectedCategory:    in.Category,
			CorrectedSubcategory: in.Subcategory,
			CorrectedType:        in.Type,
			Note:                 in.Note,
		}
		if err := tx.Create(&fb).Error; err != nil {
			return err
		}

		updates := map[string]any{
			"category":              in.Category,
			"subcategory":           in.Subcategory,
			"classification_status": models.ClassificationCorrected,
			"updated_at":            time.Now().UTC(),
		}
		if in.Type != nil {
			updates["transaction_type"] = in.Type
		}
		if err := tx.Model(&models.Transaction{}).Where("id = ?", txnID).Updates(updates).Error; err != nil {
			return err
		}

		if in.ApplyToMatching && pattern != "" {
			res := tx.Model(&models.Transaction{}).
				Where("id <> ? AND original_description ILIKE ?", txnID, "%"+pattern+"%").
				Updates(updates)
			if res.Error != nil {
				return res.Error
			}
			applied += int(res.RowsAffected)
		}

		if createRule && pattern != "" {
			rule := models.UserRule{
				ID:              uuid.New(),
				Pattern:         pattern,
				MatchType:       "contains",
				Category:        in.Category,
				Subcategory:     in.Subcategory,
				TransactionType: in.Type,
				Priority:        10,
				Active:          true,
			}
			if err := tx.Create(&rule).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).First(&txn, "id = ?", txnID).Error; err != nil {
		return nil, err
	}
	return &CorrectResult{
		Transaction:  &txn,
		AppliedCount: applied,
		Pattern:      pattern,
	}, nil
}

func (s *TransactionService) ListRules(ctx context.Context) ([]models.UserRule, error) {
	var rules []models.UserRule
	err := s.DB.WithContext(ctx).Where("active = ?", true).Order("priority DESC").Find(&rules).Error
	return rules, err
}
