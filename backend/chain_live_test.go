package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestLiveBaseMainnetRead(t *testing.T) {
	if os.Getenv("LIVE_CHAIN") == "" {
		t.Skip("set LIVE_CHAIN=1")
	}
	cfg := LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cc, err := NewChainClient(ctx, cfg.RPCURL, cfg.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	defer cc.Close()
	if cc.ChainID().Int64() != cfg.ChainID {
		t.Fatalf("chain id=%s want %d", cc.ChainID(), cfg.ChainID)
	}
	blk, err := cc.LatestBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if blk == 0 {
		t.Fatal("latest block is zero")
	}
	bal, err := cc.ERC20Balance(ctx, cfg.Assets["USDC"].Address, common.HexToAddress("0x0000000000000000000000000000000000000000"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s block=%d zero-usdc-balance=%s", cfg.Network, blk, bal.String())
}
