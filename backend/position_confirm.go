package main

import "time"

func (p *Position) MarkConfirmed(at time.Time, tx string) {
	t := at.UTC()
	p.Status = StatusConfirmed
	p.ConfirmedAt = &t
	p.UnlockAt = t.Add(time.Duration(p.LockSeconds) * time.Second)
	p.DepositTx = tx
}

func (p *Position) AccrualStart() time.Time {
	if p.ConfirmedAt != nil {
		return p.ConfirmedAt.UTC()
	}
	return p.CreatedAt.UTC()
}
