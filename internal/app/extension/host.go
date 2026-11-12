package extension

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/octays/opskat/pkg/extension"

	"github.com/octays/opskat/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The host capabilities below are the ones only the desktop window can offer. The
// rest — asset config and credentials, KV, the asset connection path — need no
// window and live in extension_svc (Service.HostConfig), shared with opsctl.

// fileDialogOpener implements extension.FileDialogOpener
type fileDialogOpener struct {
	ctx context.Context // Wails app context
}

func (o *fileDialogOpener) FileDialog(dialogType string, opts extension.DialogOptions) (string, error) {
	switch dialogType {
	case "open":
		return desktop.OpenFile(application.OpenFileDialogOptions{
			Title:   opts.Title,
			Filters: toWailsFilters(opts.Filters),
		})
	case "save":
		return desktop.SaveFile(application.SaveFileDialogOptions{
			Title:    opts.Title,
			Filename: opts.DefaultName,
			Filters:  toWailsFilters(opts.Filters),
		})
	default:
		return "", fmt.Errorf("unknown dialog type: %q", dialogType)
	}
}

func toWailsFilters(filters []string) []application.FileFilter {
	if len(filters) == 0 {
		return nil
	}
	result := make([]application.FileFilter, 0, len(filters))
	for _, f := range filters {
		result = append(result, application.FileFilter{
			DisplayName: f,
			Pattern:     f,
		})
	}
	return result
}

// toolEventHandler implements extension.ToolEventHandler for one extension.
type toolEventHandler struct {
	ext     *Extension
	extName string
}

// OnToolEvent forwards one tool event to the page that made the call, as
// "ext:tool:event". Only a page call has a listener: an event from an AI exec or
// opsctl call of the same tool is dropped here, so the tool never has to know
// who called it.
//
// invocationId rides along because one handler serves every concurrent call of
// this extension: without it a listener sees the extension's events merged into
// one stream and attributes another upload's progress to its own.
func (h *toolEventHandler) OnToolEvent(invocationID, eventType string, data json.RawMessage) error {
	if !h.ext.toolCalls.isPageCall(h.extName, invocationID) {
		return nil
	}
	h.ext.emit("ext:tool:event", map[string]any{
		"extension":    h.extName,
		"invocationId": invocationID,
		"eventType":    eventType,
		"data":         data,
	})
	return nil
}
