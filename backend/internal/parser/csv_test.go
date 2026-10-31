package parser

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCSVParseSingleMonth(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample_single_month.csv")
	res, err := (&CSVParser{}).Parse(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Transactions) != 15 {
		t.Fatalf("got %d txns", len(res.Transactions))
	}
	month := res.Transactions[0].Date.Month()
	for _, txn := range res.Transactions {
		if txn.Date.Month() != month {
			t.Fatalf("expected single month, got %v and %v", month, txn.Date.Month())
		}
		if txn.Date.Year() != 2026 {
			t.Fatalf("year %d", txn.Date.Year())
		}
	}
	// Boundary: Oct 1 … same month.
	if res.Transactions[0].Date.Day() != 1 {
		t.Fatalf("first day %d", res.Transactions[0].Date.Day())
	}
}

func TestCSVParseMultiMonthFile(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample_multi_month.csv")
	res, err := (&CSVParser{}).Parse(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Transactions) != 2 {
		t.Fatalf("got %d", len(res.Transactions))
	}
	if res.Transactions[0].Date.Month() == res.Transactions[1].Date.Month() &&
		res.Transactions[0].Date.Year() == res.Transactions[1].Date.Year() {
		t.Fatal("expected different months in sample")
	}
}

func TestDetectFileType(t *testing.T) {
	ft, err := DetectFileType("stmt.XLSX")
	if err != nil || ft != "xlsx" {
		t.Fatalf("%s %v", ft, err)
	}
	if _, err := DetectFileType("stmt.pdf"); err == nil {
		t.Fatal("expected error for pdf")
	}
}

func TestParseDateIndian(t *testing.T) {
	d, err := ParseDate("31/03/2024")
	if err != nil {
		t.Fatal(err)
	}
	if d.Year() != 2024 || d.Month() != time.March || d.Day() != 31 {
		t.Fatalf("%v", d)
	}
}
