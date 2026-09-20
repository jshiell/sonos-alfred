package state_test

import (
	"testing"
	"time"

	"sonos-alfred/state"
)

const lockStaleAfter = 30 * time.Second

func TestRefreshLockRefusesASecondHolderWhileHeld(t *testing.T) {
	dir := t.TempDir()
	clk := newClock()

	_, first := state.NewRefreshLock(dir, clk.Now, lockStaleAfter).TryAcquire()
	_, second := state.NewRefreshLock(dir, clk.Now, lockStaleAfter).TryAcquire()

	if !first {
		t.Error("first TryAcquire = false, want true")
	}
	if second {
		t.Error("second TryAcquire = true while the lock is held, want false")
	}
}

func TestRefreshLockCanBeTakenAgainOnceReleased(t *testing.T) {
	dir := t.TempDir()
	clk := newClock()
	release, _ := state.NewRefreshLock(dir, clk.Now, lockStaleAfter).TryAcquire()

	release()

	if _, again := state.NewRefreshLock(dir, clk.Now, lockStaleAfter).TryAcquire(); !again {
		t.Error("TryAcquire = false after release, want true")
	}
}
