package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreMigratesOldPendingRequestedPrincipal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	old := StoreState{Positions: []*Position{{
		ID:             "old",
		Asset:          "USDC",
		Decimals:       6,
		Principal:      "10000000",
		Plan:           PlanClassic8,
		Status:         StatusPending,
		DepositAddress: "0xabc",
		CreatedAt:      time.Unix(1, 0),
		UnlockAt:       time.Unix(366, 0),
		LockSeconds:    365,
		InterestPaid:   "0",
		InstantPaid:    "0",
	}}}
	b, _ := json.Marshal(old)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := st.Get("old")
	if !ok {
		t.Fatal("missing")
	}
	if p.ExpectedPrincipal != "10000000" || p.Principal != "0" {
		t.Fatalf("expected migrated expected=10000000 principal=0, got expected=%s principal=%s", p.ExpectedPrincipal, p.Principal)
	}
}

func TestStoreLoadsPreCursorStateAndScansFromStartBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	raw := `{"next_derive_index":1,"positions":[{"id":"legacy","asset":"USDC","decimals":6,"expected_principal":"10000000","principal":"0","plan":"classic8","status":"pending_deposit","deposit_address":"0xabc","start_block":100,"interest_paid":"0","instant_paid":"0"}]}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := st.Get("legacy")
	if !ok {
		t.Fatal("missing legacy position")
	}
	from, to, ok := ConfirmedScanRange(p, 120, 3)
	if !ok || from != 100 || to != 118 {
		t.Fatalf("legacy scan range = %d..%d ok=%v", from, to, ok)
	}
}
