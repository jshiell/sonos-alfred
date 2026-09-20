// Package app runs the three commands the workflow is made of: filter, refresh and do.
package app

import (
	"time"

	"sonos-alfred/alfredjson"
	"sonos-alfred/hub"
	"sonos-alfred/state"
)

// StateEntry is the cache entry holding the hub.State that refresh writes and filter renders.
const StateEntry = "state"

// freshFor is how long a refresh's result is trusted before filter asks for another.
const freshFor = 10 * time.Second

// lockStaleAfter is how long a refresh may hold the lock before it is presumed dead.
const lockStaleAfter = 30 * time.Second

// Env is what the commands depend on. Dir is the workflow's data directory.
type Env struct {
	Dir   string
	Now   func() time.Time
	Spawn func() error // starts a detached refresh
}

// Filter renders the rows for a query from the cache alone. It never goes to the network.
func Filter(env Env, query string) ([]byte, error) {
	var current hub.State
	age, found := state.NewCache(env.Dir, env.Now).ReadWithAge(StateEntry, &current)
	if !found {
		current = hub.State{Cold: true}
	}
	refreshPending := false
	if !found || age > freshFor {
		lock := state.NewRefreshLock(env.Dir, env.Now, lockStaleAfter)
		refreshPending = lock.Held() || env.Spawn() == nil
	}
	return alfredjson.Render(hub.Items(current, query), refreshPending)
}
