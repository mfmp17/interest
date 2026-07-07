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
