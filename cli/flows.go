package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	qrterminal "github.com/mdp/qrterminal/v3"
)

var stdin = bufio.NewReader(os.Stdin)

func prompt(label string) string {
	fmt.Print(label)
	line, _ := stdin.ReadString('\n')
	return strings.TrimSpace(line)
}

func assetName(n int) string {
	switch n {
	case 1:
		return "USDC"
	case 2:
		return "USDT"
	case 3:
		return "DAI"
	case 4:
		return "OUSD"
	}
	return "USDC"
}

// ---- deposit ----------------------------------------------------------------
func localDeposit() {
	fmt.Printf("\n%s%s Fred deposit %s\n\n", bold, cyan, reset)

	// amount
	amtStr := prompt("  How much would you like to deposit? ")
	amt, err := strconv.ParseFloat(strings.TrimPrefix(amtStr, "$"), 64)
	if err != nil || amt <= 0 {
		fmt.Printf("%s  Invalid amount.%s\n", "\033[31m", reset)
		os.Exit(1)
	}

	// asset
	fmt.Printf("\n  Asset?  %s[1]%s USDC   %s[2]%s USDT   %s[3]%s DAI   %s[4]%s OUSD\n", bold, reset, bold, reset, bold, reset, bold, reset)
	aStr := prompt("  Select [1-4]: ")
	aNum, _ := strconv.Atoi(aStr)
	if aNum < 1 || aNum > 4 {
		aNum = 1
	}
	asset := assetName(aNum)

	now := time.Now()
	unlock := now.AddDate(0, 0, lockDays)

	// summary
	fmt.Printf("\n  %s┌─ Deposit summary ─────────────────────────────┐%s\n", dim, reset)
	fmt.Printf("  %s│%s  Deposit   %s%.2f %s%s\n", dim, reset, bold, amt, asset, reset)
	fmt.Printf("  %s│%s  Network   Base\n", dim, reset)
	fmt.Printf("  %s│%s  Lock      %d days  (unlocks %s)\n", dim, reset, lockDays, unlock.Format("2006-01-02"))
	fmt.Printf("  %s└───────────────────────────────────────────────┘%s\n", dim, reset)

	// the two yield options, with live math
	classicTotal := amt * 0.08
	monthly := classicTotal / 12
	instantNow := amt * 0.05

	fmt.Printf("\n  %sChoose your yield:%s\n\n", bold, reset)

	fmt.Printf("   %s[1] Classic 8%%%s  ·  stream monthly\n", green+bold, reset)
	fmt.Printf("       +%.2f %s over the year  →  %.2f %s / month\n", classicTotal, asset, monthly, asset)
	fmt.Printf("       %sclaim interest anytime · principal unlocks in %dd%s\n", dim, lockDays, reset)
	fmt.Printf("       %s── you end with %.2f %s%s\n\n", dim, amt+classicTotal, asset, reset)

	fmt.Printf("   %s[2] Instant 5%%%s  ·  cash today\n", cyan+bold, reset)
	fmt.Printf("       +%.2f %s right now (withdraw immediately)\n", instantNow, asset)
	fmt.Printf("       %sprincipal locked %dd, no further yield%s\n", dim, lockDays, reset)
	fmt.Printf("       %s── %.2f now, %.2f back in %dd%s\n\n", dim, instantNow, amt, lockDays, reset)

	plan := planClassic
	pStr := prompt("  Select [1/2]: ")
	if pStr == "2" {
		plan = planInstant
	}

	// build position + deposit address
	id := fmt.Sprintf("%d", now.UnixNano())
	addr := mockAddress(id)
	pos := Position{
		ID:          id,
		Asset:       asset,
		Principal:   amt,
		Plan:        plan,
		DepositAddr: addr,
		CreatedAt:   now,
		UnlockAt:    unlock,
	}
	if plan == planInstant {
		pos.InstantPaid = instantNow
	}

	// show QR + address
	fmt.Printf("\n  Send exactly %s%.2f %s%s (Base) to:\n\n", bold, amt, asset, reset)
	renderQR(eip681(asset, addr, amt))
	fmt.Printf("\n    %s%s%s\n", bold, addr, reset)
	fmt.Printf("    %s(scan with any wallet, or copy the address)%s\n\n", dim, reset)

	// simulate the on-chain watcher
	fmt.Printf("  %s⏳ Watching Base for your deposit…%s\n", dim, reset)
	watchDemo(amt, asset)

	pos.Confirmed = true
	st := loadState()
	st.Positions = append(st.Positions, pos)
	st.save()

	fmt.Printf("\n  %s%s● Locked!%s %.2f %s secured. Unlocks %s.\n", green, bold, reset, amt, asset, unlock.Format("2006-01-02"))
	if plan == planInstant {
		fmt.Printf("  Your %s%.2f %s instant payout%s is available now — run %sfred.cash withdraw%s.\n", bold, instantNow, asset, reset, cyan, reset)
	} else {
		fmt.Printf("  Interest streams monthly — run %sfred.cash balance%s to track it.\n", cyan, reset)
	}
}

// eip681 builds an EIP-681 payment URI so good wallets pre-fill token+amount.
func eip681(asset, addr string, amt float64) string {
	// In production: ethereum:<tokenContract>/transfer?address=<to>&uint256=<amt*1e6>
	// For the demo we encode a readable URI carrying the same intent.
	units := int64(amt * 1e6) // 6-decimal stables
	return fmt.Sprintf("ethereum:%s@8453/transfer?address=%s&uint256=%d&asset=%s", addr, addr, units, asset)
}

func renderQR(data string) {
	cfg := qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     os.Stdout,
		HalfBlocks: true,
		BlackChar:  qrterminal.BLACK_BLACK,
		WhiteChar:  qrterminal.WHITE_WHITE,
		QuietZone:  1,
	}
	qrterminal.GenerateWithConfig(data, cfg)
}

// watchDemo simulates a chain watcher confirming the deposit.
func watchDemo(amt float64, asset string) {
	if os.Getenv("INTEREST_FAST") != "" {
		fmt.Printf("     %s✓ Seen %.2f %s — confirmed (demo fast mode)%s\n", green, amt, asset, reset)
		return
	}
	steps := []string{
		fmt.Sprintf("     %s✓ Seen %.2f %s in mempool…%s", dim, amt, asset, reset),
		fmt.Sprintf("     %s✓ 3/12 confirmations…%s", dim, reset),
		fmt.Sprintf("     %s✓ 9/12 confirmations…%s", dim, reset),
		fmt.Sprintf("     %s✓ 12/12 confirmed%s", green, reset),
	}
	for _, s := range steps {
		time.Sleep(700 * time.Millisecond)
		fmt.Println(s)
	}
}

// ---- balance ----------------------------------------------------------------
func localBalance() {
	st := loadState()
	pos := st.active()
	if pos == nil {
		fmt.Printf("\n  No active position. Run %sfred.cash deposit%s to start.\n\n", cyan, reset)
		return
	}
	now := time.Now()
	fmt.Printf("\n  %s%s● Your position%s\n", green, bold, reset)
	fmt.Printf("    Principal      %s%.2f %s%s   (locked · %d days left)\n", bold, pos.Principal, pos.Asset, reset, pos.daysLeft(now))
	planLabel := "Classic 8%"
	if pos.Plan == planInstant {
		planLabel = "Instant 5%"
	}
	fmt.Printf("    Plan           %s\n", planLabel)
	if pos.Plan == planClassic {
		fmt.Printf("    Interest paid  %.2f %s\n", pos.InterestPaid, pos.Asset)
		fmt.Printf("    Claimable now  %s%.2f %s%s   ← run %sfred.cash claim%s\n", bold, pos.claimable(now), pos.Asset, reset, cyan, reset)
	} else {
		fmt.Printf("    Instant payout %s%.2f %s%s   %s(withdraw anytime)%s\n", bold, pos.claimable(now), pos.Asset, reset, dim, reset)
	}
	fmt.Printf("    Unlocks        %s\n\n", pos.UnlockAt.Format("2006-01-02"))
}

// ---- claim (classic monthly interest) --------------------------------------
func localClaim() {
	st := loadState()
	pos := st.active()
	if pos == nil {
		fmt.Printf("\n  No active position.\n\n")
		return
	}
	now := time.Now()
	if pos.Plan != planClassic {
		fmt.Printf("\n  Claim is for the Classic 8%% plan. Use %sfred.cash withdraw%s for your instant payout.\n\n", cyan, reset)
		return
	}
	c := pos.claimable(now)
	if c < 0.01 {
		fmt.Printf("\n  Nothing to claim yet — interest is still accruing. Check %sfred.cash balance%s.\n\n", cyan, reset)
		return
	}
	fmt.Printf("\n  Claimable: %s%.2f %s%s\n", bold, c, pos.Asset, reset)
	to := prompt("  Withdraw to address (enter = your deposit wallet): ")
	if to == "" {
		to = pos.DepositAddr
	}
	if strings.ToLower(prompt(fmt.Sprintf("  Send %.2f %s to %s? [y/N] ", c, pos.Asset, short(to)))) != "y" {
		fmt.Println("  Cancelled.")
		return
	}
	pos.InterestPaid += c
	st.save()
	fmt.Printf("  %s✓ Sent %.2f %s%s   tx: %s\n\n", green, c, pos.Asset, reset, mockAddress("tx" + pos.ID + to)[:18]+"…")
}

// ---- withdraw ---------------------------------------------------------------
func localWithdraw() {
	st := loadState()
	pos := st.active()
	if pos == nil {
		fmt.Printf("\n  No active position.\n\n")
		return
	}
	now := time.Now()

	// instant plan: the 5% is withdrawable now; principal still locked
	if pos.Plan == planInstant && pos.InstantPaid > 0 {
		fmt.Printf("\n  Instant payout available: %s%.2f %s%s\n", bold, pos.InstantPaid, pos.Asset, reset)
		if strings.ToLower(prompt("  Withdraw it now? [y/N] ")) == "y" {
			amtPaid := pos.InstantPaid
			pos.InstantPaid = 0
			st.save()
			fmt.Printf("  %s✓ Sent %.2f %s%s to %s\n", green, amtPaid, pos.Asset, reset, short(pos.DepositAddr))
		}
	}

	// principal
	if !pos.unlocked(now) {
		fmt.Printf("\n  %sPrincipal %.2f %s is locked for %d more days%s (unlocks %s).\n\n",
			dim, pos.Principal, pos.Asset, pos.daysLeft(now), reset, pos.UnlockAt.Format("2006-01-02"))
		return
	}
	fmt.Printf("\n  %sPrincipal unlocked!%s Withdraw %s%.2f %s%s? [y/N] ", green, reset, bold, pos.Principal, pos.Asset, reset)
	if strings.ToLower(prompt("")) == "y" {
		fmt.Printf("  %s✓ Sent %.2f %s%s to %s. Position closed.\n\n", green, pos.Principal, pos.Asset, reset, short(pos.DepositAddr))
		// remove position
		out := st.Positions[:0]
		for _, p := range st.Positions {
			if p.ID != pos.ID {
				out = append(out, p)
			}
		}
		st.Positions = out
		st.save()
	}
}

func short(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:6] + "…" + s[len(s)-4:]
}
