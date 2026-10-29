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

// HealthSignals are deterministic month aggregates (safe to send to a local model).
type HealthSignals struct {
	IncomePaise         int64            `json:"income_paise"`
	ExpensePaise        int64            `json:"expense_paise"`
	UncategorizedPaise  int64            `json:"uncategorized_paise"`
	InvestmentPaise     int64            `json:"investment_paise"`
	TransactionCount    int64            `json:"transaction_count"`
	PendingCount        int64            `json:"pending_count"`
	NeedsReviewCount    int64            `json:"needs_review_count"`
	SavingsRate         float64          `json:"savings_rate"`          // (income-expense)/income; 0 if no income
	InvestShare         float64          `json:"invest_share"`          // investment / expense (0 if no expense)
	UncategorizedRatio  float64          `json:"uncategorized_ratio"`   // uncategorized / (income+expense+uncategorized amounts) simplified
	TopCategories       []CategoryShare  `json:"top_categories"`
}

type CategoryShare struct {
	Category    string  `json:"category"`
	AmountPaise int64   `json:"amount_paise"`
	Share       float64 `json:"share"`
}

type HealthScore struct {
	Score   int    `json:"score"`
	Verdict string `json:"verdict"`
}

// ScoreHealth computes a 0–100 score from signals (no LLM).
func ScoreHealth(s HealthSignals) HealthScore {
	score := 100.0

	// Uncategorized hurts trust in the numbers.
	switch {
	case s.UncategorizedRatio >= 0.4:
		score -= 35
	case s.UncategorizedRatio >= 0.2:
		score -= 20
	case s.UncategorizedRatio >= 0.1:
		score -= 10
	}

	// Savings rate (only meaningful with income).
	if s.IncomePaise > 0 {
		switch {
		case s.SavingsRate < 0:
			score -= 25
		case s.SavingsRate < 0.05:
			score -= 15
		case s.SavingsRate < 0.1:
			score -= 5
		case s.SavingsRate >= 0.2:
			score += 8
		}
	} else if s.ExpensePaise > 0 {
		score -= 10 // spend with no recognized income
	}

	// Investing is a positive signal when present.
	switch {
	case s.InvestShare >= 0.15 && s.InvestShare <= 0.5:
		score += 10
	case s.InvestShare > 0.5:
		score += 5
	case s.InvestShare > 0 && s.InvestShare < 0.05:
		score += 2
	}

	// Review backlog.
	if s.TransactionCount > 0 {
		reviewRatio := float64(s.NeedsReviewCount) / float64(s.TransactionCount)
		if reviewRatio >= 0.3 {
			score -= 12
		} else if reviewRatio >= 0.15 {
			score -= 6
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
	return HealthScore{Score: n, Verdict: verdict}
}

// HealthClient calls a heavier Ollama model once per month for a short tip.
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
	Tip string `json:"tip"`
}

func (c *HealthClient) GenerateTip(ctx context.Context, monthKey string, signals HealthSignals, score HealthScore) (string, error) {
	payload, _ := json.Marshal(map[string]any{
		"month":   monthKey,
		"score":   score.Score,
		"verdict": score.Verdict,
		"signals": signals,
	})
	prompt := fmt.Sprintf(`You are a concise India personal-finance coach. Given ONLY these monthly aggregates (JSON), write ONE tip of at most 2 short sentences.
Focus on spending vs income, investing, savings, or uncategorized risk. No bullet lists. No markdown. No disclaimers. Plain text only.

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
				"tip": map[string]any{"type": "string"},
			},
			"required": []string{"tip"},
		},
		Options: map[string]any{
			"temperature": 0.3,
			"num_predict": 120,
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama health request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama health status %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	var or ollamaResponse
	if err := json.Unmarshal(raw, &or); err != nil {
		return "", err
	}
	tip := strings.TrimSpace(or.Response)
	if i := strings.Index(tip, "{"); i >= 0 {
		if j := strings.LastIndex(tip, "}"); j > i {
			var ht healthTipResponse
			if err := json.Unmarshal([]byte(tip[i:j+1]), &ht); err == nil && strings.TrimSpace(ht.Tip) != "" {
				return clampTip(ht.Tip), nil
			}
		}
	}
	if tip == "" {
		return FallbackTip(signals, score), nil
	}
	return clampTip(tip), nil
}

func clampTip(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	if len(s) > 320 {
		s = s[:317] + "..."
	}
	return s
}

// FallbackTip is used when the health model is unavailable.
func FallbackTip(s HealthSignals, score HealthScore) string {
	switch score.Verdict {
	case "Strong":
		return "Solid month — keep the savings rate and investment habit steady."
	case "Stable":
		if s.UncategorizedRatio >= 0.15 {
			return "Numbers look okay, but clear Uncategorized items so the score stays trustworthy."
		}
		return "A balanced month. Watch top spend categories and protect your savings rate."
	case "Needs attention":
		return "Spending or uncategorized share is elevated — review big categories and pending items."
	default:
		return "This month needs cleanup: reduce overspend and categorize pending transactions first."
	}
}
