package hub_test

import (
	"strings"
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

func TestNowPlayingRowSaysSoWhenNothingIsLoaded(t *testing.T) {
	items := hub.Items(hub.State{}, "")

	if len(items) == 0 {
		t.Fatal("no items")
	}
	if row := items[0]; row.Title != "Nothing playing" || row.Subtitle != "" {
		t.Errorf("row = %q / %q, want 'Nothing playing' with no subtitle", row.Title, row.Subtitle)
	}
}

func TestNowPlayingSubtitleSkipsMissingArtistOrAlbum(t *testing.T) {
	for name, tc := range map[string]struct {
		track sonos.NowPlaying
		want  string
	}{
		"stream, title only": {sonos.NowPlaying{Title: "Groove Salad"}, ""},
		"no album":           {sonos.NowPlaying{Title: "Song", Artist: "Band"}, "Band"},
		"no artist":          {sonos.NowPlaying{Title: "Song", Album: "Record"}, "Record"},
	} {
		t.Run(name, func(t *testing.T) {
			row := hub.Items(hub.State{NowPlaying: tc.track}, "")[0]

			if row.Subtitle != tc.want {
				t.Errorf("Subtitle = %q, want %q", row.Subtitle, tc.want)
			}
		})
	}
}

func TestVolumeRowFollowsNowPlaying(t *testing.T) {
	items := hub.Items(hub.State{Volume: 12}, "")

	if len(items) < 2 {
		t.Fatalf("got %d items, want at least 2", len(items))
	}
	if items[0].Title != "Nothing playing" || items[1].Title != "Volume 12" {
		t.Errorf("first two rows = %q, %q; want the now-playing row then 'Volume 12'", items[0].Title, items[1].Title)
	}
}

func TestVolumeRowStepsTheVolumeUpOnEnterAndDownOnAlt(t *testing.T) {
	row := hub.Items(hub.State{Volume: 12}, "")[1]

	if !row.Valid || row.Enter != (hub.Action{Verb: "volume-change", Payload: "5"}) {
		t.Errorf("Enter = %+v (valid %v), want volume-change 5", row.Enter, row.Valid)
	}
	if row.Alt == nil || *row.Alt != (hub.Action{Verb: "volume-change", Payload: "-5"}) {
		t.Errorf("Alt = %+v, want volume-change -5", row.Alt)
	}
	if row.Cmd != nil {
		t.Errorf("Cmd = %+v, want none", row.Cmd)
	}
}

func TestTypingVolAndANumberOffersToSetThatVolume(t *testing.T) {
	items := hub.Items(hub.State{Volume: 12}, "vol 35")

	if len(items) == 0 {
		t.Fatal("no items")
	}
	row := items[0]
	if row.Title != "Set volume 35" || !row.Valid || row.Enter != (hub.Action{Verb: "volume-set", Payload: "35"}) {
		t.Errorf("first row = %q valid=%v enter=%+v, want 'Set volume 35' doing volume-set 35", row.Title, row.Valid, row.Enter)
	}
}

func TestTypedVolumeIsClampedToZeroThroughHundred(t *testing.T) {
	for query, want := range map[string]string{"vol 150": "100", "vol 100": "100", "vol 0": "0", "vol -5": "0", "vol 007": "7"} {
		t.Run(query, func(t *testing.T) {
			row := hub.Items(hub.State{}, query)[0]

			if row.Title != "Set volume "+want || row.Enter != (hub.Action{Verb: "volume-set", Payload: want}) {
				t.Errorf("first row = %q enter=%+v, want volume %s", row.Title, row.Enter, want)
			}
		})
	}
}

func TestGarbageAfterVolOffersNoSetVolumeRow(t *testing.T) {
	for _, query := range []string{"vol abc", "vol ", "vol 3x", "vol 1.5", "vol"} {
		t.Run(query, func(t *testing.T) {
			for _, item := range hub.Items(hub.State{}, query) {
				if strings.HasPrefix(item.Title, "Set volume") {
					t.Errorf("Items(%q) offered %q", query, item.Title)
				}
			}
		})
	}
}
