package ssh

import (
	"context"
	"strings"
	"testing"
)

// TestConnectionIDsSurviveRestart is the connecting-phase variant of issue #141:
// a connection ID is the frontend tab id while a connection is establishing, so
// it can be persisted and restored. A fresh binder (after an app restart) must
// not re-mint connection IDs an earlier run already handed out, or a restored
// connecting tab collides with a new one. Tests through New to also catch a
// forgotten generator init (nil-deref in production).
func TestConnectionIDsSurviveRestart(t *testing.T) {
	before := New(context.TODO(), nil, nil, nil, nil)
	after := New(context.TODO(), nil, nil, nil, nil) // models the process after a restart

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

// TestResolveConnectionIDHonorsClientID：连接进度事件可能跑赢绑定调用的 HTTP 响应，
// 前端只有预先拿到 id 才能先订阅再连接。因此调用方带的合法 id 必须原样采纳，
// 非法 id 报错（IPC 边界输入），留空回退服务端生成。
func TestResolveConnectionIDHonorsClientID(t *testing.T) {
	s := New(context.TODO(), connIDTestLang{}, nil, nil, nil)

	// A client-supplied id is adopted verbatim: the frontend subscribed to
	// `<prefix>:connect:<id>` before calling, so any other id loses the events.
	const clientID = "conn-0b56f6d2-8b4e-4f30-9d5a-6f1a2f3e4d5b"
	got, err := s.resolveConnectionID(clientID)
	if err != nil || got != clientID {
		t.Fatalf("client connection id not honored: got %q, err %v", got, err)
	}

	// Empty falls back to the server generator (callers that never subscribe).
	serverID, err := s.resolveConnectionID("")
	if err != nil || !strings.HasPrefix(serverID, "conn-") {
		t.Fatalf("empty connection id must fall back to a server id, got %q, err %v", serverID, err)
	}

	// The id becomes a map key and an event name: reject anything else at the boundary.
	for _, bad := range []string{"spaces in id", "slash/id", "-leading-dash", strings.Repeat("a", 65), "id\n"} {
		if _, err := s.resolveConnectionID(bad); err == nil {
			t.Fatalf("invalid connection id %q was accepted", bad)
		}
	}
}
