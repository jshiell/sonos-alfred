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
		rendered.Mods = map[string]mod{}
		if row.Cmd != nil {
			rendered.Mods["cmd"] = mod{Arg: hub.Encode(*row.Cmd), Valid: true}
		}
		if row.Alt != nil {
			rendered.Mods["alt"] = mod{Arg: hub.Encode(*row.Alt), Valid: true}
		}
		out.Items = append(out.Items, rendered)
	}
	return json.Marshal(out)
}
