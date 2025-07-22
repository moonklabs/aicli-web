package config

import "time"

// Config는 AICode Manager의 전체 설정을 나타냅니다
type Config struct {
	// Claude 관련 설정
	Claude ClaudeConfig `yaml:"claude" mapstructure:"claude" json:"claude"`
	
	// 워크스페이스 관련 설정
	Workspace WorkspaceConfig `yaml:"workspace" mapstructure:"workspace" json:"workspace"`
	
	// 출력 관련 설정
	Output OutputConfig `yaml:"output" mapstructure:"output" json:"output"`
	
	// 로깅 관련 설정
	Logging LoggingConfig `yaml:"logging" mapstructure:"logging" json:"logging"`
	
	// Docker 관련 설정
	Docker DockerConfig `yaml:"docker" mapstructure:"docker" json:"docker"`
	
	// API 서버 관련 설정
	API APIConfig `yaml:"api" mapstructure:"api" json:"api"`
	
	// 스토리지 관련 설정
	Storage StorageConfig `yaml:"storage" mapstructure:"storage" json:"storage"`
}

// ClaudeConfig는 Claude CLI 관련 설정을 정의합니다
type ClaudeConfig struct {
	// API 키
	APIKey string `yaml:"api_key" mapstructure:"api_key" json:"api_key" validate:"required,min=20"`
	
	// 사용할 모델
	Model string `yaml:"model" mapstructure:"model" json:"model" validate:"required,oneof=claude-3-opus-20240229 claude-3-sonnet-20240229 claude-3-haiku-20240307"`
	
	// Temperature 설정 (0.0-1.0)
	Temperature float64 `yaml:"temperature" mapstructure:"temperature" json:"temperature" validate:"min=0,max=1"`
	
	// 타임아웃 (초)
	Timeout int `yaml:"timeout" mapstructure:"timeout" json:"timeout" validate:"min=1,max=3600"`
	
	// 최대 토큰 수
	MaxTokens int `yaml:"max_tokens" mapstructure:"max_tokens" json:"max_tokens" validate:"min=1,max=200000"`
	
	// 재시도 횟수
	RetryCount int `yaml:"retry_count" mapstructure:"retry_count" json:"retry_count" validate:"min=0,max=10"`
	
	// 재시도 간격 (초)
	RetryDelay time.Duration `yaml:"retry_delay" mapstructure:"retry_delay" json:"retry_delay"`
}

// WorkspaceConfig는 워크스페이스 관련 설정을 정의합니다
type WorkspaceConfig struct {
	// 기본 워크스페이스 경로
	DefaultPath string `yaml:"default_path" mapstructure:"default_path" json:"default_path" validate:"required,dir"`
	
	// 자동 동기화 활성화
	AutoSync bool `yaml:"auto_sync" mapstructure:"auto_sync" json:"auto_sync"`
	
	// 동시 실행 가능한 최대 프로젝트 수
	MaxProjects int `yaml:"max_projects" mapstructure:"max_projects" json:"max_projects" validate:"min=1,max=100"`
	
	// 워크스페이스 격리 모드
	IsolationMode string `yaml:"isolation_mode" mapstructure:"isolation_mode" json:"isolation_mode" validate:"oneof=docker process none"`
	
	// 파일 감시 활성화
	WatchFiles bool `yaml:"watch_files" mapstructure:"watch_files" json:"watch_files"`
	
	// 제외 패턴 (glob)
	ExcludePatterns []string `yaml:"exclude_patterns" mapstructure:"exclude_patterns" json:"exclude_patterns"`
}

// OutputConfig는 출력 형식 관련 설정을 정의합니다
type OutputConfig struct {
	// 출력 형식
	Format string `yaml:"format" mapstructure:"format" json:"format" validate:"oneof=table json yaml pretty plain"`
	
	// 색상 모드
	ColorMode string `yaml:"color_mode" mapstructure:"color_mode" json:"color_mode" validate:"oneof=auto always never"`
	
	// 출력 너비
	Width int `yaml:"width" mapstructure:"width" json:"width" validate:"min=40,max=300"`
	
	// 상세 출력 레벨
	Verbosity int `yaml:"verbosity" mapstructure:"verbosity" json:"verbosity" validate:"min=0,max=3"`
	
	// 진행 표시기 활성화
	ShowProgress bool `yaml:"show_progress" mapstructure:"show_progress" json:"show_progress"`
	
	// 타임스탬프 표시
	ShowTimestamp bool `yaml:"show_timestamp" mapstructure:"show_timestamp" json:"show_timestamp"`
}

// LoggingConfig는 로깅 관련 설정을 정의합니다
type LoggingConfig struct {
	// 로그 레벨
	Level string `yaml:"level" mapstructure:"level" json:"level" validate:"oneof=debug info warn error fatal"`
	
	// 로그 파일 경로
	FilePath string `yaml:"file_path" mapstructure:"file_path" json:"file_path"`
	
	// 로그 파일 최대 크기 (MB)
	MaxSize int `yaml:"max_size" mapstructure:"max_size" json:"max_size" validate:"min=1,max=1000"`
	
	// 로그 파일 최대 보관 개수
	MaxBackups int `yaml:"max_backups" mapstructure:"max_backups" json:"max_backups" validate:"min=0,max=100"`
	
	// 로그 파일 최대 보관 일수
	MaxAge int `yaml:"max_age" mapstructure:"max_age" json:"max_age" validate:"min=0,max=365"`
	
	// 로그 압축 활성화
	Compress bool `yaml:"compress" mapstructure:"compress" json:"compress"`
	
	// JSON 형식 로깅
	JSONFormat bool `yaml:"json_format" mapstructure:"json_format" json:"json_format"`
}

// DockerConfig는 Docker 관련 설정을 정의합니다
type DockerConfig struct {
	// Docker 소켓 경로
	SocketPath string `yaml:"socket_path" mapstructure:"socket_path" json:"socket_path"`
	
	// 기본 이미지
	DefaultImage string `yaml:"default_image" mapstructure:"default_image" json:"default_image" validate:"required"`
	
	// 컨테이너 메모리 제한 (MB)
	MemoryLimit int64 `yaml:"memory_limit" mapstructure:"memory_limit" json:"memory_limit" validate:"min=128"`
	
	// 컨테이너 CPU 제한
	CPULimit float64 `yaml:"cpu_limit" mapstructure:"cpu_limit" json:"cpu_limit" validate:"min=0.1"`
	
	// 네트워크 모드
	NetworkMode string `yaml:"network_mode" mapstructure:"network_mode" json:"network_mode" validate:"oneof=bridge host none"`
	
	// 자동 정리 활성화
	AutoCleanup bool `yaml:"auto_cleanup" mapstructure:"auto_cleanup" json:"auto_cleanup"`
	
	// 컨테이너 접두사
	ContainerPrefix string `yaml:"container_prefix" mapstructure:"container_prefix" json:"container_prefix" validate:"required"`
}

// APIConfig는 API 서버 관련 설정을 정의합니다
type APIConfig struct {
	// 리스닝 주소
	Address string `yaml:"address" mapstructure:"address" json:"address" validate:"required,hostname_port"`
	
	// TLS 활성화
	TLSEnabled bool `yaml:"tls_enabled" mapstructure:"tls_enabled" json:"tls_enabled"`
	
	// TLS 인증서 경로
	TLSCertPath string `yaml:"tls_cert_path" mapstructure:"tls_cert_path" json:"tls_cert_path"`
	
	// TLS 키 경로
	TLSKeyPath string `yaml:"tls_key_path" mapstructure:"tls_key_path" json:"tls_key_path"`
	
	// CORS 허용 오리진
	CORSOrigins []string `yaml:"cors_origins" mapstructure:"cors_origins" json:"cors_origins"`
	
	// 요청 제한 (분당)
	RateLimit int `yaml:"rate_limit" mapstructure:"rate_limit" json:"rate_limit" validate:"min=0,max=10000"`
	
	// JWT 비밀 키
	JWTSecret string `yaml:"jwt_secret" mapstructure:"jwt_secret" json:"jwt_secret" validate:"required,min=32"`
	
	// JWT 만료 시간 (시간)
	JWTExpiration time.Duration `yaml:"jwt_expiration" mapstructure:"jwt_expiration" json:"jwt_expiration"`
	
	// 액세스 토큰 만료 시간
	AccessTokenExpiry time.Duration `yaml:"access_token_expiry" mapstructure:"access_token_expiry" json:"access_token_expiry"`
	
	// 리프레시 토큰 만료 시간
	RefreshTokenExpiry time.Duration `yaml:"refresh_token_expiry" mapstructure:"refresh_token_expiry" json:"refresh_token_expiry"`
	
	// OAuth 설정
	OAuth OAuthConfig `yaml:"oauth" mapstructure:"oauth" json:"oauth"`
}

// StorageConfig는 스토리지 관련 설정을 정의합니다
type StorageConfig struct {
	// Type 스토리지 타입 (memory, sqlite, boltdb)
	Type string `yaml:"type" mapstructure:"type" json:"type" validate:"oneof=memory sqlite boltdb"`
	
	// DataSource 데이터 소스 (파일 경로 또는 연결 문자열)
	DataSource string `yaml:"data_source" mapstructure:"data_source" json:"data_source"`
	
	// MaxConns 최대 연결 수
	MaxConns int `yaml:"max_conns" mapstructure:"max_conns" json:"max_conns" validate:"min=1,max=100"`
	
	// MaxIdleConns 최대 유휴 연결 수
	MaxIdleConns int `yaml:"max_idle_conns" mapstructure:"max_idle_conns" json:"max_idle_conns" validate:"min=0"`
	
	// ConnMaxLifetime 연결 최대 생명 시간
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime" json:"conn_max_lifetime"`
	
	// Timeout 연결 타임아웃
	Timeout time.Duration `yaml:"timeout" mapstructure:"timeout" json:"timeout"`
	
	// RetryCount 재시도 횟수
	RetryCount int `yaml:"retry_count" mapstructure:"retry_count" json:"retry_count" validate:"min=0,max=10"`
	
	// RetryInterval 재시도 간격
	RetryInterval time.Duration `yaml:"retry_interval" mapstructure:"retry_interval" json:"retry_interval"`
}

// OAuthConfig는 OAuth 인증 관련 설정을 정의합니다
type OAuthConfig struct {
	// 전체 OAuth 활성화 여부
	Enabled bool `yaml:"enabled" mapstructure:"enabled" json:"enabled"`
	
	// 기본 리다이렉트 URL
	BaseRedirectURL string `yaml:"base_redirect_url" mapstructure:"base_redirect_url" json:"base_redirect_url"`
	
	// Google OAuth 설정
	Google GoogleOAuthConfig `yaml:"google" mapstructure:"google" json:"google"`
	
	// GitHub OAuth 설정
	GitHub GitHubOAuthConfig `yaml:"github" mapstructure:"github" json:"github"`
	
	// Microsoft OAuth 설정 (선택적)
	Microsoft MicrosoftOAuthConfig `yaml:"microsoft" mapstructure:"microsoft" json:"microsoft"`
	
	// state 파라미터 유효 시간 (분)
	StateExpiry int `yaml:"state_expiry" mapstructure:"state_expiry" json:"state_expiry" validate:"min=1,max=60"`
}

// GoogleOAuthConfig Google OAuth 설정
type GoogleOAuthConfig struct {
	Enabled      bool     `yaml:"enabled" mapstructure:"enabled" json:"enabled"`
	ClientID     string   `yaml:"client_id" mapstructure:"client_id" json:"client_id"`
	ClientSecret string   `yaml:"client_secret" mapstructure:"client_secret" json:"client_secret"`
	Scopes       []string `yaml:"scopes" mapstructure:"scopes" json:"scopes"`
}

// GitHubOAuthConfig GitHub OAuth 설정
type GitHubOAuthConfig struct {
	Enabled      bool     `yaml:"enabled" mapstructure:"enabled" json:"enabled"`
	ClientID     string   `yaml:"client_id" mapstructure:"client_id" json:"client_id"`
	ClientSecret string   `yaml:"client_secret" mapstructure:"client_secret" json:"client_secret"`
	Scopes       []string `yaml:"scopes" mapstructure:"scopes" json:"scopes"`
}

// MicrosoftOAuthConfig Microsoft OAuth 설정
type MicrosoftOAuthConfig struct {
	Enabled      bool     `yaml:"enabled" mapstructure:"enabled" json:"enabled"`
	ClientID     string   `yaml:"client_id" mapstructure:"client_id" json:"client_id"`
	ClientSecret string   `yaml:"client_secret" mapstructure:"client_secret" json:"client_secret"`
	TenantID     string   `yaml:"tenant_id" mapstructure:"tenant_id" json:"tenant_id"`
	Scopes       []string `yaml:"scopes" mapstructure:"scopes" json:"scopes"`
}