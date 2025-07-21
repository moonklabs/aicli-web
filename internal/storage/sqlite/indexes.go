package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// IndexManager SQLite 인덱스 관리자
type IndexManager struct {
	storage *Storage
	logger  *zap.Logger
}

// IndexInfo 인덱스 정보
type IndexInfo struct {
	Name        string   `json:"name"`
	TableName   string   `json:"table_name"`
	Columns     []string `json:"columns"`
	Unique      bool     `json:"unique"`
	Partial     bool     `json:"partial"`
	WhereClause string   `json:"where_clause,omitempty"`
	SQLDefinition string `json:"sql_definition"`
}

// IndexStats 인덱스 통계
type IndexStats struct {
	Name     string `json:"name"`
	Table    string `json:"table"`
	Size     int64  `json:"size"`      // 페이지 수
	Entries  int64  `json:"entries"`   // 엔트리 수  
	Used     bool   `json:"used"`      // 사용 여부
	LastUsed *time.Time `json:"last_used,omitempty"`
}

// IndexAnalysis 인덱스 분석 결과
type IndexAnalysis struct {
	TableName       string      `json:"table_name"`
	ExistingIndexes []IndexInfo `json:"existing_indexes"`
	SuggestedIndexes []SuggestedIndex `json:"suggested_indexes"`
	UnusedIndexes   []string    `json:"unused_indexes"`
	QueryPatterns   []QueryPattern `json:"query_patterns"`
}

// SuggestedIndex 제안된 인덱스
type SuggestedIndex struct {
	Name        string   `json:"name"`
	Columns     []string `json:"columns"`
	Reason      string   `json:"reason"`
	Priority    string   `json:"priority"` // high, medium, low
	SQLCommand  string   `json:"sql_command"`
	EstimatedBenefit string `json:"estimated_benefit"`
}

// QueryPattern 쿼리 패턴 분석
type QueryPattern struct {
	Pattern     string `json:"pattern"`
	Frequency   int    `json:"frequency"`
	AvgDuration time.Duration `json:"avg_duration"`
	Columns     []string `json:"columns"`
}

// newIndexManager 새 인덱스 매니저 생성
func newIndexManager(storage *Storage) *IndexManager {
	logger := zap.NewNop()
	if storage.logger != nil {
		logger = storage.logger
	}
	
	return &IndexManager{
		storage: storage,
		logger:  logger,
	}
}

// CreateIndex 인덱스 생성
func (im *IndexManager) CreateIndex(ctx context.Context, indexDef IndexDefinition) error {
	// 인덱스 존재 확인
	exists, err := im.IndexExists(ctx, indexDef.Name)
	if err != nil {
		return fmt.Errorf("인덱스 존재 확인 실패: %w", err)
	}
	
	if exists {
		im.logger.Warn("인덱스가 이미 존재합니다",
			zap.String("index_name", indexDef.Name),
		)
		return nil
	}
	
	// SQL 생성
	sql := im.buildCreateIndexSQL(indexDef)
	
	im.logger.Info("인덱스 생성 시작",
		zap.String("index_name", indexDef.Name),
		zap.String("sql", sql),
	)
	
	start := time.Now()
	
	// 인덱스 생성 실행
	_, err = im.storage.execContext(ctx, sql)
	if err != nil {
		im.logger.Error("인덱스 생성 실패",
			zap.String("index_name", indexDef.Name),
			zap.String("sql", sql),
			zap.Error(err),
		)
		return fmt.Errorf("인덱스 생성 실패: %w", err)
	}
	
	duration := time.Since(start)
	
	im.logger.Info("인덱스 생성 완료",
		zap.String("index_name", indexDef.Name),
		zap.Duration("duration", duration),
	)
	
	return nil
}

// IndexDefinition 인덱스 정의
type IndexDefinition struct {
	Name        string
	TableName   string
	Columns     []string
	Unique      bool
	IfNotExists bool
	Where       string
}

// buildCreateIndexSQL CREATE INDEX SQL 생성
func (im *IndexManager) buildCreateIndexSQL(def IndexDefinition) string {
	var sql strings.Builder
	
	sql.WriteString("CREATE")
	if def.Unique {
		sql.WriteString(" UNIQUE")
	}
	sql.WriteString(" INDEX")
	
	if def.IfNotExists {
		sql.WriteString(" IF NOT EXISTS")
	}
	
	sql.WriteString(" ")
	sql.WriteString(def.Name)
	sql.WriteString(" ON ")
	sql.WriteString(def.TableName)
	sql.WriteString("(")
	sql.WriteString(strings.Join(def.Columns, ", "))
	sql.WriteString(")")
	
	if def.Where != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(def.Where)
	}
	
	return sql.String()
}

// DropIndex 인덱스 삭제
func (im *IndexManager) DropIndex(ctx context.Context, indexName string) error {
	// 인덱스 존재 확인
	exists, err := im.IndexExists(ctx, indexName)
	if err != nil {
		return fmt.Errorf("인덱스 존재 확인 실패: %w", err)
	}
	
	if !exists {
		im.logger.Warn("삭제하려는 인덱스가 존재하지 않습니다",
			zap.String("index_name", indexName),
		)
		return nil
	}
	
	sql := fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
	
	im.logger.Info("인덱스 삭제 시작",
		zap.String("index_name", indexName),
		zap.String("sql", sql),
	)
	
	_, err = im.storage.execContext(ctx, sql)
	if err != nil {
		im.logger.Error("인덱스 삭제 실패",
			zap.String("index_name", indexName),
			zap.String("sql", sql),
			zap.Error(err),
		)
		return fmt.Errorf("인덱스 삭제 실패: %w", err)
	}
	
	im.logger.Info("인덱스 삭제 완료",
		zap.String("index_name", indexName),
	)
	
	return nil
}

// IndexExists 인덱스 존재 여부 확인
func (im *IndexManager) IndexExists(ctx context.Context, indexName string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM sqlite_master 
		WHERE type = 'index' AND name = ?
	`
	
	var count int
	err := im.storage.queryRowContext(ctx, query, indexName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("인덱스 존재 확인 쿼리 실패: %w", err)
	}
	
	return count > 0, nil
}

// GetIndexInfo 인덱스 정보 조회
func (im *IndexManager) GetIndexInfo(ctx context.Context, indexName string) (*IndexInfo, error) {
	query := `
		SELECT name, tbl_name, sql
		FROM sqlite_master 
		WHERE type = 'index' AND name = ?
	`
	
	var info IndexInfo
	var sqlDef sql.NullString
	
	err := im.storage.queryRowContext(ctx, query, indexName).Scan(
		&info.Name,
		&info.TableName,
		&sqlDef,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("인덱스를 찾을 수 없습니다: %s", indexName)
		}
		return nil, fmt.Errorf("인덱스 정보 조회 실패: %w", err)
	}
	
	if sqlDef.Valid {
		info.SQLDefinition = sqlDef.String
		
		// SQL에서 추가 정보 파싱
		im.parseIndexSQL(sqlDef.String, &info)
	}
	
	return &info, nil
}

// parseIndexSQL SQL에서 인덱스 정보 파싱
func (im *IndexManager) parseIndexSQL(sql string, info *IndexInfo) {
	sqlLower := strings.ToLower(sql)
	
	// UNIQUE 여부 확인
	info.Unique = strings.Contains(sqlLower, "unique")
	
	// WHERE 절 확인
	if whereIdx := strings.Index(sqlLower, " where "); whereIdx != -1 {
		info.Partial = true
		info.WhereClause = strings.TrimSpace(sql[whereIdx+7:])
	}
	
	// 컬럼 추출
	startIdx := strings.Index(sqlLower, "(")
	endIdx := strings.Index(sqlLower, ")")
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		columnStr := sql[startIdx+1 : endIdx]
		columns := strings.Split(columnStr, ",")
		for i, col := range columns {
			columns[i] = strings.TrimSpace(col)
		}
		info.Columns = columns
	}
}

// ListIndexes 테이블의 모든 인덱스 목록
func (im *IndexManager) ListIndexes(ctx context.Context, tableName string) ([]IndexInfo, error) {
	query := `
		SELECT name, tbl_name, sql
		FROM sqlite_master 
		WHERE type = 'index' AND tbl_name = ?
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`
	
	rows, err := im.storage.queryContext(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("인덱스 목록 조회 실패: %w", err)
	}
	defer rows.Close()
	
	var indexes []IndexInfo
	
	for rows.Next() {
		var info IndexInfo
		var sqlDef sql.NullString
		
		err := rows.Scan(&info.Name, &info.TableName, &sqlDef)
		if err != nil {
			im.logger.Warn("인덱스 정보 스캔 실패", zap.Error(err))
			continue
		}
		
		if sqlDef.Valid {
			info.SQLDefinition = sqlDef.String
			im.parseIndexSQL(sqlDef.String, &info)
		}
		
		indexes = append(indexes, info)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("인덱스 목록 처리 실패: %w", err)
	}
	
	return indexes, nil
}

// GetIndexStats 인덱스 통계 조회
func (im *IndexManager) GetIndexStats(ctx context.Context, indexName string) (*IndexStats, error) {
	// PRAGMA index_info로 기본 정보 조회
	infoQuery := fmt.Sprintf("PRAGMA index_info('%s')", indexName)
	
	rows, err := im.storage.queryContext(ctx, infoQuery)
	if err != nil {
		return nil, fmt.Errorf("인덱스 정보 조회 실패: %w", err)
	}
	defer rows.Close()
	
	var entryCount int64
	for rows.Next() {
		var seqno, cid int
		var name string
		err := rows.Scan(&seqno, &cid, &name)
		if err == nil {
			entryCount++
		}
	}
	
	// 테이블명 조회
	var tableName string
	tableQuery := `
		SELECT tbl_name 
		FROM sqlite_master 
		WHERE type = 'index' AND name = ?
	`
	err = im.storage.queryRowContext(ctx, tableQuery, indexName).Scan(&tableName)
	if err != nil {
		return nil, fmt.Errorf("인덱스 테이블명 조회 실패: %w", err)
	}
	
	stats := &IndexStats{
		Name:    indexName,
		Table:   tableName,
		Entries: entryCount,
		Used:    true, // SQLite에서는 사용 통계를 직접 제공하지 않음
	}
	
	return stats, nil
}

// AnalyzeTable 테이블 인덱스 분석
func (im *IndexManager) AnalyzeTable(ctx context.Context, tableName string) (*IndexAnalysis, error) {
	analysis := &IndexAnalysis{
		TableName: tableName,
	}
	
	// 기존 인덱스 조회
	existingIndexes, err := im.ListIndexes(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("기존 인덱스 조회 실패: %w", err)
	}
	analysis.ExistingIndexes = existingIndexes
	
	// 테이블 스키마 분석하여 제안된 인덱스 생성
	suggestedIndexes, err := im.generateIndexSuggestions(ctx, tableName, existingIndexes)
	if err != nil {
		im.logger.Warn("인덱스 제안 생성 실패", zap.Error(err))
	} else {
		analysis.SuggestedIndexes = suggestedIndexes
	}
	
	return analysis, nil
}

// generateIndexSuggestions 인덱스 제안 생성
func (im *IndexManager) generateIndexSuggestions(ctx context.Context, tableName string, existing []IndexInfo) ([]SuggestedIndex, error) {
	var suggestions []SuggestedIndex
	
	// 기존 인덱스를 맵으로 변환
	existingMap := make(map[string]bool)
	for _, idx := range existing {
		key := fmt.Sprintf("%s_%s", idx.TableName, strings.Join(idx.Columns, "_"))
		existingMap[key] = true
	}
	
	// 테이블별 일반적인 인덱스 패턴 제안
	suggestions = append(suggestions, im.getTableSpecificSuggestions(tableName, existingMap)...)
	
	return suggestions, nil
}

// getTableSpecificSuggestions 테이블별 특정 제안
func (im *IndexManager) getTableSpecificSuggestions(tableName string, existing map[string]bool) []SuggestedIndex {
	var suggestions []SuggestedIndex
	
	switch tableName {
	case "workspaces":
		// owner_id + status 복합 인덱스
		if !existing["workspaces_owner_id_status"] {
			suggestions = append(suggestions, SuggestedIndex{
				Name:    "idx_workspaces_owner_status_optimized",
				Columns: []string{"owner_id", "status"},
				Reason:  "사용자별 워크스페이스 조회 쿼리 최적화",
				Priority: "high",
				SQLCommand: "CREATE INDEX IF NOT EXISTS idx_workspaces_owner_status_optimized ON workspaces(owner_id, status) WHERE deleted_at IS NULL",
				EstimatedBenefit: "사용자별 워크스페이스 목록 조회 성능 향상",
			})
		}
		
		// 생성일 기준 인덱스
		if !existing["workspaces_created_at"] {
			suggestions = append(suggestions, SuggestedIndex{
				Name:    "idx_workspaces_created_at",
				Columns: []string{"created_at"},
				Reason:  "최신 워크스페이스 조회 최적화",
				Priority: "medium",
				SQLCommand: "CREATE INDEX IF NOT EXISTS idx_workspaces_created_at ON workspaces(created_at DESC)",
				EstimatedBenefit: "시간 순 정렬 쿼리 성능 향상",
			})
		}
		
	case "projects":
		// workspace_id + name 복합 인덱스
		if !existing["projects_workspace_id_name"] {
			suggestions = append(suggestions, SuggestedIndex{
				Name:    "idx_projects_workspace_name",
				Columns: []string{"workspace_id", "name"},
				Reason:  "워크스페이스 내 프로젝트 이름 검색 최적화",
				Priority: "high",
				SQLCommand: "CREATE INDEX IF NOT EXISTS idx_projects_workspace_name ON projects(workspace_id, name) WHERE deleted_at IS NULL",
				EstimatedBenefit: "프로젝트 이름 기반 검색 성능 향상",
			})
		}
		
		// path 인덱스
		if !existing["projects_path"] {
			suggestions = append(suggestions, SuggestedIndex{
				Name:    "idx_projects_path",
				Columns: []string{"path"},
				Reason:  "경로 기반 프로젝트 조회 최적화",
				Priority: "medium",
				SQLCommand: "CREATE INDEX IF NOT EXISTS idx_projects_path ON projects(path)",
				EstimatedBenefit: "경로 기반 프로젝트 검색 성능 향상",
			})
		}
		
	case "sessions":
		// project_id + status + last_active 복합 인덱스
		if !existing["sessions_project_id_status_last_active"] {
			suggestions = append(suggestions, SuggestedIndex{
				Name:    "idx_sessions_active_monitoring",
				Columns: []string{"project_id", "status", "last_active"},
				Reason:  "활성 세션 모니터링 쿼리 최적화",
				Priority: "high",
				SQLCommand: "CREATE INDEX IF NOT EXISTS idx_sessions_active_monitoring ON sessions(project_id, status, last_active) WHERE status IN ('active', 'idle')",
				EstimatedBenefit: "활성 세션 조회 및 타임아웃 처리 성능 향상",
			})
		}
		
	case "tasks":
		// session_id + created_at 복합 인덱스
		if !existing["tasks_session_id_created_at"] {
			suggestions = append(suggestions, SuggestedIndex{
				Name:    "idx_tasks_session_timeline",
				Columns: []string{"session_id", "created_at"},
				Reason:  "세션별 태스크 시간순 조회 최적화",
				Priority: "medium",
				SQLCommand: "CREATE INDEX IF NOT EXISTS idx_tasks_session_timeline ON tasks(session_id, created_at DESC)",
				EstimatedBenefit: "태스크 히스토리 조회 성능 향상",
			})
		}
		
		// status + started_at 복합 인덱스 
		if !existing["tasks_status_started_at"] {
			suggestions = append(suggestions, SuggestedIndex{
				Name:    "idx_tasks_running_duration",
				Columns: []string{"status", "started_at"},
				Reason:  "실행 중인 태스크 모니터링 최적화",
				Priority: "medium",
				SQLCommand: "CREATE INDEX IF NOT EXISTS idx_tasks_running_duration ON tasks(status, started_at) WHERE status = 'running'",
				EstimatedBenefit: "실행 시간 모니터링 성능 향상",
			})
		}
	}
	
	return suggestions
}

// ApplyIndexSuggestions 제안된 인덱스 적용
func (im *IndexManager) ApplyIndexSuggestions(ctx context.Context, suggestions []SuggestedIndex, priorities []string) error {
	if len(priorities) == 0 {
		priorities = []string{"high", "medium", "low"}
	}
	
	priorityMap := make(map[string]bool)
	for _, p := range priorities {
		priorityMap[p] = true
	}
	
	applied := 0
	for _, suggestion := range suggestions {
		if !priorityMap[suggestion.Priority] {
			continue
		}
		
		im.logger.Info("인덱스 제안 적용 중",
			zap.String("name", suggestion.Name),
			zap.String("priority", suggestion.Priority),
			zap.String("reason", suggestion.Reason),
		)
		
		_, err := im.storage.execContext(ctx, suggestion.SQLCommand)
		if err != nil {
			im.logger.Error("인덱스 제안 적용 실패",
				zap.String("name", suggestion.Name),
				zap.String("sql", suggestion.SQLCommand),
				zap.Error(err),
			)
			continue
		}
		
		applied++
		im.logger.Info("인덱스 제안 적용 완료",
			zap.String("name", suggestion.Name),
		)
	}
	
	im.logger.Info("인덱스 제안 적용 완료",
		zap.Int("total_suggestions", len(suggestions)),
		zap.Int("applied", applied),
	)
	
	return nil
}

// RebuildIndexes 인덱스 재구축
func (im *IndexManager) RebuildIndexes(ctx context.Context, tableName string) error {
	im.logger.Info("인덱스 재구축 시작",
		zap.String("table", tableName),
	)
	
	// REINDEX 명령 실행
	var sql string
	if tableName != "" {
		sql = fmt.Sprintf("REINDEX %s", tableName)
	} else {
		sql = "REINDEX"
	}
	
	start := time.Now()
	_, err := im.storage.execContext(ctx, sql)
	if err != nil {
		im.logger.Error("인덱스 재구축 실패",
			zap.String("table", tableName),
			zap.String("sql", sql),
			zap.Error(err),
		)
		return fmt.Errorf("인덱스 재구축 실패: %w", err)
	}
	
	duration := time.Since(start)
	im.logger.Info("인덱스 재구축 완료",
		zap.String("table", tableName),
		zap.Duration("duration", duration),
	)
	
	return nil
}