package boltdb

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// Serializer는 BoltDB용 직렬화 인터페이스 구현입니다
type Serializer struct {
	// JSON 직렬화를 기본으로 사용
}

// NewSerializer 새 직렬화 도구 생성
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Marshal 객체를 바이트로 직렬화
func (s *Serializer) Marshal(v interface{}) ([]byte, error) {
	return MarshalJSON(v)
}

// Unmarshal 바이트를 객체로 역직렬화
func (s *Serializer) Unmarshal(data []byte, v interface{}) error {
	return UnmarshalJSON(data, v)
}

// MarshalProject 프로젝트를 직렬화
func (s *Serializer) MarshalProject(project *models.Project) ([]byte, error) {
	return s.Marshal(project)
}

// UnmarshalProject 프로젝트를 역직렬화
func (s *Serializer) UnmarshalProject(data []byte, project *models.Project) error {
	return s.Unmarshal(data, project)
}

// MarshalSession 세션을 직렬화
func (s *Serializer) MarshalSession(session *models.Session) ([]byte, error) {
	return s.Marshal(session)
}

// UnmarshalSession 세션을 역직렬화
func (s *Serializer) UnmarshalSession(data []byte, session *models.Session) error {
	return s.Unmarshal(data, session)
}

// MarshalTask 태스크를 직렬화
func (s *Serializer) MarshalTask(task *models.Task) ([]byte, error) {
	return s.Marshal(task)
}

// UnmarshalTask 태스크를 역직렬화  
func (s *Serializer) UnmarshalTask(data []byte, task *models.Task) error {
	return s.Unmarshal(data, task)
}

// SerializationError 직렬화 에러
type SerializationError struct {
	Type string
	Err  error
}

func (e *SerializationError) Error() string {
	return fmt.Sprintf("serialization error for type %s: %v", e.Type, e.Err)
}

// MarshalJSON 객체를 JSON 바이트로 직렬화
func MarshalJSON(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, &SerializationError{
			Type: fmt.Sprintf("%T", v),
			Err:  err,
		}
	}
	return data, nil
}

// UnmarshalJSON JSON 바이트를 객체로 역직렬화
func UnmarshalJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	
	err := json.Unmarshal(data, v)
	if err != nil {
		return &SerializationError{
			Type: fmt.Sprintf("%T", v),
			Err:  err,
		}
	}
	return nil
}

// WorkspaceSerializer 워크스페이스 직렬화 헬퍼
type WorkspaceSerializer struct{}

// Marshal 워크스페이스를 JSON으로 직렬화
func (ws *WorkspaceSerializer) Marshal(workspace *models.Workspace) ([]byte, error) {
	if workspace == nil {
		return nil, fmt.Errorf("workspace is nil")
	}
	
	// 민감한 정보 마스킹을 위한 복사본 생성
	wsClone := *workspace
	if wsClone.ClaudeKey != "" {
		// API 키는 암호화된 형태로 저장해야 하지만, 여기서는 단순 마스킹
		wsClone.ClaudeKey = maskAPIKey(wsClone.ClaudeKey)
	}
	
	return MarshalJSON(wsClone)
}

// Unmarshal JSON을 워크스페이스로 역직렬화
func (ws *WorkspaceSerializer) Unmarshal(data []byte) (*models.Workspace, error) {
	var workspace models.Workspace
	err := UnmarshalJSON(data, &workspace)
	if err != nil {
		return nil, err
	}
	return &workspace, nil
}

// ProjectSerializer 프로젝트 직렬화 헬퍼
type ProjectSerializer struct{}

// Marshal 프로젝트를 JSON으로 직렬화
func (ps *ProjectSerializer) Marshal(project *models.Project) ([]byte, error) {
	if project == nil {
		return nil, fmt.Errorf("project is nil")
	}
	
	// 민감한 정보 처리
	projectClone := *project
	if projectClone.Config.ClaudeAPIKey != "" {
		projectClone.Config.ClaudeAPIKey = ""
		projectClone.Config.EncryptedAPIKey = maskAPIKey(project.Config.ClaudeAPIKey)
	}
	
	return MarshalJSON(projectClone)
}

// Unmarshal JSON을 프로젝트로 역직렬화
func (ps *ProjectSerializer) Unmarshal(data []byte) (*models.Project, error) {
	var project models.Project
	err := UnmarshalJSON(data, &project)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// SessionSerializer 세션 직렬화 헬퍼  
type SessionSerializer struct{}

// Marshal 세션을 JSON으로 직렬화
func (ss *SessionSerializer) Marshal(session *models.Session) ([]byte, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}
	
	// 세션 데이터는 별도의 마스킹이 필요하지 않음
	return MarshalJSON(session)
}

// Unmarshal JSON을 세션으로 역직렬화
func (ss *SessionSerializer) Unmarshal(data []byte) (*models.Session, error) {
	var session models.Session
	err := UnmarshalJSON(data, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// TaskSerializer 태스크 직렬화 헬퍼
type TaskSerializer struct{}

// Marshal 태스크를 JSON으로 직렬화
func (ts *TaskSerializer) Marshal(task *models.Task) ([]byte, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	
	return MarshalJSON(task)
}

// Unmarshal JSON을 태스크로 역직렬화
func (ts *TaskSerializer) Unmarshal(data []byte) (*models.Task, error) {
	var task models.Task
	err := UnmarshalJSON(data, &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// IndexEntry 인덱스 엔트리 구조
type IndexEntry struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// IndexSerializer 인덱스 직렬화 헬퍼
type IndexSerializer struct{}

// MarshalStringList 문자열 리스트를 JSON으로 직렬화
func (is *IndexSerializer) MarshalStringList(list []string) ([]byte, error) {
	if list == nil {
		list = []string{}
	}
	return MarshalJSON(list)
}

// UnmarshalStringList JSON을 문자열 리스트로 역직렬화
func (is *IndexSerializer) UnmarshalStringList(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}
	
	var list []string
	err := UnmarshalJSON(data, &list)
	if err != nil {
		return nil, err
	}
	
	if list == nil {
		list = []string{}
	}
	
	return list, nil
}

// MarshalIndexEntry 인덱스 엔트리를 JSON으로 직렬화
func (is *IndexSerializer) MarshalIndexEntry(entry *IndexEntry) ([]byte, error) {
	if entry == nil {
		return nil, fmt.Errorf("index entry is nil")
	}
	
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	
	return MarshalJSON(entry)
}

// UnmarshalIndexEntry JSON을 인덱스 엔트리로 역직렬화
func (is *IndexSerializer) UnmarshalIndexEntry(data []byte) (*IndexEntry, error) {
	var entry IndexEntry
	err := UnmarshalJSON(data, &entry)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// PaginationData 페이지네이션 데이터
type PaginationData struct {
	Items      []json.RawMessage `json:"items"`
	TotalCount int               `json:"total_count"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	HasMore    bool              `json:"has_more"`
}

// MarshalPagination 페이지네이션 데이터를 JSON으로 직렬화
func MarshalPagination(data *PaginationData) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("pagination data is nil")
	}
	return MarshalJSON(data)
}

// UnmarshalPagination JSON을 페이지네이션 데이터로 역직렬화
func UnmarshalPagination(data []byte) (*PaginationData, error) {
	var pd PaginationData
	err := UnmarshalJSON(data, &pd)
	if err != nil {
		return nil, err
	}
	return &pd, nil
}

// 유틸리티 함수들

// maskAPIKey API 키를 마스킹 처리
func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	
	if len(apiKey) <= 10 {
		return "***"
	}
	
	return apiKey[:10] + "..."
}

// GenerateKey 키 생성 헬퍼
func GenerateKey(prefix, id string) string {
	if prefix == "" {
		return id
	}
	return prefix + ":" + id
}

// ParseKey 키 파싱 헬퍼  
func ParseKey(key string) (prefix, id string) {
	for i, r := range key {
		if r == ':' {
			return key[:i], key[i+1:]
		}
	}
	return "", key
}

// ValidateRequiredFields 필수 필드 검증
func ValidateRequiredFields(entity interface{}) error {
	switch v := entity.(type) {
	case *models.Workspace:
		if v.ID == "" {
			return fmt.Errorf("workspace ID is required")
		}
		if v.Name == "" {
			return fmt.Errorf("workspace name is required")
		}
		if v.OwnerID == "" {
			return fmt.Errorf("workspace owner ID is required")
		}
	case *models.Project:
		if v.ID == "" {
			return fmt.Errorf("project ID is required")
		}
		if v.Name == "" {
			return fmt.Errorf("project name is required")
		}
		if v.WorkspaceID == "" {
			return fmt.Errorf("project workspace ID is required")
		}
	case *models.Session:
		if v.ID == "" {
			return fmt.Errorf("session ID is required")
		}
		if v.ProjectID == "" {
			return fmt.Errorf("session project ID is required")
		}
	case *models.Task:
		if v.ID == "" {
			return fmt.Errorf("task ID is required")
		}
		if v.SessionID == "" {
			return fmt.Errorf("task session ID is required")
		}
		if v.Command == "" {
			return fmt.Errorf("task command is required")
		}
	default:
		return fmt.Errorf("unknown entity type: %T", entity)
	}
	
	return nil
}

// NormalizeTimestamps 타임스탬프 정규화
func NormalizeTimestamps(entity interface{}) {
	now := time.Now()
	
	switch v := entity.(type) {
	case *models.Workspace:
		if v.CreatedAt.IsZero() {
			v.CreatedAt = now
		}
		if v.UpdatedAt.IsZero() {
			v.UpdatedAt = now
		}
	case *models.Project:
		if v.CreatedAt.IsZero() {
			v.CreatedAt = now
		}
		if v.UpdatedAt.IsZero() {
			v.UpdatedAt = now
		}
	case *models.Session:
		if v.CreatedAt.IsZero() {
			v.CreatedAt = now
		}
		if v.UpdatedAt.IsZero() {
			v.UpdatedAt = now
		}
		if v.LastActive.IsZero() {
			v.LastActive = now
		}
	case *models.Task:
		if v.CreatedAt.IsZero() {
			v.CreatedAt = now
		}
		if v.UpdatedAt.IsZero() {
			v.UpdatedAt = now
		}
	}
}