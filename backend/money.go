package main

import (
	"fmt"
	"math/big"
	"strings"
)

func ParseUnits(s string, decimals uint8) (*big.Int, error) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "$"))
	if s == "" {
		return nil, fmt.Errorf("empty amount")
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid amount %q", s)
	}
	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if whole == "" {
		whole = "0"
	}
	if len(frac) > int(decimals) {
		return nil, fmt.Errorf("too many decimal places for %d-decimal token", decimals)
	}
	for _, r := range whole + frac {
		if r < '0' || r > '9' {
			return nil, fmt.Errorf("invalid amount %q", s)
		}
	}
	frac += strings.Repeat("0", int(decimals)-len(frac))
	out := new(big.Int)
	if _, ok := out.SetString(whole+frac, 10); !ok {
		return nil, fmt.Errorf("invalid amount %q", s)
	}
	return out, nil
}

func FormatUnits(v *big.Int, decimals uint8) string {
	if v == nil {
		return "0.00"
	}
	s := v.String()
	if decimals == 0 {
		return s
	}
	if len(s) <= int(decimals) {
		s = strings.Repeat("0", int(decimals)-len(s)+1) + s
	}
	cut := len(s) - int(decimals)
	whole, frac := s[:cut], s[cut:]
	frac = strings.TrimRight(frac, "0")
	if frac == "" {
		frac = "00"
	}
	if len(frac) == 1 {
		frac += "0"
	}
	return whole + "." + frac
}
