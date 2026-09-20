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

func TestActionPayloadIsURLEscapedAndSurvivesTheRoundTrip(t *testing.T) {
	action := hub.Action{Verb: "note", Payload: "a b\n:%é"}

	encoded := hub.Encode(action)
	decoded, err := hub.Decode(encoded)

	if encoded != "note:a%20b%0A:%25%C3%A9" {
		t.Errorf("Encode = %q, want %q", encoded, "note:a%20b%0A:%25%C3%A9")
	}
	if err != nil || decoded != action {
		t.Errorf("Decode(%q) = %+v, %v; want %+v", encoded, decoded, err, action)
	}
}
