package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202611120001 给 grant_items 增加 expiretime：单条授权的可选到期时间。
// 限时全量授权（审批弹窗里"30分钟/1小时/2小时内不再询问"）落到一条 `*` pattern 的
// grant item 上并写入到期时间；0 保持旧语义——跟随会话的 24 小时上限。
func migration202611120001() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202611120001",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`ALTER TABLE grant_items ADD COLUMN expiretime INTEGER NOT NULL DEFAULT 0`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			// SQLite 不支持 DROP COLUMN（旧版本），重建表的代价大于收益：回滚仅见于
			// 开发期，列存在不影响旧代码路径（默认 0 = 永随会话）。
			return nil
		},
	}
}
