// API 공통 타입 정의

export interface ApiResponse<T = any> {
  success: boolean
  data: T
  message?: string
  error?: string
  timestamp: string
}

export interface PaginatedResponse<T = any> {
  items: T[]
  total: number
  page: number
  limit: number
  totalPages: number
  hasNext: boolean
  hasPrev: boolean
}

export interface ApiError {
  code: string
  message: string
  details?: Record<string, any>
  timestamp: string
}

// 인증 관련 타입
export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  refreshToken: string
  user: {
    id: string
    username: string
    email: string
    displayName?: string
    avatar?: string
    roles?: string[]
  }
  expiresIn: number
}

export interface RefreshTokenRequest {
  refreshToken: string
}

// OAuth 관련 타입
export interface OAuthProvider {
  name: 'google' | 'github'
  displayName: string
  icon: string
  enabled: boolean
}

export interface OAuthAuthUrlRequest {
  provider: string
  state?: string
}

export interface OAuthAuthUrlResponse {
  authUrl: string
  state: string
}

export interface OAuthCallbackRequest {
  provider: string
  code: string
  state: string
}

export interface OAuthUserInfo {
  id: string
  email: string
  name: string
  picture?: string
  verified: boolean
  provider: string
}

export interface OAuthAccount {
  id: string
  provider: string
  providerId: string
  email: string
  name: string
  picture?: string
  connected: boolean
  connectedAt: string
}

export interface LinkOAuthRequest {
  provider: string
  code: string
  state: string
}

export interface UnlinkOAuthRequest {
  provider: string
}

// 워크스페이스 관련 타입
export interface CreateWorkspaceRequest {
  name: string
  path: string
  description?: string
  gitRemote?: string
  gitBranch?: string
  config?: {
    baseImage?: string
    workingDir?: string
    environment?: Record<string, string>
    ports?: number[]
    volumes?: string[]
  }
}

export interface UpdateWorkspaceRequest {
  name?: string
  description?: string
  gitRemote?: string
  gitBranch?: string
}

// 터미널 관련 타입
export interface CreateTerminalRequest {
  workspaceId: string
  title?: string
  workingDir?: string
  environment?: Record<string, string>
}

export interface ExecuteCommandRequest {
  command: string
  workingDir?: string
  environment?: Record<string, string>
}

// Docker 관련 타입
export interface DockerContainerInfo {
  id: string
  name: string
  image: string
  status: string
  state: string
  ports: Array<{
    privatePort: number
    publicPort?: number
    type: string
    ip?: string
  }>
  mounts: Array<{
    source: string
    destination: string
    mode: string
    type: string
  }>
  createdAt: string
  startedAt?: string
  finishedAt?: string
  workspaceId?: string
  environment?: Record<string, string>
}

export interface DockerImageInfo {
  id: string
  repository: string
  tag: string
  size: number
  created: string
}

export interface DockerStats {
  containerId: string
  cpuPercent: number
  memoryUsage: number
  memoryLimit: number
  memoryPercent: number
  networkRx: number
  networkTx: number
  blockRead: number
  blockWrite: number
  pids: number
  timestamp: string
}

// 작업 관련 타입
export interface TaskInfo {
  id: string
  type: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  progress: number
  message?: string
  result?: any
  error?: string
  createdAt: string
  updatedAt: string
  completedAt?: string
}

// RBAC 관련 타입
export interface Role {
  id: string
  name: string
  description: string
  parentId?: string
  level: number
  isSystem: boolean
  isActive: boolean
  permissions?: Permission[]
  createdAt: string
  updatedAt: string
}

export interface Permission {
  id: string
  name: string
  description: string
  resourceType: ResourceType
  action: ActionType
  effect: PermissionEffect
  conditions?: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export type ResourceType = 'workspace' | 'project' | 'session' | 'task' | 'user' | 'system'
export type ActionType = 'create' | 'read' | 'update' | 'delete' | 'execute' | 'manage'
export type PermissionEffect = 'allow' | 'deny'

export interface UserRole {
  userId: string
  roleId: string
  assignedBy: string
  resourceId?: string
  expiresAt?: string
  isActive: boolean
  role?: Role
  createdAt: string
  updatedAt: string
}

export interface PermissionCheck {
  userID: string
  resourceType: ResourceType
  resourceID: string
  action: ActionType
  attributes?: Record<string, string>
}

export interface PermissionCheckResponse {
  allowed: boolean
  decision: {
    resourceType: ResourceType
    resourceId: string
    action: ActionType
    effect: PermissionEffect
    source: string
    reason: string
    conditions?: string
  }
  evaluation: string[]
}

export interface UserPermissions {
  userId: string
  directRoles: string[]
  inheritedRoles: string[]
  groupRoles: string[]
  finalPermissions: Record<string, {
    resourceType: ResourceType
    resourceId: string
    action: ActionType
    effect: PermissionEffect
    source: string
    reason: string
    conditions?: string
  }>
  computedAt: string
}

// 세션 관리 관련 타입
export interface UserSession {
  id: string
  userId: string
  deviceInfo: {
    browser: string
    os: string
    device: string
    userAgent: string
  }
  locationInfo?: {
    ip: string
    country?: string
    city?: string
    timezone?: string
  }
  isCurrentSession: boolean
  createdAt: string
  lastActivityAt: string
  expiresAt: string
  status: 'active' | 'expired' | 'terminated'
}

export interface SessionSecurityEvent {
  id: string
  sessionId: string
  userId: string
  eventType: 'login' | 'logout' | 'suspicious_activity' | 'password_change' | 'device_change' | 'location_change'
  severity: 'low' | 'medium' | 'high' | 'critical'
  description: string
  metadata: Record<string, any>
  ipAddress: string
  userAgent: string
  createdAt: string
}

export interface SessionSecuritySettings {
  userId: string
  sessionTimeoutMinutes: number
  maxConcurrentSessions: number
  allowMultipleDevices: boolean
  requireReauthForSensitiveActions: boolean
  notifyOnNewDevice: boolean
  notifyOnSuspiciousActivity: boolean
  autoTerminateInactiveSessions: boolean
  inactivityTimeoutMinutes: number
  updatedAt: string
}

export interface TerminateSessionRequest {
  sessionId: string
  reason?: string
}

export interface TerminateAllSessionsRequest {
  excludeCurrentSession: boolean
  reason?: string
}

export interface UpdateSessionSettingsRequest {
  sessionTimeoutMinutes?: number
  maxConcurrentSessions?: number
  allowMultipleDevices?: boolean
  requireReauthForSensitiveActions?: boolean
  notifyOnNewDevice?: boolean
  notifyOnSuspiciousActivity?: boolean
  autoTerminateInactiveSessions?: boolean
  inactivityTimeoutMinutes?: number
}

export interface SessionStatsResponse {
  totalActiveSessions: number
  currentDevices: number
  suspiciousActivities: number
  lastPasswordChange?: string
  accountCreatedAt: string
}

// WebSocket 메시지 타입
export interface WebSocketMessage {
  type: string
  payload: any
  timestamp: string
}

export interface TerminalMessage extends WebSocketMessage {
  sessionId: string
  payload: {
    type: 'input' | 'output' | 'error' | 'system'
    content: string
    level?: 'info' | 'warn' | 'error' | 'debug'
  }
}

export interface WorkspaceStatusMessage extends WebSocketMessage {
  workspaceId: string
  payload: {
    status: string
    containerId?: string
    message?: string
  }
}

export interface SessionUpdateMessage extends WebSocketMessage {
  payload: {
    type: 'session_created' | 'session_terminated' | 'session_activity' | 'security_event'
    sessionId: string
    userId: string
    data: UserSession | SessionSecurityEvent
  }
}

// 사용자 프로파일 관련 타입
export interface UserProfile {
  id: string
  username: string
  email: string
  displayName?: string
  firstName?: string
  lastName?: string
  avatar?: string
  bio?: string
  phone?: string
  birthDate?: string
  website?: string
  location?: string
  timezone?: string
  language?: string
  theme?: 'light' | 'dark' | 'auto'
  isActive: boolean
  isEmailVerified: boolean
  isPhoneVerified: boolean
  twoFactorEnabled: boolean
  lastPasswordChange?: string
  createdAt: string
  updatedAt: string
  lastLoginAt?: string
}

export interface UpdateProfileRequest {
  displayName?: string
  firstName?: string
  lastName?: string
  bio?: string
  phone?: string
  birthDate?: string
  website?: string
  location?: string
  timezone?: string
  language?: string
  theme?: 'light' | 'dark' | 'auto'
}

export interface ChangePasswordRequest {
  currentPassword: string
  newPassword: string
  confirmPassword: string
}

export interface PasswordStrength {
  score: number // 0-4
  feedback: {
    warning: string
    suggestions: string[]
  }
  crackTime: {
    offlineSlowHashing: string
    onlineThrottling: string
  }
}

export interface TwoFactorAuthSettings {
  enabled: boolean
  secret?: string
  qrCodeUrl?: string
  backupCodes?: string[]
  setupComplete: boolean
  lastUsed?: string
}

export interface EnableTwoFactorRequest {
  token: string
}

export interface VerifyTwoFactorRequest {
  token: string
}

export interface NotificationSettings {
  userId: string
  emailNotifications: {
    loginNotifications: boolean
    securityAlerts: boolean
    workspaceUpdates: boolean
    systemUpdates: boolean
    promotionalEmails: boolean
  }
  pushNotifications: {
    enabled: boolean
    securityAlerts: boolean
    workspaceUpdates: boolean
    directMessages: boolean
  }
  smsNotifications: {
    enabled: boolean
    securityAlerts: boolean
    criticalUpdates: boolean
  }
  updatedAt: string
}

export interface UpdateNotificationSettingsRequest {
  emailNotifications?: Partial<NotificationSettings['emailNotifications']>
  pushNotifications?: Partial<NotificationSettings['pushNotifications']>
  smsNotifications?: Partial<NotificationSettings['smsNotifications']>
}

export interface PrivacySettings {
  userId: string
  profileVisibility: 'public' | 'private' | 'contacts'
  showEmail: boolean
  showPhone: boolean
  showOnlineStatus: boolean
  allowDirectMessages: boolean
  allowFriendRequests: boolean
  dataProcessingConsent: boolean
  marketingConsent: boolean
  analyticsConsent: boolean
  updatedAt: string
}

export interface UpdatePrivacySettingsRequest {
  profileVisibility?: 'public' | 'private' | 'contacts'
  showEmail?: boolean
  showPhone?: boolean
  showOnlineStatus?: boolean
  allowDirectMessages?: boolean
  allowFriendRequests?: boolean
  dataProcessingConsent?: boolean
  marketingConsent?: boolean
  analyticsConsent?: boolean
}

export interface ProfileImageUploadRequest {
  file: File
  cropData?: {
    x: number
    y: number
    width: number
    height: number
    rotate?: number
    scaleX?: number
    scaleY?: number
  }
}

export interface ProfileImageUploadResponse {
  imageUrl: string
  thumbnailUrl: string
  originalUrl: string
}

export interface AccountDeletionRequest {
  password: string
  reason?: string
  feedback?: string
  deleteImmediately?: boolean
}

export interface AccountDeactivationRequest {
  password: string
  reason?: string
}

export interface AccountReactivationRequest {
  email: string
}

// 보안 모니터링 및 감사 로그 관련 타입
export interface AuditLog {
  id: string
  userId: string
  sessionId?: string
  action: string
  resourceType: string
  resourceId?: string
  ipAddress: string
  userAgent: string
  location?: {
    country?: string
    city?: string
    timezone?: string
  }
  metadata: Record<string, any>
  severity: 'low' | 'medium' | 'high' | 'critical'
  status: 'success' | 'failure' | 'blocked'
  createdAt: string
}

export interface SecurityEventFilter {
  userId?: string
  eventType?: string[]
  severity?: string[]
  startDate?: string
  endDate?: string
  ipAddress?: string
  status?: string[]
  limit?: number
  offset?: number
}

export interface LoginHistory {
  id: string
  userId: string
  sessionId: string
  ipAddress: string
  userAgent: string
  location?: {
    country?: string
    city?: string
    timezone?: string
  }
  deviceInfo: {
    browser: string
    os: string
    device: string
  }
  loginMethod: 'password' | 'oauth' | 'sso' | 'token'
  provider?: string // OAuth provider if applicable
  status: 'success' | 'failure' | 'blocked'
  failureReason?: string
  isSuspicious: boolean
  riskScore: number // 0-100
  createdAt: string
}

export interface SecurityStats {
  totalLogins: number
  successfulLogins: number
  failedLogins: number
  suspiciousAttempts: number
  blockedAttempts: number
  uniqueDevices: number
  uniqueLocations: number
  riskScore: number // 0-100, overall risk score
  lastActivity: string
  periodStart: string
  periodEnd: string
  trends: {
    loginsChange: number // percentage change
    suspiciousChange: number
    riskScoreChange: number
  }
}

export interface SuspiciousActivity {
  id: string
  userId: string
  activityType: 'unusual_location' | 'unusual_device' | 'brute_force' | 'credential_stuffing' | 'account_takeover' | 'privilege_escalation'
  description: string
  riskScore: number // 0-100
  indicators: string[]
  metadata: Record<string, any>
  ipAddress: string
  userAgent: string
  location?: {
    country?: string
    city?: string
    timezone?: string
  }
  isResolved: boolean
  resolvedBy?: string
  resolvedAt?: string
  resolution?: string
  createdAt: string
}

export interface LogExportRequest {
  type: 'audit' | 'security' | 'login_history' | 'suspicious_activity'
  format: 'csv' | 'json' | 'xlsx'
  filters?: SecurityEventFilter
  fields?: string[]
  includeMetadata?: boolean
}

export interface LogExportResponse {
  downloadUrl: string
  filename: string
  size: number
  recordCount: number
  expiresAt: string
}