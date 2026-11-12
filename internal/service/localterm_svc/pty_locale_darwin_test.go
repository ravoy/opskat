package localterm_svc

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLocalZshAcceptsChineseWhenLaunchedWithoutLocale(t *testing.T) {
	// Finder-launched applications do not inherit Terminal.app's locale.
	for _, key := range []string{"LANG", "LC_ALL", "LC_CTYPE"} {
		t.Setenv(key, "")
	}
	t.Setenv("ZDOTDIR", t.TempDir())
	t.Setenv("PS1", "OPSKAT-INPUT> ")
	proc, err := startPTY(ptySpec{Shell: "/bin/zsh", Args: []string{"-f", "-i"}, Cols: 120, Rows: 24})
	require.NoError(t, err)
	defer func() { _ = proc.Close() }()
	output := make(chan string, 64)
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := proc.Read(buffer)
			if n > 0 {
				output <- string(buffer[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	readUntil := func(match string) string {
		t.Helper()
		var result strings.Builder
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()
		for {
			select {
			case chunk := <-output:
				result.WriteString(chunk)
				if strings.Contains(result.String(), match) {
					return result.String()
				}
			case <-timer.C:
				t.Fatalf("terminal did not produce %q; output=%q", match, result.String())
				return ""
			}
		}
	}
	readUntil("OPSKAT-INPUT> ")
	_, err = proc.Write([]byte("中文"))
	require.NoError(t, err)
	inputEcho := readUntil("文")
	require.Contains(t, inputEcho, "中文")
	require.NotContains(t, inputEcho, "<00", "zsh must edit UTF-8 characters instead of escaping individual bytes")
}
