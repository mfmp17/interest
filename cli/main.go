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
	Service    string   `json:"service"`
	APR        float64  `json:"apr"`
	LockDays   int      `json:"lock_days"`
	Assets     []string `json:"assets"`
	TVL        string   `json:"tvl"`
	Status     string   `json:"status"`
	ServerTime string   `json:"server_time"`
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
	case "version", "--version", "-v":
		fmt.Printf("interest %s\n", version)
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
	fmt.Printf("\n%sYou're in. Run %sinterest status%s%s anytime.%s\n", dim, reset+cyan, reset, dim, reset)
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
	fmt.Printf("  Time:   %s\n", s.ServerTime)
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
	fmt.Printf(`%sinterest%s — plug into Fred yield

%sUsage:%s
  interest            connect and show current yield
  interest deposit    deposit & lock funds, choose your yield plan
  interest balance    show your position, accrued interest, unlock date
  interest claim      claim streamed interest (Classic 8%% plan)
  interest withdraw   withdraw instant payout / unlocked principal
  interest status     show live service status
  interest version    print version
  interest help       show this

%sEnv:%s
  INTEREST_API        override API base (default https://api.fred.cash)
  INTEREST_FAST       skip the chain-watch animation in demos
`, bold, reset, bold, reset, bold, reset)
}
