package sonos_test

import (
	"testing"

	"sonos-alfred/sonos"
)

// syntheticSSDPReply is modelled on the documented shape of a ZonePlayer M-SEARCH reply.
// It was NOT captured from a real speaker (the sandbox cannot receive multicast).
const syntheticSSDPReply = "HTTP/1.1 200 OK\r\n" +
	"CACHE-CONTROL: max-age = 1800\r\n" +
	"EXT:\r\n" +
	"LOCATION: http://192.0.2.34:1400/xml/device_description.xml\r\n" +
	"SERVER: Linux UPnP/1.0 Sonos/97.1-80312 (ZPS9)\r\n" +
	"ST: urn:schemas-upnp-org:device:ZonePlayer:1\r\n" +
	"USN: uuid:RINCON_22222222222201400::urn:schemas-upnp-org:device:ZonePlayer:1\r\n\r\n"

func TestParseSSDPReplyReturnsTheSpeakersHost(t *testing.T) {
	host, ok := sonos.ParseSSDPReply([]byte(syntheticSSDPReply))

	if !ok || host != "192.0.2.34" {
		t.Errorf("ParseSSDPReply = %q, %v; want 192.0.2.34, true", host, ok)
	}
}
