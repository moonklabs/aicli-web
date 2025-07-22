package config

import (
	"os"
	"path/filepath"
	"time"
)

// 기본값 상수 정의
const (
	// Claude 기본값
	DefaultClaudeModel       = "claude-3-opus"
	DefaultClaudeTemperature = 0.7
	DefaultClaudeTimeout     = 30 // 30초
	DefaultClaudeMaxTokens   = 100000
	DefaultClaudeRetryCount  = 3
	DefaultClaudeRetryDelay  = 2 * time.Second

	// 워크스페이스 기본값
	DefaultMaxProjects    = 10
	DefaultIsolationMode  = "docker"

	// 출력 기본값
	DefaultOutputFormat   = "table"
	DefaultColorMode      = "auto"
	DefaultOutputWidth    = 120
	DefaultVerbosity      = 1

	// 로깅 기본값
	DefaultLogLevel      = "info"
	DefaultLogMaxSize    = 100 // MB
	DefaultLogMaxBackups = 5
	DefaultLogMaxAge     = 30 // days

	// Docker 기본값
	DefaultDockerSocketPath    = "/var/run/docker.sock"
	DefaultDockerImage         = "aicli-workspace:latest"
	DefaultDockerMemoryLimit   = 2048 // MB
	DefaultDockerCPULimit      = 2.0
	DefaultDockerNetworkMode   = "bridge"
	DefaultContainerPrefix     = "aicli"

	// API 기본값
	DefaultAPIAddress     = "localhost:8080"
	DefaultRateLimit      = 100 // requests per minute
	DefaultJWTExpiration  = 24 * time.Hour
	
	// JWT 기본값
	DefaultAccessTokenExpiry  = 15 * time.Minute
	DefaultRefreshTokenExpiry = 7 * 24 * time.Hour
	DefaultJWTSecretKey      = "default-secret-key-change-in-production"
)

// GetDefaultConfig는 기본 설정을 반환합니다
func GetDefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultWorkspacePath := filepath.Join(homeDir, ".aicli", "workspaces")
	defaultLogPath := filepath.Join(homeDir, ".aicli", "logs", "aicli.log")

	return &Config{
		Claude: ClaudeConfig{
			Model:       DefaultClaudeModel,
			Temperature: DefaultClaudeTemperature,
			Timeout:     DefaultClaudeTimeout,
			MaxTokens:   DefaultClaudeMaxTokens,
			RetryCount:  DefaultClaudeRetryCount,
			RetryDelay:  DefaultClaudeRetryDelay,
		},
		Workspace: WorkspaceConfig{
			DefaultPath:     defaultWorkspacePath,
			AutoSync:        true,
			MaxProjects:     DefaultMaxProjects,
			IsolationMode:   DefaultIsolationMode,
			WatchFiles:      true,
			ExcludePatterns: []string{
				"*.tmp",
				"*.log",
				".git/**",
				"node_modules/**",
				"__pycache__/**",
				"*.pyc",
				".DS_Store",
				"vendor/**",
			},
		},
		Output: OutputConfig{
			Format:        DefaultOutputFormat,
			ColorMode:     DefaultColorMode,
			Width:         DefaultOutputWidth,
			Verbosity:     DefaultVerbosity,
			ShowProgress:  true,
			ShowTimestamp: false,
		},
		Logging: LoggingConfig{
			Level:      DefaultLogLevel,
			FilePath:   defaultLogPath,
			MaxSize:    DefaultLogMaxSize,
			MaxBackups: DefaultLogMaxBackups,
			MaxAge:     DefaultLogMaxAge,
			Compress:   true,
			JSONFormat: false,
		},
		Docker: DockerConfig{
			SocketPath:      DefaultDockerSocketPath,
			DefaultImage:    DefaultDockerImage,
			MemoryLimit:     DefaultDockerMemoryLimit,
			CPULimit:        DefaultDockerCPULimit,
			NetworkMode:     DefaultDockerNetworkMode,
			AutoCleanup:     true,
			ContainerPrefix: DefaultContainerPrefix,
		},
		API: APIConfig{
			Address:            DefaultAPIAddress,
			TLSEnabled:         false,
			CORSOrigins:        []string{"http://localhost:3000"},
			RateLimit:          DefaultRateLimit,
			JWTExpiration:      DefaultJWTExpiration,
			JWTSecret:          DefaultJWTSecretKey,
			AccessTokenExpiry:  DefaultAccessTokenExpiry,
			RefreshTokenExpiry: DefaultRefreshTokenExpiry,
		},
		
		Storage: StorageConfig{
			Type:            "memory",
			DataSource:      "",
			MaxConns:        10,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
			Timeout:         time.Second * 30,
			RetryCount:      3,
			RetryInterval:   time.Second,
		},
	}
}

// GetConfigDir는 설정 디렉토리 경로를 반환합니다
func GetConfigDir() string {
	if configDir := os.Getenv("AICLI_CONFIG_DIR"); configDir != "" {
		return configDir
	}
	
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".aicli")
}

// GetConfigPath는 기본 설정 파일 경로를 반환합니다
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.yaml")
}

// GetLogDir는 로그 디렉토리 경로를 반환합니다
func GetLogDir() string {
	return filepath.Join(GetConfigDir(), "logs")
}

// GetWorkspaceDir는 기본 워크스페이스 디렉토리 경로를 반환합니다
func GetWorkspaceDir() string {
	return filepath.Join(GetConfigDir(), "workspaces")
}