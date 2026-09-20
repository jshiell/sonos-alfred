package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"sonos-alfred/app"
	"sonos-alfred/hub"
	"sonos-alfred/internal/fakespeaker"
	"sonos-alfred/sonos"
	"sonos-alfred/state"
)

const livingRoomUUID = "RINCON_22222222222201400"

func TestRefreshWritesTheStateOfTheActiveGroupToTheCache(t *testing.T) {
	w := newWorkflow(t)
	if err := state.NewSettings(w.env.Dir).SetActiveGroup(livingRoomUUID); err != nil {
		t.Fatal(err)
	}
	speaker := fakespeaker.New(t,
		fakespeaker.Topology(t, "zonegroupstate-home-theatre.xml"),
		fakespeaker.Recorded(t, "GetPositionInfo", 0),
		fakespeaker.Recorded(t, "GetMediaInfo", 3), // playing from the queue
		fakespeaker.Recorded(t, "GetGroupVolume", 0),
		fakespeaker.Recorded(t, "GetTransportSettings", 0), // SHUFFLE
		fakespeaker.Recorded(t, "GetRemainingSleepTimerDuration", 0),
		fakespeaker.Browse(t, "FV:2", "browse-favorites-albums-and-shortcuts.xml"),
		fakespeaker.Browse(t, "SQ:", "browse-playlists-empty.xml"),
		fakespeaker.Browse(t, "Q:0", "browse-queue-apple-music.xml"),
	)
	w.speakerHost = strings.TrimPrefix(speaker.URL, "http://")
	var connectedTo []string
	w.env.Discover = func(context.Context) (string, error) { return "192.0.2.99", nil }
	w.env.Connect = func(host string) sonos.Backend {
		connectedTo = append(connectedTo, host)
		return sonos.NewClient(speaker.URL)
	}

	if err := app.Refresh(context.Background(), w.env); err != nil {
		t.Fatal(err)
	}

	var got hub.State
	if !w.cache.Read(app.StateEntry, time.Minute, &got) {
		t.Fatal("refresh wrote no state")
	}
	if want := []string{"192.0.2.99", "192.0.2.34"}; !slices.Equal(connectedTo, want) {
		t.Errorf("connected to %v, want the discovered speaker then the Living Room coordinator %v", connectedTo, want)
	}
	if got.Target != livingRoomUUID {
		t.Errorf("target = %q, want the Living Room coordinator", got.Target)
	}
	if len(got.Topology.Groups) != 6 {
		t.Errorf("topology has %d groups, want 6", len(got.Topology.Groups))
	}
	if got.NowPlaying.Title != "Saxon (2015 - Remaster)" || got.NowPlaying.Track != 1 {
		t.Errorf("now playing = %+v, want Saxon at track 1", got.NowPlaying)
	}
	if !got.PlayingFromQueue {
		t.Error("PlayingFromQueue = false, want true")
	}
	if got.Volume != 12 {
		t.Errorf("volume = %d, want 12", got.Volume)
	}
	if want := (sonos.PlayMode{Shuffle: true, Repeat: sonos.RepeatAll}); got.PlayMode != want {
		t.Errorf("play mode = %+v, want %+v", got.PlayMode, want)
	}
	if got.SleepRemaining != 15*time.Minute {
		t.Errorf("sleep remaining = %v, want 15m", got.SleepRemaining)
	}
	if len(got.Favorites) != 4 || len(got.Playlists) != 0 || len(got.Queue) != 46 {
		t.Errorf("got %d favorites, %d playlists, %d queue tracks, want 4, 0, 46",
			len(got.Favorites), len(got.Playlists), len(got.Queue))
	}
	if got.Cold || got.Unreachable {
		t.Errorf("state is marked cold=%v unreachable=%v, want neither", got.Cold, got.Unreachable)
	}
}

func TestRefreshRecordsUnreachableWhenNoSpeakerIsFound(t *testing.T) {
	w := newWorkflow(t)
	w.env.Discover = func(context.Context) (string, error) { return "", errors.New("no speakers answered") }

	if err := app.Refresh(context.Background(), w.env); err != nil {
		t.Fatalf("Refresh = %v, want the outcome recorded and no error", err)
	}

	var got hub.State
	if !w.cache.Read(app.StateEntry, time.Minute, &got) {
		t.Fatal("refresh wrote no state")
	}
	if !got.Unreachable {
		t.Errorf("state = %+v, want Unreachable", got)
	}
}

func TestRefreshDoesNothingWhileAnotherRefreshIsRunning(t *testing.T) {
	w := newWorkflow(t)
	if _, acquired := state.NewRefreshLock(w.env.Dir, w.clock.Now, time.Hour).TryAcquire(); !acquired {
		t.Fatal("could not take the refresh lock")
	}
	w.env.Discover = func(context.Context) (string, error) {
		t.Error("Discover called while another refresh held the lock")
		return "", errors.New("unreachable")
	}

	if err := app.Refresh(context.Background(), w.env); err != nil {
		t.Fatal(err)
	}

	var got hub.State
	if w.cache.Read(app.StateEntry, time.Minute, &got) {
		t.Errorf("refresh wrote %+v while another was running, want nothing", got)
	}
}

func TestRefreshGivesTheLockBackWhenItIsDone(t *testing.T) {
	w := newWorkflow(t)
	w.env.Discover = func(context.Context) (string, error) { return "", errors.New("no speakers answered") }

	if err := app.Refresh(context.Background(), w.env); err != nil {
		t.Fatal(err)
	}

	if _, acquired := state.NewRefreshLock(w.env.Dir, w.clock.Now, time.Hour).TryAcquire(); !acquired {
		t.Error("the refresh lock is still held after Refresh returned")
	}
}

func TestRefreshTargetsTheGroupThatIsPlayingWhenNoRoomWasChosen(t *testing.T) {
	w := newWorkflow(t)
	speaker := fakespeaker.New(t,
		fakespeaker.Topology(t, "zonegroupstate-home-theatre.xml"), // groups: Living Room, Garden Room, Office, ...
		fakespeaker.Recorded(t, "GetTransportInfo", 3),             // Living Room: STOPPED
		fakespeaker.Recorded(t, "GetTransportInfo", 3),             // Garden Room: STOPPED
		fakespeaker.Recorded(t, "GetTransportInfo", 0),             // Office: PLAYING
		fakespeaker.Recorded(t, "GetPositionInfo", 0),
		fakespeaker.Recorded(t, "GetMediaInfo", 3),
		fakespeaker.Recorded(t, "GetGroupVolume", 0),
		fakespeaker.Recorded(t, "GetTransportSettings", 0),
		fakespeaker.Recorded(t, "GetRemainingSleepTimerDuration", 0),
		fakespeaker.Browse(t, "FV:2", "browse-favorites-albums-and-shortcuts.xml"),
		fakespeaker.Browse(t, "SQ:", "browse-playlists-empty.xml"),
		fakespeaker.Browse(t, "Q:0", "browse-queue-apple-music.xml"),
	)
	w.speakerHost = strings.TrimPrefix(speaker.URL, "http://")
	var connectedTo []string
	w.env.Discover = func(context.Context) (string, error) { return "192.0.2.99", nil }
	w.env.Connect = func(host string) sonos.Backend {
		connectedTo = append(connectedTo, host)
		return sonos.NewClient(speaker.URL)
	}

	if err := app.Refresh(context.Background(), w.env); err != nil {
		t.Fatal(err)
	}

	var got hub.State
	if !w.cache.Read(app.StateEntry, time.Minute, &got) {
		t.Fatal("refresh wrote no state")
	}
	if got.Unreachable || got.Target != "RINCON_55555555555501400" {
		t.Errorf("state = unreachable %v, target %q, want the Office coordinator RINCON_55555555555501400", got.Unreachable, got.Target)
	}
	if last := connectedTo[len(connectedTo)-1]; last != "192.0.2.48" {
		t.Errorf("read the group's state from %s, want the Office at 192.0.2.48", last)
	}
}
