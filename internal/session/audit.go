package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// AuditLogger는 세션 감사 로그 시스템입니다.
type AuditLogger struct {
	client    redis.UniversalClient
	keyPrefix string
	
	// 감사 로그 설정
	retention    time.Duration // 로그 보관 기간
	batchSize    int          // 배치 쓰기 크기
	enableAsync  bool         // 비동기 로깅 활성화
	
	// 비동기 처리
	logChannel chan *AuditLog
	stopCh     chan struct{}
}

// NewAuditLogger는 새로운 감사 로거를 생성합니다.
func NewAuditLogger(client redis.UniversalClient, keyPrefix string) *AuditLogger {
	return &AuditLogger{
		client:     client,
		keyPrefix:  keyPrefix,
		retention:  time.Hour * 24 * 90, // 90일 보관
		batchSize:  100,
		enableAsync: true,
		logChannel: make(chan *AuditLog, 1000),
		stopCh:     make(chan struct{}),
	}
}

// auditKey는 감사 로그 키를 생성합니다.
func (a *AuditLogger) auditKey(category string) string {
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s:audit:%s:%s", a.keyPrefix, category, date)
}

// userAuditKey는 사용자별 감사 로그 키를 생성합니다.
func (a *AuditLogger) userAuditKey(userID string) string {
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s:audit:user:%s:%s", a.keyPrefix, userID, date)
}

// Start는 감사 로거를 시작합니다 (비동기 모드).
func (a *AuditLogger) Start(ctx context.Context) {
	if !a.enableAsync {
		return
	}
	
	go a.processBatchLogs(ctx)
}

// Stop은 감사 로거를 중지합니다.
func (a *AuditLogger) Stop() {
	if a.enableAsync {
		close(a.stopCh)
	}
}

// processBatchLogs는 배치로 로그를 처리합니다.
func (a *AuditLogger) processBatchLogs(ctx context.Context) {
	batch := make([]*AuditLog, 0, a.batchSize)
	ticker := time.NewTicker(time.Second * 5) // 5초마다 배치 처리
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			// 남은 로그들 처리
			if len(batch) > 0 {
				a.writeBatch(ctx, batch)
			}
			return
			
		case <-a.stopCh:
			if len(batch) > 0 {
				a.writeBatch(ctx, batch)
			}
			return
			
		case log := <-a.logChannel:
			batch = append(batch, log)
			if len(batch) >= a.batchSize {
				a.writeBatch(ctx, batch)
				batch = batch[:0] // 배치 초기화
			}
			
		case <-ticker.C:
			if len(batch) > 0 {
				a.writeBatch(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

// writeBatch는 배치 로그를 Redis에 쓰기합니다.
func (a *AuditLogger) writeBatch(ctx context.Context, logs []*AuditLog) {
	for _, log := range logs {
		if err := a.writeLog(ctx, log); err != nil {
			// 로깅 실패 시 stderr에 출력 (추후 다른 로거로 대체 가능)
			fmt.Printf("감사 로그 쓰기 실패: %v\n", err)
		}
	}
}

// writeLog는 개별 로그를 Redis에 쓰기합니다.
func (a *AuditLogger) writeLog(ctx context.Context, log *AuditLog) error {
	// 로그 직렬화
	logData, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("로그 직렬화 실패: %w", err)
	}
	
	// 전체 감사 로그에 추가
	auditKey := a.auditKey(log.Category)
	if err := a.client.LPush(ctx, auditKey, logData).Err(); err != nil {
		return fmt.Errorf("감사 로그 저장 실패: %w", err)
	}
	
	// 사용자별 감사 로그에도 추가 (사용자 ID가 있는 경우)
	if log.UserID != "" {
		userKey := a.userAuditKey(log.UserID)
		a.client.LPush(ctx, userKey, logData)
		a.client.Expire(ctx, userKey, a.retention)
	}
	
	// TTL 설정
	a.client.Expire(ctx, auditKey, a.retention)
	
	return nil
}

// LogSessionEvent는 세션 이벤트를 감사 로그에 기록합니다.
func (a *AuditLogger) LogSessionEvent(ctx context.Context, event *SessionEvent) error {
	auditLog := &AuditLog{
		ID:          generateAuditID(),
		Timestamp:   event.Timestamp,
		Category:    "session",
		Action:      string(event.EventType),
		UserID:      event.UserID,
		SessionID:   event.SessionID,
		IPAddress:   event.IPAddress,
		UserAgent:   event.UserAgent,
		Location:    event.Location,
		Severity:    string(event.Severity),
		Description: event.Description,
		Details:     event.EventData,
		Source:      "session_manager",
	}
	
	if a.enableAsync {
		select {
		case a.logChannel <- auditLog:
			return nil
		default:
			// 채널이 가득 찬 경우 동기적으로 처리
			return a.writeLog(ctx, auditLog)
		}
	}
	
	return a.writeLog(ctx, auditLog)
}

// LogSecurityEvent는 보안 이벤트를 감사 로그에 기록합니다.
func (a *AuditLogger) LogSecurityEvent(ctx context.Context, userID, action, description string, details map[string]interface{}) error {
	auditLog := &AuditLog{
		ID:          generateAuditID(),
		Timestamp:   time.Now(),
		Category:    "security",
		Action:      action,
		UserID:      userID,
		Severity:    "critical",
		Description: description,
		Details:     details,
		Source:      "security_checker",
	}
	
	if a.enableAsync {
		select {
		case a.logChannel <- auditLog:
			return nil
		default:
			return a.writeLog(ctx, auditLog)
		}
	}
	
	return a.writeLog(ctx, auditLog)
}

// LogAdminAction은 관리자 작업을 감사 로그에 기록합니다.
func (a *AuditLogger) LogAdminAction(ctx context.Context, adminID, action, targetUserID, description string, details map[string]interface{}) error {
	auditLog := &AuditLog{
		ID:          generateAuditID(),
		Timestamp:   time.Now(),
		Category:    "admin",
		Action:      action,
		UserID:      adminID,
		TargetID:    targetUserID,
		Severity:    "warning",
		Description: description,
		Details:     details,
		Source:      "admin_panel",
	}
	
	if a.enableAsync {
		select {
		case a.logChannel <- auditLog:
			return nil
		default:
			return a.writeLog(ctx, auditLog)
		}
	}
	
	return a.writeLog(ctx, auditLog)
}

// GetAuditLogs는 감사 로그를 조회합니다.
func (a *AuditLogger) GetAuditLogs(ctx context.Context, filter *AuditFilter) ([]*AuditLog, error) {
	if filter == nil {
		filter = &AuditFilter{Limit: 100}
	}
	
	var keys []string
	
	// 사용자별 필터가 있는 경우
	if filter.UserID != "" {
		if filter.StartDate != "" && filter.EndDate != "" {
			// 날짜 범위별로 키 생성
			start, _ := time.Parse("2006-01-02", filter.StartDate)
			end, _ := time.Parse("2006-01-02", filter.EndDate)
			
			for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
				key := fmt.Sprintf("%s:audit:user:%s:%s", a.keyPrefix, filter.UserID, d.Format("2006-01-02"))
				keys = append(keys, key)
			}
		} else {
			// 오늘 로그만
			key := a.userAuditKey(filter.UserID)
			keys = append(keys, key)
		}
	} else {
		// 전체 감사 로그
		if filter.Category != "" {
			key := a.auditKey(filter.Category)
			keys = append(keys, key)
		} else {
			// 모든 카테고리
			categories := []string{"session", "security", "admin"}
			for _, cat := range categories {
				key := a.auditKey(cat)
				keys = append(keys, key)
			}
		}
	}
	
	var allLogs []*AuditLog
	
	// 각 키에서 로그 조회
	for _, key := range keys {
		logs, err := a.client.LRange(ctx, key, 0, int64(filter.Limit-1)).Result()
		if err != nil {
			continue // 에러가 있는 키는 건너뛰기
		}
		
		for _, logData := range logs {
			var auditLog AuditLog
			if err := json.Unmarshal([]byte(logData), &auditLog); err != nil {
				continue
			}
			
			// 필터 적용
			if a.matchFilter(&auditLog, filter) {
				allLogs = append(allLogs, &auditLog)
			}
		}
		
		if len(allLogs) >= filter.Limit {
			break
		}
	}
	
	// 시간순 정렬 (최신순)
	for i := 0; i < len(allLogs)-1; i++ {
		for j := i + 1; j < len(allLogs); j++ {
			if allLogs[i].Timestamp.Before(allLogs[j].Timestamp) {
				allLogs[i], allLogs[j] = allLogs[j], allLogs[i]
			}
		}
	}
	
	// 제한 적용
	if len(allLogs) > filter.Limit {
		allLogs = allLogs[:filter.Limit]
	}
	
	return allLogs, nil
}

// matchFilter는 로그가 필터 조건에 맞는지 확인합니다.
func (a *AuditLogger) matchFilter(log *AuditLog, filter *AuditFilter) bool {
	if filter.Action != "" && log.Action != filter.Action {
		return false
	}
	
	if filter.Severity != "" && log.Severity != filter.Severity {
		return false
	}
	
	if filter.IPAddress != "" && log.IPAddress != filter.IPAddress {
		return false
	}
	
	return true
}

// CleanupOldLogs는 오래된 감사 로그를 정리합니다.
func (a *AuditLogger) CleanupOldLogs(ctx context.Context) error {
	// 보관 기간을 넘은 키들을 찾아서 삭제
	cutoffDate := time.Now().Add(-a.retention)
	
	patterns := []string{
		fmt.Sprintf("%s:audit:session:*", a.keyPrefix),
		fmt.Sprintf("%s:audit:security:*", a.keyPrefix),
		fmt.Sprintf("%s:audit:admin:*", a.keyPrefix),
		fmt.Sprintf("%s:audit:user:*", a.keyPrefix),
	}
	
	for _, pattern := range patterns {
		var cursor uint64
		
		for {
			keys, nextCursor, err := a.client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				break
			}
			
			for _, key := range keys {
				// 키에서 날짜 추출하여 비교
				// 간단한 구현: TTL 기반으로 처리 (Redis의 자동 만료 활용)
				ttl := a.client.TTL(ctx, key).Val()
				if ttl < 0 {
					// TTL이 설정되지 않은 키는 새로 설정
					a.client.Expire(ctx, key, a.retention)
				}
			}
			
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
	}
	
	return nil
}

// AuditLog는 감사 로그 항목입니다.
type AuditLog struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Category    string                 `json:"category"`    // session, security, admin
	Action      string                 `json:"action"`      // 수행된 작업
	UserID      string                 `json:"user_id"`     // 작업 수행자
	TargetID    string                 `json:"target_id,omitempty"` // 작업 대상 (관리자 작업 시)
	SessionID   string                 `json:"session_id,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Location    interface{}            `json:"location,omitempty"`
	Severity    string                 `json:"severity"`    // info, warning, critical
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Source      string                 `json:"source"`      // 로그 생성 소스
}

// AuditFilter는 감사 로그 필터입니다.
type AuditFilter struct {
	UserID     string `json:"user_id,omitempty"`
	Category   string `json:"category,omitempty"`
	Action     string `json:"action,omitempty"`
	Severity   string `json:"severity,omitempty"`
	IPAddress  string `json:"ip_address,omitempty"`
	StartDate  string `json:"start_date,omitempty"` // YYYY-MM-DD 형식
	EndDate    string `json:"end_date,omitempty"`
	Limit      int    `json:"limit"`
}

// generateAuditID는 고유한 감사 로그 ID를 생성합니다.
func generateAuditID() string {
	return fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), time.Now().UnixMilli()%1000)
}