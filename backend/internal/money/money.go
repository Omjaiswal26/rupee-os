package money

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseINR converts Indian amount strings to paise (1 INR = 100 paise).
// Accepts: "1,234.56", "1234.56", "1,234", "-500.00", "₹1,234.50"
func ParseINR(s string) (paise int64, err error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "₹", "")
	s = strings.ReplaceAll(s, "Rs.", "")
	s = strings.ReplaceAll(s, "Rs", "")
	s = strings.ReplaceAll(s, "INR", "")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	if s == "" || s == "-" || s == "." {
		return 0, fmt.Errorf("empty amount")
	}

	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg = true
		s = strings.TrimPrefix(s, "(")
		s = strings.TrimSuffix(s, ")")
	}
	if strings.HasPrefix(s, "-") {
		neg = true
		s = strings.TrimPrefix(s, "-")
	}
	if strings.HasPrefix(s, "+") {
		s = strings.TrimPrefix(s, "+")
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", s, err)
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("invalid amount %q", s)
	}

	// Round to nearest paise to avoid float dust.
	p := int64(math.Round(f * 100))
	if neg {
		p = -p
	}
	return p, nil
}

// FormatINR formats paise as a rupee string with 2 decimals (absolute value).
func FormatINR(paise int64) string {
	neg := paise < 0
	if neg {
		paise = -paise
	}
	rupees := paise / 100
	frac := paise % 100
	out := fmt.Sprintf("%d.%02d", rupees, frac)
	if neg {
		return "-" + out
	}
	return out
}

// Abs returns absolute paise.
func Abs(paise int64) int64 {
	if paise < 0 {
		return -paise
	}
	return paise
}
