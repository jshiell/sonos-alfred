package sonos

import (
	"context"
	"time"
)

// Backend is what the workflow needs from a speaker, so the UPnP client can be swapped for another protocol.
// It talks to one speaker: for a group, that speaker must be the group's coordinator.
type Backend interface {
	Topology(ctx context.Context) (Topology, error)
	NowPlaying(ctx context.Context) (NowPlaying, error)
	PlayingFromQueue(ctx context.Context) (bool, error)
	GroupVolume(ctx context.Context) (int, error)
	PlayMode(ctx context.Context) (PlayMode, error)
	SleepTimerRemaining(ctx context.Context) (time.Duration, error)
	Favorites(ctx context.Context) ([]Item, error)
	Playlists(ctx context.Context) ([]Item, error)
	Queue(ctx context.Context) ([]Item, error)
}

var _ Backend = (*Client)(nil)
