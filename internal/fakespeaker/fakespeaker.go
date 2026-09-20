// Package fakespeaker is a strict fake Sonos speaker for tests. It replays a scripted sequence of SOAP exchanges.
package fakespeaker

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

// Response is what the speaker answers with.
type Response struct {
	Status int
	Body   string
}

// Exchange is one expected request and the response to give.
type Exchange struct {
	Path       string
	SOAPAction string
	Body       string
	Respond    Response
}

// Speaker serves a script of exchanges over HTTP.
type Speaker struct {
	*httptest.Server

	mu       sync.Mutex
	script   []Exchange
	problems []string
}

// NewUnchecked starts a speaker that records problems but does not fail a test itself.
func NewUnchecked(script ...Exchange) *Speaker {
	s := &Speaker{script: script}
	s.Server = httptest.NewServer(http.HandlerFunc(s.serve))
	return s
}

// Problems lists everything the speaker saw that did not match its script.
func (s *Speaker) Problems() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.problems...)
}

// Verify lists everything wrong so far: mismatched requests plus scripted exchanges still waiting to happen.
func (s *Speaker) Verify() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	problems := append([]string(nil), s.problems...)
	for _, pending := range s.script {
		problems = append(problems, fmt.Sprintf("expected request never made: %s %s", pending.Path, pending.SOAPAction))
	}
	return problems
}

func (s *Speaker) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	body, _ := io.ReadAll(r.Body)
	got := Exchange{Path: r.URL.Path, SOAPAction: r.Header.Get("SOAPACTION"), Body: string(body)}

	if len(s.script) == 0 {
		s.reject(w, fmt.Sprintf("unexpected request after the script was exhausted: %s %s", got.Path, got.SOAPAction))
		return
	}
	next := s.script[0]
	if problem := mismatch(next, got); problem != "" {
		s.reject(w, problem)
		return
	}
	s.script = s.script[1:]
	w.WriteHeader(next.Respond.Status)
	fmt.Fprint(w, next.Respond.Body)
}

func (s *Speaker) reject(w http.ResponseWriter, problem string) {
	s.problems = append(s.problems, problem)
	w.WriteHeader(http.StatusInternalServerError)
}

func mismatch(want, got Exchange) string {
	switch {
	case want.Path != got.Path:
		return fmt.Sprintf("path: got %q, want %q", got.Path, want.Path)
	case want.SOAPAction != got.SOAPAction:
		return fmt.Sprintf("SOAPACTION: got %s, want %s", got.SOAPAction, want.SOAPAction)
	case want.Body != got.Body:
		return fmt.Sprintf("body differs:\n got: %s\nwant: %s", got.Body, want.Body)
	}
	return ""
}
