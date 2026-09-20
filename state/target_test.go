package state_test

import (
	"testing"

	"sonos-alfred/sonos"
	"sonos-alfred/state"
)

var (
	kitchen    = sonos.Member{UUID: "RINCON_KITCHEN", Name: "Kitchen", Host: "10.0.0.1"}
	office     = sonos.Member{UUID: "RINCON_OFFICE", Name: "Office", Host: "10.0.0.2"}
	diningRoom = sonos.Member{UUID: "RINCON_DINING", Name: "Dining Room", Host: "10.0.0.3"}
	livingRoom = sonos.Member{UUID: "RINCON_LIVING", Name: "Living Room", Host: "10.0.0.4"}

	// Office coordinates a group that Dining Room has joined.
	household = sonos.Topology{Groups: []sonos.Group{
		{Coordinator: kitchen, Members: []sonos.Member{kitchen}},
		{Coordinator: office, Members: []sonos.Member{office, diningRoom}},
		{Coordinator: livingRoom, Members: []sonos.Member{livingRoom}},
	}}
)

func TestTargetIsTheCurrentGroupOfTheStoredPlayerEvenWhenItIsNoLongerCoordinator(t *testing.T) {
	got, ok := state.ResolveTarget(household, diningRoom.UUID, "")

	if !ok {
		t.Fatal("no target found")
	}
	if got.Coordinator != office {
		t.Errorf("coordinator = %s, want Office (Dining Room joined its group)", got.Coordinator.Name)
	}
}

func TestTargetFallsBackToThePlayingGroupWhenTheStoredPlayerIsGone(t *testing.T) {
	got, ok := state.ResolveTarget(household, "RINCON_UNPLUGGED", livingRoom.UUID)

	if !ok {
		t.Fatal("no target found")
	}
	if got.Coordinator != livingRoom {
		t.Errorf("coordinator = %s, want Living Room (the group that is playing)", got.Coordinator.Name)
	}
}
