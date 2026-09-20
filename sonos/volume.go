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
