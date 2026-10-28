package validation

import (
	"fmt"
	"time"

	"github.com/rupee-os/rupee-os/internal/parser"
)

const (
	ErrCodeEmptyStatement    = "EMPTY_STATEMENT"
	ErrCodeInvalidDates      = "INVALID_DATES"
	ErrCodeMultiMonth        = "MULTI_MONTH"
	ErrCodeParseFailed       = "PARSE_FAILED"
	ErrCodeDuplicateImport   = "DUPLICATE_IMPORT"
	ErrCodeUnsupportedType   = "UNSUPPORTED_TYPE"
	ErrCodePasswordRequired  = "PASSWORD_REQUIRED"
	ErrCodeBadPassword       = "BAD_PASSWORD"
)

// Result is the outcome of one-calendar-month validation.
type Result struct {
	Month        time.Time // first day of the validated calendar month (UTC)
	Transactions []parser.ExtractedTransaction
}

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ValidateSingleMonth enforces that all transactions belong to exactly one calendar month.
// Month is determined from transaction dates only (not filename/title).
func ValidateSingleMonth(txns []parser.ExtractedTransaction) (*Result, error) {
	if len(txns) == 0 {
		return nil, &Error{
			Code:    ErrCodeEmptyStatement,
			Message: "statement contains no valid transactions; check that the file has date, description, and amount columns",
		}
	}

	type monthKey struct{ y int; m time.Month }
	months := map[monthKey]int{}
	var firstValid *time.Time

	for i, t := range txns {
		if t.Date.IsZero() {
			return nil, &Error{
				Code:    ErrCodeInvalidDates,
				Message: fmt.Sprintf("transaction %d has an invalid or missing date", i+1),
			}
		}
		y, m, _ := t.Date.Date()
		if y < 1990 || y > 2100 {
			return nil, &Error{
				Code:    ErrCodeInvalidDates,
				Message: fmt.Sprintf("transaction %d has an unreliable date %s", i+1, t.Date.Format("2006-01-02")),
			}
		}
		key := monthKey{y, m}
		months[key]++
		if firstValid == nil {
			d := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
			firstValid = &d
		}
	}

	if len(months) > 1 {
		parts := make([]string, 0, len(months))
		for k, count := range months {
			parts = append(parts, fmt.Sprintf("%04d-%02d (%d txns)", k.y, int(k.m), count))
		}
		return nil, &Error{
			Code: ErrCodeMultiMonth,
			Message: fmt.Sprintf(
				"statement spans %d calendar months (%v). Upload exactly one calendar month of transactions",
				len(months), parts,
			),
		}
	}

	return &Result{
		Month:        *firstValid,
		Transactions: txns,
	}, nil
}

// MonthKey formats a month as YYYY-MM.
func MonthKey(t time.Time) string {
	return t.Format("2006-01")
}

// ParseMonthKey parses YYYY-MM into first-of-month UTC.
func ParseMonthKey(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01", s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month %q; use YYYY-MM", s)
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}
