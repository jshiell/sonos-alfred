package fakespeaker_test

import (
	"fmt"
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

func TestRejectsRequestsThatDoNotMatchTheScript(t *testing.T) {
	const pauseAction = `"urn:schemas-upnp-org:service:AVTransport:1#Pause"`
	speedThenInstance := strings.Replace(playRequest, "<InstanceID>0</InstanceID><Speed>1</Speed>", "<Speed>1</Speed><InstanceID>0</InstanceID>", 1)
	cases := []struct {
		name       string
		script     []fakespeaker.Exchange
		path       string
		soapAction string
		body       string
	}{
		{"wrong path", []fakespeaker.Exchange{playExchange()}, "/MediaRenderer/RenderingControl/Control", playAction, playRequest},
		{"wrong action", []fakespeaker.Exchange{playExchange()}, avTransportPath, pauseAction, playRequest},
		{"wrong argument value", []fakespeaker.Exchange{playExchange()}, avTransportPath, playAction, strings.Replace(playRequest, "<Speed>1</Speed>", "<Speed>2</Speed>", 1)},
		{"arguments in the wrong order", []fakespeaker.Exchange{playExchange()}, avTransportPath, playAction, speedThenInstance},
		{"call after the script is exhausted", nil, avTransportPath, playAction, playRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			speaker := fakespeaker.NewUnchecked(tc.script...)
			defer speaker.Close()

			status, _ := post(t, speaker.URL+tc.path, tc.soapAction, tc.body)

			if status != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500", status)
			}
			if got := len(speaker.Problems()); got != 1 {
				t.Errorf("recorded %d problems, want 1: %v", got, speaker.Problems())
			}
		})
	}
}

func playExchange() fakespeaker.Exchange {
	return fakespeaker.Exchange{
		Path:       avTransportPath,
		SOAPAction: playAction,
		Body:       playRequest,
		Respond:    fakespeaker.Response{Status: http.StatusOK, Body: playResponse},
	}
}

func TestVerifyReportsScriptedExchangesThatNeverHappened(t *testing.T) {
	speaker := fakespeaker.NewUnchecked(playExchange())
	defer speaker.Close()

	if got := len(speaker.Verify()); got != 1 {
		t.Fatalf("before any request: %d problems, want 1 (the pending Play): %v", got, speaker.Verify())
	}

	post(t, speaker.URL+avTransportPath, playAction, playRequest)

	if problems := speaker.Verify(); len(problems) != 0 {
		t.Errorf("after the Play request: unexpected problems %v", problems)
	}
}

// recordingT stands in for *testing.T so the test can observe what New reports at cleanup.
type recordingT struct {
	testing.TB
	errors   []string
	cleanups []func()
}

func (r *recordingT) Helper() {}

func (r *recordingT) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *recordingT) Cleanup(f func()) { r.cleanups = append(r.cleanups, f) }

func (r *recordingT) runCleanups() {
	for i := len(r.cleanups) - 1; i >= 0; i-- {
		r.cleanups[i]()
	}
}

func TestNewFailsTheTestAtCleanupWhenTheScriptWasNotFollowed(t *testing.T) {
	rt := &recordingT{}
	fakespeaker.New(rt, playExchange())

	rt.runCleanups()

	if len(rt.errors) != 1 {
		t.Errorf("reported %d errors, want 1 (the Play that never happened): %v", len(rt.errors), rt.errors)
	}
}
