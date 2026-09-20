package sonos_test

import (
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
