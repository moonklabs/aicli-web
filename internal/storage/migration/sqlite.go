package migration

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
	
	_ "github.com/mattn/go-sqlite3" // SQLite 드라이버
)

// SQLiteTracker SQLite 마이그레이션 추적기
type SQLiteTracker struct {
	db        *sql.DB
	tableName string
}

// NewSQLiteTracker 새 SQLite 추적기 생성
func NewSQLiteTracker(db *sql.DB) *SQLiteTracker {
	return &SQLiteTracker{
		db:        db,
		tableName: "schema_migrations",
	}
}

// EnsureTable 마이그레이션 테이블 생성
func (t *SQLiteTracker) EnsureTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version TEXT PRIMARY KEY,
			direction TEXT NOT NULL,
			status TEXT NOT NULL,
			started_at DATETIME NOT NULL,
			completed_at DATETIME,
			duration_ms INTEGER DEFAULT 0,
			error_message TEXT,
			checksum TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`, t.tableName)
	
	_, err := t.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("마이그레이션 테이블 생성 실패: %w", err)
	}
	
	// 인덱스 생성
	indexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_status_started 
		ON %s (status, started_at)
	`, t.tableName, t.tableName)
	
	_, err = t.db.ExecContext(ctx, indexQuery)
	if err != nil {
		return fmt.Errorf("마이그레이션 인덱스 생성 실패: %w", err)
	}
	
	return nil
}

// GetApplied 적용된 마이그레이션 목록 반환
func (t *SQLiteTracker) GetApplied(ctx context.Context) ([]MigrationRecord, error) {
	query := fmt.Sprintf(`
		SELECT version, direction, status, started_at, completed_at, 
		       duration_ms, COALESCE(error_message, ''), COALESCE(checksum, '')
		FROM %s 
		WHERE status = 'completed'
		ORDER BY version ASC
	`, t.tableName)
	
	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("적용된 마이그레이션 조회 실패: %w", err)
	}
	defer rows.Close()
	
	var records []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		var completedAt sql.NullTime
		var durationMs int64
		var direction, status string
		
		err := rows.Scan(
			&record.Version,
			&direction,
			&status,
			&record.StartedAt,
			&completedAt,
			&durationMs,
			&record.Error,
			&record.Checksum,
		)
		if err != nil {
			return nil, fmt.Errorf("마이그레이션 레코드 스캔 실패: %w", err)
		}
		
		record.Direction = Direction(direction)
		record.Status = MigrationStatus(status)
		record.Duration = time.Duration(durationMs) * time.Millisecond
		
		if completedAt.Valid {
			record.CompletedAt = &completedAt.Time
		}
		
		records = append(records, record)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("마이그레이션 행 반복 중 에러: %w", err)
	}
	
	return records, nil
}

// IsApplied 특정 버전이 적용되었는지 확인
func (t *SQLiteTracker) IsApplied(ctx context.Context, version string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s 
		WHERE version = ? AND status = 'completed' AND direction = 'up'
	`, t.tableName)
	
	var count int
	err := t.db.QueryRowContext(ctx, query, version).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("마이그레이션 적용 상태 확인 실패: %w", err)
	}
	
	return count > 0, nil
}

// RecordUp 업그레이드 마이그레이션 기록
func (t *SQLiteTracker) RecordUp(ctx context.Context, version string, duration time.Duration) error {
	return t.recordMigration(ctx, version, DirectionUp, MigrationStatusCompleted, duration, "")
}

// RecordDown 다운그레이드 마이그레이션 기록
func (t *SQLiteTracker) RecordDown(ctx context.Context, version string, duration time.Duration) error {
	return t.recordMigration(ctx, version, DirectionDown, MigrationStatusCompleted, duration, "")
}

// RecordFailed 실패한 마이그레이션 기록
func (t *SQLiteTracker) RecordFailed(ctx context.Context, version string, direction Direction, err error) error {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return t.recordMigration(ctx, version, direction, MigrationStatusFailed, 0, errMsg)
}

// RemoveRecord 마이그레이션 기록 제거 (롤백 시 사용)
func (t *SQLiteTracker) RemoveRecord(ctx context.Context, version string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE version = ?`, t.tableName)
	
	result, err := t.db.ExecContext(ctx, query, version)
	if err != nil {
		return fmt.Errorf("마이그레이션 기록 제거 실패: %w", err)
	}
	
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("영향받은 행 수 확인 실패: %w", err)
	}
	
	if affected == 0 {
		return fmt.Errorf("제거할 마이그레이션 기록을 찾을 수 없습니다: %s", version)
	}
	
	return nil
}

// recordMigration 마이그레이션 기록 저장
func (t *SQLiteTracker) recordMigration(ctx context.Context, version string, direction Direction, status MigrationStatus, duration time.Duration, errorMsg string) error {
	// 기존 레코드 삭제 (있다면)
	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE version = ?`, t.tableName)
	_, err := t.db.ExecContext(ctx, deleteQuery, version)
	if err != nil {
		return fmt.Errorf("기존 마이그레이션 기록 삭제 실패: %w", err)
	}
	
	// 새 레코드 삽입
	insertQuery := fmt.Sprintf(`
		INSERT INTO %s (version, direction, status, started_at, completed_at, duration_ms, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, t.tableName)
	
	now := time.Now().UTC()
	var completedAt *time.Time
	if status == MigrationStatusCompleted {
		completedAt = &now
	}
	
	durationMs := duration.Milliseconds()
	
	_, err = t.db.ExecContext(ctx, insertQuery,
		version,
		string(direction),
		string(status),
		now,
		completedAt,
		durationMs,
		errorMsg,
	)
	
	if err != nil {
		return fmt.Errorf("마이그레이션 기록 저장 실패: %w", err)
	}
	
	return nil
}

// SQLiteMigrator SQLite 마이그레이션 실행기
type SQLiteMigrator struct {
	db      *sql.DB
	source  MigrationSource
	tracker MigrationTracker
	options MigrationOptions
}

// NewSQLiteMigrator 새 SQLite 마이그레이션 실행기 생성
func NewSQLiteMigrator(db *sql.DB, source MigrationSource, options MigrationOptions) *SQLiteMigrator {
	tracker := NewSQLiteTracker(db)
	
	return &SQLiteMigrator{
		db:      db,
		source:  source,
		tracker: tracker,
		options: options,
	}
}

// Current 현재 스키마 버전 반환
func (m *SQLiteMigrator) Current() (string, error) {
	ctx := context.Background()
	
	if err := m.tracker.EnsureTable(ctx); err != nil {
		return "", err
	}
	
	applied, err := m.tracker.GetApplied(ctx)
	if err != nil {
		return "", err
	}
	
	if len(applied) == 0 {
		return "", nil // 마이그레이션 없음
	}
	
	// 가장 높은 버전 반환
	versions := make([]string, len(applied))
	for i, record := range applied {
		if record.Direction == DirectionUp && record.Status == MigrationStatusCompleted {
			versions[i] = record.Version
		}
	}
	
	sort.Strings(versions)
	
	for i := len(versions) - 1; i >= 0; i-- {
		if versions[i] != "" {
			return versions[i], nil
		}
	}
	
	return "", nil
}

// Migrate 특정 버전으로 마이그레이션
func (m *SQLiteMigrator) Migrate(ctx context.Context, target string) error {
	if err := m.tracker.EnsureTable(ctx); err != nil {
		return err
	}
	
	current, err := m.Current()
	if err != nil {
		return err
	}
	
	if current == target {
		return nil // 이미 목표 버전
	}
	
	// 사용 가능한 마이그레이션 로드
	migrations, err := m.source.Load()
	if err != nil {
		return fmt.Errorf("마이그레이션 로드 실패: %w", err)
	}
	
	// 실행할 마이그레이션들 결정
	toExecute, direction, err := m.planMigrations(migrations, current, target)
	if err != nil {
		return err
	}
	
	// 마이그레이션 실행
	for _, migration := range toExecute {
		if err := m.executeSingleMigration(ctx, migration, direction); err != nil {
			return err
		}
	}
	
	return nil
}

// MigrateUp 최신 버전으로 업그레이드
func (m *SQLiteMigrator) MigrateUp(ctx context.Context) error {
	migrations, err := m.source.Load()
	if err != nil {
		return fmt.Errorf("마이그레이션 로드 실패: %w", err)
	}
	
	if len(migrations) == 0 {
		return nil // 마이그레이션 없음
	}
	
	// 최신 버전 찾기
	versions := make([]string, len(migrations))
	for i, migration := range migrations {
		versions[i] = migration.Version()
	}
	sort.Strings(versions)
	
	latestVersion := versions[len(versions)-1]
	return m.Migrate(ctx, latestVersion)
}

// Rollback N단계 롤백
func (m *SQLiteMigrator) Rollback(ctx context.Context, steps int) error {
	if steps <= 0 {
		return fmt.Errorf("롤백 단계는 양수여야 합니다")
	}
	
	applied, err := m.tracker.GetApplied(ctx)
	if err != nil {
		return err
	}
	
	if len(applied) == 0 {
		return fmt.Errorf("롤백할 마이그레이션이 없습니다")
	}
	
	// 역순으로 정렬
	sort.Slice(applied, func(i, j int) bool {
		return CompareVersions(applied[i].Version, applied[j].Version) > 0
	})
	
	// 지정된 단계만큼 롤백
	if steps > len(applied) {
		steps = len(applied)
	}
	
	for i := 0; i < steps; i++ {
		record := applied[i]
		migration, err := m.source.Get(record.Version)
		if err != nil {
			return fmt.Errorf("마이그레이션 %s 로드 실패: %w", record.Version, err)
		}
		
		if err := m.executeSingleMigration(ctx, migration, DirectionDown); err != nil {
			return err
		}
	}
	
	return nil
}

// RollbackTo 특정 버전으로 롤백
func (m *SQLiteMigrator) RollbackTo(ctx context.Context, target string) error {
	applied, err := m.tracker.GetApplied(ctx)
	if err != nil {
		return err
	}
	
	// 목표 버전 이후의 마이그레이션들 찾기
	var toRollback []MigrationRecord
	for _, record := range applied {
		if CompareVersions(record.Version, target) > 0 {
			toRollback = append(toRollback, record)
		}
	}
	
	// 역순으로 정렬
	sort.Slice(toRollback, func(i, j int) bool {
		return CompareVersions(toRollback[i].Version, toRollback[j].Version) > 0
	})
	
	// 롤백 실행
	for _, record := range toRollback {
		migration, err := m.source.Get(record.Version)
		if err != nil {
			return fmt.Errorf("마이그레이션 %s 로드 실패: %w", record.Version, err)
		}
		
		if err := m.executeSingleMigration(ctx, migration, DirectionDown); err != nil {
			return err
		}
	}
	
	return nil
}

// List 모든 마이그레이션 목록 반환
func (m *SQLiteMigrator) List() ([]MigrationInfo, error) {
	ctx := context.Background()
	
	if err := m.tracker.EnsureTable(ctx); err != nil {
		return nil, err
	}
	
	// 사용 가능한 마이그레이션
	migrations, err := m.source.Load()
	if err != nil {
		return nil, err
	}
	
	// 적용된 마이그레이션
	applied, err := m.tracker.GetApplied(ctx)
	if err != nil {
		return nil, err
	}
	
	appliedMap := make(map[string]MigrationRecord)
	for _, record := range applied {
		if record.Direction == DirectionUp {
			appliedMap[record.Version] = record
		}
	}
	
	// 마이그레이션 정보 생성
	var infos []MigrationInfo
	for _, migration := range migrations {
		info := MigrationInfo{
			Version:     migration.Version(),
			Description: migration.Description(),
			Status:      MigrationStatusPending,
			CanRollback: migration.CanRollback(),
		}
		
		if record, exists := appliedMap[migration.Version()]; exists {
			info.Status = record.Status
			info.AppliedAt = record.CompletedAt
			info.Duration = record.Duration
			info.Error = record.Error
		}
		
		infos = append(infos, info)
	}
	
	return infos, nil
}

// Status 현재 마이그레이션 상태 반환
func (m *SQLiteMigrator) Status() (*MigrationStatus, []MigrationInfo, error) {
	infos, err := m.List()
	if err != nil {
		return nil, nil, err
	}
	
	// 전체 상태 계산
	var overallStatus MigrationStatus = MigrationStatusCompleted
	
	for _, info := range infos {
		if info.Status == MigrationStatusFailed {
			overallStatus = MigrationStatusFailed
			break
		} else if info.Status == MigrationStatusPending {
			overallStatus = MigrationStatusPending
		}
	}
	
	return &overallStatus, infos, nil
}

// Validate 마이그레이션 파일 검증
func (m *SQLiteMigrator) Validate() error {
	migrations, err := m.source.Load()
	if err != nil {
		return err
	}
	
	// 버전 검증
	versions := make(map[string]bool)
	for _, migration := range migrations {
		version := migration.Version()
		
		if err := ValidateVersion(version); err != nil {
			return fmt.Errorf("마이그레이션 %s: %w", version, err)
		}
		
		if versions[version] {
			return fmt.Errorf("중복된 마이그레이션 버전: %s", version)
		}
		versions[version] = true
	}
	
	return nil
}

// Close 리소스 정리
func (m *SQLiteMigrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// planMigrations 실행할 마이그레이션들과 방향 결정
func (m *SQLiteMigrator) planMigrations(migrations []Migration, current, target string) ([]Migration, Direction, error) {
	// 버전별 마이그레이션 맵 생성
	migrationMap := make(map[string]Migration)
	versions := make([]string, len(migrations))
	
	for i, migration := range migrations {
		version := migration.Version()
		migrationMap[version] = migration
		versions[i] = version
	}
	
	sort.Strings(versions)
	
	// 현재와 목표 버전의 인덱스 찾기
	currentIdx := -1
	targetIdx := -1
	
	for i, version := range versions {
		if version == current {
			currentIdx = i
		}
		if version == target {
			targetIdx = i
		}
	}
	
	if targetIdx == -1 {
		return nil, "", fmt.Errorf("목표 버전을 찾을 수 없습니다: %s", target)
	}
	
	var toExecute []Migration
	var direction Direction
	
	if currentIdx < targetIdx {
		// 업그레이드
		direction = DirectionUp
		for i := currentIdx + 1; i <= targetIdx; i++ {
			toExecute = append(toExecute, migrationMap[versions[i]])
		}
	} else {
		// 다운그레이드
		direction = DirectionDown
		for i := currentIdx; i > targetIdx; i-- {
			migration := migrationMap[versions[i]]
			if !migration.CanRollback() {
				return nil, "", fmt.Errorf("마이그레이션 %s는 롤백을 지원하지 않습니다", versions[i])
			}
			toExecute = append(toExecute, migration)
		}
	}
	
	return toExecute, direction, nil
}

// executeSingleMigration 단일 마이그레이션 실행
func (m *SQLiteMigrator) executeSingleMigration(ctx context.Context, migration Migration, direction Direction) error {
	version := migration.Version()
	
	if m.options.DryRun {
		fmt.Printf("DRY RUN: %s 마이그레이션 %s (%s)\n", 
			strings.ToUpper(string(direction)), version, migration.Description())
		return nil
	}
	
	startTime := time.Now()
	
	// 트랜잭션 시작
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}
	
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	
	// 마이그레이션 실행
	if direction == DirectionUp {
		err = migration.Up(ctx, tx)
	} else {
		err = migration.Down(ctx, tx)
	}
	
	if err != nil {
		m.tracker.RecordFailed(ctx, version, direction, err)
		return NewMigrationError(version, direction, err, true)
	}
	
	// 트랜잭션 커밋
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("트랜잭션 커밋 실패: %w", err)
	}
	
	duration := time.Since(startTime)
	
	// 기록 저장
	if direction == DirectionUp {
		err = m.tracker.RecordUp(ctx, version, duration)
	} else {
		// 다운 마이그레이션 시에는 기록을 제거
		err = m.tracker.RemoveRecord(ctx, version)
	}
	
	if err != nil {
		return fmt.Errorf("마이그레이션 기록 저장 실패: %w", err)
	}
	
	if m.options.Verbose {
		fmt.Printf("✅ %s 마이그레이션 %s 완료 (%v)\n", 
			strings.ToUpper(string(direction)), version, duration)
	}
	
	return nil
}