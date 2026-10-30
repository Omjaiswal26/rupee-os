package classification

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	railTokenRe = regexp.MustCompile(`(?i)^(WDL|TFR|DEP|UPI|IMPS|NEFT|RTGS|DR|CR|INFT|NWD|ATW|POS|PUR|GOL|PAY|PAYM|PAYMENT|SENT|NA|TO|FROM|BY)$`)
	bankCodeRe  = regexp.MustCompile(`(?i)^(HDFC|YESB|UTIB|ICIC|SBIN|AXIS|PAYTM|OKBIZ|P2M|P2A|AIRP|PUNB|KKBK|CNRB|IDFB|RATN|FDRL|MAHB|IOBA|BKID|BARB|UBIN|INDB|YESB)$`)
	// IFSC-like or bank+digits refs: HDFC0000240, UTIB0000052
	ifscLikeRe = regexp.MustCompile(`(?i)^[A-Z]{4}0[A-Z0-9]{4,}$`)
	// Long opaque refs: HDFCH01354473826, AXNH262900833411
	opaqueRefRe = regexp.MustCompile(`(?i)^[A-Z]{2,}\d{6,}$|^[A-Z0-9]{12,}$`)
	digitsHeavy = regexp.MustCompile(`^\d{3,}$`)
	atBranchRe  = regexp.MustCompile(`(?i)^AT\d*$`)
)

// ExtractPayeeCandidate returns the most likely payee/merchant from a bank narration.
func ExtractPayeeCandidate(description string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return ""
	}

	// NEFT often encodes the counterparty after the last '*'.
	if neft := extractNEFTPayee(desc); neft != "" {
		return neft
	}

	parts := splitNarration(desc)
	var nameParts []string
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		// Collapse spaced single letters later; normalize zero→O for brand matching later.
		up := strings.ToUpper(p)
		if skipNarrationToken(up) {
			if len(nameParts) > 0 {
				// Bank / ref / digits after a name → stop (don't glue UPI handle on).
				break
			}
			continue
		}
		if isAddressTail(up) {
			break
		}
		letters := countLetters(up)
		if letters == 0 {
			if len(nameParts) > 0 {
				break
			}
			continue
		}
		nameParts = append(nameParts, cleanToken(p))
		if len(nameParts) >= 3 {
			break
		}
	}
	return strings.TrimSpace(strings.Join(nameParts, " "))
}

func extractNEFTPayee(desc string) string {
	upper := strings.ToUpper(desc)
	if !strings.Contains(upper, "NEFT") || !strings.Contains(desc, "*") {
		return ""
	}
	segs := strings.Split(desc, "*")
	if len(segs) < 2 {
		return ""
	}
	// Last star segment usually holds the name + trailing account/AT noise.
	last := strings.TrimSpace(segs[len(segs)-1])
	if last == "" {
		return ""
	}
	parts := splitNarration(last)
	var nameParts []string
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		up := strings.ToUpper(p)
		if skipNarrationToken(up) || isAddressTail(up) || digitsHeavy.MatchString(up) {
			if len(nameParts) > 0 {
				break
			}
			continue
		}
		if countLetters(up) == 0 {
			if len(nameParts) > 0 {
				break
			}
			continue
		}
		nameParts = append(nameParts, cleanToken(p))
		if len(nameParts) >= 4 {
			break
		}
	}
	return strings.TrimSpace(strings.Join(nameParts, " "))
}

func splitNarration(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '/' || r == '\\' || r == '|' || r == '*' || unicode.IsSpace(r)
	})
}

func skipNarrationToken(up string) bool {
	if railTokenRe.MatchString(up) || bankCodeRe.MatchString(up) || digitsHeavy.MatchString(up) {
		return true
	}
	if ifscLikeRe.MatchString(up) || opaqueRefRe.MatchString(up) {
		return true
	}
	if atBranchRe.MatchString(up) {
		return true
	}
	return false
}

func isAddressTail(up string) bool {
	switch up {
	case "NEW", "RAJINDER", "NAGAR", "DELHI", "MUMBAI", "BANGALORE", "BENGALURU":
		return true
	}
	return false
}

func countLetters(s string) int {
	n := 0
	for _, r := range s {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			n++
		}
	}
	return n
}

func cleanToken(p string) string {
	p = strings.Trim(p, ".,;:")
	// OCR/bank quirks: GR0WW → GROWW for display/matching.
	if strings.ContainsAny(p, "0") && countLetters(strings.ToUpper(p)) >= 3 {
		repl := strings.Map(func(r rune) rune {
			if r == '0' {
				return 'O'
			}
			return r
		}, p)
		if LooksLikeKnownMerchant(repl) || LooksLikeKnownMerchant(strings.ToLower(repl)) {
			return repl
		}
		// Also try if lower contains groww after replace
		if strings.Contains(strings.ToLower(repl), "groww") {
			return repl
		}
	}
	return p
}
