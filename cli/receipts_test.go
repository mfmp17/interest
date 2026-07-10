package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelectReceiptAcceptsUniqueIDPrefix(t *testing.T) {
	st := receiptStore{Receipts: []receipt{{ID: "alpha-123"}, {ID: "beta-456"}}}
	got, err := SelectReceipt(&st, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "beta-456" || st.Active != "beta-456" {
		t.Fatalf("got=%+v active=%q", got, st.Active)
	}
}

func TestMergeReceiptStoresDeduplicatesByID(t *testing.T) {
	dst := receiptStore{Active: "old", Receipts: []receipt{{ID: "old", Token: "old-token"}}}
	src := receiptStore{Active: "new", Receipts: []receipt{{ID: "old", Token: "replacement"}, {ID: "new", Token: "new-token"}}}
	got := MergeReceiptStores(dst, src)
	if got.Active != "new" || len(got.Receipts) != 2 {
		t.Fatalf("unexpected merged store: %+v", got)
	}
	if got.Receipts[0].Token != "replacement" {
		t.Fatalf("existing receipt was not updated: %+v", got.Receipts[0])
	}
}

func TestExportAndImportReceiptStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	want := receiptStore{Active: "p1", Receipts: []receipt{{ID: "p1", Token: "secret"}}}
	if err := ExportReceiptStore(path, want); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	got, err := ImportReceiptStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active != "p1" || len(got.Receipts) != 1 || got.Receipts[0].Token != "secret" {
		t.Fatalf("unexpected imported store: %+v", got)
	}
}
