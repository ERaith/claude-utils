package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestKeyShape(t *testing.T) {
	got := Key("github", "owner/name", 42, "abc123")
	want := "github:owner/name:42:abc123"
	if got != want {
		t.Fatalf("Key: got %q, want %q", got, want)
	}
}

func TestMarkSeenDedup(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e := Entry{Provider: "github", Repo: "o/r", Number: 1, HeadSHA: "sha1"}
	if !s.MarkSeen(e) {
		t.Fatal("first MarkSeen: want new=true")
	}
	if s.MarkSeen(e) {
		t.Fatal("second MarkSeen with identical key: want new=false")
	}
	// Different head sha = new entry
	e2 := e
	e2.HeadSHA = "sha2"
	if !s.MarkSeen(e2) {
		t.Fatal("MarkSeen with new head_sha: want new=true")
	}
	// Different number = new entry
	e3 := e
	e3.Number = 2
	if !s.MarkSeen(e3) {
		t.Fatal("MarkSeen with new number: want new=true")
	}
	if got, want := len(s.Snapshot()), 3; got != want {
		t.Fatalf("Snapshot len: got %d, want %d", got, want)
	}
}

func TestSeenAndGet(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e := Entry{Provider: "github", Repo: "o/r", Number: 1, HeadSHA: "sha1", Title: "T"}
	s.MarkSeen(e)
	key := Key("github", "o/r", 1, "sha1")
	if !s.Seen(key) {
		t.Fatal("Seen: want true")
	}
	got, ok := s.Get(key)
	if !ok {
		t.Fatal("Get: want ok")
	}
	if got.Title != "T" {
		t.Fatalf("Get title: got %q, want T", got.Title)
	}
	if got.FirstSeen.IsZero() {
		t.Fatal("Get FirstSeen: want non-zero (auto-stamped)")
	}
	if got.Status != "seen" {
		t.Fatalf("Get Status: got %q, want 'seen'", got.Status)
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e := Entry{Provider: "github", Repo: "o/r", Number: 1, HeadSHA: "sha1"}
	s.MarkSeen(e)
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	s2, err := New(path)
	if err != nil {
		t.Fatalf("New (reload): %v", err)
	}
	if !s2.Seen(Key("github", "o/r", 1, "sha1")) {
		t.Fatal("after reload: entry missing")
	}
}

func TestReviewsToday(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Now()
	s.MarkSeen(Entry{Provider: "github", Repo: "o/r", Number: 1, HeadSHA: "a"})
	s.Update(Key("github", "o/r", 1, "a"), func(e Entry) Entry {
		e.LastReviewed = now.Add(-1 * time.Hour)
		return e
	})
	s.MarkSeen(Entry{Provider: "github", Repo: "o/r", Number: 2, HeadSHA: "b"})
	s.Update(Key("github", "o/r", 2, "b"), func(e Entry) Entry {
		e.LastReviewed = now.Add(-25 * time.Hour)
		return e
	})
	s.MarkSeen(Entry{Provider: "github", Repo: "o/r", Number: 3, HeadSHA: "c"})
	// Number 3: never reviewed, should not count
	if got := s.ReviewsToday("", ""); got != 1 {
		t.Fatalf("ReviewsToday (all): got %d, want 1", got)
	}
	if got := s.ReviewsToday("github", "o/r"); got != 1 {
		t.Fatalf("ReviewsToday (filtered): got %d, want 1", got)
	}
	if got := s.ReviewsToday("github", "other/repo"); got != 0 {
		t.Fatalf("ReviewsToday (no match): got %d, want 0", got)
	}
}
