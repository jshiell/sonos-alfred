package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"sonos-alfred/hub"
	"sonos-alfred/sonos"
	"sonos-alfred/state"
)

// Do performs an action, as Alfred passes it from a row's arg, on the coordinator of the group being controlled.
func Do(ctx context.Context, env Env, encoded string) error {
	action, err := hub.Decode(encoded)
	if err != nil {
		return err
	}
	var current hub.State
	if _, found := state.NewCache(env.Dir, env.Now).ReadWithAge(StateEntry, &current); !found {
		return errors.New("Sonos hasn't been read yet, try again in a moment")
	}
	group, found := state.ResolveTarget(current.Topology, state.NewSettings(env.Dir).ActiveGroup(), current.Target)
	if !found {
		return errors.New("the household has no speakers")
	}
	speaker := env.Connect(group.Coordinator.Host)

	switch action.Verb {
	case "playpause":
		transport, err := speaker.TransportState(ctx)
		if err != nil {
			return err
		}
		if transport == sonos.StatePlaying {
			return speaker.Pause(ctx)
		}
		return speaker.Play(ctx)
	case "next":
		return speaker.Next(ctx)
	case "previous":
		return speaker.Previous(ctx)
	case "volume-change":
		step, err := strconv.Atoi(action.Payload)
		if err != nil {
			return fmt.Errorf("bad volume step %q", action.Payload)
		}
		_, err = speaker.ChangeGroupVolume(ctx, step)
		return err
	}
	return fmt.Errorf("unknown action %q", action.Verb)
}
