package state

import "sonos-alfred/sonos"

// ResolveTarget picks the group to control: the one the stored player is in now, else the group that is playing.
// playingCoordinatorUUID is "" when nothing is playing.
func ResolveTarget(topology sonos.Topology, storedPlayerUUID, playingCoordinatorUUID string) (sonos.Group, bool) {
	for _, group := range topology.Groups {
		for _, member := range group.Members {
			if member.UUID == storedPlayerUUID {
				return group, true
			}
		}
	}
	return sonos.Group{}, false
}
