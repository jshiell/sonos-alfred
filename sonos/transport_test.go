package sonos_test

import (
	"context"
	"testing"

	"sonos-alfred/sonos"
)

func TestPlaySendsPlayToTheSpeaker(t *testing.T) {
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, recorded(t, "Play", 0))

	if err := sonos.NewClient(speaker.URL).Play(context.Background()); err != nil {
		t.Fatal(err)
	}
}
