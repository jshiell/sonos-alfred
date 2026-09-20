package state_test

import (
	"testing"

	"sonos-alfred/state"
)

func TestActiveGroupSurvivesAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	if err := state.NewSettings(dir).SetActiveGroup("RINCON_11111111111101400"); err != nil {
		t.Fatal(err)
	}

	got := state.NewSettings(dir).ActiveGroup()

	if got != "RINCON_11111111111101400" {
		t.Errorf("ActiveGroup = %q, want the stored player UUID", got)
	}
}
