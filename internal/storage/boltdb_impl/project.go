package boltdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.etcd.io/bbolt"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/storage/interfaces"
)

// projectStorage BoltDB Project 스토리지 구현
type projectStorage struct {
	db         *bbolt.DB
	serializer *Serializer
	indexer    *IndexManager
	querier    *QueryHelper
}

// newProjectStorage 프로젝트 스토리지 생성자
func newProjectStorage(storage *Storage) *projectStorage {
	return &projectStorage{
		db:         storage.db,
		serializer: storage.serializer,
		indexer:    storage.indexer,
		querier:    storage.querier,
	}
}

// Create 새 프로젝트 생성
func (ps *projectStorage) Create(ctx context.Context, project *models.Project) error {
	return ps.db.Update(func(tx *bbolt.Tx) error {
		// 프로젝트 버킷 가져오기
		projectBucket := tx.Bucket([]byte(bucketProjects))
		if projectBucket == nil {
			return fmt.Errorf("projects bucket not found")
		}
		
		// ID 중복 체크
		if existing := projectBucket.Get([]byte(project.ID)); existing != nil {
			return storage.ErrAlreadyExists
		}
		
		// 워크스페이스 존재 확인
		workspaceBucket := tx.Bucket([]byte(bucketWorkspaces))
		if workspaceBucket == nil {
			return fmt.Errorf("workspaces bucket not found")
		}
		
		workspaceData := workspaceBucket.Get([]byte(project.WorkspaceID))
		if workspaceData == nil {
			return storage.ErrNotFound
		}
		
		// 워크스페이스가 삭제되지 않았는지 확인
		var workspace models.Workspace
		if err := ps.serializer.UnmarshalWorkspace(workspaceData, &workspace); err != nil {
			return err
		}
		if workspace.DeletedAt != nil {
			return storage.ErrNotFound
		}
		
		// 같은 워크스페이스 내 프로젝트 이름 중복 체크
		exists, err := ps.existsByNameInWorkspace(tx, project.WorkspaceID, project.Name)
		if err != nil {
			return err
		}
		if exists {
			return storage.ErrAlreadyExists
		}
		
		// 타임스탬프 설정
		now := time.Now()
		if project.CreatedAt.IsZero() {
			project.CreatedAt = now
		}
		project.UpdatedAt = now
		if project.Version == 0 {
			project.Version = 1
		}
		
		// 프로젝트 직렬화 및 저장
		data, err := ps.serializer.MarshalProject(project)
		if err != nil {
			return err
		}
		
		if err := projectBucket.Put([]byte(project.ID), data); err != nil {
			return err
		}
		
		// 인덱스 업데이트
		if err := ps.indexer.AddToIndex(tx, IndexWorkspaceProjects, project.WorkspaceID, project.ID); err != nil {
			return err
		}
		
		if err := ps.indexer.AddToIndex(tx, IndexProjectName, project.Name, project.ID); err != nil {
			return err
		}
		
		if err := ps.indexer.AddToIndex(tx, IndexProjectStatus, string(project.Status), project.ID); err != nil {
			return err
		}
		
		if project.Path != "" {
			if err := ps.indexer.AddToIndex(tx, IndexProjectPath, project.Path, project.ID); err != nil {
				return err
			}
		}
		
		if project.Language != "" {
			if err := ps.indexer.AddToIndex(tx, IndexProjectLanguage, project.Language, project.ID); err != nil {
				return err
			}
		}
		
		return nil
	})
}

// GetByID ID로 프로젝트 조회
func (ps *projectStorage) GetByID(ctx context.Context, id string) (*models.Project, error) {
	var project models.Project
	
	err := ps.db.View(func(tx *bbolt.Tx) error {
		projectBucket := tx.Bucket([]byte(bucketProjects))
		if projectBucket == nil {
			return fmt.Errorf("projects bucket not found")
		}
		
		data := projectBucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		if err := ps.serializer.UnmarshalProject(data, &project); err != nil {
			return err
		}
		
		// Soft Delete 확인
		if project.DeletedAt != nil {
			return storage.ErrNotFound
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return &project, nil
}

// GetByWorkspaceID 워크스페이스별 프로젝트 목록 조회
func (ps *projectStorage) GetByWorkspaceID(ctx context.Context, workspaceID string, pagination *models.PaginationRequest) ([]*models.Project, int64, error) {
	var projects []*models.Project
	var total int64
	
	err := ps.db.View(func(tx *bbolt.Tx) error {
		// 인덱스에서 프로젝트 ID 목록 가져오기
		projectIDs, err := ps.indexer.GetFromIndex(tx, IndexWorkspaceProjects, workspaceID)
		if err != nil {
			return err
		}
		
		// 활성 프로젝트 필터링 및 정렬
		var activeProjects []*models.Project
		projectBucket := tx.Bucket([]byte(bucketProjects))
		if projectBucket == nil {
			return fmt.Errorf("projects bucket not found")
		}
		
		for _, projectID := range projectIDs {
			data := projectBucket.Get([]byte(projectID))
			if data == nil {
				continue
			}
			
			var project models.Project
			if err := ps.serializer.UnmarshalProject(data, &project); err != nil {
				continue
			}
			
			// Soft Delete 확인
			if project.DeletedAt != nil {
				continue
			}
			
			activeProjects = append(activeProjects, &project)
		}
		
		// 생성 시간 역순 정렬
		ps.querier.SortProjects(activeProjects, "created_at", false)
		
		total = int64(len(activeProjects))
		
		// 페이지네이션 적용
		start, end := ps.querier.CalculatePagination(len(activeProjects), pagination)
		projects = activeProjects[start:end]
		
		return nil
	})
	
	if err != nil {
		return nil, 0, err
	}
	
	return projects, total, nil
}

// GetByPath 경로로 프로젝트 조회
func (ps *projectStorage) GetByPath(ctx context.Context, path string) (*models.Project, error) {
	var project models.Project
	
	err := ps.db.View(func(tx *bbolt.Tx) error {
		// 인덱스에서 프로젝트 ID 찾기
		projectIDs, err := ps.indexer.GetFromIndex(tx, IndexProjectPath, path)
		if err != nil {
			return err
		}
		
		if len(projectIDs) == 0 {
			return storage.ErrNotFound
		}
		
		// 첫 번째 매치 (경로는 고유해야 함)
		projectBucket := tx.Bucket([]byte(bucketProjects))
		if projectBucket == nil {
			return fmt.Errorf("projects bucket not found")
		}
		
		data := projectBucket.Get([]byte(projectIDs[0]))
		if data == nil {
			return storage.ErrNotFound
		}
		
		if err := ps.serializer.UnmarshalProject(data, &project); err != nil {
			return err
		}
		
		// Soft Delete 확인
		if project.DeletedAt != nil {
			return storage.ErrNotFound
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return &project, nil
}

// ExistsByName 워크스페이스 내 프로젝트 이름 중복 확인
func (ps *projectStorage) ExistsByName(ctx context.Context, workspaceID, name string) (bool, error) {
	exists := false
	
	err := ps.db.View(func(tx *bbolt.Tx) error {
		var err error
		exists, err = ps.existsByNameInWorkspace(tx, workspaceID, name)
		return err
	})
	
	return exists, err
}

// Update 프로젝트 정보 업데이트
func (ps *projectStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return ps.db.Update(func(tx *bbolt.Tx) error {
		projectBucket := tx.Bucket([]byte(bucketProjects))
		if projectBucket == nil {
			return fmt.Errorf("projects bucket not found")
		}
		
		// 기존 프로젝트 조회
		data := projectBucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		var project models.Project
		if err := ps.serializer.UnmarshalProject(data, &project); err != nil {
			return err
		}
		
		// Soft Delete 확인
		if project.DeletedAt != nil {
			return storage.ErrNotFound
		}
		
		// 인덱스에서 이전 값 제거를 위해 저장
		oldName := project.Name
		oldPath := project.Path
		oldStatus := string(project.Status)
		oldLanguage := project.Language
		
		// 업데이트 적용
		if name, ok := updates["name"]; ok {
			if nameStr, ok := name.(string); ok {
				// 새 이름 중복 체크 (기존 이름과 다를 때만)
				if nameStr != oldName {
					exists, err := ps.existsByNameInWorkspace(tx, project.WorkspaceID, nameStr)
					if err != nil {
						return err
					}
					if exists {
						return storage.ErrAlreadyExists
					}
				}
				project.Name = nameStr
			}
		}
		
		if path, ok := updates["path"]; ok {
			if pathStr, ok := path.(string); ok {
				project.Path = pathStr
			}
		}
		
		if language, ok := updates["language"]; ok {
			if langStr, ok := language.(string); ok {
				project.Language = langStr
			}
		}
		
		if status, ok := updates["status"]; ok {
			if statusVal, ok := status.(models.ProjectStatus); ok {
				project.Status = statusVal
			} else if statusStr, ok := status.(string); ok {
				project.Status = models.ProjectStatus(statusStr)
			}
		}
		
		if config, ok := updates["config"]; ok {
			if configVal, ok := config.(models.ProjectConfig); ok {
				project.Config = configVal
			}
		}
		
		if gitInfo, ok := updates["git_info"]; ok {
			if gitVal, ok := gitInfo.(models.GitInfo); ok {
				project.GitInfo = gitVal
			}
		}
		
		// 타임스탬프 및 버전 업데이트
		project.UpdatedAt = time.Now()
		project.Version++
		
		// 직렬화 및 저장
		updatedData, err := ps.serializer.MarshalProject(&project)
		if err != nil {
			return err
		}
		
		if err := projectBucket.Put([]byte(id), updatedData); err != nil {
			return err
		}
		
		// 인덱스 업데이트
		if project.Name != oldName {
			ps.indexer.RemoveFromIndex(tx, IndexProjectName, oldName, id)
			ps.indexer.AddToIndex(tx, IndexProjectName, project.Name, id)
		}
		
		if project.Path != oldPath {
			if oldPath != "" {
				ps.indexer.RemoveFromIndex(tx, IndexProjectPath, oldPath, id)
			}
			if project.Path != "" {
				ps.indexer.AddToIndex(tx, IndexProjectPath, project.Path, id)
			}
		}
		
		if string(project.Status) != oldStatus {
			ps.indexer.RemoveFromIndex(tx, IndexProjectStatus, oldStatus, id)
			ps.indexer.AddToIndex(tx, IndexProjectStatus, string(project.Status), id)
		}
		
		if project.Language != oldLanguage {
			if oldLanguage != "" {
				ps.indexer.RemoveFromIndex(tx, IndexProjectLanguage, oldLanguage, id)
			}
			if project.Language != "" {
				ps.indexer.AddToIndex(tx, IndexProjectLanguage, project.Language, id)
			}
		}
		
		return nil
	})
}

// Delete 프로젝트 삭제 (Soft Delete)
func (ps *projectStorage) Delete(ctx context.Context, id string) error {
	return ps.db.Update(func(tx *bbolt.Tx) error {
		projectBucket := tx.Bucket([]byte(bucketProjects))
		if projectBucket == nil {
			return fmt.Errorf("projects bucket not found")
		}
		
		// 기존 프로젝트 조회
		data := projectBucket.Get([]byte(id))
		if data == nil {
			return storage.ErrNotFound
		}
		
		var project models.Project
		if err := ps.serializer.UnmarshalProject(data, &project); err != nil {
			return err
		}
		
		// 이미 삭제된 경우
		if project.DeletedAt != nil {
			return storage.ErrNotFound
		}
		
		// 관련 세션이 있는지 확인
		sessionIDs, err := ps.indexer.GetFromIndex(tx, IndexProjectSessions, id)
		if err != nil {
			return err
		}
		
		// 활성 세션이 있는지 확인
		if len(sessionIDs) > 0 {
			sessionBucket := tx.Bucket([]byte(bucketSessions))
			if sessionBucket != nil {
				for _, sessionID := range sessionIDs {
					sessionData := sessionBucket.Get([]byte(sessionID))
					if sessionData != nil {
						var session models.Session
						if ps.serializer.UnmarshalSession(sessionData, &session) == nil {
							if session.Status == models.SessionStatusActive {
								return fmt.Errorf("cannot delete project with active sessions")
							}
						}
					}
				}
			}
		}
		
		// Soft Delete
		now := time.Now()
		project.DeletedAt = &now
		project.UpdatedAt = now
		project.Version++
		
		// 직렬화 및 저장
		updatedData, err := ps.serializer.MarshalProject(&project)
		if err != nil {
			return err
		}
		
		if err := projectBucket.Put([]byte(id), updatedData); err != nil {
			return err
		}
		
		// 인덱스에서 제거 (삭제된 항목은 일반 조회에서 제외)
		ps.indexer.RemoveFromIndex(tx, IndexProjectName, project.Name, id)
		ps.indexer.RemoveFromIndex(tx, IndexProjectStatus, string(project.Status), id)
		if project.Path != "" {
			ps.indexer.RemoveFromIndex(tx, IndexProjectPath, project.Path, id)
		}
		if project.Language != "" {
			ps.indexer.RemoveFromIndex(tx, IndexProjectLanguage, project.Language, id)
		}
		
		return nil
	})
}

// 헬퍼 메서드들

// existsByNameInWorkspace 워크스페이스 내 프로젝트 이름 중복 체크 (트랜잭션 내)
func (ps *projectStorage) existsByNameInWorkspace(tx *bbolt.Tx, workspaceID, name string) (bool, error) {
	// 워크스페이스의 모든 프로젝트 조회
	projectIDs, err := ps.indexer.GetFromIndex(tx, IndexWorkspaceProjects, workspaceID)
	if err != nil {
		return false, err
	}
	
	projectBucket := tx.Bucket([]byte(bucketProjects))
	if projectBucket == nil {
		return false, fmt.Errorf("projects bucket not found")
	}
	
	for _, projectID := range projectIDs {
		data := projectBucket.Get([]byte(projectID))
		if data == nil {
			continue
		}
		
		var project models.Project
		if err := ps.serializer.UnmarshalProject(data, &project); err != nil {
			continue
		}
		
		// 삭제되지 않은 프로젝트만 확인
		if project.DeletedAt == nil && strings.EqualFold(project.Name, name) {
			return true, nil
		}
	}
	
	return false, nil
}

// GetByLanguage 언어별 프로젝트 조회
func (ps *projectStorage) GetByLanguage(ctx context.Context, language string, pagination *models.PaginationRequest) ([]*models.Project, int64, error) {
	var projects []*models.Project
	var total int64
	
	err := ps.db.View(func(tx *bbolt.Tx) error {
		// 인덱스에서 프로젝트 ID 목록 가져오기
		projectIDs, err := ps.indexer.GetFromIndex(tx, IndexProjectLanguage, language)
		if err != nil {
			return err
		}
		
		// 활성 프로젝트 필터링
		var activeProjects []*models.Project
		projectBucket := tx.Bucket([]byte(bucketProjects))
		if projectBucket == nil {
			return fmt.Errorf("projects bucket not found")
		}
		
		for _, projectID := range projectIDs {
			data := projectBucket.Get([]byte(projectID))
			if data == nil {
				continue
			}
			
			var project models.Project
			if err := ps.serializer.UnmarshalProject(data, &project); err != nil {
				continue
			}
			
			// Soft Delete 확인
			if project.DeletedAt != nil {
				continue
			}
			
			activeProjects = append(activeProjects, &project)
		}
		
		// 이름순 정렬
		ps.querier.SortProjects(activeProjects, "name", true)
		
		total = int64(len(activeProjects))
		
		// 페이지네이션 적용
		start, end := ps.querier.CalculatePagination(len(activeProjects), pagination)
		projects = activeProjects[start:end]
		
		return nil
	})
	
	if err != nil {
		return nil, 0, err
	}
	
	return projects, total, nil
}

// GetByStatus 상태별 프로젝트 조회
func (ps *projectStorage) GetByStatus(ctx context.Context, status models.ProjectStatus, pagination *models.PaginationRequest) ([]*models.Project, int64, error) {
	var projects []*models.Project
	var total int64
	
	err := ps.db.View(func(tx *bbolt.Tx) error {
		// 인덱스에서 프로젝트 ID 목록 가져오기
		projectIDs, err := ps.indexer.GetFromIndex(tx, IndexProjectStatus, string(status))
		if err != nil {
			return err
		}
		
		// 활성 프로젝트 필터링
		var activeProjects []*models.Project
		projectBucket := tx.Bucket([]byte(bucketProjects))
		if projectBucket == nil {
			return fmt.Errorf("projects bucket not found")
		}
		
		for _, projectID := range projectIDs {
			data := projectBucket.Get([]byte(projectID))
			if data == nil {
				continue
			}
			
			var project models.Project
			if err := ps.serializer.UnmarshalProject(data, &project); err != nil {
				continue
			}
			
			// Soft Delete 확인
			if project.DeletedAt != nil {
				continue
			}
			
			activeProjects = append(activeProjects, &project)
		}
		
		// 업데이트 시간 역순 정렬
		ps.querier.SortProjects(activeProjects, "updated_at", false)
		
		total = int64(len(activeProjects))
		
		// 페이지네이션 적용
		start, end := ps.querier.CalculatePagination(len(activeProjects), pagination)
		projects = activeProjects[start:end]
		
		return nil
	})
	
	if err != nil {
		return nil, 0, err
	}
	
	return projects, total, nil
}