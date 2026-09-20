package hub

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"sonos-alfred/sonos"
)

// State is everything the hub knows, read from the cache.
type State struct {
	NowPlaying sonos.NowPlaying
	Volume     int
	Favorites  []sonos.Item
	Playlists  []sonos.Item
	Queue      []sonos.Item
	Topology   sonos.Topology
	PlayMode   sonos.PlayMode
	// Target is the coordinator UUID of the group being controlled.
	Target string
	// PlayingFromQueue says NowPlaying.Track is a queue position. A stream reports Track 1 too, so Track alone can't tell.
	PlayingFromQueue bool
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
	for _, playable := range slices.Concat(state.Favorites, state.Playlists) {
		if playable.URI == "" { // Sonos Radio shortcuts carry nothing the speaker can be told to play
			continue
		}
		items = append(items, playableRow(playable))
	}
	for i, track := range state.Queue {
		items = append(items, queueRow(track, i+1, state.PlayingFromQueue && state.NowPlaying.Track == i+1))
	}
	for _, room := range roomsByName(state.Topology, state.Target) {
		items = append(items, roomRow(room))
	}
	items = append(items, shuffleRow(state.PlayMode), repeatRow(state.PlayMode))
	for _, minutes := range []int{15, 30, 60} {
		items = append(items, sleepRow(minutes))
	}
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

// playableRow is a favorite or playlist: Enter replaces the queue and plays, ⌘ adds to the end, ⌥ plays next.
func playableRow(item sonos.Item) Item {
	payload := itemPayload(item)
	add := Action{Verb: "add-item", Payload: payload}
	playNext := Action{Verb: "play-next-item", Payload: payload}
	return Item{
		Title: item.Title,
		Valid: true,
		Enter: Action{Verb: "play-item", Payload: payload},
		Cmd:   &add,
		Alt:   &playNext,
	}
}

// itemPayload carries what `do` needs to play an item without looking anything up.
func itemPayload(item sonos.Item) string {
	return url.Values{"uri": {item.URI}, "metadata": {item.Metadata}}.Encode()
}

// ParseItemPayload reads back what itemPayload wrote.
func ParseItemPayload(payload string) (sonos.Item, error) {
	values, err := url.ParseQuery(payload)
	if err != nil {
		return sonos.Item{}, err
	}
	return sonos.Item{URI: values.Get("uri"), Metadata: values.Get("metadata")}, nil
}

// queueRow is the track at a 1-based position in the queue.
func queueRow(track sonos.Item, position int, current bool) Item {
	row := Item{
		Title: track.Title,
		Valid: true,
		Enter: Action{Verb: "jump", Payload: strconv.Itoa(position)},
	}
	if current {
		row.Subtitle = "Playing now"
	}
	return row
}

type room struct {
	sonos.Member
	active bool
}

func roomsByName(topology sonos.Topology, target string) []room {
	var rooms []room
	for _, group := range topology.Groups {
		for _, member := range group.Members {
			rooms = append(rooms, room{Member: member, active: group.Coordinator.UUID == target})
		}
	}
	slices.SortStableFunc(rooms, func(a, b room) int { return strings.Compare(a.Name, b.Name) })
	return rooms
}

// roomRow makes the room's group the one to control.
func roomRow(r room) Item {
	row := Item{
		Title: r.Name,
		Valid: true,
		Enter: Action{Verb: "room", Payload: r.UUID},
	}
	if r.active {
		row.Subtitle = "Active"
	}
	return row
}

func shuffleRow(mode sonos.PlayMode) Item {
	title := "Shuffle: off"
	if mode.Shuffle {
		title = "Shuffle: on"
	}
	return Item{Title: title, Valid: true, Enter: Action{Verb: "shuffle"}}
}

func repeatRow(mode sonos.PlayMode) Item {
	setting := map[sonos.Repeat]string{sonos.RepeatOff: "off", sonos.RepeatAll: "all", sonos.RepeatOne: "one"}[mode.Repeat]
	return Item{Title: "Repeat: " + setting, Valid: true, Enter: Action{Verb: "repeat"}}
}

func sleepRow(minutes int) Item {
	return Item{
		Title: fmt.Sprintf("Sleep in %d minutes", minutes),
		Valid: true,
		Enter: Action{Verb: "sleep", Payload: strconv.Itoa(minutes)},
	}
}
