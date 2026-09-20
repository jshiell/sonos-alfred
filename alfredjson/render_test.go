package alfredjson_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"sonos-alfred/alfredjson"
	"sonos-alfred/hub"
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
