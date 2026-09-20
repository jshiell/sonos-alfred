package app

import (
	"context"
	"errors"

	"sonos-alfred/hub"
	"sonos-alfred/state"
)

// Refresh reads the household from the speakers and writes it to the cache. When the speakers can't be read it
// writes that instead, so filter can say so. It does nothing when another refresh is already running.
func Refresh(ctx context.Context, env Env) error {
	release, acquired := state.NewRefreshLock(env.Dir, env.Now, lockStaleAfter).TryAcquire()
	if !acquired {
		return nil
	}
	defer release()

	current, err := readHousehold(ctx, env)
	if err != nil {
		current = hub.State{Unreachable: true}
	}
	return state.NewCache(env.Dir, env.Now).Write(StateEntry, current)
}

func readHousehold(ctx context.Context, env Env) (hub.State, error) {
	host, err := env.Discover(ctx)
	if err != nil {
		return hub.State{}, err
	}
	topology, err := env.Connect(host).Topology(ctx)
	if err != nil {
		return hub.State{}, err
	}
	group, found := state.ResolveTarget(topology, state.NewSettings(env.Dir).ActiveGroup(), "")
	if !found {
		return hub.State{}, errors.New("the household has no speakers")
	}

	current := hub.State{Topology: topology, Target: group.Coordinator.UUID}
	coordinator := env.Connect(group.Coordinator.Host)
	steps := []func() error{
		func() (err error) { current.NowPlaying, err = coordinator.NowPlaying(ctx); return },
		func() (err error) { current.PlayingFromQueue, err = coordinator.PlayingFromQueue(ctx); return },
		func() (err error) { current.Volume, err = coordinator.GroupVolume(ctx); return },
		func() (err error) { current.PlayMode, err = coordinator.PlayMode(ctx); return },
		func() (err error) { current.SleepRemaining, err = coordinator.SleepTimerRemaining(ctx); return },
		func() (err error) { current.Favorites, err = coordinator.Favorites(ctx); return },
		func() (err error) { current.Playlists, err = coordinator.Playlists(ctx); return },
		func() (err error) { current.Queue, err = coordinator.Queue(ctx); return },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return hub.State{}, err
		}
	}
	return current, nil
}
