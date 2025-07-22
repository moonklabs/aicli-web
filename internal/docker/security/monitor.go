package security

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SecurityMonitor 보안 모니터링 시스템
type SecurityMonitor struct {
	config           *IsolationConfig
	alertChan        chan SecurityAlert
	violations       map[string][]ResourceViolation
	violationsMutex  sync.RWMutex
	subscribers      map[string][]AlertHandler
	subscribersMutex sync.RWMutex
	running          bool
	runningMutex     sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
}

// AlertHandler 알림 핸들러 함수 타입
type AlertHandler func(SecurityAlert)

// NewSecurityMonitor 새로운 보안 모니터 생성
func NewSecurityMonitor(config *IsolationConfig) *SecurityMonitor {
	if config == nil {
		config = DefaultIsolationConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SecurityMonitor{
		config:      config,
		alertChan:   make(chan SecurityAlert, 100),
		violations:  make(map[string][]ResourceViolation),
		subscribers: make(map[string][]AlertHandler),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// StartMonitoring 모니터링 시작
func (sm *SecurityMonitor) StartMonitoring() <-chan SecurityAlert {
	sm.runningMutex.Lock()
	defer sm.runningMutex.Unlock()
	
	if sm.running {
		return sm.alertChan
	}
	
	sm.running = true
	go sm.monitoringLoop()
	
	return sm.alertChan
}

// StopMonitoring 모니터링 중지
func (sm *SecurityMonitor) StopMonitoring() {
	sm.runningMutex.Lock()
	defer sm.runningMutex.Unlock()
	
	if !sm.running {
		return
	}
	
	sm.running = false
	sm.cancel()
	close(sm.alertChan)
}

// monitoringLoop 모니터링 메인 루프
func (sm *SecurityMonitor) monitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.performSecurityCheck()
		}
	}
}

// performSecurityCheck 보안 검사 수행
func (sm *SecurityMonitor) performSecurityCheck() {
	// 리소스 사용량 검사
	sm.checkResourceUsage()
	
	// 네트워크 활동 감시
	sm.monitorNetworkActivity()
	
	// 비정상 프로세스 감지
	sm.detectAnomalousProcesses()
	
	// 파일 시스템 접근 모니터링
	if sm.config.EnableAuditLog {
		sm.monitorFileSystemAccess()
	}
}

// checkResourceUsage 리소스 사용량 검사
func (sm *SecurityMonitor) checkResourceUsage() {
	// 실제 구현에서는 활성 워크스페이스들의 메트릭 수집 및 분석
	// 현재는 모의 구현
	workspaces := []string{"workspace-1", "workspace-2"} // 모의 워크스페이스 목록
	
	for _, workspaceID := range workspaces {
		// 모의 메트릭 데이터
		metrics := &WorkspaceMetrics{
			WorkspaceID:  workspaceID,
			CPUPercent:   75.5,
			MemoryUsage:  800 * 1024 * 1024, // 800MB
			MemoryLimit:  1024 * 1024 * 1024, // 1GB
			NetworkRx:    50 * 1024 * 1024,   // 50MB/s
			NetworkTx:    30 * 1024 * 1024,   // 30MB/s
			ProcessCount: 25,
			Timestamp:    time.Now(),
		}
		
		// ResourceManager를 통한 위반 검사
		rm := NewResourceManager(sm.config)
		violations := rm.ValidateResourceUsage(metrics)
		
		// 위반사항 처리
		for _, violation := range violations {
			sm.ReportViolation(workspaceID, violation)
		}
	}
}

// monitorNetworkActivity 네트워크 활동 감시
func (sm *SecurityMonitor) monitorNetworkActivity() {
	// 실제 구현에서는 네트워크 트래픽 분석
	// 현재는 모의 구현으로 의심스러운 활동 시뮬레이션
	
	suspiciousActivity := false // 모의 플래그
	if suspiciousActivity {
		alert := SecurityAlert{
			Type:        AlertTypeNetworkAnomaly,
			WorkspaceID: "workspace-1",
			Severity:    "warning",
			Message:     "Suspicious network activity detected: high outbound traffic to unknown hosts",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"traffic_type": "outbound",
				"bytes_sent":   100 * 1024 * 1024, // 100MB
				"destinations": []string{"192.168.1.100", "10.0.0.50"},
			},
		}
		sm.sendAlert(alert)
	}
}

// detectAnomalousProcesses 비정상 프로세스 감지
func (sm *SecurityMonitor) detectAnomalousProcesses() {
	// 실제 구현에서는 컨테이너 내부 프로세스 모니터링
	// 현재는 모의 구현
	
	anomalousProcess := false // 모의 플래그
	if anomalousProcess {
		alert := SecurityAlert{
			Type:        AlertTypeProcessAnomaly,
			WorkspaceID: "workspace-2",
			Severity:    "critical",
			Message:     "Anomalous process detected: unauthorized privilege escalation attempt",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"process_name": "suspicious_binary",
				"pid":          12345,
				"ppid":         1,
				"user":         "root",
				"command_line": "/tmp/suspicious_binary --escalate",
			},
		}
		sm.sendAlert(alert)
	}
}

// monitorFileSystemAccess 파일 시스템 접근 모니터링
func (sm *SecurityMonitor) monitorFileSystemAccess() {
	// 실제 구현에서는 auditd 또는 파일 시스템 이벤트 모니터링
	// 현재는 모의 구현
	
	unauthorizedAccess := false // 모의 플래그
	if unauthorizedAccess {
		alert := SecurityAlert{
			Type:        AlertTypeSecurityBreach,
			WorkspaceID: "workspace-1",
			Severity:    "error",
			Message:     "Unauthorized file access detected: attempt to read sensitive configuration",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"file_path":   "/etc/passwd",
				"access_type": "read",
				"process":     "cat",
				"user":        "user1",
			},
		}
		sm.sendAlert(alert)
	}
}

// ReportViolation 보안 위반사항 보고
func (sm *SecurityMonitor) ReportViolation(workspaceID string, violation ResourceViolation) {
	sm.violationsMutex.Lock()
	sm.violations[workspaceID] = append(sm.violations[workspaceID], violation)
	sm.violationsMutex.Unlock()
	
	alert := SecurityAlert{
		Type:        AlertTypeResourceViolation,
		WorkspaceID: workspaceID,
		Severity:    violation.Severity,
		Message:     fmt.Sprintf("Resource violation detected: %s", violation.Description),
		Timestamp:   time.Now(),
		Data:        violation,
	}
	
	sm.sendAlert(alert)
}

// ReportSecurityBreach 보안 침해 보고
func (sm *SecurityMonitor) ReportSecurityBreach(workspaceID string, breach SecurityBreach) {
	alert := SecurityAlert{
		Type:        AlertTypeSecurityBreach,
		WorkspaceID: workspaceID,
		Severity:    breach.RiskLevel,
		Message:     fmt.Sprintf("Security breach detected: %s", breach.Description),
		Timestamp:   time.Now(),
		Data:        breach,
	}
	
	sm.sendAlert(alert)
	sm.HandleSecurityBreach(workspaceID, breach)
}

// sendAlert 알림 전송
func (sm *SecurityMonitor) sendAlert(alert SecurityAlert) {
	select {
	case sm.alertChan <- alert:
		// 구독자들에게 알림 전파
		sm.notifySubscribers(alert)
	default:
		// 채널이 가득 찬 경우 로깅 (실제 구현에서는 로거 사용)
		fmt.Printf("Alert channel full, dropping alert: %s\n", alert.Message)
	}
}

// notifySubscribers 구독자들에게 알림 전파
func (sm *SecurityMonitor) notifySubscribers(alert SecurityAlert) {
	sm.subscribersMutex.RLock()
	handlers := append([]AlertHandler{}, sm.subscribers[alert.WorkspaceID]...)
	globalHandlers := append([]AlertHandler{}, sm.subscribers["*"]...)
	sm.subscribersMutex.RUnlock()
	
	// 워크스페이스별 핸들러 실행
	for _, handler := range handlers {
		go handler(alert)
	}
	
	// 글로벌 핸들러 실행
	for _, handler := range globalHandlers {
		go handler(alert)
	}
}

// Subscribe 알림 구독
func (sm *SecurityMonitor) Subscribe(workspaceID string, handler AlertHandler) {
	sm.subscribersMutex.Lock()
	defer sm.subscribersMutex.Unlock()
	
	sm.subscribers[workspaceID] = append(sm.subscribers[workspaceID], handler)
}

// Unsubscribe 알림 구독 해제
func (sm *SecurityMonitor) Unsubscribe(workspaceID string) {
	sm.subscribersMutex.Lock()
	defer sm.subscribersMutex.Unlock()
	
	delete(sm.subscribers, workspaceID)
}

// HandleSecurityBreach 보안 침해 자동 대응
func (sm *SecurityMonitor) HandleSecurityBreach(workspaceID string, breach SecurityBreach) {
	switch breach.Type {
	case "privilege_escalation":
		sm.handlePrivilegeEscalation(workspaceID, breach)
	case "suspicious_network_activity":
		sm.handleNetworkAnomaly(workspaceID, breach)
	case "unauthorized_file_access":
		sm.handleFileAccessViolation(workspaceID, breach)
	case "resource_exhaustion":
		sm.handleResourceExhaustion(workspaceID, breach)
	default:
		sm.handleGenericBreach(workspaceID, breach)
	}
}

// handlePrivilegeEscalation 권한 상승 시도 처리
func (sm *SecurityMonitor) handlePrivilegeEscalation(workspaceID string, breach SecurityBreach) {
	// 1. 컨테이너 일시 중지 (실제 구현에서는 Docker API 호출)
	fmt.Printf("Pausing container for workspace %s due to privilege escalation\n", workspaceID)
	
	// 2. 관리자 알림
	adminAlert := SecurityAlert{
		Type:        AlertTypeSecurityBreach,
		WorkspaceID: workspaceID,
		Severity:    "critical",
		Message:     "CRITICAL: Privilege escalation detected - container paused",
		Timestamp:   time.Now(),
		Data:        breach,
	}
	sm.sendAlert(adminAlert)
	
	// 3. 보안 로그 기록
	sm.logSecurityEvent(workspaceID, "privilege_escalation", breach)
}

// handleNetworkAnomaly 네트워크 이상 활동 처리
func (sm *SecurityMonitor) handleNetworkAnomaly(workspaceID string, breach SecurityBreach) {
	// 네트워크 트래픽 제한 (실제 구현에서는 iptables 규칙 적용)
	fmt.Printf("Applying network restrictions for workspace %s\n", workspaceID)
	
	// 모니터링 강화
	sm.increaseMonitoringFrequency(workspaceID)
}

// handleFileAccessViolation 파일 접근 위반 처리
func (sm *SecurityMonitor) handleFileAccessViolation(workspaceID string, breach SecurityBreach) {
	// 파일 시스템 감시 강화
	fmt.Printf("Increasing file system monitoring for workspace %s\n", workspaceID)
	
	// 접근 로그 상세 기록
	sm.enableDetailedAuditLog(workspaceID)
}

// handleResourceExhaustion 리소스 고갈 처리
func (sm *SecurityMonitor) handleResourceExhaustion(workspaceID string, breach SecurityBreach) {
	// 리소스 제한 강화
	fmt.Printf("Applying stricter resource limits for workspace %s\n", workspaceID)
	
	// 프로세스 모니터링 강화
	sm.increaseProcessMonitoring(workspaceID)
}

// handleGenericBreach 일반적인 보안 침해 처리
func (sm *SecurityMonitor) handleGenericBreach(workspaceID string, breach SecurityBreach) {
	// 기본 대응: 로깅 및 알림
	sm.logSecurityEvent(workspaceID, "generic_breach", breach)
	
	// 모니터링 레벨 증가
	sm.increaseMonitoringFrequency(workspaceID)
}

// logSecurityEvent 보안 이벤트 로깅
func (sm *SecurityMonitor) logSecurityEvent(workspaceID, eventType string, data interface{}) {
	// 실제 구현에서는 구조화된 로깅 시스템 사용
	fmt.Printf("[SECURITY] %s - %s: %v\n", time.Now().Format(time.RFC3339), eventType, data)
}

// increaseMonitoringFrequency 모니터링 빈도 증가
func (sm *SecurityMonitor) increaseMonitoringFrequency(workspaceID string) {
	// 실제 구현에서는 워크스페이스별 모니터링 설정 조정
	fmt.Printf("Increasing monitoring frequency for workspace %s\n", workspaceID)
}

// enableDetailedAuditLog 상세 감사 로그 활성화
func (sm *SecurityMonitor) enableDetailedAuditLog(workspaceID string) {
	// 실제 구현에서는 auditd 설정 동적 변경
	fmt.Printf("Enabling detailed audit log for workspace %s\n", workspaceID)
}

// increaseProcessMonitoring 프로세스 모니터링 강화
func (sm *SecurityMonitor) increaseProcessMonitoring(workspaceID string) {
	// 실제 구현에서는 프로세스 감시 도구 설정 변경
	fmt.Printf("Increasing process monitoring for workspace %s\n", workspaceID)
}

// GetSecurityDashboard 보안 대시보드 데이터 반환
func (sm *SecurityMonitor) GetSecurityDashboard() *SecurityDashboard {
	sm.violationsMutex.RLock()
	defer sm.violationsMutex.RUnlock()
	
	totalAlerts := len(sm.alertChan)
	criticalAlerts := sm.countAlertsBySeverity("critical")
	warningAlerts := sm.countAlertsBySeverity("warning")
	violationSummary := sm.getViolationSummary()
	
	return &SecurityDashboard{
		TotalAlerts:      totalAlerts,
		CriticalAlerts:   criticalAlerts,
		WarningAlerts:    warningAlerts,
		ViolationSummary: violationSummary,
		LastUpdated:      time.Now(),
		MonitoringStatus: sm.getMonitoringStatus(),
		ActiveWorkspaces: sm.getActiveWorkspaces(),
	}
}

// countAlertsBySeverity 심각도별 알림 수 집계
func (sm *SecurityMonitor) countAlertsBySeverity(severity string) int {
	// 실제 구현에서는 알림 히스토리 저장소에서 쿼리
	// 현재는 모의 구현
	switch severity {
	case "critical":
		return 2
	case "warning":
		return 5
	case "info":
		return 10
	default:
		return 0
	}
}

// getViolationSummary 위반사항 요약 반환
func (sm *SecurityMonitor) getViolationSummary() map[string]int {
	summary := make(map[string]int)
	for _, violations := range sm.violations {
		for _, violation := range violations {
			summary[violation.Type]++
		}
	}
	return summary
}

// getMonitoringStatus 모니터링 상태 반환
func (sm *SecurityMonitor) getMonitoringStatus() string {
	sm.runningMutex.RLock()
	defer sm.runningMutex.RUnlock()
	
	if sm.running {
		return "active"
	}
	return "stopped"
}

// getActiveWorkspaces 활성 워크스페이스 목록 반환
func (sm *SecurityMonitor) getActiveWorkspaces() []string {
	sm.violationsMutex.RLock()
	defer sm.violationsMutex.RUnlock()
	
	var workspaces []string
	for workspaceID := range sm.violations {
		workspaces = append(workspaces, workspaceID)
	}
	return workspaces
}

// GetWorkspaceViolations 워크스페이스별 위반사항 반환
func (sm *SecurityMonitor) GetWorkspaceViolations(workspaceID string) []ResourceViolation {
	sm.violationsMutex.RLock()
	defer sm.violationsMutex.RUnlock()
	
	return sm.violations[workspaceID]
}

// ClearViolations 위반사항 기록 정리
func (sm *SecurityMonitor) ClearViolations(workspaceID string) {
	sm.violationsMutex.Lock()
	defer sm.violationsMutex.Unlock()
	
	if workspaceID == "" {
		// 모든 워크스페이스 정리
		sm.violations = make(map[string][]ResourceViolation)
	} else {
		// 특정 워크스페이스만 정리
		delete(sm.violations, workspaceID)
	}
}

// SecurityAlert 보안 알림
type SecurityAlert struct {
	Type        AlertType   `json:"type"`
	WorkspaceID string      `json:"workspace_id"`
	Severity    string      `json:"severity"`
	Message     string      `json:"message"`
	Timestamp   time.Time   `json:"timestamp"`
	Data        interface{} `json:"data,omitempty"`
}

// AlertType 알림 타입
type AlertType string

const (
	AlertTypeResourceViolation AlertType = "resource_violation"
	AlertTypeSecurityBreach    AlertType = "security_breach"
	AlertTypeNetworkAnomaly    AlertType = "network_anomaly"
	AlertTypeProcessAnomaly    AlertType = "process_anomaly"
)

// SecurityBreach 보안 침해 정보
type SecurityBreach struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Evidence    interface{} `json:"evidence"`
	RiskLevel   string      `json:"risk_level"`
	Timestamp   time.Time   `json:"timestamp"`
}

// SecurityDashboard 보안 대시보드 데이터
type SecurityDashboard struct {
	TotalAlerts      int            `json:"total_alerts"`
	CriticalAlerts   int            `json:"critical_alerts"`
	WarningAlerts    int            `json:"warning_alerts"`
	ViolationSummary map[string]int `json:"violation_summary"`
	LastUpdated      time.Time      `json:"last_updated"`
	MonitoringStatus string         `json:"monitoring_status"`
	ActiveWorkspaces []string       `json:"active_workspaces"`
}