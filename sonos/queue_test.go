package sonos_test

import (
	"context"
	"testing"

	"sonos-alfred/sonos"
)

const diningRoomUUID = "RINCON_11111111111101400"

func TestReplaceAndPlayReplacesTheQueueWithTheItemAndPlaysIt(t *testing.T) {
	addAlbum := recorded(t, "AddURIToQueue", 0)
	args := argsOf(t, addAlbum.Request) // InstanceID, EnqueuedURI, EnqueuedURIMetaData, ...
	album := sonos.Item{URI: args[1].Value, Metadata: args[2].Value}
	speaker := speakerReplaying(t, avTransportPath, avTransportURN,
		recorded(t, "RemoveAllTracksFromQueue", 0),
		addAlbum,
		recorded(t, "SetAVTransportURI", 0), // x-rincon-queue:<uuid>#0
		recorded(t, "Play", 0),
	)

	if err := sonos.NewClient(speaker.URL).ReplaceAndPlay(context.Background(), diningRoomUUID, album); err != nil {
		t.Fatal(err)
	}
}

func TestReplaceAndPlayPlaysAStreamDirectlyWithoutTouchingTheQueue(t *testing.T) {
	setStream := recorded(t, "SetAVTransportURI", 1) // x-rincon-mp3radio://ice1.somafm.com/..., empty metadata
	args := argsOf(t, setStream.Request)             // InstanceID, CurrentURI, CurrentURIMetaData
	stream := sonos.Item{URI: args[1].Value, Metadata: args[2].Value}
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, setStream, recorded(t, "Play", 1))

	if err := sonos.NewClient(speaker.URL).ReplaceAndPlay(context.Background(), diningRoomUUID, stream); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueAtEndAppendsTheItem(t *testing.T) {
	appendAlbum := recorded(t, "AddURIToQueue", 1) // EnqueueAsNext=0, DesiredFirstTrackNumberEnqueued=0
	args := argsOf(t, appendAlbum.Request)
	nier := sonos.Item{URI: args[1].Value, Metadata: args[2].Value}
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, appendAlbum)

	if err := sonos.NewClient(speaker.URL).EnqueueAtEnd(context.Background(), nier); err != nil {
		t.Fatal(err)
	}
}
