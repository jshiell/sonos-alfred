// Package state holds what the Alfred workflow remembers between runs.
package state

import (
	"encoding/json"
	"errors"
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
	return replaceFile(c.path(name), data)
}

// replaceFile swaps path's content in one step, so a reader sees the old file or the new one, never half of either.
func replaceFile(path string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".write-*")
	if err != nil {
		return err
	}
	_, writeErr := temp.Write(data)
	closeErr := temp.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		os.Remove(temp.Name())
		return err
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		os.Remove(temp.Name())
		return err
	}
	return nil
}

// Read fills into and reports true when name holds a value no older than ttl.
func (c *Cache) Read(name string, ttl time.Duration, into any) bool {
	e, ok := c.load(name)
	if !ok || c.now().Sub(e.WrittenAt) > ttl {
		return false
	}
	return json.Unmarshal(e.Value, into) == nil
}

// ReadWithAge fills into with name's value however old it is, and says how old that is.
func (c *Cache) ReadWithAge(name string, into any) (age time.Duration, found bool) {
	e, ok := c.load(name)
	if !ok || json.Unmarshal(e.Value, into) != nil {
		return 0, false
	}
	return c.now().Sub(e.WrittenAt), true
}

func (c *Cache) load(name string) (entry, bool) {
	data, err := os.ReadFile(c.path(name))
	if err != nil {
		return entry{}, false
	}
	var e entry
	if json.Unmarshal(data, &e) != nil {
		return entry{}, false
	}
	return e, true
}

func (c *Cache) path(name string) string {
	return filepath.Join(c.dir, name+".json")
}
