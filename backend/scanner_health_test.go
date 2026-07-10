package main

import (
	"testing"
	"time"
)

func TestBuildScannerHealthReportsOldestCursorAndPendingCount(t *testing.T) {
	t1 := time.Unix(1_000, 0).UTC()
	t2 := time.Unix(2_000, 0).UTC()
	positions := []*Position{
		{Status: StatusPending, StartBlock: 90, LastScannedBlock: 100, LastScanAt: &t1},
		{Status: StatusConfirmed, StartBlock: 95, LastScannedBlock: 110, LastScanAt: &t2},
		{Status: StatusClosed, StartBlock: 1, LastScannedBlock: 1},
	}

	got := BuildScannerHealth(positions, 120)
	if got.ActivePositions != 2 || got.PendingPositions != 1 || got.LastScannedBlock != 100 || got.LagBlocks != 20 || got.LastScanAt != t1.Format(time.RFC3339) {
		t.Fatalf("unexpected scanner health: %+v", got)
	}
}

func TestBuildScannerHealthIsCurrentWithNoActivePositions(t *testing.T) {
	got := BuildScannerHealth(nil, 120)
	if got.ActivePositions != 0 || got.LastScannedBlock != 120 || got.LagBlocks != 0 {
		t.Fatalf("unexpected empty scanner health: %+v", got)
	}
}
