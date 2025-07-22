// Package docker Docker 클라이언트와 컨테이너 관리를 담당합니다.
package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Client Docker 클라이언트를 래핑하고 기본 설정을 관리합니다.
type Client struct {
	cli         *client.Client
	config      *Config
	networkID   string
	labelPrefix string
}

// Config Docker 클라이언트 설정 구조체
type Config struct {
	// 연결 설정
	Host    string        `yaml:"host" json:"host"`       // Docker daemon 주소
	Version string        `yaml:"version" json:"version"` // API 버전
	Timeout time.Duration `yaml:"timeout" json:"timeout"` // 연결 타임아웃

	// 기본값
	DefaultImage string   `yaml:"default_image" json:"default_image"` // 기본 이미지
	DefaultShell []string `yaml:"default_shell" json:"default_shell"` // 기본 쉘
	NetworkName  string   `yaml:"network_name" json:"network_name"`   // 네트워크 이름

	// 리소스 제한
	CPULimit    float64 `yaml:"cpu_limit" json:"cpu_limit"`       // CPU 제한 (1.0 = 1 CPU)
	MemoryLimit int64   `yaml:"memory_limit" json:"memory_limit"` // 메모리 제한 (bytes)

	// 보안 설정
	Privileged   bool     `yaml:"privileged" json:"privileged"`       // 특권 모드
	ReadOnly     bool     `yaml:"read_only" json:"read_only"`         // 읽기 전용 루트
	SecurityOpts []string `yaml:"security_opts" json:"security_opts"` // 보안 옵션
}

// NewClient 새로운 Docker 클라이언트를 생성합니다.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Docker 클라이언트 생성
	cli, err := client.NewClientWithOpts(
		client.WithHost(config.Host),
		client.WithVersion(config.Version),
		client.WithTimeout(config.Timeout),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping docker daemon: %w", err)
	}

	dockerClient := &Client{
		cli:         cli,
		config:      config,
		labelPrefix: "aicli",
	}

	// 네트워크 설정
	if err := dockerClient.setupNetwork(context.Background()); err != nil {
		return nil, fmt.Errorf("setup network: %w", err)
	}

	return dockerClient, nil
}

// DefaultConfig 기본 설정을 반환합니다.
func DefaultConfig() *Config {
	return &Config{
		Host:         client.DefaultDockerHost,
		Version:      "1.41", // Docker API 1.41
		Timeout:      30 * time.Second,
		DefaultImage: "alpine:latest",
		DefaultShell: []string{"/bin/sh"},
		NetworkName:  "aicli-network",
		CPULimit:     1.0,                   // 1 CPU
		MemoryLimit:  512 * 1024 * 1024,    // 512MB
		Privileged:   false,
		ReadOnly:     true,
		SecurityOpts: []string{"no-new-privileges:true"},
	}
}

// setupNetwork aicli 전용 네트워크를 설정합니다.
func (c *Client) setupNetwork(ctx context.Context) error {
	// 기존 네트워크 확인
	networks, err := c.cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return fmt.Errorf("list networks: %w", err)
	}

	for _, network := range networks {
		if network.Name == c.config.NetworkName {
			c.networkID = network.ID
			return nil
		}
	}

	// 새 네트워크 생성
	resp, err := c.cli.NetworkCreate(ctx, c.config.NetworkName, types.NetworkCreate{
		Driver:     "bridge",
		Attachable: true,
		Internal:   false, // 외부 인터넷 접근 허용
		Labels: map[string]string{
			c.labelKey("managed"): "true",
			c.labelKey("created"): time.Now().Format(time.RFC3339),
		},
	})
	if err != nil {
		return fmt.Errorf("create network: %w", err)
	}

	c.networkID = resp.ID
	return nil
}

// labelKey aicli 레이블 키를 생성합니다.
func (c *Client) labelKey(key string) string {
	return fmt.Sprintf("%s.%s", c.labelPrefix, key)
}

// GetNetworkID 네트워크 ID를 반환합니다.
func (c *Client) GetNetworkID() string {
	return c.networkID
}

// GetConfig 설정을 반환합니다.
func (c *Client) GetConfig() *Config {
	return c.config
}

// Close Docker 클라이언트 연결을 닫습니다.
func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}

// Ping Docker daemon과의 연결을 확인합니다.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	return err
}

// GenerateContainerName 워크스페이스 ID로부터 컨테이너 이름을 생성합니다.
func (c *Client) GenerateContainerName(workspaceID string) string {
	return fmt.Sprintf("%s-workspace-%s", c.labelPrefix, workspaceID)
}

// WorkspaceLabels 워크스페이스 컨테이너용 레이블을 생성합니다.
func (c *Client) WorkspaceLabels(workspaceID, workspaceName string) map[string]string {
	return map[string]string{
		c.labelKey("managed"):      "true",
		c.labelKey("type"):         "workspace",
		c.labelKey("workspace.id"): workspaceID,
		c.labelKey("workspace.name"): workspaceName,
		c.labelKey("created"):      time.Now().Format(time.RFC3339),
	}
}