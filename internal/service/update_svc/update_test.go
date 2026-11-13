package update_svc

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

func TestHasUpdate(t *testing.T) {
	convey.Convey("更新判断", t, func() {
		convey.Convey("dev 或空版本始终有更新", func() {
			assert.True(t, hasUpdate(ChannelStable, "dev", "v1.0.0"))
			assert.True(t, hasUpdate(ChannelStable, "", "v1.0.0"))
		})

		convey.Convey("stable 通道", func() {
			convey.Convey("有新版本", func() {
				assert.True(t, hasUpdate(ChannelStable, "v1.0.0", "v1.0.1"))
				assert.True(t, hasUpdate(ChannelStable, "v1.0.0", "v2.0.0"))
			})

			convey.Convey("同版本无更新", func() {
				assert.False(t, hasUpdate(ChannelStable, "v1.0.0", "v1.0.0"))
			})

			convey.Convey("远端版本更旧无更新", func() {
				assert.False(t, hasUpdate(ChannelStable, "v1.0.1", "v1.0.0"))
			})

			convey.Convey("当前是 nightly 切换到 stable 始终更新", func() {
				assert.True(t, hasUpdate(ChannelStable, "v1.0.0-nightly.20260325", "v1.0.0"))
			})
		})

		convey.Convey("beta 通道", func() {
			convey.Convey("有新 beta 版本", func() {
				assert.True(t, hasUpdate(ChannelBeta, "v1.0.0-beta.1", "v1.0.0-beta.2"))
			})

			convey.Convey("当前是 nightly 切换到 beta 始终更新", func() {
				assert.True(t, hasUpdate(ChannelBeta, "v1.0.0-nightly.20260325", "v1.0.0-beta.1"))
			})
		})

		convey.Convey("nightly 通道", func() {
			convey.Convey("从 stable 切换到 nightly 始终更新", func() {
				assert.True(t, hasUpdate(ChannelNightly, "v1.0.0", "v1.0.0-nightly.20260325"))
			})

			convey.Convey("旧格式 nightly 字符串比较", func() {
				assert.True(t, hasUpdate(ChannelNightly, "nightly-20260324-abc", "nightly-20260325-def"))
				assert.False(t, hasUpdate(ChannelNightly, "nightly-20260325-abc", "nightly-20260325-abc"))
			})

			convey.Convey("新格式 nightly 语义化比较", func() {
				assert.True(t, hasUpdate(ChannelNightly, "v1.0.0-nightly.20260324", "v1.0.0-nightly.20260325"))
				assert.False(t, hasUpdate(ChannelNightly, "v1.0.0-nightly.20260325", "v1.0.0-nightly.20260325"))
				assert.False(t, hasUpdate(ChannelNightly, "v1.0.0-nightly.20260326", "v1.0.0-nightly.20260325"))
			})
		})
	})
}

func TestFetchChecksumsErrorPrefix(t *testing.T) {
	convey.Convey("校验文件获取失败返回特定前缀", t, func() {
		convey.Convey("空 assets 返回 nil（兼容旧版本）", func() {
			checksums, err := FetchChecksums(nil)
			assert.NoError(t, err)
			assert.Nil(t, checksums)
		})

		convey.Convey("无 SHA256SUMS.txt asset 返回 nil", func() {
			assets := []ReleaseAsset{
				{Name: "opskat-v1.0.0-darwin-arm64.dmg", BrowserDownloadURL: "https://example.com/file.dmg"},
			}
			checksums, err := FetchChecksums(assets)
			assert.NoError(t, err)
			assert.Nil(t, checksums)
		})
	})
}

func TestReleaseInfoDownloadURL(t *testing.T) {
	convey.Convey("release-info.json URL 构造", t, func() {
		convey.Convey("stable 通道", func() {
			url := releaseInfoURL(ChannelStable)
			assert.Equal(t, "https://github.com/"+githubRepo+"/releases/latest/download/release-info.json", url)
		})

		convey.Convey("nightly 通道", func() {
			url := releaseInfoURL(ChannelNightly)
			assert.Equal(t, "https://github.com/"+githubRepo+"/releases/download/nightly/release-info.json", url)
		})

		convey.Convey("beta 通道返回空（不支持镜像回退）", func() {
			url := releaseInfoURL(ChannelBeta)
			assert.Equal(t, "", url)
		})
	})
}

func TestParseChecksums(t *testing.T) {
	convey.Convey("解析 SHA256SUMS.txt", t, func() {
		convey.Convey("正常格式", func() {
			input := "abc123def456  opskat-1.0.0-darwin-arm64.dmg\n" +
				"789abc012def  opskat-1.0.0-linux-amd64.tar.gz\n"
			result := parseChecksums(input)
			assert.Equal(t, "abc123def456", result["opskat-1.0.0-darwin-arm64.dmg"])
			assert.Equal(t, "789abc012def", result["opskat-1.0.0-linux-amd64.tar.gz"])
		})

		convey.Convey("忽略空行", func() {
			input := "abc123  file1.tar.gz\n\n789def  file2.dmg\n"
			result := parseChecksums(input)
			assert.Len(t, result, 2)
		})

		convey.Convey("忽略格式不正确的行", func() {
			input := "abc123  file1.tar.gz\nbadline\nabc123  file2.tar.gz\n"
			result := parseChecksums(input)
			assert.Len(t, result, 2)
		})

		convey.Convey("空输入", func() {
			result := parseChecksums("")
			assert.Empty(t, result)
		})

		convey.Convey("单空格分隔也支持", func() {
			input := "abc123 file1.tar.gz\n"
			result := parseChecksums(input)
			assert.Equal(t, "abc123", result["file1.tar.gz"])
		})

		convey.Convey("二进制模式 * 前缀", func() {
			input := "abc123 *file1.tar.gz\n"
			result := parseChecksums(input)
			assert.Equal(t, "abc123", result["file1.tar.gz"])
		})
	})
}

// describeAPIError 必须把 GitHub 的官方原因带出来：国内出口 IP 未认证限流（403）
// 是自动更新最常见的失败，只报状态码用户无法区分限流/仓库问题/网络问题。
func TestDescribeAPIErrorCarriesGitHubReason(t *testing.T) {
	convey.Convey("GitHub API 错误描述", t, func() {
		makeResp := func(status int, body string, reset string, remaining string) *http.Response {
			resp := &http.Response{
				StatusCode: status,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(body)),
			}
			if reset != "" {
				resp.Header.Set("X-RateLimit-Reset", reset)
				resp.Header.Set("X-RateLimit-Remaining", remaining)
			}
			return resp
		}

		convey.Convey("限流响应带 message 与重置提示", func() {
			reset := strconv.FormatInt(time.Now().Add(40*time.Minute).Unix(), 10)
			err := describeAPIError(makeResp(http.StatusForbidden,
				`{"message":"API rate limit exceeded for 1.2.3.4."}`, reset, "0"))
			assert.ErrorContains(t, err, "status 403")
			assert.ErrorContains(t, err, "API rate limit exceeded")
			assert.ErrorContains(t, err, "rate limit resets in")
		})

		convey.Convey("非限流错误只带 message", func() {
			err := describeAPIError(makeResp(http.StatusNotFound,
				`{"message":"Not Found"}`, "", ""))
			assert.ErrorContains(t, err, "status 404")
			assert.ErrorContains(t, err, "Not Found")
			assert.NotContains(t, err.Error(), "rate limit resets")
		})

		convey.Convey("无 JSON 体的响应退回纯状态码", func() {
			err := describeAPIError(makeResp(http.StatusBadGateway, "upstream down", "", ""))
			assert.ErrorContains(t, err, "status 502")
		})
	})
}
