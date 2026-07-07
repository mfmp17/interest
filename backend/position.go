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

type Funding struct {
	TxHash      string    `json:"tx_hash"`
	From        string    `json:"from"`
	Amount      string    `json:"amount"`
	Block       uint64    `json:"block"`
	ConfirmedAt time.Time `json:"confirmed_at"`
}

type AdminAlert struct {
	Kind      string    `json:"kind"`
	TxHash    string    `json:"tx_hash"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Token     string    `json:"token"`
	Amount    string    `json:"amount"`
	Block     uint64    `json:"block"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type Position struct {
	ID                string         `json:"id"`
	ClaimTokenHash    string         `json:"claim_token_hash"`
	Asset             string         `json:"asset"`
	Decimals          uint8          `json:"decimals"`
	ExpectedPrincipal string         `json:"expected_principal"`
	Principal         string         `json:"principal"`
	Plan              Plan           `json:"plan"`
	Status            PositionStatus `json:"status"`
	DepositAddress    string         `json:"deposit_address"`
	DepositorAddress  string         `json:"depositor_address,omitempty"`
	DeriveIndex       uint64         `json:"derive_index"`
	CreatedAt         time.Time      `json:"created_at"`
	ConfirmedAt       *time.Time     `json:"confirmed_at,omitempty"`
	UnlockAt          time.Time      `json:"unlock_at"`
	LockSeconds       int64          `json:"lock_seconds"`
	StartBlock        uint64         `json:"start_block"`
	Fundings          []Funding      `json:"fundings,omitempty"`
	AdminAlerts       []AdminAlert   `json:"admin_alerts,omitempty"`
	InterestPaid      string         `json:"interest_paid"`
	InstantPaid       string         `json:"instant_paid"`
	PrincipalPaid     bool           `json:"principal_paid"`
	DepositTx         string         `json:"deposit_tx,omitempty"`
	GasTx             string         `json:"gas_tx,omitempty"`
	SweepTx           string         `json:"sweep_tx,omitempty"`
}

func NewPosition(id, asset string, decimals uint8, expectedPrincipal *big.Int, plan Plan, created time.Time, lockSeconds int64, deriveIndex uint64, addr string) *Position {
	return &Position{
		ID:                id,
		Asset:             asset,
		Decimals:          decimals,
		ExpectedPrincipal: expectedPrincipal.String(),
		Principal:         "0",
		Plan:              plan,
		Status:            StatusPending,
		DepositAddress:    addr,
		DeriveIndex:       deriveIndex,
		CreatedAt:         created.UTC(),
		UnlockAt:          created.UTC().Add(time.Duration(lockSeconds) * time.Second),
		LockSeconds:       lockSeconds,
		InterestPaid:      "0",
		InstantPaid:       "0",
	}
}

func (p *Position) principalInt() *big.Int { return mustBig(p.Principal) }
func (p *Position) NormalizeFundingModel() {
	if p.ExpectedPrincipal == "" {
		p.ExpectedPrincipal = p.Principal
		if len(p.Fundings) == 0 && p.Status == StatusPending {
			p.Principal = "0"
			p.InstantPaid = "0"
		}
	}
	if p.Principal == "" {
		p.Principal = "0"
	}
	if p.InterestPaid == "" {
		p.InterestPaid = "0"
	}
	if p.InstantPaid == "" {
		p.InstantPaid = "0"
	}
}
func (p *Position) expectedPrincipalInt() *big.Int {
	if p.ExpectedPrincipal == "" {
		return p.principalInt()
	}
	return mustBig(p.ExpectedPrincipal)
}
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

func (p *Position) HasFunding(tx string) bool {
	for _, f := range p.Fundings {
		if f.TxHash == tx {
			return true
		}
	}
	return false
}

func (p *Position) HasAlert(kind, tx, token string) bool {
	for _, a := range p.AdminAlerts {
		if a.Kind == kind && a.TxHash == tx && a.Token == token {
			return true
		}
	}
	return false
}

func (p *Position) AddAdminAlert(alert AdminAlert) bool {
	if p.HasAlert(alert.Kind, alert.TxHash, alert.Token) {
		return false
	}
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}
	p.AdminAlerts = append(p.AdminAlerts, alert)
	return true
}

func (p *Position) AddFunding(tx, from string, amount *big.Int, block uint64, at time.Time) bool {
	if amount == nil || amount.Sign() <= 0 || p.HasFunding(tx) {
		return false
	}
	at = at.UTC()
	p.Fundings = append(p.Fundings, Funding{TxHash: tx, From: from, Amount: amount.String(), Block: block, ConfirmedAt: at})
	p.Principal = new(big.Int).Add(p.principalInt(), amount).String()
	if p.DepositorAddress == "" {
		p.DepositorAddress = from
	}
	if p.Plan == PlanInstant5 {
		p.InstantPaid = new(big.Int).Add(p.instantPaidInt(), pct(amount, 5)).String()
	}
	if p.Status == StatusPending {
		p.MarkConfirmed(at, tx)
	} else {
		p.DepositTx = tx
	}
	unlock := at.Add(time.Duration(p.LockSeconds) * time.Second)
	if unlock.After(p.UnlockAt) {
		p.UnlockAt = unlock
	}
	return true
}

func (p *Position) MissingPrincipal() *big.Int {
	missing := new(big.Int).Sub(p.expectedPrincipalInt(), p.principalInt())
	if missing.Sign() < 0 {
		return big.NewInt(0)
	}
	return missing
}

func (p *Position) ExtraPrincipal() *big.Int {
	extra := new(big.Int).Sub(p.principalInt(), p.expectedPrincipalInt())
	if extra.Sign() < 0 {
		return big.NewInt(0)
	}
	return extra
}

func (p *Position) Claimable(now time.Time) *big.Int {
	switch p.Plan {
	case PlanInstant5:
		return p.instantPaidInt()
	case PlanClassic8:
		accrued := big.NewInt(0)
		if len(p.Fundings) == 0 && p.principalInt().Sign() > 0 {
			accrued = accruedClassic(p.principalInt(), p.AccrualStart(), now.UTC(), p.LockSeconds)
		} else {
			for _, f := range p.Fundings {
				accrued.Add(accrued, accruedClassic(mustBig(f.Amount), f.ConfirmedAt, now.UTC(), p.LockSeconds))
			}
		}
		claimable := new(big.Int).Sub(accrued, p.interestPaidInt())
		if claimable.Sign() < 0 {
			return big.NewInt(0)
		}
		return claimable
	default:
		return big.NewInt(0)
	}
}

func accruedClassic(amount *big.Int, start, now time.Time, lockSeconds int64) *big.Int {
	elapsed := now.Sub(start.UTC()).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > float64(lockSeconds) {
		elapsed = float64(lockSeconds)
	}
	max := pct(amount, 8)
	accrued := new(big.Int).Mul(max, big.NewInt(int64(elapsed)))
	return accrued.Div(accrued, big.NewInt(lockSeconds))
}

func (p *Position) Unlocked(now time.Time) bool { return !now.UTC().Before(p.UnlockAt) }
