package target

import (
	"cmp"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPTarget 通过 HTTP 请求检查目标健康状态。
type HTTPTarget struct {
	Client           *http.Client
	Name             string
	URL              string
	Method           string
	Body             string
	ContentType      string
	ExpectedStatus   int
	ExpectedResponse string
}

// HTTPOption 用于配置 HTTPTarget。
type HTTPOption func(*HTTPTarget)

// NewHTTPTarget 创建 HTTPTarget，必要参数为 client、name、url、method、expectedStatus，其余通过 Option 配置。
// client 为 nil 时使用 http.DefaultClient；expectedStatus 为零值时默认 http.StatusOK。
func NewHTTPTarget(client *http.Client, name, url, method string, expectedStatus int, opts ...HTTPOption) *HTTPTarget {
	if expectedStatus <= 0 {
		expectedStatus = http.StatusOK
	}
	t := &HTTPTarget{
		Name:           name,
		URL:            url,
		Method:         method,
		ExpectedStatus: expectedStatus,
		Client:         cmp.Or(client, http.DefaultClient),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// WithBody 设置请求体。以字符串保存，每次 Check 都会基于它重建新的 reader，
// 因此主检查与重试之间不会共享已消费的 Reader，重复请求也不会发空 body。
func WithBody(body string) HTTPOption {
	return func(t *HTTPTarget) { t.Body = body }
}

func WithContentType(ct string) HTTPOption {
	return func(t *HTTPTarget) { t.ContentType = ct }
}

func WithExpectedResponse(expected string) HTTPOption {
	return func(t *HTTPTarget) { t.ExpectedResponse = expected }
}

func (t *HTTPTarget) GetName() string {
	return t.Name
}

// Check 执行 HTTP 健康检查。
func (t *HTTPTarget) Check() error {
	var body io.Reader
	if t.Body != "" {
		body = strings.NewReader(t.Body) // 每次重建，重试时可复用完整的 body
	}
	req, err := http.NewRequest(t.Method, t.URL, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if t.ContentType != "" {
		req.Header.Set("Content-Type", t.ContentType)
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != t.ExpectedStatus {
		return fmt.Errorf("unexpected status code: expected %d, got %d", t.ExpectedStatus, resp.StatusCode)
	}
	if t.ExpectedResponse != "" {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if string(bodyBytes) != t.ExpectedResponse {
			return fmt.Errorf("unexpected response body: expected %d bytes, got %d bytes", len(t.ExpectedResponse), len(bodyBytes))
		}
	}
	return nil
}
