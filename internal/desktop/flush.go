package desktop

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// FlushWindows waits for every live workspace, rather than treating the first
// window's acknowledgement as proof that all conversations have been saved.
func FlushWindows(ctx context.Context, app *application.App, timeout time.Duration) error {
	return flushWindows(ctx, app.Window.GetAll(), app.Event.On, timeout)
}

func flushWindows(ctx context.Context, windows []application.Window, onEvent func(string, func(*application.CustomEvent)) func(), timeout time.Duration) error {
	pending := make(map[string]bool, len(windows))
	for _, window := range windows {
		pending[window.Name()] = true
	}
	if len(pending) == 0 {
		return nil
	}
	requestID := uuid.NewString()
	var mu sync.Mutex
	done := make(chan struct{})
	unsubscribe := onEvent("ai:flush-done", func(event *application.CustomEvent) {
		token, ok := event.Data.(string)
		if !ok || token != requestID {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if pending[event.Sender] {
			delete(pending, event.Sender)
			if len(pending) == 0 {
				close(done)
			}
		}
	})
	defer unsubscribe()
	for _, window := range windows {
		window.EmitEvent("ai:flush-all", requestID)
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		mu.Lock()
		defer mu.Unlock()
		return fmt.Errorf("workspace save acknowledgement timed out: %v", pending)
	}
}
