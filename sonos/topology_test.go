package sonos_test

import (
	"context"
	"net/http"
	"testing"

	"sonos-alfred/internal/fakespeaker"
	"sonos-alfred/sonos"
)

const getZoneGroupStateRequest = `<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:GetZoneGroupState xmlns:u="urn:schemas-upnp-org:service:ZoneGroupTopology:1"></u:GetZoneGroupState></s:Body></s:Envelope>`

func topologyFrom(t *testing.T, fixture string) sonos.Topology {
	t.Helper()
	speaker := fakespeaker.New(t, fakespeaker.Exchange{
		Path:       "/ZoneGroupTopology/Control",
		SOAPAction: `"urn:schemas-upnp-org:service:ZoneGroupTopology:1#GetZoneGroupState"`,
		Body:       getZoneGroupStateRequest,
		Respond:    fakespeaker.Response{Status: http.StatusOK, Body: fixtureFile(t, fixture)},
	})
	topology, err := sonos.NewClient(speaker.URL).Topology(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return topology
}

func TestTopologyListsGroupsWithTheirCoordinators(t *testing.T) {
	topology := topologyFrom(t, "zonegroupstate-home-theatre.xml")

	type coordinator struct{ uuid, host string }
	want := map[string]coordinator{
		"Kitchen":     {"RINCON_44444444444401400", "192.0.2.130"},
		"Living Room": {"RINCON_22222222222201400", "192.0.2.34"},
		"Garden Room": {"RINCON_33333333333301400", "192.0.2.61"}, // group ID starts with Dining Room's UUID
		"Office":      {"RINCON_55555555555501400", "192.0.2.48"},
		"Dining Room": {"RINCON_11111111111101400", "192.0.2.29"},
		"Play":        {"RINCON_66666666666601400", "192.0.2.45"},
	}
	if len(topology.Groups) != len(want) {
		t.Fatalf("got %d groups, want %d", len(topology.Groups), len(want))
	}
	for _, group := range topology.Groups {
		expected, ok := want[group.Coordinator.Name]
		if !ok {
			t.Errorf("unexpected group coordinated by %q", group.Coordinator.Name)
			continue
		}
		if group.Coordinator.UUID != expected.uuid || group.Coordinator.Host != expected.host {
			t.Errorf("%s: coordinator = %s@%s, want %s@%s", group.Coordinator.Name,
				group.Coordinator.UUID, group.Coordinator.Host, expected.uuid, expected.host)
		}
	}
}

func TestTopologyHidesTheInvisiblePartnerOfAStereoPair(t *testing.T) {
	// Synthetic fixture: no stereo pair exists in the household the real fixtures came from.
	topology := topologyFrom(t, "zonegroupstate-stereo-pair.synthetic.xml")

	var bedroom *sonos.Group
	for i := range topology.Groups {
		if topology.Groups[i].Coordinator.Name == "Bedroom" {
			bedroom = &topology.Groups[i]
		}
	}
	if bedroom == nil {
		t.Fatal("no Bedroom group")
	}
	if len(bedroom.Members) != 1 || bedroom.Members[0].UUID != "RINCON_AAAAAAAAAAAA01400" {
		t.Errorf("Bedroom members = %+v, want only the visible RINCON_AAAAAAAAAAAA01400", bedroom.Members)
	}
}
