package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var dateLayouts = []string{
	"02/01/2006",
	"2/1/2006",
	"02-01-2006",
	"2-1-2006",
	"02.01.2006",
	"2006-01-02",
	"02 Jan 2006",
	"2 Jan 2006",
	"02-Jan-2006",
	"2-Jan-2006",
	"02-Jan-06",
	"2-Jan-06",
	"01/02/2006",
	"2006/01/02",
	"02/01/06",
	"2/1/06",
	"02-01-06",
	time.RFC3339,
}

func ParseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	// Excel serial date (days since 1899-12-30), sometimes returned as "45321" or "45321.0".
	if serial, ok := parseExcelSerial(s); ok {
		return serial, nil
	}
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			y, m, d := t.Date()
			return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date %q", s)
}

func parseExcelSerial(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "/") || strings.Contains(s, "-") || strings.ContainsAny(s, "AbtcdefghijklmopqrsuvwxyzABTCDEFGHIJKLMOPQRSUV") {
		return time.Time{}, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return time.Time{}, false
	}
	// Reasonable Excel serial range ~1990-01-01 .. 2100-01-01
	if f < 32874 || f > 73050 {
		return time.Time{}, false
	}
	// Excel epoch quirk: day 0 = 1899-12-30
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	days := int(f)
	t := base.AddDate(0, 0, days)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true
}
