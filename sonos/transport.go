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

// TransportState is what a group is currently doing.
type TransportState string

// StatePlaying is reported while audio is playing.
const StatePlaying TransportState = "PLAYING"

// TransportState reads whether the group is playing, paused or stopped.
func (c *Client) TransportState(ctx context.Context) (TransportState, error) {
	values, err := c.Call(ctx, AVTransport, "GetTransportInfo", Arg{"InstanceID", "0"})
	if err != nil {
		return "", err
	}
	return TransportState(values["CurrentTransportState"]), nil
}
