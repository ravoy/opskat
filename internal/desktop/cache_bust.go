package desktop

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"

	"github.com/octays/opskat/internal/buildinfo"
)

// bustCacheOnUpgrade 在应用二进制更新后的首次启动清空 WebView 的 HTTP 缓存。
//
// 背景：WKWebView 对没有缓存头的旧响应做启发式缓存——原地替换 App 后继续跑
// 上一版本的前端（响应里带 no-cache 头只对"新缓存下来的条目"生效，救不了已
// 经躺在缓存里的旧条目）。应用自己最清楚"我换版本了"：以构建指纹为标记，
// 启动时发现指纹变了就清一次缓存，用户永远不需要手动 rm。
//
// 只清 HTTP 缓存目录（~/Library/Caches/<bundle>），不碰 ~/Library/WebKit（
// localStorage：语言、快捷键、会话偏好都在里面）。必须在窗口创建之前调用，
// 此时 WKWebView 进程尚未启动，删目录是安全的。
// currentBuildFingerprint 取当前构建指纹；变量化供测试在 ldflags 未注入的
// go test 环境里固定一个值（空指纹永不清理：宁可不清也不能每次启动误删）。
var currentBuildFingerprint = buildinfo.ShortCommitID

func bustCacheOnUpgrade(cacheDir string) {
	build := currentBuildFingerprint()
	if build == "" {
		return
	}
	marker := filepath.Join(cacheDir, "opskat-build")
	if current, err := os.ReadFile(marker); err == nil && strings.TrimSpace(string(current)) == build {
		return
	}
	if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
		logger.Default().Warn("clear webview cache on upgrade failed", zap.String("dir", cacheDir), zap.Error(err))
		// 清不干净也写入新指纹：缓存只是性能优化，不值得每次启动重试删除。
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		logger.Default().Warn("recreate webview cache dir failed", zap.String("dir", cacheDir), zap.Error(err))
		return
	}
	if err := os.WriteFile(marker, []byte(build), 0o644); err != nil {
		logger.Default().Warn("write build marker failed", zap.String("marker", marker), zap.Error(err))
	}
	logger.Default().Info("webview cache cleared for new build", zap.String("build", build))
}
