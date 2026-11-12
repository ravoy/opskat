// Package local 实现 local binder：本地终端连接、读写、尺寸调整。
package local

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/octays/opskat/internal/app/i18n"
	"github.com/octays/opskat/internal/service/localterm_svc"
	"github.com/octays/opskat/internal/service/sessionid"
)

// LangProvider 由 system binder 实现。
type LangProvider interface {
	Lang() string
}

// Local binder。
type Local struct {
	appCtx    context.Context
	ctx       context.Context
	lang      LangProvider
	manager   *localterm_svc.Manager
	connIDGen *sessionid.Generator
}

// New 构造 local binder。
func New(appCtx context.Context, lang LangProvider, mgr *localterm_svc.Manager) *Local {
	return &Local{appCtx: appCtx, lang: lang, manager: mgr, connIDGen: sessionid.NewGenerator("local-conn")}
}

// nextConnectionID 生成跨重启唯一的连接中转 ID;保留 local- 前缀以兼容
// restore 时的 transport 推断(issue #141)。
func (l *Local) nextConnectionID() string {
	return l.connIDGen.Next()
}

// clientConnectionIDPattern 约束前端预生成的连接 ID（IPC 边界输入，先校验再当
// map key / 事件名使用）；与 ssh/serial 同一约定，见 SSHConnectRequest.ConnectionID。
var clientConnectionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

// resolveConnectionID 采纳前端预生成的连接 ID，非法值报错；留空回退服务端生成。
// 本地终端还要求 local- 前缀：connecting 期的 tab id 就是连接 ID，重启恢复时
// transport 靠它反推（inferTransportFromSessionId）。
func (l *Local) resolveConnectionID(clientID string) (string, error) {
	if clientID == "" {
		return l.nextConnectionID(), nil
	}
	if !strings.HasPrefix(clientID, "local-") || !clientConnectionIDPattern.MatchString(clientID) {
		return "", fmt.Errorf("%s: %q", i18n.Pick(l.lang.Lang(), "无效的连接 ID", "invalid connection id"), clientID)
	}
	return clientID, nil
}

// Startup 保存 Wails ctx。
func (l *Local) Startup(ctx context.Context) { l.ctx = ctx }

// Cleanup 关闭所有本地终端。
func (l *Local) Cleanup() {
	if l.manager != nil {
		l.manager.CloseAll()
	}
}
