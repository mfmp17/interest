package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func SelectReceipt(st *receiptStore, query string) (receipt, error) {
	if st == nil {
		return receipt{}, fmt.Errorf("receipt store is nil")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return receipt{}, fmt.Errorf("position id is required")
	}
	var matches []receipt
	for _, r := range st.Receipts {
		if r.ID == query {
			st.Active = r.ID
			return r, nil
		}
		if strings.HasPrefix(r.ID, query) {
			matches = append(matches, r)
		}
	}
	if len(matches) == 0 {
		return receipt{}, fmt.Errorf("no position matches %q", query)
	}
	if len(matches) > 1 {
		return receipt{}, fmt.Errorf("position prefix %q is ambiguous", query)
	}
	st.Active = matches[0].ID
	return matches[0], nil
}

func MergeReceiptStores(dst, src receiptStore) receiptStore {
	index := make(map[string]int, len(dst.Receipts))
	for i, r := range dst.Receipts {
		index[r.ID] = i
	}
	for _, r := range src.Receipts {
		if i, ok := index[r.ID]; ok {
			dst.Receipts[i] = r
			continue
		}
		index[r.ID] = len(dst.Receipts)
		dst.Receipts = append(dst.Receipts, r)
	}
	if src.Active != "" {
		dst.Active = src.Active
	}
	return dst
}

func ExportReceiptStore(path string, st receiptStore) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func ImportReceiptStore(path string) (receiptStore, error) {
	var st receiptStore
	b, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, err
	}
	if len(st.Receipts) == 0 {
		return st, fmt.Errorf("receipt file contains no positions")
	}
	return st, nil
}
