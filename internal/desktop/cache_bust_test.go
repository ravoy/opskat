package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

// bustCacheOnUpgrade 的契约：同构建指纹不动缓存；指纹变化（或标记缺失）清空
// 缓存目录并写入新指纹；重复启动（指纹已匹配）不再清理。
func TestBustCacheOnUpgrade(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "Cache.db")
	write := func() { os.WriteFile(stale, []byte("stale"), 0o644) }
	marker := func() string { b, _ := os.ReadFile(filepath.Join(dir, "opskat-build")); return string(b) }

	// 首次启动（无标记）：清空并写入指纹。buildinfo.CommitID 由 ldflags 注入，
	// 测试环境可能为空——空指纹不应清理（宁可不清也不能误删每次启动）。
	// 空指纹：绝不清理（ldflags 未注入的裸 go test 环境也不误删）。
	orig := currentBuildFingerprint
	t.Cleanup(func() { currentBuildFingerprint = orig })
	currentBuildFingerprint = func() string { return "" }
	write()
	bustCacheOnUpgrade(dir)
	if b, _ := os.ReadFile(stale); string(b) != "stale" {
		t.Fatal("empty build fingerprint must not clear the cache")
	}
	currentBuildFingerprint = func() string { return "newbld" }

	write()
	bustCacheOnUpgrade(dir)
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("first boot (no marker) must clear the cache dir")
	}
	if marker() != currentBuildFingerprint() {
		t.Fatalf("marker = %q, want current build fingerprint", marker())
	}

	// 同指纹二次启动：缓存保留。
	os.WriteFile(stale, []byte("fresh"), 0o644)
	bustCacheOnUpgrade(dir)
	if b, _ := os.ReadFile(stale); string(b) != "fresh" {
		t.Fatal("same build must not clear the cache on every boot")
	}

	// 模拟替换新二进制：指纹变化 → 清空并更新标记。
	os.WriteFile(filepath.Join(dir, "opskat-build"), []byte("oldbuild"), 0o644)
	os.WriteFile(stale, []byte("stale-again"), 0o644)
	bustCacheOnUpgrade(dir)
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("changed build fingerprint must clear the cache dir")
	}
	if marker() != currentBuildFingerprint() {
		t.Fatalf("marker = %q, want refreshed fingerprint", marker())
	}
}
