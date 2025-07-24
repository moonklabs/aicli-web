package boltdb

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"go.etcd.io/bbolt"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/interfaces"
)

// QueryHelper 쿼리 헬퍼
type QueryHelper struct {
	storage     *Storage
	indexMgr    *IndexManager
	serializers *Serializers
}

// Serializers 모든 직렬화기들
type Serializers struct {
	Workspace *WorkspaceSerializer
	Project   *ProjectSerializer
	Session   *SessionSerializer
	Task      *TaskSerializer
	Index     *IndexSerializer
}

// newQueryHelper 새 쿼리 헬퍼 생성
func newQueryHelper(storage *Storage, indexMgr *IndexManager) *QueryHelper {
	return &QueryHelper{
		storage:  storage,
		indexMgr: indexMgr,
		serializers: &Serializers{
			Workspace: &WorkspaceSerializer{},
			Project:   &ProjectSerializer{},
			Session:   &SessionSerializer{},
			Task:      &TaskSerializer{},
			Index:     &IndexSerializer{},
		},
	}
}

// SortOrder 정렬 순서
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// QueryOptions 쿼리 옵션
type QueryOptions struct {
	Page      int       // 페이지 번호 (1부터 시작)
	Limit     int       // 페이지당 항목 수
	Sort      string    // 정렬 필드
	Order     SortOrder // 정렬 순서
	Filter    map[string]interface{} // 필터 조건
	StartKey  string    // 커서 시작점
}

// Normalize 쿼리 옵션 정규화
func (qo *QueryOptions) Normalize() {
	if qo.Page < 1 {
		qo.Page = 1
	}
	if qo.Limit < 1 {
		qo.Limit = 20
	}
	if qo.Limit > 100 {
		qo.Limit = 100
	}
	if qo.Sort == "" {
		qo.Sort = "created_at"
	}
	if qo.Order != SortOrderAsc && qo.Order != SortOrderDesc {
		qo.Order = SortOrderDesc
	}
	if qo.Filter == nil {
		qo.Filter = make(map[string]interface{})
	}
}

// GetOffset 오프셋 계산
func (qo *QueryOptions) GetOffset() int {
	return (qo.Page - 1) * qo.Limit
}

// QueryResult 쿼리 결과
type QueryResult[T any] struct {
	Items      []T    `json:"items"`
	TotalCount int    `json:"total_count"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// WorkspaceQuery 워크스페이스 쿼리
func (qh *QueryHelper) WorkspaceQuery(tx *bbolt.Tx, options *QueryOptions) (*QueryResult[*models.Workspace], error) {
	if options == nil {
		options = &QueryOptions{}
	}
	options.Normalize()
	
	bucket := tx.Bucket([]byte(BucketWorkspaces))
	if bucket == nil {
		return nil, fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
	}
	
	// 필터 조건 확인
	var targetKeys []string
	if ownerID, ok := options.Filter["owner_id"].(string); ok && ownerID != "" {
		// 소유자 ID 기반 필터링
		keys, err := qh.indexMgr.GetFromIndex(tx, IndexWorkspaceOwner, ownerID)
		if err != nil {
			return nil, fmt.Errorf("소유자 인덱스 조회 실패: %w", err)
		}
		targetKeys = keys
	}
	
	// 데이터 조회 및 필터링
	var workspaces []*models.Workspace
	var totalCount int
	
	if len(targetKeys) > 0 {
		// 인덱스 기반 조회
		for _, key := range targetKeys {
			data := bucket.Get([]byte(key))
			if data == nil {
				continue
			}
			
			workspace, err := qh.serializers.Workspace.Unmarshal(data)
			if err != nil {
				continue
			}
			
			// 추가 필터 적용
			if qh.matchWorkspaceFilter(workspace, options.Filter) {
				workspaces = append(workspaces, workspace)
				totalCount++
			}
		}
	} else {
		// 전체 스캔
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			workspace, err := qh.serializers.Workspace.Unmarshal(v)
			if err != nil {
				continue
			}
			
			if qh.matchWorkspaceFilter(workspace, options.Filter) {
				workspaces = append(workspaces, workspace)
				totalCount++
			}
		}
	}
	
	// 정렬
	qh.sortWorkspaces(workspaces, options.Sort, options.Order)
	
	// 페이지네이션
	start := options.GetOffset()
	end := start + options.Limit
	
	if start >= len(workspaces) {
		workspaces = []*models.Workspace{}
	} else {
		if end > len(workspaces) {
			end = len(workspaces)
		}
		workspaces = workspaces[start:end]
	}
	
	hasMore := (start + len(workspaces)) < totalCount
	
	return &QueryResult[*models.Workspace]{
		Items:      workspaces,
		TotalCount: totalCount,
		Page:       options.Page,
		Limit:      options.Limit,
		HasMore:    hasMore,
	}, nil
}

// ProjectQuery 프로젝트 쿼리
func (qh *QueryHelper) ProjectQuery(tx *bbolt.Tx, options *QueryOptions) (*QueryResult[*models.Project], error) {
	if options == nil {
		options = &QueryOptions{}
	}
	options.Normalize()
	
	bucket := tx.Bucket([]byte(BucketProjects))
	if bucket == nil {
		return nil, fmt.Errorf("프로젝트 버킷이 존재하지 않습니다")
	}
	
	// 필터 조건 확인
	var targetKeys []string
	if workspaceID, ok := options.Filter["workspace_id"].(string); ok && workspaceID != "" {
		// 워크스페이스 ID 기반 필터링
		keys, err := qh.indexMgr.GetFromIndex(tx, IndexProjectWorkspace, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("워크스페이스 인덱스 조회 실패: %w", err)
		}
		targetKeys = keys
	}
	
	// 데이터 조회 및 필터링
	var projects []*models.Project
	var totalCount int
	
	if len(targetKeys) > 0 {
		// 인덱스 기반 조회
		for _, key := range targetKeys {
			data := bucket.Get([]byte(key))
			if data == nil {
				continue
			}
			
			project, err := qh.serializers.Project.Unmarshal(data)
			if err != nil {
				continue
			}
			
			if qh.matchProjectFilter(project, options.Filter) {
				projects = append(projects, project)
				totalCount++
			}
		}
	} else {
		// 전체 스캔
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			project, err := qh.serializers.Project.Unmarshal(v)
			if err != nil {
				continue
			}
			
			if qh.matchProjectFilter(project, options.Filter) {
				projects = append(projects, project)
				totalCount++
			}
		}
	}
	
	// 정렬
	qh.sortProjects(projects, options.Sort, options.Order)
	
	// 페이지네이션
	start := options.GetOffset()
	end := start + options.Limit
	
	if start >= len(projects) {
		projects = []*models.Project{}
	} else {
		if end > len(projects) {
			end = len(projects)
		}
		projects = projects[start:end]
	}
	
	hasMore := (start + len(projects)) < totalCount
	
	return &QueryResult[*models.Project]{
		Items:      projects,
		TotalCount: totalCount,
		Page:       options.Page,
		Limit:      options.Limit,
		HasMore:    hasMore,
	}, nil
}

// SessionQuery 세션 쿼리
func (qh *QueryHelper) SessionQuery(tx *bbolt.Tx, filter *models.SessionFilter, options *QueryOptions) (*QueryResult[*models.Session], error) {
	if options == nil {
		options = &QueryOptions{}
	}
	options.Normalize()
	
	bucket := tx.Bucket([]byte(BucketSessions))
	if bucket == nil {
		return nil, fmt.Errorf("세션 버킷이 존재하지 않습니다")
	}
	
	// 필터를 QueryOptions 형식으로 변환
	if filter != nil {
		if filter.ProjectID != "" {
			options.Filter["project_id"] = filter.ProjectID
		}
		if filter.Status != "" {
			options.Filter["status"] = filter.Status
		}
		if filter.Active != nil {
			options.Filter["active"] = *filter.Active
		}
	}
	
	// 데이터 조회 및 필터링
	var sessions []*models.Session
	var totalCount int
	
	cursor := bucket.Cursor()
	for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
		session, err := qh.serializers.Session.Unmarshal(v)
		if err != nil {
			continue
		}
		
		if qh.matchSessionFilter(session, options.Filter) {
			sessions = append(sessions, session)
			totalCount++
		}
	}
	
	// 정렬
	qh.sortSessions(sessions, options.Sort, options.Order)
	
	// 페이지네이션
	start := options.GetOffset()
	end := start + options.Limit
	
	if start >= len(sessions) {
		sessions = []*models.Session{}
	} else {
		if end > len(sessions) {
			end = len(sessions)
		}
		sessions = sessions[start:end]
	}
	
	hasMore := (start + len(sessions)) < totalCount
	
	return &QueryResult[*models.Session]{
		Items:      sessions,
		TotalCount: totalCount,
		Page:       options.Page,
		Limit:      options.Limit,
		HasMore:    hasMore,
	}, nil
}

// TaskQuery 태스크 쿼리
func (qh *QueryHelper) TaskQuery(tx *bbolt.Tx, filter *models.TaskFilter, options *QueryOptions) (*QueryResult[*models.Task], error) {
	if options == nil {
		options = &QueryOptions{}
	}
	options.Normalize()
	
	bucket := tx.Bucket([]byte(BucketTasks))
	if bucket == nil {
		return nil, fmt.Errorf("태스크 버킷이 존재하지 않습니다")
	}
	
	// 필터를 QueryOptions 형식으로 변환
	if filter != nil {
		if filter.SessionID != nil && *filter.SessionID != "" {
			options.Filter["session_id"] = *filter.SessionID
		}
		if filter.Status != nil {
			options.Filter["status"] = *filter.Status
		}
		if filter.Active != nil {
			options.Filter["active"] = *filter.Active
		}
	}
	
	// 데이터 조회 및 필터링
	var tasks []*models.Task
	var totalCount int
	
	cursor := bucket.Cursor()
	for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
		task, err := qh.serializers.Task.Unmarshal(v)
		if err != nil {
			continue
		}
		
		if qh.matchTaskFilter(task, options.Filter) {
			tasks = append(tasks, task)
			totalCount++
		}
	}
	
	// 정렬
	qh.sortTasks(tasks, options.Sort, options.Order)
	
	// 페이지네이션
	start := options.GetOffset()
	end := start + options.Limit
	
	if start >= len(tasks) {
		tasks = []*models.Task{}
	} else {
		if end > len(tasks) {
			end = len(tasks)
		}
		tasks = tasks[start:end]
	}
	
	hasMore := (start + len(tasks)) < totalCount
	
	return &QueryResult[*models.Task]{
		Items:      tasks,
		TotalCount: totalCount,
		Page:       options.Page,
		Limit:      options.Limit,
		HasMore:    hasMore,
	}, nil
}

// 필터 매칭 함수들

func (qh *QueryHelper) matchWorkspaceFilter(workspace *models.Workspace, filter map[string]interface{}) bool {
	if ownerID, ok := filter["owner_id"].(string); ok && ownerID != "" {
		if workspace.OwnerID != ownerID {
			return false
		}
	}
	
	if status, ok := filter["status"].(models.WorkspaceStatus); ok && status != "" {
		if workspace.Status != status {
			return false
		}
	}
	
	if name, ok := filter["name"].(string); ok && name != "" {
		if !strings.Contains(strings.ToLower(workspace.Name), strings.ToLower(name)) {
			return false
		}
	}
	
	// Soft delete 체크
	if workspace.DeletedAt != nil {
		return false
	}
	
	return true
}

func (qh *QueryHelper) matchProjectFilter(project *models.Project, filter map[string]interface{}) bool {
	if workspaceID, ok := filter["workspace_id"].(string); ok && workspaceID != "" {
		if project.WorkspaceID != workspaceID {
			return false
		}
	}
	
	if status, ok := filter["status"].(models.ProjectStatus); ok && status != "" {
		if project.Status != status {
			return false
		}
	}
	
	if name, ok := filter["name"].(string); ok && name != "" {
		if !strings.Contains(strings.ToLower(project.Name), strings.ToLower(name)) {
			return false
		}
	}
	
	// Soft delete 체크
	if project.DeletedAt != nil {
		return false
	}
	
	return true
}

func (qh *QueryHelper) matchSessionFilter(session *models.Session, filter map[string]interface{}) bool {
	if projectID, ok := filter["project_id"].(string); ok && projectID != "" {
		if session.ProjectID != projectID {
			return false
		}
	}
	
	if status, ok := filter["status"].(models.SessionStatus); ok && status != "" {
		if session.Status != status {
			return false
		}
	}
	
	if active, ok := filter["active"].(bool); ok {
		isActive := session.IsActive()
		if active != isActive {
			return false
		}
	}
	
	return true
}

func (qh *QueryHelper) matchTaskFilter(task *models.Task, filter map[string]interface{}) bool {
	if sessionID, ok := filter["session_id"].(string); ok && sessionID != "" {
		if task.SessionID != sessionID {
			return false
		}
	}
	
	if status, ok := filter["status"].(models.TaskStatus); ok && status != "" {
		if task.Status != status {
			return false
		}
	}
	
	if active, ok := filter["active"].(bool); ok {
		isActive := task.IsActive()
		if active != isActive {
			return false
		}
	}
	
	return true
}

// 정렬 함수들

func (qh *QueryHelper) sortWorkspaces(workspaces []*models.Workspace, sortBy string, order SortOrder) {
	sort.Slice(workspaces, func(i, j int) bool {
		var less bool
		
		switch sortBy {
		case "created_at":
			less = workspaces[i].CreatedAt.Before(workspaces[j].CreatedAt)
		case "updated_at":
			less = workspaces[i].UpdatedAt.Before(workspaces[j].UpdatedAt)
		case "name":
			less = strings.ToLower(workspaces[i].Name) < strings.ToLower(workspaces[j].Name)
		default:
			less = workspaces[i].CreatedAt.Before(workspaces[j].CreatedAt)
		}
		
		if order == SortOrderAsc {
			return less
		}
		return !less
	})
}

func (qh *QueryHelper) sortProjects(projects []*models.Project, sortBy string, order SortOrder) {
	sort.Slice(projects, func(i, j int) bool {
		var less bool
		
		switch sortBy {
		case "created_at":
			less = projects[i].CreatedAt.Before(projects[j].CreatedAt)
		case "updated_at":
			less = projects[i].UpdatedAt.Before(projects[j].UpdatedAt)
		case "name":
			less = strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
		default:
			less = projects[i].CreatedAt.Before(projects[j].CreatedAt)
		}
		
		if order == SortOrderAsc {
			return less
		}
		return !less
	})
}

func (qh *QueryHelper) sortSessions(sessions []*models.Session, sortBy string, order SortOrder) {
	sort.Slice(sessions, func(i, j int) bool {
		var less bool
		
		switch sortBy {
		case "created_at":
			less = sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
		case "updated_at":
			less = sessions[i].UpdatedAt.Before(sessions[j].UpdatedAt)
		case "last_active":
			less = sessions[i].LastActive.Before(sessions[j].LastActive)
		default:
			less = sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
		}
		
		if order == SortOrderAsc {
			return less
		}
		return !less
	})
}

func (qh *QueryHelper) sortTasks(tasks []*models.Task, sortBy string, order SortOrder) {
	sort.Slice(tasks, func(i, j int) bool {
		var less bool
		
		switch sortBy {
		case "created_at":
			less = tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
		case "updated_at":
			less = tasks[i].UpdatedAt.Before(tasks[j].UpdatedAt)
		case "duration":
			less = tasks[i].Duration < tasks[j].Duration
		default:
			less = tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
		}
		
		if order == SortOrderAsc {
			return less
		}
		return !less
	})
}

// SearchByText 텍스트 검색
func (qh *QueryHelper) SearchByText(tx *bbolt.Tx, bucketName, text string, limit int) ([]string, error) {
	bucket := tx.Bucket([]byte(bucketName))
	if bucket == nil {
		return nil, fmt.Errorf("버킷 '%s'이 존재하지 않습니다", bucketName)
	}
	
	if limit <= 0 {
		limit = 50
	}
	
	var results []string
	searchText := strings.ToLower(text)
	
	cursor := bucket.Cursor()
	for k, v := cursor.First(); k != nil && len(results) < limit; k, v = cursor.Next() {
		// JSON 데이터에서 텍스트 검색 (단순한 포함 여부 체크)
		if strings.Contains(strings.ToLower(string(v)), searchText) {
			results = append(results, string(k))
		}
	}
	
	return results, nil
}

// CountBy 조건별 카운트
func (qh *QueryHelper) CountBy(tx *bbolt.Tx, bucketName string, filter map[string]interface{}) (int, error) {
	bucket := tx.Bucket([]byte(bucketName))
	if bucket == nil {
		return 0, fmt.Errorf("버킷 '%s'이 존재하지 않습니다", bucketName)
	}
	
	count := 0
	cursor := bucket.Cursor()
	
	for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
		// 여기서는 단순히 전체 카운트 반환
		// 실제로는 filter 조건에 맞는 항목만 카운트해야 함
		count++
	}
	
	return count, nil
}

// GetLatest 최신 항목 조회
func (qh *QueryHelper) GetLatest(tx *bbolt.Tx, bucketName string, limit int) ([][]byte, error) {
	bucket := tx.Bucket([]byte(bucketName))
	if bucket == nil {
		return nil, fmt.Errorf("버킷 '%s'이 존재하지 않습니다", bucketName)
	}
	
	if limit <= 0 {
		limit = 10
	}
	
	var results [][]byte
	cursor := bucket.Cursor()
	
	// BoltDB는 키 순서로 정렬되므로, 역순으로 순회하여 최신 항목 조회
	for k, v := cursor.Last(); k != nil && len(results) < limit; k, v = cursor.Prev() {
		dataCopy := make([]byte, len(v))
		copy(dataCopy, v)
		results = append(results, dataCopy)
	}
	
	return results, nil
}