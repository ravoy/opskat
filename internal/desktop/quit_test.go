package desktop

import (
	"testing"
	"time"
)

func TestQuitGateWaitsForFrontendWithoutBlockingNativeDispatch(t *testing.T) {
	ack := make(chan struct{})
	retried := make(chan struct{})
	gate := QuitGate(func() bool { <-ack; return true }, func() { close(retried) })
	// Repeated native dispatches while the frontend has not acknowledged must each
	// stay gated (two asserts, not `gate() || gate()` — staticcheck reads the latter
	// as a pointless identical comparison).
	if gate() {
		t.Fatal("quit must wait for frontend acknowledgement")
	}
	if gate() {
		t.Fatal("quit must stay gated on a repeated native dispatch")
	}
	close(ack)
	select {
	case <-retried:
	case <-time.After(time.Second):
		t.Fatal("native quit was not retried after acknowledgement")
	}
	if !gate() {
		t.Fatal("prepared quit must be accepted")
	}
}
