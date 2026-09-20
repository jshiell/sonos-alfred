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
	TransportState(ctx context.Context) (TransportState, error)
	Play(ctx context.Context) error
	Pause(ctx context.Context) error
	EnqueueAtEnd(ctx context.Context, item Item) error
	EnqueueNext(ctx context.Context, item Item) error
	JumpToQueueTrack(ctx context.Context, coordinatorUUID string, track int) error
	ReplaceAndPlay(ctx context.Context, coordinatorUUID string, item Item) error
	Next(ctx context.Context) error
	Previous(ctx context.Context) error
	PlayingFromQueue(ctx context.Context) (bool, error)
	GroupVolume(ctx context.Context) (int, error)
	SetGroupVolume(ctx context.Context, volume int) error
	ChangeGroupVolume(ctx context.Context, adjustment int) (int, error)
	PlayMode(ctx context.Context) (PlayMode, error)
	SetPlayMode(ctx context.Context, mode PlayMode) error
	SetSleepTimer(ctx context.Context, d time.Duration) error
	CancelSleepTimer(ctx context.Context) error
	SleepTimerRemaining(ctx context.Context) (time.Duration, error)
	Favorites(ctx context.Context) ([]Item, error)
	Playlists(ctx context.Context) ([]Item, error)
	Queue(ctx context.Context) ([]Item, error)
}

var _ Backend = (*Client)(nil)
