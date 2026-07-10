package main

import "testing"

func TestBlockChunksCapsEachRange(t *testing.T) {
	got := blockChunks(100, 20_000, 9_500)
	want := [][2]uint64{{100, 9_599}, {9_600, 19_099}, {19_100, 20_000}}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chunk %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestBlockChunksRejectsEmptyRange(t *testing.T) {
	if got := blockChunks(20, 10, 9_500); len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
}
