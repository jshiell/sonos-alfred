package state_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sonos-alfred/state"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

func newClock() *clock {
	return &clock{now: time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)}
}

type rooms struct {
	Names []string
}

func TestCacheReturnsAFreshValueAsAHit(t *testing.T) {
	clk := newClock()
	cache := state.NewCache(t.TempDir(), clk.Now)
	if err := cache.Write("rooms", rooms{Names: []string{"Kitchen", "Office"}}); err != nil {
		t.Fatal(err)
	}
	clk.now = clk.now.Add(30 * time.Second)

	var got rooms
	hit := cache.Read("rooms", time.Minute, &got)

	if !hit {
		t.Fatal("hit = false, want true")
	}
	if len(got.Names) != 2 || got.Names[0] != "Kitchen" || got.Names[1] != "Office" {
		t.Errorf("got %+v, want Kitchen and Office", got)
	}
}

func TestCacheTreatsAStaleValueAsAMiss(t *testing.T) {
	clk := newClock()
	cache := state.NewCache(t.TempDir(), clk.Now)
	if err := cache.Write("rooms", rooms{Names: []string{"Kitchen"}}); err != nil {
		t.Fatal(err)
	}
	clk.now = clk.now.Add(time.Minute + time.Second)

	var got rooms
	if cache.Read("rooms", time.Minute, &got) {
		t.Errorf("hit = true for a value older than the TTL, got %+v", got)
	}
}

func TestCacheTreatsACorruptFileAsAMiss(t *testing.T) {
	dir := t.TempDir()
	clk := newClock()
	cache := state.NewCache(dir, clk.Now)
	if err := cache.Write("rooms", rooms{Names: []string{"Kitchen", "Office"}}); err != nil {
		t.Fatal(err)
	}
	truncate(t, dir)

	var got rooms
	if cache.Read("rooms", time.Minute, &got) {
		t.Errorf("hit = true for a truncated file, got %+v", got)
	}
}

// truncate cuts every file in dir in half, as a crash mid-write would.
func truncate(t *testing.T, dir string) {
	t.Helper()
	files, err := os.ReadDir(dir)
	if err != nil || len(files) == 0 {
		t.Fatalf("expected the cache to have written a file: %v", err)
	}
	for _, f := range files {
		path := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data[:len(data)/2], 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
