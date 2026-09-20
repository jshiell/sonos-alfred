package sonos_test

import (
	"context"
	"html"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"sonos-alfred/internal/fakespeaker"
	"sonos-alfred/sonos"
)

func browseRequest(objectID string) string { return browseRequestFrom(objectID, 0) }

func browseRequestFrom(objectID string, startingIndex int) string {
	return `<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:Browse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><ObjectID>` +
		objectID + `</ObjectID><BrowseFlag>BrowseDirectChildren</BrowseFlag><Filter>*</Filter><StartingIndex>` + strconv.Itoa(startingIndex) +
		`</StartingIndex><RequestedCount>100</RequestedCount><SortCriteria></SortCriteria></u:Browse></s:Body></s:Envelope>`
}

func contentDirectorySpeaker(t *testing.T, objectID, responseFixture string) *fakespeaker.Speaker {
	t.Helper()
	return fakespeaker.New(t, fakespeaker.Exchange{
		Path:       "/MediaServer/ContentDirectory/Control",
		SOAPAction: `"urn:schemas-upnp-org:service:ContentDirectory:1#Browse"`,
		Body:       browseRequest(objectID),
		Respond:    fakespeaker.Response{Status: http.StatusOK, Body: fixtureFile(t, responseFixture)},
	})
}

func TestFavoritesListsPlayableAndShortcutFavorites(t *testing.T) {
	speaker := contentDirectorySpeaker(t, "FV:2", "browse-favorites-albums-and-shortcuts.xml")

	favorites, err := sonos.NewClient(speaker.URL).Favorites(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	titles := []string{"Discover Sonos Radio", "Marbles (20th Anniversary Edition - 2015 Remaster)", "Niero:Atlas Original Soundtrack", "Trending Now"}
	if len(favorites) != len(titles) {
		t.Fatalf("got %d favorites, want %d: %+v", len(favorites), len(titles), favorites)
	}
	for i, title := range titles {
		if favorites[i].Title != title {
			t.Errorf("favorite %d title = %q, want %q", i, favorites[i].Title, title)
		}
	}

	garbage := favorites[1]
	const wantURI = "x-rincon-cpcontainer:1004206calbum%3A9000000001?sid=204&flags=8300&sn=5"
	const wantMetadata = `<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns:r="urn:schemas-rinconnetworks-com:metadata-1-0/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"><item id="1004206calbum%3A9000000001" parentID="1004206calbum%3A9000000001" restricted="true"><dc:title>Marbles (20th Anniversary Edition - 2015 Remaster)</dc:title><upnp:class>object.container.album.musicAlbum</upnp:class><desc id="cdudn" nameSpace="urn:schemas-rinconnetworks-com:metadata-1-0/">SA_RINCON52231_X_#Svc52231-00000000-Token</desc></item></DIDL-Lite>`
	if garbage.URI != wantURI {
		t.Errorf("album URI = %q, want %q", garbage.URI, wantURI)
	}
	if garbage.Metadata != wantMetadata {
		t.Errorf("album metadata = %q, want %q", garbage.Metadata, wantMetadata)
	}
	if shortcut := favorites[0]; shortcut.URI != "" {
		t.Errorf("Sonos Radio shortcut URI = %q, want empty (not playable)", shortcut.URI)
	}
}

func TestPlaylistsIsEmptyWhenTheHouseholdHasNoSonosPlaylists(t *testing.T) {
	speaker := contentDirectorySpeaker(t, "SQ:", "browse-playlists-empty.xml")

	playlists, err := sonos.NewClient(speaker.URL).Playlists(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if len(playlists) != 0 {
		t.Errorf("got %d playlists, want none: %+v", len(playlists), playlists)
	}
}

func TestQueueListsTracksInOrder(t *testing.T) {
	speaker := contentDirectorySpeaker(t, "Q:0", "browse-queue-apple-music.xml")

	queue, err := sonos.NewClient(speaker.URL).Queue(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if len(queue) != 46 {
		t.Fatalf("got %d tracks, want 46", len(queue))
	}
	first := queue[0]
	if first.ID != "Q:0/1" || first.Title != "Sample Track 01 - O’Brien" {
		t.Errorf("first track = %q %q, want Q:0/1 %q", first.ID, first.Title, "Sample Track 01 - O’Brien")
	}
	if want := "x-sonos-http:song%3a9000000002.mp4?sid=204&flags=8232&sn=5"; first.URI != want {
		t.Errorf("first track URI = %q, want %q", first.URI, want)
	}
}

// queuePage builds a Browse response page of tracks "Track from+1" .. "Track to" out of total, in the real response shape.
func queuePage(from, to, total int) string {
	var didl strings.Builder
	didl.WriteString(`<DIDL-Lite xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/" xmlns:r="urn:schemas-rinconnetworks-com:metadata-1-0/" xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/">`)
	for n := from + 1; n <= to; n++ {
		didl.WriteString(`<item id="Q:0/` + strconv.Itoa(n) + `" parentID="Q:0" restricted="true"><res protocolInfo="x-sonos-http:*:*:*">x-sonos-http:song` + strconv.Itoa(n) + `.mp4</res><dc:title>Track ` + strconv.Itoa(n) + `</dc:title></item>`)
	}
	didl.WriteString(`</DIDL-Lite>`)
	return `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:BrowseResponse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><Result>` +
		html.EscapeString(didl.String()) + `</Result><NumberReturned>` + strconv.Itoa(to-from) + `</NumberReturned><TotalMatches>` + strconv.Itoa(total) +
		`</TotalMatches><UpdateID>1</UpdateID></u:BrowseResponse></s:Body></s:Envelope>`
}

func TestQueueFollowsPagingBeyondTheFirstHundredTracks(t *testing.T) {
	// The real Dining Room queue was 151 tracks; a single Browse returns at most 100.
	const soapAction = `"urn:schemas-upnp-org:service:ContentDirectory:1#Browse"`
	const path = "/MediaServer/ContentDirectory/Control"
	speaker := fakespeaker.New(t,
		fakespeaker.Exchange{Path: path, SOAPAction: soapAction, Body: browseRequestFrom("Q:0", 0),
			Respond: fakespeaker.Response{Status: http.StatusOK, Body: queuePage(0, 100, 151)}},
		fakespeaker.Exchange{Path: path, SOAPAction: soapAction, Body: browseRequestFrom("Q:0", 100),
			Respond: fakespeaker.Response{Status: http.StatusOK, Body: queuePage(100, 151, 151)}},
	)

	queue, err := sonos.NewClient(speaker.URL).Queue(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if len(queue) != 151 {
		t.Fatalf("got %d tracks, want 151", len(queue))
	}
	if queue[0].Title != "Track 1" || queue[100].Title != "Track 101" || queue[150].Title != "Track 151" {
		t.Errorf("tracks out of order: first %q, 101st %q, last %q", queue[0].Title, queue[100].Title, queue[150].Title)
	}
}
