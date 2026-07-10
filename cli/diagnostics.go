package main

import "time"

type SupportBundle struct {
	GeneratedAt string           `json:"generated_at"`
	CLIVersion  string           `json:"cli_version"`
	API         string           `json:"api"`
	Service     *SupportService  `json:"service,omitempty"`
	Receipt     *SupportReceipt  `json:"receipt,omitempty"`
	Position    *SupportPosition `json:"position,omitempty"`
	Errors      []string         `json:"errors,omitempty"`
}

type SupportService struct {
	Status             string `json:"status"`
	Network            string `json:"network"`
	ChainID            int64  `json:"chain_id"`
	TVL                string `json:"tvl"`
	TreasuryUSDC       string `json:"treasury_usdc"`
	TreasuryETH        string `json:"treasury_eth"`
	TotalLiability     string `json:"total_liability"`
	ReserveRatio       string `json:"reserve_ratio"`
	Underfunded        bool   `json:"underfunded"`
	Shortfall          string `json:"shortfall"`
	ScannerLatestBlock uint64 `json:"scanner_latest_block"`
	ScannerLastBlock   uint64 `json:"scanner_last_block"`
	ScannerLagBlocks   uint64 `json:"scanner_lag_blocks"`
	ScannerLastScanAt  string `json:"scanner_last_scan_at"`
}

type SupportReceipt struct {
	ID             string `json:"id"`
	Asset          string `json:"asset"`
	Plan           string `json:"plan"`
	DepositAddress string `json:"deposit_address"`
	Network        string `json:"network"`
	ChainID        int64  `json:"chain_id"`
	CreatedAt      string `json:"created_at"`
}

type SupportPosition struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	DepositAddress   string `json:"deposit_address"`
	Principal        string `json:"principal"`
	Expected         string `json:"expected"`
	Missing          string `json:"missing"`
	Extra            string `json:"extra"`
	Claimable        string `json:"claimable"`
	DepositBalance   string `json:"deposit_balance"`
	FundingCount     int    `json:"funding_count"`
	DepositTx        string `json:"deposit_tx,omitempty"`
	SweepTx          string `json:"sweep_tx,omitempty"`
	LastScannedBlock uint64 `json:"last_scanned_block"`
	LastScanAt       string `json:"last_scan_at,omitempty"`
}

func BuildSupportBundle(cliVersion, api string, status *statusResp, r *receipt, pos *positionResp) SupportBundle {
	bundle := SupportBundle{GeneratedAt: time.Now().UTC().Format(time.RFC3339), CLIVersion: cliVersion, API: api}
	if status != nil {
		bundle.Service = &SupportService{
			Status: status.Status, Network: status.Network, ChainID: status.ChainID, TVL: status.TVL,
			TreasuryUSDC: status.TreasuryUSDC, TreasuryETH: status.TreasuryETH,
			TotalLiability: status.TotalLiability, ReserveRatio: status.ReserveRatio,
			Underfunded: status.Underfunded, Shortfall: status.Shortfall,
			ScannerLatestBlock: status.ScannerLatestBlock, ScannerLastBlock: status.ScannerLastBlock,
			ScannerLagBlocks: status.ScannerLagBlocks, ScannerLastScanAt: status.ScannerLastScanAt,
		}
	}
	if r != nil {
		bundle.Receipt = &SupportReceipt{ID: r.ID, Asset: r.Asset, Plan: r.Plan, DepositAddress: r.DepositAddress, Network: r.Network, ChainID: r.ChainID, CreatedAt: r.CreatedAt}
	}
	if pos != nil {
		bundle.Position = &SupportPosition{
			ID: pos.Position.ID, Status: pos.Position.Status, DepositAddress: pos.Position.DepositAddress,
			Principal: pos.PrincipalDisplay, Expected: pos.ExpectedDisplay, Missing: pos.MissingDisplay,
			Extra: pos.ExtraDisplay, Claimable: pos.ClaimableDisplay, DepositBalance: pos.DepositBalanceDisplay,
			FundingCount: pos.FundingCount, DepositTx: pos.Position.DepositTx, SweepTx: pos.Position.SweepTx,
			LastScannedBlock: pos.Position.LastScannedBlock, LastScanAt: pos.Position.LastScanAt,
		}
	}
	return bundle
}
