package main

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type TxResult struct {
	TxHash string `json:"tx_hash"`
	Amount string `json:"amount"`
	Asset  string `json:"asset"`
}

func (a *App) Claim(ctx context.Context, id, token, to string) (*TxResult, error) {
	p, ok := a.store.Get(id)
	if !ok {
		return nil, errors.New("position not found")
	}
	if !CheckToken(p.ClaimTokenHash, token) {
		return nil, errors.New("bad claim token")
	}
	if p.Status != StatusConfirmed {
		return nil, errors.New("position not confirmed")
	}
	amt := p.Claimable(time.Now())
	if amt.Sign() <= 0 {
		return nil, errors.New("nothing claimable")
	}
	asset := a.cfg.Assets[p.Asset]
	hash, err := a.sendFromTreasury(ctx, asset.Address, common.HexToAddress(to), amt)
	if err != nil {
		return nil, err
	}
	if p.Plan == PlanClassic8 {
		p.InterestPaid = new(big.Int).Add(p.interestPaidInt(), amt).String()
	} else {
		p.InstantPaid = "0"
	}
	_ = a.store.Upsert(p)
	return &TxResult{TxHash: hash.Hex(), Amount: amt.String(), Asset: p.Asset}, nil
}

func (a *App) WithdrawPrincipal(ctx context.Context, id, token, to string) (*TxResult, error) {
	p, ok := a.store.Get(id)
	if !ok {
		return nil, errors.New("position not found")
	}
	if !CheckToken(p.ClaimTokenHash, token) {
		return nil, errors.New("bad claim token")
	}
	if p.Status != StatusConfirmed {
		return nil, errors.New("position not confirmed")
	}
	if !p.Unlocked(time.Now()) {
		return nil, errors.New("principal still locked")
	}
	if p.PrincipalPaid {
		return nil, errors.New("principal already paid")
	}
	asset := a.cfg.Assets[p.Asset]
	amt := p.principalInt()
	hash, err := a.sendFromTreasury(ctx, asset.Address, common.HexToAddress(to), amt)
	if err != nil {
		return nil, err
	}
	p.PrincipalPaid = true
	p.Status = StatusClosed
	_ = a.store.Upsert(p)
	return &TxResult{TxHash: hash.Hex(), Amount: amt.String(), Asset: p.Asset}, nil
}

func (a *App) sendFromTreasury(ctx context.Context, token, to common.Address, amount *big.Int) (common.Hash, error) {
	if a.cfg.TreasuryKey == "" {
		return common.Hash{}, errors.New("treasury key missing")
	}
	pk, err := PrivateKeyFromHex(a.cfg.TreasuryKey)
	if err != nil {
		return common.Hash{}, err
	}
	return a.chain.SendERC20(ctx, pk, token, to, amount)
}
