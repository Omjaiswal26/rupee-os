package money

import "testing"

func TestParseINR(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"1,234.56", 123456},
		{"₹500", 50000},
		{"-100.50", -10050},
		{"(250.00)", -25000},
		{"0.01", 1},
	}
	for _, c := range cases {
		got, err := ParseINR(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("%q: got %d want %d", c.in, got, c.want)
		}
	}
}
