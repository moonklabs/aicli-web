package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// 환경 변수 이름 정의
const (
	// Claude 환경 변수
	EnvClaudeAPIKey      = "AICLI_CLAUDE_API_KEY"
	EnvClaudeModel       = "AICLI_CLAUDE_MODEL"
	EnvClaudeTemperature = "AICLI_CLAUDE_TEMPERATURE"
	EnvClaudeTimeout     = "AICLI_CLAUDE_TIMEOUT"
	EnvClaudeMaxTokens   = "AICLI_CLAUDE_MAX_TOKENS"
	EnvClaudeRetryCount  = "AICLI_CLAUDE_RETRY_COUNT"
	EnvClaudeRetryDelay  = "AICLI_CLAUDE_RETRY_DELAY"

	// 워크스페이스 환경 변수
	EnvWorkspaceDefaultPath     = "AICLI_WORKSPACE_DEFAULT_PATH"
	EnvWorkspaceAutoSync        = "AICLI_WORKSPACE_AUTO_SYNC"
	EnvWorkspaceMaxProjects     = "AICLI_WORKSPACE_MAX_PROJECTS"
	EnvWorkspaceIsolationMode   = "AICLI_WORKSPACE_ISOLATION_MODE"
	EnvWorkspaceWatchFiles      = "AICLI_WORKSPACE_WATCH_FILES"
	EnvWorkspaceExcludePatterns = "AICLI_WORKSPACE_EXCLUDE_PATTERNS"

	// 출력 환경 변수
	EnvOutputFormat       = "AICLI_OUTPUT_FORMAT"
	EnvOutputColorMode    = "AICLI_OUTPUT_COLOR_MODE"
	EnvOutputWidth        = "AICLI_OUTPUT_WIDTH"
	EnvOutputVerbosity    = "AICLI_OUTPUT_VERBOSITY"
	EnvOutputShowProgress = "AICLI_OUTPUT_SHOW_PROGRESS"
	EnvOutputShowTimestamp = "AICLI_OUTPUT_SHOW_TIMESTAMP"

	// 로깅 환경 변수
	EnvLogLevel      = "AICLI_LOG_LEVEL"
	EnvLogFilePath   = "AICLI_LOG_FILE_PATH"
	EnvLogMaxSize    = "AICLI_LOG_MAX_SIZE"
	EnvLogMaxBackups = "AICLI_LOG_MAX_BACKUPS"
	EnvLogMaxAge     = "AICLI_LOG_MAX_AGE"
	EnvLogCompress   = "AICLI_LOG_COMPRESS"
	EnvLogJSONFormat = "AICLI_LOG_JSON_FORMAT"

	// Docker 환경 변수
	EnvDockerSocketPath     = "AICLI_DOCKER_SOCKET_PATH"
	EnvDockerDefaultImage   = "AICLI_DOCKER_DEFAULT_IMAGE"
	EnvDockerMemoryLimit    = "AICLI_DOCKER_MEMORY_LIMIT"
	EnvDockerCPULimit       = "AICLI_DOCKER_CPU_LIMIT"
	EnvDockerNetworkMode    = "AICLI_DOCKER_NETWORK_MODE"
	EnvDockerAutoCleanup    = "AICLI_DOCKER_AUTO_CLEANUP"
	EnvDockerContainerPrefix = "AICLI_DOCKER_CONTAINER_PREFIX"

	// API 환경 변수
	EnvAPIAddress      = "AICLI_API_ADDRESS"
	EnvAPITLSEnabled   = "AICLI_API_TLS_ENABLED"
	EnvAPITLSCertPath  = "AICLI_API_TLS_CERT_PATH"
	EnvAPITLSKeyPath   = "AICLI_API_TLS_KEY_PATH"
	EnvAPICORSOrigins  = "AICLI_API_CORS_ORIGINS"
	EnvAPIRateLimit    = "AICLI_API_RATE_LIMIT"
	EnvAPIJWTSecret    = "AICLI_API_JWT_SECRET"
	EnvAPIJWTExpiration = "AICLI_API_JWT_EXPIRATION"
)

// LoadFromEnv는 환경 변수에서 설정을 읽어 기존 설정에 적용합니다
func LoadFromEnv(cfg *Config) error {
	// Claude 설정
	if apiKey := os.Getenv(EnvClaudeAPIKey); apiKey != "" {
		cfg.Claude.APIKey = apiKey
	}
	if model := os.Getenv(EnvClaudeModel); model != "" {
		cfg.Claude.Model = model
	}
	if temp := os.Getenv(EnvClaudeTemperature); temp != "" {
		if f, err := strconv.ParseFloat(temp, 64); err == nil {
			cfg.Claude.Temperature = f
		}
	}
	if timeout := os.Getenv(EnvClaudeTimeout); timeout != "" {
		if i, err := strconv.Atoi(timeout); err == nil {
			cfg.Claude.Timeout = i
		}
	}
	if maxTokens := os.Getenv(EnvClaudeMaxTokens); maxTokens != "" {
		if i, err := strconv.Atoi(maxTokens); err == nil {
			cfg.Claude.MaxTokens = i
		}
	}
	if retryCount := os.Getenv(EnvClaudeRetryCount); retryCount != "" {
		if i, err := strconv.Atoi(retryCount); err == nil {
			cfg.Claude.RetryCount = i
		}
	}
	if retryDelay := os.Getenv(EnvClaudeRetryDelay); retryDelay != "" {
		if d, err := time.ParseDuration(retryDelay); err == nil {
			cfg.Claude.RetryDelay = d
		}
	}

	// 워크스페이스 설정
	if path := os.Getenv(EnvWorkspaceDefaultPath); path != "" {
		cfg.Workspace.DefaultPath = path
	}
	if autoSync := os.Getenv(EnvWorkspaceAutoSync); autoSync != "" {
		cfg.Workspace.AutoSync = parseBool(autoSync)
	}
	if maxProjects := os.Getenv(EnvWorkspaceMaxProjects); maxProjects != "" {
		if i, err := strconv.Atoi(maxProjects); err == nil {
			cfg.Workspace.MaxProjects = i
		}
	}
	if isolationMode := os.Getenv(EnvWorkspaceIsolationMode); isolationMode != "" {
		cfg.Workspace.IsolationMode = isolationMode
	}
	if watchFiles := os.Getenv(EnvWorkspaceWatchFiles); watchFiles != "" {
		cfg.Workspace.WatchFiles = parseBool(watchFiles)
	}
	if excludePatterns := os.Getenv(EnvWorkspaceExcludePatterns); excludePatterns != "" {
		cfg.Workspace.ExcludePatterns = strings.Split(excludePatterns, ",")
	}

	// 출력 설정
	if format := os.Getenv(EnvOutputFormat); format != "" {
		cfg.Output.Format = format
	}
	if colorMode := os.Getenv(EnvOutputColorMode); colorMode != "" {
		cfg.Output.ColorMode = colorMode
	}
	if width := os.Getenv(EnvOutputWidth); width != "" {
		if i, err := strconv.Atoi(width); err == nil {
			cfg.Output.Width = i
		}
	}
	if verbosity := os.Getenv(EnvOutputVerbosity); verbosity != "" {
		if i, err := strconv.Atoi(verbosity); err == nil {
			cfg.Output.Verbosity = i
		}
	}
	if showProgress := os.Getenv(EnvOutputShowProgress); showProgress != "" {
		cfg.Output.ShowProgress = parseBool(showProgress)
	}
	if showTimestamp := os.Getenv(EnvOutputShowTimestamp); showTimestamp != "" {
		cfg.Output.ShowTimestamp = parseBool(showTimestamp)
	}

	// 로깅 설정
	if level := os.Getenv(EnvLogLevel); level != "" {
		cfg.Logging.Level = level
	}
	if filePath := os.Getenv(EnvLogFilePath); filePath != "" {
		cfg.Logging.FilePath = filePath
	}
	if maxSize := os.Getenv(EnvLogMaxSize); maxSize != "" {
		if i, err := strconv.Atoi(maxSize); err == nil {
			cfg.Logging.MaxSize = i
		}
	}
	if maxBackups := os.Getenv(EnvLogMaxBackups); maxBackups != "" {
		if i, err := strconv.Atoi(maxBackups); err == nil {
			cfg.Logging.MaxBackups = i
		}
	}
	if maxAge := os.Getenv(EnvLogMaxAge); maxAge != "" {
		if i, err := strconv.Atoi(maxAge); err == nil {
			cfg.Logging.MaxAge = i
		}
	}
	if compress := os.Getenv(EnvLogCompress); compress != "" {
		cfg.Logging.Compress = parseBool(compress)
	}
	if jsonFormat := os.Getenv(EnvLogJSONFormat); jsonFormat != "" {
		cfg.Logging.JSONFormat = parseBool(jsonFormat)
	}

	// Docker 설정
	if socketPath := os.Getenv(EnvDockerSocketPath); socketPath != "" {
		cfg.Docker.SocketPath = socketPath
	}
	if defaultImage := os.Getenv(EnvDockerDefaultImage); defaultImage != "" {
		cfg.Docker.DefaultImage = defaultImage
	}
	if memoryLimit := os.Getenv(EnvDockerMemoryLimit); memoryLimit != "" {
		if i, err := strconv.ParseInt(memoryLimit, 10, 64); err == nil {
			cfg.Docker.MemoryLimit = i
		}
	}
	if cpuLimit := os.Getenv(EnvDockerCPULimit); cpuLimit != "" {
		if f, err := strconv.ParseFloat(cpuLimit, 64); err == nil {
			cfg.Docker.CPULimit = f
		}
	}
	if networkMode := os.Getenv(EnvDockerNetworkMode); networkMode != "" {
		cfg.Docker.NetworkMode = networkMode
	}
	if autoCleanup := os.Getenv(EnvDockerAutoCleanup); autoCleanup != "" {
		cfg.Docker.AutoCleanup = parseBool(autoCleanup)
	}
	if containerPrefix := os.Getenv(EnvDockerContainerPrefix); containerPrefix != "" {
		cfg.Docker.ContainerPrefix = containerPrefix
	}

	// API 설정
	if address := os.Getenv(EnvAPIAddress); address != "" {
		cfg.API.Address = address
	}
	if tlsEnabled := os.Getenv(EnvAPITLSEnabled); tlsEnabled != "" {
		cfg.API.TLSEnabled = parseBool(tlsEnabled)
	}
	if tlsCertPath := os.Getenv(EnvAPITLSCertPath); tlsCertPath != "" {
		cfg.API.TLSCertPath = tlsCertPath
	}
	if tlsKeyPath := os.Getenv(EnvAPITLSKeyPath); tlsKeyPath != "" {
		cfg.API.TLSKeyPath = tlsKeyPath
	}
	if corsOrigins := os.Getenv(EnvAPICORSOrigins); corsOrigins != "" {
		cfg.API.CORSOrigins = strings.Split(corsOrigins, ",")
	}
	if rateLimit := os.Getenv(EnvAPIRateLimit); rateLimit != "" {
		if i, err := strconv.Atoi(rateLimit); err == nil {
			cfg.API.RateLimit = i
		}
	}
	if jwtSecret := os.Getenv(EnvAPIJWTSecret); jwtSecret != "" {
		cfg.API.JWTSecret = jwtSecret
	}
	if jwtExpiration := os.Getenv(EnvAPIJWTExpiration); jwtExpiration != "" {
		if d, err := time.ParseDuration(jwtExpiration); err == nil {
			cfg.API.JWTExpiration = d
		}
	}

	return nil
}

// parseBool은 문자열을 bool로 변환합니다
func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// GetEnvWithDefault는 환경 변수를 읽거나 기본값을 반환합니다
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}