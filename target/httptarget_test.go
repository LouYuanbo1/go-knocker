package target

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHTTPTarget_NilClient_UsesDefaultClient(t *testing.T) {
	target := NewHTTPTarget(nil, "test", "http://example.com", "GET", 200)
	assert.Equal(t, http.DefaultClient, target.Client)
	assert.Equal(t, "test", target.Name)
	assert.Equal(t, "http://example.com", target.URL)
	assert.Equal(t, "GET", target.Method)
	assert.Equal(t, 200, target.ExpectedStatus)
}

func TestNewHTTPTarget_CustomClient(t *testing.T) {
	client := &http.Client{}
	target := NewHTTPTarget(client, "test", "http://example.com", "GET", 200)
	assert.Same(t, client, target.Client)
}

func TestWithBody(t *testing.T) {
	body := strings.NewReader("hello")
	target := NewHTTPTarget(http.DefaultClient, "test", "http://example.com", "POST", 201, WithBody(body))
	assert.Same(t, body, target.Body)
}

func TestWithContentType(t *testing.T) {
	target := NewHTTPTarget(http.DefaultClient, "test", "http://example.com", "POST", 201, WithContentType("application/json"))
	assert.Equal(t, "application/json", target.ContentType)
}

func TestWithExpectedResponse(t *testing.T) {
	target := NewHTTPTarget(http.DefaultClient, "test", "http://example.com", "GET", 200, WithExpectedResponse("ok"))
	assert.NotNil(t, target.ExpectedResponse)
	assert.Equal(t, "ok", *target.ExpectedResponse)
}

func TestHTTPTarget_GetName(t *testing.T) {
	target := NewHTTPTarget(nil, "my-service", "http://example.com", "GET", 200)
	assert.Equal(t, "my-service", target.GetName())
}

func TestHTTPTarget_Check_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(200)
	}))
	defer server.Close()

	target := NewHTTPTarget(server.Client(), "test", server.URL, "GET", 200)
	err := target.Check()
	assert.NoError(t, err)
}

func TestHTTPTarget_Check_WrongStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	target := NewHTTPTarget(server.Client(), "test", server.URL, "GET", 200)
	err := target.Check()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code")
	assert.Contains(t, err.Error(), "expected 200")
	assert.Contains(t, err.Error(), "got 500")
}

func TestHTTPTarget_Check_ExpectedResponseMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "hello")
	}))
	defer server.Close()

	target := NewHTTPTarget(server.Client(), "test", server.URL, "GET", 200, WithExpectedResponse("hello"))
	err := target.Check()
	assert.NoError(t, err)
}

func TestHTTPTarget_Check_ExpectedResponseMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "world")
	}))
	defer server.Close()

	target := NewHTTPTarget(server.Client(), "test", server.URL, "GET", 200, WithExpectedResponse("hello"))
	err := target.Check()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response body")
}

func TestHTTPTarget_Check_ConnectionRefused(t *testing.T) {
	target := NewHTTPTarget(http.DefaultClient, "test", "http://127.0.0.1:1", "GET", 200)
	err := target.Check()
	assert.Error(t, err)
}

func TestHTTPTarget_Check_POSTWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, `{"key":"value"}`, string(body))
		w.WriteHeader(201)
	}))
	defer server.Close()

	body := strings.NewReader(`{"key":"value"}`)
	target := NewHTTPTarget(server.Client(), "test", server.URL, "POST", 201,
		WithBody(body),
		WithContentType("application/json"),
	)
	err := target.Check()
	assert.NoError(t, err)
}

func TestHTTPTarget_MultipleOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "pong")
	}))
	defer server.Close()

	target := NewHTTPTarget(server.Client(), "test", server.URL, "GET", 200,
		WithExpectedResponse("pong"),
		WithContentType("text/plain"),
	)
	assert.Equal(t, "text/plain", target.ContentType)
	assert.Equal(t, "pong", *target.ExpectedResponse)
	err := target.Check()
	assert.NoError(t, err)
}
