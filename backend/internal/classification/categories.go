package classification

// AllowedCategories for Indian personal finance V1.
var AllowedCategories = []string{
	"Income",
	"Food & Dining",
	"Groceries",
	"Transport",
	"Shopping",
	"Bills & Utilities",
	"Rent & Housing",
	"Healthcare",
	"Entertainment",
	"Travel",
	"Education",
	"Investments",
	"Transfers",
	"Fees & Charges",
	"Personal Care",
	"Insurance",
	"Taxes",
	"Other",
}

var AllowedTypes = []string{
	"expense",
	"income",
	"transfer",
	"investment",
	"fee",
}

func IsAllowedCategory(c string) bool {
	for _, a := range AllowedCategories {
		if a == c {
			return true
		}
	}
	return false
}

func IsAllowedType(t string) bool {
	for _, a := range AllowedTypes {
		if a == t {
			return true
		}
	}
	return false
}
