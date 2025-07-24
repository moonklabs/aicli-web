package memory

import (
	"context"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage"
)

// SessionStorage 메모리 기반 세션 스토리지 구현
type SessionStorage struct {
	mu       sync.RWMutex
	sessions map[string]*models.Session
	order    []string // 생성 순서 추적
}

// storage.SessionStorage 인터페이스 구현 확인
var _ storage.SessionStorage = (*SessionStorage)(nil)

// NewSessionStorage 새 세션 스토리지 생성
func NewSessionStorage() *SessionStorage {
	return &SessionStorage{
		sessions: make(map[string]*models.Session),
		order:    make([]string, 0),
	}
}

// Create 새 세션 생성
func (s *SessionStorage) Create(ctx context.Context, session *models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.ID]; exists {
		return ErrAlreadyExists
	}

	// 생성 시간 설정
	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	// 깊은 복사
	sessionCopy := *session
	s.sessions[session.ID] = &sessionCopy
	s.order = append(s.order, session.ID)

	return nil
}

// GetByID ID로 세션 조회
func (s *SessionStorage) GetByID(ctx context.Context, id string) (*models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[id]
	if !exists {
		return nil, ErrNotFound
	}

	// 깊은 복사하여 반환
	sessionCopy := *session
	return &sessionCopy, nil
}

// List 세션 목록 조회
func (s *SessionStorage) List(ctx context.Context, filter *models.SessionFilter, paging *models.PaginationRequest) (*models.PaginationResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 필터링된 세션 수집
	filtered := make([]*models.Session, 0)
	
	for i := len(s.order) - 1; i >= 0; i-- { // 최신 순으로
		id := s.order[i]
		session := s.sessions[id]
		
		// 필터 적용
		if filter != nil {
			if filter.ProjectID != "" && session.ProjectID != filter.ProjectID {
				continue
			}
			if filter.Status != "" && session.Status != filter.Status {
				continue
			}
			if filter.Active != nil {
				isActive := session.IsActive()
				if *filter.Active != isActive {
					continue
				}
			}
		}
		
		// 깊은 복사
		sessionCopy := *session
		filtered = append(filtered, &sessionCopy)
	}

	// 페이징 적용
	total := len(filtered)
	start := (paging.Page - 1) * paging.Limit
	end := start + paging.Limit
	
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	
	items := filtered[start:end]
	
	return &models.PaginationResponse{
		Data: items,
		Meta: models.NewPaginationMeta(paging.Page, paging.Limit, total),
	}, nil
}

// Update 세션 업데이트
func (s *SessionStorage) Update(ctx context.Context, session *models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.ID]; !exists {
		return ErrNotFound
	}

	// 업데이트 시간 설정
	session.UpdatedAt = time.Now()

	// 깊은 복사
	sessionCopy := *session
	s.sessions[session.ID] = &sessionCopy

	return nil
}

// Delete 세션 삭제
func (s *SessionStorage) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[id]; !exists {
		return ErrNotFound
	}

	delete(s.sessions, id)
	
	// order에서도 제거
	newOrder := make([]string, 0, len(s.order)-1)
	for _, oid := range s.order {
		if oid != id {
			newOrder = append(newOrder, oid)
		}
	}
	s.order = newOrder

	return nil
}

// GetActiveCount 활성 세션 수 조회
func (s *SessionStorage) GetActiveCount(ctx context.Context, projectID string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	for _, session := range s.sessions {
		if (projectID == "" || session.ProjectID == projectID) && session.IsActive() {
			count++
		}
	}

	return count, nil
}