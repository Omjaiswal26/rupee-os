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
	Log      *slog.Logger
}

func (h *Handlers) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(jobs.TaskImportStatement, h.HandleImport)
	mux.HandleFunc(jobs.TaskClassifyBatch, h.HandleClassify)
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
