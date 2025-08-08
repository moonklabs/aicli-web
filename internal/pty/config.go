package pty

import (
	"os"
	"strconv"
	"time"
	
	log "github.com/sirupsen/logrus"
)

// ConfigLoader 설정 로더
type ConfigLoader struct {
	envPrefix string
}

// NewConfigLoader 새 설정 로더 생성
func NewConfigLoader(envPrefix string) *ConfigLoader {
	if envPrefix == "" {
		envPrefix = "PTY"
	}
	return &ConfigLoader{
		envPrefix: envPrefix,
	}
}

// LoadSessionConfig 세션 설정 로드
func (cl *ConfigLoader) LoadSessionConfig() *SessionConfig {
	config := DefaultSessionConfig()

	// 환경변수에서 설정 로드
	if val := cl.getEnvInt("MAX_SESSIONS"); val > 0 {
		config.MaxSessions = val
	}

	if val := cl.getEnvDuration("IDLE_TIMEOUT"); val > 0 {
		config.IdleTimeout = val
	}

	if val := cl.getEnvDuration("CLEANUP_INTERVAL"); val > 0 {
		config.CleanupInterval = val
	}

	if val := cl.getEnvDuration("MAX_SESSION_AGE"); val > 0 {
		config.MaxSessionAge = val
	}

	if val := cl.getEnvBool("ENABLE_POOLING"); val != nil {
		config.EnablePooling = *val
	}

	if val := cl.getEnvInt("POOL_SIZE"); val > 0 {
		config.PoolSize = val
	}

	return config
}

// LoadPTYConfig PTY 설정 로드
func (cl *ConfigLoader) LoadPTYConfig() *PTYConfig {
	config := DefaultPTYConfig()

	// 환경변수에서 설정 로드
	if val := cl.getEnvInt("TERM_ROWS"); val > 0 {
		config.Rows = val
	}

	if val := cl.getEnvInt("TERM_COLS"); val > 0 {
		config.Cols = val
	}

	if val := cl.getEnvString("TERM_TYPE"); val != "" {
		config.Term = val
	}

	if val := cl.getEnvString("SHELL"); val != "" {
		config.Shell = val
	}

	if val := cl.getEnvString("WORKING_DIR"); val != "" {
		config.WorkingDir = val
	}

	// 환경변수 로드
	config.Environment = cl.loadEnvironmentVars()

	return config
}

// getEnvString 환경변수 문자열 조회
func (cl *ConfigLoader) getEnvString(key string) string {
	return os.Getenv(cl.envPrefix + "_" + key)
}

// getEnvInt 환경변수 정수 조회
func (cl *ConfigLoader) getEnvInt(key string) int {
	val := cl.getEnvString(key)
	if val == "" {
		return 0
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		log.Warnf("Invalid integer value for %s_%s: %s", cl.envPrefix, key, val)
		return 0
	}

	return i
}

// getEnvBool 환경변수 불린 조회
func (cl *ConfigLoader) getEnvBool(key string) *bool {
	val := cl.getEnvString(key)
	if val == "" {
		return nil
	}

	b, err := strconv.ParseBool(val)
	if err != nil {
		log.Warnf("Invalid boolean value for %s_%s: %s", cl.envPrefix, key, val)
		return nil
	}

	return &b
}

// getEnvDuration 환경변수 Duration 조회
func (cl *ConfigLoader) getEnvDuration(key string) time.Duration {
	val := cl.getEnvString(key)
	if val == "" {
		return 0
	}

	d, err := time.ParseDuration(val)
	if err != nil {
		log.Warnf("Invalid duration value for %s_%s: %s", cl.envPrefix, key, val)
		return 0
	}

	return d
}

// loadEnvironmentVars 환경변수 맵 로드
func (cl *ConfigLoader) loadEnvironmentVars() map[string]string {
	env := make(map[string]string)

	// 기본 환경변수
	env["TERM"] = cl.getEnvString("TERM_TYPE")
	if env["TERM"] == "" {
		env["TERM"] = "xterm-256color"
	}

	// PATH 설정
	if path := os.Getenv("PATH"); path != "" {
		env["PATH"] = path
	} else {
		env["PATH"] = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	}

	// 홈 디렉토리
	if home := os.Getenv("HOME"); home != "" {
		env["HOME"] = home
	}

	// 언어 설정
	if lang := os.Getenv("LANG"); lang != "" {
		env["LANG"] = lang
	} else {
		env["LANG"] = "en_US.UTF-8"
	}

	// 추가 환경변수
	prefix := cl.envPrefix + "_ENV_"
	for _, e := range os.Environ() {
		if len(e) > len(prefix) && e[:len(prefix)] == prefix {
			key := e[len(prefix):]
			if idx := strToIndex(key, '='); idx > 0 {
				envKey := key[:idx]
				envVal := key[idx+1:]
				env[envKey] = envVal
			}
		}
	}

	return env
}

// strToIndex 문자열에서 문자 인덱스 찾기
func strToIndex(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// ValidateSessionConfig 세션 설정 검증
func ValidateSessionConfig(config *SessionConfig) error {
	if config.MaxSessions <= 0 {
		config.MaxSessions = 100
	}

	if config.IdleTimeout <= 0 {
		config.IdleTimeout = 15 * time.Minute
	}

	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 1 * time.Minute
	}

	if config.PoolSize < 0 {
		config.PoolSize = 0
	}

	if config.PoolSize > config.MaxSessions {
		config.PoolSize = config.MaxSessions / 10
	}

	return nil
}

// ValidatePTYConfig PTY 설정 검증
func ValidatePTYConfig(config *PTYConfig) error {
	if config.Rows <= 0 {
		config.Rows = 24
	}

	if config.Cols <= 0 {
		config.Cols = 80
	}

	if config.Term == "" {
		config.Term = "xterm-256color"
	}

	if config.Shell == "" {
		// 기본 셸 찾기
		shells := []string{
			"/bin/bash",
			"/bin/sh",
			"/usr/bin/bash",
			"/usr/bin/sh",
		}

		for _, shell := range shells {
			if _, err := os.Stat(shell); err == nil {
				config.Shell = shell
				break
			}
		}

		if config.Shell == "" {
			config.Shell = "/bin/sh"
		}
	}

	if config.WorkingDir == "" {
		config.WorkingDir = "/"
	}

	if config.Environment == nil {
		config.Environment = make(map[string]string)
	}

	return nil
}

// MergeConfigs 설정 병합
func MergeConfigs(base, override *PTYConfig) *PTYConfig {
	if override == nil {
		return base
	}

	if base == nil {
		return override
	}

	merged := &PTYConfig{
		Rows:        override.Rows,
		Cols:        override.Cols,
		Term:        override.Term,
		Shell:       override.Shell,
		WorkingDir:  override.WorkingDir,
		Environment: make(map[string]string),
	}

	// 기본값 사용
	if merged.Rows == 0 {
		merged.Rows = base.Rows
	}
	if merged.Cols == 0 {
		merged.Cols = base.Cols
	}
	if merged.Term == "" {
		merged.Term = base.Term
	}
	if merged.Shell == "" {
		merged.Shell = base.Shell
	}
	if merged.WorkingDir == "" {
		merged.WorkingDir = base.WorkingDir
	}

	// 환경변수 병합
	for k, v := range base.Environment {
		merged.Environment[k] = v
	}
	for k, v := range override.Environment {
		merged.Environment[k] = v
	}

	return merged
}