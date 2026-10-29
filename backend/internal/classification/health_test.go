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
