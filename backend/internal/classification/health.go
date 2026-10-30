package classification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// UncategorizedReliableMax — at/above this ratio the report is marked unreliable.
const UncategorizedReliableMax = 0.20

// HealthSignals are deterministic month aggregates (safe to send to a local model).
type HealthSignals struct {
	IncomePaise        int64           `json:"income_paise"`
	ExpensePaise       int64           `json:"expense_paise"`
	UncategorizedPaise int64           `json:"uncategorized_paise"`
	InvestmentPaise    int64           `json:"investment_paise"`
	TransactionCount   int64           `json:"transaction_count"`
	PendingCount       int64           `json:"pending_count"`
	NeedsReviewCount   int64           `json:"needs_review_count"`
	SavingsRate        float64         `json:"savings_rate"`
	InvestShare        float64         `json:"invest_share"`
	UncategorizedRatio float64         `json:"uncategorized_ratio"`
	TopCategories      []CategoryShare `json:"top_categories"`

	// Derived (stored with signals so the UI stays self-contained).
	Reliable         bool           `json:"reliable"`
	UnreliableReason string         `json:"unreliable_reason,omitempty"`
	Insights         []HealthInsight `json:"insights,omitempty"`
	Drivers          []HealthDriver  `json:"drivers,omitempty"`
	Headline         string         `json:"headline,omitempty"`
	FocusCategory    string         `json:"focus_category,omitempty"`
}

type CategoryShare struct {
	Category    string  `json:"category"`
	AmountPaise int64   `json:"amount_paise"`
	Share       float64 `json:"share"`
}

type HealthInsight struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
	Tone  string `json:"tone"` // good | ok | warn | bad
}

type HealthDriver struct {
	Label string `json:"label"`
	Delta int    `json:"delta"`
}

type HealthScore struct {
	Score   int    `json:"score"`
	Verdict string `json:"verdict"`
}

// ScoreHealth computes a 0–100 score from signals (no LLM).
func ScoreHealth(s HealthSignals) HealthScore {
	score, _ := scoreHealthWithDrivers(s)
	return score
}

func scoreHealthWithDrivers(s HealthSignals) (HealthScore, []HealthDriver) {
	score := 100.0
	var drivers []HealthDriver

	add := func(label string, delta float64) {
		if delta == 0 {
			return
		}
		score += delta
		drivers = append(drivers, HealthDriver{Label: label, Delta: int(delta)})
	}

	switch {
	case s.UncategorizedRatio >= 0.4:
		add("Uncategorized", -35)
	case s.UncategorizedRatio >= 0.2:
		add("Uncategorized", -20)
	case s.UncategorizedRatio >= 0.1:
		add("Uncategorized", -10)
	}

	if s.IncomePaise > 0 {
		switch {
		case s.SavingsRate < 0:
			add("Savings", -25)
		case s.SavingsRate < 0.05:
			add("Savings", -15)
		case s.SavingsRate < 0.1:
			add("Savings", -5)
		case s.SavingsRate >= 0.2:
			add("Savings", 8)
		}
	} else if s.ExpensePaise > 0 {
		add("No income", -10)
	}

	switch {
	case s.InvestShare >= 0.15 && s.InvestShare <= 0.5:
		add("Investing", 10)
	case s.InvestShare > 0.5:
		add("Investing", 5)
	case s.InvestShare > 0 && s.InvestShare < 0.05:
		add("Investing", 2)
	}

	if s.TransactionCount > 0 {
		reviewRatio := float64(s.NeedsReviewCount) / float64(s.TransactionCount)
		if reviewRatio >= 0.3 {
			add("Needs review", -12)
		} else if reviewRatio >= 0.15 {
			add("Needs review", -6)
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	n := int(score + 0.5)

	verdict := "Stable"
	switch {
	case n >= 80:
		verdict = "Strong"
	case n >= 60:
		verdict = "Stable"
	case n >= 40:
		verdict = "Needs attention"
	default:
		verdict = "Critical"
	}
	return HealthScore{Score: n, Verdict: verdict}, drivers
}

// EnrichHealth fills reliability, insights, drivers, and a short headline on signals.
func EnrichHealth(s *HealthSignals) HealthScore {
	scored, drivers := scoreHealthWithDrivers(*s)
	s.Drivers = drivers
	s.Reliable = true
	s.UnreliableReason = ""

	if s.UncategorizedRatio >= UncategorizedReliableMax {
		s.Reliable = false
		s.UnreliableReason = fmt.Sprintf(
			"%.0f%% of money is Uncategorized — this report may not be reliable. Categorize those transactions, then regenerate.",
			s.UncategorizedRatio*100,
		)
	} else if s.TransactionCount > 0 {
		reviewRatio := float64(s.NeedsReviewCount) / float64(s.TransactionCount)
		if reviewRatio >= 0.4 {
			s.Reliable = false
			s.UnreliableReason = fmt.Sprintf(
				"%d of %d transactions still need review — finish Review, then regenerate for a trustworthy score.",
				s.NeedsReviewCount, s.TransactionCount,
			)
		}
	}

	s.Insights = buildInsights(*s)
	s.FocusCategory = pickFocusCategory(*s)
	s.Headline = FallbackHeadline(*s, scored)
	return scored
}

func buildInsights(s HealthSignals) []HealthInsight {
	out := make([]HealthInsight, 0, 4)

	// Data quality
	uncatTone := "good"
	uncatVal := fmt.Sprintf("%.0f%%", s.UncategorizedRatio*100)
	switch {
	case s.UncategorizedRatio >= 0.2:
		uncatTone = "bad"
	case s.UncategorizedRatio >= 0.1:
		uncatTone = "warn"
	}
	out = append(out, HealthInsight{ID: "uncategorized", Label: "Uncategorized", Value: uncatVal, Tone: uncatTone})

	// Savings
	savTone := "ok"
	savVal := "—"
	if s.IncomePaise >= 10_000 {
		savVal = fmt.Sprintf("%.0f%%", s.SavingsRate*100)
		switch {
		case s.SavingsRate >= 0.2:
			savTone = "good"
		case s.SavingsRate >= 0.1:
			savTone = "ok"
		case s.SavingsRate >= 0:
			savTone = "warn"
		default:
			savTone = "bad"
		}
	} else {
		savTone = "warn"
		savVal = "n/a"
	}
	out = append(out, HealthInsight{ID: "savings", Label: "Savings", Value: savVal, Tone: savTone})

	// Investing
	invTone := "ok"
	invVal := fmt.Sprintf("%.0f%%", s.InvestShare*100)
	switch {
	case s.InvestShare >= 0.15:
		invTone = "good"
	case s.InvestShare > 0:
		invTone = "warn"
	default:
		invTone = "ok"
		invVal = "0%"
	}
	out = append(out, HealthInsight{ID: "invest", Label: "Invested", Value: invVal, Tone: invTone})

	// Top spend category (skip Other / Transfers when possible)
	for _, c := range s.TopCategories {
		if c.Category == "" || c.Category == "Other" || c.Category == "Transfers" || c.Category == "Income" {
			continue
		}
		tone := "ok"
		if c.Share >= 0.35 {
			tone = "warn"
		}
		if c.Share >= 0.5 {
			tone = "bad"
		}
		out = append(out, HealthInsight{
			ID:    "top_category",
			Label: c.Category,
			Value: fmt.Sprintf("%.0f%%", c.Share*100),
			Tone:  tone,
		})
		break
	}

	return out
}

func pickFocusCategory(s HealthSignals) string {
	if s.UncategorizedRatio >= UncategorizedReliableMax {
		return "Other"
	}
	for _, c := range s.TopCategories {
		if c.Category == "" || c.Category == "Other" || c.Category == "Transfers" || c.Category == "Income" || c.Category == "Investments" {
			continue
		}
		if c.Share >= 0.2 {
			return c.Category
		}
	}
	if s.NeedsReviewCount > 0 {
		return ""
	}
	return ""
}

// FallbackHeadline is a short non-paragraph summary.
func FallbackHeadline(s HealthSignals, score HealthScore) string {
	if !s.Reliable {
		// UI shows the reliability callout; keep tip empty-ish / unused when unreliable.
		return ""
	}
	if s.FocusCategory != "" && s.FocusCategory != "Other" {
		for _, c := range s.TopCategories {
			if c.Category == s.FocusCategory {
				return fmt.Sprintf("%s is %.0f%% of activity", c.Category, c.Share*100)
			}
		}
	}
	switch score.Verdict {
	case "Strong":
		return "Solid month — keep the rhythm"
	case "Stable":
		return "Balanced month — protect savings"
	case "Needs attention":
		return "Watch spend and data quality"
	default:
		return "Cleanup needed this month"
	}
}

// HealthClient calls a heavier Ollama model once per month for a short headline.
type HealthClient struct {
	BaseURL string
	Model   string
	HTTP    *http.Client
}

func NewHealthClient(baseURL, model string) *HealthClient {
	return &HealthClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
		HTTP:    &http.Client{Timeout: 120 * time.Second},
	}
}

type healthTipResponse struct {
	Headline      string `json:"headline"`
	Tip           string `json:"tip"` // legacy
	FocusCategory string `json:"focus_category"`
}

func (c *HealthClient) GenerateTip(ctx context.Context, period string, signals HealthSignals, score HealthScore) (string, error) {
	headline, _, err := c.GenerateHeadline(ctx, period, signals, score)
	return headline, err
}

// GenerateHeadline asks the model for a short headline (+ optional focus category).
func (c *HealthClient) GenerateHeadline(ctx context.Context, period string, signals HealthSignals, score HealthScore) (headline string, focus string, err error) {
	// Strip derived fields from the model payload to reduce confusion.
	payloadSignals := signals
	payloadSignals.Insights = nil
	payloadSignals.Drivers = nil
	payloadSignals.Headline = ""
	payloadSignals.UnreliableReason = ""

	payload, _ := json.Marshal(map[string]any{
		"period":   period,
		"score":    score.Score,
		"verdict":  score.Verdict,
		"reliable": signals.Reliable,
		"signals":  payloadSignals,
		"focus_hint": signals.FocusCategory,
	})
	prompt := fmt.Sprintf(`You are a concise India personal-finance coach. Given ONLY these period aggregates (JSON), return JSON with:
- "headline": max 10 words, no markdown, no period at end unless needed. Not a paragraph.
- "focus_category": one category from top_categories, or "" if none.

If reliable is false, headline must urge categorizing / review first.
Do not invent numbers or merchants.

Data:
%s
`, string(payload))

	body, _ := json.Marshal(ollamaRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
		Format: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"headline":       map[string]any{"type": "string"},
				"focus_category": map[string]any{"type": "string"},
			},
			"required": []string{"headline"},
		},
		Options: map[string]any{
			"temperature": 0.25,
			"num_predict": 80,
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("ollama health request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("ollama health status %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var or ollamaResponse
	if err := json.Unmarshal(raw, &or); err != nil {
		return "", "", err
	}
	tip := strings.TrimSpace(or.Response)
	if i := strings.Index(tip, "{"); i >= 0 {
		if j := strings.LastIndex(tip, "}"); j > i {
			var ht healthTipResponse
			if err := json.Unmarshal([]byte(tip[i:j+1]), &ht); err == nil {
				h := strings.TrimSpace(ht.Headline)
				if h == "" {
					h = strings.TrimSpace(ht.Tip)
				}
				if h != "" {
					return clampHeadline(h), sanitizeFocus(ht.FocusCategory, signals), nil
				}
			}
		}
	}
	return FallbackHeadline(signals, score), signals.FocusCategory, nil
}

func sanitizeFocus(focus string, s HealthSignals) string {
	focus = strings.TrimSpace(focus)
	if focus == "" {
		return s.FocusCategory
	}
	for _, c := range s.TopCategories {
		if strings.EqualFold(c.Category, focus) {
			return c.Category
		}
	}
	if strings.EqualFold(focus, "Uncategorized") || strings.EqualFold(focus, "Other") {
		return "Other"
	}
	return s.FocusCategory
}

func clampHeadline(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	// Collapse to one line.
	s = strings.ReplaceAll(s, "\n", " ")
	words := strings.Fields(s)
	if len(words) > 12 {
		words = words[:12]
	}
	s = strings.Join(words, " ")
	if len(s) > 90 {
		s = s[:87] + "..."
	}
	return s
}

// FallbackTip kept for older call sites / year health.
func FallbackTip(s HealthSignals, score HealthScore) string {
	return FallbackHeadline(s, score)
}
