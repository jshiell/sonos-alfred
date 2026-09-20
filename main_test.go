package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"sonos-alfred/app"
	"sonos-alfred/hub"
	"sonos-alfred/sonos"
	"sonos-alfred/state"
)

// withFreshState fills the data directory with a state that needs no refresh, so filter never starts one.
func withFreshState(t *testing.T, current hub.State) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("alfred_workflow_data", dir)
	if err := state.NewCache(dir, time.Now).Write(app.StateEntry, current); err != nil {
		t.Fatal(err)
	}
}

func TestFilterPrintsTheRowsAsJSONAndExitsZero(t *testing.T) {
	withFreshState(t, hub.State{NowPlaying: sonos.NowPlaying{Track: 1, Title: "Nightcall"}})
	var out bytes.Buffer

	exitCode := run([]string{"filter", ""}, &out)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !json.Valid(out.Bytes()) || !strings.Contains(out.String(), "Nightcall") {
		t.Errorf("stdout = %s, want Script Filter JSON with the now-playing row", out.String())
	}
}
