package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	ctx := context.Background()
	cfg := LoadConfig()
	var err error
	cfg, err = EnsureDevSecrets(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "print-config" {
		fmt.Printf("network=%s\n", cfg.Network)
		fmt.Printf("chain_id=%d\n", cfg.ChainID)
		fmt.Printf("lock_seconds=%d\n", cfg.LockSeconds)
		fmt.Printf("rpc=%s\n", cfg.RPCURL)
		fmt.Printf("state_path=%s\n", cfg.StatePath)
		fmt.Printf("treasury_address=%s\n", cfg.TreasuryAddr.Hex())
		fmt.Printf("usdc=%s\n", cfg.Assets["USDC"].Address.Hex())
		return
	}
	app, err := NewApp(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	fmt.Printf("fred backend on %s chain=%d lock=%ds\n", cfg.Network, cfg.ChainID, cfg.LockSeconds)
	fmt.Printf("treasury address: %s\n", cfg.TreasuryAddr.Hex())
	fmt.Printf("fund treasury with Base ETH for gas + USDC for payouts before accepting deposits\n")

	go func() {
		t := time.NewTicker(8 * time.Second)
		defer t.Stop()
		for range t.C {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			if err := app.ScanOnce(ctx); err != nil {
				log.Printf("scan: %v", err)
			}
			cancel()
		}
	}()

	s := &Server{app: app}
	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, s.routes()))
}
