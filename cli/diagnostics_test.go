package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildSupportBundleNeverIncludesClaimToken(t *testing.T) {
	r := receipt{ID: "pos-123", Token: "super-secret-claim-token", Asset: "USDC", Plan: "classic8", DepositAddress: "0xabc", Network: "base-mainnet", ChainID: 8453}
	pos := positionResp{Position: backendPosition{ID: "pos-123", Status: "pending_deposit", DepositAddress: "0xabc"}, ExpectedDisplay: "10.00", PrincipalDisplay: "1.00", MissingDisplay: "9.00", DepositBalanceDisplay: "1.00"}
	status := statusResp{Status: "operational", TreasuryUSDC: "55.00", TreasuryETH: "0.003", ScannerLagBlocks: 2}

	bundle := BuildSupportBundle("0.3.0", "https://api.fred.cash", &status, &r, &pos)
	b, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, r.Token) {
		t.Fatalf("support bundle leaked claim token: %s", s)
	}
	for _, want := range []string{"pos-123", "0xabc", "pending_deposit", "base-mainnet"} {
		if !strings.Contains(s, want) {
			t.Fatalf("support bundle missing %q: %s", want, s)
		}
	}
}
