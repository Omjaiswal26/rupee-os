package parser

import (
	"strings"

	"github.com/rupee-os/rupee-os/internal/money"
	"github.com/rupee-os/rupee-os/internal/models"
)

type columnMap struct {
	date        int
	description int
	debit       int
	credit      int
	amount      int
	crdr        int // single amount + Cr/Dr marker (ICICI-style)
	balance     int
}

func normalizeHeader(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.ReplaceAll(h, "\n", " ")
	h = strings.ReplaceAll(h, ".", " ")
	h = strings.ReplaceAll(h, "/", " ")
	h = strings.ReplaceAll(h, "(", " ")
	h = strings.ReplaceAll(h, ")", " ")
	h = strings.Join(strings.Fields(h), " ")
	return h
}

func mapColumns(headers []string) (columnMap, error) {
	cm := columnMap{date: -1, description: -1, debit: -1, credit: -1, amount: -1, crdr: -1, balance: -1}
	for i, h := range headers {
		n := normalizeHeader(h)
		if n == "" {
			continue
		}
		switch {
		case cm.date < 0 && isDateHeader(n):
			cm.date = i
		case cm.description < 0 && isDescHeader(n):
			cm.description = i
		case cm.crdr < 0 && isCrDrHeader(n):
			cm.crdr = i
		case cm.debit < 0 && isDebitHeader(n):
			cm.debit = i
		case cm.credit < 0 && isCreditHeader(n):
			cm.credit = i
		case cm.amount < 0 && isAmountHeader(n):
			cm.amount = i
		case cm.balance < 0 && strings.Contains(n, "balance"):
			cm.balance = i
		}
	}

	// Fallback: first text-ish column that isn't date/amount as description.
	if cm.description < 0 {
		for i, h := range headers {
			n := normalizeHeader(h)
			if n == "" || i == cm.date || i == cm.debit || i == cm.credit || i == cm.amount || i == cm.crdr || i == cm.balance {
				continue
			}
			if strings.Contains(n, "chq") || strings.Contains(n, "ref") || strings.Contains(n, "cheque") ||
				strings.Contains(n, "type") || strings.Contains(n, "branch") || strings.Contains(n, "sno") ||
				n == "s no" || n == "sr no" || n == "no" {
				continue
			}
			cm.description = i
			break
		}
	}

	if cm.date < 0 {
		return cm, errMissingColumn("date")
	}
	if cm.description < 0 {
		return cm, errMissingColumn("description/narration")
	}
	if cm.debit < 0 && cm.credit < 0 && cm.amount < 0 {
		return cm, errMissingColumn("debit/credit or amount")
	}
	return cm, nil
}

func isDateHeader(n string) bool {
	if strings.Contains(n, "balance") {
		return false
	}
	return containsAny(n,
		"txn date", "tran date", "trans date", "transaction date",
		"value date", "value dt", "booking date", "posted date",
		"posting date", "entry date", "txn dt", "date",
	)
}

func isDescHeader(n string) bool {
	return containsAny(n,
		"narration", "description", "particulars", "remarks",
		"details", "transaction remarks", "txn remarks", "txn details",
		"transaction details", "transaction description", "info",
		"merchant", "payee",
	)
}

func isDebitHeader(n string) bool {
	if strings.Contains(n, "credit") || strings.Contains(n, "deposit") {
		return false
	}
	return containsAny(n,
		"debit", "withdrawal", "withdrawals", "withdraw",
		"dr amount", "amount dr", "debit amt", "withdrawal amt",
		"debit amount",
	) || n == "dr" || n == "wdl" || n == "withdrawal amt"
}

func isCreditHeader(n string) bool {
	if strings.Contains(n, "debit") || strings.Contains(n, "withdraw") {
		return false
	}
	return containsAny(n,
		"credit", "deposit", "deposits",
		"cr amount", "amount cr", "credit amt", "deposit amt",
		"credit amount",
	) || n == "cr"
}

func isAmountHeader(n string) bool {
	if strings.Contains(n, "balance") || strings.Contains(n, "debit") || strings.Contains(n, "credit") ||
		strings.Contains(n, "withdraw") || strings.Contains(n, "deposit") {
		return false
	}
	return n == "amount" ||
		strings.HasSuffix(n, " amount") ||
		n == "txn amount" ||
		n == "transaction amount" ||
		n == "transaction amount inr" ||
		n == "amount inr" ||
		n == "txn amount inr"
}

func isCrDrHeader(n string) bool {
	return n == "cr dr" || n == "dr cr" || n == "crdr" ||
		strings.Contains(n, "cr dr") || strings.Contains(n, "dr cr")
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if s == n || strings.Contains(s, n) {
			return true
		}
	}
	return false
}

type columnError string

func (e columnError) Error() string { return string(e) }

func errMissingColumn(name string) error {
	return columnError("missing required column: " + name)
}

func rowToTransaction(headers []string, row []string, cm columnMap) (*ExtractedTransaction, error) {
	get := func(idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	dateStr := get(cm.date)
	if dateStr == "" {
		return nil, nil
	}
	desc := get(cm.description)
	if desc == "" {
		return nil, nil
	}

	dt, err := ParseDate(dateStr)
	if err != nil {
		// Likely a non-data row under the header (titles, totals).
		return nil, nil
	}

	raw := map[string]string{}
	for i, h := range headers {
		if i < len(row) {
			raw[h] = row[i]
		}
	}

	var amountPaise int64
	var direction models.Direction

	debitStr := get(cm.debit)
	creditStr := get(cm.credit)
	amountStr := get(cm.amount)
	crdrStr := strings.ToLower(get(cm.crdr))

	switch {
	case cm.debit >= 0 || cm.credit >= 0:
		if isNonZeroAmount(debitStr) {
			p, err := money.ParseINR(debitStr)
			if err != nil {
				return nil, nil
			}
			amountPaise = money.Abs(p)
			direction = models.DirectionDebit
		} else if isNonZeroAmount(creditStr) {
			p, err := money.ParseINR(creditStr)
			if err != nil {
				return nil, nil
			}
			amountPaise = money.Abs(p)
			direction = models.DirectionCredit
		} else {
			return nil, nil
		}
	case cm.amount >= 0 && cm.crdr >= 0:
		if !isNonZeroAmount(amountStr) {
			return nil, nil
		}
		p, err := money.ParseINR(amountStr)
		if err != nil {
			return nil, nil
		}
		amountPaise = money.Abs(p)
		if strings.Contains(crdrStr, "cr") || strings.Contains(crdrStr, "credit") {
			direction = models.DirectionCredit
		} else {
			direction = models.DirectionDebit
		}
	default:
		p, err := money.ParseINR(amountStr)
		if err != nil {
			return nil, nil
		}
		if p < 0 {
			amountPaise = money.Abs(p)
			direction = models.DirectionDebit
		} else if p > 0 {
			amountPaise = p
			direction = models.DirectionCredit
		} else {
			return nil, nil
		}
	}

	if amountPaise == 0 {
		return nil, nil
	}

	return &ExtractedTransaction{
		Date:        dt,
		Description: desc,
		AmountPaise: amountPaise,
		Direction:   string(direction),
		Raw:         raw,
	}, nil
}

func isNonZeroAmount(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "—" || s == "0" || s == "0.0" || s == "0.00" {
		return false
	}
	return true
}
