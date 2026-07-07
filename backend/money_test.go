package main

import "testing"

func TestParseFormatUnits(t *testing.T) {
	cases := []struct {
		in       string
		decimals uint8
		want     string
		fmtWant  string
	}{
		{"1000", 6, "1000000000", "1000.00"},
		{"6.67", 6, "6670000", "6.67"},
		{"0.000001", 6, "1", "0.000001"},
		{"1.234567890123456789", 18, "1234567890123456789", "1.234567890123456789"},
	}
	for _, tc := range cases {
		u, err := ParseUnits(tc.in, tc.decimals)
		if err != nil {
			t.Fatalf("ParseUnits(%q): %v", tc.in, err)
		}
		if u.String() != tc.want {
			t.Fatalf("ParseUnits(%q)=%s want %s", tc.in, u.String(), tc.want)
		}
		if got := FormatUnits(u, tc.decimals); got != tc.fmtWant {
			t.Fatalf("FormatUnits(%s)=%q want %q", u.String(), got, tc.fmtWant)
		}
	}
}
