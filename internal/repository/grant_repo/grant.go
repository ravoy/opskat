package grant_repo

import (
	"context"
	"time"

	"github.com/octays/opskat/internal/model/entity/grant_entity"

	"github.com/cago-frame/cago/database/db"
)

// GrantRepo 授权审批数据访问接口
type GrantRepo interface {
	CreateSession(ctx context.Context, session *grant_entity.GrantSession) error
	GetSession(ctx context.Context, id string) (*grant_entity.GrantSession, error)
	UpdateSessionStatus(ctx context.Context, id string, status int) error
	CreateItems(ctx context.Context, items []*grant_entity.GrantItem) error
	UpdateItems(ctx context.Context, sessionID string, items []*grant_entity.GrantItem) error
	ListItems(ctx context.Context, sessionID string) ([]*grant_entity.GrantItem, error)
	// ListApprovedItems 获取某个会话下所有已批准且未过期的 grant items。过期过滤
	// （expiretime = 0 表示未单独设期，跟随会话上限）收敛在这里：所有授权消费方
	// （AI exec / opsctl / policy show）看到的就是同一份"仍有效"集合。
	ListApprovedItems(ctx context.Context, sessionID string) ([]*grant_entity.GrantItem, error)
}

var defaultGrant GrantRepo

// Grant 获取 GrantRepo 实例
func Grant() GrantRepo {
	return defaultGrant
}

// RegisterGrant 注册 GrantRepo 实现
func RegisterGrant(i GrantRepo) {
	defaultGrant = i
}

// grantRepo 默认实现
type grantRepo struct{}

// NewGrant 创建默认实现
func NewGrant() GrantRepo {
	return &grantRepo{}
}

func (r *grantRepo) CreateSession(ctx context.Context, session *grant_entity.GrantSession) error {
	return db.Ctx(ctx).Create(session).Error
}

func (r *grantRepo) GetSession(ctx context.Context, id string) (*grant_entity.GrantSession, error) {
	var session grant_entity.GrantSession
	if err := db.Ctx(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *grantRepo) UpdateSessionStatus(ctx context.Context, id string, status int) error {
	return db.Ctx(ctx).Model(&grant_entity.GrantSession{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"updatetime": time.Now().Unix(),
		}).Error
}

func (r *grantRepo) CreateItems(ctx context.Context, items []*grant_entity.GrantItem) error {
	if len(items) == 0 {
		return nil
	}
	return db.Ctx(ctx).Create(items).Error
}

func (r *grantRepo) ListItems(ctx context.Context, sessionID string) ([]*grant_entity.GrantItem, error) {
	var items []*grant_entity.GrantItem
	if err := db.Ctx(ctx).Where("grant_session_id = ?", sessionID).
		Order("item_index ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *grantRepo) UpdateItems(ctx context.Context, sessionID string, items []*grant_entity.GrantItem) error {
	// 删除旧 items 并重建
	if err := db.Ctx(ctx).Where("grant_session_id = ?", sessionID).Delete(&grant_entity.GrantItem{}).Error; err != nil {
		return err
	}
	if len(items) > 0 {
		return db.Ctx(ctx).Create(items).Error
	}
	return nil
}

func (r *grantRepo) ListApprovedItems(ctx context.Context, sessionID string) ([]*grant_entity.GrantItem, error) {
	var items []*grant_entity.GrantItem
	// 查找该 sessionID 关联的所有已批准 grant 的 items；expiretime=0 未单独设期，
	// 否则只取未过期的（限时全量授权）。
	if err := db.Ctx(ctx).
		Joins("JOIN grant_sessions ON grant_sessions.id = grant_items.grant_session_id").
		Where("grant_sessions.status = ? AND grant_items.grant_session_id = ? AND (grant_items.expiretime = 0 OR grant_items.expiretime > ?)",
			grant_entity.GrantStatusApproved, sessionID, time.Now().Unix()).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
