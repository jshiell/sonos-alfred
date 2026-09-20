package fakespeaker_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"sonos-alfred/internal/fakespeaker"
)

const (
	avTransportPath = "/MediaRenderer/AVTransport/Control"
	playAction      = `"urn:schemas-upnp-org:service:AVTransport:1#Play"`
	playRequest     = `<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:Play xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><Speed>1</Speed></u:Play></s:Body></s:Envelope>`
	playResponse    = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:PlayResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:PlayResponse></s:Body></s:Envelope>`
)

func post(t *testing.T, url, soapAction, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("SOAPACTION", soapAction)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(got)
}

func TestReplaysScriptedResponseForMatchingRequest(t *testing.T) {
	speaker := fakespeaker.NewUnchecked(fakespeaker.Exchange{
		Path:       avTransportPath,
		SOAPAction: playAction,
		Body:       playRequest,
		Respond:    fakespeaker.Response{Status: http.StatusOK, Body: playResponse},
	})
	defer speaker.Close()

	status, body := post(t, speaker.URL+avTransportPath, playAction, playRequest)

	if status != http.StatusOK || body != playResponse {
		t.Errorf("got %d %q, want 200 %q", status, body, playResponse)
	}
	if problems := speaker.Problems(); len(problems) != 0 {
		t.Errorf("unexpected problems: %v", problems)
	}
}
