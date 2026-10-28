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

func TestNormalizePredictionExpenseNotTransfers(t *testing.T) {
	p := &Prediction{Category: "Transfers", Type: "expense", Confidence: 0.9, Reasoning: "shop"}
	NormalizePrediction("debit", p)
	if p.Category == "Transfers" || p.Type != "expense" {
		t.Fatalf("expected spend remap, got %+v", p)
	}
}

func TestNormalizePredictionAlignsInvestmentType(t *testing.T) {
	p := &Prediction{Category: "Investments", Type: "transfer", Confidence: 0.9}
	NormalizePrediction("debit", p)
	if p.Type != "investment" || p.Category != "Investments" {
		t.Fatalf("%+v", p)
	}
}

func TestNormalizePredictionFoodTransferTypeFixed(t *testing.T) {
	p := &Prediction{Category: "Food & Dining", Type: "transfer", Confidence: 0.9}
	NormalizePrediction("debit", p)
	if err := ValidatePrediction(p); err != nil {
		t.Fatal(err)
	}
	if p.Type != "expense" {
		t.Fatalf("%+v", p)
	}
}

func TestValidateTransferTypeMismatch(t *testing.T) {
	err := ValidatePrediction(&Prediction{Category: "Food & Dining", Type: "transfer", Confidence: 0.9})
	if err == nil {
		t.Fatal("expected error")
	}
}
