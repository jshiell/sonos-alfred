package hub

import "sonos-alfred/sonos"

// State is everything the hub knows, read from the cache.
type State struct {
	NowPlaying sonos.NowPlaying
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
	next, previous := Action{Verb: "next"}, Action{Verb: "previous"}
	return []Item{{
		Title:    state.NowPlaying.Title,
		Subtitle: state.NowPlaying.Artist + " · " + state.NowPlaying.Album,
		Valid:    true,
		Enter:    Action{Verb: "playpause"},
		Cmd:      &next,
		Alt:      &previous,
	}}
}
