package sonos_test

import (
	"context"
	"errors"
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

func TestEnqueueNextInsertsAfterTheCurrentTrack(t *testing.T) {
	onQueue := recorded(t, "GetMediaInfo", 3)         // CurrentURI x-rincon-queue:<uuid>#0
	currentTrack := recorded(t, "GetPositionInfo", 1) // Track 1
	insertAfterOne := recorded(t, "AddURIToQueue", 2) // EnqueueAsNext=1, DesiredFirstTrackNumberEnqueued=2
	args := argsOf(t, insertAfterOne.Request)
	nier := sonos.Item{URI: args[1].Value, Metadata: args[2].Value}
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, onQueue, currentTrack, insertAfterOne)

	if err := sonos.NewClient(speaker.URL).EnqueueNext(context.Background(), nier); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueNextAppendsWhenTheGroupIsPlayingAStream(t *testing.T) {
	onStream := recorded(t, "GetMediaInfo", 0) // CurrentURI x-rincon-mp3radio://...: no queue position
	appendAlbum := recorded(t, "AddURIToQueue", 1)
	args := argsOf(t, appendAlbum.Request)
	nier := sonos.Item{URI: args[1].Value, Metadata: args[2].Value}
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, onStream, appendAlbum)

	if err := sonos.NewClient(speaker.URL).EnqueueNext(context.Background(), nier); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueReportsItemsTheSpeakerRefusesAsNotEnqueueable(t *testing.T) {
	refused := recorded(t, "AddURIToQueue", 5) // UPnP error 800
	args := argsOf(t, refused.Request)
	item := sonos.Item{URI: args[1].Value, Metadata: args[2].Value}
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, refused)

	err := sonos.NewClient(speaker.URL).EnqueueAtEnd(context.Background(), item)

	if !errors.Is(err, sonos.ErrNotEnqueueable) {
		t.Errorf("error = %v, want sonos.ErrNotEnqueueable", err)
	}
}
