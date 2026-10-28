package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

type CSVParser struct{}

func (p *CSVParser) Parse(filePath string, _ string) (*ParseResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	var rows [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv: %w", err)
		}
		if len(rows) == 0 && len(record) > 0 {
			record[0] = strings.TrimPrefix(record[0], "\ufeff")
		}
		rows = append(rows, record)
	}

	res, err := parseSheetRows(rows)
	if err != nil {
		return nil, err
	}
	res.FileType = "csv"
	return res, nil
}

func isEmptyRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
