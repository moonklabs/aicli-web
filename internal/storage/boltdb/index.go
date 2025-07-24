package boltdb

import (
	"fmt"
	"sort"
	"strings"

	"go.etcd.io/bbolt"
)

// IndexManager 인덱스 관리자
type IndexManager struct {
	storage    *Storage
	serializer *IndexSerializer
}

// newIndexManager 새 인덱스 관리자 생성
func newIndexManager(storage *Storage) *IndexManager {
	return &IndexManager{
		storage:    storage,
		serializer: &IndexSerializer{},
	}
}

// NewIndexManager 새 인덱스 관리자 생성 (전역 생성자)
func NewIndexManager() *IndexManager {
	return &IndexManager{
		serializer: &IndexSerializer{},
	}
}

// IndexType 인덱스 타입
type IndexType string

const (
	IndexTypeUnique   IndexType = "unique"   // 유니크 인덱스 (1:1)
	IndexTypeMultiple IndexType = "multiple" // 다중 인덱스 (1:N)
)

// Index 인덱스 정의
type Index struct {
	Name       string    // 인덱스 이름
	BucketName string    // 버킷 이름
	Type       IndexType // 인덱스 타입
}

// 미리 정의된 인덱스들
var (
	// 워크스페이스 인덱스
	IndexWorkspaceOwner = Index{
		Name:       "workspace_owner",
		BucketName: BucketIndexOwner,
		Type:       IndexTypeMultiple,
	}
	
	IndexWorkspaceName = Index{
		Name:       "workspace_name",
		BucketName: BucketIndexName,
		Type:       IndexTypeUnique,
	}
	
	// 프로젝트 인덱스
	IndexProjectWorkspace = Index{
		Name:       "project_workspace",
		BucketName: BucketIndexWorkspace,
		Type:       IndexTypeMultiple,
	}
	
	IndexProjectPath = Index{
		Name:       "project_path", 
		BucketName: BucketIndexPath,
		Type:       IndexTypeUnique,
	}
	
	// 세션 인덱스
	IndexSessionProject = Index{
		Name:       "session_project",
		BucketName: BucketIndexProject,
		Type:       IndexTypeMultiple,
	}
	
	// 태스크 인덱스
	IndexTaskSession = Index{
		Name:       "task_session",
		BucketName: BucketIndexSession,
		Type:       IndexTypeMultiple,
	}
	
	// 상태 인덱스
	IndexEntityStatus = Index{
		Name:       "entity_status",
		BucketName: BucketIndexStatus,
		Type:       IndexTypeMultiple,
	}
	
	// 프로젝트 관련 추가 인덱스 (별칭)
	IndexWorkspaceProjects = IndexProjectWorkspace
	IndexProjectName = Index{
		Name:       "project_name",
		BucketName: BucketIndexWorkspace,
		Type:       IndexTypeMultiple,
	}
	IndexProjectStatus = Index{
		Name:       "project_status",
		BucketName: BucketIndexStatus,
		Type:       IndexTypeMultiple,
	}
	IndexProjectLanguage = Index{
		Name:       "project_language",
		BucketName: BucketIndexStatus,
		Type:       IndexTypeMultiple,
	}
	IndexProjectSessions = Index{
		Name:       "project_sessions",
		BucketName: BucketIndexProject,
		Type:       IndexTypeMultiple,
	}
	
	// 세션 관련 인덱스
	IndexSessionStatus = Index{
		Name:       "session_status",
		BucketName: BucketIndexStatus,
		Type:       IndexTypeMultiple,
	}
	
	IndexSessionProcess = Index{
		Name:       "session_process",
		BucketName: BucketIndexSession,
		Type:       IndexTypeMultiple,
	}
	
	IndexSessionTasks = Index{
		Name:       "session_tasks",
		BucketName: BucketIndexSession,
		Type:       IndexTypeMultiple,
	}
	
	// 태스크 상태 인덱스
	IndexTaskStatus = Index{
		Name:       "task_status",
		BucketName: BucketIndexStatus,
		Type:       IndexTypeMultiple,
	}
	
	// 태스크 명령어 인덱스
	IndexTaskCommand = Index{
		Name:       "task_command",
		BucketName: BucketIndexSession,
		Type:       IndexTypeMultiple,
	}
)

// AddToIndex 인덱스에 항목 추가
func (im *IndexManager) AddToIndex(tx *bbolt.Tx, index Index, key, value string) error {
	bucket := tx.Bucket([]byte(index.BucketName))
	if bucket == nil {
		return fmt.Errorf("인덱스 버킷 '%s'이 존재하지 않습니다", index.BucketName)
	}
	
	switch index.Type {
	case IndexTypeUnique:
		return im.addUniqueIndex(bucket, key, value)
	case IndexTypeMultiple:
		return im.addMultipleIndex(bucket, key, value)
	default:
		return fmt.Errorf("알 수 없는 인덱스 타입: %s", index.Type)
	}
}

// RemoveFromIndex 인덱스에서 항목 제거
func (im *IndexManager) RemoveFromIndex(tx *bbolt.Tx, index Index, key, value string) error {
	bucket := tx.Bucket([]byte(index.BucketName))
	if bucket == nil {
		return fmt.Errorf("인덱스 버킷 '%s'이 존재하지 않습니다", index.BucketName)
	}
	
	switch index.Type {
	case IndexTypeUnique:
		return im.removeUniqueIndex(bucket, key)
	case IndexTypeMultiple:
		return im.removeMultipleIndex(bucket, key, value)
	default:
		return fmt.Errorf("알 수 없는 인덱스 타입: %s", index.Type)
	}
}

// GetFromIndex 인덱스에서 항목 조회
func (im *IndexManager) GetFromIndex(tx *bbolt.Tx, index Index, key string) ([]string, error) {
	bucket := tx.Bucket([]byte(index.BucketName))
	if bucket == nil {
		return nil, fmt.Errorf("인덱스 버킷 '%s'이 존재하지 않습니다", index.BucketName)
	}
	
	data := bucket.Get([]byte(key))
	if data == nil {
		return []string{}, nil
	}
	
	switch index.Type {
	case IndexTypeUnique:
		return []string{string(data)}, nil
	case IndexTypeMultiple:
		return im.serializer.UnmarshalStringList(data)
	default:
		return nil, fmt.Errorf("알 수 없는 인덱스 타입: %s", index.Type)
	}
}

// ExistsInIndex 인덱스에 키가 존재하는지 확인
func (im *IndexManager) ExistsInIndex(tx *bbolt.Tx, index Index, key string) (bool, error) {
	bucket := tx.Bucket([]byte(index.BucketName))
	if bucket == nil {
		return false, fmt.Errorf("인덱스 버킷 '%s'이 존재하지 않습니다", index.BucketName)
	}
	
	data := bucket.Get([]byte(key))
	return data != nil, nil
}

// ClearIndex 인덱스 전체 삭제
func (im *IndexManager) ClearIndex(tx *bbolt.Tx, index Index) error {
	bucket := tx.Bucket([]byte(index.BucketName))
	if bucket == nil {
		return fmt.Errorf("인덱스 버킷 '%s'이 존재하지 않습니다", index.BucketName)
	}
	
	// 모든 키 수집
	var keys [][]byte
	cursor := bucket.Cursor()
	for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
		keyCopy := make([]byte, len(k))
		copy(keyCopy, k)
		keys = append(keys, keyCopy)
	}
	
	// 키들 삭제
	for _, key := range keys {
		if err := bucket.Delete(key); err != nil {
			return fmt.Errorf("인덱스 키 삭제 실패: %w", err)
		}
	}
	
	return nil
}

// ListKeys 인덱스의 모든 키 조회
func (im *IndexManager) ListKeys(tx *bbolt.Tx, index Index, prefix string) ([]string, error) {
	bucket := tx.Bucket([]byte(index.BucketName))
	if bucket == nil {
		return nil, fmt.Errorf("인덱스 버킷 '%s'이 존재하지 않습니다", index.BucketName)
	}
	
	var keys []string
	cursor := bucket.Cursor()
	
	if prefix == "" {
		// 전체 키 조회
		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			keys = append(keys, string(k))
		}
	} else {
		// 프리픽스 기반 조회
		prefixBytes := []byte(prefix)
		for k, _ := cursor.Seek(prefixBytes); k != nil && strings.HasPrefix(string(k), prefix); k, _ = cursor.Next() {
			keys = append(keys, string(k))
		}
	}
	
	return keys, nil
}

// Statistics 인덱스 통계 정보
type Statistics struct {
	TotalKeys   int `json:"total_keys"`
	TotalValues int `json:"total_values"`
	AvgValues   float64 `json:"avg_values"`
}

// GetIndexStats 인덱스 통계 조회
func (im *IndexManager) GetIndexStats(tx *bbolt.Tx, index Index) (*Statistics, error) {
	bucket := tx.Bucket([]byte(index.BucketName))
	if bucket == nil {
		return nil, fmt.Errorf("인덱스 버킷 '%s'이 존재하지 않습니다", index.BucketName)
	}
	
	stats := &Statistics{}
	cursor := bucket.Cursor()
	
	for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
		stats.TotalKeys++
		
		switch index.Type {
		case IndexTypeUnique:
			stats.TotalValues++
		case IndexTypeMultiple:
			values, err := im.serializer.UnmarshalStringList(v)
			if err != nil {
				continue
			}
			stats.TotalValues += len(values)
		}
	}
	
	if stats.TotalKeys > 0 {
		stats.AvgValues = float64(stats.TotalValues) / float64(stats.TotalKeys)
	}
	
	return stats, nil
}

// 내부 헬퍼 메서드들

// addUniqueIndex 유니크 인덱스에 추가
func (im *IndexManager) addUniqueIndex(bucket *bbolt.Bucket, key, value string) error {
	// 기존 값이 있는지 확인
	existing := bucket.Get([]byte(key))
	if existing != nil {
		return fmt.Errorf("인덱스 키 '%s'가 이미 존재합니다 (기존 값: %s)", key, string(existing))
	}
	
	return bucket.Put([]byte(key), []byte(value))
}

// addMultipleIndex 다중 인덱스에 추가
func (im *IndexManager) addMultipleIndex(bucket *bbolt.Bucket, key, value string) error {
	// 기존 값들 조회
	existing := bucket.Get([]byte(key))
	var values []string
	
	if existing != nil {
		var err error
		values, err = im.serializer.UnmarshalStringList(existing)
		if err != nil {
			return fmt.Errorf("기존 인덱스 값 파싱 실패: %w", err)
		}
	}
	
	// 중복 확인 및 추가
	for _, v := range values {
		if v == value {
			// 이미 존재하는 값이면 추가하지 않음
			return nil
		}
	}
	
	values = append(values, value)
	
	// 정렬
	sort.Strings(values)
	
	// 직렬화 및 저장
	data, err := im.serializer.MarshalStringList(values)
	if err != nil {
		return fmt.Errorf("인덱스 값 직렬화 실패: %w", err)
	}
	
	return bucket.Put([]byte(key), data)
}

// removeUniqueIndex 유니크 인덱스에서 제거
func (im *IndexManager) removeUniqueIndex(bucket *bbolt.Bucket, key string) error {
	return bucket.Delete([]byte(key))
}

// removeMultipleIndex 다중 인덱스에서 특정 값 제거
func (im *IndexManager) removeMultipleIndex(bucket *bbolt.Bucket, key, value string) error {
	// 기존 값들 조회
	existing := bucket.Get([]byte(key))
	if existing == nil {
		// 키가 없으면 제거할 것도 없음
		return nil
	}
	
	values, err := im.serializer.UnmarshalStringList(existing)
	if err != nil {
		return fmt.Errorf("기존 인덱스 값 파싱 실패: %w", err)
	}
	
	// 해당 값 제거
	var newValues []string
	for _, v := range values {
		if v != value {
			newValues = append(newValues, v)
		}
	}
	
	// 값이 모두 제거되었으면 키 자체를 삭제
	if len(newValues) == 0 {
		return bucket.Delete([]byte(key))
	}
	
	// 정렬
	sort.Strings(newValues)
	
	// 직렬화 및 저장
	data, err := im.serializer.MarshalStringList(newValues)
	if err != nil {
		return fmt.Errorf("인덱스 값 직렬화 실패: %w", err)
	}
	
	return bucket.Put([]byte(key), data)
}

// UpdateIndexValue 인덱스 값 업데이트 (제거 후 추가)
func (im *IndexManager) UpdateIndexValue(tx *bbolt.Tx, index Index, key, oldValue, newValue string) error {
	if oldValue == newValue {
		return nil
	}
	
	// 기존 값 제거
	if oldValue != "" {
		if err := im.RemoveFromIndex(tx, index, key, oldValue); err != nil {
			return fmt.Errorf("기존 인덱스 값 제거 실패: %w", err)
		}
	}
	
	// 새 값 추가
	if newValue != "" {
		if err := im.AddToIndex(tx, index, key, newValue); err != nil {
			return fmt.Errorf("새 인덱스 값 추가 실패: %w", err)
		}
	}
	
	return nil
}

// BatchUpdate 배치 업데이트
func (im *IndexManager) BatchUpdate(tx *bbolt.Tx, updates []IndexUpdate) error {
	for _, update := range updates {
		switch update.Operation {
		case IndexOpAdd:
			if err := im.AddToIndex(tx, update.Index, update.Key, update.Value); err != nil {
				return fmt.Errorf("인덱스 추가 실패 (키: %s, 값: %s): %w", update.Key, update.Value, err)
			}
		case IndexOpRemove:
			if err := im.RemoveFromIndex(tx, update.Index, update.Key, update.Value); err != nil {
				return fmt.Errorf("인덱스 제거 실패 (키: %s, 값: %s): %w", update.Key, update.Value, err)
			}
		case IndexOpUpdate:
			if err := im.UpdateIndexValue(tx, update.Index, update.Key, update.OldValue, update.Value); err != nil {
				return fmt.Errorf("인덱스 업데이트 실패 (키: %s): %w", update.Key, err)
			}
		default:
			return fmt.Errorf("알 수 없는 인덱스 연산: %s", update.Operation)
		}
	}
	
	return nil
}

// IndexOperation 인덱스 연산 타입
type IndexOperation string

const (
	IndexOpAdd    IndexOperation = "add"
	IndexOpRemove IndexOperation = "remove"
	IndexOpUpdate IndexOperation = "update"
)

// IndexUpdate 인덱스 업데이트 정보
type IndexUpdate struct {
	Index     Index          // 대상 인덱스
	Operation IndexOperation // 연산 타입
	Key       string         // 인덱스 키
	Value     string         // 인덱스 값
	OldValue  string         // 기존 값 (Update 시)
}