package main

import "testing"

func TestDeriveDepositAddressDeterministicAndUnique(t *testing.T) {
	seed := []byte("test master seed: never use this in prod")

	a1, err := DeriveDepositAddress(seed, 0)
	if err != nil {
		t.Fatalf("derive index 0: %v", err)
	}
	a1b, err := DeriveDepositAddress(seed, 0)
	if err != nil {
		t.Fatalf("derive index 0 again: %v", err)
	}
	a2, err := DeriveDepositAddress(seed, 1)
	if err != nil {
		t.Fatalf("derive index 1: %v", err)
	}

	if a1.Address != a1b.Address {
		t.Fatalf("same seed/index produced different addresses: %s vs %s", a1.Address.Hex(), a1b.Address.Hex())
	}
	if a1.Address == a2.Address {
		t.Fatalf("different indexes produced same address: %s", a1.Address.Hex())
	}
	if a1.PrivateKey == nil || a2.PrivateKey == nil {
		t.Fatal("private keys must be returned for custodial signing/sweeping")
	}
}
