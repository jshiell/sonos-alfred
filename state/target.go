package state

import "sonos-alfred/sonos"

// ResolveTarget picks the group to control: the one the stored player is in now, else the group that is playing.
// playingCoordinatorUUID is "" when nothing is playing.
func ResolveTarget(topology sonos.Topology, storedPlayerUUID, playingCoordinatorUUID string) (sonos.Group, bool) {
	if group, found := groupContaining(topology, storedPlayerUUID); found {
		return group, true
	}
	return groupContaining(topology, playingCoordinatorUUID)
}

func groupContaining(topology sonos.Topology, playerUUID string) (sonos.Group, bool) {
	for _, group := range topology.Groups {
		for _, member := range group.Members {
			if member.UUID == playerUUID {
				return group, true
			}
		}
	}
	return sonos.Group{}, false
}
