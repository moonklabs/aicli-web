package query

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

// QueryOptimizer 쿼리 최적화기
type QueryOptimizer struct {
	logger *zap.Logger
}

// OptimizationSuggestion 최적화 제안
type OptimizationSuggestion struct {
	Type        string  `json:"type"`        // index, query_rewrite, etc.
	Severity    string  `json:"severity"`    // low, medium, high
	Message     string  `json:"message"`
	Suggestion  string  `json:"suggestion"`
	Impact      string  `json:"impact"`
	Confidence  float64 `json:"confidence"`  // 0.0 - 1.0
}

// QueryAnalysisResult 쿼리 분석 결과
type QueryAnalysisResult struct {
	OriginalQuery string                   `json:"original_query"`
	OptimizedQuery string                  `json:"optimized_query,omitempty"`
	Suggestions   []OptimizationSuggestion `json:"suggestions"`
	EstimatedImprovement float64           `json:"estimated_improvement"` // 예상 성능 향상 (%)
	CanOptimize   bool                     `json:"can_optimize"`
}

// NewQueryOptimizer 새 쿼리 최적화기 생성
func NewQueryOptimizer(logger *zap.Logger) *QueryOptimizer {
	if logger == nil {
		logger = zap.NewNop()
	}
	
	return &QueryOptimizer{
		logger: logger,
	}
}

// AnalyzeQuery 쿼리 분석
func (qo *QueryOptimizer) AnalyzeQuery(query string) *QueryAnalysisResult {
	result := &QueryAnalysisResult{
		OriginalQuery: query,
		Suggestions:   make([]OptimizationSuggestion, 0),
	}
	
	queryLower := strings.ToLower(strings.TrimSpace(query))
	
	// 다양한 분석 수행
	qo.analyzeSelectStatement(queryLower, result)
	qo.analyzeWhereClause(queryLower, result)
	qo.analyzeJoins(queryLower, result)
	qo.analyzeOrderBy(queryLower, result)
	qo.analyzeLimitOffset(queryLower, result)
	qo.analyzeSubqueries(queryLower, result)
	
	// 최적화 가능 여부 판단
	result.CanOptimize = len(result.Suggestions) > 0
	
	// 예상 성능 향상 계산
	result.EstimatedImprovement = qo.calculateEstimatedImprovement(result.Suggestions)
	
	// 최적화된 쿼리 생성 (간단한 케이스만)
	if optimized := qo.generateOptimizedQuery(query, result.Suggestions); optimized != query {
		result.OptimizedQuery = optimized
	}
	
	qo.logger.Debug("쿼리 분석 완료",
		zap.String("original_query", query),
		zap.Int("suggestions_count", len(result.Suggestions)),
		zap.Float64("estimated_improvement", result.EstimatedImprovement),
	)
	
	return result
}

// analyzeSelectStatement SELECT 문 분석
func (qo *QueryOptimizer) analyzeSelectStatement(query string, result *QueryAnalysisResult) {
	// SELECT * 패턴 감지
	if strings.Contains(query, "select *") {
		result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
			Type:        "select_optimization",
			Severity:    "medium",
			Message:     "SELECT * 사용 감지됨",
			Suggestion:  "필요한 컬럼만 명시적으로 선택하세요",
			Impact:      "네트워크 트래픽 감소, 메모리 사용량 감소",
			Confidence:  0.9,
		})
	}
	
	// DISTINCT 남용 감지
	distinctCount := strings.Count(query, "distinct")
	if distinctCount > 1 {
		result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
			Type:        "distinct_optimization",
			Severity:    "medium", 
			Message:     "여러 DISTINCT 사용 감지됨",
			Suggestion:  "GROUP BY나 EXISTS를 사용하는 것을 고려하세요",
			Impact:      "쿼리 성능 향상",
			Confidence:  0.7,
		})
	}
}

// analyzeWhereClause WHERE 절 분석
func (qo *QueryOptimizer) analyzeWhereClause(query string, result *QueryAnalysisResult) {
	// 함수 사용 감지 (인덱스 사용 불가)
	functionPattern := regexp.MustCompile(`where\s+\w+\s*\(\s*\w+\s*\)\s*[=<>]`)
	if functionPattern.MatchString(query) {
		result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
			Type:        "index_optimization",
			Severity:    "high",
			Message:     "WHERE 절에서 함수 사용 감지됨",
			Suggestion:  "컬럼에 직접 조건을 적용하거나 함수 기반 인덱스를 생성하세요",
			Impact:      "인덱스 활용으로 큰 성능 향상",
			Confidence:  0.8,
		})
	}
	
	// LIKE '%pattern%' 패턴 감지
	if strings.Contains(query, "like '%") && strings.Contains(query, "%'") {
		likePattern := regexp.MustCompile(`like\s+'%[^%']+%'`)
		if likePattern.MatchString(query) {
			result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
				Type:        "search_optimization",
				Severity:    "medium",
				Message:     "전방 와일드카드 LIKE 패턴 감지됨",
				Suggestion:  "전문 검색 인덱스(FTS)나 접두어 검색을 고려하세요",
				Impact:      "검색 성능 대폭 향상",
				Confidence:  0.85,
			})
		}
	}
	
	// OR 조건 많음 감지
	orCount := strings.Count(query, " or ")
	if orCount > 5 {
		result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
			Type:        "query_rewrite",
			Severity:    "medium",
			Message:     fmt.Sprintf("많은 OR 조건 감지됨 (%d개)", orCount),
			Suggestion:  "IN 절이나 UNION으로 분리하는 것을 고려하세요",
			Impact:      "쿼리 실행 계획 최적화",
			Confidence:  0.7,
		})
	}
	
	// IS NULL 조건 확인
	if strings.Contains(query, "is null") || strings.Contains(query, "is not null") {
		result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
			Type:        "index_optimization",
			Severity:    "low",
			Message:     "NULL 체크 조건 감지됨",
			Suggestion:  "NULL 값이 많은 컬럼의 경우 부분 인덱스(WHERE NOT NULL)를 고려하세요",
			Impact:      "NULL 체크 성능 향상",
			Confidence:  0.6,
		})
	}
}

// analyzeJoins JOIN 분석
func (qo *QueryOptimizer) analyzeJoins(query string, result *QueryAnalysisResult) {
	// 카티젼 곱 감지 (FROM에 여러 테이블, WHERE에 조인 조건 없음)
	fromTables := qo.countTablesInFrom(query)
	joinCount := strings.Count(query, "join")
	whereJoinConditions := qo.countJoinConditionsInWhere(query)
	
	if fromTables > 1 && joinCount == 0 && whereJoinConditions == 0 {
		result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
			Type:        "join_optimization",
			Severity:    "high",
			Message:     "카티젼 곱 (Cartesian Product) 가능성 감지됨",
			Suggestion:  "명시적 JOIN 구문과 조인 조건을 사용하세요",
			Impact:      "잘못된 결과 방지, 성능 대폭 향상",
			Confidence:  0.95,
		})
	}
	
	// RIGHT JOIN 사용 감지
	if strings.Contains(query, "right join") {
		result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
			Type:        "join_optimization", 
			Severity:    "low",
			Message:     "RIGHT JOIN 사용 감지됨",
			Suggestion:  "LEFT JOIN으로 변경하여 가독성을 높이는 것을 고려하세요",
			Impact:      "쿼리 가독성 향상",
			Confidence:  0.5,
		})
	}
	
	// 많은 JOIN 감지
	if joinCount > 5 {
		result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
			Type:        "query_structure",
			Severity:    "medium",
			Message:     fmt.Sprintf("많은 JOIN 감지됨 (%d개)", joinCount),
			Suggestion:  "쿼리를 여러 개로 분리하거나 비정규화를 고려하세요",
			Impact:      "복잡도 감소, 성능 향상 가능",
			Confidence:  0.6,
		})
	}
}

// analyzeOrderBy ORDER BY 분석
func (qo *QueryOptimizer) analyzeOrderBy(query string, result *QueryAnalysisResult) {
	// ORDER BY 없이 LIMIT 사용
	if strings.Contains(query, "limit") && !strings.Contains(query, "order by") {
		result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
			Type:        "consistency",
			Severity:    "medium",
			Message:     "ORDER BY 없이 LIMIT 사용됨",
			Suggestion:  "일관된 결과를 위해 ORDER BY를 추가하세요",
			Impact:      "결과 일관성 보장",
			Confidence:  0.8,
		})
	}
	
	// 복잡한 ORDER BY 표현식
	orderByPattern := regexp.MustCompile(`order\s+by\s+[^,\s]*\s*\([^)]+\)`)
	if orderByPattern.MatchString(query) {
		result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
			Type:        "index_optimization",
			Severity:    "medium", 
			Message:     "복잡한 ORDER BY 표현식 감지됨",
			Suggestion:  "계산된 컬럼이나 함수 기반 인덱스를 고려하세요",
			Impact:      "정렬 성능 향상",
			Confidence:  0.7,
		})
	}
}

// analyzeLimitOffset LIMIT/OFFSET 분석
func (qo *QueryOptimizer) analyzeLimitOffset(query string, result *QueryAnalysisResult) {
	// 큰 OFFSET 감지
	offsetPattern := regexp.MustCompile(`offset\s+(\d+)`)
	matches := offsetPattern.FindStringSubmatch(query)
	if len(matches) > 1 {
		if offset := parseInt(matches[1]); offset > 10000 {
			result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
				Type:        "pagination_optimization",
				Severity:    "high",
				Message:     fmt.Sprintf("큰 OFFSET 값 감지됨 (%d)", offset),
				Suggestion:  "커서 기반 페이지네이션을 사용하세요",
				Impact:      "페이지네이션 성능 대폭 향상",
				Confidence:  0.9,
			})
		}
	}
}

// analyzeSubqueries 서브쿼리 분석
func (qo *QueryOptimizer) analyzeSubqueries(query string, result *QueryAnalysisResult) {
	// 상관 서브쿼리 감지
	subqueryPattern := regexp.MustCompile(`\(\s*select\s+[^)]+\)`)
	subqueries := subqueryPattern.FindAllString(query, -1)
	
	if len(subqueries) > 0 {
		// 각 서브쿼리를 분석하여 상관 서브쿼리인지 확인
		for _, subquery := range subqueries {
			if qo.isCorrelatedSubquery(query, subquery) {
				result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
					Type:        "subquery_optimization",
					Severity:    "high",
					Message:     "상관 서브쿼리 감지됨",
					Suggestion:  "JOIN으로 변경하거나 EXISTS를 사용하는 것을 고려하세요",
					Impact:      "쿼리 성능 대폭 향상",
					Confidence:  0.8,
				})
				break
			}
		}
		
		// IN 절의 서브쿼리
		inSubqueryPattern := regexp.MustCompile(`in\s*\(\s*select\s+[^)]+\)`)
		if inSubqueryPattern.MatchString(query) {
			result.Suggestions = append(result.Suggestions, OptimizationSuggestion{
				Type:        "subquery_optimization",
				Severity:    "medium",
				Message:     "IN 절에 서브쿼리 사용 감지됨",
				Suggestion:  "EXISTS나 JOIN을 사용하는 것을 고려하세요",
				Impact:      "성능 향상 가능",
				Confidence:  0.7,
			})
		}
	}
}

// 유틸리티 메서드들

// countTablesInFrom FROM 절의 테이블 수 계산
func (qo *QueryOptimizer) countTablesInFrom(query string) int {
	fromPattern := regexp.MustCompile(`from\s+([^where\s]+)`)
	matches := fromPattern.FindStringSubmatch(query)
	if len(matches) < 2 {
		return 0
	}
	
	tables := strings.Split(matches[1], ",")
	return len(tables)
}

// countJoinConditionsInWhere WHERE 절의 조인 조건 수 계산
func (qo *QueryOptimizer) countJoinConditionsInWhere(query string) int {
	// 간단한 패턴으로 table1.col = table2.col 형태 감지
	joinConditionPattern := regexp.MustCompile(`\w+\.\w+\s*=\s*\w+\.\w+`)
	matches := joinConditionPattern.FindAllString(query, -1)
	return len(matches)
}

// isCorrelatedSubquery 상관 서브쿼리 여부 확인
func (qo *QueryOptimizer) isCorrelatedSubquery(mainQuery, subquery string) bool {
	// 메인 쿼리의 테이블 별칭들을 찾아서 서브쿼리에서 참조하는지 확인
	// 간단한 구현 - 실제로는 더 정교한 파싱 필요
	
	// 메인 쿼리에서 테이블 별칭 추출
	aliasPattern := regexp.MustCompile(`(?:from|join)\s+\w+\s+(?:as\s+)?(\w+)`)
	aliases := aliasPattern.FindAllStringSubmatch(mainQuery, -1)
	
	// 서브쿼리에서 해당 별칭 참조하는지 확인
	for _, alias := range aliases {
		if len(alias) > 1 && strings.Contains(subquery, alias[1]+".") {
			return true
		}
	}
	
	return false
}

// calculateEstimatedImprovement 예상 성능 향상 계산
func (qo *QueryOptimizer) calculateEstimatedImprovement(suggestions []OptimizationSuggestion) float64 {
	var totalImprovement float64
	
	for _, suggestion := range suggestions {
		var impact float64
		
		switch suggestion.Type {
		case "index_optimization":
			impact = 30.0 * suggestion.Confidence
		case "subquery_optimization":
			impact = 40.0 * suggestion.Confidence
		case "join_optimization":
			impact = 35.0 * suggestion.Confidence
		case "pagination_optimization":
			impact = 50.0 * suggestion.Confidence
		case "select_optimization":
			impact = 15.0 * suggestion.Confidence
		case "query_rewrite":
			impact = 25.0 * suggestion.Confidence
		default:
			impact = 10.0 * suggestion.Confidence
		}
		
		// 심각도에 따른 가중치
		switch suggestion.Severity {
		case "high":
			impact *= 1.5
		case "medium":
			impact *= 1.0
		case "low":
			impact *= 0.7
		}
		
		totalImprovement += impact
	}
	
	// 최대 100%로 제한
	if totalImprovement > 100 {
		totalImprovement = 100
	}
	
	return totalImprovement
}

// generateOptimizedQuery 최적화된 쿼리 생성 (간단한 케이스만)
func (qo *QueryOptimizer) generateOptimizedQuery(originalQuery string, suggestions []OptimizationSuggestion) string {
	optimized := originalQuery
	
	for _, suggestion := range suggestions {
		switch suggestion.Type {
		case "select_optimization":
			// SELECT * -> SELECT explicit columns (예시)
			if strings.Contains(strings.ToLower(optimized), "select *") {
				// 실제로는 스키마 정보가 필요하므로 예시만
				qo.logger.Debug("SELECT * 최적화 제안됨 (실제 변경 안됨)")
			}
		}
	}
	
	return optimized
}

// parseInt 문자열을 int로 변환 (에러 무시)
func parseInt(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}

// QueryExecutor 최적화된 쿼리 실행기
type QueryExecutor struct {
	optimizer *QueryOptimizer
	logger    *zap.Logger
}

// NewQueryExecutor 새 쿼리 실행기 생성
func NewQueryExecutor(optimizer *QueryOptimizer, logger *zap.Logger) *QueryExecutor {
	if logger == nil {
		logger = zap.NewNop()
	}
	
	return &QueryExecutor{
		optimizer: optimizer,
		logger:    logger,
	}
}

// ExecuteWithAnalysis 분석과 함께 쿼리 실행
func (qe *QueryExecutor) ExecuteWithAnalysis(query string, executor func(string, []interface{}) error, params []interface{}) (*QueryAnalysisResult, error) {
	start := time.Now()
	
	// 쿼리 분석
	analysis := qe.optimizer.AnalyzeQuery(query)
	
	// 최적화된 쿼리가 있으면 사용
	queryToExecute := query
	if analysis.OptimizedQuery != "" {
		queryToExecute = analysis.OptimizedQuery
		qe.logger.Info("최적화된 쿼리 사용",
			zap.String("original", query),
			zap.String("optimized", queryToExecute),
		)
	}
	
	// 쿼리 실행
	err := executor(queryToExecute, params)
	
	duration := time.Since(start)
	
	if err != nil {
		qe.logger.Error("쿼리 실행 실패",
			zap.String("query", queryToExecute),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return analysis, err
	}
	
	qe.logger.Debug("쿼리 실행 완료",
		zap.String("query", queryToExecute),
		zap.Duration("duration", duration),
		zap.Int("suggestions_count", len(analysis.Suggestions)),
	)
	
	return analysis, nil
}