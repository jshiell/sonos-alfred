package sonos

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
)

// NowPlaying describes the current track of a group.
type NowPlaying struct {
	Track  int // 1-based queue position; 0 when nothing is loaded
	Title  string
	Artist string
	Album  string
}

// NowPlaying reads the group's current track.
func (c *Client) NowPlaying(ctx context.Context) (NowPlaying, error) {
	values, err := c.Call(ctx, AVTransport, "GetPositionInfo", Arg{"InstanceID", "0"})
	if err != nil {
		return NowPlaying{}, err
	}
	track, _ := strconv.Atoi(values["Track"])
	if values["TrackMetaData"] == "" {
		return NowPlaying{Track: track}, nil
	}

	var didl struct {
		Title  string `xml:"item>title"`
		Artist string `xml:"item>creator"`
		Album  string `xml:"item>album"`
	}
	err = xml.Unmarshal([]byte(values["TrackMetaData"]), &didl)
	if errors.Is(err, io.EOF) { // no XML at all, like the NOT_IMPLEMENTED that AirPlay reports
		return NowPlaying{Track: track}, nil
	}
	if err != nil {
		return NowPlaying{}, err
	}
	return NowPlaying{Track: track, Title: didl.Title, Artist: didl.Artist, Album: didl.Album}, nil
}
