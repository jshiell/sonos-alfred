package sonos_test

import (
	"context"
	"net/http"
	"testing"

	"sonos-alfred/internal/fakespeaker"
	"sonos-alfred/sonos"
)

const (
	getTransportInfoRequest  = `<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:GetTransportInfo xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID></u:GetTransportInfo></s:Body></s:Envelope>`
	getTransportInfoResponse = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:GetTransportInfoResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><CurrentTransportState>PLAYING</CurrentTransportState><CurrentTransportStatus>OK</CurrentTransportStatus><CurrentSpeed>1</CurrentSpeed></u:GetTransportInfoResponse></s:Body></s:Envelope>`
)

func TestCallSendsSOAPRequestAndParsesResponseValues(t *testing.T) {
	speaker := fakespeaker.New(t, fakespeaker.Exchange{
		Path:       "/MediaRenderer/AVTransport/Control",
		SOAPAction: `"urn:schemas-upnp-org:service:AVTransport:1#GetTransportInfo"`,
		Body:       getTransportInfoRequest,
		Respond:    fakespeaker.Response{Status: http.StatusOK, Body: getTransportInfoResponse},
	})
	client := sonos.NewClient(speaker.URL)

	values, err := client.Call(context.Background(), sonos.AVTransport, "GetTransportInfo", sonos.Arg{Name: "InstanceID", Value: "0"})

	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"CurrentTransportState": "PLAYING", "CurrentTransportStatus": "OK", "CurrentSpeed": "1"}
	for name, wantValue := range want {
		if values[name] != wantValue {
			t.Errorf("%s = %q, want %q", name, values[name], wantValue)
		}
	}
}
