package local

import (
	"context"
	"strings"
	"testing"
)

// TestConnectionIDsSurviveRestart is the local connecting-phase variant of
// issue #141: a fresh binder (after an app restart) must not re-mint connection
// IDs an earlier run already handed out. The "local-" prefix is preserved
// because restore infers the transport from it (inferTransportFromSessionId).
func TestConnectionIDsSurviveRestart(t *testing.T) {
	before := New(context.TODO(), nil, nil)
	after := New(context.TODO(), nil, nil)

	seen := make(map[string]struct{})
	for i := 0; i < 50; i++ {
		id := before.nextConnectionID()
		if !strings.HasPrefix(id, "local-conn-") {
			t.Fatalf("connection id %q lost the local-conn- prefix", id)
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

// TestResolveConnectionIDHonorsClientID：同 ssh/serial——合法 id 原样采纳、非法 id
// 报错、留空回退。本地终端额外要求 local- 前缀：connecting 期的 tab id 就是连接
// ID，重启恢复时 transport 靠前缀反推（inferTransportFromSessionId）。
func TestResolveConnectionIDHonorsClientID(t *testing.T) {
	l := New(context.TODO(), connIDTestLang{}, nil)

	const clientID = "local-conn-3e1f8a64-90bc-4d27-8f5a-2c6d1b4a9e0c"
	if got, err := l.resolveConnectionID(clientID); err != nil || got != clientID {
		t.Fatalf("client connection id not honored: got %q, err %v", got, err)
	}
	if got, err := l.resolveConnectionID(""); err != nil || !strings.HasPrefix(got, "local-") {
		t.Fatalf("empty connection id must fall back to a server id, got %q, err %v", got, err)
	}
	// 缺 local- 前缀的 id 会让恢复期的 transport 反推失效，必须在边界拒绝。
	for _, bad := range []string{"conn-no-local-prefix", "local-/slash", strings.Repeat("local-a", 40)} {
		if _, err := l.resolveConnectionID(bad); err == nil {
			t.Fatalf("invalid local connection id %q was accepted", bad)
		}
	}
}
