package system

import (
	"fmt"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/octays/opskat/internal/desktop"

	"github.com/cago-frame/cago/pkg/logger"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"go.uber.org/zap"
)

// OpenWorkspaceWindow opens an independent native workspace. An asset starts a
// new connection in that window; live sessions remain owned by their source.
func (s *System) OpenWorkspaceWindow(assetID int64) (string, error) {
	if assetID < 0 {
		return "", fmt.Errorf("asset ID must not be negative")
	}
	name := "workspace-" + uuid.NewString()
	query := url.Values{"workspace": {name}, "asset": {strconv.FormatInt(assetID, 10)}}
	if assetID > 0 {
		query.Set("view", "connection")
	}
	logger.Ctx(s.appCtx).Info("open workspace window start", zap.String("window", name), zap.Int64("assetID", assetID))
	s.openNativeWindow(name, query)
	logger.Ctx(s.appCtx).Info("open workspace window end", zap.String("window", name), zap.Int64("assetID", assetID))
	return name, nil
}

var pageWindowMu sync.Mutex

// OpenPageWindow opens or focuses a dedicated tool without reloading its state.
func (s *System) OpenPageWindow(request desktop.PageWindowRequest) (string, error) {
	spec, err := desktop.ResolvePageWindow(request)
	if err != nil {
		logger.Ctx(s.appCtx).Warn("open page window fail", zap.String("page", request.Page), zap.Error(err))
		return "", err
	}
	logger.Ctx(s.appCtx).Info("open page window start", zap.String("window", spec.Name), zap.String("page", request.Page))
	pageWindowMu.Lock()
	defer pageWindowMu.Unlock()
	if window, exists := application.Get().Window.GetByName(spec.Name); exists {
		window.Show()
		window.Focus()
		window.EmitEvent("desktop:page-requested", request)
	} else {
		s.openNativeWindow(spec.Name, spec.Query)
	}
	logger.Ctx(s.appCtx).Info("open page window end", zap.String("window", spec.Name), zap.String("page", request.Page))
	return spec.Name, nil
}

func (s *System) openNativeWindow(name string, query url.Values) {
	app := application.Get()
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: name, Title: "OpsKat", Width: 1440, Height: 900, MinWidth: 1000, MinHeight: 640,
		URL:       desktop.WindowURL(query),
		Frameless: runtime.GOOS == "windows", EnableFileDrop: true, ZoomControlEnabled: true,
		Mac: application.MacWindow{TitleBar: application.MacTitleBarHiddenInset, Backdrop: application.MacBackdropTransparent},
	})
	var approved atomic.Bool
	unsubscribe := app.Event.On("desktop:close-ready", func(event *application.CustomEvent) {
		if event.Sender == name {
			approved.Store(true)
			window.Close()
		}
	})
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !approved.Load() {
			event.Cancel()
			window.EmitEvent("desktop:close-requested")
		}
	})
	window.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		unsubscribe()
		logger.Ctx(s.appCtx).Info("workspace window closed", zap.String("window", name))
	})
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		emitDroppedFiles(window, event)
	})
}

func emitDroppedFiles(window application.Window, event *application.WindowEvent) {
	details := event.Context().DropTargetDetails()
	// A native drop outside a recognized target has no routing coordinates.
	if details == nil {
		return
	}
	window.EmitEvent("desktop:files-dropped", map[string]any{
		"paths": event.Context().DroppedFiles(), "x": details.X, "y": details.Y,
	})
}

// RegisterMainWindowDrops connects native file drops to the main workspace.
func RegisterMainWindowDrops(window *application.WebviewWindow) {
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		emitDroppedFiles(window, event)
	})
}
