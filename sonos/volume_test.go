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

func TestSetGroupVolumeSendsTheDesiredVolume(t *testing.T) {
	speaker := speakerReplaying(t, groupRenderingControlPath, groupRenderingControlURN, recorded(t, "SetGroupVolume", 0))

	if err := sonos.NewClient(speaker.URL).SetGroupVolume(context.Background(), 12); err != nil {
		t.Fatal(err)
	}
}

func TestChangeGroupVolumeReturnsTheNewVolume(t *testing.T) {
	cases := []struct {
		name       string
		recordedN  int
		adjustment int
		wantVolume int
	}{
		{"up by two", 0, 2, 14},
		{"down by two", 1, -2, 12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			speaker := speakerReplaying(t, groupRenderingControlPath, groupRenderingControlURN, recorded(t, "SetRelativeGroupVolume", tc.recordedN))

			got, err := sonos.NewClient(speaker.URL).ChangeGroupVolume(context.Background(), tc.adjustment)

			if err != nil {
				t.Fatal(err)
			}
			if got != tc.wantVolume {
				t.Errorf("new volume = %d, want %d", got, tc.wantVolume)
			}
		})
	}
}
