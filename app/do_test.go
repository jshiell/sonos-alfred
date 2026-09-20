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
	"sonos-alfred/state"
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

func TestDoSetsTheGroupVolume(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t, fakespeaker.Recorded(t, "SetGroupVolume", 0)) // DesiredVolume 12

	if err := app.Do(context.Background(), w.env, "volume-set:12"); err != nil {
		t.Fatal(err)
	}
}

// rowsFor is the rows the hub shows for a household with these favorites.
func rowsFor(favorites ...sonos.Item) []hub.Item {
	return hub.Items(hub.State{Favorites: favorites}, "")
}

func rowTitled(t *testing.T, rows []hub.Item, title string) hub.Item {
	t.Helper()
	for _, row := range rows {
		if row.Title == title {
			return row
		}
	}
	t.Fatalf("no row titled %q", title)
	return hub.Item{}
}

var somaFM = sonos.Item{Title: "Groove Salad", URI: "x-rincon-mp3radio://ice1.somafm.com/groovesalad-128-mp3"}

func TestDoPlaysAFavoriteStreamDirectly(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t,
		fakespeaker.Recorded(t, "SetAVTransportURI", 1), // the SomaFM stream, no metadata
		fakespeaker.Recorded(t, "Play", 1),
	)
	enter := hub.Encode(rowTitled(t, rowsFor(somaFM), "Groove Salad").Enter)

	if err := app.Do(context.Background(), w.env, enter); err != nil {
		t.Fatal(err)
	}
}

var nasAlbum = sonos.Item{Title: "Album", URI: "x-file-cifs://nas/album"}

func TestDoAddsAFavoriteToTheEndOfTheQueue(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t, fakespeaker.AVTransport("AddURIToQueue",
		"<InstanceID>0</InstanceID><EnqueuedURI>x-file-cifs://nas/album</EnqueuedURI><EnqueuedURIMetaData></EnqueuedURIMetaData>"+
			"<DesiredFirstTrackNumberEnqueued>0</DesiredFirstTrackNumberEnqueued><EnqueueAsNext>0</EnqueueAsNext>"))
	cmd := hub.Encode(*rowTitled(t, rowsFor(nasAlbum), "Album").Cmd)

	if err := app.Do(context.Background(), w.env, cmd); err != nil {
		t.Fatal(err)
	}
}

func TestDoPlaysAFavoriteNextAfterTheCurrentTrack(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t,
		fakespeaker.Recorded(t, "GetMediaInfo", 3),    // playing from the queue
		fakespeaker.Recorded(t, "GetPositionInfo", 1), // at track 1
		fakespeaker.AVTransport("AddURIToQueue",
			"<InstanceID>0</InstanceID><EnqueuedURI>x-file-cifs://nas/album</EnqueuedURI><EnqueuedURIMetaData></EnqueuedURIMetaData>"+
				"<DesiredFirstTrackNumberEnqueued>2</DesiredFirstTrackNumberEnqueued><EnqueueAsNext>1</EnqueueAsNext>"),
	)
	alt := hub.Encode(*rowTitled(t, rowsFor(nasAlbum), "Album").Alt)

	if err := app.Do(context.Background(), w.env, alt); err != nil {
		t.Fatal(err)
	}
}

func TestDoJumpsToAQueueTrackOnTheTargetsQueue(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t,
		fakespeaker.Recorded(t, "SetAVTransportURI", 2), // x-rincon-queue:<Dining Room>#0
		fakespeaker.Recorded(t, "Seek", 1),              // TRACK_NR 3
		fakespeaker.Recorded(t, "Play", 2),
	)

	if err := app.Do(context.Background(), w.env, "jump:3"); err != nil {
		t.Fatal(err)
	}
}

func TestDoRemembersTheChosenRoomWithoutTouchingASpeaker(t *testing.T) {
	w := newWorkflow(t) // nothing cached, and any request fails the test

	if err := app.Do(context.Background(), w.env, "room:RINCON_KITCHEN"); err != nil {
		t.Fatal(err)
	}

	if got := state.NewSettings(w.env.Dir).ActiveGroup(); got != "RINCON_KITCHEN" {
		t.Errorf("active group = %q, want RINCON_KITCHEN", got)
	}
}

func TestDoSendsLaterActionsToTheChosenRoomBeforeTheCacheCatchesUp(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil { // still says the Dining Room is the target
		t.Fatal(err)
	}
	hosts := w.speakersAre(t, fakespeaker.AVTransport("Next", "<InstanceID>0</InstanceID>"))
	if err := app.Do(context.Background(), w.env, "room:RINCON_KITCHEN"); err != nil {
		t.Fatal(err)
	}

	if err := app.Do(context.Background(), w.env, "next:"); err != nil {
		t.Fatal(err)
	}

	if want := []string{"192.0.2.130"}; !slices.Equal(*hosts, want) {
		t.Errorf("connected to %v, want the Kitchen coordinator %v", *hosts, want)
	}
}

func TestDoTurnsShuffleOnFromTheLiveSetting(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t,
		fakespeaker.Recorded(t, "GetTransportSettings", 2), // NORMAL
		fakespeaker.Recorded(t, "SetPlayMode", 1),          // SHUFFLE_NOREPEAT
	)

	if err := app.Do(context.Background(), w.env, "shuffle:"); err != nil {
		t.Fatal(err)
	}
}

func TestDoCyclesRepeatFromTheLiveSettingAndKeepsShuffle(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t,
		fakespeaker.Recorded(t, "GetTransportSettings", 1), // SHUFFLE_NOREPEAT
		fakespeaker.Recorded(t, "SetPlayMode", 0),          // SHUFFLE: shuffle with repeat all
	)

	if err := app.Do(context.Background(), w.env, "repeat:"); err != nil {
		t.Fatal(err)
	}
}

func TestDoSetsASleepTimerInMinutes(t *testing.T) {
	w := newWorkflow(t)
	if err := w.cache.Write(app.StateEntry, diningRoomAndKitchen()); err != nil {
		t.Fatal(err)
	}
	w.speakersAre(t, fakespeaker.Recorded(t, "ConfigureSleepTimer", 0)) // 00:15:00

	if err := app.Do(context.Background(), w.env, "sleep:15"); err != nil {
		t.Fatal(err)
	}
}
