package urlkit

import (
	"testing"
	"time"
)

func TestRuntimeStateFreezeWaitsForActiveMutation(t *testing.T) {
	state := newRuntimeState()

	release, err := state.beginMutation("add routes", "api")
	if err != nil {
		t.Fatalf("beginMutation failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		state.freeze()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("freeze should wait for active mutation to finish")
	case <-time.After(20 * time.Millisecond):
	}

	release()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("freeze did not complete after mutation release")
	}

	if !state.isFrozen() {
		t.Fatal("expected runtime state to be frozen")
	}

	if _, err := state.beginMutation("add routes", "api"); err == nil {
		t.Fatal("expected new mutations to fail after freeze")
	}
}
