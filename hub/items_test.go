package hub_test

import (
	"testing"

	"sonos-alfred/hub"
	"sonos-alfred/sonos"
)

func TestNowPlayingRowIsFirstAndControlsTransport(t *testing.T) {
	state := hub.State{NowPlaying: sonos.NowPlaying{Title: "Saxon", Artist: "Marbles", Album: "Marbles (20th Anniversary)"}}

	items := hub.Items(state, "")

	if len(items) == 0 {
		t.Fatal("no items")
	}
	row := items[0]
	if row.Title != "Saxon" || row.Subtitle != "Marbles · Marbles (20th Anniversary)" {
		t.Errorf("row = %q / %q, want the track and 'artist · album'", row.Title, row.Subtitle)
	}
	if !row.Valid || row.Enter != (hub.Action{Verb: "playpause"}) {
		t.Errorf("Enter = %+v (valid %v), want playpause", row.Enter, row.Valid)
	}
	if row.Cmd == nil || *row.Cmd != (hub.Action{Verb: "next"}) {
		t.Errorf("Cmd = %+v, want next", row.Cmd)
	}
	if row.Alt == nil || *row.Alt != (hub.Action{Verb: "previous"}) {
		t.Errorf("Alt = %+v, want previous", row.Alt)
	}
}
