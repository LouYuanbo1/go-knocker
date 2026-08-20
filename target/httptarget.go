package target

import (
	"fmt"
	"io"
	"net/http"
)

// HTTPTarget 通过 HTTP 请求检查目标健康状态。
type HTTPTarget struct {
	Client           *http.Client
	Name             string
	URL              string
	Method           string
	Body             io.Reader
	ContentType      string
	ExpectedStatus   int
	ExpectedResponse *string
}

// HTTPOption 用于配置 HTTPTarget。
type HTTPOption func(*HTTPTarget)

// NewHTTPTarget 创建 HTTPTarget，必要参数为 name、url、method、expectedStatus、client，其余通过 Option 配置。
// client 为 nil 时使用 http.DefaultClient。
func NewHTTPTarget(client *http.Client, name, url, method string, expectedStatus int, opts ...HTTPOption) *HTTPTarget {
	if client == nil {
		client = http.DefaultClient
	}
	t := &HTTPTarget{
		Name:           name,
		URL:            url,
		Method:         method,
		ExpectedStatus: expectedStatus,
		Client:         client,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func WithBody(body io.Reader) HTTPOption {
	return func(t *HTTPTarget) { t.Body = body }
}

func WithContentType(ct string) HTTPOption {
	return func(t *HTTPTarget) { t.ContentType = ct }
}

func WithExpectedResponse(expected string) HTTPOption {
	return func(t *HTTPTarget) { t.ExpectedResponse = &expected }
}

func (t *HTTPTarget) GetName() string {
	return t.Name
}

// Check 执行 HTTP 健康检查。
func (t *HTTPTarget) Check() error {
	req, err := http.NewRequest(t.Method, t.URL, t.Body)
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
	if t.ExpectedResponse != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if string(bodyBytes) != *t.ExpectedResponse {
			return fmt.Errorf("unexpected response body: expected %d bytes, got %d bytes", len(*t.ExpectedResponse), len(bodyBytes))
		}
	}
	return nil
}
