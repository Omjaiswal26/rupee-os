package classification

import (
	"strings"
)

// MerchantKeyword is a curated India brand → category mapping (not user_rules).
type MerchantKeyword struct {
	// Needle is matched case-insensitively as a substring (spaces allowed, e.g. "safe gold").
	Needle      string
	Category    string
	Subcategory string
	Type        string
}

// MerchantKeywords covers high-signal UPI/bank narration brands.
// Order: longer / more specific needles first where overlap matters.
var MerchantKeywords = []MerchantKeyword{
	// Investments
	{Needle: "safe gold", Category: "Investments", Subcategory: "Gold", Type: "investment"},
	{Needle: "safegold", Category: "Investments", Subcategory: "Gold", Type: "investment"},
	{Needle: "groww", Category: "Investments", Subcategory: "Broker", Type: "investment"},
	{Needle: "zerodha", Category: "Investments", Subcategory: "Broker", Type: "investment"},
	{Needle: "upstox", Category: "Investments", Subcategory: "Broker", Type: "investment"},
	{Needle: "angelone", Category: "Investments", Subcategory: "Broker", Type: "investment"},
	{Needle: "angel one", Category: "Investments", Subcategory: "Broker", Type: "investment"},
	{Needle: "kuvera", Category: "Investments", Subcategory: "Mutual Fund", Type: "investment"},
	{Needle: "coin by zerodha", Category: "Investments", Subcategory: "Mutual Fund", Type: "investment"},
	{Needle: "cdsl", Category: "Investments", Subcategory: "Demat", Type: "investment"},
	{Needle: "mutual f", Category: "Investments", Subcategory: "Mutual Fund", Type: "investment"},

	// Food
	{Needle: "swiggy", Category: "Food & Dining", Subcategory: "Delivery", Type: "expense"},
	{Needle: "zomato", Category: "Food & Dining", Subcategory: "Delivery", Type: "expense"},
	{Needle: "eatsure", Category: "Food & Dining", Subcategory: "Delivery", Type: "expense"},
	{Needle: "dominos", Category: "Food & Dining", Subcategory: "Restaurant", Type: "expense"},
	{Needle: "domino's", Category: "Food & Dining", Subcategory: "Restaurant", Type: "expense"},

	// Groceries
	{Needle: "bigbasket", Category: "Groceries", Subcategory: "Supermarket", Type: "expense"},
	{Needle: "blinkit", Category: "Groceries", Subcategory: "Quick commerce", Type: "expense"},
	{Needle: "zepto", Category: "Groceries", Subcategory: "Quick commerce", Type: "expense"},
	{Needle: "dunzo", Category: "Groceries", Subcategory: "Delivery", Type: "expense"},
	{Needle: "jiomart", Category: "Groceries", Subcategory: "Supermarket", Type: "expense"},

	// Shopping
	{Needle: "amazon", Category: "Shopping", Subcategory: "Online", Type: "expense"},
	{Needle: "flipkart", Category: "Shopping", Subcategory: "Online", Type: "expense"},
	{Needle: "myntra", Category: "Shopping", Subcategory: "Fashion", Type: "expense"},
	{Needle: "ajio", Category: "Shopping", Subcategory: "Fashion", Type: "expense"},
	{Needle: "meesho", Category: "Shopping", Subcategory: "Online", Type: "expense"},
	{Needle: "nykaa", Category: "Shopping", Subcategory: "Beauty", Type: "expense"},
	{Needle: "toy corner", Category: "Shopping", Subcategory: "Toys", Type: "expense"},

	// Transport
	{Needle: "uber", Category: "Transport", Subcategory: "Ride", Type: "expense"},
	{Needle: "ola", Category: "Transport", Subcategory: "Ride", Type: "expense"},
	{Needle: "rapido", Category: "Transport", Subcategory: "Ride", Type: "expense"},
	{Needle: "irctc", Category: "Transport", Subcategory: "Rail", Type: "expense"},
	{Needle: "redbus", Category: "Transport", Subcategory: "Bus", Type: "expense"},

	// Bills
	{Needle: "airtel", Category: "Bills & Utilities", Subcategory: "Telecom", Type: "expense"},
	{Needle: "jio", Category: "Bills & Utilities", Subcategory: "Telecom", Type: "expense"},
	{Needle: "act fibernet", Category: "Bills & Utilities", Subcategory: "Internet", Type: "expense"},
	{Needle: "bescom", Category: "Bills & Utilities", Subcategory: "Electricity", Type: "expense"},

	// Entertainment
	{Needle: "netflix", Category: "Entertainment", Subcategory: "Streaming", Type: "expense"},
	{Needle: "spotify", Category: "Entertainment", Subcategory: "Streaming", Type: "expense"},
	{Needle: "hotstar", Category: "Entertainment", Subcategory: "Streaming", Type: "expense"},
	{Needle: "youtube", Category: "Entertainment", Subcategory: "Streaming", Type: "expense"},
}

// MatchMerchantKeyword returns a prediction if description contains a curated brand needle.
func MatchMerchantKeyword(description string) *Prediction {
	lower := strings.ToLower(description)
	for _, m := range MerchantKeywords {
		if m.Needle == "" || !strings.Contains(lower, m.Needle) {
			continue
		}
		// Avoid false positives: short "ola" / "jio" / "vi " need word-ish boundaries.
		if !keywordBoundaryOK(lower, m.Needle) {
			continue
		}
		return &Prediction{
			Category:    m.Category,
			Subcategory: m.Subcategory,
			Type:        m.Type,
			Confidence:  0.95,
			Reasoning:   "merchant keyword: " + m.Needle,
		}
	}
	return nil
}

// LooksLikeKnownMerchant reports whether description matches any curated brand needle.
func LooksLikeKnownMerchant(s string) bool {
	return MatchMerchantKeyword(s) != nil
}

func keywordBoundaryOK(lower, needle string) bool {
	// Longer needles are safe as substring.
	if len(needle) >= 4 {
		return true
	}
	idx := strings.Index(lower, needle)
	if idx < 0 {
		return false
	}
	leftOK := idx == 0 || !isASCIILetterOrDigit(lower[idx-1])
	right := idx + len(needle)
	rightOK := right >= len(lower) || !isASCIILetterOrDigit(lower[right])
	return leftOK && rightOK
}

func isASCIILetterOrDigit(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}
