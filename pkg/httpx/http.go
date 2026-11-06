package httpx

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout       = 5 * time.Second
	maxResponseBodyBytes = 10 << 20 // 10 MiB
)

var defaultClient = http.Client{
	Timeout:   defaultTimeout,
	Transport: http.DefaultTransport.(*http.Transport).Clone(),
}

// SetClient 替换包级 HTTP 客户端。应在初始化阶段、发起任何请求之前调用；
// 客户端使用期间调用方不得修改它。
func SetClient(client *http.Client) error {
	if client == nil {
		return errors.New("httpx: client must not be nil")
	}
	defaultClient = *client
	return nil
}

// GetClient 返回包级 HTTP 客户端。应在初始化阶段、发起任何请求之前完成配置。
func GetClient() *http.Client {
	return &defaultClient
}

// SetInsecure 控制包级客户端是否验证 TLS 证书。应在初始化阶段、发起任何请求
// 之前调用；只能在可信的开发或测试环境中关闭证书验证。
func SetInsecure(insecure bool) error {
	transport, ok := defaultClient.Transport.(*http.Transport)
	if !ok {
		return fmt.Errorf("httpx: cannot configure TLS on transport %T", defaultClient.Transport)
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.InsecureSkipVerify = insecure //nolint:gosec
	return nil
}

// ResponseHandler 处理完整的 HTTP 响应。
type ResponseHandler func(code int, header http.Header, body []byte) error

type requestConfig struct {
	header http.Header
}

// Option 配置 HTTP 请求。
type Option func(*requestConfig) error

// WithHeader 设置一个请求头。
func WithHeader(key, value string) Option {
	return func(config *requestConfig) error {
		config.header.Set(key, value)
		return nil
	}
}

// WithHeaders 批量设置请求头。
func WithHeaders(headers map[string]string) Option {
	return func(config *requestConfig) error {
		for key, value := range headers {
			config.header.Set(key, value)
		}
		return nil
	}
}

// Get 发送 GET 请求。
func Get(ctx context.Context, requestURL string, handler ResponseHandler, options ...Option) error {
	return Request(ctx, http.MethodGet, requestURL, nil, handler, options...)
}

// PostForm 发送请求体采用 URL 编码表单的 POST 请求。
func PostForm(ctx context.Context, requestURL string, form url.Values, handler ResponseHandler, options ...Option) error {
	options = append([]Option{WithHeader("Content-Type", "application/x-www-form-urlencoded")}, options...)
	return Request(ctx, http.MethodPost, requestURL, strings.NewReader(form.Encode()), handler, options...)
}

func bodyReader(value any) (io.Reader, error) {
	if value == nil {
		return nil, nil
	}

	switch value := value.(type) {
	case []byte:
		return bytes.NewReader(value), nil
	case io.Reader:
		return value, nil
	default:
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(value); err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		return buf, nil
	}
}

func Post(ctx context.Context, requestURL string, value any, handler ResponseHandler, options ...Option) error {
	return requestWithBody(ctx, http.MethodPost, requestURL, value, handler, options...)
}

func Put(ctx context.Context, requestURL string, value any, handler ResponseHandler, options ...Option) error {
	return requestWithBody(ctx, http.MethodPut, requestURL, value, handler, options...)
}

func Delete(ctx context.Context, requestURL string, value any, handler ResponseHandler, options ...Option) error {
	return requestWithBody(ctx, http.MethodDelete, requestURL, value, handler, options...)
}

func requestWithBody(ctx context.Context, method, requestURL string, value any, handler ResponseHandler, options ...Option) error {
	body, err := bodyReader(value)
	if err != nil {
		return err
	}
	return Request(ctx, method, requestURL, body, handler, options...)
}

func PostReader(ctx context.Context, requestURL string, body io.Reader, handler ResponseHandler, options ...Option) error {
	return Request(ctx, http.MethodPost, requestURL, body, handler, options...)
}

// Request 发送 HTTP 请求。未配置响应处理器时，非 2xx 响应会作为状态错误返回。
func Request(ctx context.Context, method, requestURL string, body io.Reader, handler ResponseHandler, options ...Option) error {
	config := requestConfig{
		header: make(http.Header),
	}
	for _, option := range options {
		if option == nil {
			return errors.New("httpx: option must not be nil")
		}
		if err := option(&config); err != nil {
			return err
		}
	}

	requestURL, err := normalizeURL(requestURL)
	if err != nil {
		return fmt.Errorf("httpx: build request URL: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return fmt.Errorf("httpx: create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, values := range config.header {
		req.Header[key] = append([]string(nil), values...)
	}
	if requestID := RequestID(ctx); requestID != "" && req.Header.Get(RequestIDHeader()) == "" {
		req.Header.Set(RequestIDHeader(), requestID)
	}

	resp, err := defaultClient.Do(req)
	if err != nil {
		return NewNetErr(err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if handler == nil {
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				return NewNetErr(err)
			}
			return nil
		}
		responseBody, err := readResponseBody(resp.Body)
		if err != nil {
			return err
		}
		return &statusErr{code: resp.StatusCode, body: responseBody}
	}

	responseBody, err := readResponseBody(resp.Body)
	if err != nil {
		return err
	}
	return handler(resp.StatusCode, resp.Header.Clone(), responseBody)
}

func normalizeURL(requestURL string) (string, error) {
	if !strings.Contains(requestURL, "://") {
		requestURL = "http://" + requestURL
	}
	parsed, err := url.Parse(requestURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", errors.New("missing host")
	}
	return parsed.String(), nil
}

func readResponseBody(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, NewNetErr(err)
	}
	if len(data) > maxResponseBodyBytes {
		return nil, fmt.Errorf("httpx: response body exceeds %d bytes", maxResponseBodyBytes)
	}
	return data, nil
}

type statusErr struct {
	code int
	body []byte
}

func (e *statusErr) Error() string { return fmt.Sprintf("http status: %d", e.code) }

func IsStatusErr(err error) bool {
	var target *statusErr
	return errors.As(err, &target)
}

func StatusCode(err error) (int, bool) {
	var target *statusErr
	if !errors.As(err, &target) {
		return 0, false
	}
	return target.code, true
}

func StatusBody(err error) ([]byte, bool) {
	var target *statusErr
	if !errors.As(err, &target) {
		return nil, false
	}
	return bytes.Clone(target.body), true
}

type netErr struct{ cause error }

// NewNetErr 将 err 标记为网络错误，同时保留原始错误链。
func NewNetErr(err error) error {
	if err == nil {
		return nil
	}
	return &netErr{cause: err}
}

func (e *netErr) Error() string { return e.cause.Error() }
func (e *netErr) Unwrap() error { return e.cause }

func IsNetErr(err error) bool {
	var target *netErr
	return errors.As(err, &target)
}
