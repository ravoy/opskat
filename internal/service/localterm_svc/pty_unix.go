//go:build !windows

package localterm_svc

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/creack/pty"

	"github.com/octays/opskat/internal/pkg/shellutil"
)

type unixPTY struct {
	f   *os.File
	cmd *exec.Cmd
}

func startPTY(spec ptySpec) (ptyProcess, error) {
	shell := spec.Shell
	if shell == "" {
		shell = shellutil.DefaultShell()
	}
	cwd, err := expandHomeDir(spec.Cwd)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(shell, loginDefaultArgs(shell, spec.Args)...) //nolint:gosec // G204: 启动用户在本地终端资产里选择的 shell 是本功能的核心意图
	cmd.Dir = cwd
	// COLORTERM=truecolor：TERM=xterm-256color 的能力位只到 256 色，部分 TUI 会据此
	// 关掉 24-bit 输出。补这个变量让"看 COLORTERM 才发真彩 SGR"的程序也能出真彩；
	// 前端 xterm 本就按 24-bit 渲染，两端对齐。
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	// GUI launches may have no locale. Without one, zsh edits UTF-8 input as
	// individual bytes and displays <00xx> escapes. Preserve explicit locales.
	if os.Getenv("LC_ALL") == "" && os.Getenv("LC_CTYPE") == "" && os.Getenv("LANG") == "" {
		locale := "C.UTF-8"
		if runtime.GOOS == "darwin" {
			locale = "en_US.UTF-8"
		}
		cmd.Env = append(cmd.Env, "LANG="+locale)
	}

	cols, rows := clampSize(spec.Cols, spec.Rows)
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	return &unixPTY{f: f, cmd: cmd}, nil
}

func (p *unixPTY) Read(b []byte) (int, error)  { return p.f.Read(b) }
func (p *unixPTY) Write(b []byte) (int, error) { return p.f.Write(b) }

func (p *unixPTY) Resize(cols, rows int) error {
	cols, rows = clampSize(cols, rows)
	return pty.Setsize(p.f, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func (p *unixPTY) Close() error {
	// 先给前台进程组发 SIGHUP 触发 shell 退出,再关 PTY master。
	_ = p.cmd.Process.Signal(syscall.SIGHUP)
	err := p.f.Close()
	// 后台带超时回收,避免僵尸进程,也避免 Close 阻塞调用方。
	go func() {
		done := make(chan struct{})
		go func() { _ = p.cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = p.cmd.Process.Kill()
			<-done
		}
	}()
	return err
}

// loginDefaultArgs 无自定义参数时给登录型 shell 默认加 -l：PTY 让 shell 交互
// （读 .zshrc/.bashrc），但不登录——.zprofile/.bash_profile 里写的 nvm、pyenv、
// PATH 改写等就全丢了。macOS Terminal.app 默认也是登录 shell。用户自定义了
// 参数则原样使用（想非登录就自己传空串以外的参数）。
func loginDefaultArgs(shell string, args []string) []string {
	if len(args) > 0 {
		return args
	}
	base := filepath.Base(shell)
	switch base {
	case "zsh", "bash", "sh", "dash", "fish":
		return []string{"-l"}
	default:
		return nil
	}
}
