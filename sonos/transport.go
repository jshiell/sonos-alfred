package sonos

import "context"

// Play starts or resumes playback on the group this client is talking to (its coordinator).
func (c *Client) Play(ctx context.Context) error {
	_, err := c.Call(ctx, AVTransport, "Play", Arg{"InstanceID", "0"}, Arg{"Speed", "1"})
	return err
}

// Pause pauses playback.
func (c *Client) Pause(ctx context.Context) error {
	_, err := c.Call(ctx, AVTransport, "Pause", Arg{"InstanceID", "0"})
	return err
}

// Next skips to the next track.
func (c *Client) Next(ctx context.Context) error {
	_, err := c.Call(ctx, AVTransport, "Next", Arg{"InstanceID", "0"})
	return err
}

// Previous returns to the previous track.
func (c *Client) Previous(ctx context.Context) error {
	_, err := c.Call(ctx, AVTransport, "Previous", Arg{"InstanceID", "0"})
	return err
}
