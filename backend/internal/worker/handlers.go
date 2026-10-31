package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/rupee-os/rupee-os/internal/jobs"
	"github.com/rupee-os/rupee-os/internal/services"
	"github.com/rupee-os/rupee-os/internal/validation"
)

// Handlers wires Asynq tasks to services (kept out of jobs/ to avoid import cycles).
type Handlers struct {
	Imports  *services.ImportService
	Classify *services.ClassifyService
	Health   *services.HealthService
	Log      *slog.Logger
}

func (h *Handlers) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(jobs.TaskImportStatement, h.HandleImport)
	mux.HandleFunc(jobs.TaskClassifyBatch, h.HandleClassify)
	mux.HandleFunc(jobs.TaskHealthMonth, h.HandleHealthMonth)
	mux.HandleFunc(jobs.TaskHealthYear, h.HandleHealthYear)
}

func (h *Handlers) HandleImport(ctx context.Context, t *asynq.Task) error {
	p, err := jobs.ParseImportPayload(t)
	if err != nil {
		return fmt.Errorf("bad import payload: %w", err)
	}
	h.Log.Info("processing import job", "import_id", p.ImportID)
	err = h.Imports.ProcessImport(ctx, p.ImportID)
	if err != nil {
		if _, ok := err.(*validation.Error); ok {
			h.Log.Warn("import validation failed (no retry)", "import_id", p.ImportID, "err", err)
			return nil
		}
		return err
	}
	return nil
}

func (h *Handlers) HandleClassify(ctx context.Context, t *asynq.Task) error {
	p, err := jobs.ParseClassifyPayload(t)
	if err != nil {
		return fmt.Errorf("bad classify payload: %w", err)
	}
	h.Log.Info("processing classify job", "import_id", p.ImportID)
	return h.Classify.ClassifyImport(ctx, p.ImportID)
}

func (h *Handlers) HandleHealthMonth(ctx context.Context, t *asynq.Task) error {
	p, err := jobs.ParseHealthMonthPayload(t)
	if err != nil {
		return fmt.Errorf("bad health month payload: %w", err)
	}
	if h.Health == nil {
		return fmt.Errorf("health service not configured")
	}
	h.Log.Info("processing health month job", "month", p.Month)
	if _, err := h.Health.Generate(ctx, p.Month); err != nil {
		h.Log.Warn("health month failed", "month", p.Month, "err", err)
		return err
	}
	// Run year in the same job so a later month isn't skipped while an earlier
	// year TaskID is still held. health queue concurrency is 1, so months serialize.
	start, err := validation.ParseMonthKey(p.Month)
	if err != nil {
		return fmt.Errorf("bad month key after health: %w", err)
	}
	year := start.UTC().Year()
	h.Log.Info("processing health year after month", "year", year, "month", p.Month)
	if _, err := h.Health.GenerateYearHealth(ctx, year); err != nil {
		h.Log.Warn("health year failed", "year", year, "err", err)
		return err
	}
	return nil
}

func (h *Handlers) HandleHealthYear(ctx context.Context, t *asynq.Task) error {
	p, err := jobs.ParseHealthYearPayload(t)
	if err != nil {
		return fmt.Errorf("bad health year payload: %w", err)
	}
	if h.Health == nil {
		return fmt.Errorf("health service not configured")
	}
	h.Log.Info("processing health year job", "year", p.Year)
	if _, err := h.Health.GenerateYearHealth(ctx, p.Year); err != nil {
		h.Log.Warn("health year failed", "year", p.Year, "err", err)
		return err
	}
	return nil
}
