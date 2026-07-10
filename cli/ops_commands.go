package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func doctorCommand() {
	fmt.Printf("\n%s%s Fred doctor %s\n\n", bold, cyan, reset)
	fmt.Printf("  CLI                  fred.cash %s\n", version)
	fmt.Printf("  API                  %s\n", apiBase())

	status, statusErr := fetchStatus()
	if statusErr != nil {
		fmt.Printf("  API status           %sFAIL%s · %v\n", "\033[31m", reset, statusErr)
	} else {
		fmt.Printf("  API status           %s%s%s\n", green, status.Status, reset)
		fmt.Printf("  Network              %s · chain %d\n", valueOrUnknown(status.Network), status.ChainID)
		fmt.Printf("  Treasury             %s USDC · %s ETH\n", valueOrUnknown(status.TreasuryUSDC), valueOrUnknown(status.TreasuryETH))
		fmt.Printf("  Liabilities          %s USDC\n", valueOrUnknown(status.TotalLiability))
		fmt.Printf("  Reserve ratio        %s\n", valueOrUnknown(status.ReserveRatio))
		fmt.Printf("  Scanner              block %d / %d · lag %d\n", status.ScannerLastBlock, status.ScannerLatestBlock, status.ScannerLagBlocks)
		if status.ScannerLastScanAt != "" {
			fmt.Printf("  Last scan            %s\n", status.ScannerLastScanAt)
		}
		if status.TreasuryWarning != "" {
			fmt.Printf("  %s⚠ %s%s\n", "\033[33m", status.TreasuryWarning, reset)
		}
	}

	r, ok := activeReceipt()
	if !ok {
		fmt.Printf("  Local receipt        none\n\n")
		return
	}
	fmt.Printf("  Local receipt        yes · %s\n", r.ID)
	fmt.Printf("  Deposit address      %s\n", r.DepositAddress)
	pos, err := fetchPosition(r)
	if err != nil {
		fmt.Printf("  Position API         %sFAIL%s · %v\n\n", "\033[31m", reset, err)
		return
	}
	fmt.Printf("  Position status      %s\n", pos.Position.Status)
	fmt.Printf("  Received / target    %s / %s %s\n", pos.PrincipalDisplay, pos.ExpectedDisplay, pos.Position.Asset)
	fmt.Printf("  Missing              %s %s\n", pos.MissingDisplay, pos.Position.Asset)
	fmt.Printf("  Deposit wallet       %s %s\n", valueOrUnknown(pos.DepositBalanceDisplay), pos.Position.Asset)
	fmt.Printf("  Position scan        block %d", pos.Position.LastScannedBlock)
	if pos.Position.LastScanAt != "" {
		fmt.Printf(" · %s", pos.Position.LastScanAt)
	}
	fmt.Println()
	if pos.Position.DepositTx != "" {
		fmt.Printf("  Deposit tx           %s\n", pos.Position.DepositTx)
	}
	if pos.Position.SweepTx != "" {
		fmt.Printf("  Sweep tx             %s\n", pos.Position.SweepTx)
	}
	fmt.Println()
}

func supportCommand() {
	bundle := BuildSupportBundle(version, apiBase(), nil, nil, nil)
	status, err := fetchStatus()
	if err != nil {
		bundle.Errors = append(bundle.Errors, "status: "+err.Error())
	} else {
		bundle = BuildSupportBundle(version, apiBase(), status, nil, nil)
	}
	if r, ok := activeReceipt(); ok {
		pos, err := fetchPosition(r)
		if err != nil {
			bundle.Errors = append(bundle.Errors, "position: "+err.Error())
			base := BuildSupportBundle(version, apiBase(), status, &r, nil)
			base.Errors = append(base.Errors, bundle.Errors...)
			bundle = base
		} else {
			base := BuildSupportBundle(version, apiBase(), status, &r, pos)
			base.Errors = append(base.Errors, bundle.Errors...)
			bundle = base
		}
	} else {
		bundle.Errors = append(bundle.Errors, "local receipt: none")
	}
	b, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		fmt.Printf("support bundle failed: %v\n", err)
		return
	}
	fmt.Println(string(b))
}

func positionsCommand() {
	st := loadReceipts()
	if len(st.Receipts) == 0 {
		fmt.Printf("\n  No positions. Run %sfred.cash deposit%s.\n\n", cyan, reset)
		return
	}
	fmt.Printf("\n%s%s Fred positions %s\n\n", bold, cyan, reset)
	for _, r := range st.Receipts {
		marker := " "
		if r.ID == st.Active {
			marker = "*"
		}
		fmt.Printf("  %s %-14s %-6s %-10s %s\n", marker, r.ID, r.Asset, planLabel(r.Plan), short(r.DepositAddress))
	}
	fmt.Printf("\n  Switch with %sfred.cash use <position-id>%s\n\n", cyan, reset)
}

func useCommand(query string) {
	st := loadReceipts()
	r, err := SelectReceipt(&st, query)
	if err != nil {
		fmt.Printf("\n%s✗ use failed:%s %v\n\n", "\033[31m", reset, err)
		return
	}
	if err := ExportReceiptStore(receiptPath(), st); err != nil {
		fmt.Printf("\n%s✗ could not save active position:%s %v\n\n", "\033[31m", reset, err)
		return
	}
	fmt.Printf("\n  %s✓ Active position%s %s · %s\n\n", green, reset, r.ID, short(r.DepositAddress))
}

func receiptCommand(args []string) {
	if len(args) == 0 {
		fmt.Printf("usage: fred.cash receipt export [path] | fred.cash receipt import <path>\n")
		return
	}
	switch args[0] {
	case "export":
		path := filepath.Join(os.Getenv("HOME"), ".interest", "receipts-export.json")
		if len(args) > 1 {
			path = args[1]
		}
		if err := ExportReceiptStore(path, loadReceipts()); err != nil {
			fmt.Printf("%s✗ receipt export failed:%s %v\n", "\033[31m", reset, err)
			return
		}
		fmt.Printf("%s✓ Receipt backup exported%s %s\n", green, reset, path)
		fmt.Printf("%s  Contains claim tokens. Keep this file private.%s\n", dim, reset)
	case "import":
		if len(args) < 2 {
			fmt.Println("usage: fred.cash receipt import <path>")
			return
		}
		incoming, err := ImportReceiptStore(args[1])
		if err != nil {
			fmt.Printf("%s✗ receipt import failed:%s %v\n", "\033[31m", reset, err)
			return
		}
		merged := MergeReceiptStores(loadReceipts(), incoming)
		if err := ExportReceiptStore(receiptPath(), merged); err != nil {
			fmt.Printf("%s✗ receipt import save failed:%s %v\n", "\033[31m", reset, err)
			return
		}
		fmt.Printf("%s✓ Imported %d receipt(s)%s\n", green, len(incoming.Receipts), reset)
	default:
		fmt.Printf("unknown receipt command: %s\n", args[0])
	}
}
