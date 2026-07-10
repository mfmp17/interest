package main

import (
	"testing"
	"time"
)

func TestConfirmedScanRangeStartsAtPositionBlockAndStopsAtConfirmedHead(t *testing.T) {
	p := &Position{StartBlock: 100}
	from, to, ok := ConfirmedScanRange(p, 120, 3)
	if !ok || from != 100 || to != 118 {
		t.Fatalf("range = %d..%d ok=%v, want 100..118 true", from, to, ok)
	}
}

func TestConfirmedScanRangeContinuesAfterCursor(t *testing.T) {
	p := &Position{StartBlock: 100, LastScannedBlock: 110}
	from, to, ok := ConfirmedScanRange(p, 120, 3)
	if !ok || from != 111 || to != 118 {
		t.Fatalf("range = %d..%d ok=%v, want 111..118 true", from, to, ok)
	}
}

func TestConfirmedScanRangeWaitsWhenNothingNewIsConfirmed(t *testing.T) {
	p := &Position{StartBlock: 100, LastScannedBlock: 118}
	_, _, ok := ConfirmedScanRange(p, 120, 3)
	if ok {
		t.Fatal("expected no scan range")
	}
}

func TestMarkScanCompleteStoresCursorAndTime(t *testing.T) {
	p := &Position{}
	now := time.Unix(2_000, 0).UTC()
	MarkScanComplete(p, 123, now)
	if p.LastScannedBlock != 123 || p.LastScanAt == nil || !p.LastScanAt.Equal(now) {
		t.Fatalf("scan metadata = block %d time %v", p.LastScannedBlock, p.LastScanAt)
	}
}
