package validation_test

import (
	"path/filepath"
	"testing"

	"github.com/rupee-os/rupee-os/internal/parser"
	"github.com/rupee-os/rupee-os/internal/validation"
)

func TestValidateParsedSingleMonthCSV(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample_single_month.csv")
	res, err := (&parser.CSVParser{}).Parse(path, "")
	if err != nil {
		t.Fatal(err)
	}
	out, err := validation.ValidateSingleMonth(res.Transactions)
	if err != nil {
		t.Fatal(err)
	}
	if out.Month.Format("2006-01") != "2024-03" {
		t.Fatalf("month %v", out.Month)
	}
}

func TestValidateParsedMultiMonthCSV(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample_multi_month.csv")
	res, err := (&parser.CSVParser{}).Parse(path, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = validation.ValidateSingleMonth(res.Transactions)
	ve, ok := err.(*validation.Error)
	if !ok || ve.Code != validation.ErrCodeMultiMonth {
		t.Fatalf("got %v", err)
	}
}
