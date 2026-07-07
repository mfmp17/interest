package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	qrterminal "github.com/mdp/qrterminal/v3"
)

type receipt struct {
	ID             string `json:"id"`
	Token          string `json:"token"`
	Asset          string `json:"asset"`
	Plan           string `json:"plan"`
	DepositAddress string `json:"deposit_address"`
	Network        string `json:"network"`
	ChainID        int64  `json:"chain_id"`
	CreatedAt      string `json:"created_at"`
}

type receiptStore struct {
	Active   string    `json:"active"`
	Receipts []receipt `json:"receipts"`
}

type backendPosition struct {
	ID               string `json:"id"`
	Asset            string `json:"asset"`
	Decimals         uint8  `json:"decimals"`
	Principal        string `json:"principal"`
	Plan             string `json:"plan"`
	Status           string `json:"status"`
	DepositAddress   string `json:"deposit_address"`
	DepositorAddress string `json:"depositor_address"`
	CreatedAt        string `json:"created_at"`
	ConfirmedAt      string `json:"confirmed_at"`
	UnlockAt         string `json:"unlock_at"`
	LockSeconds      int64  `json:"lock_seconds"`
	InterestPaid     string `json:"interest_paid"`
	InstantPaid      string `json:"instant_paid"`
	PrincipalPaid    bool   `json:"principal_paid"`
	DepositTx        string `json:"deposit_tx"`
	GasTx            string `json:"gas_tx"`
	SweepTx          string `json:"sweep_tx"`
}

type createDepositResp struct {
	Position   backendPosition `json:"position"`
	ClaimToken string          `json:"claim_token"`
	DepositURI string          `json:"deposit_uri"`
	Network    string          `json:"network"`
	ChainID    int64           `json:"chain_id"`
}

type positionResp struct {
	Position         backendPosition `json:"position"`
	Claimable        string          `json:"claimable"`
	ClaimableDisplay string          `json:"claimable_display"`
	PrincipalDisplay string          `json:"principal_display"`
}

type txResp struct {
	TxHash string `json:"tx_hash"`
	Amount string `json:"amount"`
	Asset  string `json:"asset"`
}

func deposit() {
	if os.Getenv("INTEREST_LOCAL") != "" {
		localDeposit()
		return
	}
	backendDeposit()
}

func balance() {
	if os.Getenv("INTEREST_LOCAL") != "" {
		localBalance()
		return
	}
	backendBalance()
}

func claim() {
	if os.Getenv("INTEREST_LOCAL") != "" {
		localClaim()
		return
	}
	backendClaim()
}

func withdraw() {
	if os.Getenv("INTEREST_LOCAL") != "" {
		localWithdraw()
		return
	}
	backendWithdraw()
}

func backendDeposit() {
	fmt.Printf("\n%s%s Fred mainnet deposit %s\n\n", bold, cyan, reset)
	amt := prompt("  Amount USDC to deposit: ")
	if strings.TrimSpace(amt) == "" {
		fmt.Println("  Cancelled.")
		return
	}

	fmt.Printf("\n  %sChoose your yield:%s\n\n", bold, reset)
	fmt.Printf("   %s[1] Classic 8%%%s  ·  stream over 365 days\n", green+bold, reset)
	fmt.Printf("   %s[2] Instant 5%%%s  ·  cash now, principal locked\n", cyan+bold, reset)
	choice := prompt("  Select [1/2]: ")
	plan := planClassic
	if strings.TrimSpace(choice) == "2" {
		plan = planInstant
	}

	var out createDepositResp
	err := apiJSON("POST", "/v1/deposits", map[string]string{"amount": amt, "asset": "USDC", "plan": plan}, &out)
	if err != nil {
		fmt.Printf("\n%s✗ backend deposit failed:%s %v\n", "\033[31m", reset, err)
		fmt.Printf("%s  If testing locally, run: INTEREST_API=http://localhost:8090 interest deposit%s\n\n", dim, reset)
		os.Exit(1)
	}

	saveReceipt(receipt{ID: out.Position.ID, Token: out.ClaimToken, Asset: out.Position.Asset, Plan: out.Position.Plan, DepositAddress: out.Position.DepositAddress, Network: out.Network, ChainID: out.ChainID, CreatedAt: time.Now().UTC().Format(time.RFC3339)})

	fmt.Printf("\n  %sSend exactly %s USDC on Base mainnet to:%s\n\n", bold, amt, reset)
	qrterminal.GenerateHalfBlock(out.DepositURI, qrterminal.L, os.Stdout)
	fmt.Printf("\n  %s%s%s\n", cyan+bold, out.Position.DepositAddress, reset)
	fmt.Printf("\n  %sReceipt saved locally. Do not delete ~/.interest/receipts.json%s\n", dim, reset)
	fmt.Printf("  Run %sinterest balance%s after sending to watch confirmation.\n\n", cyan, reset)
}

func backendBalance() {
	r, ok := activeReceipt()
	if !ok {
		fmt.Printf("\n  No receipt found. Run %sinterest deposit%s first.\n\n", cyan, reset)
		return
	}
	pos, err := fetchPosition(r)
	if err != nil {
		fmt.Printf("\n%s✗ balance failed:%s %v\n\n", "\033[31m", reset, err)
		os.Exit(1)
	}
	p := pos.Position
	fmt.Printf("\n  %s%s● Fred position%s\n", green, bold, reset)
	fmt.Printf("    Status          %s\n", p.Status)
	fmt.Printf("    Principal       %s %s\n", pos.PrincipalDisplay, p.Asset)
	fmt.Printf("    Plan            %s\n", planLabel(p.Plan))
	fmt.Printf("    Claimable       %s %s\n", pos.ClaimableDisplay, p.Asset)
	fmt.Printf("    Deposit address %s\n", short(p.DepositAddress))
	if p.DepositTx != "" {
		fmt.Printf("    Deposit tx      %s\n", short(p.DepositTx))
	}
	if p.SweepTx != "" {
		fmt.Printf("    Sweep tx        %s\n", short(p.SweepTx))
	}
	if p.UnlockAt != "" {
		fmt.Printf("    Unlocks         %s\n", p.UnlockAt)
	}
	fmt.Println()
}

func backendClaim() {
	r, ok := activeReceipt()
	if !ok {
		fmt.Printf("\n  No receipt found.\n\n")
		return
	}
	pos, err := fetchPosition(r)
	if err != nil {
		fmt.Printf("\n%s✗ claim failed:%s %v\n\n", "\033[31m", reset, err)
		os.Exit(1)
	}
	if pos.Position.Plan != planClassic {
		fmt.Printf("\n  Claim is for Classic 8%%. Use %sinterest withdraw%s for Instant 5%%.\n\n", cyan, reset)
		return
	}
	to := payoutAddress(pos.Position)
	if to == "" {
		return
	}
	var out txResp
	if err := apiJSON("POST", "/v1/positions/"+r.ID+"/claim", map[string]string{"token": r.Token, "to": to}, &out); err != nil {
		fmt.Printf("\n%s✗ claim failed:%s %v\n\n", "\033[31m", reset, err)
		os.Exit(1)
	}
	fmt.Printf("\n  %s✓ Claim sent%s tx: %s\n\n", green, reset, out.TxHash)
}

func backendWithdraw() {
	r, ok := activeReceipt()
	if !ok {
		fmt.Printf("\n  No receipt found.\n\n")
		return
	}
	pos, err := fetchPosition(r)
	if err != nil {
		fmt.Printf("\n%s✗ withdraw failed:%s %v\n\n", "\033[31m", reset, err)
		os.Exit(1)
	}
	to := payoutAddress(pos.Position)
	if to == "" {
		return
	}
	if pos.Position.Plan == planInstant && pos.Claimable != "0" {
		var out txResp
		if err := apiJSON("POST", "/v1/positions/"+r.ID+"/claim", map[string]string{"token": r.Token, "to": to}, &out); err != nil {
			fmt.Printf("\n%s✗ instant payout failed:%s %v\n", "\033[31m", reset, err)
		} else {
			fmt.Printf("\n  %s✓ Instant payout sent%s tx: %s\n", green, reset, out.TxHash)
		}
	}
	var out txResp
	if err := apiJSON("POST", "/v1/positions/"+r.ID+"/withdraw", map[string]string{"token": r.Token, "to": to}, &out); err != nil {
		fmt.Printf("\n  Principal not withdrawn: %v\n\n", err)
		return
	}
	fmt.Printf("\n  %s✓ Principal withdrawn%s tx: %s\n\n", green, reset, out.TxHash)
}

func apiJSON(method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(apiBase(), "/")+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("content-type", "application/json")
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out != nil {
		return json.Unmarshal(b, out)
	}
	return nil
}

func fetchPosition(r receipt) (*positionResp, error) {
	var out positionResp
	err := apiJSON("GET", "/v1/positions/"+r.ID+"?token="+r.Token, nil, &out)
	return &out, err
}

func receiptPath() string {
	dir := filepath.Join(os.Getenv("HOME"), ".interest")
	_ = os.MkdirAll(dir, 0o700)
	return filepath.Join(dir, "receipts.json")
}

func loadReceipts() receiptStore {
	var st receiptStore
	b, err := os.ReadFile(receiptPath())
	if err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return st
}

func saveReceipt(r receipt) {
	st := loadReceipts()
	st.Active = r.ID
	st.Receipts = append(st.Receipts, r)
	b, _ := json.MarshalIndent(st, "", "  ")
	_ = os.WriteFile(receiptPath(), b, 0o600)
}

func activeReceipt() (receipt, bool) {
	st := loadReceipts()
	for _, r := range st.Receipts {
		if r.ID == st.Active {
			return r, true
		}
	}
	if len(st.Receipts) > 0 {
		return st.Receipts[len(st.Receipts)-1], true
	}
	return receipt{}, false
}

func payoutAddress(p backendPosition) string {
	def := p.DepositorAddress
	if def == "" {
		def = p.DepositAddress
	}
	to := prompt(fmt.Sprintf("  Withdraw to address (enter = %s): ", short(def)))
	if to == "" {
		to = def
	}
	if strings.ToLower(prompt("  Confirm send to "+short(to)+"? [y/N] ")) != "y" {
		fmt.Println("  Cancelled.")
		return ""
	}
	return to
}

func planLabel(p string) string {
	if p == planInstant {
		return "Instant 5%"
	}
	return "Classic 8%"
}
