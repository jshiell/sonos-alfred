package sonos

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrNotEnqueueable is returned when the speaker refuses to add an item to the queue.
var ErrNotEnqueueable = errors.New("speaker refused to add the item to the queue")

// enqueueRefusedCode is the UPnP error seen when an item cannot be enqueued. Only one observation backs
// this (S2, a Sonos Radio URI), so treat it as "the speaker said no" rather than a precise diagnosis.
const enqueueRefusedCode = 800

// streamSchemes are URI schemes played directly rather than through the queue.
// Only x-rincon-mp3radio was verified against a real speaker; the others are unverified.
var streamSchemes = []string{"x-rincon-mp3radio:", "x-sonosapi-stream:", "x-sonosapi-radio:"}

func isStream(uri string) bool {
	for _, scheme := range streamSchemes {
		if strings.HasPrefix(uri, scheme) {
			return true
		}
	}
	return false
}

// ReplaceAndPlay clears the queue, enqueues item and starts playing it.
// coordinatorUUID is the UUID of the speaker this client talks to.
func (c *Client) ReplaceAndPlay(ctx context.Context, coordinatorUUID string, item Item) error {
	if isStream(item.URI) {
		if _, err := c.Call(ctx, AVTransport, "SetAVTransportURI",
			Arg{"InstanceID", "0"}, Arg{"CurrentURI", item.URI}, Arg{"CurrentURIMetaData", item.Metadata}); err != nil {
			return err
		}
		return c.Play(ctx)
	}
	if _, err := c.Call(ctx, AVTransport, "RemoveAllTracksFromQueue", Arg{"InstanceID", "0"}); err != nil {
		return err
	}
	if err := c.addToQueue(ctx, item, 0, false); err != nil {
		return err
	}
	if _, err := c.Call(ctx, AVTransport, "SetAVTransportURI",
		Arg{"InstanceID", "0"},
		Arg{"CurrentURI", "x-rincon-queue:" + coordinatorUUID + "#0"},
		Arg{"CurrentURIMetaData", ""}); err != nil {
		return err
	}
	return c.Play(ctx)
}

// EnqueueAtEnd appends item to the end of the queue.
func (c *Client) EnqueueAtEnd(ctx context.Context, item Item) error {
	return c.addToQueue(ctx, item, 0, false)
}

// addToQueue enqueues item at position (1-based; 0 means the end). With asNext the speaker inserts it
// at position; asNext with position 0 still appends, so "play next" must pass a real position.
func (c *Client) addToQueue(ctx context.Context, item Item, position int, asNext bool) error {
	enqueueAsNext := "0"
	if asNext {
		enqueueAsNext = "1"
	}
	_, err := c.Call(ctx, AVTransport, "AddURIToQueue",
		Arg{"InstanceID", "0"},
		Arg{"EnqueuedURI", item.URI},
		Arg{"EnqueuedURIMetaData", item.Metadata},
		Arg{"DesiredFirstTrackNumberEnqueued", strconv.Itoa(position)},
		Arg{"EnqueueAsNext", enqueueAsNext})
	var fault *Fault
	if errors.As(err, &fault) && fault.Code == enqueueRefusedCode {
		return fmt.Errorf("%w: %w", ErrNotEnqueueable, err)
	}
	return err
}

// EnqueueNext inserts item right after the track that is playing. It needs the current queue position
// because the speaker appends instead when asked to enqueue "as next" without one.
func (c *Client) EnqueueNext(ctx context.Context, item Item) error {
	media, err := c.Call(ctx, AVTransport, "GetMediaInfo", Arg{"InstanceID", "0"})
	if err != nil {
		return err
	}
	if !strings.HasPrefix(media["CurrentURI"], "x-rincon-queue:") {
		return c.EnqueueAtEnd(ctx, item) // not playing from the queue, so there is no "next" position
	}
	current, err := c.NowPlaying(ctx)
	if err != nil {
		return err
	}
	return c.addToQueue(ctx, item, current.Track+1, true)
}

// JumpToQueueTrack plays the queue from the given 1-based track. It selects the queue first because a
// bare Seek fails when the group is playing a stream.
func (c *Client) JumpToQueueTrack(ctx context.Context, coordinatorUUID string, track int) error {
	if _, err := c.Call(ctx, AVTransport, "SetAVTransportURI",
		Arg{"InstanceID", "0"},
		Arg{"CurrentURI", "x-rincon-queue:" + coordinatorUUID + "#0"},
		Arg{"CurrentURIMetaData", ""}); err != nil {
		return err
	}
	if _, err := c.Call(ctx, AVTransport, "Seek",
		Arg{"InstanceID", "0"}, Arg{"Unit", "TRACK_NR"}, Arg{"Target", strconv.Itoa(track)}); err != nil {
		return err
	}
	return c.Play(ctx)
}
