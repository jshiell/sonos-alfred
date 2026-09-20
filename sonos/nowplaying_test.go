package sonos_test

import (
	"context"
	"testing"

	"sonos-alfred/sonos"
)

func TestNowPlayingReadsTheCurrentTrack(t *testing.T) {
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, recorded(t, "GetPositionInfo", 0))

	got, err := sonos.NewClient(speaker.URL).NowPlaying(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	want := sonos.NowPlaying{
		Track:  1,
		Title:  "Saxon (2015 - Remaster)",
		Artist: "Marbles",
		Album:  "Marbles (20th Anniversary Edition - 2015 Remaster)",
	}
	if got != want {
		t.Errorf("NowPlaying = %+v, want %+v", got, want)
	}
}
