package parser

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// ExtractedTransaction is one row from a bank statement export.
type ExtractedTransaction struct {
	Date        time.Time
	Description string
	AmountPaise int64 // always absolute; Direction indicates sign
	Direction   string // debit | credit
	Raw         map[string]string
}

// ParseResult holds all extracted rows from a file.
type ParseResult struct {
	Transactions []ExtractedTransaction
	FileType     string
}

// StatementParser extracts transactions from statement files.
// password is only used for protected XLSX; never log it.
type StatementParser interface {
	Parse(filePath string, password string) (*ParseResult, error)
}

// DetectFileType returns xlsx or csv from extension.
func DetectFileType(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".xlsx", ".xls":
		return "xlsx", nil
	case ".csv":
		return "csv", nil
	default:
		return "", fmt.Errorf("unsupported file type %q; upload .xlsx or .csv", ext)
	}
}

// NewParser returns a parser for the given file type.
func NewParser(fileType string) (StatementParser, error) {
	switch strings.ToLower(fileType) {
	case "csv":
		return &CSVParser{}, nil
	case "xlsx", "xls":
		return &XLSXParser{}, nil
	default:
		return nil, fmt.Errorf("no parser for file type %q", fileType)
	}
}
