package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Server struct{ app *App }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/v1/status", s.status)
	mux.HandleFunc("/v1/deposits", s.createDeposit)
	mux.HandleFunc("/v1/admin/positions/", s.adminPositionAction)
	mux.HandleFunc("/v1/positions/", s.positionAction)
	return cors(mux)
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	asset := s.app.cfg.Assets["USDC"]
	ethBal, ethErr := s.app.chain.NativeBalance(r.Context(), s.app.cfg.TreasuryAddr)
	usdcBal, usdcErr := s.app.chain.ERC20Balance(r.Context(), asset.Address, s.app.cfg.TreasuryAddr)
	latestBlock, latestErr := s.app.chain.LatestBlock(r.Context())
	now := time.Now().UTC()
	positions := s.app.store.All()
	accounting := BuildStatusAccounting(positions, now, usdcBal, asset.Decimals)
	scanner := BuildScannerHealth(positions, latestBlock)
	ethDisplay, usdcDisplay := "unknown", "unknown"
	warning := ""
	if ethErr == nil {
		ethDisplay = FormatUnits(ethBal, 18)
	}
	if usdcErr == nil {
		usdcDisplay = FormatUnits(usdcBal, asset.Decimals)
	}
	if ethErr != nil || usdcErr != nil {
		warning = "Could not read treasury balance from Base mainnet right now."
	} else if latestErr != nil {
		warning = "Could not read the latest Base mainnet block; deposit scanner health is unknown."
	} else if scanner.LagBlocks > 100 {
		warning = fmt.Sprintf("Deposit scanner is %d blocks behind Base mainnet.", scanner.LagBlocks)
	} else if accounting.Underfunded {
		warning = "Treasury is underfunded by " + accounting.Shortfall + " USDC versus current principal and claimable liabilities."
	} else if ethBal.Sign() == 0 && usdcBal.Sign() == 0 {
		warning = "Treasury currently holds 0.00 ETH and 0.00 USDC; claims/withdrawals require treasury funding."
	} else if ethBal.Sign() == 0 {
		warning = "Treasury currently holds 0.00 ETH for gas; claims/withdrawals may fail until funded."
	} else if usdcBal.Sign() == 0 {
		warning = "Treasury currently holds 0.00 USDC; claims/withdrawals may fail until funded."
	}

	writeJSON(w, map[string]any{
		"service":              "fred",
		"status":               "operational",
		"apr":                  8,
		"lock_days":            365,
		"assets":               []string{"USDC"},
		"tvl":                  accounting.TVL,
		"principal_liability":  accounting.PrincipalLiability,
		"claimable_liability":  accounting.ClaimableLiability,
		"total_liability":      accounting.TotalLiability,
		"reserve_ratio":        accounting.ReserveRatio,
		"underfunded":          accounting.Underfunded,
		"shortfall":            accounting.Shortfall,
		"scanner_latest_block": latestBlock,
		"scanner_last_block":   scanner.LastScannedBlock,
		"scanner_lag_blocks":   scanner.LagBlocks,
		"scanner_last_scan_at": scanner.LastScanAt,
		"active_positions":     scanner.ActivePositions,
		"pending_positions":    scanner.PendingPositions,
		"network":              s.app.cfg.Network,
		"chain_id":             s.app.cfg.ChainID,
		"lock_seconds":         s.app.cfg.LockSeconds,
		"treasury":             s.app.cfg.TreasuryAddr.Hex(),
		"treasury_eth":         ethDisplay,
		"treasury_usdc":        usdcDisplay,
		"treasury_warning":     warning,
		"server_time":          now.Format(time.RFC3339),
	})
}

type createDepositReq struct {
	Amount, Asset string
	Plan          Plan
}

func (s *Server) createDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	var req createDepositReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	res, err := s.app.CreateDeposit(r.Context(), req.Amount, strings.ToUpper(req.Asset), req.Plan)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, res)
}

func (s *Server) positionAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/positions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		s.getPosition(w, r, id)
		return
	}
	if len(parts) == 2 && r.Method == http.MethodPost {
		switch parts[1] {
		case "claim":
			s.claim(w, r, id)
			return
		case "withdraw":
			s.withdraw(w, r, id)
			return
		}
	}
	http.NotFound(w, r)
}

func (s *Server) getPosition(w http.ResponseWriter, r *http.Request, id string) {
	p, ok := s.app.store.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !CheckToken(p.ClaimTokenHash, r.URL.Query().Get("token")) {
		http.Error(w, "bad claim token", 403)
		return
	}
	_ = s.app.ScanOnce(r.Context())
	p, _ = s.app.store.Get(id)
	asset := s.app.cfg.Assets[p.Asset]
	balance, err := s.app.chain.ERC20Balance(r.Context(), asset.Address, common.HexToAddress(p.DepositAddress))
	if err != nil {
		balance = nil
	}
	writeJSON(w, PositionDTOWithDepositBalance(p, balance))
}

type payoutReq struct{ Token, To string }

func (s *Server) claim(w http.ResponseWriter, r *http.Request, id string) {
	var req payoutReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	res, err := s.app.Claim(r.Context(), id, req.Token, req.To)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, res)
}

func (s *Server) withdraw(w http.ResponseWriter, r *http.Request, id string) {
	var req payoutReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	res, err := s.app.WithdrawPrincipal(r.Context(), id, req.Token, req.To)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, res)
}

func PositionDTOWithDepositBalance(p *Position, balance *big.Int) map[string]any {
	dto := positionDTO(p)
	if balance != nil {
		dto["deposit_balance"] = balance.String()
		dto["deposit_balance_display"] = FormatUnits(balance, p.Decimals)
	}
	return dto
}

func positionDTO(p *Position) map[string]any {
	claimable := p.Claimable(time.Now())
	return map[string]any{
		"position":          p,
		"claimable":         claimable.String(),
		"claimable_display": FormatUnits(claimable, p.Decimals),
		"principal_display": FormatUnits(p.principalInt(), p.Decimals),
		"expected_display":  FormatUnits(p.expectedPrincipalInt(), p.Decimals),
		"missing_display":   FormatUnits(p.MissingPrincipal(), p.Decimals),
		"extra_display":     FormatUnits(p.ExtraPrincipal(), p.Decimals),
		"funding_count":     len(p.Fundings),
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}
