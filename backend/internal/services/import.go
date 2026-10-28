package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rupee-os/rupee-os/internal/jobs"
	"github.com/rupee-os/rupee-os/internal/models"
	"github.com/rupee-os/rupee-os/internal/parser"
	"github.com/rupee-os/rupee-os/internal/validation"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ImportService struct {
	DB        *gorm.DB
	Asynq     *asynq.Client
	UploadDir string
	Log       *slog.Logger
}

type CreateImportInput struct {
	Filename     string
	TempPath     string
	Password     string // never logged or stored; used only to unlock XLSX at upload
	ReplaceMonth string // optional YYYY-MM — clear that month before import (re-upload)
}

func (s *ImportService) Create(ctx context.Context, in CreateImportInput) (*models.Import, error) {
	ft, err := parser.DetectFileType(in.Filename)
	if err != nil {
		return nil, &validation.Error{Code: validation.ErrCodeUnsupportedType, Message: err.Error()}
	}

	hash, err := HashFile(in.TempPath)
	if err != nil {
		return nil, fmt.Errorf("hash file: %w", err)
	}

	// Always allow re-uploading the same file: remove prior import with this hash.
	var existing models.Import
	err = s.DB.WithContext(ctx).Where("file_hash = ?", hash).First(&existing).Error
	if err == nil {
		_ = os.Remove(existing.StoragePath)
		_ = s.DB.WithContext(ctx).Delete(&existing).Error
		s.Log.Info("replaced previous import with same file hash", "old_import_id", existing.ID)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if in.ReplaceMonth != "" {
		if _, err := validation.ParseMonthKey(in.ReplaceMonth); err != nil {
			return nil, &validation.Error{Code: validation.ErrCodeParseFailed, Message: err.Error()}
		}
		if err := s.ClearMonth(ctx, in.ReplaceMonth); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(s.UploadDir, 0o750); err != nil {
		return nil, err
	}
	id := uuid.New()
	destPath := filepath.Join(s.UploadDir, id.String()+filepath.Ext(in.Filename))

	// Unlock password-protected XLSX into an unprotected copy for the worker.
	// Password is never written to DB, logs, or disk metadata.
	if ft == "xlsx" {
		if err := parser.NormalizeXLSX(in.TempPath, destPath, in.Password); err != nil {
			if pe, ok := err.(*parser.PasswordError); ok {
				code := validation.ErrCodeBadPassword
				if pe.Code == "PASSWORD_REQUIRED" {
					code = validation.ErrCodePasswordRequired
				}
				return nil, &validation.Error{Code: code, Message: pe.Message}
			}
			return nil, &validation.Error{Code: validation.ErrCodeParseFailed, Message: err.Error()}
		}
		_ = os.Remove(in.TempPath)
	} else if err := moveFile(in.TempPath, destPath); err != nil {
		return nil, err
	}

	imp := models.Import{
		ID:          id,
		Filename:    filepath.Base(in.Filename),
		FileHash:    hash,
		StoragePath: destPath,
		FileType:    ft,
		Status:      models.ImportStatusQueued,
	}
	if err := s.DB.WithContext(ctx).Create(&imp).Error; err != nil {
		_ = os.Remove(destPath)
		return nil, err
	}

	task, err := jobs.NewImportTask(imp.ID)
	if err != nil {
		return nil, err
	}
	if _, err := s.Asynq.EnqueueContext(ctx, task, asynq.MaxRetry(3), asynq.Timeout(10*time.Minute)); err != nil {
		return nil, fmt.Errorf("enqueue import: %w", err)
	}
	s.Log.Info("import queued", "import_id", imp.ID, "file_type", ft, "password_provided", strings.TrimSpace(in.Password) != "")
	return &imp, nil
}

func (s *ImportService) Get(ctx context.Context, id uuid.UUID) (*models.Import, error) {
	var imp models.Import
	if err := s.DB.WithContext(ctx).First(&imp, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &imp, nil
}

func (s *ImportService) List(ctx context.Context) ([]models.Import, error) {
	var items []models.Import
	err := s.DB.WithContext(ctx).Order("created_at DESC").Find(&items).Error
	return items, err
}

// ProcessImport: parse → validate one month → store → enqueue classify.
func (s *ImportService) ProcessImport(ctx context.Context, importID uuid.UUID) error {
	var imp models.Import
	if err := s.DB.WithContext(ctx).First(&imp, "id = ?", importID).Error; err != nil {
		return err
	}
	if imp.Status == models.ImportStatusCompleted {
		s.Log.Info("import already completed (idempotent)", "import_id", importID)
		return nil
	}

	_ = s.DB.WithContext(ctx).Model(&imp).Updates(map[string]any{
		"status":     models.ImportStatusProcessing,
		"updated_at": time.Now().UTC(),
	}).Error

	fail := func(code, msg string) error {
		_ = s.DB.WithContext(ctx).Model(&models.Import{}).Where("id = ?", importID).Updates(map[string]any{
			"status":        models.ImportStatusFailed,
			"error_code":    code,
			"error_message": msg,
			"updated_at":    time.Now().UTC(),
		}).Error
		s.Log.Warn("import failed", "import_id", importID, "code", code)
		return &validation.Error{Code: code, Message: msg}
	}

	p, err := parser.NewParser(imp.FileType)
	if err != nil {
		return fail(validation.ErrCodeUnsupportedType, err.Error())
	}
	parsed, err := p.Parse(imp.StoragePath, "")
	if err != nil {
		if pe, ok := err.(*parser.PasswordError); ok {
			return fail(pe.Code, pe.Message)
		}
		return fail(validation.ErrCodeParseFailed, "failed to parse statement: "+err.Error())
	}

	validated, err := validation.ValidateSingleMonth(parsed.Transactions)
	if err != nil {
		if ve, ok := err.(*validation.Error); ok {
			return fail(ve.Code, ve.Message)
		}
		return fail(validation.ErrCodeParseFailed, err.Error())
	}

	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inserted := 0
		for _, et := range validated.Transactions {
			raw, _ := json.Marshal(et.Raw)
			row := models.Transaction{
				ID:                   uuid.New(),
				ImportID:             importID,
				TransactionDate:      et.Date,
				OriginalDescription:  et.Description,
				AmountPaise:          et.AmountPaise,
				Currency:             "INR",
				Direction:            models.Direction(et.Direction),
				ClassificationStatus: models.ClassificationPending,
				DedupHash:            TransactionDedupHash(et),
				RawData:              datatypes.JSON(raw),
			}
			res := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "dedup_hash"}},
				DoNothing: true,
			}).Create(&row)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected > 0 {
				inserted++
			}
		}
		return tx.Model(&models.Import{}).Where("id = ?", importID).Updates(map[string]any{
			"validated_month":   validated.Month,
			"transaction_count": inserted,
			"status":            models.ImportStatusCompleted,
			"error_code":        nil,
			"error_message":     nil,
			"updated_at":        time.Now().UTC(),
		}).Error
	})
	if err != nil {
		return fail(validation.ErrCodeParseFailed, err.Error())
	}

	task, err := jobs.NewClassifyTask(importID)
	if err != nil {
		return err
	}
	if _, err := s.Asynq.EnqueueContext(ctx, task, asynq.MaxRetry(5), asynq.Timeout(30*time.Minute)); err != nil {
		s.Log.Error("enqueue classify failed", "import_id", importID, "err", err)
		return err
	}
	s.Log.Info("import completed", "import_id", importID, "month", validated.Month.Format("2006-01"))
	return nil
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Remove(src)
}
