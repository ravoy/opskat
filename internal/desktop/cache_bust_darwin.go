//go:build darwin

package desktop

import (
	"os"
	"path/filepath"
)

// BustWebviewCacheOnUpgrade 是 bustCacheOnUpgrade 的 darwin 入口：清
// ~/Library/Caches/<bundle>（WKWebView 的 HTTP 缓存，含缓存数据库）。
func BustWebviewCacheOnUpgrade() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	bustCacheOnUpgrade(filepath.Join(home, "Library", "Caches", "com.wails.opskat"))
}
