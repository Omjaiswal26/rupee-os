package jobs

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	TaskImportStatement = "import:statement"
	TaskClassifyBatch   = "classify:batch"
	TaskHealthMonth     = "health:month"
	TaskHealthYear      = "health:year"
)

type ImportPayload struct {
	ImportID uuid.UUID `json:"import_id"`
}

type ClassifyPayload struct {
	ImportID uuid.UUID `json:"import_id"`
}

type HealthMonthPayload struct {
	Month string `json:"month"` // YYYY-MM
}

type HealthYearPayload struct {
	Year int `json:"year"`
}

func NewImportTask(importID uuid.UUID) (*asynq.Task, error) {
	body, err := json.Marshal(ImportPayload{ImportID: importID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskImportStatement, body), nil
}

func NewClassifyTask(importID uuid.UUID) (*asynq.Task, error) {
	body, err := json.Marshal(ClassifyPayload{ImportID: importID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskClassifyBatch, body), nil
}

func NewHealthMonthTask(monthKey string) (*asynq.Task, error) {
	body, err := json.Marshal(HealthMonthPayload{Month: monthKey})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskHealthMonth, body), nil
}

func NewHealthYearTask(year int) (*asynq.Task, error) {
	body, err := json.Marshal(HealthYearPayload{Year: year})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskHealthYear, body), nil
}

func ParseImportPayload(t *asynq.Task) (ImportPayload, error) {
	var p ImportPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}

func ParseClassifyPayload(t *asynq.Task) (ClassifyPayload, error) {
	var p ClassifyPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}

func ParseHealthMonthPayload(t *asynq.Task) (HealthMonthPayload, error) {
	var p HealthMonthPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}

func ParseHealthYearPayload(t *asynq.Task) (HealthYearPayload, error) {
	var p HealthYearPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}
