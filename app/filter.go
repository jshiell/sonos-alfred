// Package app runs the three commands the workflow is made of: filter, refresh and do.
package app

import (
	"context"
	"time"

	"sonos-alfred/alfredjson"
	"sonos-alfred/hub"
	"sonos-alfred/sonos"
	"sonos-alfred/state"
)

// StateEntry is the cache entry holding the hub.State that refresh writes and filter renders.
const StateEntry = "state"

// freshFor is how long a refresh's result is trusted before filter asks for another.
const freshFor = 10 * time.Second

// lockStaleAfter is how long a refresh may hold the lock before it is presumed dead.
const lockStaleAfter = 30 * time.Second

// rerunsEntry counts the reruns asked of Alfred, so that a refresh that never finishes cannot keep Alfred polling.
const rerunsEntry = "reruns"

const (
	// maxReruns is about 30s of Alfred's 0.3s reruns, which is longer than a refresh is allowed to run.
	maxReruns = 100
	// rerunCountResetsAfter is the pause in filter runs after which the count starts again from zero.
	rerunCountResetsAfter = 10 * time.Second
)

// Env is what the commands depend on. Dir is the workflow's data directory.
type Env struct {
	Dir      string
	Now      func() time.Time
	Spawn    func() error                              // starts a detached refresh
	Discover func(ctx context.Context) (string, error) // finds the host of any speaker on the network
	Connect  func(host string) sonos.Backend           // reaches the speaker at host
}

// Filter renders the rows for a query from the cache alone. It never goes to the network.
func Filter(env Env, query string) ([]byte, error) {
	cache := state.NewCache(env.Dir, env.Now)
	var current hub.State
	age, found := cache.ReadWithAge(StateEntry, &current)
	if !found {
		current = hub.State{Cold: true}
	}
	refreshPending := false
	if !found || age > freshFor {
		lock := state.NewRefreshLock(env.Dir, env.Now, lockStaleAfter)
		refreshPending = lock.Held() || env.Spawn() == nil
	}
	return alfredjson.Render(hub.Items(current, query), refreshPending && takeRerun(cache))
}

// takeRerun spends one of the reruns Alfred may be asked for and says whether there was one to spend. The count is
// kept on disk because every filter run is a new process. A rerun that can't be counted is refused, as it could
// never be capped.
func takeRerun(cache *state.Cache) bool {
	var used int
	if age, found := cache.ReadWithAge(rerunsEntry, &used); !found || age > rerunCountResetsAfter {
		used = 0
	}
	if used >= maxReruns {
		return false
	}
	return cache.Write(rerunsEntry, used+1) == nil
}
