package oss_svc

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
)

// minioAdapter 把 *minio.Client 适配成窄接口 Client。
type minioAdapter struct {
	mc         *minio.Client
	httpClient *http.Client
	partSizeMB int

	// redirectDial 在桶实际位于另一区域（阿里云 PermanentRedirect）时按响应
	// 给出的真实 endpoint 另拨客户端；nil 则不跟随（测试默认不设）。
	redirectDial func(endpoint string) (*minio.Client, error)
	redirectMu   sync.Mutex
	redirected   map[string]*minio.Client
}

func newMinioAdapter(mc *minio.Client, partSizeMB ...int) *minioAdapter {
	return newMinioAdapterWithHTTP(mc, http.DefaultClient, partSizeMB...)
}

func newMinioAdapterWithHTTP(mc *minio.Client, httpClient *http.Client, partSizeMB ...int) *minioAdapter {
	a := &minioAdapter{mc: mc, httpClient: httpClient, redirected: map[string]*minio.Client{}}
	if len(partSizeMB) > 0 {
		a.partSizeMB = partSizeMB[0]
	}
	return a
}

func (a *minioAdapter) withRedirectDial(fn func(endpoint string) (*minio.Client, error)) *minioAdapter {
	a.redirectDial = fn
	return a
}

// clientForEndpoint 取重定向 endpoint 的客户端（首个桶区域解析结果复用）。
func (a *minioAdapter) clientForEndpoint(endpoint string) (*minio.Client, error) {
	a.redirectMu.Lock()
	defer a.redirectMu.Unlock()
	if c, ok := a.redirected[endpoint]; ok {
		return c, nil
	}
	c, err := a.redirectDial(endpoint)
	if err != nil {
		return nil, err
	}
	a.redirected[endpoint] = c
	return c, nil
}

func (a *minioAdapter) ListBuckets(ctx context.Context) ([]BucketItem, error) {
	bs, err := a.mc.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]BucketItem, 0, len(bs))
	for _, b := range bs {
		out = append(out, BucketItem{Name: b.Name, CreationDate: b.CreationDate.Unix()})
	}
	return out, nil
}

func (a *minioAdapter) ListObjects(
	ctx context.Context, bucket, prefix string, maxKeys int, continuationToken string,
) (*ListObjectsPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	page, err := a.listOnce(ctx, a.mc, bucket, prefix, maxKeys, continuationToken)
	if endpoint, redirect := permanentRedirectEndpoint(err); redirect {
		// 桶在另一区域（资产 endpoint 与桶实际区域不一致）。minio 客户端的常规
		// 操作自行解析桶区域，这条 Presign+裸 HTTP 的路径不会——按响应给出的
		// 真实 endpoint 重签重试一次；解析结果按 endpoint 缓存，翻页直达。
		if a.redirectDial == nil {
			return nil, err
		}
		c, dialErr := a.clientForEndpoint(endpoint)
		if dialErr != nil {
			return nil, dialErr
		}
		return a.listOnce(ctx, c, bucket, prefix, maxKeys, continuationToken)
	}
	return page, err
}

// listOnce 用指定客户端签名并取一页列表。
func (a *minioAdapter) listOnce(
	ctx context.Context, mc *minio.Client, bucket, prefix string, maxKeys int, continuationToken string,
) (*ListObjectsPage, error) {
	params := url.Values{"list-type": {"2"}, "encoding-type": {"url"}, "delimiter": {"/"}, "prefix": {prefix}}
	if continuationToken != "" {
		params.Set("continuation-token", continuationToken)
	}
	if maxKeys > 0 {
		params.Set("max-keys", strconv.Itoa(maxKeys))
	}
	u, err := mc.Presign(ctx, http.MethodGet, bucket, "", time.Minute, params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newS3StatusError("list objects", resp, raw)
	}
	var result minio.ListBucketV2Result
	if err := xml.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result.IsTruncated && result.NextContinuationToken == "" {
		return nil, fmt.Errorf("list objects response is truncated without a continuation token")
	}
	items := make([]ObjectItem, 0, len(result.Contents)+len(result.CommonPrefixes))
	for _, obj := range result.Contents {
		if obj.Key, err = decodeListedName(obj.Key, result.EncodingType); err != nil {
			return nil, err
		}
		items = append(items, toObjectItem(obj))
	}
	for _, p := range result.CommonPrefixes {
		name, err := decodeListedName(p.Prefix, result.EncodingType)
		if err != nil {
			return nil, err
		}
		items = append(items, ObjectItem{Key: name, IsPrefix: true})
	}
	return &ListObjectsPage{
		Items: items, IsTruncated: result.IsTruncated,
		NextContinuationToken: result.NextContinuationToken,
	}, nil
}

// decodeListedName 还原列表响应里的 key / 前缀。请求带了 encoding-type=url,服务端在响应里
// 以 EncodingType 声明它确实编码了;各家编码范围不同(R2 连 "/" 也编码成 %2F),只按声明解码。
func decodeListedName(name, encodingType string) (string, error) {
	if encodingType != "url" {
		return name, nil
	}
	decoded, err := url.QueryUnescape(name)
	if err != nil {
		return "", fmt.Errorf("list objects: decode %q: %w", name, err)
	}
	return decoded, nil
}

func (a *minioAdapter) StatObject(ctx context.Context, bucket, key string) (ObjectItem, error) {
	info, err := a.mc.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return ObjectItem{}, err
	}
	return toObjectItem(info), nil
}

func (a *minioAdapter) RemoveObject(ctx context.Context, bucket, key string) error {
	return a.mc.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

func (a *minioAdapter) CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	_, err := a.mc.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: dstBucket, Object: dstKey},
		minio.CopySrcOptions{Bucket: srcBucket, Object: srcKey},
	)
	return err
}

func (a *minioAdapter) RemoveObjects(ctx context.Context, bucket string, keys []string) error {
	objCh := make(chan minio.ObjectInfo, len(keys))
	for _, k := range keys {
		objCh <- minio.ObjectInfo{Key: k}
	}
	close(objCh)
	var errs []minio.RemoveObjectError
	for rerr := range a.mc.RemoveObjects(ctx, bucket, objCh, minio.RemoveObjectsOptions{}) {
		errs = append(errs, rerr)
	}
	return aggregateRemoveErrors(errs)
}

// aggregateRemoveErrors 把 minio 批量删除通道里的逐项错误聚合成一个显式报告;
// 空则 nil。绝不静默丢弃部分失败。
func aggregateRemoveErrors(errs []minio.RemoveObjectError) error {
	if len(errs) == 0 {
		return nil
	}
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, fmt.Sprintf("%s: %v", e.ObjectName, e.Err))
	}
	return fmt.Errorf("批量删除部分失败(%d): %s", len(errs), strings.Join(parts, "; "))
}

func (a *minioAdapter) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{ContentType: contentType}
	if a.partSizeMB > 0 {
		opts.PartSize = uint64(a.partSizeMB) * 1024 * 1024
	}
	_, err := a.mc.PutObject(ctx, bucket, key, r, size, opts)
	return err
}

func (a *minioAdapter) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, int64, error) {
	obj, err := a.mc.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}
	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, 0, err
	}
	return obj, info.Size, nil
}

func (a *minioAdapter) PresignGet(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := a.mc.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (a *minioAdapter) PresignPut(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := a.mc.PresignedPutObject(ctx, bucket, key, expiry)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (a *minioAdapter) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return a.mc.BucketExists(ctx, bucket)
}

// toObjectItem 把 minio.ObjectInfo 映射为 DTO;"文件夹"前缀表现为 Key 以 "/" 结尾且 Size 为 0。
func toObjectItem(o minio.ObjectInfo) ObjectItem {
	item := ObjectItem{
		Key: o.Key, Size: o.Size, ETag: o.ETag,
		StorageClass: o.StorageClass, ContentType: o.ContentType,
	}
	if !o.LastModified.IsZero() {
		item.LastModified = o.LastModified.Unix()
	}
	if o.Size == 0 && len(o.Key) > 0 && o.Key[len(o.Key)-1] == '/' {
		item.IsPrefix = true
	}
	return item
}

// s3StatusError 保留非 200 响应的状态码、错误体摘要，以及（阿里云
// PermanentRedirect 时）桶实际所在区域的 endpoint——文本形式给用户看，
// 结构字段给重试逻辑用。
type s3StatusError struct {
	what     string
	status   string
	summary  string
	endpoint string
}

func newS3StatusError(what string, resp *http.Response, body []byte) error {
	e := &s3StatusError{what: what, status: resp.Status, summary: describeS3ErrorBytes(body)}
	var payload struct {
		Code     string `xml:"Code"`
		Endpoint string `xml:"Endpoint"`
	}
	_ = xml.Unmarshal(body, &payload)
	if payload.Code == "PermanentRedirect" && payload.Endpoint != "" {
		e.endpoint = payload.Endpoint
	}
	return e
}

func (e *s3StatusError) Error() string {
	return fmt.Sprintf("%s: unexpected HTTP status %s: %s", e.what, e.status, e.summary)
}

// permanentRedirectEndpoint 报告错误是否为「桶在另一区域」并给出真实 endpoint。
func permanentRedirectEndpoint(err error) (endpoint string, ok bool) {
	var e *s3StatusError
	if errors.As(err, &e) {
		return e.endpoint, e.endpoint != ""
	}
	return "", false
}

func describeS3Error(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		return "no error body"
	}
	return describeS3ErrorBytes(data)
}

func describeS3ErrorBytes(data []byte) string {
	if len(data) == 0 {
		return "no error body"
	}
	var e struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	}
	if xml.Unmarshal(data, &e) != nil || e.Code == "" {
		return strings.TrimSpace(string(data))
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}
