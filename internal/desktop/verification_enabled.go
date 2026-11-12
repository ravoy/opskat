//go:build opskat_e2e

package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cago-frame/cago/pkg/logger"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"
)

// verificationTransport preserves the native app while exposing its IPC to the
// existing Playwright harness. It is compiled only into isolated test builds.
type verificationTransport struct {
	*application.HTTPTransport
	address string
	server  *http.Server
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func VerificationTransport() (application.Transport, error) {
	if os.Getenv("OPSKAT_E2E") != "1" {
		return nil, nil
	}
	port, err := strconv.Atoi(os.Getenv("OPSKAT_E2E_PORT"))
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("OPSKAT_E2E_PORT must be a valid loopback port")
	}
	return &verificationTransport{HTTPTransport: application.NewHTTPTransport(),
		address: fmt.Sprintf("127.0.0.1:%d", port), clients: make(map[chan []byte]struct{})}, nil
}

func (t *verificationTransport) ServeAssets(assets http.Handler) error {
	listener, err := net.Listen("tcp", t.address)
	if err != nil {
		return err
	}
	t.server = &http.Server{ReadHeaderTimeout: 5 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A loopback listener alone does not prevent browser cross-origin calls.
		if r.Host != listener.Addr().String() && r.Host != "localhost:"+strconv.Itoa(listener.Addr().(*net.TCPAddr).Port) {
			http.Error(w, "invalid host", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
			http.Error(w, "invalid origin", http.StatusForbidden)
			return
		}
		switch r.URL.Path {
		case "/wails/verification-windows":
			// Read Go registry names without querying native objects that may
			// already be closing on the UI thread.
			names := make([]string, 0)
			for _, window := range application.Get().Window.GetAll() {
				names = append(names, window.Name())
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(names); err != nil {
				logger.Default().Error("write verification window list", zap.Error(err))
			}
			return
		case "/wails/verification-events":
			t.events(w, r)
			return
		case "/wails/transport.js":
			w.Header().Set("Content-Type", "text/javascript")
			_, _ = w.Write(t.JSClient())
			return
		}
		if r.URL.Path == "/wails/runtime" {
			name := r.Header.Get("x-wails-window-name")
			if name == "" {
				name = "main"
			}
			window, ok := application.Get().Window.GetByName(name)
			if !ok {
				http.Error(w, "window unavailable", http.StatusNotFound)
				return
			}
			r.Header.Set("x-wails-window-id", strconv.FormatUint(uint64(window.ID()), 10))
			r.Header.Set("x-wails-window-name", name)
		}
		if r.URL.Path == "/" {
			recorder := httptest.NewRecorder()
			assets.ServeHTTP(recorder, r)
			for name, values := range recorder.Header() {
				for _, value := range values {
					w.Header().Add(name, value)
				}
			}
			w.Header().Del("Content-Length")
			w.WriteHeader(recorder.Code)
			_, _ = w.Write([]byte(strings.Replace(recorder.Body.String(), "</head>", `<script src="/wails/transport.js"></script></head>`, 1)))
			return
		}
		assets.ServeHTTP(w, r)
	})}
	go func() {
		if err := t.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Default().Error("verification bridge failed", zap.Error(err))
		}
	}()
	return nil
}

func (t *verificationTransport) JSClient() []byte {
	return []byte(`(() => {
const originalFetch = window.fetch.bind(window);
window.fetch = (input, options = {}) => {
  if (typeof input === "string" && new URL(input, location.href).pathname === "/wails/runtime") {
    const headers = new Headers(options.headers);
    headers.set("x-wails-window-name", new URLSearchParams(location.search).get("workspace") || "main");
    options = {...options, headers};
  }
  return originalFetch(input, options);
};
const events = new EventSource("/wails/verification-events");
events.onmessage = ({data}) => window._wails?.dispatchWailsEvent?.(JSON.parse(data));
window.addEventListener("beforeunload", () => events.close());
})();`)
}

func (t *verificationTransport) events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher := w.(http.Flusher)
	client := make(chan []byte, 256)
	t.mu.Lock()
	t.clients[client] = struct{}{}
	t.mu.Unlock()
	defer func() { t.mu.Lock(); delete(t.clients, client); t.mu.Unlock() }()
	_, _ = fmt.Fprint(w, ": ready\n\n")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-client:
			if _, err := fmt.Fprintf(w, "data: %s\n\n", event); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (t *verificationTransport) DispatchWailsEvent(event *application.CustomEvent) {
	for _, window := range application.Get().Window.GetAll() {
		window.DispatchWailsEvent(event)
	}
	data, err := json.Marshal(event)
	if err != nil {
		logger.Default().Error("encode verification event", zap.Error(err))
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for client := range t.clients {
		select {
		case client <- data:
		default:
			logger.Default().Error("verification event subscriber too slow", zap.String("event", event.Name))
		}
	}
}

func (t *verificationTransport) Stop() error {
	var serverErr error
	if t.server != nil {
		serverErr = t.server.Close()
	}
	return errors.Join(serverErr, t.HTTPTransport.Stop())
}
