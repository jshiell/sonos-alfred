// THROWAWAY SPIKE — generic SOAP caller for probing speakers. Deleted once fixtures are captured.
//
//	go run ./spikes/soap <ip> <Service> <Action> [Key=Value ...]
//
// Prints the exact request body, HTTP status and raw response body. Set OUT=<file> to also save the response.
package main

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var services = map[string]struct{ path, urn string }{
	"AVTransport":          {"/MediaRenderer/AVTransport/Control", "urn:schemas-upnp-org:service:AVTransport:1"},
	"RenderingControl":     {"/MediaRenderer/RenderingControl/Control", "urn:schemas-upnp-org:service:RenderingControl:1"},
	"GroupRenderingControl": {"/MediaRenderer/GroupRenderingControl/Control", "urn:schemas-upnp-org:service:GroupRenderingControl:1"},
	"ContentDirectory":     {"/MediaServer/ContentDirectory/Control", "urn:schemas-upnp-org:service:ContentDirectory:1"},
	"ZoneGroupTopology":    {"/ZoneGroupTopology/Control", "urn:schemas-upnp-org:service:ZoneGroupTopology:1"},
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: soap <ip> <Service> <Action> [Key=Value ...]")
		os.Exit(2)
	}
	ip, svcName, action := os.Args[1], os.Args[2], os.Args[3]
	svc, ok := services[svcName]
	if !ok {
		fmt.Println("unknown service", svcName)
		os.Exit(2)
	}
	var args strings.Builder
	for _, kv := range os.Args[4:] {
		k, v, _ := strings.Cut(kv, "=")
		fmt.Fprintf(&args, "<%s>%s</%s>", k, html.EscapeString(v), k)
	}
	body := `<?xml version="1.0" encoding="utf-8"?>` +
		`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">` +
		`<s:Body><u:` + action + ` xmlns:u="` + svc.urn + `">` + args.String() + `</u:` + action + `></s:Body></s:Envelope>`

	req, _ := http.NewRequest("POST", "http://"+ip+":1400"+svc.path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	req.Header.Set("SOAPACTION", `"`+svc.urn+`#`+action+`"`)
	fmt.Println(">>> REQUEST", svcName, action)
	fmt.Println(body)
	t := time.Now()
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		fmt.Println("request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	fmt.Printf("<<< HTTP %d in %v (%d bytes)\n%s\n", resp.StatusCode, time.Since(t), len(out), out)
	if f := os.Getenv("OUT"); f != "" {
		_ = os.WriteFile(f, out, 0o644)
	}
}
