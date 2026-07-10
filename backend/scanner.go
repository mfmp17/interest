package main

import "time"

type ScannerHealth struct {
	ActivePositions  int
	PendingPositions int
	LastScannedBlock uint64
	LagBlocks        uint64
	LastScanAt       string
}

func BuildScannerHealth(positions []*Position, latest uint64) ScannerHealth {
	h := ScannerHealth{LastScannedBlock: latest}
	var oldestScan *time.Time
	initialized := false
	for _, p := range positions {
		if p == nil || p.Status == StatusClosed || p.PrincipalPaid {
			continue
		}
		h.ActivePositions++
		if p.Status == StatusPending {
			h.PendingPositions++
		}
		cursor := p.LastScannedBlock
		if cursor == 0 && p.StartBlock > 0 {
			cursor = p.StartBlock - 1
		}
		if !initialized || cursor < h.LastScannedBlock {
			h.LastScannedBlock = cursor
			initialized = true
		}
		if p.LastScanAt != nil && (oldestScan == nil || p.LastScanAt.Before(*oldestScan)) {
			t := p.LastScanAt.UTC()
			oldestScan = &t
		}
	}
	if h.LastScannedBlock < latest {
		h.LagBlocks = latest - h.LastScannedBlock
	}
	if oldestScan != nil {
		h.LastScanAt = oldestScan.Format(time.RFC3339)
	}
	return h
}

func MarkScanComplete(p *Position, block uint64, at time.Time) {
	if p == nil {
		return
	}
	at = at.UTC()
	p.LastScannedBlock = block
	p.LastScanAt = &at
}

func ConfirmedScanRange(p *Position, latest, confirmations uint64) (from, to uint64, ok bool) {
	if p == nil {
		return 0, 0, false
	}
	if confirmations == 0 {
		confirmations = 1
	}
	if latest+1 < confirmations {
		return 0, 0, false
	}
	confirmedThrough := latest - confirmations + 1
	from = p.StartBlock
	if p.LastScannedBlock >= from {
		if p.LastScannedBlock == ^uint64(0) {
			return 0, 0, false
		}
		from = p.LastScannedBlock + 1
	}
	if from > confirmedThrough {
		return 0, 0, false
	}
	return from, confirmedThrough, true
}
