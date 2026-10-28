package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rupee-os/rupee-os/internal/jobs"
	"github.com/rupee-os/rupee-os/internal/models"
	"github.com/rupee-os/rupee-os/internal/validation"
)

// MonthStatus for calendar UI.
const (
	MonthStatusEmpty       = "empty"        // no statement
	MonthStatusPending     = "pending"      // uploaded, classification incomplete
	MonthStatusNeedsReview = "needs_review" // done but items need review
	MonthStatusReady       = "ready"        // uploaded + classified
)

type MonthStatusInfo struct {
	Month             string `json:"month"` // YYYY-MM
	Label             string `json:"label"`
	Status            string `json:"status"`
	TransactionCount  int64  `json:"transaction_count"`
	PendingCount      int64  `json:"pending_count"`
	NeedsReviewCount  int64  `json:"needs_review_count"`
	HasStatement      bool   `json:"has_statement"`
}

type YearCalendar struct {
	Year   int               `json:"year"`
	Years  []int             `json:"years"`
	Months []MonthStatusInfo `json:"months"`
}

func (s *TransactionService) Calendar(ctx context.Context, year int) (*YearCalendar, error) {
	if year < 1990 || year > 2100 {
		return nil, fmt.Errorf("invalid year")
	}

	years, err := s.listYears(ctx)
	if err != nil {
		return nil, err
	}
	hasYear := false
	for _, y := range years {
		if y == year {
			hasYear = true
			break
		}
	}
	if !hasYear {
		years = append([]int{year}, years...)
	}

	months := make([]MonthStatusInfo, 0, 12)
	for m := time.January; m <= time.December; m++ {
		key := fmt.Sprintf("%04d-%02d", year, int(m))
		start := time.Date(year, m, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)

		var total, pending, review int64
		_ = s.DB.WithContext(ctx).Model(&models.Transaction{}).
			Where("transaction_date >= ? AND transaction_date < ?", start, end).
			Count(&total).Error
		_ = s.DB.WithContext(ctx).Model(&models.Transaction{}).
			Where("transaction_date >= ? AND transaction_date < ? AND classification_status = ?",
				start, end, models.ClassificationPending).
			Count(&pending).Error
		_ = s.DB.WithContext(ctx).Model(&models.Transaction{}).
			Where("transaction_date >= ? AND transaction_date < ? AND classification_status = ?",
				start, end, models.ClassificationNeedsReview).
			Count(&review).Error

		info := MonthStatusInfo{
			Month:            key,
			Label:            start.Format("January"),
			TransactionCount: total,
			PendingCount:     pending,
			NeedsReviewCount: review,
			HasStatement:     total > 0,
		}
		switch {
		case total == 0:
			info.Status = MonthStatusEmpty
		case pending > 0:
			info.Status = MonthStatusPending
		case review > 0:
			info.Status = MonthStatusNeedsReview
		default:
			info.Status = MonthStatusReady
		}
		months = append(months, info)
	}

	return &YearCalendar{Year: year, Years: years, Months: months}, nil
}

func (s *TransactionService) listYears(ctx context.Context) ([]int, error) {
	now := time.Now().UTC().Year()
	var months []time.Time
	err := s.DB.WithContext(ctx).
		Model(&models.Import{}).
		Where("status = ? AND validated_month IS NOT NULL", models.ImportStatusCompleted).
		Distinct("validated_month").
		Pluck("validated_month", &months).Error
	if err != nil {
		return nil, err
	}
	seen := map[int]bool{now: true}
	out := []int{now}
	for _, m := range months {
		y := m.Year()
		if !seen[y] {
			seen[y] = true
			out = append(out, y)
		}
	}
	// sort desc
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] > out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

// ClearMonth deletes all imports (and cascaded transactions) for a calendar month.
func (s *ImportService) ClearMonth(ctx context.Context, monthKey string) error {
	start, err := validation.ParseMonthKey(monthKey)
	if err != nil {
		return err
	}
	var imports []models.Import
	if err := s.DB.WithContext(ctx).
		Where("validated_month = ?", start).
		Find(&imports).Error; err != nil {
		return err
	}
	for _, imp := range imports {
		_ = os.Remove(imp.StoragePath)
		if err := s.DB.WithContext(ctx).Delete(&imp).Error; err != nil {
			return err
		}
	}
	s.Log.Info("cleared month", "month", monthKey, "imports", len(imports))
	return nil
}

// ReclassifyMonth resets non-corrected txns to pending and enqueues classify jobs.
func (s *ImportService) ReclassifyMonth(ctx context.Context, monthKey string) (int64, error) {
	start, err := validation.ParseMonthKey(monthKey)
	if err != nil {
		return 0, err
	}
	end := start.AddDate(0, 1, 0)

	res := s.DB.WithContext(ctx).Model(&models.Transaction{}).
		Where("transaction_date >= ? AND transaction_date < ? AND classification_status <> ?",
			start, end, models.ClassificationCorrected).
		Updates(map[string]any{
			"classification_status": models.ClassificationPending,
			"category":              nil,
			"subcategory":           nil,
			"transaction_type":      nil,
			"updated_at":            time.Now().UTC(),
		})
	if res.Error != nil {
		return 0, res.Error
	}

	var importIDs []uuid.UUID
	_ = s.DB.WithContext(ctx).Model(&models.Transaction{}).
		Where("transaction_date >= ? AND transaction_date < ?", start, end).
		Distinct("import_id").
		Pluck("import_id", &importIDs).Error

	for _, id := range importIDs {
		task, err := jobs.NewClassifyTask(id)
		if err != nil {
			continue
		}
		if _, err := s.Asynq.EnqueueContext(ctx, task, asynq.MaxRetry(5), asynq.Timeout(30*time.Minute)); err != nil {
			s.Log.Error("enqueue reclassify failed", "import_id", id, "err", err)
		}
	}
	s.Log.Info("reclassify queued", "month", monthKey, "reset", res.RowsAffected, "imports", len(importIDs))
	return res.RowsAffected, nil
}
