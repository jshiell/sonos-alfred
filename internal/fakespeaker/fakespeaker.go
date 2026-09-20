// Package fakespeaker is a strict fake Sonos speaker for tests. It replays a scripted sequence of SOAP exchanges.
package fakespeaker

import (
	"fmt"
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

func (s *Speaker) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.script[0]
	s.script = s.script[1:]
	w.WriteHeader(next.Respond.Status)
	fmt.Fprint(w, next.Respond.Body)
}
