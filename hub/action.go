// Package hub turns cached Sonos state and a query into the items Alfred shows. It is pure: no I/O.
package hub

import "strings"

// Action is what `do` is asked to perform. It travels through Alfred as the item's arg, encoded as "verb:payload".
type Action struct {
	Verb    string
	Payload string
}

func Encode(a Action) string {
	return a.Verb + ":" + a.Payload
}

func Decode(encoded string) (Action, error) {
	verb, payload, _ := strings.Cut(encoded, ":")
	return Action{Verb: verb, Payload: payload}, nil
}
