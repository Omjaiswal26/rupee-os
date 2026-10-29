package classification

import "testing"

func TestMatchMerchantKeywordGroww(t *testing.T) {
	p := MatchMerchantKeyword("WDL TFR UPI/DR/666/GROWW IN/HDFC/groww.b rk@/Paid")
	if p == nil || p.Category != "Investments" || p.Type != "investment" {
		t.Fatalf("%+v", p)
	}
}

func TestMatchMerchantKeywordSwiggy(t *testing.T) {
	p := MatchMerchantKeyword("WDL TFR UPI/DR/123/SWIGGY I/UTIB/swiggyinst/Pay")
	if p == nil || p.Category != "Food & Dining" {
		t.Fatalf("%+v", p)
	}
}

func TestMatchMerchantKeywordSafeGold(t *testing.T) {
	p := MatchMerchantKeyword("WDL TFR UPI/DR/111/SAFE GOLD/YESB/SAFEGOLD")
	if p == nil || p.Category != "Investments" || p.Subcategory != "Gold" {
		t.Fatalf("%+v", p)
	}
}

func TestMatchMerchantKeywordNoBarePhonePe(t *testing.T) {
	if p := MatchMerchantKeyword("UPI/DR/1/RAHUL SHARMA/ICIC/phonepe"); p != nil {
		t.Fatalf("expected nil for person+phonepe rail, got %+v", p)
	}
}

func TestMatchMerchantKeywordShortOlaBoundary(t *testing.T) {
	if p := MatchMerchantKeyword("UPI/DR/1/SOLACE/ICIC"); p != nil {
		t.Fatalf("ola must not match inside SOLACE: %+v", p)
	}
	p := MatchMerchantKeyword("UPI/DR/1/OLA/ICIC")
	if p == nil || p.Category != "Transport" {
		t.Fatalf("%+v", p)
	}
}
