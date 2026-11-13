package desktop

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// NoCacheAssets 给资源响应统一加 Cache-Control: no-cache：WebView 必须回源
// 重新验证。产物文件名带内容哈希，回源命中时代价极小；而「原地替换 App」后
// WKWebView 若继续用无缓存头的旧响应（启发式缓存），就会跑上一个版本的前端。
func TestNoCacheAssetsSetsHeaderAndPassesThrough(t *testing.T) {
	var innerHit bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		innerHit = true
		_, _ = w.Write([]byte("ok"))
	})

	rec := httptest.NewRecorder()
	NoCacheAssets(inner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/index.html", nil))

	if !innerHit {
		t.Fatal("middleware must pass the request through")
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want passthrough body", rec.Body.String())
	}
}
