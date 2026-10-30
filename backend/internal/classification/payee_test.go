package classification

import "testing"

func TestExtractPayeeSkipsDEP(t *testing.T) {
	cases := []struct {
		desc string
		want string
	}{
		{
			"DEP TFR UPI/CR/111007773482/TUSHAR C/PUNB/tushar .544/na 0097732162091 AT 01278 NEW RAJINDER NAGAR",
			"TUSHAR C",
		},
		{
			"DEP TFR UPI/CR/330326720294/PANKAJ J/SBIN/9926781 219/Paym 0097735162098 AT 01278 NEW RAJINDER NAGAR",
			"PANKAJ J",
		},
		{
			"DEP TFR NEFT*HDFC0000240*HDFCH01354473826*PPFAS M UTUAL FU 0099509044300 AT 01278 NEW RAJINDER NAGAR",
			"PPFAS M UTUAL FU",
		},
		{
			"DEP TFR NEFT*UTIB0000052*AXNH262900833411*GR0WW I NVEST TE 0099509044300 AT 01278 NEW RAJINDER NAGAR",
			"GROWW I NVEST TE",
		},
		{
			"WDL TFR UPI/DR/131/Shubhi J/AIRP/8827809/Paym",
			"Shubhi J",
		},
	}
	for _, tc := range cases {
		got := ExtractPayeeCandidate(tc.desc)
		if got != tc.want {
			t.Fatalf("%q\ngot  %q\nwant %q", tc.desc, got, tc.want)
		}
	}
}

func TestMatchMerchantGrowwWithZero(t *testing.T) {
	p := MatchMerchantKeyword("DEP TFR NEFT*UTIB*AXNH*GR0WW I NVEST TE 0099 AT 01278")
	if p == nil || p.Category != "Investments" {
		t.Fatalf("%+v", p)
	}
}

func TestMatchMerchantPPFAS(t *testing.T) {
	p := MatchMerchantKeyword("DEP TFR NEFT*HDFC*REF*PPFAS M UTUAL FU 0099 AT 01278")
	if p == nil || p.Category != "Investments" {
		t.Fatalf("%+v", p)
	}
}
