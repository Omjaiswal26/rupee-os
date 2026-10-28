package classification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type Client struct {
	BaseURL       string
	Model         string
	MinConfidence float64
	HTTP          *http.Client
}

type Prediction struct {
	Category    string  `json:"category"`
	Subcategory string  `json:"subcategory"`
	Type        string  `json:"type"`
	Confidence  float64 `json:"confidence"`
	Reasoning   string  `json:"reasoning"`
}

type ollamaRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	Format  any            `json:"format,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func NewClient(baseURL, model string, minConfidence float64) *Client {
	return &Client{
		BaseURL:       strings.TrimRight(baseURL, "/"),
		Model:         model,
		MinConfidence: minConfidence,
		HTTP:          &http.Client{Timeout: 60 * time.Second},
	}
}

func responseSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"category":    map[string]any{"type": "string", "enum": AllowedCategories},
			"subcategory": map[string]any{"type": "string"},
			"type":        map[string]any{"type": "string", "enum": AllowedTypes},
			"confidence":  map[string]any{"type": "number"},
			"reasoning":   map[string]any{"type": "string"},
		},
		"required": []string{"category", "type", "confidence"},
	}
}

func (c *Client) Classify(ctx context.Context, description string, amountPaise int64, direction string) (*Prediction, error) {
	prompt := buildPrompt(description, amountPaise, direction)
	body, _ := json.Marshal(ollamaRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
		Format: responseSchema(),
		Options: map[string]any{
			"temperature":    0,
			"num_predict":    128,
			"top_p":          0.9,
			"repeat_penalty": 1.1,
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var or ollamaResponse
	if err := json.Unmarshal(raw, &or); err != nil {
		return nil, fmt.Errorf("decode ollama envelope: %w", err)
	}
	pred, err := ParsePrediction(or.Response)
	if err != nil {
		return nil, fmt.Errorf("%w (raw=%s)", err, truncate(or.Response, 200))
	}
	NormalizePrediction(description, direction, amountPaise, pred)
	if err := ValidatePrediction(pred); err != nil {
		return nil, err
	}
	return pred, nil
}

// LikelySalaryMinPaise is ₹50,000 — large employer-style credits are usually salary.
const LikelySalaryMinPaise int64 = 50_000_00

func buildPrompt(description string, amountPaise int64, direction string) string {
	cats := strings.Join(AllowedCategories, ", ")
	types := strings.Join(AllowedTypes, ", ")
	return fmt.Sprintf(`Classify this Indian bank transaction. Reply with one JSON object only. No markdown, no thinking, no extra text.
Keys: category, subcategory, type, confidence, reasoning.
category ∈ [%s]
type ∈ [%s]
confidence: 0.0-1.0. reasoning: ≤12 words.

Hard rules:
- Ignore rail/noise: WDL, TFR, UPI, IMPS, NEFT, RTGS, DR, CR, bank codes (HDFC/YESB/UTIB/ICIC/AIRP/…), account/ref numbers, AT #### addresses.
- Classify by merchant/purpose name between slashes — never by the word UPI alone.
- Credit ≥ ₹50,000 (amount_paise ≥ 5000000) from a company/employer/payroll-like payer → Income + type income (salary). Amount is a strong hint when the payer is not a person name.
- Person names / initials (e.g. "Shubhi J", "Rahul Sharma") → category Other + low confidence (needs review). NEVER Transport/Shopping/Food/Income for a bare person name.
- True self/family wallet moves with no ambiguity → Transfers + type transfer.
- Known merchants (Swiggy, shops, apps) → spend category + type expense.
- Digi-gold / broker / mutual fund → Investments + type investment.
- If unsure → category Other + confidence ≤0.45.
- Never pair a spend category with type transfer. Never pair Transfers with type expense/income.

Examples:
desc: "NEFT CR SALARY ACME PVT LTD" amount_paise 8500000 direction credit → {"category":"Income","subcategory":"Salary","type":"income","confidence":0.95,"reasoning":"Large salary credit"}
desc: "NEFT CR CLOSERX TECHNOLOGIES" amount_paise 12000000 direction credit → {"category":"Income","subcategory":"Salary","type":"income","confidence":0.9,"reasoning":"Large employer credit"}
desc: "WDL TFR UPI/DR/123/SWIGGY I/UTIB/swiggyinst/Pay" direction debit → {"category":"Food & Dining","subcategory":"Delivery","type":"expense","confidence":0.92,"reasoning":"Swiggy merchant"}
desc: "WDL TFR UPI/DR/456/TOY CORNER/HDFC/vyapar" direction debit → {"category":"Shopping","subcategory":"Toys","type":"expense","confidence":0.85,"reasoning":"Toy shop"}
desc: "WDL TFR UPI/DR/111/SAFE GOLD/YESB/SAFEGOLD" direction debit → {"category":"Investments","subcategory":"Gold","type":"investment","confidence":0.95,"reasoning":"Digital gold"}
desc: "UPI/DR/222/RAHUL SHARMA/ICIC" direction debit → {"category":"Other","subcategory":"","type":"expense","confidence":0.4,"reasoning":"Person name; review"}
desc: "UPI/DR/333/Shubhi J/AIRP/8827809/Paym" direction debit → {"category":"Other","subcategory":"","type":"expense","confidence":0.4,"reasoning":"Person initials; review"}
desc: "UPI/CR/444/Shubhi J/AIRP/Paym" direction credit → {"category":"Other","subcategory":"","type":"income","confidence":0.4,"reasoning":"Person credit; review"}

Transaction:
description: %q
amount_paise: %d
direction: %s
`, cats, types, description, amountPaise, direction)
}

func ParsePrediction(raw string) (*Prediction, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty model response")
	}
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var p Prediction
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, fmt.Errorf("invalid classification JSON: %w", err)
	}
	if err := validateAllowed(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

// NormalizePrediction aligns type↔category, blocks person-name spend hallucinations,
// and promotes large non-person credits toward salary/income.
func NormalizePrediction(description, direction string, amountPaise int64, p *Prediction) {
	if p == nil {
		return
	}

	personLike := LooksLikePersonPayee(description) && !LooksLikeKnownMerchant(description)

	// Person-like UPI payee → Other + low confidence (uncategorized / needs review).
	if personLike {
		p.Category = "Other"
		p.Subcategory = ""
		if direction == "credit" {
			p.Type = "income"
		} else {
			p.Type = "expense"
		}
		if p.Confidence > 0.45 || p.Confidence <= 0 {
			p.Confidence = 0.45
		}
		p.Reasoning = "person-like payee; needs review"
	}

	// Large credit from a non-person payer → assume salary/income.
	if !personLike && direction == "credit" && amountPaise >= LikelySalaryMinPaise {
		if p.Category != "Investments" && p.Category != "Transfers" && p.Category != "Fees & Charges" {
			p.Category = "Income"
			p.Type = "income"
			if p.Subcategory == "" {
				p.Subcategory = "Salary"
			}
			if p.Confidence < 0.8 {
				p.Confidence = 0.8
			}
			if strings.TrimSpace(p.Reasoning) == "" || strings.Contains(strings.ToLower(p.Reasoning), "review") {
				p.Reasoning = "large credit; likely salary"
			}
		}
	}

	// Model said Transfers but typed it as spend/invest → trust the type, drop Transfers.
	if p.Category == "Transfers" {
		switch p.Type {
		case "investment":
			p.Category = "Investments"
		case "fee":
			p.Category = "Fees & Charges"
		case "expense":
			p.Category = "Other"
			if p.Confidence > 0.45 {
				p.Confidence = 0.45
			}
			if strings.TrimSpace(p.Reasoning) == "" {
				p.Reasoning = "spend labeled transfer; needs review"
			}
		case "income":
			if personLike {
				p.Category = "Other"
				p.Type = "income"
			} else {
				p.Category = "Income"
			}
		}
	}

	// Person-like credits must not stay as Income.
	if personLike && p.Category == "Income" {
		p.Category = "Other"
		p.Type = "income"
		if p.Confidence > 0.45 {
			p.Confidence = 0.45
		}
		p.Reasoning = "person-like payee; needs review"
	}

	// Align canonical type from category.
	switch p.Category {
	case "Transfers":
		p.Type = "transfer"
	case "Investments":
		p.Type = "investment"
	case "Fees & Charges":
		p.Type = "fee"
	case "Income":
		p.Type = "income"
	case "Other":
		if p.Type == "transfer" || p.Type == "" {
			if direction == "credit" {
				p.Type = "income"
			} else {
				p.Type = "expense"
			}
		}
	default:
		if p.Type == "transfer" || p.Type == "" {
			if direction == "credit" {
				p.Type = "income"
				p.Category = "Other"
				if p.Confidence > 0.45 {
					p.Confidence = 0.45
				}
			} else {
				p.Type = "expense"
			}
		}
	}
}

// RulePatternFromDescription picks a stable contains-pattern (payee token preferred).
func RulePatternFromDescription(desc string) string {
	if payee := ExtractPayeeCandidate(desc); payee != "" {
		return payee
	}
	words := strings.Fields(strings.TrimSpace(desc))
	if len(words) == 0 {
		return ""
	}
	n := len(words)
	if n > 3 {
		n = 3
	}
	return strings.Join(words[:n], " ")
}

var (
	railTokenRe = regexp.MustCompile(`(?i)^(WDL|TFR|UPI|IMPS|NEFT|RTGS|DR|CR|INFT|NWD|ATW|POS|PUR|GOL|PAY|PAYM|PAYMENT)$`)
	bankCodeRe  = regexp.MustCompile(`(?i)^(HDFC|YESB|UTIB|ICIC|SBIN|AXIS|PAYTM|OKBIZ|P2M|P2A|AIRP)$`)
	digitsHeavy = regexp.MustCompile(`^\d{3,}$`)
	knownMerchantRe = regexp.MustCompile(`(?i)\b(SWIGGY|ZOMATO|AMAZON|FLIPKART|GROWW|ZERODHA|SAFE\s*GOLD|SAFEGOLD|TOY\s*CORNER|UBER|OLA|IRCTC|BIGBASKET|BLINKIT|MYNTRA|AJIO|NETFLIX|SPOTIFY|GOOGLE|APPLE|CRED|PHONEPE|BHARATPE)\b`)
	salaryCueRe = regexp.MustCompile(`(?i)\b(SALARY|PAYROLL|WAGES|DIVIDEND|INTEREST|NEFT\s*CR.*LTD|PVT\s*LTD|PRIVATE\s*LIMITED)\b`)
)

// ExtractPayeeCandidate returns the most likely payee token from a UPI-like narration.
func ExtractPayeeCandidate(description string) string {
	parts := strings.FieldsFunc(description, func(r rune) bool {
		return r == '/' || r == '\\' || r == '|' || unicode.IsSpace(r)
	})
	var nameParts []string
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if len(p) < 1 {
			continue
		}
		up := strings.ToUpper(p)
		if railTokenRe.MatchString(up) || bankCodeRe.MatchString(up) || digitsHeavy.MatchString(up) {
			// Flush incomplete name if we hit rail after collecting.
			if len(nameParts) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(up, "AT") && len(up) <= 6 {
			continue
		}
		letters := 0
		for _, r := range up {
			if r >= 'A' && r <= 'Z' {
				letters++
			}
		}
		if letters == 0 {
			if len(nameParts) > 0 {
				break
			}
			continue
		}
		nameParts = append(nameParts, p)
		// Typical person/merchant: 1–3 tokens then bank code / digits.
		if len(nameParts) >= 3 {
			break
		}
	}
	if len(nameParts) == 0 {
		return ""
	}
	return strings.Join(nameParts, " ")
}

// LooksLikePersonPayee detects person-style UPI names (e.g. "Shubhi J", "Rahul Sharma").
func LooksLikePersonPayee(description string) bool {
	if salaryCueRe.MatchString(description) {
		return false
	}
	upper := strings.ToUpper(description)
	if !strings.Contains(upper, "UPI") && !strings.Contains(upper, "IMPS") {
		return false
	}
	payee := ExtractPayeeCandidate(description)
	if payee == "" {
		return false
	}
	if LooksLikeKnownMerchant(payee) || LooksLikeKnownMerchant(description) {
		return false
	}

	tokens := strings.Fields(payee)
	if len(tokens) == 0 || len(tokens) > 4 {
		return false
	}
	// Reject if any token looks commercial (has digits, @, long brand blob).
	for _, tok := range tokens {
		if strings.ContainsAny(tok, "0123456789@._") {
			return false
		}
		if len(tok) > 18 {
			return false
		}
	}

	// "Shubhi J" / "Rahul Sharma" / "MOM HAND" — letters + optional single-letter initial.
	letterTokens := 0
	initials := 0
	for _, tok := range tokens {
		runes := []rune(tok)
		allLetters := true
		for _, r := range runes {
			if !unicode.IsLetter(r) {
				allLetters = false
				break
			}
		}
		if !allLetters {
			return false
		}
		if len(runes) == 1 {
			initials++
		} else {
			letterTokens++
		}
	}
	if letterTokens == 0 {
		return false
	}
	// At least one word of length >= 2 and overall looks like a name (1–3 words).
	return letterTokens+initials <= 4
}

func LooksLikeKnownMerchant(s string) bool {
	return knownMerchantRe.MatchString(s)
}

// HasMerchantSignal reports a non-rail name token in a UPI/IMPS/POS narration.
func HasMerchantSignal(description string) bool {
	upper := strings.ToUpper(description)
	if !strings.Contains(upper, "UPI") && !strings.Contains(upper, "IMPS") && !strings.Contains(upper, "POS") {
		return false
	}
	return ExtractPayeeCandidate(description) != ""
}

func validateAllowed(p *Prediction) error {
	if p.Category == "" || !IsAllowedCategory(p.Category) {
		return fmt.Errorf("invalid category %q", p.Category)
	}
	if p.Type == "" || !IsAllowedType(p.Type) {
		return fmt.Errorf("invalid type %q", p.Type)
	}
	if p.Confidence < 0 || p.Confidence > 1 {
		return fmt.Errorf("confidence out of range: %v", p.Confidence)
	}
	return nil
}

func ValidatePrediction(p *Prediction) error {
	if err := validateAllowed(p); err != nil {
		return err
	}
	if p.Category == "Transfers" && p.Type != "transfer" {
		return fmt.Errorf("Transfers requires type transfer, got %q", p.Type)
	}
	if p.Type == "transfer" && p.Category != "Transfers" {
		return fmt.Errorf("type transfer requires category Transfers, got %q", p.Category)
	}
	return nil
}

func (c *Client) NeedsReview(p *Prediction) bool {
	return p.Confidence < c.MinConfidence || p.Category == "Other"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
