// AICLI Web 상수 정의

// 애플리케이션 정보
export const APP_NAME = 'AICLI Web'
export const APP_VERSION = '1.0.0'
export const APP_DESCRIPTION = 'AI Code Manager - Web platform for managing Claude CLI'

// API 관련 상수
export const API_ENDPOINTS = {
  AUTH: '/auth',
  WORKSPACES: '/workspaces',
  DOCKER: '/docker',
  CLAUDE: '/claude',
  SESSIONS: '/sessions',
  TASKS: '/tasks',
} as const

// WebSocket 이벤트 타입
export const WS_EVENTS = {
  // 연결 관련
  CONNECT: 'connect',
  DISCONNECT: 'disconnect',
  ERROR: 'error',

  // 터미널 관련
  TERMINAL_OUTPUT: 'terminal:output',
  TERMINAL_INPUT: 'terminal:input',
  TERMINAL_ERROR: 'terminal:error',
  TERMINAL_CLOSE: 'terminal:close',

  // 워크스페이스 관련
  WORKSPACE_STATUS: 'workspace:status',
  WORKSPACE_CREATED: 'workspace:created',
  WORKSPACE_UPDATED: 'workspace:updated',
  WORKSPACE_DELETED: 'workspace:deleted',

  // Docker 관련
  DOCKER_CONTAINER_STATUS: 'docker:container:status',
  DOCKER_STATS: 'docker:stats',

  // 태스크 관련
  TASK_UPDATE: 'task:update',
  TASK_COMPLETE: 'task:complete',
  TASK_ERROR: 'task:error',
} as const

// 컨테이너 상태
export const CONTAINER_STATUS = {
  CREATED: 'created',
  RUNNING: 'running',
  PAUSED: 'paused',
  RESTARTING: 'restarting',
  REMOVING: 'removing',
  DEAD: 'dead',
  EXITED: 'exited',
} as const

// 워크스페이스 상태
export const WORKSPACE_STATUS = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  ERROR: 'error',
  CREATING: 'creating',
  DELETING: 'deleting',
} as const

// 터미널 세션 상태
export const TERMINAL_STATUS = {
  CONNECTED: 'connected',
  DISCONNECTED: 'disconnected',
  ERROR: 'error',
  CONNECTING: 'connecting',
} as const

// 태스크 상태
export const TASK_STATUS = {
  PENDING: 'pending',
  RUNNING: 'running',
  COMPLETED: 'completed',
  FAILED: 'failed',
  CANCELLED: 'cancelled',
} as const

// 로그 레벨
export const LOG_LEVELS = {
  DEBUG: 'debug',
  INFO: 'info',
  WARN: 'warn',
  ERROR: 'error',
} as const

// 로그 타입
export const LOG_TYPES = {
  INPUT: 'input',
  OUTPUT: 'output',
  ERROR: 'error',
  SYSTEM: 'system',
} as const

// 페이지네이션 기본값
export const PAGINATION_DEFAULTS = {
  PAGE: 1,
  LIMIT: 20,
  MAX_LIMIT: 100,
} as const

// 로컬 스토리지 키
export const STORAGE_KEYS = {
  AUTH_TOKEN: 'auth_token',
  REFRESH_TOKEN: 'refresh_token',
  USER_PREFERENCES: 'user_preferences',
  THEME: 'theme',
  LANGUAGE: 'language',
  SIDEBAR_COLLAPSED: 'sidebar_collapsed',
  TERMINAL_SETTINGS: 'terminal_settings',
} as const

// 테마
export const THEMES = {
  LIGHT: 'light',
  DARK: 'dark',
  SYSTEM: 'system',
} as const

// 언어
export const LANGUAGES = {
  KOREAN: 'ko',
  ENGLISH: 'en',
} as const

// 파일 타입
export const FILE_TYPES = {
  TEXT: 'text',
  CODE: 'code',
  IMAGE: 'image',
  DOCUMENT: 'document',
  ARCHIVE: 'archive',
  OTHER: 'other',
} as const

// 코드 언어 확장자 매핑
export const CODE_EXTENSIONS = {
  'js': 'javascript',
  'jsx': 'javascript',
  'ts': 'typescript',
  'tsx': 'typescript',
  'vue': 'vue',
  'html': 'html',
  'css': 'css',
  'scss': 'scss',
  'sass': 'scss',
  'less': 'less',
  'json': 'json',
  'xml': 'xml',
  'yaml': 'yaml',
  'yml': 'yaml',
  'md': 'markdown',
  'py': 'python',
  'go': 'go',
  'java': 'java',
  'c': 'c',
  'cpp': 'cpp',
  'cs': 'csharp',
  'php': 'php',
  'rb': 'ruby',
  'rs': 'rust',
  'sh': 'bash',
  'sql': 'sql',
} as const

// HTTP 상태 코드
export const HTTP_STATUS = {
  OK: 200,
  CREATED: 201,
  NO_CONTENT: 204,
  BAD_REQUEST: 400,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  CONFLICT: 409,
  UNPROCESSABLE_ENTITY: 422,
  INTERNAL_SERVER_ERROR: 500,
  BAD_GATEWAY: 502,
  SERVICE_UNAVAILABLE: 503,
} as const

// 디바운스/스로틀 기본값
export const DEBOUNCE_DELAYS = {
  SEARCH: 300,
  INPUT: 500,
  SCROLL: 100,
  RESIZE: 250,
  API_CALL: 1000,
} as const

// 애니메이션 지속 시간
export const ANIMATION_DURATIONS = {
  FAST: 150,
  NORMAL: 300,
  SLOW: 500,
  EXTRA_SLOW: 1000,
} as const

// 브레이크포인트
export const BREAKPOINTS = {
  SM: 640,
  MD: 768,
  LG: 1024,
  XL: 1280,
  '2XL': 1536,
} as const

// 최대 파일 크기 (바이트)
export const MAX_FILE_SIZES = {
  IMAGE: 5 * 1024 * 1024,        // 5MB
  DOCUMENT: 10 * 1024 * 1024,    // 10MB
  CODE: 1024 * 1024,             // 1MB
  LOG: 50 * 1024 * 1024,         // 50MB
} as const

// 기본 포트 번호
export const DEFAULT_PORTS = {
  HTTP: 80,
  HTTPS: 443,
  SSH: 22,
  FTP: 21,
  MYSQL: 3306,
  POSTGRES: 5432,
  REDIS: 6379,
  MONGODB: 27017,
  DOCKER: 2375,
  DOCKER_TLS: 2376,
} as const

// 정규 표현식 패턴
export const REGEX_PATTERNS = {
  EMAIL: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
  URL: /^https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)$/,
  PHONE: /^(01[016789]-?)?[0-9]{3,4}-?[0-9]{4}$/,
  PASSWORD: /^(?=.*[a-zA-Z])(?=.*[0-9])(?=.*[!@#$%^&*])[a-zA-Z0-9!@#$%^&*]{8,}$/,
  SLUG: /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
  VERSION: /^\d+\.\d+\.\d+$/,
  IPV4: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/,
  PORT: /^([0-9]{1,4}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$/,
} as const

// 에러 메시지
export const ERROR_MESSAGES = {
  NETWORK_ERROR: '네트워크 연결을 확인해주세요.',
  SERVER_ERROR: '서버 오류가 발생했습니다.',
  UNAUTHORIZED: '인증이 필요합니다.',
  FORBIDDEN: '접근 권한이 없습니다.',
  NOT_FOUND: '요청한 리소스를 찾을 수 없습니다.',
  VALIDATION_ERROR: '입력값을 확인해주세요.',
  TIMEOUT_ERROR: '요청 시간이 초과되었습니다.',
  UNKNOWN_ERROR: '알 수 없는 오류가 발생했습니다.',
} as const

// 성공 메시지
export const SUCCESS_MESSAGES = {
  LOGIN: '로그인되었습니다.',
  LOGOUT: '로그아웃되었습니다.',
  SAVE: '저장되었습니다.',
  DELETE: '삭제되었습니다.',
  CREATE: '생성되었습니다.',
  UPDATE: '업데이트되었습니다.',
  COPY: '클립보드에 복사되었습니다.',
} as const

// 기본 설정값
export const DEFAULT_SETTINGS = {
  THEME: THEMES.SYSTEM,
  LANGUAGE: LANGUAGES.KOREAN,
  PAGINATION_LIMIT: PAGINATION_DEFAULTS.LIMIT,
  AUTO_REFRESH_INTERVAL: 5000, // 5초
  TERMINAL_FONT_SIZE: 14,
  TERMINAL_THEME: 'dark',
  NOTIFICATION_DURATION: 5000, // 5초
} as const