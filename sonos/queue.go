package sonos

import "context"

// ReplaceAndPlay clears the queue, enqueues item and starts playing it.
// coordinatorUUID is the UUID of the speaker this client talks to.
func (c *Client) ReplaceAndPlay(ctx context.Context, coordinatorUUID string, item Item) error {
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
