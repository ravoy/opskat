package desktop

import (
	"net/url"
	"runtime"
)

// WindowURL preserves the v2 WebView origin, which owns the user's localStorage.
// Windows already uses the same http://wails.localhost origin in both versions.
func WindowURL(query url.Values) string {
	target := url.URL{Path: "/", RawQuery: query.Encode()}
	if runtime.GOOS != "windows" {
		target.Scheme = "wails"
		target.Host = "wails"
	}
	return target.String()
}
