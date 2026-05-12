package ws

import (
	"sync"
	"testing"
	"time"
)

// TestBroadcastNonBlocking ensures Publish returns promptly even when a client's
// send buffer is full — the slow consumer should be moved into a background
// drain goroutine instead of stalling the caller.
func TestBroadcastNonBlocking(t *testing.T) {
	h := New(nil)
	// Plant a synthetic client whose send buffer is size 1 and is already full.
	c := &client{send: make(chan []byte, 1)}
	c.send <- []byte("filler")
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	done := make(chan struct{})
	go func() {
		h.BroadcastRaw([]byte("payload"))
		close(done)
	}()
	select {
	case <-done:
		// good — returned without blocking on the full buffer
	case <-time.After(500 * time.Millisecond):
		t.Fatal("broadcast blocked on slow consumer")
	}
}

// TestBroadcastDeliversToFastConsumer verifies a non-saturated client receives
// the payload via its send channel.
func TestBroadcastDeliversToFastConsumer(t *testing.T) {
	h := New(nil)
	c := &client{send: make(chan []byte, 4)}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	h.BroadcastRaw([]byte("hello"))
	select {
	case got := <-c.send:
		if string(got) != "hello" {
			t.Fatalf("got %q, want hello", string(got))
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("payload not delivered")
	}
}

// TestBroadcastMultipleClients ensures every client sees the payload (those
// that have buffer headroom).
func TestBroadcastMultipleClients(t *testing.T) {
	h := New(nil)
	var clients []*client
	for i := 0; i < 5; i++ {
		c := &client{send: make(chan []byte, 4)}
		clients = append(clients, c)
		h.mu.Lock()
		h.clients[c] = struct{}{}
		h.mu.Unlock()
	}
	h.BroadcastRaw([]byte("x"))
	var wg sync.WaitGroup
	for _, c := range clients {
		wg.Add(1)
		go func(c *client) {
			defer wg.Done()
			select {
			case <-c.send:
			case <-time.After(200 * time.Millisecond):
				t.Errorf("client did not receive payload")
			}
		}(c)
	}
	wg.Wait()
}
