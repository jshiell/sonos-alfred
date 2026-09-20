package sonos

import (
	"bufio"
	"bytes"
	"net/http"
	"net/url"
)

// ParseSSDPReply extracts the speaker's host from an SSDP M-SEARCH reply.
func ParseSSDPReply(datagram []byte) (host string, ok bool) {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(datagram)), nil)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil || location.Hostname() == "" {
		return "", false
	}
	return location.Hostname(), true
}
