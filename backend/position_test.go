package main

import (
	"math/big"
	"testing"
	"time"
)

func TestClaimableMath(t *testing.T) {
	principal := big.NewInt(1_000_000_000) // 1000 USDC, 6 decimals
	created := time.Unix(1000, 0).UTC()
	lock := int64(100)

	classic := NewPosition("p1", "USDC", 6, principal, PlanClassic8, created, lock, 0, "0xabc")
	mid := created.Add(50 * time.Second)
	got := classic.Claimable(mid).String()
	want := "40000000" // 40 USDC halfway through an 80 USDC year/lock
	if got != want {
		t.Fatalf("classic halfway claimable=%s want %s", got, want)
	}
	end := created.Add(time.Duration(lock*2) * time.Second)
	got = classic.Claimable(end).String()
	want = "80000000" // capped at 8%
	if got != want {
		t.Fatalf("classic max claimable=%s want %s", got, want)
	}

	instant := NewPosition("p2", "USDC", 6, principal, PlanInstant5, created, lock, 1, "0xdef")
	got = instant.Claimable(created).String()
	want = "50000000" // 50 USDC immediately
	if got != want {
		t.Fatalf("instant claimable=%s want %s", got, want)
	}
}
