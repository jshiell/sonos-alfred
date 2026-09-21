package hub_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

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

// AirPlay, for one, gives a track with no title.
func TestNowPlayingRowSaysPlayingWhenATrackHasNoTitle(t *testing.T) {
	items := hub.Items(hub.State{NowPlaying: sonos.NowPlaying{Track: 1}}, "")

	if row := items[0]; row.Title != "Playing" || row.Subtitle != "" {
		t.Errorf("row = %q / %q, want 'Playing' with no subtitle", row.Title, row.Subtitle)
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

// assertRowsInOrder checks the titles appear one straight after another, in this order, wherever they sit in the list.
func assertRowsInOrder(t *testing.T, items []hub.Item, want ...string) {
	t.Helper()
	got := titles(items)
	for start := 0; start+len(want) <= len(got); start++ {
		if slices.Equal(got[start:start+len(want)], want) {
			return
		}
	}
	t.Errorf("titles = %v, want %v one straight after another", got, want)
}

func findItem(t *testing.T, items []hub.Item, title string) hub.Item {
	t.Helper()
	for _, item := range items {
		if item.Title == title {
			return item
		}
	}
	t.Fatalf("no item titled %q in %v", title, titles(items))
	return hub.Item{}
}

func titles(items []hub.Item) []string {
	var got []string
	for _, item := range items {
		got = append(got, item.Title)
	}
	return got
}

func assertItemAction(t *testing.T, name string, got hub.Action, wantVerb string, want sonos.Item) {
	t.Helper()
	if got.Verb != wantVerb {
		t.Errorf("%s verb = %q, want %q", name, got.Verb, wantVerb)
		return
	}
	item, err := hub.ParseItemPayload(got.Payload)
	if err != nil || item.URI != want.URI || item.Metadata != want.Metadata {
		t.Errorf("%s payload = %+v, %v; want URI %q and metadata %q", name, item, err, want.URI, want.Metadata)
	}
}

func TestFavoriteReplacesOnEnterAddsOnCmdAndPlaysNextOnAlt(t *testing.T) {
	album := sonos.Item{ID: "FV:2/50", Title: "Niero:Atlas", URI: "x-rincon-cpcontainer:1004206c?sid=204&flags=8300", Metadata: `<DIDL-Lite><item id="1004206c"><dc:title>Niero & Co</dc:title></item></DIDL-Lite>`}

	row := findItem(t, hub.Items(hub.State{Favorites: []sonos.Item{album}}, ""), "Niero:Atlas")

	if !row.Valid {
		t.Error("favorite is not valid")
	}
	assertItemAction(t, "Enter", row.Enter, "play-item", album)
	if row.Cmd == nil || row.Alt == nil {
		t.Fatalf("Cmd = %v, Alt = %v, want both", row.Cmd, row.Alt)
	}
	assertItemAction(t, "Cmd", *row.Cmd, "add-item", album)
	assertItemAction(t, "Alt", *row.Alt, "play-next-item", album)
}

func TestFavoritesWithNothingToPlayAreHidden(t *testing.T) {
	album := sonos.Item{Title: "Niero:Atlas", URI: "x-rincon-cpcontainer:1004206c"}
	radioShortcut := sonos.Item{Title: "Discover Sonos Radio", URI: ""}

	got := titles(hub.Items(hub.State{Favorites: []sonos.Item{radioShortcut, album}}, ""))

	for _, title := range got {
		if title == "Discover Sonos Radio" {
			t.Errorf("shortcut favorite is shown: %v", got)
		}
	}
	findItem(t, hub.Items(hub.State{Favorites: []sonos.Item{radioShortcut, album}}, ""), "Niero:Atlas")
}

func TestPlaylistsFollowFavoritesWithTheSameActions(t *testing.T) {
	album := sonos.Item{Title: "Niero:Atlas", URI: "x-rincon-cpcontainer:1004206c"}
	playlist := sonos.Item{ID: "SQ:1", Title: "Focus", URI: "file:///jffs/settings/savedqueues.rsq#1", Metadata: "<DIDL-Lite/>"}

	items := hub.Items(hub.State{Favorites: []sonos.Item{album}, Playlists: []sonos.Item{playlist}}, "")

	assertRowsInOrder(t, items, "Niero:Atlas", "Focus")
	row := findItem(t, items, "Focus")
	assertItemAction(t, "Enter", row.Enter, "play-item", playlist)
	if row.Cmd == nil || row.Alt == nil {
		t.Fatalf("Cmd = %v, Alt = %v, want both", row.Cmd, row.Alt)
	}
	assertItemAction(t, "Cmd", *row.Cmd, "add-item", playlist)
	assertItemAction(t, "Alt", *row.Alt, "play-next-item", playlist)
}

func TestFavoriteAndPlaylistRowsSayWhatEachKeyDoes(t *testing.T) {
	album := sonos.Item{Title: "Niero:Atlas", URI: "x-rincon-cpcontainer:1004206c"}
	playlist := sonos.Item{Title: "Focus", URI: "file:///jffs/settings/savedqueues.rsq#1"}

	items := hub.Items(hub.State{Favorites: []sonos.Item{album}, Playlists: []sonos.Item{playlist}}, "")

	const want = "↩ play · ⌘↩ add to end · ⌥↩ play next"
	for _, title := range []string{"Niero:Atlas", "Focus"} {
		if got := findItem(t, items, title).Subtitle; got != want {
			t.Errorf("%s subtitle = %q, want %q", title, got, want)
		}
	}
}

func queueOf(titles ...string) []sonos.Item {
	var queue []sonos.Item
	for _, title := range titles {
		queue = append(queue, sonos.Item{Title: title})
	}
	return queue
}

func TestQueueRowsFollowPlaylistsAndJumpToTheirPosition(t *testing.T) {
	playlist := sonos.Item{Title: "Focus", URI: "file:///jffs/settings/savedqueues.rsq#1"}

	items := hub.Items(hub.State{Playlists: []sonos.Item{playlist}, Queue: queueOf("Saxon", "Quartz", "Only Sunny When It Snows")}, "")

	assertRowsInOrder(t, items, "Focus", "Saxon", "Quartz", "Only Sunny When It Snows")
	row := findItem(t, items, "Only Sunny When It Snows")
	if !row.Valid || row.Enter != (hub.Action{Verb: "jump", Payload: "3"}) {
		t.Errorf("Enter = %+v (valid %v), want jump 3", row.Enter, row.Valid)
	}
	if row.Cmd != nil || row.Alt != nil {
		t.Errorf("Cmd = %v, Alt = %v, want none", row.Cmd, row.Alt)
	}
}

func TestQueueRowForTheCurrentTrackIsMarked(t *testing.T) {
	state := hub.State{
		NowPlaying:       sonos.NowPlaying{Track: 2, Title: "Quartz"},
		PlayingFromQueue: true,
		Queue:            queueOf("Saxon", "Quartz", "Only Sunny When It Snows"),
	}

	items := hub.Items(state, "")

	for title, want := range map[string]string{"Saxon": "", "Only Sunny When It Snows": ""} {
		if got := findItem(t, items, title).Subtitle; got != want {
			t.Errorf("%s subtitle = %q, want %q", title, got, want)
		}
	}
	if got := queueRowSubtitle(t, items, 2); got != "Playing now" {
		t.Errorf("current track subtitle = %q, want 'Playing now'", got)
	}
}

// queueRowSubtitle finds the queue row that jumps to a 1-based position: the now-playing row can share a title with it.
func queueRowSubtitle(t *testing.T, items []hub.Item, position int) string {
	t.Helper()
	jump := hub.Action{Verb: "jump", Payload: strconv.Itoa(position)}
	for _, item := range items {
		if item.Enter == jump {
			return item.Subtitle
		}
	}
	t.Fatalf("no row jumps to position %d", position)
	return ""
}

func TestNoQueueRowIsMarkedWhileAStreamPlays(t *testing.T) {
	state := hub.State{
		NowPlaying: sonos.NowPlaying{Track: 1, Title: "Groove Salad"}, // a stream reports Track 1
		Queue:      queueOf("Saxon", "Quartz"),
	}

	items := hub.Items(state, "")

	for position := 1; position <= 2; position++ {
		if got := queueRowSubtitle(t, items, position); got != "" {
			t.Errorf("queue row %d subtitle = %q, want none", position, got)
		}
	}
}

var (
	kitchen    = sonos.Member{UUID: "RINCON_KITCHEN", Name: "Kitchen"}
	office     = sonos.Member{UUID: "RINCON_OFFICE", Name: "Office"}
	diningRoom = sonos.Member{UUID: "RINCON_DINING", Name: "Dining Room"}
	livingRoom = sonos.Member{UUID: "RINCON_LIVING", Name: "Living Room"}

	// Office coordinates a group that Dining Room has joined.
	household = sonos.Topology{Groups: []sonos.Group{
		{Coordinator: office, Members: []sonos.Member{office, diningRoom}},
		{Coordinator: kitchen, Members: []sonos.Member{kitchen}},
		{Coordinator: livingRoom, Members: []sonos.Member{livingRoom}},
	}}
)

func TestRoomsAreListedByNameAfterTheQueueAndSetTheActiveGroupOnEnter(t *testing.T) {
	items := hub.Items(hub.State{Topology: household, Queue: queueOf("Saxon")}, "")

	assertRowsInOrder(t, items, "Saxon", "Dining Room", "Kitchen", "Living Room", "Office")
	row := findItem(t, items, "Dining Room")
	if !row.Valid || row.Enter != (hub.Action{Verb: "room", Payload: "RINCON_DINING"}) {
		t.Errorf("Enter = %+v (valid %v), want room RINCON_DINING", row.Enter, row.Valid)
	}
}

func TestRoomsInTheActiveGroupAreMarked(t *testing.T) {
	items := hub.Items(hub.State{Topology: household, Target: office.UUID}, "")

	for room, want := range map[string]string{"Office": "Active", "Dining Room": "Active", "Kitchen": "", "Living Room": ""} {
		if got := findItem(t, items, room).Subtitle; got != want {
			t.Errorf("%s subtitle = %q, want %q", room, got, want)
		}
	}
}

func TestShuffleRowFollowsTheRoomsShowsTheModeAndTogglesOnEnter(t *testing.T) {
	for shuffle, want := range map[bool]string{false: "Shuffle: off", true: "Shuffle: on"} {
		t.Run(want, func(t *testing.T) {
			items := hub.Items(hub.State{Topology: household, PlayMode: sonos.PlayMode{Shuffle: shuffle}}, "")

			row := findItem(t, items, want)
			if !row.Valid || row.Enter != (hub.Action{Verb: "shuffle"}) {
				t.Errorf("Enter = %+v (valid %v), want shuffle", row.Enter, row.Valid)
			}
			assertRowsInOrder(t, items, "Office", want)
		})
	}
}

func TestRepeatRowFollowsShuffleShowsTheModeAndCyclesOnEnter(t *testing.T) {
	for repeat, want := range map[sonos.Repeat]string{sonos.RepeatOff: "Repeat: off", sonos.RepeatAll: "Repeat: all", sonos.RepeatOne: "Repeat: one"} {
		t.Run(want, func(t *testing.T) {
			items := hub.Items(hub.State{PlayMode: sonos.PlayMode{Repeat: repeat}}, "")

			row := findItem(t, items, want)
			if !row.Valid || row.Enter != (hub.Action{Verb: "repeat"}) {
				t.Errorf("Enter = %+v (valid %v), want repeat", row.Enter, row.Valid)
			}
			assertRowsInOrder(t, items, "Shuffle: off", want)
		})
	}
}

func TestSleepPresetsFollowRepeatAndSetTheTimerOnEnter(t *testing.T) {
	items := hub.Items(hub.State{}, "")

	assertRowsInOrder(t, items, "Repeat: off", "Sleep in 15 minutes", "Sleep in 30 minutes", "Sleep in 60 minutes")
	for minutes, title := range map[string]string{"15": "Sleep in 15 minutes", "30": "Sleep in 30 minutes", "60": "Sleep in 60 minutes"} {
		row := findItem(t, items, title)
		if !row.Valid || row.Enter != (hub.Action{Verb: "sleep", Payload: minutes}) {
			t.Errorf("%s: Enter = %+v (valid %v), want sleep %s", title, row.Enter, row.Valid, minutes)
		}
	}
}

func TestRunningSleepTimerOffersCancelWithTheTimeLeft(t *testing.T) {
	items := hub.Items(hub.State{SleepRemaining: 22*time.Minute + 41*time.Second}, "")

	assertRowsInOrder(t, items, "Repeat: off", "Cancel sleep timer", "Sleep in 15 minutes")
	row := findItem(t, items, "Cancel sleep timer")
	if row.Subtitle != "23 min left" || !row.Valid || row.Enter != (hub.Action{Verb: "sleep-cancel"}) {
		t.Errorf("row = subtitle %q valid %v enter %+v, want '23 min left' doing sleep-cancel", row.Subtitle, row.Valid, row.Enter)
	}
}

func TestNoCancelRowWithoutASleepTimer(t *testing.T) {
	for _, title := range titles(hub.Items(hub.State{}, "")) {
		if title == "Cancel sleep timer" {
			t.Error("cancel row shown with no timer running")
		}
	}
}

func fullHousehold() hub.State {
	return hub.State{
		Volume:    12,
		Favorites: []sonos.Item{{Title: "Niero:Atlas", URI: "x-rincon-cpcontainer:1"}, {Title: "Marbles", URI: "x-rincon-cpcontainer:2"}},
		Queue:     queueOf("Saxon", "Quartz"),
		Topology:  household,
	}
}

func TestQueryKeepsOnlyRowsWhoseTitleMatches(t *testing.T) {
	got := titles(hub.Items(fullHousehold(), "nier"))

	if want := []string{"Niero:Atlas"}; !slices.Equal(got, want) {
		t.Errorf("titles = %v, want %v", got, want)
	}
}

func TestQueryMatchesLettersInOrderNotNecessarilyAdjacent(t *testing.T) {
	for query, want := range map[string][]string{
		"nra":  {"Niero:Atlas"},
		"NRA":  {"Niero:Atlas"},
		"sx":   {"Saxon"},
		"xs":   nil, // Same letters as sx in the wrong order: no title has an s after its x
		"shuf": {"Shuffle: off"},
	} {
		t.Run(query, func(t *testing.T) {
			got := titles(hub.Items(fullHousehold(), query))

			if !slices.Equal(got, want) {
				t.Errorf("titles = %v, want %v", got, want)
			}
		})
	}
}

func TestRowsWhoseTitleContainsTheQueryComeBeforeScatteredMatches(t *testing.T) {
	state := hub.State{Queue: queueOf("Kinetic"), Topology: household}

	got := titles(hub.Items(state, "kit"))

	if want := []string{"Kitchen", "Kinetic"}; !slices.Equal(got, want) {
		t.Errorf("titles = %v, want %v", got, want)
	}
}

func TestColdCacheShowsOnlyALoadingRowWhateverIsTyped(t *testing.T) {
	for _, query := range []string{"", "nier", "vol 35"} {
		t.Run(query, func(t *testing.T) {
			items := hub.Items(hub.State{Cold: true}, query)

			if len(items) != 1 || items[0].Title != "Loading Sonos…" || items[0].Valid {
				t.Errorf("items = %+v, want one invalid 'Loading Sonos…' row", items)
			}
		})
	}
}

func TestUnreachableSonosShowsOnlyThatWhetherOrNotThereIsCachedData(t *testing.T) {
	warm := fullHousehold()
	warm.Unreachable = true
	cold := hub.State{Cold: true, Unreachable: true}

	for name, state := range map[string]hub.State{"warm cache": warm, "cold cache": cold} {
		t.Run(name, func(t *testing.T) {
			items := hub.Items(state, "")

			if len(items) != 1 || items[0].Title != "Can't reach Sonos" || items[0].Valid || items[0].Subtitle == "" {
				t.Errorf("items = %+v, want one invalid 'Can't reach Sonos' row with a hint", items)
			}
		})
	}
}
