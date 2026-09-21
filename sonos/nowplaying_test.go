package sonos_test

import (
	"context"
	"net/http"
	"testing"

	"sonos-alfred/internal/fakespeaker"
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

func TestNowPlayingToleratesEmptyMetadata(t *testing.T) {
	// Recorded after the transport was emptied: Track 0, empty TrackURI and TrackMetaData.
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, recorded(t, "GetPositionInfo", 5))

	got, err := sonos.NewClient(speaker.URL).NowPlaying(context.Background())

	if err != nil {
		t.Fatalf("empty metadata should not be an error: %v", err)
	}
	if got != (sonos.NowPlaying{}) {
		t.Errorf("NowPlaying = %+v, want nothing playing", got)
	}
}

// AirPlay reports NOT_IMPLEMENTED in place of track metadata (a real capture from a speaker playing over AirPlay).
func TestNowPlayingToleratesMetadataThatIsNotXML(t *testing.T) {
	speaker := fakespeaker.New(t, fakespeaker.Exchange{
		Path:       avTransportPath,
		SOAPAction: `"` + avTransportURN + `#GetPositionInfo"`,
		Body:       recorded(t, "GetPositionInfo", 0).Request,
		Respond:    fakespeaker.Response{Status: http.StatusOK, Body: fixtureFile(t, "getpositioninfo-airplay.xml")},
	})

	got, err := sonos.NewClient(speaker.URL).NowPlaying(context.Background())

	if err != nil {
		t.Fatalf("AirPlay metadata should not be an error: %v", err)
	}
	if want := (sonos.NowPlaying{Track: 1}); got != want {
		t.Errorf("NowPlaying = %+v, want %+v", got, want)
	}
}
