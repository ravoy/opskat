//go:build darwin

package desktop

import (
	"os/exec"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"
)

// DisablePressAndHold 在应用域注册 ApplePressAndHoldEnabled=false：长按字母键
// 按键重复，而不是弹出变音符号选择器（长按 d 提交 ď/ð 一类字符进终端）。
// 只写本应用自己的 defaults 域（iTerm2 同款做法），幂等，改完全局行为不变。
func DisablePressAndHold() {
	cmd := exec.Command("defaults", "write", "com.wails.opskat", "ApplePressAndHoldEnabled", "-bool", "false")
	if err := cmd.Run(); err != nil {
		logger.Default().Warn("disable press-and-hold failed", zap.Error(err))
	}
}
