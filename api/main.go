package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	resp := statusResp{
		Service:    "fred",
		APR:        8.0,
		LockDays:   365,
		Assets:     []string{"USDC", "USDT", "DAI", "OUSD"},
		TVL:        "$0", // wire to real number later
		Status:     "operational",
		ServerTime: time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(resp)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/status", statusHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("fred api\n"))
	})

	log.Printf("fred api listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
