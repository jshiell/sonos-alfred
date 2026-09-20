package sonos

import (
	"context"
	"strconv"
)

// GroupVolume reads the group's volume (0-100).
func (c *Client) GroupVolume(ctx context.Context) (int, error) {
	values, err := c.Call(ctx, GroupRenderingControl, "GetGroupVolume", Arg{"InstanceID", "0"})
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(values["CurrentVolume"])
}

// SetGroupVolume sets the group's volume (0-100).
func (c *Client) SetGroupVolume(ctx context.Context, volume int) error {
	_, err := c.Call(ctx, GroupRenderingControl, "SetGroupVolume",
		Arg{"InstanceID", "0"}, Arg{"DesiredVolume", strconv.Itoa(volume)})
	return err
}
