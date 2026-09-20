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

func TestCacheKeepsThePreviousValueWhenAWriteFails(t *testing.T) {
	dir := t.TempDir()
	clk := newClock()
	cache := state.NewCache(dir, clk.Now)
	if err := cache.Write("rooms", rooms{Names: []string{"Kitchen"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil { // no new files can be created
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })

	if err := cache.Write("rooms", rooms{Names: []string{"Office"}}); err == nil {
		t.Fatal("Write succeeded in a directory it cannot create files in, want an error")
	}

	var got rooms
	if !cache.Read("rooms", time.Minute, &got) || len(got.Names) != 1 || got.Names[0] != "Kitchen" {
		t.Errorf("after the failed write got %+v, want the previous Kitchen value", got)
	}
}

func TestCacheLeavesNoTemporaryFilesBehind(t *testing.T) {
	dir := t.TempDir()
	cache := state.NewCache(dir, newClock().Now)

	for _, name := range []string{"rooms", "rooms", "queue"} {
		if err := cache.Write(name, rooms{}); err != nil {
			t.Fatal(err)
		}
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("directory holds %d files after writing 2 names, want 2: %v", len(files), files)
	}
}

func TestCacheReadWithAgeReturnsAStaleValueAndHowOldItIs(t *testing.T) {
	clk := newClock()
	cache := state.NewCache(t.TempDir(), clk.Now)
	if err := cache.Write("rooms", rooms{Names: []string{"Kitchen"}}); err != nil {
		t.Fatal(err)
	}
	clk.now = clk.now.Add(2 * time.Hour)

	var got rooms
	age, found := cache.ReadWithAge("rooms", &got)

	if !found {
		t.Fatal("found = false, want true for an old but intact value")
	}
	if age != 2*time.Hour {
		t.Errorf("age = %v, want 2h", age)
	}
	if len(got.Names) != 1 || got.Names[0] != "Kitchen" {
		t.Errorf("got %+v, want Kitchen", got)
	}
}

func TestCacheExpireMakesAValueStaleButKeepsIt(t *testing.T) {
	clk := newClock()
	cache := state.NewCache(t.TempDir(), clk.Now)
	if err := cache.Write("rooms", rooms{Names: []string{"Kitchen"}}); err != nil {
		t.Fatal(err)
	}

	if err := cache.Expire("rooms"); err != nil {
		t.Fatal(err)
	}

	var fresh rooms
	if cache.Read("rooms", 24*time.Hour, &fresh) {
		t.Errorf("Read hit for an expired value, got %+v", fresh)
	}
	var kept rooms
	if _, found := cache.ReadWithAge("rooms", &kept); !found || len(kept.Names) != 1 || kept.Names[0] != "Kitchen" {
		t.Errorf("ReadWithAge found=%v %+v, want the Kitchen value still there", found, kept)
	}
}
