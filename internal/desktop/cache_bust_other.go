//go:build !darwin

package desktop

// BustWebviewCacheOnUpgrade 在 Windows/Linux 上清 WebView 数据目录里的缓存
// 子目录（WebView2 的 Cache / Code Cache / GPUCache；WebKit 的 cache 目录）。
// localStorage 与 Cookie 不动。
func BustWebviewCacheOnUpgrade() {
	// Windows 的 WebView2 数据目录由 main.go 指到 dataDir/webview2，目录在这里
	// 不可知；由 main.go 在确定 dataDir 后调用 bustCacheOnUpgrade。
}
