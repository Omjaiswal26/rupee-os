package validation

import (
	"testing"
	"time"

	"github.com/rupee-os/rupee-os/internal/parser"
)

func txn(y int, m time.Month, d int) parser.ExtractedTransaction {
	return parser.ExtractedTransaction{
		Date:        time.Date(y, m, d, 0, 0, 0, 0, time.UTC),
		Description: "test",
		AmountPaise: 10000,
		Direction:   "debit",
	}
}

func TestSingleMonthOK(t *testing.T) {
	txns := []parser.ExtractedTransaction{
		txn(2024, time.March, 1),
		txn(2024, time.March, 15),
		txn(2024, time.March, 31),
	}
	res, err := ValidateSingleMonth(txns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Month.Year() != 2024 || res.Month.Month() != time.March {
		t.Fatalf("month = %v", res.Month)
	}
}

func TestMultiMonthRejected(t *testing.T) {
	txns := []parser.ExtractedTransaction{
		txn(2024, time.March, 31),
		txn(2024, time.April, 1),
	}
	_, err := ValidateSingleMonth(txns)
	if err == nil {
		t.Fatal("expected multi-month error")
	}
	ve, ok := err.(*Error)
	if !ok || ve.Code != ErrCodeMultiMonth {
		t.Fatalf("got %v", err)
	}
}

func TestYearBoundaryRejected(t *testing.T) {
	txns := []parser.ExtractedTransaction{
		txn(2024, time.December, 31),
		txn(2025, time.January, 1),
	}
	_, err := ValidateSingleMonth(txns)
	ve, ok := err.(*Error)
	if !ok || ve.Code != ErrCodeMultiMonth {
		t.Fatalf("got %v", err)
	}
}

func TestEmptyRejected(t *testing.T) {
	_, err := ValidateSingleMonth(nil)
	ve, ok := err.(*Error)
	if !ok || ve.Code != ErrCodeEmptyStatement {
		t.Fatalf("got %v", err)
	}
}

func TestInvalidDateRejected(t *testing.T) {
	txns := []parser.ExtractedTransaction{
		{Date: time.Time{}, Description: "x", AmountPaise: 1, Direction: "debit"},
	}
	_, err := ValidateSingleMonth(txns)
	ve, ok := err.(*Error)
	if !ok || ve.Code != ErrCodeInvalidDates {
		t.Fatalf("got %v", err)
	}
}

func TestUnreliableYearRejected(t *testing.T) {
	txns := []parser.ExtractedTransaction{txn(1800, time.January, 1)}
	_, err := ValidateSingleMonth(txns)
	ve, ok := err.(*Error)
	if !ok || ve.Code != ErrCodeInvalidDates {
		t.Fatalf("got %v", err)
	}
}
