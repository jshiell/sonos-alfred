package sonos

import (
	"context"
	"encoding/xml"
	"strconv"
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

const browsePageSize = 100

// browse returns every child of objectID, following the speaker's paging.
func (c *Client) browse(ctx context.Context, objectID string) ([]Item, error) {
	var all []Item
	for {
		values, err := c.Call(ctx, ContentDirectory, "Browse",
			Arg{"ObjectID", objectID},
			Arg{"BrowseFlag", "BrowseDirectChildren"},
			Arg{"Filter", "*"},
			Arg{"StartingIndex", strconv.Itoa(len(all))},
			Arg{"RequestedCount", strconv.Itoa(browsePageSize)},
			Arg{"SortCriteria", ""})
		if err != nil {
			return nil, err
		}
		page, err := parseDIDL(values["Result"])
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		total, _ := strconv.Atoi(values["TotalMatches"])
		if len(page) == 0 || len(all) >= total {
			return all, nil
		}
	}
}

// didlEntry is an <item> or a <container>: the speaker lists tracks and favorites as items, playlists as containers.
type didlEntry struct {
	ID    string `xml:"id,attr"`
	Title string `xml:"title"`
	Res   string `xml:"res"`
	ResMD string `xml:"resMD"`
}

func parseDIDL(didl string) ([]Item, error) {
	var parsed struct {
		Items      []didlEntry `xml:"item"`
		Containers []didlEntry `xml:"container"`
	}
	if err := xml.Unmarshal([]byte(didl), &parsed); err != nil {
		return nil, err
	}
	entries := append(parsed.Items, parsed.Containers...)
	items := make([]Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, Item{ID: entry.ID, Title: entry.Title, URI: entry.Res, Metadata: entry.ResMD})
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
