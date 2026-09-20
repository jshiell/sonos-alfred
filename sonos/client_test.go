package sonos_test

import (
	"context"
	"errors"
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

func TestCallReturnsTypedFaultWithUPnPErrorCode(t *testing.T) {
	const seekRequest = `<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:Seek xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><Unit>TRACK_NR</Unit><Target>3</Target></u:Seek></s:Body></s:Envelope>`
	const fault701 = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><s:Fault><faultcode>s:Client</faultcode><faultstring>UPnPError</faultstring><detail><UPnPError xmlns="urn:schemas-upnp-org:control-1-0"><errorCode>701</errorCode></UPnPError></detail></s:Fault></s:Body></s:Envelope>`
	speaker := fakespeaker.New(t, fakespeaker.Exchange{
		Path:       "/MediaRenderer/AVTransport/Control",
		SOAPAction: `"urn:schemas-upnp-org:service:AVTransport:1#Seek"`,
		Body:       seekRequest,
		Respond:    fakespeaker.Response{Status: http.StatusInternalServerError, Body: fault701},
	})
	client := sonos.NewClient(speaker.URL)

	_, err := client.Call(context.Background(), sonos.AVTransport, "Seek",
		sonos.Arg{Name: "InstanceID", Value: "0"}, sonos.Arg{Name: "Unit", Value: "TRACK_NR"}, sonos.Arg{Name: "Target", Value: "3"})

	var fault *sonos.Fault
	if !errors.As(err, &fault) {
		t.Fatalf("error = %v, want a *sonos.Fault", err)
	}
	if fault.Code != 701 {
		t.Errorf("fault code = %d, want 701", fault.Code)
	}
}

func TestCallEscapesArgumentsExactlyAsTheSpeakerAccepted(t *testing.T) {
	addAlbum := recorded(t, "AddURIToQueue", 0)
	speaker := fakespeaker.New(t, fakespeaker.Exchange{
		Path:       "/MediaRenderer/AVTransport/Control",
		SOAPAction: `"urn:schemas-upnp-org:service:AVTransport:1#AddURIToQueue"`,
		Body:       addAlbum.Request,
		Respond:    fakespeaker.Response{Status: http.StatusOK, Body: addAlbum.Response},
	})
	client := sonos.NewClient(speaker.URL)

	values, err := client.Call(context.Background(), sonos.AVTransport, "AddURIToQueue", argsOf(t, addAlbum.Request)...)

	if err != nil {
		t.Fatal(err)
	}
	if values["NumTracksAdded"] != "12" {
		t.Errorf("NumTracksAdded = %q, want 12", values["NumTracksAdded"])
	}
}
