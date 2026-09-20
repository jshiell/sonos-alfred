package sonos

import (
	"context"
	"encoding/xml"
	"net/url"
)

// Topology is the household's current grouping of speakers.
type Topology struct {
	Groups []Group
}

// Group is a set of rooms playing together. All commands go to its Coordinator.
type Group struct {
	Coordinator Member
	Members     []Member
}

// Member is one room in a group.
type Member struct {
	UUID string
	Name string
	Host string
}

// Topology reads the current zone groups from the speaker.
func (c *Client) Topology(ctx context.Context) (Topology, error) {
	values, err := c.Call(ctx, ZoneGroupTopology, "GetZoneGroupState")
	if err != nil {
		return Topology{}, err
	}
	return parseTopology(values["ZoneGroupState"])
}

func parseTopology(zoneGroupState string) (Topology, error) {
	var state struct {
		Groups []struct {
			Coordinator string `xml:"Coordinator,attr"`
			Members     []struct {
				UUID      string `xml:"UUID,attr"`
				ZoneName  string `xml:"ZoneName,attr"`
				Location  string `xml:"Location,attr"`
				Invisible string `xml:"Invisible,attr"`
			} `xml:"ZoneGroupMember"`
		} `xml:"ZoneGroups>ZoneGroup"`
	}
	if err := xml.Unmarshal([]byte(zoneGroupState), &state); err != nil {
		return Topology{}, err
	}

	var topology Topology
	for _, parsed := range state.Groups {
		var group Group
		for _, m := range parsed.Members {
			if m.Invisible == "1" {
				continue
			}
			member := Member{UUID: m.UUID, Name: m.ZoneName, Host: hostOf(m.Location)}
			group.Members = append(group.Members, member)
			if m.UUID == parsed.Coordinator {
				group.Coordinator = member
			}
		}
		topology.Groups = append(topology.Groups, group)
	}
	return topology, nil
}

func hostOf(location string) string {
	parsed, err := url.Parse(location)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}
