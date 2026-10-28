package classification

import "testing"

func TestParsePredictionOK(t *testing.T) {
	raw := `{"category":"Food & Dining","subcategory":"Restaurant","type":"expense","confidence":0.91,"reasoning":"Zomato order"}`
	p, err := ParsePrediction(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Category != "Food & Dining" || p.Confidence != 0.91 {
		t.Fatalf("%+v", p)
	}
}

func TestParsePredictionWrapped(t *testing.T) {
	raw := "Here you go:\n{\"category\":\"Groceries\",\"subcategory\":\"Supermarket\",\"type\":\"expense\",\"confidence\":0.8,\"reasoning\":\"BigBasket\"}\n"
	if _, err := ParsePrediction(raw); err != nil {
		t.Fatal(err)
	}
}

func TestParsePredictionInvalidCategory(t *testing.T) {
	raw := `{"category":"Crypto","subcategory":"x","type":"expense","confidence":0.9,"reasoning":"n"}`
	if _, err := ParsePrediction(raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestParsePredictionBadConfidence(t *testing.T) {
	raw := `{"category":"Other","subcategory":"","type":"expense","confidence":1.5,"reasoning":"n"}`
	if _, err := ParsePrediction(raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestHasMerchantSignal(t *testing.T) {
	cases := []struct {
		desc string
		want bool
	}{
		{"WDL TFR UPI/DR/630190/TOY CORNER/HDFC/vyapar", true},
		{"WDL TFR UPI/DR/627521/SAFE GOLD/YESB/SAFEGOLD", true},
		{"UPI/DR/222/RAHUL SHARMA/ICIC", true},
		{"NEFT CR FROM SALARY ACCOUNT", false},
		{"ATM WDL", false},
	}
	for _, tc := range cases {
		if got := HasMerchantSignal(tc.desc); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.desc, got, tc.want)
		}
	}
}

func TestLooksLikePersonPayee(t *testing.T) {
	cases := []struct {
		desc string
		want bool
	}{
		{"UPI/DR/333/Shubhi J/AIRP/8827809/Paym", true},
		{"UPI/DR/222/RAHUL SHARMA/ICIC", true},
		{"WDL TFR UPI/DR/123/SWIGGY I/UTIB/swiggyinst/Pay", false},
		{"WDL TFR UPI/DR/111/SAFE GOLD/YESB/SAFEGOLD", false},
		{"NEFT CR SALARY ACME PVT LTD", false},
		{"ATM WDL", false},
	}
	for _, tc := range cases {
		if got := LooksLikePersonPayee(tc.desc); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.desc, got, tc.want)
		}
	}
}

func TestNormalizePredictionPersonForcedOther(t *testing.T) {
	p := &Prediction{Category: "Transport", Type: "expense", Confidence: 0.9, Reasoning: "cab"}
	NormalizePrediction("UPI/DR/333/Shubhi J/AIRP/8827809/Paym", "debit", 1000000, p)
	if p.Category != "Other" || p.Type != "expense" || p.Confidence > 0.45 {
		t.Fatalf("%+v", p)
	}
	if err := ValidatePrediction(p); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizePredictionLargeCreditSalary(t *testing.T) {
	p := &Prediction{Category: "Other", Type: "income", Confidence: 0.4}
	NormalizePrediction("NEFT CR CLOSERX TECHNOLOGIES PVT LTD", "credit", 12_000_000, p)
	if p.Category != "Income" || p.Type != "income" || p.Confidence < 0.8 {
		t.Fatalf("%+v", p)
	}
}

func TestNormalizePredictionPersonCreditNotSalary(t *testing.T) {
	p := &Prediction{Category: "Income", Type: "income", Confidence: 0.9}
	NormalizePrediction("UPI/CR/444/Shubhi J/AIRP/Paym", "credit", 12_000_000, p)
	if p.Category != "Other" {
		t.Fatalf("person credit must stay uncategorized: %+v", p)
	}
}

func TestNormalizePredictionExpenseNotTransfers(t *testing.T) {
	p := &Prediction{Category: "Transfers", Type: "expense", Confidence: 0.9, Reasoning: "shop"}
	NormalizePrediction("WDL TFR UPI/DR/456/TOY CORNER/HDFC/vyapar", "debit", 50000, p)
	if p.Category == "Transfers" || p.Type != "expense" {
		t.Fatalf("expected spend remap, got %+v", p)
	}
}

func TestNormalizePredictionAlignsInvestmentType(t *testing.T) {
	p := &Prediction{Category: "Investments", Type: "transfer", Confidence: 0.9}
	NormalizePrediction("SAFE GOLD", "debit", 100000, p)
	if p.Type != "investment" || p.Category != "Investments" {
		t.Fatalf("%+v", p)
	}
}

func TestNormalizePredictionFoodTransferTypeFixed(t *testing.T) {
	p := &Prediction{Category: "Food & Dining", Type: "transfer", Confidence: 0.9}
	NormalizePrediction("SWIGGY", "debit", 50000, p)
	if err := ValidatePrediction(p); err != nil {
		t.Fatal(err)
	}
	if p.Type != "expense" {
		t.Fatalf("%+v", p)
	}
}

func TestRulePatternFromDescription(t *testing.T) {
	pat := RulePatternFromDescription("WDL TFR UPI/DR/131/Shubhi J/AIRP/8827809/Paym")
	if pat != "Shubhi J" {
		t.Fatalf("got %q", pat)
	}
}

func TestValidateTransferTypeMismatch(t *testing.T) {
	err := ValidatePrediction(&Prediction{Category: "Food & Dining", Type: "transfer", Confidence: 0.9})
	if err == nil {
		t.Fatal("expected error")
	}
}
