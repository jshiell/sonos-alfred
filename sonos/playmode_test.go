package sonos_test

import (
	"context"
	"testing"

	"sonos-alfred/sonos"
)

func TestPlayModeEncodesAndDecodesAllSixWireValues(t *testing.T) {
	cases := []struct {
		wire string
		want sonos.PlayMode
	}{
		{"NORMAL", sonos.PlayMode{Shuffle: false, Repeat: sonos.RepeatOff}},
		{"REPEAT_ALL", sonos.PlayMode{Shuffle: false, Repeat: sonos.RepeatAll}},
		{"REPEAT_ONE", sonos.PlayMode{Shuffle: false, Repeat: sonos.RepeatOne}},
		{"SHUFFLE_NOREPEAT", sonos.PlayMode{Shuffle: true, Repeat: sonos.RepeatOff}},
		{"SHUFFLE", sonos.PlayMode{Shuffle: true, Repeat: sonos.RepeatAll}}, // SHUFFLE means shuffle + repeat all
		{"SHUFFLE_REPEAT_ONE", sonos.PlayMode{Shuffle: true, Repeat: sonos.RepeatOne}},
	}
	for _, tc := range cases {
		t.Run(tc.wire, func(t *testing.T) {
			got, err := sonos.ParsePlayMode(tc.wire)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("ParsePlayMode(%q) = %+v, want %+v", tc.wire, got, tc.want)
			}
			if encoded := tc.want.String(); encoded != tc.wire {
				t.Errorf("%+v encodes as %q, want %q", tc.want, encoded, tc.wire)
			}
		})
	}
}

func TestPlayModeReadsTheCurrentModeFromTheSpeaker(t *testing.T) {
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, recorded(t, "GetTransportSettings", 0)) // SHUFFLE

	got, err := sonos.NewClient(speaker.URL).PlayMode(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if want := (sonos.PlayMode{Shuffle: true, Repeat: sonos.RepeatAll}); got != want {
		t.Errorf("PlayMode = %+v, want %+v", got, want)
	}
}

func mode(t *testing.T, wire string) sonos.PlayMode {
	t.Helper()
	parsed, err := sonos.ParsePlayMode(wire)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestToggleShuffleKeepsTheRepeatSetting(t *testing.T) {
	cases := []struct{ from, to string }{
		{"NORMAL", "SHUFFLE_NOREPEAT"},
		{"SHUFFLE_NOREPEAT", "NORMAL"},
		{"SHUFFLE", "REPEAT_ALL"},
		{"REPEAT_ONE", "SHUFFLE_REPEAT_ONE"},
	}
	for _, tc := range cases {
		t.Run(tc.from, func(t *testing.T) {
			if got := mode(t, tc.from).ToggleShuffle(); got != mode(t, tc.to) {
				t.Errorf("ToggleShuffle(%s) = %s, want %s", tc.from, got, tc.to)
			}
		})
	}
}
