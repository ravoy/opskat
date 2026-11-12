//go:build windows

package system

import (
	"context"

	"github.com/octays/opskat/internal/pkg/winclip"
)

func readClipboardText(context.Context) (string, error) {
	return winclip.ReadText()
}
