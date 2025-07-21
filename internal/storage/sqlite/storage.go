package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
	
	_ "github.com/mattn/go-sqlite3" // SQLite 드라이버
	"go.uber.org/zap"
	
	"github.com/aicli/aicli-web/internal/storage"
)

// Storage SQLite 기반 스토리지 구현
type Storage struct {
	db        *sql.DB
	stmtCache map[string]*sql.Stmt
	mu        sync.RWMutex
	logger    *zap.Logger
	
	// 스토리지 구현체들
	workspace *workspaceStorage
	project   *projectStorage
	session   *sessionStorage
	task      *taskStorage
	
	// 최적화 도구들
	indexManager *IndexManager
}

// Config SQLite 설정
type Config struct {
	DataSource      string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PragmaOptions   map[string]string
	Logger          *zap.Logger
}

// DefaultConfig 기본 설정 반환
func DefaultConfig() Config {
	return Config{
		DataSource:      ":memory:",
		MaxOpenConns:    1, // SQLite는 단일 쓰기 연결
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 10,
		PragmaOptions: map[string]string{
			"journal_mode":    "WAL",     // Write-Ahead Logging 모드
			"foreign_keys":    "ON",      // 외래키 제약조건 활성화
			"synchronous":     "NORMAL",  // 동기화 모드
			"cache_size":      "-64000",  // 64MB 캐시
			"temp_store":      "MEMORY",  // 임시 저장소를 메모리로
			"mmap_size":       "67108864", // 64MB 메모리 맵 크기
		},
	}
}

// New SQLite 스토리지 생성
func New(config Config) (*Storage, error) {
	// 설정 검증
	if config.DataSource == "" {
		return nil, fmt.Errorf("데이터 소스가 비어있습니다")
	}
	
	// SQLite 연결
	db, err := sql.Open("sqlite3", config.DataSource)
	if err != nil {
		return nil, fmt.Errorf("SQLite 연결 실패: %w", err)
	}
	
	// 연결 풀 설정
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	
	// 연결 테스트
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("SQLite 연결 테스트 실패: %w", err)
	}
	
	storage := &Storage{
		db:        db,
		stmtCache: make(map[string]*sql.Stmt),
		logger:    config.Logger,
	}
	
	if storage.logger == nil {
		storage.logger = zap.NewNop()
	}
	
	// PRAGMA 설정 적용
	if err := storage.applyPragmaOptions(config.PragmaOptions); err != nil {
		return nil, fmt.Errorf("PRAGMA 설정 적용 실패: %w", err)
	}
	
	// 스토리지 구현체들 초기화
	storage.workspace = newWorkspaceStorage(storage)
	storage.project = newProjectStorage(storage)
	storage.session = newSessionStorage(storage)
	storage.task = newTaskStorage(storage)
	
	// 최적화 도구들 초기화
	storage.indexManager = newIndexManager(storage)
	
	return storage, nil
}

// NewFromDataSource 데이터 소스 문자열로 SQLite 스토리지 생성
func NewFromDataSource(dataSource string) (*Storage, error) {
	config := DefaultConfig()
	config.DataSource = dataSource
	return New(config)
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}
	
	return newTransaction(tx, s), nil
}

// WithTx 트랜잭션 내에서 작업 실행
func (s *Storage) WithTx(ctx context.Context, fn func(tx storage.Transaction) error) error {
	tx, err := s.BeginTx(ctx)
	if err != nil {
		return err
	}
	
	defer func() {
		if !tx.IsClosed() {
			if err != nil {
				tx.Rollback()
			} else {
				tx.Commit()
			}
		}
	}()
	
	err = fn(tx)
	return err
}

// Close 스토리지 연결 종료
func (s *Storage) Close() error {
	// Prepared Statement 정리
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, stmt := range s.stmtCache {
		stmt.Close()
	}
	s.stmtCache = make(map[string]*sql.Stmt)
	
	// 데이터베이스 연결 종료
	return s.db.Close()
}

// prepareStmt Prepared Statement 준비 및 캐싱
func (s *Storage) prepareStmt(ctx context.Context, query string) (*sql.Stmt, error) {
	// 캐시에서 먼저 확인
	s.mu.RLock()
	if stmt, ok := s.stmtCache[query]; ok {
		s.mu.RUnlock()
		return stmt, nil
	}
	s.mu.RUnlock()
	
	// 새 statement 준비
	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("statement 준비 실패: %w", err)
	}
	
	// 캐시에 저장
	s.mu.Lock()
	s.stmtCache[query] = stmt
	s.mu.Unlock()
	
	return stmt, nil
}

// execContext Context를 지원하는 실행
func (s *Storage) execContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	stmt, err := s.prepareStmt(ctx, query)
	if err != nil {
		return nil, err
	}
	
	result, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		return nil, storage.ConvertError(err, "exec", "sqlite")
	}
	
	return result, nil
}

// queryContext Context를 지원하는 쿼리
func (s *Storage) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := s.prepareStmt(ctx, query)
	if err != nil {
		return nil, err
	}
	
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, storage.ConvertError(err, "query", "sqlite")
	}
	
	return rows, nil
}

// queryRowContext Context를 지원하는 단일 행 쿼리
func (s *Storage) queryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := s.prepareStmt(ctx, query)
	if err != nil {
		// sql.Row는 에러를 나중에 Scan에서 처리
		return s.db.QueryRowContext(ctx, query, args...)
	}
	
	return stmt.QueryRowContext(ctx, args...)
}

// applyPragmaOptions PRAGMA 옵션들 적용
func (s *Storage) applyPragmaOptions(options map[string]string) error {
	for key, value := range options {
		pragmaSQL := fmt.Sprintf("PRAGMA %s = %s", key, value)
		_, err := s.db.Exec(pragmaSQL)
		if err != nil {
			return fmt.Errorf("PRAGMA %s 설정 실패: %w", key, err)
		}
	}
	return nil
}

// Health 건강성 확인
func (s *Storage) Health(ctx context.Context) error {
	// 간단한 쿼리로 연결 상태 확인
	var result int
	query := "SELECT 1"
	err := s.db.QueryRowContext(ctx, query).Scan(&result)
	if err != nil {
		return fmt.Errorf("건강성 확인 실패: %w", err)
	}
	
	if result != 1 {
		return fmt.Errorf("건강성 확인 결과 불일치: %d", result)
	}
	
	return nil
}

// Stats 스토리지 통계 반환
func (s *Storage) Stats() sql.DBStats {
	return s.db.Stats()
}

// getDB 내부 데이터베이스 연결 반환 (테스트용)
func (s *Storage) getDB() *sql.DB {
	return s.db
}

// clearStmtCache Statement 캐시 정리 (테스트용)
func (s *Storage) clearStmtCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, stmt := range s.stmtCache {
		stmt.Close()
	}
	s.stmtCache = make(map[string]*sql.Stmt)
}

// IndexManager 인덱스 관리자 반환
func (s *Storage) IndexManager() *IndexManager {
	return s.indexManager
}

// OptimizeIndexes 모든 테이블의 인덱스 최적화
func (s *Storage) OptimizeIndexes(ctx context.Context) error {
	tables := []string{"workspaces", "projects", "sessions", "tasks"}
	
	for _, table := range tables {
		s.logger.Info("테이블 인덱스 분석 시작", zap.String("table", table))
		
		analysis, err := s.indexManager.AnalyzeTable(ctx, table)
		if err != nil {
			s.logger.Error("테이블 인덱스 분석 실패", 
				zap.String("table", table),
				zap.Error(err),
			)
			continue
		}
		
		// 높은 우선순위 제안만 적용
		if len(analysis.SuggestedIndexes) > 0 {
			err = s.indexManager.ApplyIndexSuggestions(ctx, analysis.SuggestedIndexes, []string{"high"})
			if err != nil {
				s.logger.Error("인덱스 제안 적용 실패",
					zap.String("table", table),
					zap.Error(err),
				)
			}
		}
	}
	
	return nil
}

// GetIndexAnalysis 인덱스 분석 결과 반환
func (s *Storage) GetIndexAnalysis(ctx context.Context) (map[string]*IndexAnalysis, error) {
	tables := []string{"workspaces", "projects", "sessions", "tasks"}
	results := make(map[string]*IndexAnalysis)
	
	for _, table := range tables {
		analysis, err := s.indexManager.AnalyzeTable(ctx, table)
		if err != nil {
			s.logger.Error("테이블 인덱스 분석 실패", 
				zap.String("table", table),
				zap.Error(err),
			)
			continue
		}
		results[table] = analysis
	}
	
	return results, nil
}