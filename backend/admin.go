package main

import (
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func (s *Server) adminPositionAction(w http.ResponseWriter, r *http.Request) {
	if s.app.cfg.AdminToken == "" {
		http.Error(w, "admin disabled", http.StatusNotFound)
		return
	}
	if r.Header.Get("x-admin-token") != s.app.cfg.AdminToken {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/v1/admin/positions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	id, action := parts[0], parts[1]
	p, ok := s.app.store.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch action {
	case "alerts":
		writeJSON(w, map[string]any{"position_id": id, "deposit_address": p.DepositAddress, "alerts": p.AdminAlerts})
	case "export-key":
		wallet, err := DeriveDepositAddress(s.app.cfg.MasterSeed, p.DeriveIndex)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, map[string]any{
			"position_id":      id,
			"deposit_address":  wallet.Address.Hex(),
			"private_key_hex":  hex.EncodeToString(crypto.FromECDSA(wallet.PrivateKey)),
			"recovery_warning": "import only into a secure admin wallet; never share with users",
		})
	default:
		http.NotFound(w, r)
	}
}
