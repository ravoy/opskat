package connpool

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cago-frame/cago/pkg/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/octays/opskat/internal/assetconn"
	"github.com/octays/opskat/internal/model/entity/asset_entity"
	"github.com/octays/opskat/internal/pkg/netdial"
	"go.uber.org/zap"
)

func init() {
	assetconn.RegisterInvalidator("oss", func(_ context.Context, assetID int64) error {
		InvalidateOSS(assetID)
		return nil
	})
}

// buildMinioOptions 从 OSS 配置 + 解密后的密钥推导 minio 端点与选项(纯函数,单测)。
func buildMinioOptions(ctx context.Context, cfg *asset_entity.OSSConfig, secret string) (string, *minio.Options, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return "", nil, fmt.Errorf("oss endpoint is empty")
	}
	secure := cfg.UseSSL
	if strings.Contains(endpoint, "://") {
		u, err := url.Parse(endpoint)
		if err != nil {
			return "", nil, fmt.Errorf("parse oss endpoint: %w", err)
		}
		secure = u.Scheme == "https"
		endpoint = u.Host
	}
	// use_path_style is an explicit addressing-mode choice. BucketLookupAuto
	// falls back to path-style for providers it does not recognize (including
	// Tencent COS), so it would silently ignore the user's virtual-hosted choice.
	lookup := minio.BucketLookupDNS
	if cfg.UsePathStyle {
		lookup = minio.BucketLookupPath
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = netdial.Default().DialContext
	if cfg.ConnectTimeout > 0 {
		timeout := time.Duration(cfg.ConnectTimeout) * time.Second
		transport.DialContext = (&netdial.Dialer{Dialer: net.Dialer{Timeout: timeout}}).DialContext
		transport.TLSHandshakeTimeout = timeout
	}
	if cfg.ProxyChain != nil {
		dial, err := ProxyChainDialContext(ctx, cfg.ProxyChain)
		if err != nil {
			return "", nil, fmt.Errorf("resolve oss proxy chain: %w", err)
		}
		transport.DialContext = dial
	}
	if cfg.SkipTLSVerify {
		// explicit user setting for private S3-compatible endpoints (gosec G402 is excluded project-wide)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKeyID, secret, ""),
		Secure:       secure,
		Region:       cfg.Region,
		BucketLookup: lookup,
		Transport:    transport,
	}, nil
}

// DialOSS 新建一个 minio 客户端(HTTP 客户端,无需显式 Close)。
func DialOSS(ctx context.Context, cfg *asset_entity.OSSConfig, secret string) (*minio.Client, error) {
	endpoint, opts, err := buildMinioOptions(ctx, cfg, secret)
	if err != nil {
		return nil, err
	}
	client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("dial oss: %w", err)
	}
	return client, nil
}

type ossPool struct {
	mu         sync.Mutex
	clients    map[int64]*minio.Client
	transports map[int64]http.RoundTripper
}

var globalOSSPool = &ossPool{clients: map[int64]*minio.Client{}, transports: map[int64]http.RoundTripper{}}

// GetOrDialOSS 返回缓存的 minio 客户端,没有则新建并缓存。
func GetOrDialOSS(ctx context.Context, assetID int64, cfg *asset_entity.OSSConfig, secret string) (*minio.Client, error) {
	c, _, err := GetOrDialOSSWithTransport(ctx, assetID, cfg, secret)
	return c, err
}

func GetOrDialOSSWithTransport(ctx context.Context, assetID int64, cfg *asset_entity.OSSConfig, secret string) (*minio.Client, http.RoundTripper, error) {
	logger.Ctx(ctx).Info("oss connection open start", zap.Int64("assetId", assetID))
	globalOSSPool.mu.Lock()
	defer globalOSSPool.mu.Unlock()
	if c, ok := globalOSSPool.clients[assetID]; ok {
		logger.Ctx(ctx).Info("oss connection open end", zap.Int64("assetId", assetID), zap.Bool("cached", true))
		return c, globalOSSPool.transports[assetID], nil
	}
	endpoint, opts, err := buildMinioOptions(ctx, cfg, secret)
	if err != nil {
		return nil, nil, err
	}
	c, err := minio.New(endpoint, opts)
	if err != nil {
		logger.Ctx(ctx).Error("oss connection open fail", zap.Int64("assetId", assetID), zap.Error(err))
		return nil, nil, err
	}
	if assetID > 0 {
		globalOSSPool.clients[assetID] = c
		globalOSSPool.transports[assetID] = opts.Transport
	}
	logger.Ctx(ctx).Info("oss connection open end", zap.Int64("assetId", assetID), zap.Bool("cached", false))
	return c, opts.Transport, nil
}

// InvalidateOSS 丢弃某资产的缓存客户端(配置更新/删除时调用)。
func InvalidateOSS(assetID int64) {
	logger.Default().Info("oss connection close start", zap.Int64("assetId", assetID))
	globalOSSPool.mu.Lock()
	delete(globalOSSPool.clients, assetID)
	delete(globalOSSPool.transports, assetID)
	globalOSSPool.mu.Unlock()
	logger.Default().Info("oss connection close end", zap.Int64("assetId", assetID))
}

// DialOSSEndpoint 以同一份凭据与传输配置拨一个「指定 endpoint」的 minio 客户端。
// 供 oss_svc 在收到阿里云 PermanentRedirect（桶在另一区域）时按响应里给出的
// 真实 endpoint 重签重试；endpoint 形如 oss-cn-beijing.aliyuncs.com（无 scheme，
// 沿用 cfg 的 TLS 选择），带 scheme 时以 scheme 为准。
func DialOSSEndpoint(ctx context.Context, cfg *asset_entity.OSSConfig, secret, endpoint string) (*minio.Client, error) {
	redirected := *cfg
	redirected.Endpoint = endpoint
	// scheme 在 buildMinioOptions 里决定 secure；无 scheme 的重定向目标沿用原配置。
	if !strings.Contains(endpoint, "://") && strings.Contains(cfg.Endpoint, "://") {
		if u, err := url.Parse(cfg.Endpoint); err == nil {
			redirected.Endpoint = u.Scheme + "://" + endpoint
		}
	}
	endpoint, opts, err := buildMinioOptions(ctx, &redirected, secret)
	if err != nil {
		return nil, err
	}
	return minio.New(endpoint, opts)
}
