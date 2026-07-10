package main

import (
	"math/big"
	"testing"
	"time"
)

func TestCalculateLiabilitiesSumsOutstandingPrincipalAndClaimable(t *testing.T) {
	now := time.Unix(2_000, 0).UTC()

	classic := NewPosition("classic", "USDC", 6, big.NewInt(10_000_000), PlanClassic8, now.Add(-365*time.Second), 365, 0, "0x1")
	classic.AddFunding("0xclassic", "0xfrom", big.NewInt(10_000_000), 1, now.Add(-365*time.Second))

	instant := NewPosition("instant", "USDC", 6, big.NewInt(20_000_000), PlanInstant5, now.Add(-10*time.Second), 365, 1, "0x2")
	instant.AddFunding("0xinstant", "0xfrom", big.NewInt(20_000_000), 2, now.Add(-10*time.Second))

	closed := NewPosition("closed", "USDC", 6, big.NewInt(30_000_000), PlanClassic8, now.Add(-365*time.Second), 365, 2, "0x3")
	closed.AddFunding("0xclosed", "0xfrom", big.NewInt(30_000_000), 3, now.Add(-365*time.Second))
	closed.PrincipalPaid = true
	closed.Status = StatusClosed

	s := CalculateLiabilities([]*Position{classic, instant, closed}, now)
	if s.Principal.Cmp(big.NewInt(30_000_000)) != 0 {
		t.Fatalf("principal = %s, want 30000000", s.Principal)
	}
	if s.Claimable.Cmp(big.NewInt(1_800_000)) != 0 {
		t.Fatalf("claimable = %s, want 1800000", s.Claimable)
	}
	if s.Total.Cmp(big.NewInt(31_800_000)) != 0 {
		t.Fatalf("total = %s, want 31800000", s.Total)
	}
}

func TestReserveRatioDisplay(t *testing.T) {
	if got := ReserveRatioDisplay(big.NewInt(55_724_870), big.NewInt(31_800_000)); got != "175.23%" {
		t.Fatalf("ratio = %q, want 175.23%%", got)
	}
	if got := ReserveRatioDisplay(big.NewInt(1), big.NewInt(0)); got != "n/a" {
		t.Fatalf("zero liabilities ratio = %q, want n/a", got)
	}
}

func TestBuildStatusAccountingFormatsUSDCFields(t *testing.T) {
	now := time.Unix(2_000, 0).UTC()
	p := NewPosition("p", "USDC", 6, big.NewInt(10_000_000), PlanClassic8, now.Add(-365*time.Second), 365, 0, "0x1")
	p.AddFunding("0xtx", "0xfrom", big.NewInt(10_000_000), 1, now.Add(-365*time.Second))

	got := BuildStatusAccounting([]*Position{p}, now, big.NewInt(55_724_870), 6)
	if got.TVL != "$10.00" || got.PrincipalLiability != "10.00" || got.ClaimableLiability != "0.80" || got.TotalLiability != "10.80" || got.ReserveRatio != "515.97%" || got.Underfunded || got.Shortfall != "0.00" {
		t.Fatalf("unexpected accounting: %+v", got)
	}

	under := BuildStatusAccounting([]*Position{p}, now, big.NewInt(5_000_000), 6)
	if !under.Underfunded || under.Shortfall != "5.80" {
		t.Fatalf("unexpected underfunded accounting: %+v", under)
	}
}
