package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sonos-alfred/app"
	"sonos-alfred/hub"
	"sonos-alfred/sonos"
	"sonos-alfred/state"
)

// runCommand runs the command line against an empty workflow data directory, as Alfred would.
func runCommand(t *testing.T, args ...string) (exitCode int, stdout string) {
	t.Helper()
	t.Setenv("alfred_workflow_data", t.TempDir())
	var out bytes.Buffer
	exitCode = run(args, &out)
	return exitCode, out.String()
}

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

func TestFilterThatFailsStillPrintsAnErrorRowAndExitsZero(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("alfred_workflow_data", filepath.Join(file, "data")) // no directory can be made inside a file
	var out bytes.Buffer

	exitCode := run([]string{"filter", ""}, &out)

	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0: Alfred shows nothing for a Script Filter that fails", exitCode)
	}
	if !json.Valid(out.Bytes()) || !strings.Contains(out.String(), "Sonos hit a problem") {
		t.Errorf("stdout = %s, want Script Filter JSON with the problem row", out.String())
	}
}

func TestDoThatFailsPrintsWhyForTheNotificationAndExitsNonZero(t *testing.T) {
	exitCode, stdout := runCommand(t, "do", hub.Encode(hub.Action{Verb: "playpause"})) // nothing is cached yet, so there is no speaker to tell

	if exitCode == 0 {
		t.Error("exit code = 0, want non-zero")
	}
	if strings.TrimSpace(stdout) == "" {
		t.Error("stdout is empty, want the reason: the notification shows what do prints")
	}
}
