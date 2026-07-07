package main

import (
	"math/big"
	"time"
)

type Plan string

const (
	PlanClassic8 Plan = "classic8"
	PlanInstant5 Plan = "instant5"
)

type PositionStatus string

const (
	StatusPending   PositionStatus = "pending_deposit"
	StatusConfirmed PositionStatus = "confirmed_locked"
	StatusClosed    PositionStatus = "closed"
)

type Position struct {
	ID               string         `json:"id"`
	ClaimTokenHash   string         `json:"claim_token_hash"`
	Asset            string         `json:"asset"`
	Decimals         uint8          `json:"decimals"`
	Principal        string         `json:"principal"`
	Plan             Plan           `json:"plan"`
	Status           PositionStatus `json:"status"`
	DepositAddress   string         `json:"deposit_address"`
	DepositorAddress string         `json:"depositor_address,omitempty"`
	DeriveIndex      uint64         `json:"derive_index"`
	CreatedAt        time.Time      `json:"created_at"`
	ConfirmedAt      *time.Time     `json:"confirmed_at,omitempty"`
	UnlockAt         time.Time      `json:"unlock_at"`
	LockSeconds      int64          `json:"lock_seconds"`
	StartBlock       uint64         `json:"start_block"`
	InterestPaid     string         `json:"interest_paid"`
	InstantPaid      string         `json:"instant_paid"`
	PrincipalPaid    bool           `json:"principal_paid"`
	DepositTx        string         `json:"deposit_tx,omitempty"`
	GasTx            string         `json:"gas_tx,omitempty"`
	SweepTx          string         `json:"sweep_tx,omitempty"`
}

func NewPosition(id, asset string, decimals uint8, principal *big.Int, plan Plan, created time.Time, lockSeconds int64, deriveIndex uint64, addr string) *Position {
	p := &Position{
		ID:             id,
		Asset:          asset,
		Decimals:       decimals,
		Principal:      principal.String(),
		Plan:           plan,
		Status:         StatusPending,
		DepositAddress: addr,
		DeriveIndex:    deriveIndex,
		CreatedAt:      created.UTC(),
		UnlockAt:       created.UTC().Add(time.Duration(lockSeconds) * time.Second),
		LockSeconds:    lockSeconds,
		InterestPaid:   "0",
		InstantPaid:    "0",
	}
	if plan == PlanInstant5 {
		p.InstantPaid = pct(principal, 5).String()
	}
	return p
}

func (p *Position) principalInt() *big.Int    { return mustBig(p.Principal) }
func (p *Position) interestPaidInt() *big.Int { return mustBig(p.InterestPaid) }
func (p *Position) instantPaidInt() *big.Int  { return mustBig(p.InstantPaid) }

func mustBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return big.NewInt(0)
	}
	return v
}

func pct(v *big.Int, percent int64) *big.Int {
	out := new(big.Int).Mul(v, big.NewInt(percent))
	return out.Div(out, big.NewInt(100))
}

func (p *Position) Claimable(now time.Time) *big.Int {
	switch p.Plan {
	case PlanInstant5:
		return p.instantPaidInt()
	case PlanClassic8:
		elapsed := now.UTC().Sub(p.AccrualStart()).Seconds()
		if elapsed < 0 {
			elapsed = 0
		}
		if elapsed > float64(p.LockSeconds) {
			elapsed = float64(p.LockSeconds)
		}
		max := pct(p.principalInt(), 8)
		accrued := new(big.Int).Mul(max, big.NewInt(int64(elapsed)))
		accrued.Div(accrued, big.NewInt(p.LockSeconds))
		claimable := new(big.Int).Sub(accrued, p.interestPaidInt())
		if claimable.Sign() < 0 {
			return big.NewInt(0)
		}
		return claimable
	default:
		return big.NewInt(0)
	}
}

func (p *Position) Unlocked(now time.Time) bool { return !now.UTC().Before(p.UnlockAt) }
