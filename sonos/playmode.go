package sonos

import (
	"context"
	"fmt"
)

// Repeat is how the queue repeats.
type Repeat int

// The repeat settings.
const (
	RepeatOff Repeat = iota
	RepeatAll
	RepeatOne
)

// PlayMode is shuffle and repeat. On the wire it is a single value: NORMAL, REPEAT_ALL, REPEAT_ONE,
// SHUFFLE_NOREPEAT, SHUFFLE (which is shuffle with repeat all) or SHUFFLE_REPEAT_ONE.
type PlayMode struct {
	Shuffle bool
	Repeat  Repeat
}

var playModeWire = map[PlayMode]string{
	{false, RepeatOff}: "NORMAL",
	{false, RepeatAll}: "REPEAT_ALL",
	{false, RepeatOne}: "REPEAT_ONE",
	{true, RepeatOff}:  "SHUFFLE_NOREPEAT",
	{true, RepeatAll}:  "SHUFFLE",
	{true, RepeatOne}:  "SHUFFLE_REPEAT_ONE",
}

// ParsePlayMode decodes a wire value.
func ParsePlayMode(wire string) (PlayMode, error) {
	for mode, w := range playModeWire {
		if w == wire {
			return mode, nil
		}
	}
	return PlayMode{}, fmt.Errorf("unknown play mode %q", wire)
}

// String is the wire value.
func (m PlayMode) String() string { return playModeWire[m] }

// PlayMode reads the group's shuffle and repeat settings.
func (c *Client) PlayMode(ctx context.Context) (PlayMode, error) {
	values, err := c.Call(ctx, AVTransport, "GetTransportSettings", Arg{"InstanceID", "0"})
	if err != nil {
		return PlayMode{}, err
	}
	return ParsePlayMode(values["PlayMode"])
}

// ToggleShuffle flips shuffle and keeps the repeat setting.
func (m PlayMode) ToggleShuffle() PlayMode {
	m.Shuffle = !m.Shuffle
	return m
}
