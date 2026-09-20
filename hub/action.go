// Package hub turns cached Sonos state and a query into the items Alfred shows. It is pure: no I/O.
package hub

import (
	"fmt"
	"net/url"
	"strings"
)

// Action is what `do` is asked to perform. It travels through Alfred as the item's arg,
// encoded as "verb:payload" with the payload URL-escaped.
type Action struct {
	Verb    string
	Payload string
}

func Encode(a Action) string {
	return a.Verb + ":" + url.PathEscape(a.Payload)
}

func Decode(encoded string) (Action, error) {
	verb, escaped, found := strings.Cut(encoded, ":")
	if !found || verb == "" {
		return Action{}, fmt.Errorf("malformed action %q: want verb:payload", encoded)
	}
	payload, err := url.PathUnescape(escaped)
	if err != nil {
		return Action{}, fmt.Errorf("malformed action %q: %w", encoded, err)
	}
	return Action{Verb: verb, Payload: payload}, nil
}
