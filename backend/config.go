package main

import (
	"encoding/hex"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

type AssetConfig struct {
	Symbol   string         `json:"symbol"`
	Address  common.Address `json:"address"`
	Decimals uint8          `json:"decimals"`
}

type Config struct {
	Network       string
	RPCURL        string
	ChainID       int64
	LockSeconds   int64
	Port          string
	StatePath     string
	MasterSeed    []byte
	TreasuryKey   string
	TreasuryAddr  common.Address
	AdminToken    string
	Confirmations uint64
	Assets        map[string]AssetConfig
}

func LoadConfig() Config {
	// Mainnet defaults. For a short mainnet canary, set FRED_LOCK_SECONDS=365.
	lock := envInt64("FRED_LOCK_SECONDS", 31536000) // 365 days
	cfg := Config{
		Network:       env("FRED_NETWORK", "base-mainnet"),
		RPCURL:        env("FRED_RPC_URL", "https://mainnet.base.org"),
		ChainID:       envInt64("FRED_CHAIN_ID", 8453),
		LockSeconds:   lock,
		Port:          env("PORT", "8090"),
		StatePath:     env("FRED_STATE_PATH", os.Getenv("HOME")+"/.fred/mainnet-state.json"),
		Confirmations: uint64(envInt64("FRED_CONFIRMATIONS", 4)),
		Assets: map[string]AssetConfig{
			// Base mainnet native USDC. Start USDC-only; enable other tokens only after verifying official Base contracts.
			"USDC": {Symbol: "USDC", Address: common.HexToAddress(env("FRED_USDC_ADDRESS", "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")), Decimals: 6},
			"USDT": {Symbol: "USDT", Address: common.HexToAddress(env("FRED_USDT_ADDRESS", "0x0000000000000000000000000000000000000000")), Decimals: 6},
			"DAI":  {Symbol: "DAI", Address: common.HexToAddress(env("FRED_DAI_ADDRESS", "0x0000000000000000000000000000000000000000")), Decimals: 18},
			"OUSD": {Symbol: "OUSD", Address: common.HexToAddress(env("FRED_OUSD_ADDRESS", "0x0000000000000000000000000000000000000000")), Decimals: 18},
		},
	}
	if hexSeed := os.Getenv("FRED_MASTER_SEED_HEX"); hexSeed != "" {
		if b, err := hex.DecodeString(hexSeed); err == nil {
			cfg.MasterSeed = b
		}
	}
	cfg.TreasuryKey = os.Getenv("FRED_TREASURY_PRIVATE_KEY")
	cfg.AdminToken = os.Getenv("FRED_ADMIN_TOKEN")
	return cfg
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envInt64(k string, d int64) int64 {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return d
}
