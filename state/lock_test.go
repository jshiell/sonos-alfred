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

func TestRefreshLockIsRecoveredFromAHolderThatNeverReleased(t *testing.T) {
	dir := t.TempDir()
	clk := newClock()
	state.NewRefreshLock(dir, clk.Now, lockStaleAfter).TryAcquire() // then the process dies
	clk.now = clk.now.Add(lockStaleAfter + time.Second)

	if _, recovered := state.NewRefreshLock(dir, clk.Now, lockStaleAfter).TryAcquire(); !recovered {
		t.Error("TryAcquire = false for a lock older than the stale time, want true")
	}
}

func TestRefreshLockIsStillHeldJustBeforeItGoesStale(t *testing.T) {
	dir := t.TempDir()
	clk := newClock()
	state.NewRefreshLock(dir, clk.Now, lockStaleAfter).TryAcquire()
	clk.now = clk.now.Add(lockStaleAfter - time.Second)

	if _, taken := state.NewRefreshLock(dir, clk.Now, lockStaleAfter).TryAcquire(); taken {
		t.Error("TryAcquire = true for a lock younger than the stale time, want false")
	}
}
