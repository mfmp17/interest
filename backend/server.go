package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Server struct{ app *App }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/v1/status", s.status)
	mux.HandleFunc("/v1/deposits", s.createDeposit)
	mux.HandleFunc("/v1/positions/", s.positionAction)
	return cors(mux)
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"service":      "fred",
		"status":       "operational",
		"apr":          8,
		"lock_days":    365,
		"assets":       []string{"USDC"},
		"tvl":          "$0",
		"network":      s.app.cfg.Network,
		"chain_id":     s.app.cfg.ChainID,
		"lock_seconds": s.app.cfg.LockSeconds,
		"treasury":     s.app.cfg.TreasuryAddr.Hex(),
		"server_time":  time.Now().UTC().Format(time.RFC3339),
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
	writeJSON(w, positionDTO(p))
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

func positionDTO(p *Position) map[string]any {
	return map[string]any{"position": p, "claimable": p.Claimable(time.Now()).String(), "claimable_display": FormatUnits(p.Claimable(time.Now()), p.Decimals), "principal_display": FormatUnits(p.principalInt(), p.Decimals)}
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
