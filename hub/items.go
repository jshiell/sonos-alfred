package hub

import (
	"fmt"
	"strconv"
	"strings"

	"sonos-alfred/sonos"
)

// State is everything the hub knows, read from the cache.
type State struct {
	NowPlaying sonos.NowPlaying
	Volume     int
}

// Item is one row Alfred shows. Enter, Cmd and Alt are what happens on Enter, ⌘-Enter and ⌥-Enter.
type Item struct {
	Title    string
	Subtitle string
	Valid    bool
	Enter    Action
	Cmd      *Action
	Alt      *Action
}

// Items returns the rows for a query, in display order.
func Items(state State, query string) []Item {
	items := []Item{nowPlayingRow(state.NowPlaying), volumeRow(state.Volume)}
	if setVolume, ok := setVolumeRow(query); ok {
		items = append([]Item{setVolume}, items...)
	}
	return items
}

// setVolumeRow answers a query like "vol 35".
func setVolumeRow(query string) (Item, bool) {
	typed, found := strings.CutPrefix(query, "vol ")
	if !found {
		return Item{}, false
	}
	number, err := strconv.Atoi(typed)
	if err != nil {
		return Item{}, false
	}
	volume := strconv.Itoa(min(max(number, 0), 100))
	return Item{
		Title: "Set volume " + volume,
		Valid: true,
		Enter: Action{Verb: "volume-set", Payload: volume},
	}, true
}

func nowPlayingRow(track sonos.NowPlaying) Item {
	next, previous := Action{Verb: "next"}, Action{Verb: "previous"}
	row := Item{Valid: true, Enter: Action{Verb: "playpause"}, Cmd: &next, Alt: &previous}
	if track.Title == "" {
		row.Title = "Nothing playing"
		return row
	}
	row.Title = track.Title
	row.Subtitle = joinPresent(" · ", track.Artist, track.Album)
	return row
}

func joinPresent(separator string, parts ...string) string {
	var present []string
	for _, part := range parts {
		if part != "" {
			present = append(present, part)
		}
	}
	return strings.Join(present, separator)
}

const volumeStep = 5

func volumeRow(volume int) Item {
	down := Action{Verb: "volume-change", Payload: strconv.Itoa(-volumeStep)}
	return Item{
		Title:    fmt.Sprintf("Volume %d", volume),
		Subtitle: fmt.Sprintf("↩ up %d · ⌥↩ down %d", volumeStep, volumeStep),
		Valid:    true,
		Enter:    Action{Verb: "volume-change", Payload: strconv.Itoa(volumeStep)},
		Alt:      &down,
	}
}
