package sonos

import (
	"context"
	"fmt"
	"net"
	"time"
)

const (
	ssdpAddress      = "239.255.255.250:1900"
	discoveryTimeout = 3 * time.Second
)

// Discover asks the network for Sonos speakers by SSDP multicast and returns the host of the first to answer.
// It is not unit-tested: multicast can't be received inside the test sandbox, so it is checked by hand on the network.
func Discover(ctx context.Context) (string, error) {
	group, err := net.ResolveUDPAddr("udp4", ssdpAddress)
	if err != nil {
		return "", err
	}
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	deadline := time.Now().Add(discoveryTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return "", err
	}

	search := "M-SEARCH * HTTP/1.1\r\nHOST: " + ssdpAddress + "\r\nMAN: \"ssdp:discover\"\r\nMX: 1\r\n" +
		"ST: urn:schemas-upnp-org:device:ZonePlayer:1\r\n\r\n"
	for range 2 { // UDP can drop a datagram
		if _, err := conn.WriteTo([]byte(search), group); err != nil {
			return "", err
		}
	}

	reply := make([]byte, 4096)
	for {
		n, _, err := conn.ReadFrom(reply)
		if err != nil {
			return "", fmt.Errorf("no Sonos speaker answered: %w", err)
		}
		if host, ok := ParseSSDPReply(reply[:n]); ok {
			return host, nil
		}
	}
}
