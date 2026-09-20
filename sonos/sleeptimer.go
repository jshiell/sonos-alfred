package sonos

import (
	"context"
	"fmt"
	"time"
)

// SetSleepTimer stops playback after d.
func (c *Client) SetSleepTimer(ctx context.Context, d time.Duration) error {
	seconds := int(d.Seconds())
	formatted := fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds%3600/60, seconds%60)
	_, err := c.Call(ctx, AVTransport, "ConfigureSleepTimer", Arg{"InstanceID", "0"}, Arg{"NewSleepTimerDuration", formatted})
	return err
}
