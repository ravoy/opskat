package desktop

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type flushWindow struct {
	application.Window
	name string
	emit func(string, ...any) bool
}

func (w flushWindow) Name() string                            { return w.name }
func (w flushWindow) EmitEvent(name string, data ...any) bool { return w.emit(name, data...) }

func TestFlushRequiresEachWindowAndIgnoresStaleDuplicateAndMalformedAcks(t *testing.T) {
	var acknowledge func(*application.CustomEvent)
	emitted := make(chan string, 2)
	emit := func(_ string, data ...any) bool { emitted <- data[0].(string); return false }
	windows := []application.Window{flushWindow{name: "main", emit: emit}, flushWindow{name: "workspace-one", emit: emit}}
	result := make(chan error, 1)
	go func() {
		result <- flushWindows(context.Background(), windows, func(_ string, callback func(*application.CustomEvent)) func() {
			acknowledge = callback
			return func() {}
		}, time.Second)
	}()
	token := <-emitted
	if second := <-emitted; second != token {
		t.Fatal("windows must save for the same request")
	}
	for _, ack := range []*application.CustomEvent{
		{Sender: "main", Data: []any{"invalid"}},
		{Sender: "workspace-one", Data: "old-request"},
		{Sender: "unrelated-window", Data: token},
		{Sender: "main", Data: token},
		{Sender: "main", Data: token},
	} {
		acknowledge(ack)
	}
	select {
	case err := <-result:
		t.Fatalf("finished before the second window saved: %v", err)
	default:
	}
	acknowledge(&application.CustomEvent{Sender: "workspace-one", Data: token})
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestFlushNamesTheUnresponsiveWindow(t *testing.T) {
	err := flushWindows(context.Background(), []application.Window{flushWindow{name: "workspace-two", emit: func(string, ...any) bool { return false }}},
		func(string, func(*application.CustomEvent)) func() { return func() {} }, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "workspace-two") {
		t.Fatalf("missing window was not reported: %v", err)
	}
}
