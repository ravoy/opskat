package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/octays/opskat/internal/model/entity/grant_entity"
)

// 迁移加出的 expiretime 列必须能直接承载实体写入，且默认 0（未单独设期）。
func TestMigration202611120001_GrantItemExpiretime(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migration202611120001?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 旧库里的表没有 expiretime 列——按初始迁移之后的形态先建出来，再跑本迁移。
	require.NoError(t, db.Exec(`
		CREATE TABLE grant_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			grant_session_id VARCHAR(36) NOT NULL,
			item_index INTEGER NOT NULL,
			tool_name VARCHAR(100) NOT NULL,
			asset_id INTEGER DEFAULT 0,
			asset_name VARCHAR(255),
			group_id INTEGER DEFAULT 0,
			group_name VARCHAR(255),
			command TEXT,
			detail TEXT,
			createtime INTEGER
		)`).Error)

	m := migration202611120001()
	require.NoError(t, m.Migrate(db))

	require.NoError(t, db.AutoMigrate(&grant_entity.GrantSession{}))
	item := &grant_entity.GrantItem{
		GrantSessionID: "s1", ToolName: "exec", Command: "*", Expiretime: 1750000000,
	}
	require.NoError(t, db.Create(item).Error)

	var read grant_entity.GrantItem
	require.NoError(t, db.Where("grant_session_id = ?", "s1").First(&read).Error)
	assert.Equal(t, int64(1750000000), read.Expiretime)

	// 旧路径不设期：默认 0。
	legacy := &grant_entity.GrantItem{GrantSessionID: "s1", ToolName: "exec", Command: "uptime"}
	require.NoError(t, db.Create(legacy).Error)
	assert.Equal(t, int64(0), legacy.Expiretime)
}
