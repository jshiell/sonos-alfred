package hub_test

import (
	"testing"

	"sonos-alfred/hub"
)

func TestActionEncodesAsVerbColonPayloadAndDecodesBack(t *testing.T) {
	action := hub.Action{Verb: "volume", Payload: "35"}

	encoded := hub.Encode(action)
	decoded, err := hub.Decode(encoded)

	if encoded != "volume:35" {
		t.Errorf("Encode = %q, want %q", encoded, "volume:35")
	}
	if err != nil || decoded != action {
		t.Errorf("Decode(%q) = %+v, %v; want %+v", encoded, decoded, err, action)
	}
}
