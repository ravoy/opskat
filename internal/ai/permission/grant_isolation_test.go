package permission

import (
	"context"
	"testing"
	"time"

	"github.com/octays/opskat/internal/ai/aictx"
	"github.com/octays/opskat/internal/ai/policy"
	"github.com/octays/opskat/internal/model/entity/asset_entity"
	"github.com/octays/opskat/internal/repository/asset_repo"
	"github.com/octays/opskat/internal/repository/asset_repo/mock_asset_repo"
	"github.com/octays/opskat/internal/repository/grant_repo"

	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

// withStubGrant 注册 stubGrantRepo 与一个只认 assetID=1 的 mock asset repo，
// 返回带 sessionID 的 ctx。两件事都是必须的：grant 匹配依赖 aictx.GetSessionID
// （不注入会直接返回空串，测试假绿），而按资产的匹配链会解析资产拿组链
// （asset_repo 未注册时是空指针崩溃，不是失败）。
func withStubGrant(t *testing.T) context.Context {
	stub := newStubGrantRepo()
	origGrant := grant_repo.Grant()
	grant_repo.RegisterGrant(stub)

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockAsset := mock_asset_repo.NewMockAssetRepo(ctrl)
	mockAsset.EXPECT().Find(gomock.Any(), int64(1)).
		Return(&asset_entity.Asset{ID: 1, Name: "web-01", Type: asset_entity.AssetTypeSSH}, nil).AnyTimes()
	origAsset := asset_repo.Asset()
	asset_repo.RegisterAsset(mockAsset)

	t.Cleanup(func() {
		if origGrant != nil {
			grant_repo.RegisterGrant(origGrant)
		}
		if origAsset != nil {
			asset_repo.RegisterAsset(origAsset)
		}
	})
	return aictx.WithSessionID(context.Background(), "sess-cp")
}

func TestGrantIsolation(t *testing.T) {
	Convey("grant 池按工具面隔离", t, func() {
		// 用户在 cp 审批里点"本次会话允许"并把 pattern 编辑成 `*`（很常见）时，
		// 这条路径授权一旦被命令面看见，MatchCommandRule 的 `*` 会放行任意命令。
		Convey("cp 的路径授权不能放行命令", func() {
			ctx := withStubGrant(t)
			SaveGrantPattern(ctx, "sess-cp", 1, "web-01", GrantToolCp, "*")

			got := matchGrantPatternsWith(ctx, 1, nil, []string{"rm -rf /var/log"}, "exec", policy.MatchCommandRule)
			So(got, ShouldEqual, "")
		})

		Convey("整串通配 `*` 的命令授权跨面放行文件传输", func() {
			ctx := withStubGrant(t)
			SaveGrantPattern(ctx, "sess-cp", 1, "web-01", "exec", "*")

			got := matchGrantPatternsWith(ctx, 1, nil, []string{"/etc/cron.d/backup"}, GrantToolCp, policy.MatchPathRule)
			So(got, ShouldEqual, "*")
		})

		Convey("非整串通配的命令授权不能放行文件传输", func() {
			ctx := withStubGrant(t)
			SaveGrantPattern(ctx, "sess-cp", 1, "web-01", "exec", "systemctl *")

			got := matchGrantPatternsWith(ctx, 1, nil, []string{"/etc/cron.d/backup"}, GrantToolCp, policy.MatchPathRule)
			So(got, ShouldEqual, "")
		})

		Convey("存量 tool_name=exec 的行仍然为非 cp 检查生效", func() {
			ctx := withStubGrant(t)
			SaveGrantPattern(ctx, "sess-cp", 1, "web-01", "exec", "redis-cli get *")

			got := matchGrantPatternsWith(ctx, 1, nil, []string{"redis-cli get foo"}, "redis", policy.MatchCommandRule)
			So(got, ShouldEqual, "redis-cli get *")
		})
	})
}

// 限时全量授权落的 item 形状与效力：`*` 挂命令面（cp 触发也归一），AllAssets 时
// AssetID 归 0，到期时间随时长写入；随后同范围命令在 grant 匹配里直接通过。
func TestTimedAllowGrantShapeAndEffect(t *testing.T) {
	Convey("限时全量授权", t, func() {
		Convey("本资产 30 分钟", func() {
			ctx := withStubGrant(t)
			stub := grant_repo.Grant().(*stubGrantRepo)
			SaveTimedAllowGrant(ctx, "sess-cp", 1, "web-01", "exec", 30, false)

			items := stub.items["sess-cp"]
			So(len(items), ShouldEqual, 1)
			item := items[0]
			So(item.Command, ShouldEqual, "*")
			So(item.AssetID, ShouldEqual, 1)
			So(item.ToolName, ShouldEqual, "exec")
			So(item.Expiretime, ShouldBeBetween, time.Now().Add(29*time.Minute).Unix(), time.Now().Add(31*time.Minute).Unix())

			Convey("同资产的命令直接通过", func() {
				got := matchGrantPatternsWith(ctx, 1, nil, []string{"systemctl restart nginx"}, "exec", policy.MatchCommandRule)
				So(got, ShouldEqual, "*")
			})
		})

		Convey("全部资产：AssetID 归 0，其它资产也放行", func() {
			ctx := withStubGrant(t)
			stub := grant_repo.Grant().(*stubGrantRepo)
			SaveTimedAllowGrant(ctx, "sess-cp", 1, "web-01", "exec", 120, true)

			item := stub.items["sess-cp"][0]
			So(item.AssetID, ShouldEqual, 0)

			// assetID=9 未在 mock asset repo 注册——全部资产的 0/0 匹配在解析组链之前，
			// 不需要资产存在。
			got := matchGrantPatternsWith(ctx, 9, nil, []string{"rm -rf /tmp/x"}, "exec", policy.MatchCommandRule)
			So(got, ShouldEqual, "*")
		})

		Convey("cp 触发的限时授权归一到命令面，跨面覆盖文件传输", func() {
			ctx := withStubGrant(t)
			stub := grant_repo.Grant().(*stubGrantRepo)
			SaveTimedAllowGrant(ctx, "sess-cp", 1, "web-01", GrantToolCp, 60, false)

			So(stub.items["sess-cp"][0].ToolName, ShouldEqual, "exec")
			got := matchGrantPatternsWith(ctx, 1, nil, []string{"/etc/cron.d/backup"}, GrantToolCp, policy.MatchPathRule)
			So(got, ShouldEqual, "*")
		})
	})
}
