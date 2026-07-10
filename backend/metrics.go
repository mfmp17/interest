package main

import (
	"fmt"
	"math/big"
	"time"
)

type LiabilitySnapshot struct {
	Principal *big.Int
	Claimable *big.Int
	Total     *big.Int
}

type StatusAccounting struct {
	TVL                string
	PrincipalLiability string
	ClaimableLiability string
	TotalLiability     string
	ReserveRatio       string
	Underfunded        bool
	Shortfall          string
}

func BuildStatusAccounting(positions []*Position, now time.Time, treasury *big.Int, decimals uint8) StatusAccounting {
	s := CalculateLiabilities(positions, now)
	shortfall := big.NewInt(0)
	underfunded := false
	if treasury != nil && treasury.Cmp(s.Total) < 0 {
		shortfall.Sub(s.Total, treasury)
		underfunded = true
	}
	return StatusAccounting{
		TVL:                "$" + FormatUnits(s.Principal, decimals),
		PrincipalLiability: FormatUnits(s.Principal, decimals),
		ClaimableLiability: FormatUnits(s.Claimable, decimals),
		TotalLiability:     FormatUnits(s.Total, decimals),
		ReserveRatio:       ReserveRatioDisplay(treasury, s.Total),
		Underfunded:        underfunded,
		Shortfall:          FormatUnits(shortfall, decimals),
	}
}

func CalculateLiabilities(positions []*Position, now time.Time) LiabilitySnapshot {
	principal := big.NewInt(0)
	claimable := big.NewInt(0)
	for _, p := range positions {
		if p == nil || p.Status == StatusClosed || p.PrincipalPaid {
			continue
		}
		principal.Add(principal, p.principalInt())
		claimable.Add(claimable, p.Claimable(now))
	}
	return LiabilitySnapshot{
		Principal: principal,
		Claimable: claimable,
		Total:     new(big.Int).Add(new(big.Int).Set(principal), claimable),
	}
}

func ReserveRatioDisplay(treasury, liabilities *big.Int) string {
	if treasury == nil || liabilities == nil || liabilities.Sign() == 0 {
		return "n/a"
	}
	hundredths := new(big.Int).Mul(treasury, big.NewInt(10_000))
	hundredths.Div(hundredths, liabilities)
	whole := new(big.Int).Div(new(big.Int).Set(hundredths), big.NewInt(100))
	fraction := new(big.Int).Mod(hundredths, big.NewInt(100))
	return fmt.Sprintf("%s.%02d%%", whole.String(), fraction.Int64())
}
