package classification

import "testing"

func TestScoreHealthStrong(t *testing.T) {
	s := ScoreHealth(HealthSignals{
		IncomePaise:        10_000_000,
		ExpensePaise:       6_000_000,
		InvestmentPaise:    1_500_000,
		UncategorizedPaise: 100_000,
		TransactionCount:   100,
		NeedsReviewCount:   5,
		SavingsRate:        0.4,
		InvestShare:        0.25,
		UncategorizedRatio: 0.05,
	})
	if s.Score < 80 || s.Verdict != "Strong" {
		t.Fatalf("%+v", s)
	}
}

func TestScoreHealthCriticalUncategorized(t *testing.T) {
	s := ScoreHealth(HealthSignals{
		IncomePaise:        1_000_000,
		ExpensePaise:       2_000_000,
		UncategorizedPaise: 5_000_000,
		TransactionCount:   50,
		NeedsReviewCount:   20,
		SavingsRate:        -1,
		UncategorizedRatio: 0.5,
	})
	if s.Score >= 40 {
		t.Fatalf("expected low score, got %+v", s)
	}
}

func TestEnrichHealthUnreliableUncategorized(t *testing.T) {
	sig := HealthSignals{
		IncomePaise:        1_000_000,
		ExpensePaise:       800_000,
		UncategorizedPaise: 400_000,
		TransactionCount:   40,
		NeedsReviewCount:   5,
		SavingsRate:        0.2,
		InvestShare:        0.1,
		UncategorizedRatio: 0.25,
		TopCategories: []CategoryShare{
			{Category: "Food & Dining", AmountPaise: 300_000, Share: 0.3},
		},
	}
	scored := EnrichHealth(&sig)
	if sig.Reliable {
		t.Fatal("expected unreliable")
	}
	if sig.UnreliableReason == "" {
		t.Fatal("expected reason")
	}
	if len(sig.Insights) < 3 {
		t.Fatalf("insights: %+v", sig.Insights)
	}
	if len(sig.Drivers) == 0 {
		t.Fatal("expected drivers")
	}
	if sig.Headline != "" {
		t.Fatalf("unreliable should not set headline, got %q", sig.Headline)
	}
	if scored.Score < 0 || scored.Score > 100 {
		t.Fatalf("score %+v", scored)
	}
}
