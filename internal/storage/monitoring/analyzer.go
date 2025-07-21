package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// QueryAnalyzer 쿼리 분석기
type QueryAnalyzer struct {
	logger *zap.Logger
}

// NewQueryAnalyzer 새 쿼리 분석기 생성
func NewQueryAnalyzer(logger *zap.Logger) *QueryAnalyzer {
	if logger == nil {
		logger = zap.NewNop()
	}
	
	return &QueryAnalyzer{
		logger: logger,
	}
}

// QueryPlan 쿼리 실행 계획
type QueryPlan struct {
	ID      int    `json:"id"`
	Parent  int    `json:"parent"`
	NotUsed int    `json:"not_used"`
	Detail  string `json:"detail"`
}

// QueryAnalysis 쿼리 분석 결과
type QueryAnalysis struct {
	Query       string      `json:"query"`
	Plans       []QueryPlan `json:"plans"`
	IndexesUsed []string    `json:"indexes_used"`
	TableScans  []string    `json:"table_scans"`
	Suggestions []string    `json:"suggestions"`
	Complexity  string      `json:"complexity"` // low, medium, high
	Cost        int         `json:"cost"`
}

// AnalyzeSQLiteQuery SQLite 쿼리 분석
func (a *QueryAnalyzer) AnalyzeSQLiteQuery(ctx context.Context, db *sql.DB, query string) (*QueryAnalysis, error) {
	// EXPLAIN QUERY PLAN 실행
	explainQuery := "EXPLAIN QUERY PLAN " + query
	rows, err := db.QueryContext(ctx, explainQuery)
	if err != nil {
		a.logger.Error("쿼리 계획 분석 실패", 
			zap.String("query", query),
			zap.Error(err),
		)
		return nil, fmt.Errorf("쿼리 계획 분석 실패: %w", err)
	}
	defer rows.Close()

	var plans []QueryPlan
	for rows.Next() {
		var plan QueryPlan
		err := rows.Scan(&plan.ID, &plan.Parent, &plan.NotUsed, &plan.Detail)
		if err != nil {
			a.logger.Warn("쿼리 계획 스캔 실패",
				zap.Error(err),
			)
			continue
		}
		plans = append(plans, plan)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("쿼리 계획 읽기 실패: %w", err)
	}

	// 분석 결과 생성
	analysis := &QueryAnalysis{
		Query: query,
		Plans: plans,
	}

	// 인덱스 사용 분석
	analysis.IndexesUsed = a.extractIndexesUsed(plans)
	
	// 테이블 스캔 분석
	analysis.TableScans = a.extractTableScans(plans)
	
	// 복잡도 계산
	analysis.Complexity = a.calculateComplexity(plans)
	
	// 비용 계산 (근사치)
	analysis.Cost = a.calculateCost(plans)
	
	// 최적화 제안
	analysis.Suggestions = a.generateSuggestions(query, plans)

	a.logger.Debug("쿼리 분석 완료",
		zap.String("query", query),
		zap.Int("plan_count", len(plans)),
		zap.Strings("indexes_used", analysis.IndexesUsed),
		zap.Strings("table_scans", analysis.TableScans),
		zap.String("complexity", analysis.Complexity),
		zap.Int("cost", analysis.Cost),
	)

	return analysis, nil
}

// extractIndexesUsed 사용된 인덱스 추출
func (a *QueryAnalyzer) extractIndexesUsed(plans []QueryPlan) []string {
	var indexes []string
	indexSet := make(map[string]bool)
	
	for _, plan := range plans {
		detail := strings.ToLower(plan.Detail)
		
		// 인덱스 사용 패턴 감지
		if strings.Contains(detail, "using index") {
			// "USING INDEX idx_name" 패턴 추출
			parts := strings.Split(detail, "using index")
			if len(parts) > 1 {
				indexName := strings.TrimSpace(parts[1])
				// 괄호나 추가 조건 제거
				if idx := strings.Index(indexName, " "); idx != -1 {
					indexName = indexName[:idx]
				}
				if idx := strings.Index(indexName, "("); idx != -1 {
					indexName = indexName[:idx]
				}
				
				if indexName != "" && !indexSet[indexName] {
					indexes = append(indexes, indexName)
					indexSet[indexName] = true
				}
			}
		}
	}
	
	return indexes
}

// extractTableScans 테이블 스캔 추출
func (a *QueryAnalyzer) extractTableScans(plans []QueryPlan) []string {
	var tableScans []string
	scanSet := make(map[string]bool)
	
	for _, plan := range plans {
		detail := strings.ToLower(plan.Detail)
		
		// 테이블 스캔 패턴 감지
		if strings.Contains(detail, "scan table") {
			// "SCAN TABLE table_name" 패턴 추출
			parts := strings.Split(detail, "scan table")
			if len(parts) > 1 {
				tableName := strings.TrimSpace(parts[1])
				// 추가 조건 제거
				if idx := strings.Index(tableName, " "); idx != -1 {
					tableName = tableName[:idx]
				}
				
				if tableName != "" && !scanSet[tableName] {
					tableScans = append(tableScans, tableName)
					scanSet[tableName] = true
				}
			}
		}
	}
	
	return tableScans
}

// calculateComplexity 복잡도 계산
func (a *QueryAnalyzer) calculateComplexity(plans []QueryPlan) string {
	planCount := len(plans)
	hasTableScan := false
	hasSubquery := false
	hasJoin := false
	
	for _, plan := range plans {
		detail := strings.ToLower(plan.Detail)
		
		if strings.Contains(detail, "scan table") {
			hasTableScan = true
		}
		if strings.Contains(detail, "subquery") || strings.Contains(detail, "correlated") {
			hasSubquery = true
		}
		if strings.Contains(detail, "join") {
			hasJoin = true
		}
	}
	
	// 복잡도 결정
	switch {
	case planCount > 10 || hasSubquery:
		return "high"
	case planCount > 5 || hasJoin || hasTableScan:
		return "medium"
	default:
		return "low"
	}
}

// calculateCost 비용 계산 (근사치)
func (a *QueryAnalyzer) calculateCost(plans []QueryPlan) int {
	cost := 0
	
	for _, plan := range plans {
		detail := strings.ToLower(plan.Detail)
		
		// 각 연산별 비용 추정
		switch {
		case strings.Contains(detail, "scan table"):
			cost += 100 // 테이블 스캔은 높은 비용
		case strings.Contains(detail, "using index"):
			cost += 10 // 인덱스 사용은 낮은 비용
		case strings.Contains(detail, "join"):
			cost += 50 // 조인은 중간 비용
		case strings.Contains(detail, "sort"):
			cost += 30 // 정렬은 중간 비용
		case strings.Contains(detail, "search"):
			cost += 20 // 검색은 중간 비용
		default:
			cost += 5 // 기본 비용
		}
	}
	
	return cost
}

// generateSuggestions 최적화 제안 생성
func (a *QueryAnalyzer) generateSuggestions(query string, plans []QueryPlan) []string {
	var suggestions []string
	queryLower := strings.ToLower(query)
	
	// 테이블 스캔 감지
	hasTableScan := false
	scannedTables := make(map[string]bool)
	
	for _, plan := range plans {
		detail := strings.ToLower(plan.Detail)
		
		if strings.Contains(detail, "scan table") {
			hasTableScan = true
			// 테이블명 추출
			parts := strings.Split(detail, "scan table")
			if len(parts) > 1 {
				tableName := strings.TrimSpace(parts[1])
				if idx := strings.Index(tableName, " "); idx != -1 {
					tableName = tableName[:idx]
				}
				scannedTables[tableName] = true
			}
		}
	}
	
	if hasTableScan {
		suggestions = append(suggestions, "테이블 스캔이 감지되었습니다. 적절한 인덱스 생성을 고려해보세요.")
		for table := range scannedTables {
			suggestions = append(suggestions, fmt.Sprintf("테이블 '%s'에 인덱스 생성을 고려해보세요.", table))
		}
	}
	
	// SELECT * 패턴 감지
	if strings.Contains(queryLower, "select *") {
		suggestions = append(suggestions, "SELECT *보다는 필요한 컬럼만 선택하는 것을 권장합니다.")
	}
	
	// ORDER BY without LIMIT 감지
	if strings.Contains(queryLower, "order by") && !strings.Contains(queryLower, "limit") {
		suggestions = append(suggestions, "ORDER BY 사용 시 LIMIT을 함께 사용하는 것을 고려해보세요.")
	}
	
	// N+1 쿼리 패턴 감지 (단순한 휴리스틱)
	if strings.Contains(queryLower, "where") && strings.Contains(queryLower, "=") {
		suggestions = append(suggestions, "N+1 쿼리 패턴이 의심됩니다. JOIN을 사용한 최적화를 고려해보세요.")
	}
	
	// 복잡한 WHERE 조건 감지
	whereCount := strings.Count(queryLower, " and ") + strings.Count(queryLower, " or ")
	if whereCount > 5 {
		suggestions = append(suggestions, "복잡한 WHERE 조건이 감지되었습니다. 복합 인덱스 생성을 고려해보세요.")
	}
	
	return suggestions
}

// BenchmarkQuery 쿼리 벤치마크
func (a *QueryAnalyzer) BenchmarkQuery(ctx context.Context, db *sql.DB, query string, iterations int) (*BenchmarkResult, error) {
	if iterations <= 0 {
		iterations = 100
	}
	
	var totalDuration time.Duration
	var minDuration = time.Hour // 충분히 큰 값으로 초기화
	var maxDuration time.Duration
	var errorCount int
	
	a.logger.Info("쿼리 벤치마크 시작",
		zap.String("query", query),
		zap.Int("iterations", iterations),
	)
	
	for i := 0; i < iterations; i++ {
		start := time.Now()
		
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			errorCount++
			continue
		}
		
		// 모든 row를 소비하여 정확한 실행 시간 측정
		for rows.Next() {
			// 실제로는 row를 읽지 않음 (벤치마크 목적)
		}
		rows.Close()
		
		duration := time.Since(start)
		totalDuration += duration
		
		if duration < minDuration {
			minDuration = duration
		}
		if duration > maxDuration {
			maxDuration = duration
		}
	}
	
	successfulIterations := iterations - errorCount
	if successfulIterations == 0 {
		return nil, fmt.Errorf("모든 벤치마크 실행이 실패했습니다")
	}
	
	avgDuration := totalDuration / time.Duration(successfulIterations)
	
	result := &BenchmarkResult{
		Query:          query,
		Iterations:     iterations,
		SuccessCount:   successfulIterations,
		ErrorCount:     errorCount,
		TotalDuration:  totalDuration,
		AverageDuration: avgDuration,
		MinDuration:    minDuration,
		MaxDuration:    maxDuration,
		QPS:            float64(successfulIterations) / totalDuration.Seconds(),
	}
	
	a.logger.Info("쿼리 벤치마크 완료",
		zap.String("query", query),
		zap.Int("iterations", iterations),
		zap.Int("success_count", successfulIterations),
		zap.Int("error_count", errorCount),
		zap.Duration("avg_duration", avgDuration),
		zap.Float64("qps", result.QPS),
	)
	
	return result, nil
}

// BenchmarkResult 벤치마크 결과
type BenchmarkResult struct {
	Query           string        `json:"query"`
	Iterations      int           `json:"iterations"`
	SuccessCount    int           `json:"success_count"`
	ErrorCount      int           `json:"error_count"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	QPS             float64       `json:"qps"` // Queries Per Second
}