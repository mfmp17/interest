package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ---- Mock state store -------------------------------------------------------
// Simulates the custodial backend LOCALLY so you can feel the full UX with no
// real money and no chain. Persists to ~/.interest/state.json.
//
// In production this state lives server-side, keyed to a real per-deposit
// address derived from an HD wallet, and "confirmed" is driven by an on-chain
// watcher — NOT by a local timer. Everything here is clearly a simulation.

const (
	planClassic = "classic8" // 8% APR, streamed monthly, principal locked 365d
	planInstant = "instant5" // 5% paid up front, principal locked 365d, no more yield
	lockDays    = 365
)

type Position struct {
	ID           string    `json:"id"`
	Asset        string    `json:"asset"`
	Principal    float64   `json:"principal"`
	Plan         string    `json:"plan"`
	DepositAddr  string    `json:"deposit_addr"`
	CreatedAt    time.Time `json:"created_at"`
	UnlockAt     time.Time `json:"unlock_at"`
	Confirmed    bool      `json:"confirmed"`
	InstantPaid  float64   `json:"instant_paid"`  // instant-5% payout already available
	InterestPaid float64   `json:"interest_paid"` // classic: total claimed so far
}

type State struct {
	Positions []Position `json:"positions"`
}

func statePath() string {
	dir := filepath.Join(os.Getenv("HOME"), ".interest")
	os.MkdirAll(dir, 0o700)
	return filepath.Join(dir, "state.json")
}

func loadState() *State {
	s := &State{}
	b, err := os.ReadFile(statePath())
	if err == nil {
		json.Unmarshal(b, s)
	}
	return s
}

func (s *State) save() error {
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(statePath(), b, 0o600)
}

func (s *State) active() *Position {
	for i := range s.Positions {
		if s.Positions[i].Confirmed {
			return &s.Positions[i]
		}
	}
	return nil
}

// ---- Interest math ----------------------------------------------------------
// Classic 8%: linear accrual over 365 days. claimable = accrued - alreadyClaimed.
func (p *Position) classicAccrued(now time.Time) float64 {
	if p.Plan != planClassic {
		return 0
	}
	elapsed := now.Sub(p.CreatedAt).Hours() / 24.0
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > lockDays {
		elapsed = lockDays
	}
	annual := p.Principal * 0.08
	return annual * (elapsed / float64(lockDays))
}

func (p *Position) claimable(now time.Time) float64 {
	switch p.Plan {
	case planClassic:
		c := p.classicAccrued(now) - p.InterestPaid
		if c < 0 {
			return 0
		}
		return c
	case planInstant:
		return p.InstantPaid // the up-front 5%, until withdrawn
	}
	return 0
}

func (p *Position) daysLeft(now time.Time) int {
	d := int(p.UnlockAt.Sub(now).Hours() / 24.0)
	if d < 0 {
		return 0
	}
	return d
}

func (p *Position) unlocked(now time.Time) bool {
	return !now.Before(p.UnlockAt)
}

// mockAddress returns a deterministic-looking fake Base address for the demo.
func mockAddress(seed string) string {
	h := uint32(2166136261)
	for _, c := range seed {
		h ^= uint32(c)
		h *= 16777619
	}
	return fmt.Sprintf("0x%08x%08xAbCd%04x", h, h*7+13, h%65536)
}
