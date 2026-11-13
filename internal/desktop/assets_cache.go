package desktop

import "net/http"

// NoCacheAssets 给资源服务器加 Cache-Control: no-cache——WebView 每次都必须
// 回源重新验证，杜绝「原地替换 App 后 WKWebView 继续跑缓存里上一个版本前端」。
// 产物文件名带内容哈希，未变化时回源代价极小；资源由进程内的嵌入 FS 提供，
// 没有真正的网络往返。
func NoCacheAssets(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}
