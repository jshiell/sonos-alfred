// Package alfredjson renders hub items as Alfred's Script Filter JSON.
package alfredjson

import (
	"encoding/json"

	"sonos-alfred/hub"
)

type response struct {
	Items []item `json:"items"`
}

type item struct {
	Title    string         `json:"title"`
	Subtitle string         `json:"subtitle,omitempty"`
	Arg      string         `json:"arg,omitempty"`
	Valid    bool           `json:"valid"`
	Mods     map[string]mod `json:"mods,omitempty"`
}

type mod struct {
	Arg   string `json:"arg,omitempty"`
	Valid bool   `json:"valid"`
}

// Render is the Script Filter JSON for items. refreshPending asks Alfred to run the filter again shortly.
func Render(items []hub.Item, refreshPending bool) ([]byte, error) {
	var out response
	for _, row := range items {
		rendered := item{Title: row.Title, Subtitle: row.Subtitle, Arg: hub.Encode(row.Enter), Valid: row.Valid}
		rendered.Mods = map[string]mod{"cmd": modFor(row.Cmd), "alt": modFor(row.Alt)}
		out.Items = append(out.Items, rendered)
	}
	return json.Marshal(out)
}

// modFor is the action a modifier key runs, or a disabled key when the row has none: otherwise Alfred would run Enter's.
func modFor(action *hub.Action) mod {
	if action == nil {
		return mod{}
	}
	return mod{Arg: hub.Encode(*action), Valid: true}
}
