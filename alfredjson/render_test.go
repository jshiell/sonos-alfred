package alfredjson_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"sonos-alfred/alfredjson"
	"sonos-alfred/hub"
	"sonos-alfred/sonos"
)

// assertJSON compares meaning, not formatting.
func assertJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, got)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("expectation is not JSON: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderPutsEnterInArgAndCmdAndAltInMods(t *testing.T) {
	next, previous := hub.Action{Verb: "next"}, hub.Action{Verb: "previous"}
	items := []hub.Item{{
		Title: "Saxon", Subtitle: "Marbles", Valid: true,
		Enter: hub.Action{Verb: "playpause"}, Cmd: &next, Alt: &previous,
	}}

	got, err := alfredjson.Render(items, false)

	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, got, `{"items":[{
		"title": "Saxon", "subtitle": "Marbles", "valid": true, "arg": "playpause:",
		"mods": {"cmd": {"arg": "next:", "valid": true}, "alt": {"arg": "previous:", "valid": true}}
	}]}`)
}

func TestRenderDisablesModifiersTheRowHasNoActionFor(t *testing.T) {
	items := []hub.Item{{Title: "Kitchen", Valid: true, Enter: hub.Action{Verb: "room", Payload: "RINCON_KITCHEN"}}}

	got, err := alfredjson.Render(items, false)

	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, got, `{"items":[{
		"title": "Kitchen", "valid": true, "arg": "room:RINCON_KITCHEN",
		"mods": {"cmd": {"valid": false}, "alt": {"valid": false}}
	}]}`)
}

func TestRenderGivesRowsThatCannotBeActedOnNoArgAndNoMods(t *testing.T) {
	items := []hub.Item{{Title: "Can't reach Sonos", Subtitle: "Check the speakers"}}

	got, err := alfredjson.Render(items, false)

	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, got, `{"items":[{"title": "Can't reach Sonos", "subtitle": "Check the speakers", "valid": false}]}`)
}

func TestRenderAsksAlfredToRunAgainWhileARefreshIsPending(t *testing.T) {
	got, err := alfredjson.Render([]hub.Item{{Title: "Loading Sonos…"}}, true)

	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, got, `{"rerun": 0.3, "items":[{"title": "Loading Sonos…", "valid": false}]}`)
}

// Alfred may fill in {query} wherever it finds it, so a track called "{query}" must not put the token in the output.
func TestRenderNeverEmitsALiteralQueryPlaceholderButShowsTheSameText(t *testing.T) {
	action := hub.Action{Verb: "play-item", Payload: "{query}"}
	items := []hub.Item{{
		Title: "{query}", Subtitle: "Live {query} version", Valid: true,
		Enter: action, Cmd: &action, Alt: &action,
	}}

	got, err := alfredjson.Render(items, false)

	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "{query}") {
		t.Errorf("output holds a literal {query}: %s", got)
	}
	var parsed struct {
		Items []struct{ Title, Subtitle string }
	}
	if err := json.Unmarshal(got, &parsed); err != nil {
		t.Fatal(err)
	}
	row := parsed.Items[0]
	if shown := strings.ReplaceAll(row.Title, "\u200b", ""); shown != "{query}" {
		t.Errorf("title reads %q once invisible characters are ignored, want {query}", shown)
	}
	if shown := strings.ReplaceAll(row.Subtitle, "\u200b", ""); shown != "Live {query} version" {
		t.Errorf("subtitle reads %q once invisible characters are ignored, want the original", shown)
	}
}

func TestWarmHouseholdMatchesTheGoldenFile(t *testing.T) {
	kitchen := sonos.Member{UUID: "RINCON_KITCHEN", Name: "Kitchen"}
	state := hub.State{
		NowPlaying: sonos.NowPlaying{Title: "Saxon", Artist: "Marbles", Album: "Marbles (20th Anniversary)"},
		Volume:     12,
		Favorites:  []sonos.Item{{Title: "Niero:Atlas", URI: "x-rincon-cpcontainer:1004"}},
		Topology:   sonos.Topology{Groups: []sonos.Group{{Coordinator: kitchen, Members: []sonos.Member{kitchen}}}},
		Target:     kitchen.UUID,
	}
	golden, err := os.ReadFile("testdata/warm-household.golden.json")
	if err != nil {
		t.Fatal(err)
	}

	got, err := alfredjson.Render(hub.Items(state, ""), false)

	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, got, string(golden))
}
