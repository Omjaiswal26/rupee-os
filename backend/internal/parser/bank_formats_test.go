package parser

import (
	"testing"
	"time"
)

func TestMapColumnsHDFCStyle(t *testing.T) {
	headers := []string{"Date", "Narration", "Chq./Ref.No.", "Value Dt", "Withdrawal Amt.", "Deposit Amt.", "Closing Balance"}
	cm, err := mapColumns(headers)
	if err != nil {
		t.Fatal(err)
	}
	if cm.date != 0 || cm.description != 1 || cm.debit != 4 || cm.credit != 5 {
		t.Fatalf("%+v", cm)
	}
}

func TestMapColumnsICICIStyle(t *testing.T) {
	headers := []string{"S No.", "Value Date", "Txn Posted Date", "Description", "Cr/Dr", "Transaction Amount(INR)", "Available Balance(INR)"}
	cm, err := mapColumns(headers)
	if err != nil {
		t.Fatal(err)
	}
	if cm.description < 0 || cm.amount < 0 || cm.crdr < 0 {
		t.Fatalf("%+v", cm)
	}
}

func TestParseSheetHDFCLike(t *testing.T) {
	rows := [][]string{
		{"Account Statement"},
		{"Account Number", "1234"},
		{},
		{"Date", "Narration", "Chq./Ref.No.", "Value Dt", "Withdrawal Amt.", "Deposit Amt.", "Closing Balance"},
		{"01/03/2024", "UPI-ZOMATO", "123", "01/03/2024", "450.00", "", "10000"},
		{"05/03/2024", "SALARY", "456", "05/03/2024", "", "50000.00", "60000"},
	}
	res, err := parseSheetRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Transactions) != 2 {
		t.Fatalf("got %d", len(res.Transactions))
	}
	if res.Transactions[0].Direction != "debit" || res.Transactions[1].Direction != "credit" {
		t.Fatalf("%+v", res.Transactions)
	}
}

func TestParseSheetICICILike(t *testing.T) {
	rows := [][]string{
		{"Value Date", "Description", "Cr/Dr", "Transaction Amount(INR)"},
		{"01-03-2024", "UPI ZOMATO", "DR", "250.00"},
		{"02-03-2024", "NEFT SALARY", "CR", "1000.00"},
	}
	res, err := parseSheetRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Transactions) != 2 {
		t.Fatalf("got %d", len(res.Transactions))
	}
}

func TestExcelSerialDate(t *testing.T) {
	// 2024-03-01 ≈ serial 45352 (Excel)
	d, err := ParseDate("45352")
	if err != nil {
		t.Fatal(err)
	}
	if d.Year() != 2024 || d.Month() != time.March {
		t.Fatalf("%v", d)
	}
}
