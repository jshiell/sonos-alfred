package sonos_test

import (
	"context"
	"net/http"
	"testing"

	"sonos-alfred/internal/fakespeaker"
	"sonos-alfred/sonos"
)

func browseRequest(objectID string) string {
	return `<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:Browse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><ObjectID>` +
		objectID + `</ObjectID><BrowseFlag>BrowseDirectChildren</BrowseFlag><Filter>*</Filter><StartingIndex>0</StartingIndex><RequestedCount>100</RequestedCount><SortCriteria></SortCriteria></u:Browse></s:Body></s:Envelope>`
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
