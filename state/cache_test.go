package state_test

import (
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
