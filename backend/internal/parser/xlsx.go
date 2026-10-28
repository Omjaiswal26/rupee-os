package parser

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

type XLSXParser struct{}

func (p *XLSXParser) Parse(filePath string, password string) (*ParseResult, error) {
	f, err := openWorkbook(filePath, password)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("xlsx has no sheets")
	}

	var lastErr error
	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil {
			lastErr = err
			continue
		}
		res, err := parseSheetRows(rows)
		if err != nil {
			lastErr = err
			continue
		}
		if len(res.Transactions) > 0 {
			res.FileType = "xlsx"
			return res, nil
		}
		lastErr = fmt.Errorf("sheet %q: no transactions found", sheet)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no transactions found in workbook")
}

// NormalizeXLSX opens a (possibly password-protected) workbook and writes an
// unprotected copy to destPath. Callers must never log or persist the password.
func NormalizeXLSX(srcPath, destPath, password string) error {
	f, err := openWorkbook(srcPath, password)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.SaveAs(destPath); err != nil {
		return fmt.Errorf("save unlocked xlsx: %w", err)
	}
	return nil
}

func openWorkbook(path, password string) (*excelize.File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read xlsx: %w", err)
	}
	password = strings.TrimSpace(password)

	isZIP := len(raw) >= 2 && raw[0] == 'P' && raw[1] == 'K'
	isOLE := len(raw) >= 4 && raw[0] == 0xD0 && raw[1] == 0xCF && raw[2] == 0x11 && raw[3] == 0xE0

	if isOLE {
		if password == "" {
			return nil, &PasswordError{
				Code:    "PASSWORD_REQUIRED",
				Message: "this XLSX is password-protected; enter the statement password and try again",
			}
		}
		decrypted, derr := excelize.Decrypt(raw, &excelize.Options{Password: password})
		if derr != nil {
			if errors.Is(derr, excelize.ErrWorkbookPassword) || strings.Contains(derr.Error(), "password") {
				return nil, &PasswordError{
					Code:    "BAD_PASSWORD",
					Message: "incorrect spreadsheet password",
				}
			}
			return nil, &PasswordError{
				Code: "BAD_PASSWORD",
				Message: "could not decrypt XLSX with that password. " +
					"Try again, or download CSV from your bank instead.",
			}
		}
		f, oerr := excelize.OpenReader(bytes.NewReader(decrypted))
		if oerr != nil {
			return nil, fmt.Errorf("open decrypted xlsx: %w", oerr)
		}
		return f, nil
	}

	if !isZIP {
		return nil, fmt.Errorf(
			"file is not a valid Excel workbook (unexpected format). Download XLSX or CSV from your bank again",
		)
	}

	var opts []excelize.Options
	if password != "" {
		opts = append(opts, excelize.Options{Password: password})
	}
	f, err := excelize.OpenReader(bytes.NewReader(raw), opts...)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	return f, nil
}

type PasswordError struct {
	Code    string
	Message string
}

func (e *PasswordError) Error() string { return e.Message }

func parseSheetRows(rows [][]string) (*ParseResult, error) {
	var headers []string
	var cm columnMap
	var txns []ExtractedTransaction
	headerRow := -1

	for i, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		if headers == nil {
			// Prefer explicit header-like rows; also try mapping any row with a date-ish cell label.
			if mapped, err := mapColumns(row); err == nil {
				headers = row
				cm = mapped
				headerRow = i
				continue
			}
			continue
		}
		txn, err := rowToTransaction(headers, padRow(row, len(headers)), cm)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		if txn != nil {
			txns = append(txns, *txn)
		}
	}
	if headers == nil {
		return nil, fmt.Errorf("could not find a valid header row; sample: %s", sampleRows(rows, 8))
	}
	_ = headerRow
	return &ParseResult{Transactions: txns}, nil
}

func sampleRows(rows [][]string, n int) string {
	parts := make([]string, 0, n)
	for _, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		// Cap cell length to avoid huge errors.
		cells := make([]string, 0, len(row))
		for _, c := range row {
			c = strings.TrimSpace(c)
			if len(c) > 40 {
				c = c[:40] + "…"
			}
			cells = append(cells, c)
		}
		parts = append(parts, "["+strings.Join(cells, " | ")+"]")
		if len(parts) >= n {
			break
		}
	}
	if len(parts) == 0 {
		return "(empty sheet)"
	}
	return strings.Join(parts, " ; ")
}

func padRow(row []string, n int) []string {
	if len(row) >= n {
		return row
	}
	out := make([]string, n)
	copy(out, row)
	return out
}
