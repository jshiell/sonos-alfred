package state

import (
	"os"
	"path/filepath"
	"time"
)

// RefreshLock lets only one process refresh the cache at a time.
type RefreshLock struct {
	path       string
	now        func() time.Time
	staleAfter time.Duration
}

func NewRefreshLock(dir string, now func() time.Time, staleAfter time.Duration) *RefreshLock {
	return &RefreshLock{path: filepath.Join(dir, "refresh.lock"), now: now, staleAfter: staleAfter}
}

// TryAcquire takes the lock if nobody holds it, or if the holder has had it longer than the stale time
// (it crashed). The returned function gives the lock back.
func (l *RefreshLock) TryAcquire() (release func(), acquired bool) {
	release = func() { os.Remove(l.path) }
	if l.create() {
		return release, true
	}
	if !l.isStale() {
		return nil, false
	}
	os.Remove(l.path)
	if l.create() {
		return release, true
	}
	return nil, false
}

// create makes the lock file already holding its timestamp, so a rival never sees it empty.
func (l *RefreshLock) create() bool {
	temp, err := os.CreateTemp(filepath.Dir(l.path), ".lock-*")
	if err != nil {
		return false
	}
	defer os.Remove(temp.Name())
	_, writeErr := temp.WriteString(l.now().Format(time.RFC3339Nano))
	if closeErr := temp.Close(); writeErr != nil || closeErr != nil {
		return false
	}
	return os.Link(temp.Name(), l.path) == nil
}

// isStale is true for a lock older than the stale time, and for one too damaged to tell.
func (l *RefreshLock) isStale() bool {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return true
	}
	acquiredAt, err := time.Parse(time.RFC3339Nano, string(data))
	if err != nil {
		return true
	}
	return l.now().Sub(acquiredAt) > l.staleAfter
}
