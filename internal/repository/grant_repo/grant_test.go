package grant_repo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cago-frame/cago/database/db"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/octays/opskat/internal/model/entity/grant_entity"
)

func setupGrantRepo(t *testing.T) (context.Context, GrantRepo) {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gdb.AutoMigrate(&grant_entity.GrantSession{}, &grant_entity.GrantItem{}))
	db.SetDefault(gdb)
	return context.Background(), NewGrant()
}

func seedSession(t *testing.T, ctx context.Context, r GrantRepo, id string) {
	t.Helper()
	require.NoError(t, r.CreateSession(ctx, &grant_entity.GrantSession{ID: id, Status: grant_entity.GrantStatusApproved, Createtime: time.Now().Unix()}))
}

// ListApprovedItems 只返回未过期的授权：限时全量授权（expiretime>0）到期即从
// "仍有效"集合消失，未单独设期（0）的沿用会话寿命。所有授权消费方读的都是这一份。
func TestGrantRepo_ListApprovedItemsFiltersExpired(t *testing.T) {
	ctx, r := setupGrantRepo(t)
	seedSession(t, ctx, r, "sess-1")
	now := time.Now().Unix()

	require.NoError(t, r.CreateItems(ctx, []*grant_entity.GrantItem{
		{GrantSessionID: "sess-1", ToolName: "exec", Command: "uptime"},                                // 未设期：随会话
		{GrantSessionID: "sess-1", ToolName: "exec", Command: "*", AssetID: 0, Expiretime: now + 3600}, // 限时 1h：仍有效
		{GrantSessionID: "sess-1", ToolName: "exec", Command: "*", AssetID: 7, Expiretime: now - 1},    // 已过期
		{GrantSessionID: "sess-1", ToolName: "exec", Command: "systemctl *", Expiretime: now},          // 恰好现在：过期
	}))

	items, err := r.ListApprovedItems(ctx, "sess-1")
	require.NoError(t, err)
	labels := make([]string, 0, len(items))
	for _, item := range items {
		labels = append(labels, fmt.Sprintf("%s@%d", item.Command, item.AssetID))
	}
	assert.ElementsMatch(t, []string{"uptime@0", "*@0"}, labels)
}
