// Package serial 实现 serial binder：串口连接、读写、重置。
package serial

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/octays/opskat/internal/app/i18n"
	"github.com/octays/opskat/internal/model/entity/asset_entity"
	"github.com/octays/opskat/internal/service/conntest"
	"github.com/octays/opskat/internal/service/serial_svc"
	"github.com/octays/opskat/internal/service/sessionid"
)

// LangProvider 由 system binder 实现。
type LangProvider interface {
	Lang() string
}

// Serial binder。
type Serial struct {
	appCtx  context.Context
	ctx     context.Context
	lang    LangProvider
	manager *serial_svc.Manager

	connIDGen          *sessionid.Generator
	pendingConnections sync.Map // map[string]context.CancelFunc 异步连接取消用
}

// New 构造 serial binder。
func New(appCtx context.Context, lang LangProvider, mgr *serial_svc.Manager) *Serial {
	s := &Serial{appCtx: appCtx, lang: lang, manager: mgr, connIDGen: sessionid.NewGenerator("conn")}
	conntest.Register(asset_entity.AssetTypeSerial, s.testConnection)
	return s
}

// nextConnectionID 生成跨重启唯一的连接中转 ID(issue #141)。
func (s *Serial) nextConnectionID() string {
	return s.connIDGen.Next()
}

// clientConnectionIDPattern 约束前端预生成的连接 ID（IPC 边界输入，先校验再当
// map key / 事件名使用）；与 ssh/local 同一约定，见 SSHConnectRequest.ConnectionID。
var clientConnectionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

// resolveConnectionID 采纳前端预生成的连接 ID，非法值报错；留空回退服务端生成。
func (s *Serial) resolveConnectionID(clientID string) (string, error) {
	if clientID == "" {
		return s.nextConnectionID(), nil
	}
	if !clientConnectionIDPattern.MatchString(clientID) {
		return "", fmt.Errorf("%s: %q", i18n.Pick(s.lang.Lang(), "无效的连接 ID", "invalid connection id"), clientID)
	}
	return clientID, nil
}

// Startup 保存 Wails ctx。
func (s *Serial) Startup(ctx context.Context) { s.ctx = ctx }

// Cleanup 关闭所有串口。
func (s *Serial) Cleanup() {
	if s.manager != nil {
		s.manager.CloseAll()
	}
}
