package main

import (
	"math/big"
	"testing"
	"time"
)

func TestPositionDTOWithDepositBalance(t *testing.T) {
	p := NewPosition("p", "USDC", 6, big.NewInt(10_000_000), PlanClassic8, time.Unix(1_000, 0), 365, 0, "0x1")
	dto := PositionDTOWithDepositBalance(p, big.NewInt(1_234_567))
	if dto["deposit_balance"] != "1234567" || dto["deposit_balance_display"] != "1.234567" {
		t.Fatalf("unexpected deposit balance fields: %#v", dto)
	}
}
