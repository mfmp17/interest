package main

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var erc20TransferSig = crypto.Keccak256([]byte("transfer(address,uint256)"))[:4]

func erc20TransferData(to common.Address, amount *big.Int) []byte {
	data := make([]byte, 4+32+32)
	copy(data[:4], erc20TransferSig)
	copy(data[4+12:4+32], to.Bytes())
	amt := amount.Bytes()
	copy(data[68-len(amt):], amt)
	return data
}

func PrivateKeyFromHex(hexkey string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(trim0x(hexkey))
}

func AddressFromPrivateKey(pk *ecdsa.PrivateKey) common.Address {
	return crypto.PubkeyToAddress(pk.PublicKey)
}

func trim0x(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}

func (cc *ChainClient) SendETH(ctx context.Context, pk *ecdsa.PrivateKey, to common.Address, amountWei *big.Int) (common.Hash, error) {
	from := AddressFromPrivateKey(pk)
	nonce, err := cc.c.PendingNonceAt(ctx, from)
	if err != nil {
		return common.Hash{}, err
	}
	tip, err := cc.c.SuggestGasTipCap(ctx)
	if err != nil {
		tip = big.NewInt(1_000_000_000)
	}
	head, err := cc.c.HeaderByNumber(ctx, nil)
	if err != nil {
		return common.Hash{}, err
	}
	feeCap := new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), tip)
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: cc.chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: feeCap, Gas: 21000, To: &to, Value: amountWei})
	signed, err := types.SignTx(tx, types.LatestSignerForChainID(cc.chainID), pk)
	if err != nil {
		return common.Hash{}, err
	}
	return signed.Hash(), cc.c.SendTransaction(ctx, signed)
}

func (cc *ChainClient) SendERC20(ctx context.Context, pk *ecdsa.PrivateKey, token, to common.Address, amount *big.Int) (common.Hash, error) {
	from := AddressFromPrivateKey(pk)
	data := erc20TransferData(to, amount)
	nonce, err := cc.c.PendingNonceAt(ctx, from)
	if err != nil {
		return common.Hash{}, err
	}
	tip, err := cc.c.SuggestGasTipCap(ctx)
	if err != nil {
		tip = big.NewInt(1_000_000_000)
	}
	head, err := cc.c.HeaderByNumber(ctx, nil)
	if err != nil {
		return common.Hash{}, err
	}
	feeCap := new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), tip)
	gas, err := cc.c.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &token, Data: data})
	if err != nil {
		return common.Hash{}, err
	}
	tx := types.NewTx(&types.DynamicFeeTx{ChainID: cc.chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: feeCap, Gas: gas + 10000, To: &token, Value: big.NewInt(0), Data: data})
	signed, err := types.SignTx(tx, types.LatestSignerForChainID(cc.chainID), pk)
	if err != nil {
		return common.Hash{}, err
	}
	return signed.Hash(), cc.c.SendTransaction(ctx, signed)
}
