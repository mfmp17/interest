package main

import (
	"math/big"
	"testing"
	"time"
)

func TestClaimableMathWithFundingTranches(t *testing.T) {
	principal := big.NewInt(1_000_000_000) // 1000 USDC, 6 decimals
	created := time.Unix(1000, 0).UTC()
	lock := int64(100)

	classic := NewPosition("p1", "USDC", 6, principal, PlanClassic8, created, lock, 0, "0xabc")
	classic.AddFunding("0xtx1", "0xfrom", principal, 1, created)
	mid := created.Add(50 * time.Second)
	got := classic.Claimable(mid).String()
	want := "40000000" // 40 USDC halfway through an 80 USDC lock
	if got != want {
		t.Fatalf("classic halfway claimable=%s want %s", got, want)
	}
	end := created.Add(time.Duration(lock*2) * time.Second)
	got = classic.Claimable(end).String()
	want = "80000000" // capped at 8%
	if got != want {
		t.Fatalf("classic max claimable=%s want %s", got, want)
	}
}

func TestPartialAndOverpayFunding(t *testing.T) {
	expected := big.NewInt(10_000_000) // 10 USDC
	created := time.Unix(1000, 0).UTC()
	p := NewPosition("p", "USDC", 6, expected, PlanInstant5, created, 365, 0, "0xabc")

	one := big.NewInt(1_000_000)
	p.AddFunding("0xone", "0xfrom", one, 1, created)
	if p.Principal != "1000000" {
		t.Fatalf("principal=%s", p.Principal)
	}
	if p.MissingPrincipal().String() != "9000000" {
		t.Fatalf("missing=%s", p.MissingPrincipal())
	}
	if p.Claimable(created).String() != "50000" { // 5% of 1 USDC
		t.Fatalf("instant claimable=%s", p.Claimable(created))
	}

	twenty := big.NewInt(20_000_000)
	p.AddFunding("0xtwenty", "0xfrom", twenty, 2, created.Add(time.Second))
	if p.Principal != "21000000" {
		t.Fatalf("principal after overpay=%s", p.Principal)
	}
	if p.MissingPrincipal().Sign() != 0 {
		t.Fatalf("missing after overpay=%s", p.MissingPrincipal())
	}
	if p.ExtraPrincipal().String() != "11000000" {
		t.Fatalf("extra=%s", p.ExtraPrincipal())
	}
	if p.Claimable(created).String() != "1050000" { // 5% of 21 USDC
		t.Fatalf("instant total claimable=%s", p.Claimable(created))
	}
}
