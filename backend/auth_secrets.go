package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func NewToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func CheckToken(hash, token string) bool { return HashToken(token) == hash }

func EnsureDevSecrets(cfg Config) (Config, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".fred", cfg.Network)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return cfg, err
	}
	if len(cfg.MasterSeed) == 0 {
		path := filepath.Join(dir, "master-seed.hex")
		b, err := os.ReadFile(path)
		if err == nil {
			seed, _ := hex.DecodeString(strings.TrimSpace(string(b)))
			cfg.MasterSeed = seed
		} else {
			seed := make([]byte, 32)
			if _, err := rand.Read(seed); err != nil {
				return cfg, err
			}
			cfg.MasterSeed = seed
			if err := os.WriteFile(path, []byte(hex.EncodeToString(seed)+"\n"), 0o600); err != nil {
				return cfg, err
			}
			fmt.Printf("created %s master seed: %s\n", cfg.Network, path)
		}
	}
	if cfg.TreasuryKey == "" {
		path := filepath.Join(dir, "treasury.key")
		b, err := os.ReadFile(path)
		if err == nil {
			cfg.TreasuryKey = strings.TrimSpace(string(b))
		} else {
			pk, err := crypto.GenerateKey()
			if err != nil {
				return cfg, err
			}
			cfg.TreasuryKey = hex.EncodeToString(crypto.FromECDSA(pk))
			if err := os.WriteFile(path, []byte(cfg.TreasuryKey+"\n"), 0o600); err != nil {
				return cfg, err
			}
			fmt.Printf("created %s treasury key: %s\n", cfg.Network, path)
		}
	}
	pk, err := PrivateKeyFromHex(cfg.TreasuryKey)
	if err == nil {
		cfg.TreasuryAddr = AddressFromPrivateKey(pk)
	}
	return cfg, nil
}
