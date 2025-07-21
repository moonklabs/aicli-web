package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SessionService 세션 관리 서비스
type SessionService struct {
	storage        storage.Storage
	projectService *ProjectService
	logger         *zap.Logger
	
	// 동시성 제어
	mu             sync.RWMutex
	activeSessions map[string]*models.Session
	
	// 설정
	maxConcurrent  int
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
}

// SessionServiceConfig 세션 서비스 설정
type SessionServiceConfig struct {
	MaxConcurrent  int
	CleanupInterval time.Duration
}

// DefaultSessionServiceConfig 기본 세션 서비스 설정
func DefaultSessionServiceConfig() *SessionServiceConfig {
	return &SessionServiceConfig{
		MaxConcurrent:   10,
		CleanupInterval: 5 * time.Minute,
	}
}

// NewSessionService 새로운 세션 서비스 생성
func NewSessionService(storage storage.Storage, projectService *ProjectService, config *SessionServiceConfig) *SessionService {
	if config == nil {
		config = DefaultSessionServiceConfig()
	}
	
	s := &SessionService{
		storage:        storage,
		projectService: projectService,
		logger:         logger.Get(),
		activeSessions: make(map[string]*models.Session),
		maxConcurrent:  config.MaxConcurrent,
		cleanupTicker:  time.NewTicker(config.CleanupInterval),
		stopCleanup:    make(chan struct{}),
	}
	
	// 정리 고루틴 시작
	go s.cleanupRoutine()
	
	return s
}

// Create 새로운 세션 생성
func (s *SessionService) Create(ctx context.Context, req *models.SessionCreateRequest) (*models.Session, error) {
	// 프로젝트 확인
	project, err := s.projectService.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("프로젝트 조회 실패: %w", err)
	}
	
	// 동시 세션 수 확인
	if err := s.checkConcurrentLimit(); err != nil {
		return nil, err
	}
	
	// 세션 생성
	now := time.Now()
	session := &models.Session{
		BaseModel: models.BaseModel{
			ID: uuid.New().String(),
		},
		ProjectID:  project.ID,
		Status:     models.SessionPending,
		LastActive: now,
		Metadata:   req.Metadata,
	}
	
	if req.MaxIdleTime != nil {
		session.MaxIdleTime = *req.MaxIdleTime
	}
	if req.MaxLifetime != nil {
		session.MaxLifetime = *req.MaxLifetime
	}
	
	// 저장
	if err := s.storage.Session().Create(ctx, session); err != nil {
		return nil, fmt.Errorf("세션 생성 실패: %w", err)
	}
	
	// 활성 세션 추가
	s.mu.Lock()
	s.activeSessions[session.ID] = session
	s.mu.Unlock()
	
	s.logger.Info("세션 생성됨",
		zap.String("session_id", session.ID),
		zap.String("project_id", project.ID),
	)
	
	return session, nil
}

// GetByID ID로 세션 조회
func (s *SessionService) GetByID(ctx context.Context, id string) (*models.Session, error) {
	// 먼저 활성 세션에서 확인
	s.mu.RLock()
	if session, ok := s.activeSessions[id]; ok {
		s.mu.RUnlock()
		return session, nil
	}
	s.mu.RUnlock()
	
	// 저장소에서 조회
	session, err := s.storage.Session().GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	return session, nil
}

// List 세션 목록 조회
func (s *SessionService) List(ctx context.Context, filter *models.SessionFilter, paging *models.PagingRequest) (*models.PagingResponse[*models.Session], error) {
	return s.storage.Session().List(ctx, filter, paging)
}

// UpdateStatus 세션 상태 업데이트
func (s *SessionService) UpdateStatus(ctx context.Context, id string, status models.SessionStatus) error {
	session, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	
	// 상태 전이 검증
	if err := s.validateStatusTransition(session.Status, status); err != nil {
		return err
	}
	
	session.Status = status
	
	// 특정 상태 처리
	switch status {
	case models.SessionActive:
		if session.StartedAt == nil {
			now := time.Now()
			session.StartedAt = &now
		}
		session.UpdateActivity()
	case models.SessionEnded, models.SessionError:
		now := time.Now()
		session.EndedAt = &now
		// 활성 세션에서 제거
		s.mu.Lock()
		delete(s.activeSessions, id)
		s.mu.Unlock()
	}
	
	// 저장
	if err := s.storage.Session().Update(ctx, session); err != nil {
		return fmt.Errorf("세션 상태 업데이트 실패: %w", err)
	}
	
	s.logger.Info("세션 상태 업데이트",
		zap.String("session_id", id),
		zap.String("status", string(status)),
	)
	
	return nil
}

// UpdateActivity 세션 활동 업데이트
func (s *SessionService) UpdateActivity(ctx context.Context, id string) error {
	session, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	
	session.UpdateActivity()
	
	// Idle 상태인 경우 Active로 변경
	if session.Status == models.SessionIdle {
		session.Status = models.SessionActive
	}
	
	return s.storage.Session().Update(ctx, session)
}

// Terminate 세션 종료
func (s *SessionService) Terminate(ctx context.Context, id string) error {
	return s.UpdateStatus(ctx, id, models.SessionEnding)
}

// UpdateStats 세션 통계 업데이트
func (s *SessionService) UpdateStats(ctx context.Context, id string, commandDelta, bytesInDelta, bytesOutDelta, errorDelta int64) error {
	session, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	
	session.CommandCount += commandDelta
	session.BytesIn += bytesInDelta
	session.BytesOut += bytesOutDelta
	session.ErrorCount += errorDelta
	session.UpdateActivity()
	
	return s.storage.Session().Update(ctx, session)
}

// GetActiveSessions 활성 세션 목록 조회
func (s *SessionService) GetActiveSessions() []*models.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	sessions := make([]*models.Session, 0, len(s.activeSessions))
	for _, session := range s.activeSessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// checkConcurrentLimit 동시 세션 수 제한 확인
func (s *SessionService) checkConcurrentLimit() error {
	s.mu.RLock()
	activeCount := len(s.activeSessions)
	s.mu.RUnlock()
	
	if activeCount >= s.maxConcurrent {
		return fmt.Errorf("최대 동시 세션 수(%d) 초과", s.maxConcurrent)
	}
	
	return nil
}

// validateStatusTransition 상태 전이 검증
func (s *SessionService) validateStatusTransition(from, to models.SessionStatus) error {
	validTransitions := map[models.SessionStatus][]models.SessionStatus{
		models.SessionPending: {models.SessionActive, models.SessionError},
		models.SessionActive:  {models.SessionIdle, models.SessionEnding, models.SessionError},
		models.SessionIdle:    {models.SessionActive, models.SessionEnding, models.SessionError},
		models.SessionEnding:  {models.SessionEnded, models.SessionError},
		models.SessionError:   {models.SessionEnded},
		models.SessionEnded:   {},
	}
	
	allowedTransitions, ok := validTransitions[from]
	if !ok {
		return fmt.Errorf("알 수 없는 상태: %s", from)
	}
	
	for _, allowed := range allowedTransitions {
		if allowed == to {
			return nil
		}
	}
	
	return fmt.Errorf("잘못된 상태 전이: %s -> %s", from, to)
}

// cleanupRoutine 정리 고루틴
func (s *SessionService) cleanupRoutine() {
	for {
		select {
		case <-s.cleanupTicker.C:
			s.cleanupSessions()
		case <-s.stopCleanup:
			return
		}
	}
}

// cleanupSessions 타임아웃된 세션 정리
func (s *SessionService) cleanupSessions() {
	ctx := context.Background()
	
	s.mu.RLock()
	sessions := make([]*models.Session, 0, len(s.activeSessions))
	for _, session := range s.activeSessions {
		sessions = append(sessions, session)
	}
	s.mu.RUnlock()
	
	for _, session := range sessions {
		// 타임아웃 확인
		if session.IsIdleTimeout() {
			s.logger.Info("유휴 타임아웃으로 세션 종료",
				zap.String("session_id", session.ID),
			)
			if err := s.Terminate(ctx, session.ID); err != nil {
				s.logger.Error("세션 종료 실패",
					zap.String("session_id", session.ID),
					zap.Error(err),
				)
			}
		} else if session.IsLifetimeTimeout() {
			s.logger.Info("생명주기 타임아웃으로 세션 종료",
				zap.String("session_id", session.ID),
			)
			if err := s.Terminate(ctx, session.ID); err != nil {
				s.logger.Error("세션 종료 실패",
					zap.String("session_id", session.ID),
					zap.Error(err),
				)
			}
		} else if session.Status == models.SessionActive && time.Since(session.LastActive) > 5*time.Minute {
			// 5분 이상 활동이 없으면 Idle 상태로 변경
			if err := s.UpdateStatus(ctx, session.ID, models.SessionIdle); err != nil {
				s.logger.Error("세션 상태 업데이트 실패",
					zap.String("session_id", session.ID),
					zap.Error(err),
				)
			}
		}
	}
}

// Stop 서비스 중지
func (s *SessionService) Stop() {
	s.cleanupTicker.Stop()
	close(s.stopCleanup)
}