package app_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"sonos-alfred/app"
	"sonos-alfred/hub"
	"sonos-alfred/sonos"
	"sonos-alfred/state"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

// workflow is an Env over a temporary data directory, with a refresh spawner that only counts.
type workflow struct {
	env    app.Env
	clock  *clock
	cache  *state.Cache
	spawns int
}

func newWorkflow(t *testing.T) *workflow {
	t.Helper()
	failOnAnyHTTPRequest(t)
	dir := t.TempDir()
	w := &workflow{clock: &clock{now: time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)}}
	w.cache = state.NewCache(dir, w.clock.Now)
	w.env = app.Env{
		Dir:   dir,
		Now:   w.clock.Now,
		Spawn: func() error { w.spawns++; return nil },
	}
	return w
}

// failOnAnyHTTPRequest makes the test fail if the code under test goes to the network.
func failOnAnyHTTPRequest(t *testing.T) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Errorf("unexpected network request: %s %s", r.Method, r.URL)
		return nil, http.ErrHandlerTimeout
	})
	t.Cleanup(func() { http.DefaultTransport = original })
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func kitchenAndOffice() sonos.Topology {
	kitchen := sonos.Member{UUID: "RINCON_KITCHEN", Name: "Kitchen", Host: "192.0.2.130"}
	office := sonos.Member{UUID: "RINCON_OFFICE", Name: "Office", Host: "192.0.2.48"}
	return sonos.Topology{Groups: []sonos.Group{
		{Coordinator: kitchen, Members: []sonos.Member{kitchen}},
		{Coordinator: office, Members: []sonos.Member{office}},
	}}
}

// response is the part of the Script Filter JSON these tests look at.
type response struct {
	Rerun float64 `json:"rerun"`
	Items []struct {
		Title string `json:"title"`
	} `json:"items"`
}

func (r response) titles() []string {
	var titles []string
	for _, item := range r.Items {
		titles = append(titles, item.Title)
	}
	return titles
}

func runFilter(t *testing.T, w *workflow, query string) response {
	t.Helper()
	out, err := app.Filter(w.env, query)
	if err != nil {
		t.Fatal(err)
	}
	var parsed response
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("filter output is not JSON: %v\n%s", err, out)
	}
	return parsed
}

func TestFilterRendersAFreshCacheWithoutSpawningARefreshOrRerunning(t *testing.T) {
	w := newWorkflow(t)
	warm := hub.State{
		NowPlaying: sonos.NowPlaying{Track: 1, Title: "Nightcall"},
		Volume:     20,
		Topology:   kitchenAndOffice(),
	}
	if err := w.cache.Write(app.StateEntry, warm); err != nil {
		t.Fatal(err)
	}
	w.clock.now = w.clock.now.Add(5 * time.Second)

	got := runFilter(t, w, "")

	if len(got.Items) == 0 || got.Items[0].Title != "Nightcall" {
		t.Errorf("titles = %v, want the now-playing row first", got.titles())
	}
	if w.spawns != 0 {
		t.Errorf("spawned %d refreshes for a fresh cache, want 0", w.spawns)
	}
	if got.Rerun != 0 {
		t.Errorf("rerun = %v for a fresh cache, want none", got.Rerun)
	}
}
