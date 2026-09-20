// THROWAWAY SPIKE S1 — SSDP discovery + GetZoneGroupState. Deleted once fixtures are captured.
package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const msearch = "M-SEARCH * HTTP/1.1\r\n" +
	"HOST: 239.255.255.250:1900\r\n" +
	"MAN: \"ssdp:discover\"\r\n" +
	"MX: 1\r\n" +
	"ST: urn:schemas-upnp-org:device:ZonePlayer:1\r\n\r\n"

const topologyBody = `<?xml version="1.0" encoding="utf-8"?>` +
	`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">` +
	`<s:Body><u:GetZoneGroupState xmlns:u="urn:schemas-upnp-org:service:ZoneGroupTopology:1"></u:GetZoneGroupState></s:Body></s:Envelope>`

func main() {
	outDir := "testdata/raw"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	_ = os.MkdirAll(outDir, 0o755)

	start := time.Now()
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		fmt.Println("listen:", err)
		os.Exit(1)
	}
	defer conn.Close()
	dst := &net.UDPAddr{IP: net.ParseIP("239.255.255.250"), Port: 1900}
	if _, err := conn.WriteTo([]byte(msearch), dst); err != nil {
		fmt.Println("SSDP send failed:", err)
		os.Exit(1)
	}

	hosts := map[string]bool{}
	buf := make([]byte, 4096)
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	first := time.Duration(0)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			break
		}
		if first == 0 {
			first = time.Since(start)
		}
		ip := addr.(*net.UDPAddr).IP.String()
		if !hosts[ip] {
			hosts[ip] = true
			fmt.Printf("SSDP reply from %s after %v\n%s\n", ip, time.Since(start), strings.TrimSpace(string(buf[:n])))
		}
	}
	fmt.Printf("SSDP: %d speakers, first reply %v, window closed %v\n", len(hosts), first, time.Since(start))
	if len(hosts) == 0 {
		fmt.Println("no speakers found")
		os.Exit(2)
	}

	for ip := range hosts {
		req, _ := http.NewRequest("POST", "http://"+ip+":1400/ZoneGroupTopology/Control", bytes.NewBufferString(topologyBody))
		req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
		req.Header.Set("SOAPACTION", `"urn:schemas-upnp-org:service:ZoneGroupTopology:1#GetZoneGroupState"`)
		client := http.Client{Timeout: 5 * time.Second}
		t := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("%s: topology request failed: %v\n", ip, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("%s: GetZoneGroupState HTTP %d in %v (%d bytes)\n", ip, resp.StatusCode, time.Since(t), len(body))
		path := fmt.Sprintf("%s/zonegroupstate-%s.xml", outDir, strings.ReplaceAll(ip, ".", "_"))
		_ = os.WriteFile(path, body, 0o644)
		fmt.Println("  saved", path, "| members:", len(regexp.MustCompile(`&lt;ZoneGroupMember `).FindAll(body, -1)),
			"| Invisible:", strings.Count(string(body), "Invisible=&quot;1&quot;"),
			"| Satellite:", strings.Count(string(body), "&lt;Satellite "))
	}
}
