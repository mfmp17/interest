package main

import (
	"math/big"
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsPositionsAndNextIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	st, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}

	idx := st.NextIndex()
	if idx != 0 {
		t.Fatalf("first index=%d want 0", idx)
	}
	p := NewPosition("p1", "USDC", 6, big.NewInt(1000), PlanClassic8, time.Unix(1, 0), 365, idx, "0xabc")
	if err := st.Upsert(p); err != nil {
		t.Fatal(err)
	}

	st2, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if st2.NextIndex() != 1 {
		t.Fatalf("next index after reload=%d want 1", st2.NextIndex())
	}
	got, ok := st2.Get("p1")
	if !ok {
		t.Fatal("position missing after reload")
	}
	if got.DepositAddress != "0xabc" || got.ExpectedPrincipal != "1000" || got.Principal != "0" {
		t.Fatalf("bad loaded position: %+v", got)
	}
}
