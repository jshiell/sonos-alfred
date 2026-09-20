package sonos_test

import (
	"context"
	"net/http"
	"testing"

	"sonos-alfred/internal/fakespeaker"
	"sonos-alfred/sonos"
)

func TestPlaySendsPlayToTheSpeaker(t *testing.T) {
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, recorded(t, "Play", 0))

	if err := sonos.NewClient(speaker.URL).Play(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPauseSendsPauseToTheSpeaker(t *testing.T) {
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, recorded(t, "Pause", 0))

	if err := sonos.NewClient(speaker.URL).Pause(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// avTransportSpeaker expects one AVTransport action with the given argument XML and answers with an empty response.
// Used where no exchange was recorded; the request shape comes from the svrooij AVTransport docs.
func avTransportSpeaker(t *testing.T, action, argsXML string) *fakespeaker.Speaker {
	t.Helper()
	const envelopeStart = `<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body>`
	return fakespeaker.New(t, fakespeaker.Exchange{
		Path:       avTransportPath,
		SOAPAction: `"` + avTransportURN + "#" + action + `"`,
		Body:       envelopeStart + `<u:` + action + ` xmlns:u="` + avTransportURN + `">` + argsXML + `</u:` + action + `></s:Body></s:Envelope>`,
		Respond: fakespeaker.Response{Status: http.StatusOK, Body: `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:` +
			action + `Response xmlns:u="` + avTransportURN + `"></u:` + action + `Response></s:Body></s:Envelope>`},
	})
}

func TestNextSendsNextToTheSpeaker(t *testing.T) {
	speaker := avTransportSpeaker(t, "Next", "<InstanceID>0</InstanceID>")

	if err := sonos.NewClient(speaker.URL).Next(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPreviousSendsPreviousToTheSpeaker(t *testing.T) {
	speaker := avTransportSpeaker(t, "Previous", "<InstanceID>0</InstanceID>")

	if err := sonos.NewClient(speaker.URL).Previous(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestTransportStateReportsPlaying(t *testing.T) {
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, recorded(t, "GetTransportInfo", 0))

	state, err := sonos.NewClient(speaker.URL).TransportState(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if state != sonos.StatePlaying {
		t.Errorf("state = %q, want %q", state, sonos.StatePlaying)
	}
}
