package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const releaseRepo = "mfmp17/interest"

func updateCLI() {
	assetArch, err := releaseArch()
	if err != nil {
		fmt.Printf("%s✗ update failed:%s %v\n", "\033[31m", reset, err)
		os.Exit(1)
	}

	target, legacy, err := updateTargets()
	if err != nil {
		fmt.Printf("%s✗ update failed:%s %v\n", "\033[31m", reset, err)
		os.Exit(1)
	}

	url := fmt.Sprintf("https://github.com/%s/releases/latest/download/fred.cash_darwin_%s", releaseRepo, assetArch)
	fmt.Printf("%sUpdating fred.cash%s\n", dim, reset)
	fmt.Printf("  Current: %s\n", version)
	fmt.Printf("  Target:  %s\n", target)
	fmt.Printf("  Asset:   %s\n", filepath.Base(url))

	tmp, err := downloadUpdate(url)
	if err != nil {
		fmt.Printf("%s✗ download failed:%s %v\n", "\033[31m", reset, err)
		os.Exit(1)
	}
	defer os.Remove(tmp)

	if err := installUpdate(tmp, target, legacy); err != nil {
		fmt.Printf("\n%s✗ update failed:%s %v\n", "\033[31m", reset, err)
		fmt.Printf("\nRun the installer instead:\n  %scurl -fsSL https://get.fred.cash | bash%s\n", cyan, reset)
		os.Exit(1)
	}

	updatedVersion := versionFromBinary(target)
	if updatedVersion == "" {
		updatedVersion = "installed"
	}
	fmt.Printf("\n%s✓ fred.cash updated%s\n", green, reset)
	fmt.Printf("  %s\n", updatedVersion)
	if legacy != "" {
		fmt.Printf("  Legacy alias: %s\n", legacy)
	}
}

func releaseArch() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("fred.cash update currently supports macOS only (found %s)", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "arm64":
		return "arm64", nil
	case "amd64":
		return "amd64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
}

func updateTargets() (target, legacy string, err error) {
	exe, err := os.Executable()
	if err != nil {
		return "", "", err
	}
	exe, _ = filepath.Abs(exe)
	target = exe
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		target = resolved
	}

	dir := filepath.Dir(target)
	if filepath.Base(target) == "interest" {
		fred := filepath.Join(dir, "fred.cash")
		if _, err := os.Stat(fred); err == nil {
			target = fred
			dir = filepath.Dir(target)
		}
	}
	legacy = filepath.Join(dir, "interest")
	return target, legacy, nil
}

func downloadUpdate(url string) (string, error) {
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	f, err := os.CreateTemp("", "fred.cash-update-*")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	if err := os.Chmod(name, 0o755); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

func installUpdate(tmp, target, legacy string) error {
	if err := installNoSudo(tmp, target, legacy); err == nil {
		return nil
	}
	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("cannot write %s and sudo is unavailable", target)
	}
	fmt.Printf("  Need permission to write %s; sudo may ask for your Mac password.\n", target)
	if err := sudoInstall(tmp, target); err != nil {
		return err
	}
	_ = sudoSymlink(target, legacy)
	return nil
}

func installNoSudo(tmp, target, legacy string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	next := target + ".new"
	in, err := os.Open(tmp)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(next, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(next)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(next)
		return err
	}
	if err := os.Chmod(next, 0o755); err != nil {
		os.Remove(next)
		return err
	}
	if err := os.Rename(next, target); err != nil {
		os.Remove(next)
		return err
	}
	_ = os.Remove(legacy)
	_ = os.Symlink(target, legacy)
	return nil
}

func sudoInstall(tmp, target string) error {
	cmd := exec.Command("sudo", "install", "-m", "0755", tmp, target)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func sudoSymlink(target, legacy string) error {
	if legacy == "" || target == legacy {
		return nil
	}
	cmd := exec.Command("sudo", "ln", "-sf", target, legacy)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func versionFromBinary(path string) string {
	cmd := exec.Command(path, "version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
