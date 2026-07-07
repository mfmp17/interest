package main

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type DerivedWallet struct {
	Index      uint64
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
}

// DeriveDepositAddress deterministically derives a real Ethereum EOA from a
// master seed and index. This is not a mock: the returned key can sign real txs.
// Production seed must come from a secret manager / env var, never git.
func DeriveDepositAddress(masterSeed []byte, index uint64) (*DerivedWallet, error) {
	if len(masterSeed) < 32 {
		return nil, fmt.Errorf("master seed must be at least 32 bytes")
	}

	curveN := crypto.S256().Params().N
	var priv *ecdsa.PrivateKey
	var err error

	for attempt := uint32(0); attempt < 100; attempt++ {
		mac := hmac.New(sha256.New, masterSeed)
		var buf [12]byte
		binary.BigEndian.PutUint64(buf[:8], index)
		binary.BigEndian.PutUint32(buf[8:], attempt)
		mac.Write([]byte("fred-deposit-address-v1"))
		mac.Write(buf[:])
		d := mac.Sum(nil)

		k := new(big.Int).SetBytes(d)
		if k.Sign() == 0 || k.Cmp(curveN) >= 0 {
			continue
		}
		priv, err = crypto.ToECDSA(d)
		if err == nil {
			return &DerivedWallet{
				Index:      index,
				PrivateKey: priv,
				Address:    crypto.PubkeyToAddress(priv.PublicKey),
			}, nil
		}
	}
	return nil, fmt.Errorf("could not derive valid private key for index %d", index)
}
