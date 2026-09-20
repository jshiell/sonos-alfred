package sonos

import (
	"context"
	"strings"
)

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
	if _, err := c.Call(ctx, AVTransport, "AddURIToQueue",
		Arg{"InstanceID", "0"},
		Arg{"EnqueuedURI", item.URI},
		Arg{"EnqueuedURIMetaData", item.Metadata},
		Arg{"DesiredFirstTrackNumberEnqueued", "0"},
		Arg{"EnqueueAsNext", "0"}); err != nil {
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
