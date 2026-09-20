package sonos_test

import (
	"context"
	"testing"

	"sonos-alfred/sonos"
)

func TestGroupVolumeReadsTheGroupsVolume(t *testing.T) {
	speaker := speakerReplaying(t, groupRenderingControlPath, groupRenderingControlURN, recorded(t, "GetGroupVolume", 0))

	volume, err := sonos.NewClient(speaker.URL).GroupVolume(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if volume != 12 {
		t.Errorf("volume = %d, want 12", volume)
	}
}
