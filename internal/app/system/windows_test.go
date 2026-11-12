package system

import "testing"

// 工具页窗口（设置/凭据/审计等 page-*）与资产工作区窗口的默认几何是产品契约：
// 设置类是表单/列表面，用 1080×720；工作区承载终端与文件面板，与主窗口同宽。
// 名称前缀是唯一的分发依据（workspace-<uuid> / page-<page>）。
func TestWindowSizeFor(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		wantW, wantH, wantMinW, wantMinH int
	}{
		{"page-settings", 1080, 720, 860, 600},
		{"page-sshkeys", 1080, 720, 860, 600},
		{"page-ai", 1080, 720, 860, 600},
		{"workspace-0b56f6d2-8b4e-4f30-9d5a-6f1a2f3e4d5b", 1440, 900, 1000, 640},
	} {
		w, h, minW, minH := windowSizeFor(tc.name)
		if w != tc.wantW || h != tc.wantH || minW != tc.wantMinW || minH != tc.wantMinH {
			t.Fatalf("windowSizeFor(%q) = %d×%d (min %d×%d), want %d×%d (min %d×%d)",
				tc.name, w, h, minW, minH, tc.wantW, tc.wantH, tc.wantMinW, tc.wantMinH)
		}
	}
}
