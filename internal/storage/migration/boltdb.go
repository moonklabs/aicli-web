package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	
	"go.etcd.io/bbolt"
)

const (
	// BoltDBMigrationBucket 마이그레이션 버킷명
	BoltDBMigrationBucket = "_migrations"
)

// BoltDBMigration BoltDB 전용 마이그레이션 인터페이스
type BoltDBMigration interface {
	Migration
	
	// UpBolt BoltDB 업그레이드 마이그레이션
	UpBolt(ctx context.Context, tx *bbolt.Tx) error
	
	// DownBolt BoltDB 다운그레이드 마이그레이션
	DownBolt(ctx context.Context, tx *bbolt.Tx) error
}

// BoltDBFuncMigration 함수 기반 BoltDB 마이그레이션
type BoltDBFuncMigration struct {
	*BaseMigration
	upFunc   func(ctx context.Context, tx *bbolt.Tx) error
	downFunc func(ctx context.Context, tx *bbolt.Tx) error
}

// NewBoltDBFuncMigration 새 함수 기반 BoltDB 마이그레이션 생성
func NewBoltDBFuncMigration(
	version, description string,
	upFunc, downFunc func(ctx context.Context, tx *bbolt.Tx) error,
) *BoltDBFuncMigration {
	canRollback := downFunc != nil
	
	return &BoltDBFuncMigration{
		BaseMigration: NewBaseMigration(version, description, canRollback),
		upFunc:        upFunc,
		downFunc:      downFunc,
	}
}

// Up 업그레이드 실행 (인터페이스 호환성)
func (m *BoltDBFuncMigration) Up(ctx context.Context, db interface{}) error {
	tx, ok := db.(*bbolt.Tx)
	if !ok {
		return fmt.Errorf("BoltDB 트랜잭션이 필요합니다")
	}
	return m.UpBolt(ctx, tx)
}

// Down 다운그레이드 실행 (인터페이스 호환성)
func (m *BoltDBFuncMigration) Down(ctx context.Context, db interface{}) error {
	tx, ok := db.(*bbolt.Tx)
	if !ok {
		return fmt.Errorf("BoltDB 트랜잭션이 필요합니다")
	}
	return m.DownBolt(ctx, tx)
}

// UpBolt BoltDB 업그레이드 실행
func (m *BoltDBFuncMigration) UpBolt(ctx context.Context, tx *bbolt.Tx) error {
	if m.upFunc == nil {
		return fmt.Errorf("업그레이드 함수가 정의되지 않았습니다: %s", m.Version())
	}
	return m.upFunc(ctx, tx)
}

// DownBolt BoltDB 다운그레이드 실행
func (m *BoltDBFuncMigration) DownBolt(ctx context.Context, tx *bbolt.Tx) error {
	if !m.CanRollback() {
		return fmt.Errorf("마이그레이션 %s는 롤백을 지원하지 않습니다", m.Version())
	}
	if m.downFunc == nil {
		return fmt.Errorf("다운그레이드 함수가 정의되지 않았습니다: %s", m.Version())
	}
	return m.downFunc(ctx, tx)
}

// BoltDBTracker BoltDB 마이그레이션 추적기
type BoltDBTracker struct {
	db         *bbolt.DB
	bucketName string
}

// NewBoltDBTracker 새 BoltDB 추적기 생성
func NewBoltDBTracker(db *bbolt.DB) *BoltDBTracker {
	return &BoltDBTracker{
		db:         db,
		bucketName: BoltDBMigrationBucket,
	}
}

// EnsureTable 마이그레이션 버킷 생성
func (t *BoltDBTracker) EnsureTable(ctx context.Context) error {
	return t.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(t.bucketName))
		if err != nil {
			return fmt.Errorf("마이그레이션 버킷 생성 실패: %w", err)
		}
		return nil
	})
}

// GetApplied 적용된 마이그레이션 목록 반환
func (t *BoltDBTracker) GetApplied(ctx context.Context) ([]MigrationRecord, error) {
	var records []MigrationRecord
	
	err := t.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(t.bucketName))
		if bucket == nil {
			return nil // 버킷이 없으면 빈 목록 반환
		}
		
		return bucket.ForEach(func(k, v []byte) error {
			var record MigrationRecord
			if err := json.Unmarshal(v, &record); err != nil {
				return fmt.Errorf("마이그레이션 레코드 파싱 실패 (%s): %w", string(k), err)
			}
			
			if record.Status == MigrationStatusCompleted && record.Direction == DirectionUp {
				records = append(records, record)
			}
			
			return nil
		})
	})
	
	if err != nil {
		return nil, fmt.Errorf("적용된 마이그레이션 조회 실패: %w", err)
	}
	
	// 버전별로 정렬
	sort.Slice(records, func(i, j int) bool {
		return CompareVersions(records[i].Version, records[j].Version) < 0
	})
	
	return records, nil
}

// IsApplied 특정 버전이 적용되었는지 확인
func (t *BoltDBTracker) IsApplied(ctx context.Context, version string) (bool, error) {
	var applied bool
	
	err := t.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(t.bucketName))
		if bucket == nil {
			return nil
		}
		
		data := bucket.Get([]byte(version))
		if data == nil {
			return nil
		}
		
		var record MigrationRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return fmt.Errorf("마이그레이션 레코드 파싱 실패: %w", err)
		}
		
		applied = record.Status == MigrationStatusCompleted && record.Direction == DirectionUp
		return nil
	})
	
	return applied, err
}

// RecordUp 업그레이드 마이그레이션 기록
func (t *BoltDBTracker) RecordUp(ctx context.Context, version string, duration time.Duration) error {
	return t.recordMigration(ctx, version, DirectionUp, MigrationStatusCompleted, duration, "")
}

// RecordDown 다운그레이드 마이그레이션 기록
func (t *BoltDBTracker) RecordDown(ctx context.Context, version string, duration time.Duration) error {
	return t.recordMigration(ctx, version, DirectionDown, MigrationStatusCompleted, duration, "")
}

// RecordFailed 실패한 마이그레이션 기록
func (t *BoltDBTracker) RecordFailed(ctx context.Context, version string, direction Direction, err error) error {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return t.recordMigration(ctx, version, direction, MigrationStatusFailed, 0, errMsg)
}

// RemoveRecord 마이그레이션 기록 제거
func (t *BoltDBTracker) RemoveRecord(ctx context.Context, version string) error {
	return t.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(t.bucketName))
		if bucket == nil {
			return fmt.Errorf("마이그레이션 버킷을 찾을 수 없습니다")
		}
		
		if bucket.Get([]byte(version)) == nil {
			return fmt.Errorf("제거할 마이그레이션 기록을 찾을 수 없습니다: %s", version)
		}
		
		return bucket.Delete([]byte(version))
	})
}

// recordMigration 마이그레이션 기록 저장
func (t *BoltDBTracker) recordMigration(ctx context.Context, version string, direction Direction, status MigrationStatus, duration time.Duration, errorMsg string) error {
	now := time.Now().UTC()
	
	record := MigrationRecord{
		Version:   version,
		Direction: direction,
		Status:    status,
		StartedAt: now,
		Duration:  duration,
		Error:     errorMsg,
	}
	
	if status == MigrationStatusCompleted {
		record.CompletedAt = &now
	}
	
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("마이그레이션 레코드 직렬화 실패: %w", err)
	}
	
	return t.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(t.bucketName))
		if bucket == nil {
			return fmt.Errorf("마이그레이션 버킷을 찾을 수 없습니다")
		}
		
		return bucket.Put([]byte(version), data)
	})
}

// BoltDBMigrationSource BoltDB 마이그레이션 소스
type BoltDBMigrationSource struct {
	migrations map[string]BoltDBMigration
	versions   []string
}

// NewBoltDBMigrationSource 새 BoltDB 마이그레이션 소스 생성
func NewBoltDBMigrationSource() *BoltDBMigrationSource {
	return &BoltDBMigrationSource{
		migrations: make(map[string]BoltDBMigration),
	}
}

// Add 마이그레이션 추가
func (s *BoltDBMigrationSource) Add(migration BoltDBMigration) {
	version := migration.Version()
	s.migrations[version] = migration
	
	// 버전 목록 재생성 및 정렬
	s.versions = make([]string, 0, len(s.migrations))
	for v := range s.migrations {
		s.versions = append(s.versions, v)
	}
	sort.Strings(s.versions)
}

// Load 마이그레이션 목록 로드
func (s *BoltDBMigrationSource) Load() ([]Migration, error) {
	migrations := make([]Migration, 0, len(s.versions))
	for _, version := range s.versions {
		migrations = append(migrations, s.migrations[version])
	}
	return migrations, nil
}

// Get 특정 버전의 마이그레이션 반환
func (s *BoltDBMigrationSource) Get(version string) (Migration, error) {
	migration, exists := s.migrations[version]
	if !exists {
		return nil, fmt.Errorf("마이그레이션 버전 %s를 찾을 수 없습니다", version)
	}
	return migration, nil
}

// List 사용 가능한 모든 마이그레이션 버전 나열
func (s *BoltDBMigrationSource) List() ([]string, error) {
	return append([]string{}, s.versions...), nil
}

// BoltDBMigrator BoltDB 마이그레이션 실행기
type BoltDBMigrator struct {
	db      *bbolt.DB
	source  *BoltDBMigrationSource
	tracker MigrationTracker
	options MigrationOptions
}

// NewBoltDBMigrator 새 BoltDB 마이그레이션 실행기 생성
func NewBoltDBMigrator(db *bbolt.DB, source *BoltDBMigrationSource, options MigrationOptions) *BoltDBMigrator {
	tracker := NewBoltDBTracker(db)
	
	return &BoltDBMigrator{
		db:      db,
		source:  source,
		tracker: tracker,
		options: options,
	}
}

// Current 현재 스키마 버전 반환
func (m *BoltDBMigrator) Current() (string, error) {
	ctx := context.Background()
	
	if err := m.tracker.EnsureTable(ctx); err != nil {
		return "", err
	}
	
	applied, err := m.tracker.GetApplied(ctx)
	if err != nil {
		return "", err
	}
	
	if len(applied) == 0 {
		return "", nil
	}
	
	// 가장 높은 버전 반환
	versions := make([]string, len(applied))
	for i, record := range applied {
		versions[i] = record.Version
	}
	
	sort.Strings(versions)
	return versions[len(versions)-1], nil
}

// Migrate 특정 버전으로 마이그레이션
func (m *BoltDBMigrator) Migrate(ctx context.Context, target string) error {
	if err := m.tracker.EnsureTable(ctx); err != nil {
		return err
	}
	
	current, err := m.Current()
	if err != nil {
		return err
	}
	
	if current == target {
		return nil
	}
	
	migrations, err := m.source.Load()
	if err != nil {
		return fmt.Errorf("마이그레이션 로드 실패: %w", err)
	}
	
	toExecute, direction, err := m.planMigrations(migrations, current, target)
	if err != nil {
		return err
	}
	
	for _, migration := range toExecute {
		if err := m.executeSingleMigration(ctx, migration, direction); err != nil {
			return err
		}
	}
	
	return nil
}

// MigrateUp 최신 버전으로 업그레이드
func (m *BoltDBMigrator) MigrateUp(ctx context.Context) error {
	versions, err := m.source.List()
	if err != nil {
		return err
	}
	
	if len(versions) == 0 {
		return nil
	}
	
	sort.Strings(versions)
	latestVersion := versions[len(versions)-1]
	return m.Migrate(ctx, latestVersion)
}

// Rollback N단계 롤백
func (m *BoltDBMigrator) Rollback(ctx context.Context, steps int) error {
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
func (m *BoltDBMigrator) RollbackTo(ctx context.Context, target string) error {
	applied, err := m.tracker.GetApplied(ctx)
	if err != nil {
		return err
	}
	
	var toRollback []MigrationRecord
	for _, record := range applied {
		if CompareVersions(record.Version, target) > 0 {
			toRollback = append(toRollback, record)
		}
	}
	
	sort.Slice(toRollback, func(i, j int) bool {
		return CompareVersions(toRollback[i].Version, toRollback[j].Version) > 0
	})
	
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
func (m *BoltDBMigrator) List() ([]MigrationInfo, error) {
	ctx := context.Background()
	
	if err := m.tracker.EnsureTable(ctx); err != nil {
		return nil, err
	}
	
	migrations, err := m.source.Load()
	if err != nil {
		return nil, err
	}
	
	applied, err := m.tracker.GetApplied(ctx)
	if err != nil {
		return nil, err
	}
	
	appliedMap := make(map[string]MigrationRecord)
	for _, record := range applied {
		appliedMap[record.Version] = record
	}
	
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
func (m *BoltDBMigrator) Status() (*MigrationStatus, []MigrationInfo, error) {
	infos, err := m.List()
	if err != nil {
		return nil, nil, err
	}
	
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
func (m *BoltDBMigrator) Validate() error {
	migrations, err := m.source.Load()
	if err != nil {
		return err
	}
	
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
func (m *BoltDBMigrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// planMigrations 실행할 마이그레이션들과 방향 결정
func (m *BoltDBMigrator) planMigrations(migrations []Migration, current, target string) ([]Migration, Direction, error) {
	migrationMap := make(map[string]Migration)
	versions := make([]string, len(migrations))
	
	for i, migration := range migrations {
		version := migration.Version()
		migrationMap[version] = migration
		versions[i] = version
	}
	
	sort.Strings(versions)
	
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
		direction = DirectionUp
		for i := currentIdx + 1; i <= targetIdx; i++ {
			toExecute = append(toExecute, migrationMap[versions[i]])
		}
	} else {
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
func (m *BoltDBMigrator) executeSingleMigration(ctx context.Context, migration Migration, direction Direction) error {
	version := migration.Version()
	
	if m.options.DryRun {
		fmt.Printf("DRY RUN: %s 마이그레이션 %s (%s)\n", 
			strings.ToUpper(string(direction)), version, migration.Description())
		return nil
	}
	
	startTime := time.Now()
	
	var err error
	
	// BoltDB 트랜잭션에서 마이그레이션 실행
	if direction == DirectionUp {
		err = m.db.Update(func(tx *bbolt.Tx) error {
			return migration.Up(ctx, tx)
		})
	} else {
		err = m.db.Update(func(tx *bbolt.Tx) error {
			return migration.Down(ctx, tx)
		})
	}
	
	if err != nil {
		m.tracker.RecordFailed(ctx, version, direction, err)
		return NewMigrationError(version, direction, err, false)
	}
	
	duration := time.Since(startTime)
	
	// 기록 저장
	if direction == DirectionUp {
		err = m.tracker.RecordUp(ctx, version, duration)
	} else {
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