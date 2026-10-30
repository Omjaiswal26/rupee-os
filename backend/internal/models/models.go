package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ImportStatus string

const (
	ImportStatusQueued     ImportStatus = "queued"
	ImportStatusProcessing ImportStatus = "processing"
	ImportStatusCompleted  ImportStatus = "completed"
	ImportStatusFailed     ImportStatus = "failed"
)

type ClassificationStatus string

const (
	ClassificationPending     ClassificationStatus = "pending"
	ClassificationClassified  ClassificationStatus = "classified"
	ClassificationNeedsReview ClassificationStatus = "needs_review"
	ClassificationCorrected   ClassificationStatus = "corrected"
)

type Direction string

const (
	DirectionDebit  Direction = "debit"
	DirectionCredit Direction = "credit"
)

type Import struct {
	ID               uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Filename         string       `gorm:"not null" json:"filename"`
	FileHash         string       `gorm:"not null;uniqueIndex" json:"file_hash"`
	StoragePath      string       `gorm:"not null" json:"-"`
	FileType         string       `gorm:"not null" json:"file_type"`
	ValidatedMonth   *time.Time   `gorm:"type:date" json:"validated_month"`
	Status           ImportStatus `gorm:"not null;default:queued" json:"status"`
	TransactionCount int          `gorm:"not null;default:0" json:"transaction_count"`
	ErrorCode        *string      `json:"error_code,omitempty"`
	ErrorMessage     *string      `json:"error_message,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

func (Import) TableName() string { return "imports" }

type Transaction struct {
	ID                   uuid.UUID            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ImportID             uuid.UUID            `gorm:"type:uuid;not null;index" json:"import_id"`
	TransactionDate      time.Time            `gorm:"type:date;not null;index" json:"transaction_date"`
	OriginalDescription  string               `gorm:"not null" json:"original_description"`
	Merchant             *string              `json:"merchant,omitempty"`
	AmountPaise          int64                `gorm:"not null" json:"amount_paise"`
	Currency             string               `gorm:"not null;default:INR" json:"currency"`
	Direction            Direction            `gorm:"not null" json:"direction"`
	TransactionType      *string              `json:"transaction_type,omitempty"`
	Category             *string              `json:"category,omitempty"`
	Subcategory          *string              `json:"subcategory,omitempty"`
	ClassificationStatus ClassificationStatus `gorm:"not null;default:pending" json:"classification_status"`
	DedupHash            string               `gorm:"not null;uniqueIndex" json:"-"`
	RawData              datatypes.JSON       `gorm:"type:jsonb;not null;default:'{}'" json:"raw_data"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
}

func (Transaction) TableName() string { return "transactions" }

type Classification struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TransactionID        uuid.UUID `gorm:"type:uuid;not null;index" json:"transaction_id"`
	ModelName            string    `gorm:"not null" json:"model_name"`
	ModelVersion         *string   `json:"model_version,omitempty"`
	PredictedCategory    *string   `json:"predicted_category,omitempty"`
	PredictedSubcategory *string   `json:"predicted_subcategory,omitempty"`
	PredictedType        *string   `json:"predicted_type,omitempty"`
	Confidence           float64   `gorm:"type:numeric(5,4);not null;default:0" json:"confidence"`
	Reasoning            *string   `json:"reasoning,omitempty"`
	NeedsReview          bool      `gorm:"not null;default:false" json:"needs_review"`
	CreatedAt            time.Time `json:"created_at"`
}

func (Classification) TableName() string { return "classifications" }

type UserFeedback struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TransactionID        uuid.UUID `gorm:"type:uuid;not null;index" json:"transaction_id"`
	PreviousCategory     *string   `json:"previous_category,omitempty"`
	PreviousSubcategory  *string   `json:"previous_subcategory,omitempty"`
	CorrectedCategory    string    `gorm:"not null" json:"corrected_category"`
	CorrectedSubcategory *string   `json:"corrected_subcategory,omitempty"`
	CorrectedType        *string   `json:"corrected_type,omitempty"`
	Note                 *string   `json:"note,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

func (UserFeedback) TableName() string { return "user_feedback" }

type UserRule struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Pattern         string    `gorm:"not null" json:"pattern"`
	MatchType       string    `gorm:"not null;default:contains" json:"match_type"`
	Category        string    `gorm:"not null" json:"category"`
	Subcategory     *string   `json:"subcategory,omitempty"`
	TransactionType *string   `json:"transaction_type,omitempty"`
	Priority        int       `gorm:"not null;default:0" json:"priority"`
	Active          bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (UserRule) TableName() string { return "user_rules" }

// MonthHealthReport stores a per-month financial health score + AI tip.
type MonthHealthReport struct {
	Month        time.Time      `gorm:"type:date;primaryKey" json:"month"`
	Score        int            `gorm:"not null" json:"score"`
	Verdict      string         `gorm:"not null" json:"verdict"`
	Tip          string         `gorm:"not null;default:''" json:"tip"`
	Signals      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"signals"`
	ModelName    string         `gorm:"not null;default:''" json:"model_name"`
	GeneratedAt  time.Time      `gorm:"not null" json:"generated_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (MonthHealthReport) TableName() string { return "month_health_reports" }

// YearHealthReport stores a year-scoped financial health score + AI tip.
type YearHealthReport struct {
	Year        int            `gorm:"primaryKey" json:"year"`
	Score       int            `gorm:"not null" json:"score"`
	Verdict     string         `gorm:"not null" json:"verdict"`
	Tip         string         `gorm:"not null;default:''" json:"tip"`
	Signals     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"signals"`
	ModelName   string         `gorm:"not null;default:''" json:"model_name"`
	GeneratedAt time.Time      `gorm:"not null" json:"generated_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (YearHealthReport) TableName() string { return "year_health_reports" }
