package main

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	erc20TransferTopic = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	erc20BalanceOfSig  = crypto.Keccak256([]byte("balanceOf(address)"))[:4]
)

type ChainClient struct {
	rpc     string
	chainID *big.Int
	c       *ethclient.Client
}

const maxGetLogsBlockRange uint64 = 9_500

func NewChainClient(ctx context.Context, rpc string, chainID int64) (*ChainClient, error) {
	c, err := ethclient.DialContext(ctx, rpc)
	if err != nil {
		return nil, err
	}
	actual, err := c.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	want := big.NewInt(chainID)
	if chainID > 0 && actual.Cmp(want) != 0 {
		return nil, errors.New("rpc chain id mismatch")
	}
	return &ChainClient{rpc: rpc, chainID: actual, c: c}, nil
}

func (cc *ChainClient) Close() {
	if cc.c != nil {
		cc.c.Close()
	}
}
func (cc *ChainClient) ChainID() *big.Int { return new(big.Int).Set(cc.chainID) }

func (cc *ChainClient) LatestBlock(ctx context.Context) (uint64, error) {
	return cc.c.BlockNumber(ctx)
}

func (cc *ChainClient) NativeBalance(ctx context.Context, owner common.Address) (*big.Int, error) {
	return cc.c.BalanceAt(ctx, owner, nil)
}

func (cc *ChainClient) TransactionReceipt(ctx context.Context, tx common.Hash) (*types.Receipt, error) {
	return cc.c.TransactionReceipt(ctx, tx)
}

func (cc *ChainClient) ERC20Balance(ctx context.Context, token, owner common.Address) (*big.Int, error) {
	data := make([]byte, 4+32)
	copy(data[:4], erc20BalanceOfSig)
	copy(data[4+12:], owner.Bytes())
	out, err := cc.c.CallContract(ctx, ethereum.CallMsg{To: &token, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return big.NewInt(0), nil
	}
	return new(big.Int).SetBytes(out), nil
}

type InboundTransfer struct {
	TxHash common.Hash    `json:"tx_hash"`
	Token  common.Address `json:"token"`
	From   common.Address `json:"from"`
	To     common.Address `json:"to"`
	Amount *big.Int       `json:"amount"`
	Block  uint64         `json:"block"`
}

func (cc *ChainClient) InboundTransfers(ctx context.Context, token, to common.Address, fromBlock uint64) ([]*InboundTransfer, error) {
	latest, err := cc.LatestBlock(ctx)
	if err != nil {
		return nil, err
	}
	toTopic := common.BytesToHash(to.Bytes())
	logs, err := cc.filterTransferLogsChunked(ctx, fromBlock, latest, []common.Address{token}, [][]common.Hash{{erc20TransferTopic}, nil, {toTopic}})
	if err != nil {
		return nil, err
	}
	return parseTransferLogs(logs), nil
}

func (cc *ChainClient) InboundTransfersAnyToken(ctx context.Context, to common.Address, fromBlock uint64) ([]*InboundTransfer, error) {
	latest, err := cc.LatestBlock(ctx)
	if err != nil {
		return nil, err
	}
	return cc.InboundTransfersAnyTokenRange(ctx, to, fromBlock, latest)
}

func (cc *ChainClient) InboundTransfersAnyTokenRange(ctx context.Context, to common.Address, fromBlock, toBlock uint64) ([]*InboundTransfer, error) {
	toTopic := common.BytesToHash(to.Bytes())
	logs, err := cc.filterTransferLogsChunked(ctx, fromBlock, toBlock, nil, [][]common.Hash{{erc20TransferTopic}, nil, {toTopic}})
	if err != nil {
		return nil, err
	}
	return parseTransferLogs(logs), nil
}

func (cc *ChainClient) filterTransferLogsChunked(ctx context.Context, fromBlock, toBlock uint64, addresses []common.Address, topics [][]common.Hash) ([]types.Log, error) {
	var all []types.Log
	for _, chunk := range blockChunks(fromBlock, toBlock, maxGetLogsBlockRange) {
		q := ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(chunk[0]),
			ToBlock:   new(big.Int).SetUint64(chunk[1]),
			Addresses: addresses,
			Topics:    topics,
		}
		logs, err := cc.c.FilterLogs(ctx, q)
		if err != nil {
			return nil, err
		}
		all = append(all, logs...)
	}
	return all, nil
}

func blockChunks(fromBlock, toBlock, maxRange uint64) [][2]uint64 {
	if fromBlock > toBlock || maxRange == 0 {
		return nil
	}
	var chunks [][2]uint64
	for start := fromBlock; start <= toBlock; {
		end := start + maxRange - 1
		if end > toBlock || end < start {
			end = toBlock
		}
		chunks = append(chunks, [2]uint64{start, end})
		if end == toBlock {
			break
		}
		start = end + 1
	}
	return chunks
}

func parseTransferLogs(logs []types.Log) []*InboundTransfer {
	out := make([]*InboundTransfer, 0, len(logs))
	for _, lg := range logs {
		tr := parseTransferLog(lg)
		if tr != nil {
			out = append(out, tr)
		}
	}
	return out
}

func (cc *ChainClient) FindInboundTransfer(ctx context.Context, token, to common.Address, fromBlock uint64, minAmount *big.Int) (*InboundTransfer, error) {
	trs, err := cc.InboundTransfers(ctx, token, to, fromBlock)
	if err != nil {
		return nil, err
	}
	for _, tr := range trs {
		if tr.Amount.Cmp(minAmount) >= 0 {
			return tr, nil
		}
	}
	return nil, nil
}

func parseTransferLog(lg types.Log) *InboundTransfer {
	if len(lg.Topics) < 3 || len(lg.Data) != 32 {
		return nil
	}
	from := common.BytesToAddress(lg.Topics[1].Bytes()[12:])
	to := common.BytesToAddress(lg.Topics[2].Bytes()[12:])
	amt := new(big.Int).SetBytes(lg.Data)
	return &InboundTransfer{TxHash: lg.TxHash, Token: lg.Address, From: from, To: to, Amount: amt, Block: lg.BlockNumber}
}
