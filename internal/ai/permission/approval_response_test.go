package permission

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseApprovalResponseKindCapabilities(t *testing.T) {
	expected := []ApprovalItem{{Type: "exec", AssetID: 7, AssetName: "web-1", Command: "uptime"}}

	parsed, err := ParseApprovalResponse(ApprovalKindSingle, ApprovalResponse{Decision: "allow"}, expected)
	require.NoError(t, err)
	require.Equal(t, ApprovalAllow, parsed.Decision)

	parsed, err = ParseApprovalResponse(ApprovalKindOnce, ApprovalResponse{Decision: "allow"}, expected)
	require.NoError(t, err)
	require.Equal(t, ApprovalAllow, parsed.Decision)

	parsed, err = ParseApprovalResponse(ApprovalKindSingle, ApprovalResponse{
		Decision:    "allowAll",
		EditedItems: []ApprovalItem{{Type: "exec", AssetID: 7, AssetName: "web-1", Command: "uptime *"}},
	}, expected)
	require.NoError(t, err)
	require.Equal(t, ApprovalAllowAll, parsed.Decision)

	for _, kind := range []string{ApprovalKindOnce, ApprovalKindBatch, ApprovalKindGrant, ApprovalKindDelete} {
		t.Run(kind+" rejects allowAll", func(t *testing.T) {
			parsed, err := ParseApprovalResponse(kind, ApprovalResponse{Decision: "allowAll"}, expected)
			require.Error(t, err)
			require.Equal(t, ApprovalDeny, parsed.Decision)
		})
	}
}

func TestParseApprovalResponseTreatsUnchangedCommandsAsSystemSubjects(t *testing.T) {
	expected := []ApprovalItem{{Type: "oss", AssetID: 7, AssetName: "s3-prod", Command: "object.read mybucket/secrets*"}}
	parsed, err := ParseApprovalResponse(ApprovalKindSingle, ApprovalResponse{
		Decision:    "allowAll",
		EditedItems: append([]ApprovalItem(nil), expected...),
	}, expected)

	require.NoError(t, err)
	require.Equal(t, ApprovalAllowAll, parsed.Decision)
	require.Empty(t, parsed.EditedItems)
}

func TestParseApprovalResponseRejectsEditedScopeMutation(t *testing.T) {
	expected := []ApprovalItem{{Type: "exec", AssetID: 7, AssetName: "web-1", Command: "uptime"}}
	parsed, err := ParseApprovalResponse(ApprovalKindSingle, ApprovalResponse{
		Decision:    "allowAll",
		EditedItems: []ApprovalItem{{Type: "exec", AssetID: 8, AssetName: "other", Command: "uptime"}},
	}, expected)

	require.Error(t, err)
	require.Equal(t, ApprovalDeny, parsed.Decision)
}

func TestApprovalKindFor(t *testing.T) {
	tests := []struct {
		name         string
		approvalType string
		command      string
		want         string
	}{
		{name: "delete", approvalType: ApprovalTypeDelete, want: ApprovalKindDelete},
		{name: "exec with a matchable command", approvalType: "exec", command: "uptime", want: ApprovalKindSingle},
		{name: "exec without sub-commands", approvalType: "exec", command: `echo "`, want: ApprovalKindOnce},
		{name: "unregistered type", approvalType: "create", want: ApprovalKindOnce},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ApprovalKindFor(tt.approvalType, tt.command))
		})
	}
}

func TestParseApprovalResponseTimedAllow(t *testing.T) {
	expected := []ApprovalItem{{Type: "exec", AssetID: 7, AssetName: "web-1", Command: "systemctl restart nginx"}}

	// 逐条命令审批（single/once/batch）可限时全量授权，参数原样解析。
	for _, kind := range []string{ApprovalKindSingle, ApprovalKindOnce, ApprovalKindBatch} {
		parsed, err := ParseApprovalResponse(kind, ApprovalResponse{
			Decision: "allow_timed", TimedAllowMinutes: 60,
		}, expected)
		require.NoError(t, err, kind)
		require.Equal(t, ApprovalAllowTimed, parsed.Decision, kind)
		require.Equal(t, &TimedAllow{Minutes: 60}, parsed.TimedAllow, kind)
	}

	parsed, err := ParseApprovalResponse(ApprovalKindSingle, ApprovalResponse{
		Decision: "allow_timed", TimedAllowMinutes: 30, TimedAllowAllAssets: true,
	}, expected)
	require.NoError(t, err)
	require.Equal(t, &TimedAllow{Minutes: 30, AllAssets: true}, parsed.TimedAllow)

	// 授权面语义不符：delete 不可放行，grant 本身就是显式全量，local_tool 是 AI 工具面。
	for _, kind := range []string{ApprovalKindGrant, ApprovalKindDelete, ApprovalKindLocalTool} {
		parsed, err := ParseApprovalResponse(kind, ApprovalResponse{Decision: "allow_timed", TimedAllowMinutes: 60}, expected)
		require.Error(t, err, kind)
		require.Equal(t, ApprovalDeny, parsed.Decision, kind)
	}

	// 时长必须显式且在会话寿命内：0 / 负数 / 超过 24 小时一律拒绝（fail-closed）。
	for _, minutes := range []int{0, -5, MaxTimedAllowMinutes + 1} {
		parsed, err := ParseApprovalResponse(ApprovalKindSingle, ApprovalResponse{
			Decision: "allow_timed", TimedAllowMinutes: minutes,
		}, expected)
		require.Error(t, err)
		require.Equal(t, ApprovalDeny, parsed.Decision)
	}

	// 限时全量没有逐条编辑语义，混入 edited_items 视为损坏请求。
	parsed, err = ParseApprovalResponse(ApprovalKindSingle, ApprovalResponse{
		Decision: "allow_timed", TimedAllowMinutes: 60,
		EditedItems: []ApprovalItem{{Type: "exec", AssetID: 7, AssetName: "web-1", Command: "*"}},
	}, expected)
	require.Error(t, err)
	require.Equal(t, ApprovalDeny, parsed.Decision)
}
