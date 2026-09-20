package sonos

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SetSleepTimer stops playback after d.
func (c *Client) SetSleepTimer(ctx context.Context, d time.Duration) error {
	seconds := int(d.Seconds())
	formatted := fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds%3600/60, seconds%60)
	_, err := c.Call(ctx, AVTransport, "ConfigureSleepTimer", Arg{"InstanceID", "0"}, Arg{"NewSleepTimerDuration", formatted})
	return err
}

// CancelSleepTimer clears any sleep timer.
func (c *Client) CancelSleepTimer(ctx context.Context) error {
	_, err := c.Call(ctx, AVTransport, "ConfigureSleepTimer", Arg{"InstanceID", "0"}, Arg{"NewSleepTimerDuration", ""})
	return err
}

// SleepTimerRemaining reports how long until the sleep timer fires; zero when none is set.
func (c *Client) SleepTimerRemaining(ctx context.Context) (time.Duration, error) {
	values, err := c.Call(ctx, AVTransport, "GetRemainingSleepTimerDuration", Arg{"InstanceID", "0"})
	if err != nil {
		return 0, err
	}
	remaining := values["RemainingSleepTimerDuration"]
	if remaining == "" {
		return 0, nil
	}
	var hours, minutes, seconds int
	if _, err := fmt.Sscanf(remaining, "%d:%d:%d", &hours, &minutes, &seconds); err != nil {
		return 0, fmt.Errorf("unexpected sleep timer %q: %w", strings.TrimSpace(remaining), err)
	}
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second, nil
}
