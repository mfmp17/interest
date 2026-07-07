package main

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestERC20TransferCalldata(t *testing.T) {
	to := common.HexToAddress("0x1111111111111111111111111111111111111111")
	data := erc20TransferData(to, big.NewInt(123))
	if len(data) != 68 {
		t.Fatalf("len=%d want 68", len(data))
	}
	if common.Bytes2Hex(data[:4]) != "a9059cbb" {
		t.Fatalf("bad selector %s", common.Bytes2Hex(data[:4]))
	}
	if common.BytesToAddress(data[4+12:4+32]) != to {
		t.Fatal("to address not encoded")
	}
	if new(big.Int).SetBytes(data[36:68]).Int64() != 123 {
		t.Fatal("amount not encoded")
	}
}
