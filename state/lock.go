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

// TryAcquire takes the lock if nobody holds it. The returned function gives it back.
func (l *RefreshLock) TryAcquire() (release func(), acquired bool) {
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, false
	}
	file.Close()
	return func() { os.Remove(l.path) }, true
}
