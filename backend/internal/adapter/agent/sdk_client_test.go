package agent

import (
	"context"
	"sync"
	"testing"
)

// newTestSDKSession creates a minimal SDKSession for testing Close() behavior.
// It does not connect to any real Claude CLI.
func newTestSDKSession() *SDKSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &SDKSession{
		ctx:    ctx,
		cancel: cancel,
		output: make(chan []byte, 10),
		done:   make(chan struct{}),
	}
}

func TestSDKSession_Close_Concurrent(t *testing.T) {
	s := newTestSDKSession()

	var wg sync.WaitGroup
	// Call Close from multiple goroutines simultaneously — must not panic.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Close()
		}()
	}
	wg.Wait()

	// Verify channels are closed.
	select {
	case _, ok := <-s.output:
		if ok {
			t.Error("output channel should be closed")
		}
	default:
		// closed channel returns immediately
	}

	select {
	case <-s.done:
		// expected — channel closed
	default:
		t.Error("done channel should be closed")
	}
}

func TestSDKSession_Close_Sequential(t *testing.T) {
	s := newTestSDKSession()

	// Calling Close twice sequentially must not panic.
	s.Close()
	s.Close()

	// Verify context is cancelled.
	if s.ctx.Err() == nil {
		t.Error("context should be cancelled after Close()")
	}
}
