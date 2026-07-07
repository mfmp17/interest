package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type Store struct {
	mu    sync.Mutex
	path  string
	state StoreState
}

type StoreState struct {
	NextDeriveIndex uint64      `json:"next_derive_index"`
	Positions       []*Position `json:"positions"`
}

func OpenStore(path string) (*Store, error) {
	s := &Store{path: path, state: StoreState{Positions: []*Position{}}}
	b, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(b, &s.state); err != nil {
			return nil, err
		}
	}
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		if err := s.persistLocked(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) NextIndex() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.state.NextDeriveIndex
	s.state.NextDeriveIndex++
	_ = s.persistLocked()
	return idx
}

func (s *Store) Upsert(p *Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Positions {
		if s.state.Positions[i].ID == p.ID {
			s.state.Positions[i] = p
			return s.persistLocked()
		}
	}
	s.state.Positions = append(s.state.Positions, p)
	return s.persistLocked()
}

func (s *Store) Get(id string) (*Position, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.state.Positions {
		if p.ID == id {
			cp := *p
			return &cp, true
		}
	}
	return nil, false
}

func (s *Store) All() []*Position {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Position, 0, len(s.state.Positions))
	for _, p := range s.state.Positions {
		cp := *p
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *Store) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
