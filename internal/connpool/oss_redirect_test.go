package connpool

import "testing"

// redirectRegion 推导 V4 签名区域：桶在 cn-beijing 而资产配 cn-hangzhou 时，
// PermanentRedirect 给出的 oss-cn-beijing.aliyuncs.com 必须以 cn-beijing 签名，
// 沿用旧区域会被阿里云拒签。非 aliyuncs 形态返回空串（minio 自行解析）。
func TestRedirectRegion(t *testing.T) {
	cases := []struct{ endpoint, want string }{
		{"oss-cn-beijing.aliyuncs.com", "cn-beijing"},
		{"oss-cn-hangzhou-internal.aliyuncs.com", "cn-hangzhou-internal"},
		{"https://oss-ap-southeast-1.aliyuncs.com", "ap-southeast-1"},
		{"s3.example.com", ""},
		{"storage.internal", ""},
	}
	for _, c := range cases {
		if got := redirectRegion(c.endpoint); got != c.want {
			t.Fatalf("redirectRegion(%q) = %q, want %q", c.endpoint, got, c.want)
		}
	}
}
