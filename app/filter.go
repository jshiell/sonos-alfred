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

// Env is what the commands depend on. Dir is the workflow's data directory.
type Env struct {
	Dir   string
	Now   func() time.Time
	Spawn func() error // starts a detached refresh
}

// Filter renders the rows for a query from the cache alone. It never goes to the network.
func Filter(env Env, query string) ([]byte, error) {
	var current hub.State
	state.NewCache(env.Dir, env.Now).ReadWithAge(StateEntry, &current)
	return alfredjson.Render(hub.Items(current, query), false)
}
