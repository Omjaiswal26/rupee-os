package services

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rupee-os/rupee-os/internal/classification"
	"github.com/rupee-os/rupee-os/internal/models"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type ClassifyService struct {
	DB          *gorm.DB
	Client      *classification.Client
	Log         *slog.Logger
	Concurrency int // parallel Ollama classify calls; default 2
}

func (s *ClassifyService) concurrency() int {
	if s.Concurrency < 1 {
		return 2
	}
	return s.Concurrency
}

func (s *ClassifyService) ClassifyImport(ctx context.Context, importID uuid.UUID) error {
	var txns []models.Transaction
	if err := s.DB.WithContext(ctx).
		Where("import_id = ? AND classification_status = ?", importID, models.ClassificationPending).
		Order("transaction_date ASC").
		Find(&txns).Error; err != nil {
		return err
	}
	if len(txns) == 0 {
		s.Log.Info("no pending transactions to classify", "import_id", importID)
		return nil
	}

	rules, err := s.loadRules(ctx)
	if err != nil {
		return err
	}

	limit := s.concurrency()
	s.Log.Info("classify import starting", "import_id", importID, "pending", len(txns), "concurrency", limit)

	var ok, fail atomic.Int64
	sem := make(chan struct{}, limit)
	g, gctx := errgroup.WithContext(ctx)

	for i := range txns {
		txn := txns[i]
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-gctx.Done():
				return gctx.Err()
			}
			defer func() { <-sem }()

			if err := s.classifyOne(gctx, &txn, rules); err != nil {
				fail.Add(1)
				s.Log.Warn("classify failed", "txn_id", txn.ID, "err", err)
				// Leave as pending so a retry can pick it up.
				return nil
			}
			ok.Add(1)
			return nil
		})
	}

	_ = g.Wait()
	s.Log.Info("classify import finished",
		"import_id", importID,
		"ok", ok.Load(),
		"failed", fail.Load(),
		"total", len(txns),
		"concurrency", limit,
	)
	return nil
}

func (s *ClassifyService) loadRules(ctx context.Context) ([]models.UserRule, error) {
	var rules []models.UserRule
	err := s.DB.WithContext(ctx).
		Where("active = ?", true).
		Order("priority DESC, created_at ASC").
		Find(&rules).Error
	return rules, err
}

func (s *ClassifyService) classifyOne(ctx context.Context, txn *models.Transaction, rules []models.UserRule) error {
	if pred := matchRule(txn.OriginalDescription, rules); pred != nil {
		if err := s.applyPrediction(ctx, txn, pred, "user_rule", nil, false); err != nil {
			return err
		}
		s.Log.Info("classified",
			"txn_id", txn.ID,
			"source", "user_rule",
			"category", pred.Category,
			"type", pred.Type,
			"confidence", pred.Confidence,
			"status", models.ClassificationClassified,
		)
		return nil
	}

	if pred := classification.MatchMerchantKeyword(txn.OriginalDescription); pred != nil {
		if err := s.applyPrediction(ctx, txn, pred, "merchant_keyword", &pred.Reasoning, false); err != nil {
			return err
		}
		s.Log.Info("classified",
			"txn_id", txn.ID,
			"source", "merchant_keyword",
			"category", pred.Category,
			"type", pred.Type,
			"confidence", pred.Confidence,
			"status", models.ClassificationClassified,
			"reasoning", truncateReason(pred.Reasoning, 80),
		)
		return nil
	}

	pred, err := s.Client.Classify(ctx, txn.OriginalDescription, txn.AmountPaise, string(txn.Direction))
	if err != nil {
		return err
	}
	needsReview := s.Client.NeedsReview(pred)
	if err := s.applyPrediction(ctx, txn, pred, s.Client.Model, &pred.Reasoning, needsReview); err != nil {
		return err
	}
	status := models.ClassificationClassified
	if needsReview {
		status = models.ClassificationNeedsReview
	}
	s.Log.Info("classified",
		"txn_id", txn.ID,
		"source", s.Client.Model,
		"category", pred.Category,
		"subcategory", pred.Subcategory,
		"type", pred.Type,
		"confidence", pred.Confidence,
		"status", status,
		"reasoning", truncateReason(pred.Reasoning, 80),
	)
	return nil
}

func truncateReason(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (s *ClassifyService) applyPrediction(
	ctx context.Context,
	txn *models.Transaction,
	pred *classification.Prediction,
	modelName string,
	reasoning *string,
	needsReview bool,
) error {
	status := models.ClassificationClassified
	if needsReview {
		status = models.ClassificationNeedsReview
	}

	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cat := pred.Category
		sub := pred.Subcategory
		typ := pred.Type
		updates := map[string]any{
			"category":              cat,
			"subcategory":           nullIfEmpty(sub),
			"transaction_type":      typ,
			"classification_status": status,
			"updated_at":            time.Now().UTC(),
		}
		if name := classification.ExtractPayeeCandidate(txn.OriginalDescription); name != "" {
			updates["merchant"] = name
		}
		if err := tx.Model(&models.Transaction{}).Where("id = ?", txn.ID).Updates(updates).Error; err != nil {
			return err
		}
		rec := models.Classification{
			ID:                   uuid.New(),
			TransactionID:        txn.ID,
			ModelName:            modelName,
			PredictedCategory:    &cat,
			PredictedSubcategory: nullIfEmpty(sub),
			PredictedType:        &typ,
			Confidence:           pred.Confidence,
			Reasoning:            reasoning,
			NeedsReview:          needsReview,
		}
		return tx.Create(&rec).Error
	})
}

func matchRule(desc string, rules []models.UserRule) *classification.Prediction {
	lower := strings.ToLower(desc)
	for _, r := range rules {
		pat := strings.ToLower(r.Pattern)
		matched := false
		switch r.MatchType {
		case "equals":
			matched = lower == pat
		case "contains", "":
			matched = strings.Contains(lower, pat)
		default:
			matched = strings.Contains(lower, pat)
		}
		if !matched {
			continue
		}
		typ := "expense"
		if r.TransactionType != nil && *r.TransactionType != "" {
			typ = *r.TransactionType
		}
		sub := ""
		if r.Subcategory != nil {
			sub = *r.Subcategory
		}
		return &classification.Prediction{
			Category:    r.Category,
			Subcategory: sub,
			Type:        typ,
			Confidence:  1.0,
			Reasoning:   "matched user rule: " + r.Pattern,
		}
	}
	return nil
}

func nullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
