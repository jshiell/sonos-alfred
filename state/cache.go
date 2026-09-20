// Package state holds what the Alfred workflow remembers between runs.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Cache keeps named JSON values in a directory, stamped with the time they were written.
type Cache struct {
	dir string
	now func() time.Time
}

func NewCache(dir string, now func() time.Time) *Cache {
	return &Cache{dir: dir, now: now}
}

type entry struct {
	WrittenAt time.Time       `json:"writtenAt"`
	Value     json.RawMessage `json:"value"`
}

func (c *Cache) Write(name string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	data, err := json.Marshal(entry{WrittenAt: c.now(), Value: raw})
	if err != nil {
		return err
	}
	return os.WriteFile(c.path(name), data, 0o644)
}

// Read fills into and reports true when name holds a value.
func (c *Cache) Read(name string, ttl time.Duration, into any) bool {
	data, err := os.ReadFile(c.path(name))
	if err != nil {
		return false
	}
	var e entry
	if json.Unmarshal(data, &e) != nil {
		return false
	}
	if c.now().Sub(e.WrittenAt) > ttl {
		return false
	}
	return json.Unmarshal(e.Value, into) == nil
}

func (c *Cache) path(name string) string {
	return filepath.Join(c.dir, name+".json")
}
