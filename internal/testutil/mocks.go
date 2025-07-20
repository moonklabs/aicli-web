package testutil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"time"
)

// MockHTTPClient HTTP 클라이언트 모킹
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(nil),
	}, nil
}

// MockDockerClient Docker 클라이언트 모킹 인터페이스
type MockDockerClient struct {
	ContainerListFunc   func(ctx context.Context) ([]interface{}, error)
	ContainerCreateFunc func(ctx context.Context, config interface{}) (string, error)
	ContainerStartFunc  func(ctx context.Context, containerID string) error
	ContainerStopFunc   func(ctx context.Context, containerID string) error
	ContainerRemoveFunc func(ctx context.Context, containerID string) error
	ContainerLogsFunc   func(ctx context.Context, containerID string) (io.ReadCloser, error)
}

func (m *MockDockerClient) ContainerList(ctx context.Context) ([]interface{}, error) {
	if m.ContainerListFunc != nil {
		return m.ContainerListFunc(ctx)
	}
	return []interface{}{}, nil
}

func (m *MockDockerClient) ContainerCreate(ctx context.Context, config interface{}) (string, error) {
	if m.ContainerCreateFunc != nil {
		return m.ContainerCreateFunc(ctx, config)
	}
	return "mock-container-id", nil
}

func (m *MockDockerClient) ContainerStart(ctx context.Context, containerID string) error {
	if m.ContainerStartFunc != nil {
		return m.ContainerStartFunc(ctx, containerID)
	}
	return nil
}

func (m *MockDockerClient) ContainerStop(ctx context.Context, containerID string) error {
	if m.ContainerStopFunc != nil {
		return m.ContainerStopFunc(ctx, containerID)
	}
	return nil
}

func (m *MockDockerClient) ContainerRemove(ctx context.Context, containerID string) error {
	if m.ContainerRemoveFunc != nil {
		return m.ContainerRemoveFunc(ctx, containerID)
	}
	return nil
}

func (m *MockDockerClient) ContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	if m.ContainerLogsFunc != nil {
		return m.ContainerLogsFunc(ctx, containerID)
	}
	return io.NopCloser(nil), nil
}

// MockFileSystem 파일시스템 작업 모킹
type MockFileSystem struct {
	CreateFunc func(name string) error
	RemoveFunc func(name string) error
	ReadFunc   func(name string) ([]byte, error)
	WriteFunc  func(name string, data []byte) error
	ExistsFunc func(name string) bool
}

func (m *MockFileSystem) Create(name string) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(name)
	}
	return nil
}

func (m *MockFileSystem) Remove(name string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(name)
	}
	return nil
}

func (m *MockFileSystem) Read(name string) ([]byte, error) {
	if m.ReadFunc != nil {
		return m.ReadFunc(name)
	}
	return []byte{}, nil
}

func (m *MockFileSystem) Write(name string, data []byte) error {
	if m.WriteFunc != nil {
		return m.WriteFunc(name, data)
	}
	return nil
}

func (m *MockFileSystem) Exists(name string) bool {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(name)
	}
	return true
}

// MockTimeProvider 시간 관련 모킹
type MockTimeProvider struct {
	NowFunc func() time.Time
}

func (m *MockTimeProvider) Now() time.Time {
	if m.NowFunc != nil {
		return m.NowFunc()
	}
	return time.Now()
}

// TestServer 테스트용 HTTP 서버 생성
func TestServer(handler http.Handler) *httptest.Server {
	return httptest.NewServer(handler)
}

// TestRequest 테스트용 HTTP 요청 생성
func TestRequest(method, path string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, path, body)
	return req
}