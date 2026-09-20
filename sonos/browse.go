package sonos

import (
	"context"
	"encoding/xml"
)

// Item is an entry from the speaker's content directory: a favorite, a playlist or a queue track.
type Item struct {
	ID       string
	Title    string
	URI      string // what to play; empty when the item cannot be played directly (e.g. Sonos Radio shortcuts)
	Metadata string // DIDL describing the item, sent along with URI when playing or enqueuing
}

// Favorites lists the household's Sonos Favorites.
func (c *Client) Favorites(ctx context.Context) ([]Item, error) {
	return c.browse(ctx, "FV:2")
}

func (c *Client) browse(ctx context.Context, objectID string) ([]Item, error) {
	values, err := c.Call(ctx, ContentDirectory, "Browse",
		Arg{"ObjectID", objectID},
		Arg{"BrowseFlag", "BrowseDirectChildren"},
		Arg{"Filter", "*"},
		Arg{"StartingIndex", "0"},
		Arg{"RequestedCount", "100"},
		Arg{"SortCriteria", ""})
	if err != nil {
		return nil, err
	}
	return parseDIDL(values["Result"])
}

func parseDIDL(didl string) ([]Item, error) {
	var parsed struct {
		Items []struct {
			ID    string `xml:"id,attr"`
			Title string `xml:"title"`
			Res   string `xml:"res"`
			ResMD string `xml:"resMD"`
		} `xml:"item"`
	}
	if err := xml.Unmarshal([]byte(didl), &parsed); err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(parsed.Items))
	for _, it := range parsed.Items {
		items = append(items, Item{ID: it.ID, Title: it.Title, URI: it.Res, Metadata: it.ResMD})
	}
	return items, nil
}

// Playlists lists the household's Sonos playlists.
func (c *Client) Playlists(ctx context.Context) ([]Item, error) {
	return c.browse(ctx, "SQ:")
}

// Queue lists the tracks in the group's queue, in play order.
func (c *Client) Queue(ctx context.Context) ([]Item, error) {
	return c.browse(ctx, "Q:0")
}
