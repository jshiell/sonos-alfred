// Package sonos controls Sonos speakers over their local UPnP/SOAP API.
package sonos

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
)

// Service identifies a UPnP service on a speaker.
type Service struct {
	path string
	urn  string
}

// The services this client talks to.
var (
	AVTransport = Service{"/MediaRenderer/AVTransport/Control", "urn:schemas-upnp-org:service:AVTransport:1"}
)

// Arg is one SOAP argument. Arguments are sent in the order given.
type Arg struct {
	Name  string
	Value string
}

// Client talks to one speaker.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client for the speaker at baseURL, e.g. "http://10.0.0.5:1400".
func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: http.DefaultClient}
}

// Call performs a SOAP action and returns the response's values by name.
func (c *Client) Call(ctx context.Context, service Service, action string, args ...Arg) (map[string]string, error) {
	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	body.WriteString(`<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body>`)
	body.WriteString(`<u:` + action + ` xmlns:u="` + service.urn + `">`)
	for _, arg := range args {
		body.WriteString("<" + arg.Name + ">" + arg.Value + "</" + arg.Name + ">")
	}
	body.WriteString(`</u:` + action + `></s:Body></s:Envelope>`)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+service.path, strings.NewReader(body.String()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	req.Header.Set("SOAPACTION", `"`+service.urn+"#"+action+`"`)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return parseResponseValues(resp.Body)
}

// parseResponseValues returns the text of each element directly inside the response element.
func parseResponseValues(r io.Reader) (map[string]string, error) {
	values := map[string]string{}
	decoder := xml.NewDecoder(r)
	depth := 0
	var current string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return values, nil
		}
		if err != nil {
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			depth++
			if depth == 4 {
				current = t.Name.Local
				values[current] = ""
			}
		case xml.CharData:
			if depth == 4 {
				values[current] += string(t)
			}
		case xml.EndElement:
			depth--
		}
	}
}
