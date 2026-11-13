//go:build !darwin

package desktop

// DisablePressAndHold 只对 macOS 的长按变音选择器有意义。
func DisablePressAndHold() {}
