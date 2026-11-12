package serial

import (
	"context"
	"strings"
	"testing"
)

// TestConnectionIDsSurviveRestart is the serial connecting-phase variant of
// issue #141: a fresh binder (after an app restart) must not re-mint connection
// IDs an earlier run already handed out, since a connecting tab's id is the
// connection ID and can be persisted.
func TestConnectionIDsSurviveRestart(t *testing.T) {
	before := New(context.TODO(), nil, nil)
	after := New(context.TODO(), nil, nil)

	seen := make(map[string]struct{})
	for i := 0; i < 50; i++ {
		id := before.nextConnectionID()
		if !strings.HasPrefix(id, "conn-") {
			t.Fatalf("connection id %q lost the conn- prefix", id)
		}
		seen[id] = struct{}{}
	}
	for i := 0; i < 50; i++ {
		id := after.nextConnectionID()
		if _, dup := seen[id]; dup {
			t.Fatalf("after-restart connection id %q collides with a pre-restart id (issue #141)", id)
		}
	}
}

type connIDTestLang struct{}

func (connIDTestLang) Lang() string { return "en" }

// TestResolveConnectionIDHonorsClientID：连接进度事件可能跑赢绑定调用的响应，
// 前端只有预先拿到 id 才能先订阅再连接（见 SSHConnectRequest.ConnectionID）。
// 合法 id 原样采纳，非法 id 报错（IPC 边界输入），留空回退服务端生成。
func TestResolveConnectionIDHonorsClientID(t *testing.T) {
	s := New(context.TODO(), connIDTestLang{}, nil)

	const clientID = "conn-2f0d9c58-11a7-4c8f-9d3e-7a1b0c9d8e7f"
	if got, err := s.resolveConnectionID(clientID); err != nil || got != clientID {
		t.Fatalf("client connection id not honored: got %q, err %v", got, err)
	}
	if got, err := s.resolveConnectionID(""); err != nil || !strings.HasPrefix(got, "conn-") {
		t.Fatalf("empty connection id must fall back to a server id, got %q, err %v", got, err)
	}
	for _, bad := range []string{"spaces in id", "slash/id", strings.Repeat("a", 65)} {
		if _, err := s.resolveConnectionID(bad); err == nil {
			t.Fatalf("invalid connection id %q was accepted", bad)
		}
	}
}
