package jobs

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	TaskImportStatement = "import:statement"
	TaskClassifyBatch   = "classify:batch"
)

type ImportPayload struct {
	ImportID uuid.UUID `json:"import_id"`
}

type ClassifyPayload struct {
	ImportID uuid.UUID `json:"import_id"`
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
