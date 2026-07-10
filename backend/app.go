package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type App struct {
	cfg      Config
	store    *Store
	chain    *ChainClient
	treasury *big.Int
}

func NewApp(ctx context.Context, cfg Config) (*App, error) {
	st, err := OpenStore(cfg.StatePath)
	if err != nil {
		return nil, err
	}
	cc, err := NewChainClient(ctx, cfg.RPCURL, cfg.ChainID)
	if err != nil {
		return nil, err
	}
	return &App{cfg: cfg, store: st, chain: cc}, nil
}

func (a *App) Close() {
	if a.chain != nil {
		a.chain.Close()
	}
}

type CreateDepositResult struct {
	Position   *Position `json:"position"`
	ClaimToken string    `json:"claim_token"`
	DepositURI string    `json:"deposit_uri"`
	Network    string    `json:"network"`
	ChainID    int64     `json:"chain_id"`
}

func (a *App) CreateDeposit(ctx context.Context, amountStr, assetSymbol string, plan Plan) (*CreateDepositResult, error) {
	asset, ok := a.cfg.Assets[assetSymbol]
	if !ok {
		return nil, fmt.Errorf("unsupported asset %s", assetSymbol)
	}
	if asset.Address == (common.Address{}) {
		return nil, fmt.Errorf("asset %s has no token address configured", assetSymbol)
	}
	if plan != PlanClassic8 && plan != PlanInstant5 {
		return nil, errors.New("invalid plan")
	}
	amount, err := ParseUnits(amountStr, asset.Decimals)
	if err != nil {
		return nil, err
	}
	idx := a.store.NextIndex()
	wallet, err := DeriveDepositAddress(a.cfg.MasterSeed, idx)
	if err != nil {
		return nil, err
	}
	id, _ := NewToken(12)
	claim, _ := NewToken(24)
	now := time.Now().UTC()
	p := NewPosition(id, asset.Symbol, asset.Decimals, amount, plan, now, a.cfg.LockSeconds, idx, wallet.Address.Hex())
	p.ClaimTokenHash = HashToken(claim)
	if blk, err := a.chain.LatestBlock(ctx); err == nil {
		p.StartBlock = blk
	}
	if err := a.store.Upsert(p); err != nil {
		return nil, err
	}
	return &CreateDepositResult{Position: p, ClaimToken: claim, DepositURI: EIP681(asset, wallet.Address, amount, a.cfg.ChainID), Network: a.cfg.Network, ChainID: a.cfg.ChainID}, nil
}

func EIP681(asset AssetConfig, to common.Address, amount *big.Int, chainID int64) string {
	return fmt.Sprintf("ethereum:%s@%d/transfer?address=%s&uint256=%s&asset=%s", asset.Address.Hex(), chainID, to.Hex(), amount.String(), asset.Symbol)
}

func (a *App) ScanOnce(ctx context.Context) error {
	latest, err := a.chain.LatestBlock(ctx)
	if err != nil {
		return err
	}
	for _, p := range a.store.All() {
		if p.Status == StatusClosed || p.PrincipalPaid {
			continue
		}
		to := common.HexToAddress(p.DepositAddress)
		fromBlock, toBlock, hasRange := ConfirmedScanRange(p, latest, a.cfg.Confirmations)
		changed := false
		if hasRange {
			trs, err := a.chain.InboundTransfersAnyTokenRange(ctx, to, fromBlock, toBlock)
			if err != nil {
				return err
			}
			for _, tr := range trs {
				asset, supported := a.assetByToken(tr.Token)
				if supported && asset.Decimals == p.Decimals {
					if p.AddFunding(tr.TxHash.Hex(), tr.From.Hex(), tr.Amount, tr.Block, time.Now().UTC()) {
						changed = true
						if asset.Symbol != p.Asset {
							if p.AddAdminAlert(AdminAlert{Kind: "alternate_stable_accepted", TxHash: tr.TxHash.Hex(), From: tr.From.Hex(), To: tr.To.Hex(), Token: tr.Token.Hex(), Amount: tr.Amount.String(), Block: tr.Block, Message: "supported stable sent to deposit address and counted as principal"}) {
								log.Printf("ADMIN_ALERT position=%s kind=alternate_stable_accepted token=%s amount=%s tx=%s from=%s", p.ID, tr.Token.Hex(), tr.Amount.String(), tr.TxHash.Hex(), tr.From.Hex())
								changed = true
							}
						}
					}
					continue
				}
				kind := "unsupported_token"
				msg := "unsupported ERC-20 sent to deposit address; admin recovery may be needed"
				if supported && asset.Decimals != p.Decimals {
					kind = "unsupported_token_decimals"
					msg = "supported token sent but decimals do not match this position; admin review needed"
				}
				if p.AddAdminAlert(AdminAlert{Kind: kind, TxHash: tr.TxHash.Hex(), From: tr.From.Hex(), To: tr.To.Hex(), Token: tr.Token.Hex(), Amount: tr.Amount.String(), Block: tr.Block, Message: msg}) {
					log.Printf("ADMIN_ALERT position=%s kind=%s token=%s amount=%s tx=%s from=%s", p.ID, kind, tr.Token.Hex(), tr.Amount.String(), tr.TxHash.Hex(), tr.From.Hex())
					changed = true
				}
			}
			MarkScanComplete(p, toBlock, time.Now())
			changed = true
		}
		if changed {
			_ = a.store.Upsert(p)
		}
		if p.Status == StatusConfirmed {
			_ = a.TrySweep(ctx, p)
		}
	}
	return nil
}

func (a *App) assetByToken(token common.Address) (AssetConfig, bool) {
	for _, asset := range a.cfg.Assets {
		if asset.Address == (common.Address{}) {
			continue
		}
		if asset.Address == token {
			return asset, true
		}
	}
	return AssetConfig{}, false
}

func (a *App) TrySweep(ctx context.Context, p *Position) error {
	if a.cfg.TreasuryKey == "" {
		return errors.New("treasury key missing")
	}
	asset := a.cfg.Assets[p.Asset]
	dep, err := DeriveDepositAddress(a.cfg.MasterSeed, p.DeriveIndex)
	if err != nil {
		return err
	}
	treasuryPK, err := PrivateKeyFromHex(a.cfg.TreasuryKey)
	if err != nil {
		return err
	}
	treasuryAddr := AddressFromPrivateKey(treasuryPK)
	eth, err := a.chain.NativeBalance(ctx, dep.Address)
	if err != nil {
		return err
	}
	minGas := big.NewInt(30_000_000_000_000) // 0.00003 ETH
	if eth.Cmp(minGas) < 0 && p.GasTx == "" {
		h, err := a.chain.SendETH(ctx, treasuryPK, dep.Address, minGas)
		if err != nil {
			return err
		}
		p.GasTx = h.Hex()
		_ = a.store.Upsert(p)
		return nil
	}
	bal, err := a.chain.ERC20Balance(ctx, asset.Address, dep.Address)
	if err != nil {
		return err
	}
	if bal.Sign() == 0 {
		return nil
	}
	h, err := a.chain.SendERC20(ctx, dep.PrivateKey, asset.Address, treasuryAddr, bal)
	if err != nil {
		return err
	}
	p.SweepTx = h.Hex()
	return a.store.Upsert(p)
}
