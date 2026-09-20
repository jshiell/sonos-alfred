package app_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"sonos-alfred/app"
	"sonos-alfred/hub"
	"sonos-alfred/internal/fakespeaker"
	"sonos-alfred/sonos"
)

const diningRoomUUID = "RINCON_11111111111101400" // the speaker the recorded exchanges came from

// diningRoomAndKitchen is a household whose controlled group is the Dining Room.
func diningRoomAndKitchen() hub.State {
	diningRoom := sonos.Member{UUID: diningRoomUUID, Name: "Dining Room", Host: "192.0.2.29"}
	kitchen := sonos.Member{UUID: "RINCON_KITCHEN", Name: "Kitchen", Host: "192.0.2.130"}
	return hub.State{
		Target: diningRoomUUID,
		Topology: sonos.Topology{Groups: []sonos.Group{
			{Coordinator: kitchen, Members: []sonos.Member{kitchen}},
			{Coordinator: diningRoom, Members: []sonos.Member{diningRoom}},
		}},
	}
}

// speakersAre points every connection at a fake speaker that expects the exchanges, and returns the hosts connected to.
func (w *workflow) speakersAre(t *testing.T, exchanges ...fakespeaker.Exchange) *[]string {
	t.Helper()
	speaker := fakespeaker.New(t, exchanges...)
	w.speakerHost = strings.TrimPrefix(speaker.URL, "http://")
	var hosts []string
	w.env.Connect = func(host string) sonos.Backend {
		hosts = append(hosts, host)
		return sonos.NewClient(speaker.URL)
	}
	return &hosts
}

func TestDoPausesWhatIsPlayingOnTheTargetsCoordinator(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	hosts := w.speakersAre(t,
		fakespeaker.Recorded(t, "GetTransportInfo", 0), // PLAYING
		fakespeaker.Recorded(t, "Pause", 0),
	)

	if err := app.Do(context.Background(), w.env, "playpause:"); err != nil {
		t.Fatal(err)
	}

	if want := []string{"192.0.2.29"}; !slices.Equal(*hosts, want) {
		t.Errorf("connected to %v, want only the Dining Room coordinator %v", *hosts, want)
	}
}

func TestDoPlaysWhatIsStopped(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t,
		fakespeaker.Recorded(t, "GetTransportInfo", 3), // STOPPED
		fakespeaker.Recorded(t, "Play", 0),
	)

	if err := app.Do(context.Background(), w.env, "playpause:"); err != nil {
		t.Fatal(err)
	}
}

func TestDoSkipsToTheNextTrack(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t, fakespeaker.AVTransport("Next", "<InstanceID>0</InstanceID>"))

	if err := app.Do(context.Background(), w.env, "next:"); err != nil {
		t.Fatal(err)
	}
}

func TestDoGoesBackToThePreviousTrack(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t, fakespeaker.AVTransport("Previous", "<InstanceID>0</InstanceID>"))

	if err := app.Do(context.Background(), w.env, "previous:"); err != nil {
		t.Fatal(err)
	}
}

func TestDoChangesTheGroupVolumeByTheStep(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t, fakespeaker.Recorded(t, "SetRelativeGroupVolume", 3)) // Adjustment -5

	if err := app.Do(context.Background(), w.env, "volume-change:-5"); err != nil {
		t.Fatal(err)
	}
}
