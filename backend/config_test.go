package main

import "testing"

func TestDefaultConfigIsBaseMainnetUSDCOnly(t *testing.T) {
	cfg := LoadConfig()
	if cfg.Network != "base-mainnet" {
		t.Fatalf("network=%s", cfg.Network)
	}
	if cfg.ChainID != 8453 {
		t.Fatalf("chain id=%d", cfg.ChainID)
	}
	if cfg.LockSeconds != 31536000 {
		t.Fatalf("lock=%d", cfg.LockSeconds)
	}
	usdc, ok := cfg.Assets["USDC"]
	if !ok {
		t.Fatal("USDC missing")
	}
	if usdc.Decimals != 6 {
		t.Fatalf("USDC decimals=%d", usdc.Decimals)
	}
	if usdc.Address.Hex() != "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913" {
		t.Fatalf("wrong Base mainnet USDC address: %s", usdc.Address.Hex())
	}
	if cfg.Assets["USDT"].Address.Hex() != "0x0000000000000000000000000000000000000000" {
		t.Fatal("USDT should remain disabled until contract is verified")
	}
}
