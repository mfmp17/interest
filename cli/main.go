package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// apiBase is where the CLI talks to. Overridable with INTEREST_API env var
// so you can point at localhost during dev and prod later.
func apiBase() string {
	if v := os.Getenv("INTEREST_API"); v != "" {
		return v
	}
	return "https://api.fred.cash"
}

// version is stamped at build time via -ldflags "-X main.version=..."
var version = "dev"

type statusResp struct {
	Service            string   `json:"service"`
	APR                float64  `json:"apr"`
	LockDays           int      `json:"lock_days"`
	Assets             []string `json:"assets"`
	TVL                string   `json:"tvl"`
	Status             string   `json:"status"`
	Network            string   `json:"network"`
	ChainID            int64    `json:"chain_id"`
	PrincipalLiability string   `json:"principal_liability"`
	ClaimableLiability string   `json:"claimable_liability"`
	TotalLiability     string   `json:"total_liability"`
	ReserveRatio       string   `json:"reserve_ratio"`
	Underfunded        bool     `json:"underfunded"`
	Shortfall          string   `json:"shortfall"`
	ScannerLatestBlock uint64   `json:"scanner_latest_block"`
	ScannerLastBlock   uint64   `json:"scanner_last_block"`
	ScannerLagBlocks   uint64   `json:"scanner_lag_blocks"`
	ScannerLastScanAt  string   `json:"scanner_last_scan_at"`
	ActivePositions    int      `json:"active_positions"`
	PendingPositions   int      `json:"pending_positions"`
	Treasury           string   `json:"treasury"`
	TreasuryETH        string   `json:"treasury_eth"`
	TreasuryUSDC       string   `json:"treasury_usdc"`
	TreasuryWarning    string   `json:"treasury_warning"`
	ServerTime         string   `json:"server_time"`
}

const (
	green = "\033[32m"
	cyan  = "\033[36m"
	bold  = "\033[1m"
	dim   = "\033[2m"
	reset = "\033[0m"
)

func main() {
	args := os.Args[1:]
	cmd := "connect"
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "connect", "":
		connect()
	case "status":
		printStatus()
	case "deposit", "lock":
		deposit()
	case "balance", "bal":
		balance()
	case "claim":
		claim()
	case "withdraw":
		withdraw()
	case "doctor":
		doctorCommand()
	case "support":
		supportCommand()
	case "positions":
		positionsCommand()
	case "use":
		if len(args) < 2 {
			fmt.Println("usage: fred.cash use <position-id>")
			os.Exit(1)
		}
		useCommand(args[1])
	case "receipt":
		receiptCommand(args[1:])
	case "update", "upgrade":
		updateCLI()
	case "version", "--version", "-v":
		fmt.Printf("fred.cash %s\n", version)
	case "help", "--help", "-h":
		help()
	default:
		fmt.Printf("unknown command: %s\n\n", cmd)
		help()
		os.Exit(1)
	}
}

func fetchStatus() (*statusResp, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(apiBase() + "/v1/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}
	var s statusResp
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func connect() {
	fmt.Printf("%sConnecting to Fred%s", dim, reset)
	for i := 0; i < 3; i++ {
		time.Sleep(180 * time.Millisecond)
		fmt.Print(".")
	}
	fmt.Println()

	s, err := fetchStatus()
	if err != nil {
		fmt.Printf("\n%s✗ Could not reach Fred.%s\n", "\033[31m", reset)
		fmt.Printf("%s  %v%s\n", dim, err, reset)
		os.Exit(1)
	}

	fmt.Printf("\n%s%s● Connected to Fred%s\n\n", green, bold, reset)
	fmt.Printf("  %sEarn%s   %s%.0f%% APR%s on stablecoins\n", dim, reset, bold, s.APR, reset)
	fmt.Printf("  %sAssets%s %s\n", dim, reset, join(s.Assets))
	fmt.Printf("  %sLock%s   %d days\n", dim, reset, s.LockDays)
	fmt.Printf("  %sTVL%s    %s\n", dim, reset, s.TVL)
	printTreasuryNotice(s)
	fmt.Printf("\n%sYou're in. Run %sfred.cash status%s%s anytime.%s\n", dim, reset+cyan, reset, dim, reset)
}

func printStatus() {
	s, err := fetchStatus()
	if err != nil {
		fmt.Printf("%s✗ offline: %v%s\n", "\033[31m", err, reset)
		os.Exit(1)
	}
	fmt.Printf("%s%s Fred %s\n", bold, s.Status, reset)
	fmt.Printf("  APR:    %.2f%%\n", s.APR)
	fmt.Printf("  Lock:   %d days\n", s.LockDays)
	fmt.Printf("  Assets: %s\n", join(s.Assets))
	fmt.Printf("  TVL:    %s\n", s.TVL)
	if s.TotalLiability != "" {
		fmt.Printf("  Owed:   %s USDC\n", s.TotalLiability)
		fmt.Printf("  Reserve: %s\n", valueOrUnknown(s.ReserveRatio))
	}
	if s.ScannerLatestBlock > 0 {
		fmt.Printf("  Scan:   block %d / %d · lag %d\n", s.ScannerLastBlock, s.ScannerLatestBlock, s.ScannerLagBlocks)
	}
	printTreasuryNotice(s)
	fmt.Printf("  Time:   %s\n", s.ServerTime)
}

func printTreasuryNotice(s *statusResp) {
	if s == nil {
		return
	}
	if s.TreasuryUSDC != "" || s.TreasuryETH != "" {
		fmt.Printf("  %sTreasury%s %s USDC · %s ETH\n", dim, reset, valueOrUnknown(s.TreasuryUSDC), valueOrUnknown(s.TreasuryETH))
	}
	if s.TreasuryWarning != "" {
		fmt.Printf("  %s⚠ %s%s\n", "\033[33m", s.TreasuryWarning, reset)
	}
}

func valueOrUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}

func join(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += ", "
		}
		out += x
	}
	return out
}

func help() {
	fmt.Printf(`%sfred.cash%s — plug into Fred yield

%sUsage:%s
  fred.cash            connect and show current yield
  fred.cash deposit    deposit & lock funds, choose your yield plan
  fred.cash balance    show your position, accrued interest, unlock date
  fred.cash claim      claim streamed interest (Classic 8%% plan)
  fred.cash withdraw   withdraw instant payout / unlocked principal
  fred.cash status     show live service status
  fred.cash doctor     diagnose API, scanner, receipt, and deposit state
  fred.cash support    print a redacted JSON support bundle
  fred.cash positions  list local positions
  fred.cash use <id>   switch the active position
  fred.cash receipt export [path]
  fred.cash receipt import <path>
  fred.cash update     update fred.cash to the latest release
  fred.cash version    print version
  fred.cash help       show this

%sLegacy:%s
  interest             still works as a compatibility alias

%sEnv:%s
  INTEREST_API         override API base (default https://api.fred.cash)
  INTEREST_FAST        skip the chain-watch animation in demos
`, bold, reset, bold, reset, bold, reset, bold, reset)
}
