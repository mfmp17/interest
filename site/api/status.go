package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

type statusResp struct {
	Service    string   `json:"service"`
	APR        float64  `json:"apr"`
	LockDays   int      `json:"lock_days"`
	Assets     []string `json:"assets"`
	TVL        string   `json:"tvl"`
	Status     string   `json:"status"`
	ServerTime string   `json:"server_time"`
}

// Handler is the Vercel serverless entrypoint. Deployed at /api/status,
// rewritten to /v1/status by vercel.json so the CLI's default path works.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	resp := statusResp{
		Service:    "fred",
		APR:        8.0,
		LockDays:   365,
		Assets:     []string{"USDC", "USDT", "DAI", "OUSD"},
		TVL:        "$0",
		Status:     "operational",
		ServerTime: time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(resp)
}
