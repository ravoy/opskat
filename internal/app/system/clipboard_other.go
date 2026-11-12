//go:build !windows

package system

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// macOS（pbpaste）和 Linux（GTK）上 Wails 对空剪贴板返回 ""，没有 Windows 那个问题。
func readClipboardText(ctx context.Context) (string, error) {
	text, ok := application.Get().Clipboard.Text()
	if !ok {
		return "", fmt.Errorf("read clipboard text failed")
	}
	return text, nil
}
