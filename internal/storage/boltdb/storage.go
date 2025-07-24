package boltdb

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.etcd.io/bbolt"

	"github.com/aicli/aicli-web/internal/storage"
)

// 버킷 이름 정의
const (
	// 메인 데이터 버킷 (lowercase for internal use)
	bucketWorkspaces = "workspaces"
	bucketProjects   = "projects" 
	bucketSessions   = "sessions"
	bucketTasks      = "tasks"
	
	// 인덱스 버킷 (lowercase for internal use)
	bucketIndexOwner     = "idx_owner"      // owner_id -> workspace_ids
	bucketIndexWorkspace = "idx_workspace"  // workspace_id -> project_ids
	bucketIndexProject   = "idx_project"    // project_id -> session_ids
	bucketIndexSession   = "idx_session"    // session_id -> task_ids
	bucketIndexStatus    = "idx_status"     // status -> entity_ids
	bucketIndexPath      = "idx_path"       // path -> project_id
	bucketIndexName      = "idx_name"       // owner_id:name -> workspace_id
	
	// 메인 데이터 버킷 (uppercase for external access - backward compatibility)
	BucketWorkspaces = bucketWorkspaces
	BucketProjects   = bucketProjects
	BucketSessions   = bucketSessions
	BucketTasks      = bucketTasks
	
	// 인덱스 버킷 (uppercase for external access - backward compatibility) 
	BucketIndexOwner     = bucketIndexOwner
	BucketIndexWorkspace = bucketIndexWorkspace
	BucketIndexProject   = bucketIndexProject
	BucketIndexSession   = bucketIndexSession
	BucketIndexStatus    = bucketIndexStatus
	BucketIndexPath      = bucketIndexPath
	BucketIndexName      = bucketIndexName
)

// 모든 필요한 버킷 리스트
var requiredBuckets = []string{
	bucketWorkspaces,
	bucketProjects,
	bucketSessions,
	bucketTasks,
	bucketIndexOwner,
	bucketIndexWorkspace,
	bucketIndexProject,
	bucketIndexSession,
	bucketIndexStatus,
	bucketIndexPath,
	bucketIndexName,
}

// Storage BoltDB 기반 스토리지 구현
type Storage struct {
	db   *bbolt.DB
	path string
	
	// 유틸리티 도구들
	serializer *Serializer
	indexer    *IndexManager
	querier    *QueryHelper
	
	// 스토리지 구현체들
	workspace *workspaceStorage
	project   *projectStorage
	session   *sessionStorage
	task      *taskStorage
	
	// 최적화 도구들
	batchProcessor *BatchProcessor
	bulkImporter   *BulkImporter
}

// Config BoltDB 설정
type Config struct {
	Path        string
	Mode        uint32
	Options     *bbolt.Options
	BatchConfig BatchConfig
}

// DefaultConfig 기본 설정 반환
func DefaultConfig(path string) Config {
	return Config{
		Path: path,
		Mode: 0600,
		Options: &bbolt.Options{
			Timeout: 1 * time.Second,
		},
		BatchConfig: DefaultBatchConfig(),
	}
}

// New BoltDB 스토리지 생성
func New(config Config) (*Storage, error) {
	// 설정 검증
	if config.Path == "" {
		return nil, fmt.Errorf("데이터베이스 경로가 비어있습니다")
	}
	
	// BoltDB 열기
	db, err := bbolt.Open(config.Path, os.FileMode(config.Mode), config.Options)
	if err != nil {
		return nil, fmt.Errorf("BoltDB 열기 실패: %w", err)
	}
	
	storage := &Storage{
		db:   db,
		path: config.Path,
	}
	
	// 필요한 버킷들 생성
	if err := storage.initBuckets(); err != nil {
		db.Close()
		return nil, fmt.Errorf("버킷 초기화 실패: %w", err)
	}
	
	// 유틸리티 도구들 초기화
	storage.serializer = NewSerializer()
	storage.indexer = NewIndexManager()
	storage.querier = NewQueryHelper()
	
	// 스토리지 구현체들 초기화
	storage.workspace = newWorkspaceStorage(storage)
	storage.project = newProjectStorage(storage)
	storage.session = newSessionStorage(storage)
	storage.task = newTaskStorage(storage)
	
	// 최적화 도구들 초기화
	storage.batchProcessor = NewBatchProcessor(storage, config.BatchConfig)
	storage.bulkImporter = NewBulkImporter(storage, DefaultBulkImportConfig())
	
	return storage, nil
}

// NewFromPath 경로로 BoltDB 스토리지 생성
func NewFromPath(path string) (*Storage, error) {
	config := DefaultConfig(path)
	return New(config)
}

// initBuckets 필요한 버킷들 초기화
func (s *Storage) initBuckets() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		for _, bucketName := range requiredBuckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return fmt.Errorf("버킷 '%s' 생성 실패: %w", bucketName, err)
			}
		}
		return nil
	})
}

// Workspace 워크스페이스 스토리지 반환
func (s *Storage) Workspace() storage.WorkspaceStorage {
	return s.workspace
}

// Project 프로젝트 스토리지 반환
func (s *Storage) Project() storage.ProjectStorage {
	return s.project
}

// Session 세션 스토리지 반환
func (s *Storage) Session() storage.SessionStorage {
	return s.session
}

// Task 태스크 스토리지 반환
func (s *Storage) Task() storage.TaskStorage {
	return s.task
}

// BeginTx 트랜잭션 시작
func (s *Storage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	// BoltDB는 트랜잭션을 미리 시작할 수 없으므로 
	// 트랜잭션 래퍼만 반환하고 실제 트랜잭션은 사용 시 생성
	return newTransaction(s), nil
}

// WithTx 트랜잭션 내에서 작업 실행
func (s *Storage) WithTx(ctx context.Context, fn func(tx storage.Transaction) error) error {
	transaction := newTransaction(s)
	
	// BoltDB Update 트랜잭션으로 실행
	return s.db.Update(func(boltTx *bbolt.Tx) error {
		// 트랜잭션에 BoltDB 트랜잭션 설정
		transaction.setBoltTx(boltTx)
		defer transaction.reset()
		
		// 사용자 함수 실행
		return fn(transaction)
	})
}

// View 읽기 전용 트랜잭션 실행
func (s *Storage) View(fn func(*bbolt.Tx) error) error {
	return s.db.View(fn)
}

// Update 읽기/쓰기 트랜잭션 실행
func (s *Storage) Update(fn func(*bbolt.Tx) error) error {
	return s.db.Update(fn)
}

// Close 스토리지 연결 종료
func (s *Storage) Close() error {
	// 배치 처리기 종료
	if s.batchProcessor != nil {
		s.batchProcessor.Close()
	}
	
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Stats BoltDB 통계 반환
func (s *Storage) Stats() bbolt.Stats {
	return s.db.Stats()
}

// Sync 데이터베이스 동기화
func (s *Storage) Sync() error {
	return s.db.Sync()
}

// Path 데이터베이스 파일 경로 반환
func (s *Storage) Path() string {
	return s.path
}

// IsReadOnly 읽기 전용 모드 여부 반환
func (s *Storage) IsReadOnly() bool {
	return s.db.IsReadOnly()
}

// Health 건강성 확인
func (s *Storage) Health(ctx context.Context) error {
	// 간단한 읽기 작업으로 연결 상태 확인
	return s.db.View(func(tx *bbolt.Tx) error {
		// 워크스페이스 버킷 존재 확인
		bucket := tx.Bucket([]byte(BucketWorkspaces))
		if bucket == nil {
			return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
		}
		return nil
	})
}

// Compact 데이터베이스 압축 (BoltDB에서는 지원하지 않음)
func (s *Storage) Compact() error {
	return fmt.Errorf("BoltDB는 온라인 압축을 지원하지 않습니다")
}

// Backup 데이터베이스 백업
func (s *Storage) Backup(path string) error {
	return s.db.View(func(tx *bbolt.Tx) error {
		return tx.CopyFile(path, 0600)
	})
}

// BatchProcessor 배치 처리기 반환
func (s *Storage) BatchProcessor() *BatchProcessor {
	return s.batchProcessor
}

// BulkImporter 대량 가져오기 반환
func (s *Storage) BulkImporter() *BulkImporter {
	return s.bulkImporter
}

// GetBatchWriters 배치 라이터들 반환
func (s *Storage) GetBatchWriters() *BatchWriters {
	return &BatchWriters{
		Workspace: s.batchProcessor.NewBatchWriter(BucketWorkspaces, &WorkspaceSerializer{}),
		Project:   s.batchProcessor.NewBatchWriter(BucketProjects, &ProjectSerializer{}),
		Session:   s.batchProcessor.NewBatchWriter(BucketSessions, &SessionSerializer{}),
		Task:      s.batchProcessor.NewBatchWriter(BucketTasks, &TaskSerializer{}),
	}
}

// BatchWriters 배치 라이터 모음
type BatchWriters struct {
	Workspace *BatchWriter
	Project   *BatchWriter
	Session   *BatchWriter
	Task      *BatchWriter
}

// FlushAndSync 모든 배치 처리 및 동기화
func (s *Storage) FlushAndSync(timeout time.Duration) error {
	// 배치 처리 완료 대기
	if s.batchProcessor != nil {
		if err := s.batchProcessor.FlushAll(timeout); err != nil {
			return fmt.Errorf("배치 플러시 실패: %w", err)
		}
	}
	
	// 디스크 동기화
	return s.Sync()
}