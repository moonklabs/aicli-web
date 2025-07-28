package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

// ErrorTracker는 에러 추적 및 관리 시스템입니다
type ErrorTracker struct {
	errors        map[string]*ErrorInfo
	errorCounts   map[string]int64
	mutex         sync.RWMutex
	config        ErrorTrackerConfig
	listeners     []ErrorListener
	listenerMutex sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// ErrorTrackerConfig는 에러 트래커 설정입니다
type ErrorTrackerConfig struct {
	MaxErrors         int           `json:"max_errors"`
	RetentionPeriod   time.Duration `json:"retention_period"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	EnableStackTrace  bool          `json:"enable_stack_trace"`
	EnableGrouping    bool          `json:"enable_grouping"`
	AlertThreshold    int           `json:"alert_threshold"`
	AlertInterval     time.Duration `json:"alert_interval"`
}

// ErrorInfo는 에러 정보를 저장합니다
type ErrorInfo struct {
	ID            string            `json:"id"`
	Message       string            `json:"message"`
	Type          string            `json:"type"`
	StackTrace    string            `json:"stack_trace,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
	FirstOccurred time.Time         `json:"first_occurred"`
	LastOccurred  time.Time         `json:"last_occurred"`
	Count         int64             `json:"count"`
	Severity      ErrorSeverity     `json:"severity"`
	Status        ErrorStatus       `json:"status"`
	Tags          []string          `json:"tags,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// ErrorSeverity는 AlertSeverity를 사용합니다
type ErrorSeverity = AlertSeverity

// ErrorStatus는 에러 상태를 나타냅니다
type ErrorStatus string

const (
	ErrorStatusNew      ErrorStatus = "new"
	ErrorStatusOpen     ErrorStatus = "open"
	ErrorStatusResolved ErrorStatus = "resolved"
	ErrorStatusIgnored  ErrorStatus = "ignored"
)

// ErrorEvent는 에러 이벤트입니다
type ErrorEvent struct {
	ErrorInfo *ErrorInfo           `json:"error_info"`
	OccurredAt time.Time           `json:"occurred_at"`
	Context   map[string]interface{} `json:"context,omitempty"`
	UserID    string               `json:"user_id,omitempty"`
	SessionID string               `json:"session_id,omitempty"`
	RequestID string               `json:"request_id,omitempty"`
}

// ErrorListener는 에러 이벤트 리스너 인터페이스입니다
type ErrorListener interface {
	OnError(event ErrorEvent)
	OnErrorResolved(errorID string)
	OnThresholdExceeded(errorID string, count int64)
}

// ErrorSummary는 에러 요약 정보입니다
type ErrorSummary struct {
	TotalErrors     int64                    `json:"total_errors"`
	UniqueErrors    int                      `json:"unique_errors"`
	ErrorsToday     int64                    `json:"errors_today"`
	TopErrors       []*ErrorInfo             `json:"top_errors"`
	SeverityBreakdown map[ErrorSeverity]int64 `json:"severity_breakdown"`
	TypeBreakdown   map[string]int64         `json:"type_breakdown"`
	GeneratedAt     time.Time                `json:"generated_at"`
}

// DefaultErrorTrackerConfig는 기본 에러 트래커 설정을 반환합니다
func DefaultErrorTrackerConfig() ErrorTrackerConfig {
	return ErrorTrackerConfig{
		MaxErrors:        1000,
		RetentionPeriod:  7 * 24 * time.Hour, // 7일
		CleanupInterval:  1 * time.Hour,
		EnableStackTrace: true,
		EnableGrouping:   true,
		AlertThreshold:   10,
		AlertInterval:    5 * time.Minute,
	}
}

// NewErrorTracker는 새로운 에러 트래커를 생성합니다
func NewErrorTracker(config ErrorTrackerConfig) *ErrorTracker {
	ctx, cancel := context.WithCancel(context.Background())

	tracker := &ErrorTracker{
		errors:      make(map[string]*ErrorInfo),
		errorCounts: make(map[string]int64),
		config:      config,
		listeners:   make([]ErrorListener, 0),
		ctx:         ctx,
		cancel:      cancel,
	}

	// 정리 작업 시작
	tracker.startCleanupWorker()

	return tracker
}

// TrackError는 에러를 추적합니다
func (et *ErrorTracker) TrackError(err error, context map[string]interface{}) string {
	return et.TrackErrorWithSeverity(err, context, SeverityWarning)
}

// TrackErrorWithSeverity는 심각도와 함께 에러를 추적합니다
func (et *ErrorTracker) TrackErrorWithSeverity(err error, context map[string]interface{}, severity ErrorSeverity) string {
	if err == nil {
		return ""
	}

	// 에러 ID 생성 (메시지 기반 해시)
	errorID := et.generateErrorID(err.Error())

	et.mutex.Lock()
	defer et.mutex.Unlock()

	now := time.Now()
	existingError, exists := et.errors[errorID]

	if exists {
		// 기존 에러 업데이트
		existingError.Count++
		existingError.LastOccurred = now
		if context != nil {
			existingError.Context = context
		}
	} else {
		// 새로운 에러 생성
		errorInfo := &ErrorInfo{
			ID:            errorID,
			Message:       err.Error(),
			Type:          et.getErrorType(err),
			FirstOccurred: now,
			LastOccurred:  now,
			Count:         1,
			Severity:      severity,
			Status:        ErrorStatusNew,
			Context:       context,
			Metadata:      make(map[string]string),
		}

		// 스택 트레이스 추가
		if et.config.EnableStackTrace {
			errorInfo.StackTrace = et.getStackTrace()
		}

		et.errors[errorID] = errorInfo
		existingError = errorInfo

		// 최대 에러 수 확인
		if len(et.errors) > et.config.MaxErrors {
			et.removeOldestError()
		}
	}

	// 전체 에러 카운트 업데이트
	et.errorCounts[errorID] = existingError.Count

	// 리스너들에게 알림
	et.notifyListeners(ErrorEvent{
		ErrorInfo:  existingError,
		OccurredAt: now,
		Context:    context,
	})

	// 임계값 초과 확인
	if existingError.Count%int64(et.config.AlertThreshold) == 0 {
		et.notifyThresholdExceeded(errorID, existingError.Count)
	}

	return errorID
}

// GetError는 에러 ID로 에러 정보를 가져옵니다
func (et *ErrorTracker) GetError(errorID string) (*ErrorInfo, bool) {
	et.mutex.RLock()
	defer et.mutex.RUnlock()

	errorInfo, exists := et.errors[errorID]
	if !exists {
		return nil, false
	}

	// 복사본 반환
	copy := *errorInfo
	return &copy, true
}

// GetAllErrors는 모든 에러를 반환합니다
func (et *ErrorTracker) GetAllErrors() []*ErrorInfo {
	et.mutex.RLock()
	defer et.mutex.RUnlock()

	errors := make([]*ErrorInfo, 0, len(et.errors))
	for _, errorInfo := range et.errors {
		copy := *errorInfo
		errors = append(errors, &copy)
	}

	// 최근 발생 순으로 정렬
	for i := 0; i < len(errors)-1; i++ {
		for j := i + 1; j < len(errors); j++ {
			if errors[i].LastOccurred.Before(errors[j].LastOccurred) {
				errors[i], errors[j] = errors[j], errors[i]
			}
		}
	}

	return errors
}

// GetTopErrors는 가장 빈번한 에러들을 반환합니다
func (et *ErrorTracker) GetTopErrors(limit int) []*ErrorInfo {
	et.mutex.RLock()
	defer et.mutex.RUnlock()

	errors := make([]*ErrorInfo, 0, len(et.errors))
	for _, errorInfo := range et.errors {
		copy := *errorInfo
		errors = append(errors, &copy)
	}

	// 발생 빈도순으로 정렬
	for i := 0; i < len(errors)-1; i++ {
		for j := i + 1; j < len(errors); j++ {
			if errors[i].Count < errors[j].Count {
				errors[i], errors[j] = errors[j], errors[i]
			}
		}
	}

	if limit > 0 && len(errors) > limit {
		errors = errors[:limit]
	}

	return errors
}

// ResolveError는 에러를 해결됨으로 표시합니다
func (et *ErrorTracker) ResolveError(errorID string) error {
	et.mutex.Lock()
	defer et.mutex.Unlock()

	errorInfo, exists := et.errors[errorID]
	if !exists {
		return fmt.Errorf("error not found: %s", errorID)
	}

	errorInfo.Status = ErrorStatusResolved

	// 리스너들에게 알림
	et.notifyErrorResolved(errorID)

	return nil
}

// IgnoreError는 에러를 무시됨으로 표시합니다
func (et *ErrorTracker) IgnoreError(errorID string) error {
	et.mutex.Lock()
	defer et.mutex.Unlock()

	errorInfo, exists := et.errors[errorID]
	if !exists {
		return fmt.Errorf("error not found: %s", errorID)
	}

	errorInfo.Status = ErrorStatusIgnored
	return nil
}

// AddTag는 에러에 태그를 추가합니다
func (et *ErrorTracker) AddTag(errorID string, tag string) error {
	et.mutex.Lock()
	defer et.mutex.Unlock()

	errorInfo, exists := et.errors[errorID]
	if !exists {
		return fmt.Errorf("error not found: %s", errorID)
	}

	// 중복 태그 확인
	for _, existingTag := range errorInfo.Tags {
		if existingTag == tag {
			return nil // 이미 존재
		}
	}

	errorInfo.Tags = append(errorInfo.Tags, tag)
	return nil
}

// GetSummary는 에러 요약을 반환합니다
func (et *ErrorTracker) GetSummary() *ErrorSummary {
	et.mutex.RLock()
	defer et.mutex.RUnlock()

	summary := &ErrorSummary{
		UniqueErrors:      len(et.errors),
		SeverityBreakdown: make(map[ErrorSeverity]int64),
		TypeBreakdown:     make(map[string]int64),
		GeneratedAt:       time.Now(),
	}

	today := time.Now().Truncate(24 * time.Hour)
	totalErrors := int64(0)
	errorsToday := int64(0)

	for _, errorInfo := range et.errors {
		totalErrors += errorInfo.Count
		summary.SeverityBreakdown[errorInfo.Severity] += errorInfo.Count
		summary.TypeBreakdown[errorInfo.Type] += errorInfo.Count

		// 오늘 발생한 에러 카운트
		if errorInfo.LastOccurred.After(today) {
			errorsToday += errorInfo.Count
		}
	}

	summary.TotalErrors = totalErrors
	summary.ErrorsToday = errorsToday
	summary.TopErrors = et.getTopErrorsInternal(5)

	return summary
}

// AddListener는 에러 이벤트 리스너를 추가합니다
func (et *ErrorTracker) AddListener(listener ErrorListener) {
	et.listenerMutex.Lock()
	defer et.listenerMutex.Unlock()

	et.listeners = append(et.listeners, listener)
}

// RemoveListener는 에러 이벤트 리스너를 제거합니다
func (et *ErrorTracker) RemoveListener(listener ErrorListener) {
	et.listenerMutex.Lock()
	defer et.listenerMutex.Unlock()

	for i, l := range et.listeners {
		if l == listener {
			et.listeners = append(et.listeners[:i], et.listeners[i+1:]...)
			break
		}
	}
}

// ClearErrors는 모든 에러를 지웁니다
func (et *ErrorTracker) ClearErrors() {
	et.mutex.Lock()
	defer et.mutex.Unlock()

	et.errors = make(map[string]*ErrorInfo)
	et.errorCounts = make(map[string]int64)
}

// Stop은 에러 트래커를 중지합니다
func (et *ErrorTracker) Stop() {
	et.cancel()
	et.wg.Wait()
}

// 내부 메서드들

func (et *ErrorTracker) generateErrorID(message string) string {
	// 간단한 해시 기반 ID 생성
	// 실제 구현에서는 더 강력한 해시 알고리즘 사용
	hash := 0
	for _, char := range message {
		hash = hash*31 + int(char)
	}
	return fmt.Sprintf("err_%d", hash)
}

func (et *ErrorTracker) getErrorType(err error) string {
	// 에러 타입 추론
	errorType := fmt.Sprintf("%T", err)
	if errorType == "*errors.errorString" {
		return "generic"
	}
	return errorType
}

func (et *ErrorTracker) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func (et *ErrorTracker) removeOldestError() {
	var oldestID string
	var oldestTime time.Time

	for id, errorInfo := range et.errors {
		if oldestID == "" || errorInfo.FirstOccurred.Before(oldestTime) {
			oldestID = id
			oldestTime = errorInfo.FirstOccurred
		}
	}

	if oldestID != "" {
		delete(et.errors, oldestID)
		delete(et.errorCounts, oldestID)
	}
}

func (et *ErrorTracker) getTopErrorsInternal(limit int) []*ErrorInfo {
	errors := make([]*ErrorInfo, 0, len(et.errors))
	for _, errorInfo := range et.errors {
		copy := *errorInfo
		errors = append(errors, &copy)
	}

	// 발생 빈도순으로 정렬
	for i := 0; i < len(errors)-1; i++ {
		for j := i + 1; j < len(errors); j++ {
			if errors[i].Count < errors[j].Count {
				errors[i], errors[j] = errors[j], errors[i]
			}
		}
	}

	if limit > 0 && len(errors) > limit {
		errors = errors[:limit]
	}

	return errors
}

func (et *ErrorTracker) startCleanupWorker() {
	et.wg.Add(1)
	go func() {
		defer et.wg.Done()

		ticker := time.NewTicker(et.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-et.ctx.Done():
				return
			case <-ticker.C:
				et.cleanupOldErrors()
			}
		}
	}()
}

func (et *ErrorTracker) cleanupOldErrors() {
	et.mutex.Lock()
	defer et.mutex.Unlock()

	cutoff := time.Now().Add(-et.config.RetentionPeriod)
	errorsToRemove := make([]string, 0)

	for id, errorInfo := range et.errors {
		if errorInfo.LastOccurred.Before(cutoff) {
			errorsToRemove = append(errorsToRemove, id)
		}
	}

	for _, id := range errorsToRemove {
		delete(et.errors, id)
		delete(et.errorCounts, id)
	}

	if len(errorsToRemove) > 0 {
		log.Printf("Cleaned up %d old errors", len(errorsToRemove))
	}
}

func (et *ErrorTracker) notifyListeners(event ErrorEvent) {
	et.listenerMutex.RLock()
	listeners := make([]ErrorListener, len(et.listeners))
	copy(listeners, et.listeners)
	et.listenerMutex.RUnlock()

	for _, listener := range listeners {
		go func(l ErrorListener) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Error listener panic: %v", r)
				}
			}()
			l.OnError(event)
		}(listener)
	}
}

func (et *ErrorTracker) notifyErrorResolved(errorID string) {
	et.listenerMutex.RLock()
	listeners := make([]ErrorListener, len(et.listeners))
	copy(listeners, et.listeners)
	et.listenerMutex.RUnlock()

	for _, listener := range listeners {
		go func(l ErrorListener) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Error listener panic: %v", r)
				}
			}()
			l.OnErrorResolved(errorID)
		}(listener)
	}
}

func (et *ErrorTracker) notifyThresholdExceeded(errorID string, count int64) {
	et.listenerMutex.RLock()
	listeners := make([]ErrorListener, len(et.listeners))
	copy(listeners, et.listeners)
	et.listenerMutex.RUnlock()

	for _, listener := range listeners {
		go func(l ErrorListener) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Error listener panic: %v", r)
				}
			}()
			l.OnThresholdExceeded(errorID, count)
		}(listener)
	}
}

// 유틸리티 함수들

// ExportErrors는 에러 데이터를 JSON으로 내보냅니다
func (et *ErrorTracker) ExportErrors() ([]byte, error) {
	et.mutex.RLock()
	defer et.mutex.RUnlock()

	exportData := struct {
		Errors      map[string]*ErrorInfo `json:"errors"`
		Summary     *ErrorSummary         `json:"summary"`
		ExportedAt  time.Time             `json:"exported_at"`
	}{
		Errors:     et.errors,
		Summary:    et.GetSummary(),
		ExportedAt: time.Now(),
	}

	return json.MarshalIndent(exportData, "", "  ")
}

// ImportErrors는 JSON에서 에러 데이터를 가져옵니다
func (et *ErrorTracker) ImportErrors(data []byte) error {
	var importData struct {
		Errors map[string]*ErrorInfo `json:"errors"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("failed to unmarshal error data: %w", err)
	}

	et.mutex.Lock()
	defer et.mutex.Unlock()

	// 기존 데이터와 병합
	for id, errorInfo := range importData.Errors {
		existing, exists := et.errors[id]
		if exists {
			// 기존 에러와 병합
			existing.Count += errorInfo.Count
			if errorInfo.LastOccurred.After(existing.LastOccurred) {
				existing.LastOccurred = errorInfo.LastOccurred
			}
			if errorInfo.FirstOccurred.Before(existing.FirstOccurred) {
				existing.FirstOccurred = errorInfo.FirstOccurred
			}
		} else {
			et.errors[id] = errorInfo
		}
		et.errorCounts[id] = et.errors[id].Count
	}

	return nil
}
