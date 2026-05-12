// Package state persists pr-watcher dedup state to a JSON file.
//
// Dedup key format: "<provider>:<repo>:<pr_number>:<head_sha>"
// One Entry per key.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Entry tracks a single observed PR/MR head_sha.
type Entry struct {
	Provider     string    `json:"provider"`
	Repo         string    `json:"repo"`
	Number       int       `json:"number"`
	HeadSHA      string    `json:"head_sha"`
	Title        string    `json:"title,omitempty"`
	URL          string    `json:"url,omitempty"`
	FirstSeen    time.Time `json:"first_seen"`
	LastReviewed time.Time `json:"last_reviewed,omitempty"`
	ReviewURL    string    `json:"review_url,omitempty"`
	Status       string    `json:"status,omitempty"` // "seen", "queued", "reviewing", "reviewed", "failed"
	LastError    string    `json:"last_error,omitempty"`
}

// Key builds the canonical dedup key for an entry.
func Key(provider, repo string, number int, headSHA string) string {
	return fmt.Sprintf("%s:%s:%d:%s", provider, repo, number, headSHA)
}

// Store is a thread-safe in-memory map of dedup keys to Entries, backed by a JSON file.
type Store struct {
	mu      sync.RWMutex
	path    string
	entries map[string]*Entry
}

// New loads (or creates) a Store at path. Missing parent dirs are created.
func New(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("state path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir state dir: %w", err)
	}
	s := &Store{path: path, entries: map[string]*Entry{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	if len(b) == 0 {
		return s, nil
	}
	var disk map[string]*Entry
	if err := json.Unmarshal(b, &disk); err != nil {
		return nil, fmt.Errorf("parse state %q: %w", path, err)
	}
	if disk != nil {
		s.entries = disk
	}
	return s, nil
}

// Path returns the on-disk state file path.
func (s *Store) Path() string { return s.path }

// Seen returns true if the key is already present.
func (s *Store) Seen(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.entries[key]
	return ok
}

// MarkSeen inserts the entry if missing and returns true if this was a new entry.
// Existing entries are not mutated.
func (s *Store) MarkSeen(e Entry) bool {
	key := Key(e.Provider, e.Repo, e.Number, e.HeadSHA)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[key]; ok {
		return false
	}
	if e.FirstSeen.IsZero() {
		e.FirstSeen = time.Now().UTC()
	}
	if e.Status == "" {
		e.Status = "seen"
	}
	stored := e
	s.entries[key] = &stored
	return true
}

// Update mutates an existing entry through fn. fn receives a copy and must
// return the modified entry. Returns false if the key does not exist.
func (s *Store) Update(key string, fn func(Entry) Entry) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.entries[key]
	if !ok {
		return false
	}
	next := fn(*cur)
	s.entries[key] = &next
	return true
}

// Get returns a copy of the entry, or false if absent.
func (s *Store) Get(key string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[key]
	if !ok {
		return Entry{}, false
	}
	return *e, true
}

// Snapshot returns a deep copy of all entries.
func (s *Store) Snapshot() map[string]Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]Entry, len(s.entries))
	for k, v := range s.entries {
		out[k] = *v
	}
	return out
}

// ReviewsToday counts entries with LastReviewed within the last 24h, optionally
// filtered to a provider+repo pair (empty repo = all).
func (s *Store) ReviewsToday(provider, repo string) int {
	cutoff := time.Now().Add(-24 * time.Hour)
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, e := range s.entries {
		if repo != "" && (e.Provider != provider || e.Repo != repo) {
			continue
		}
		if !e.LastReviewed.IsZero() && e.LastReviewed.After(cutoff) {
			n++
		}
	}
	return n
}

// Save atomically writes the current state to disk.
func (s *Store) Save() error {
	s.mu.RLock()
	// Marshal under read-lock; copy into a slice of keys for stable JSON output.
	keys := make([]string, 0, len(s.entries))
	for k := range s.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]*Entry, len(s.entries))
	for _, k := range keys {
		ordered[k] = s.entries[k]
	}
	s.mu.RUnlock()
	b, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write tmp state: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}
	return nil
}
