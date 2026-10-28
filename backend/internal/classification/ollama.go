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
	NormalizePrediction(direction, pred)
	if err := ValidatePrediction(pred); err != nil {
		return nil, err
	}
	return pred, nil
}

func buildPrompt(description string, amountPaise int64, direction string) string {
	cats := strings.Join(AllowedCategories, ", ")
	types := strings.Join(AllowedTypes, ", ")
	return fmt.Sprintf(`Classify this Indian bank transaction. Reply with one JSON object only. No markdown, no thinking, no extra text.
Keys: category, subcategory, type, confidence, reasoning.
category ∈ [%s]
type ∈ [%s]
confidence: 0.0-1.0. reasoning: ≤12 words. If unsure, low confidence + category Other.

Hard rules:
- Ignore rail/noise tokens: WDL, TFR, UPI, IMPS, NEFT, RTGS, DR, CR, bank codes (HDFC/YESB/UTIB/…), account/branch/ref numbers, AT #### addresses.
- Classify by the merchant / purpose name in the narration (often between slashes), not by "UPI".
- UPI/IMPS is a payment rail — it does NOT mean category Transfers.
- category Transfers + type transfer ONLY for true P2P/self/family with no merchant brand (e.g. "to self", person name only, own account transfer).
- Merchant spend (shop, food, gold, apps) → expense/investment categories, NEVER Transfers.
- Digi-gold / broker / mutual fund cues → Investments + type investment.
- Food stalls / Swiggy / Zomato → Food & Dining + type expense.
- Retail / toys / Amazon / Flipkart → Shopping + type expense.
- Never pair a spend category with type transfer. Never pair Transfers with type expense.

Examples:
desc: "WDL TFR UPI/DR/123/SWIGGY I/UTIB/swiggyinst/Pay" → {"category":"Food & Dining","subcategory":"Delivery","type":"expense","confidence":0.92,"reasoning":"Swiggy merchant UPI"}
desc: "WDL TFR UPI/DR/456/TOY CORNER/HDFC/vyapar" → {"category":"Shopping","subcategory":"Toys","type":"expense","confidence":0.85,"reasoning":"Toy shop merchant"}
desc: "WDL TFR UPI/DR/789/MOM HAND/YESB/paytm" → {"category":"Food & Dining","subcategory":"Stall","type":"expense","confidence":0.8,"reasoning":"Food stall name"}
desc: "WDL TFR UPI/DR/111/SAFE GOLD/YESB/SAFEGOLD" → {"category":"Investments","subcategory":"Gold","type":"investment","confidence":0.95,"reasoning":"Digital gold purchase"}
desc: "UPI/DR/222/RAHUL SHARMA/ICIC" → {"category":"Transfers","subcategory":"P2P","type":"transfer","confidence":0.75,"reasoning":"Person name P2P"}

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

// NormalizePrediction aligns type↔category and repairs inconsistent Transfer/spend combos.
func NormalizePrediction(direction string, p *Prediction) {
	if p == nil {
		return
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
			p.Category = "Income"
		}
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
	default:
		if p.Type == "transfer" || p.Type == "" {
			if direction == "credit" {
				p.Type = "income"
			} else {
				p.Type = "expense"
			}
		}
	}
}

var (
	railTokenRe = regexp.MustCompile(`(?i)^(WDL|TFR|UPI|IMPS|NEFT|RTGS|DR|CR|INFT|NWD|ATW|POS|PUR|GOL|PAY)$`)
	bankCodeRe  = regexp.MustCompile(`(?i)^(HDFC|YESB|UTIB|ICIC|SBIN|AXIS|PAYTM|OKBIZ|P2M|P2A|AIRP)$`)
	digitsHeavy = regexp.MustCompile(`^\d{4,}$`)
)

// HasMerchantSignal reports a non-rail name token in a UPI/IMPS/POS narration.
func HasMerchantSignal(description string) bool {
	upper := strings.ToUpper(description)
	if !strings.Contains(upper, "UPI") && !strings.Contains(upper, "IMPS") && !strings.Contains(upper, "POS") {
		return false
	}
	parts := strings.FieldsFunc(description, func(r rune) bool {
		return r == '/' || r == '\\' || r == '|' || unicode.IsSpace(r)
	})
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if len(p) < 3 {
			continue
		}
		up := strings.ToUpper(p)
		if railTokenRe.MatchString(up) || bankCodeRe.MatchString(up) || digitsHeavy.MatchString(up) {
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
		if letters >= 3 {
			return true
		}
	}
	return false
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
