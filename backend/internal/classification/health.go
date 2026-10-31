package classification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrHealthModelMissing means the configured health Ollama model is not pulled.
var ErrHealthModelMissing = errors.New("health model not pulled")

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
	Reliable         bool            `json:"reliable"`
	UnreliableReason string          `json:"unreliable_reason,omitempty"`
	Insights         []HealthInsight `json:"insights,omitempty"`
	Drivers          []HealthDriver  `json:"drivers,omitempty"`
	Headline         string          `json:"headline,omitempty"`
	FocusCategory    string          `json:"focus_category,omitempty"`
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

// HealthClient calls a heavier Ollama model for the full month/year health report.
type HealthClient struct {
	BaseURL string
	Model   string
	HTTP    *http.Client
}

func NewHealthClient(baseURL, model string) *HealthClient {
	return &HealthClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
		HTTP:    &http.Client{Timeout: 180 * time.Second},
	}
}

// ModelHealthReport is the structured output we require from the health model.
type ModelHealthReport struct {
	Score            int             `json:"score"`
	Verdict          string          `json:"verdict"`
	Headline         string          `json:"headline"`
	FocusCategory    string          `json:"focus_category"`
	Reliable         bool            `json:"reliable"`
	UnreliableReason string          `json:"unreliable_reason"`
	Insights         []HealthInsight `json:"insights"`
	Drivers          []HealthDriver  `json:"drivers"`
}

// EnsureHealthModelPulled fails if the health weights are missing (no deterministic fallback).
func EnsureHealthModelPulled(ctx context.Context, baseURL, classifyModel, healthModel string) error {
	if strings.TrimSpace(healthModel) == "" {
		return fmt.Errorf("%w: OLLAMA_HEALTH_MODEL is empty — set it and run ./scripts/pull-ollama-model.sh", ErrHealthModelMissing)
	}
	st := CheckOllamaModels(ctx, baseURL, classifyModel, healthModel)
	if !st.Reachable {
		return fmt.Errorf("%w: ollama unreachable (%s) — start the ollama service, then run ./scripts/pull-ollama-model.sh", ErrHealthModelMissing, st.Error)
	}
	if !st.HealthPresent {
		return fmt.Errorf("%w: %q not found — run ./scripts/pull-ollama-model.sh", ErrHealthModelMissing, healthModel)
	}
	return nil
}

// GenerateHealthReport asks the model for score, verdict, headline, insights, and drivers.
// Aggregates are inputs only — there is no deterministic score/tip fallback.
func (c *HealthClient) GenerateHealthReport(ctx context.Context, period string, signals HealthSignals) (*ModelHealthReport, error) {
	if c == nil || c.Model == "" {
		return nil, fmt.Errorf("%w: health client not configured", ErrHealthModelMissing)
	}

	payloadSignals := map[string]any{
		"income_paise":        signals.IncomePaise,
		"expense_paise":       signals.ExpensePaise,
		"uncategorized_paise": signals.UncategorizedPaise,
		"investment_paise":    signals.InvestmentPaise,
		"transaction_count":   signals.TransactionCount,
		"pending_count":       signals.PendingCount,
		"needs_review_count":  signals.NeedsReviewCount,
		"savings_rate":        signals.SavingsRate,
		"invest_share":        signals.InvestShare,
		"uncategorized_ratio": signals.UncategorizedRatio,
		"top_categories":      signals.TopCategories,
	}
	payload, _ := json.Marshal(map[string]any{
		"period":  period,
		"signals": payloadSignals,
	})

	prompt := fmt.Sprintf(`You are an India personal-finance analyst. Using ONLY the period aggregates JSON, return one JSON object:
- score: integer 0-100
- verdict: exactly one of Strong | Stable | Needs attention | Critical
- headline: max 12 words, no markdown, not a paragraph
- focus_category: one name from top_categories, or "Other" if uncategorized is the issue, or ""
- reliable: false if uncategorized_ratio >= 0.2 OR needs_review share is very high; else true
- unreliable_reason: short reason when reliable is false, else ""
- insights: 3-4 objects {id, label, value, tone} with tone in good|ok|warn|bad. Prefer Uncategorized, Savings, Invested, and top spend category. Values like "28%%" using the given ratios — do not invent figures.
- drivers: 2-5 objects {label, delta} where delta is integer score contribution (negative hurts). Labels short.

Do not invent merchants or amounts not implied by the signals.

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
				"score":             map[string]any{"type": "integer"},
				"verdict":           map[string]any{"type": "string"},
				"headline":          map[string]any{"type": "string"},
				"focus_category":    map[string]any{"type": "string"},
				"reliable":          map[string]any{"type": "boolean"},
				"unreliable_reason": map[string]any{"type": "string"},
				"insights":          map[string]any{"type": "array"},
				"drivers":           map[string]any{"type": "array"},
			},
			"required": []string{"score", "verdict", "headline", "reliable", "insights", "drivers"},
		},
		Options: map[string]any{
			"temperature": 0.2,
			"num_predict": 400,
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama health request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		msg := truncate(string(raw), 200)
		if strings.Contains(strings.ToLower(msg), "not found") {
			return nil, fmt.Errorf("%w: %s", ErrHealthModelMissing, msg)
		}
		return nil, fmt.Errorf("ollama health status %d: %s", resp.StatusCode, msg)
	}

	var or ollamaResponse
	if err := json.Unmarshal(raw, &or); err != nil {
		return nil, err
	}
	tip := strings.TrimSpace(or.Response)
	var parsed ModelHealthReport
	if i := strings.Index(tip, "{"); i >= 0 {
		if j := strings.LastIndex(tip, "}"); j > i {
			if err := json.Unmarshal([]byte(tip[i:j+1]), &parsed); err != nil {
				return nil, fmt.Errorf("decode health model JSON: %w", err)
			}
		}
	}
	if parsed.Verdict == "" && parsed.Headline == "" {
		return nil, fmt.Errorf("health model returned empty report")
	}
	return normalizeModelHealth(&parsed, signals), nil
}

func normalizeModelHealth(m *ModelHealthReport, signals HealthSignals) *ModelHealthReport {
	if m.Score < 0 {
		m.Score = 0
	}
	if m.Score > 100 {
		m.Score = 100
	}
	switch m.Verdict {
	case "Strong", "Stable", "Needs attention", "Critical":
	default:
		switch {
		case m.Score >= 80:
			m.Verdict = "Strong"
		case m.Score >= 60:
			m.Verdict = "Stable"
		case m.Score >= 40:
			m.Verdict = "Needs attention"
		default:
			m.Verdict = "Critical"
		}
	}
	m.Headline = clampHeadline(m.Headline)
	m.FocusCategory = sanitizeFocus(m.FocusCategory, signals)

	// Hard data-quality gate — model may understate uncategorized risk.
	if signals.UncategorizedRatio >= UncategorizedReliableMax {
		m.Reliable = false
		if strings.TrimSpace(m.UnreliableReason) == "" {
			m.UnreliableReason = fmt.Sprintf(
				"%.0f%% of money is Uncategorized — categorize, then regenerate.",
				signals.UncategorizedRatio*100,
			)
		}
	}

	if len(m.Insights) > 4 {
		m.Insights = m.Insights[:4]
	}
	for i := range m.Insights {
		switch m.Insights[i].Tone {
		case "good", "ok", "warn", "bad":
		default:
			m.Insights[i].Tone = "ok"
		}
		m.Insights[i].Label = strings.TrimSpace(m.Insights[i].Label)
		m.Insights[i].Value = strings.TrimSpace(m.Insights[i].Value)
		if m.Insights[i].ID == "" {
			m.Insights[i].ID = fmt.Sprintf("insight_%d", i)
		}
	}
	if len(m.Drivers) > 6 {
		m.Drivers = m.Drivers[:6]
	}
	return m
}

// ApplyModelReport copies model output onto signals for storage/UI.
func ApplyModelReport(signals *HealthSignals, m *ModelHealthReport) HealthScore {
	signals.Reliable = m.Reliable
	signals.UnreliableReason = strings.TrimSpace(m.UnreliableReason)
	signals.Headline = m.Headline
	if !m.Reliable {
		// UI uses reliability callout; avoid competing essay.
		signals.Headline = ""
	}
	signals.FocusCategory = m.FocusCategory
	signals.Insights = m.Insights
	signals.Drivers = m.Drivers
	return HealthScore{Score: m.Score, Verdict: m.Verdict}
}

func sanitizeFocus(focus string, s HealthSignals) string {
	focus = strings.TrimSpace(focus)
	if focus == "" {
		return ""
	}
	for _, c := range s.TopCategories {
		if strings.EqualFold(c.Category, focus) {
			return c.Category
		}
	}
	if strings.EqualFold(focus, "Uncategorized") || strings.EqualFold(focus, "Other") {
		return "Other"
	}
	return ""
}

func clampHeadline(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
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

// FallbackTip kept for tests / legacy helpers.
func FallbackTip(s HealthSignals, score HealthScore) string {
	return FallbackHeadline(s, score)
}
