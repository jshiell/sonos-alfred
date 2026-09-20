package state

import (
	"os"
	"path/filepath"
)

// Settings holds choices the user made, which stay until they change them.
type Settings struct {
	dir string
}

func NewSettings(dir string) *Settings {
	return &Settings{dir: dir}
}

// SetActiveGroup remembers which group to control, by the UUID of one of its players.
func (s *Settings) SetActiveGroup(playerUUID string) error {
	return replaceFile(s.activeGroupPath(), []byte(playerUUID))
}

// ActiveGroup returns the remembered player UUID, or "" when none was chosen.
func (s *Settings) ActiveGroup() string {
	data, err := os.ReadFile(s.activeGroupPath())
	if err != nil {
		return ""
	}
	return string(data)
}

func (s *Settings) activeGroupPath() string {
	return filepath.Join(s.dir, "active-group")
}
