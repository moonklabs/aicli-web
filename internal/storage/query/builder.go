package query

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// QueryBuilder SQL 쿼리 빌더
type QueryBuilder struct {
	selectFields []string
	fromTable    string
	joins        []Join
	whereClause  []WhereCondition
	groupBy      []string
	having       []WhereCondition
	orderBy      []OrderClause
	limit        *int
	offset       *int
	params       []interface{}
	paramCount   int
}

// Join 조인 구조
type Join struct {
	Type      JoinType // INNER, LEFT, RIGHT, FULL
	Table     string
	Condition string
	Params    []interface{}
}

// JoinType 조인 타입
type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
	FullJoin  JoinType = "FULL JOIN"
)

// WhereCondition WHERE 조건
type WhereCondition struct {
	Field    string
	Operator string
	Value    interface{}
	Logic    LogicOperator // AND, OR
	Custom   string        // 커스텀 조건
	Params   []interface{} // 커스텀 조건의 파라미터
}

// LogicOperator 논리 연산자
type LogicOperator string

const (
	AndOperator LogicOperator = "AND"
	OrOperator  LogicOperator = "OR"
)

// OrderClause 정렬 조건
type OrderClause struct {
	Field string
	Dir   OrderDirection
}

// OrderDirection 정렬 방향
type OrderDirection string

const (
	ASC  OrderDirection = "ASC"
	DESC OrderDirection = "DESC"
)

// NewQueryBuilder 새 쿼리 빌더 생성
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		selectFields: make([]string, 0),
		joins:        make([]Join, 0),
		whereClause:  make([]WhereCondition, 0),
		groupBy:      make([]string, 0),
		having:       make([]WhereCondition, 0),
		orderBy:      make([]OrderClause, 0),
		params:       make([]interface{}, 0),
		paramCount:   0,
	}
}

// Select SELECT 필드 추가
func (qb *QueryBuilder) Select(fields ...string) *QueryBuilder {
	qb.selectFields = append(qb.selectFields, fields...)
	return qb
}

// SelectDistinct DISTINCT SELECT 필드 추가
func (qb *QueryBuilder) SelectDistinct(fields ...string) *QueryBuilder {
	if len(qb.selectFields) == 0 || !strings.HasPrefix(qb.selectFields[0], "DISTINCT") {
		qb.selectFields = append([]string{"DISTINCT"}, qb.selectFields...)
	}
	qb.selectFields = append(qb.selectFields, fields...)
	return qb
}

// From FROM 테이블 설정
func (qb *QueryBuilder) From(table string) *QueryBuilder {
	qb.fromTable = table
	return qb
}

// InnerJoin INNER JOIN 추가
func (qb *QueryBuilder) InnerJoin(table, condition string, params ...interface{}) *QueryBuilder {
	qb.joins = append(qb.joins, Join{
		Type:      InnerJoin,
		Table:     table,
		Condition: condition,
		Params:    params,
	})
	return qb
}

// LeftJoin LEFT JOIN 추가
func (qb *QueryBuilder) LeftJoin(table, condition string, params ...interface{}) *QueryBuilder {
	qb.joins = append(qb.joins, Join{
		Type:      LeftJoin,
		Table:     table,
		Condition: condition,
		Params:    params,
	})
	return qb
}

// Where WHERE 조건 추가
func (qb *QueryBuilder) Where(field, operator string, value interface{}) *QueryBuilder {
	qb.whereClause = append(qb.whereClause, WhereCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
		Logic:    AndOperator,
	})
	return qb
}

// WhereOr WHERE ... OR 조건 추가
func (qb *QueryBuilder) WhereOr(field, operator string, value interface{}) *QueryBuilder {
	qb.whereClause = append(qb.whereClause, WhereCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
		Logic:    OrOperator,
	})
	return qb
}

// WhereCustom 커스텀 WHERE 조건 추가
func (qb *QueryBuilder) WhereCustom(condition string, params ...interface{}) *QueryBuilder {
	qb.whereClause = append(qb.whereClause, WhereCondition{
		Custom: condition,
		Params: params,
		Logic:  AndOperator,
	})
	return qb
}

// WhereIn WHERE IN 조건 추가
func (qb *QueryBuilder) WhereIn(field string, values []interface{}) *QueryBuilder {
	if len(values) == 0 {
		return qb
	}
	
	placeholders := make([]string, len(values))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	
	condition := fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", "))
	qb.whereClause = append(qb.whereClause, WhereCondition{
		Custom: condition,
		Params: values,
		Logic:  AndOperator,
	})
	return qb
}

// WhereBetween WHERE BETWEEN 조건 추가
func (qb *QueryBuilder) WhereBetween(field string, start, end interface{}) *QueryBuilder {
	condition := fmt.Sprintf("%s BETWEEN ? AND ?", field)
	qb.whereClause = append(qb.whereClause, WhereCondition{
		Custom: condition,
		Params: []interface{}{start, end},
		Logic:  AndOperator,
	})
	return qb
}

// WhereNotNull WHERE NOT NULL 조건 추가
func (qb *QueryBuilder) WhereNotNull(field string) *QueryBuilder {
	condition := fmt.Sprintf("%s IS NOT NULL", field)
	qb.whereClause = append(qb.whereClause, WhereCondition{
		Custom: condition,
		Logic:  AndOperator,
	})
	return qb
}

// WhereNull WHERE NULL 조건 추가
func (qb *QueryBuilder) WhereNull(field string) *QueryBuilder {
	condition := fmt.Sprintf("%s IS NULL", field)
	qb.whereClause = append(qb.whereClause, WhereCondition{
		Custom: condition,
		Logic:  AndOperator,
	})
	return qb
}

// GroupBy GROUP BY 추가
func (qb *QueryBuilder) GroupBy(fields ...string) *QueryBuilder {
	qb.groupBy = append(qb.groupBy, fields...)
	return qb
}

// Having HAVING 조건 추가
func (qb *QueryBuilder) Having(field, operator string, value interface{}) *QueryBuilder {
	qb.having = append(qb.having, WhereCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
		Logic:    AndOperator,
	})
	return qb
}

// OrderBy ORDER BY 추가
func (qb *QueryBuilder) OrderBy(field string, direction OrderDirection) *QueryBuilder {
	qb.orderBy = append(qb.orderBy, OrderClause{
		Field: field,
		Dir:   direction,
	})
	return qb
}

// OrderByDesc ORDER BY DESC 추가
func (qb *QueryBuilder) OrderByDesc(field string) *QueryBuilder {
	return qb.OrderBy(field, DESC)
}

// OrderByAsc ORDER BY ASC 추가
func (qb *QueryBuilder) OrderByAsc(field string) *QueryBuilder {
	return qb.OrderBy(field, ASC)
}

// Limit LIMIT 설정
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = &limit
	return qb
}

// Offset OFFSET 설정
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = &offset
	return qb
}

// Paginate 페이지네이션 설정
func (qb *QueryBuilder) Paginate(page, perPage int) *QueryBuilder {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	
	offset := (page - 1) * perPage
	qb.Limit(perPage)
	qb.Offset(offset)
	return qb
}

// Build 쿼리 문자열과 파라미터 빌드
func (qb *QueryBuilder) Build() (string, []interface{}, error) {
	if qb.fromTable == "" {
		return "", nil, fmt.Errorf("FROM 테이블이 지정되지 않았습니다")
	}
	
	var query strings.Builder
	qb.params = make([]interface{}, 0)
	qb.paramCount = 0
	
	// SELECT
	query.WriteString("SELECT ")
	if len(qb.selectFields) == 0 {
		query.WriteString("*")
	} else {
		query.WriteString(strings.Join(qb.selectFields, ", "))
	}
	
	// FROM
	query.WriteString(" FROM ")
	query.WriteString(qb.fromTable)
	
	// JOIN
	for _, join := range qb.joins {
		query.WriteString(" ")
		query.WriteString(string(join.Type))
		query.WriteString(" ")
		query.WriteString(join.Table)
		query.WriteString(" ON ")
		query.WriteString(join.Condition)
		
		qb.params = append(qb.params, join.Params...)
	}
	
	// WHERE
	if len(qb.whereClause) > 0 {
		query.WriteString(" WHERE ")
		qb.buildConditions(&query, qb.whereClause)
	}
	
	// GROUP BY
	if len(qb.groupBy) > 0 {
		query.WriteString(" GROUP BY ")
		query.WriteString(strings.Join(qb.groupBy, ", "))
	}
	
	// HAVING
	if len(qb.having) > 0 {
		query.WriteString(" HAVING ")
		qb.buildConditions(&query, qb.having)
	}
	
	// ORDER BY
	if len(qb.orderBy) > 0 {
		query.WriteString(" ORDER BY ")
		orderParts := make([]string, len(qb.orderBy))
		for i, order := range qb.orderBy {
			orderParts[i] = fmt.Sprintf("%s %s", order.Field, order.Dir)
		}
		query.WriteString(strings.Join(orderParts, ", "))
	}
	
	// LIMIT
	if qb.limit != nil {
		query.WriteString(" LIMIT ")
		query.WriteString(strconv.Itoa(*qb.limit))
	}
	
	// OFFSET
	if qb.offset != nil {
		query.WriteString(" OFFSET ")
		query.WriteString(strconv.Itoa(*qb.offset))
	}
	
	return query.String(), qb.params, nil
}

// buildConditions WHERE/HAVING 조건들 빌드
func (qb *QueryBuilder) buildConditions(query *strings.Builder, conditions []WhereCondition) {
	for i, condition := range conditions {
		if i > 0 {
			query.WriteString(" ")
			query.WriteString(string(condition.Logic))
			query.WriteString(" ")
		}
		
		if condition.Custom != "" {
			query.WriteString(condition.Custom)
			qb.params = append(qb.params, condition.Params...)
		} else {
			query.WriteString(condition.Field)
			query.WriteString(" ")
			query.WriteString(condition.Operator)
			query.WriteString(" ?")
			qb.params = append(qb.params, condition.Value)
		}
	}
}

// BuildCount COUNT 쿼리 빌드
func (qb *QueryBuilder) BuildCount() (string, []interface{}, error) {
	// 기존 SELECT와 ORDER BY, LIMIT, OFFSET을 제거한 COUNT 쿼리 생성
	countBuilder := &QueryBuilder{
		fromTable:   qb.fromTable,
		joins:       qb.joins,
		whereClause: qb.whereClause,
		groupBy:     qb.groupBy,
		having:      qb.having,
		params:      make([]interface{}, 0),
	}
	
	if len(countBuilder.groupBy) > 0 {
		// GROUP BY가 있으면 서브쿼리로 감싸서 COUNT
		countBuilder.selectFields = []string{"COUNT(*) as count"}
		innerQuery, innerParams, err := countBuilder.Build()
		if err != nil {
			return "", nil, err
		}
		
		query := fmt.Sprintf("SELECT COUNT(*) FROM (%s) as subquery", innerQuery)
		return query, innerParams, nil
	} else {
		// 단순한 COUNT 쿼리
		countBuilder.selectFields = []string{"COUNT(*)"}
		return countBuilder.Build()
	}
}

// Clone 쿼리 빌더 복제
func (qb *QueryBuilder) Clone() *QueryBuilder {
	clone := &QueryBuilder{
		selectFields: make([]string, len(qb.selectFields)),
		fromTable:    qb.fromTable,
		joins:        make([]Join, len(qb.joins)),
		whereClause:  make([]WhereCondition, len(qb.whereClause)),
		groupBy:      make([]string, len(qb.groupBy)),
		having:       make([]WhereCondition, len(qb.having)),
		orderBy:      make([]OrderClause, len(qb.orderBy)),
		params:       make([]interface{}, 0),
		paramCount:   0,
	}
	
	copy(clone.selectFields, qb.selectFields)
	copy(clone.joins, qb.joins)
	copy(clone.whereClause, qb.whereClause)
	copy(clone.groupBy, qb.groupBy)
	copy(clone.having, qb.having)
	copy(clone.orderBy, qb.orderBy)
	
	if qb.limit != nil {
		limit := *qb.limit
		clone.limit = &limit
	}
	
	if qb.offset != nil {
		offset := *qb.offset
		clone.offset = &offset
	}
	
	return clone
}

// WorkspaceQueryBuilder 워크스페이스 전용 쿼리 빌더
type WorkspaceQueryBuilder struct {
	*QueryBuilder
}

// NewWorkspaceQueryBuilder 워크스페이스 쿼리 빌더 생성
func NewWorkspaceQueryBuilder() *WorkspaceQueryBuilder {
	return &WorkspaceQueryBuilder{
		QueryBuilder: NewQueryBuilder().From("workspaces"),
	}
}

// WithProjects 프로젝트 조인
func (wqb *WorkspaceQueryBuilder) WithProjects() *WorkspaceQueryBuilder {
	wqb.LeftJoin("projects", "projects.workspace_id = workspaces.id AND projects.deleted_at IS NULL")
	return wqb
}

// ActiveOnly 활성 워크스페이스만
func (wqb *WorkspaceQueryBuilder) ActiveOnly() *WorkspaceQueryBuilder {
	wqb.Where("workspaces.status", "=", models.WorkspaceStatusActive)
	wqb.WhereNull("workspaces.deleted_at")
	return wqb
}

// ByOwner 소유자별 필터
func (wqb *WorkspaceQueryBuilder) ByOwner(ownerID string) *WorkspaceQueryBuilder {
	wqb.Where("workspaces.owner_id", "=", ownerID)
	return wqb
}

// SearchByName 이름으로 검색
func (wqb *WorkspaceQueryBuilder) SearchByName(name string) *WorkspaceQueryBuilder {
	wqb.Where("workspaces.name", "LIKE", "%"+name+"%")
	return wqb
}

// CreatedAfter 특정 날짜 이후 생성
func (wqb *WorkspaceQueryBuilder) CreatedAfter(date time.Time) *WorkspaceQueryBuilder {
	wqb.Where("workspaces.created_at", ">=", date)
	return wqb
}

// ProjectQueryBuilder 프로젝트 전용 쿼리 빌더
type ProjectQueryBuilder struct {
	*QueryBuilder
}

// NewProjectQueryBuilder 프로젝트 쿼리 빌더 생성
func NewProjectQueryBuilder() *ProjectQueryBuilder {
	return &ProjectQueryBuilder{
		QueryBuilder: NewQueryBuilder().From("projects"),
	}
}

// WithWorkspace 워크스페이스 조인
func (pqb *ProjectQueryBuilder) WithWorkspace() *ProjectQueryBuilder {
	pqb.LeftJoin("workspaces", "workspaces.id = projects.workspace_id AND workspaces.deleted_at IS NULL")
	return pqb
}

// WithSessions 세션 조인
func (pqb *ProjectQueryBuilder) WithSessions() *ProjectQueryBuilder {
	pqb.LeftJoin("sessions", "sessions.project_id = projects.id")
	return pqb
}

// ActiveOnly 활성 프로젝트만
func (pqb *ProjectQueryBuilder) ActiveOnly() *ProjectQueryBuilder {
	pqb.Where("projects.status", "=", models.ProjectStatusActive)
	pqb.WhereNull("projects.deleted_at")
	return pqb
}

// ByWorkspace 워크스페이스별 필터
func (pqb *ProjectQueryBuilder) ByWorkspace(workspaceID string) *ProjectQueryBuilder {
	pqb.Where("projects.workspace_id", "=", workspaceID)
	return pqb
}

// ByLanguage 언어별 필터
func (pqb *ProjectQueryBuilder) ByLanguage(language string) *ProjectQueryBuilder {
	pqb.Where("projects.language", "=", language)
	return pqb
}

// SessionQueryBuilder 세션 전용 쿼리 빌더
type SessionQueryBuilder struct {
	*QueryBuilder
}

// NewSessionQueryBuilder 세션 쿼리 빌더 생성
func NewSessionQueryBuilder() *SessionQueryBuilder {
	return &SessionQueryBuilder{
		QueryBuilder: NewQueryBuilder().From("sessions"),
	}
}

// WithProject 프로젝트 조인
func (sqb *SessionQueryBuilder) WithProject() *SessionQueryBuilder {
	sqb.LeftJoin("projects", "projects.id = sessions.project_id AND projects.deleted_at IS NULL")
	return sqb
}

// WithTasks 태스크 조인
func (sqb *SessionQueryBuilder) WithTasks() *SessionQueryBuilder {
	sqb.LeftJoin("tasks", "tasks.session_id = sessions.id")
	return sqb
}

// ActiveOnly 활성 세션만
func (sqb *SessionQueryBuilder) ActiveOnly() *SessionQueryBuilder {
	sqb.WhereIn("sessions.status", []interface{}{
		models.SessionStatusActive,
		models.SessionStatusIdle,
	})
	return sqb
}

// ByProject 프로젝트별 필터
func (sqb *SessionQueryBuilder) ByProject(projectID string) *SessionQueryBuilder {
	sqb.Where("sessions.project_id", "=", projectID)
	return sqb
}

// ActiveSince 특정 시간 이후 활성
func (sqb *SessionQueryBuilder) ActiveSince(since time.Time) *SessionQueryBuilder {
	sqb.Where("sessions.last_active", ">=", since)
	return sqb
}

// TaskQueryBuilder 태스크 전용 쿼리 빌더
type TaskQueryBuilder struct {
	*QueryBuilder
}

// NewTaskQueryBuilder 태스크 쿼리 빌더 생성
func NewTaskQueryBuilder() *TaskQueryBuilder {
	return &TaskQueryBuilder{
		QueryBuilder: NewQueryBuilder().From("tasks"),
	}
}

// WithSession 세션 조인
func (tqb *TaskQueryBuilder) WithSession() *TaskQueryBuilder {
	tqb.LeftJoin("sessions", "sessions.id = tasks.session_id")
	return tqb
}

// RunningOnly 실행 중인 태스크만
func (tqb *TaskQueryBuilder) RunningOnly() *TaskQueryBuilder {
	tqb.WhereIn("tasks.status", []interface{}{
		models.TaskStatusPending,
		models.TaskStatusRunning,
	})
	return tqb
}

// BySession 세션별 필터
func (tqb *TaskQueryBuilder) BySession(sessionID string) *TaskQueryBuilder {
	tqb.Where("tasks.session_id", "=", sessionID)
	return tqb
}

// CompletedOnly 완료된 태스크만
func (tqb *TaskQueryBuilder) CompletedOnly() *TaskQueryBuilder {
	tqb.Where("tasks.status", "=", models.TaskStatusCompleted)
	return tqb
}

// LongRunning 오래 실행되는 태스크
func (tqb *TaskQueryBuilder) LongRunning(minDuration time.Duration) *TaskQueryBuilder {
	tqb.Where("tasks.duration", ">=", minDuration.Milliseconds())
	return tqb
}

// Recent 최근 태스크
func (tqb *TaskQueryBuilder) Recent(since time.Time) *TaskQueryBuilder {
	tqb.Where("tasks.created_at", ">=", since)
	return tqb
}