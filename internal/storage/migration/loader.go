package migration

import (
	"context"
	"crypto/md5"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SQLMigration SQL 기반 마이그레이션
type SQLMigration struct {
	*BaseMigration
	upSQL   string
	downSQL string
}

// NewSQLMigration 새 SQL 마이그레이션 생성
func NewSQLMigration(version, description, upSQL, downSQL string) *SQLMigration {
	canRollback := strings.TrimSpace(downSQL) != ""
	
	return &SQLMigration{
		BaseMigration: NewBaseMigration(version, description, canRollback),
		upSQL:         upSQL,
		downSQL:       downSQL,
	}
}

// Up SQL 업그레이드 실행
func (m *SQLMigration) Up(ctx context.Context, db interface{}) error {
	if strings.TrimSpace(m.upSQL) == "" {
		return fmt.Errorf("UP SQL이 비어있습니다: %s", m.Version())
	}
	
	// db는 *sql.DB 또는 *sql.Tx 타입이어야 함
	execer, ok := db.(interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (interface{}, error)
	})
	
	if !ok {
		return fmt.Errorf("데이터베이스 실행기가 지원되지 않는 타입입니다")
	}
	
	_, err := execer.ExecContext(ctx, m.upSQL)
	return err
}

// Down SQL 다운그레이드 실행
func (m *SQLMigration) Down(ctx context.Context, db interface{}) error {
	if !m.CanRollback() {
		return fmt.Errorf("마이그레이션 %s는 롤백을 지원하지 않습니다", m.Version())
	}
	
	if strings.TrimSpace(m.downSQL) == "" {
		return fmt.Errorf("DOWN SQL이 비어있습니다: %s", m.Version())
	}
	
	execer, ok := db.(interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (interface{}, error)
	})
	
	if !ok {
		return fmt.Errorf("데이터베이스 실행기가 지원되지 않는 타입입니다")
	}
	
	_, err := execer.ExecContext(ctx, m.downSQL)
	return err
}

// GetUpSQL UP SQL 반환
func (m *SQLMigration) GetUpSQL() string {
	return m.upSQL
}

// GetDownSQL DOWN SQL 반환
func (m *SQLMigration) GetDownSQL() string {
	return m.downSQL
}

// Checksum 마이그레이션 체크섬 계산
func (m *SQLMigration) Checksum() string {
	content := m.upSQL + "|" + m.downSQL
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// MigrationFile 마이그레이션 파일 정보
type MigrationFile struct {
	Path        string
	Version     string
	Description string
	Direction   Direction
	Content     string
}

// FileSystemSource 파일 시스템 마이그레이션 소스
type FileSystemSource struct {
	fs        fs.FS
	pattern   string
	migrations map[string]*SQLMigration
}

// NewFileSystemSource 파일 시스템 소스 생성
func NewFileSystemSource(filesystem fs.FS, pattern string) *FileSystemSource {
	return &FileSystemSource{
		fs:        filesystem,
		pattern:   pattern,
		migrations: make(map[string]*SQLMigration),
	}
}

// NewEmbedSource embed.FS 소스 생성
func NewEmbedSource(embedFS embed.FS, pattern string) *FileSystemSource {
	return NewFileSystemSource(embedFS, pattern)
}

// Load 마이그레이션 파일들을 로드
func (s *FileSystemSource) Load() ([]Migration, error) {
	files, err := s.scanMigrationFiles()
	if err != nil {
		return nil, fmt.Errorf("마이그레이션 파일 스캔 실패: %w", err)
	}
	
	// 버전별로 파일들을 그룹화
	versionFiles := make(map[string][]MigrationFile)
	for _, file := range files {
		versionFiles[file.Version] = append(versionFiles[file.Version], file)
	}
	
	// 각 버전에 대해 마이그레이션 생성
	for version, files := range versionFiles {
		migration, err := s.createMigration(version, files)
		if err != nil {
			return nil, fmt.Errorf("마이그레이션 생성 실패 (버전 %s): %w", version, err)
		}
		s.migrations[version] = migration
	}
	
	// 정렬된 마이그레이션 목록 반환
	versions := s.getSortedVersions()
	migrations := make([]Migration, 0, len(versions))
	for _, version := range versions {
		migrations = append(migrations, s.migrations[version])
	}
	
	return migrations, nil
}

// Get 특정 버전의 마이그레이션 반환
func (s *FileSystemSource) Get(version string) (Migration, error) {
	if len(s.migrations) == 0 {
		_, err := s.Load()
		if err != nil {
			return nil, err
		}
	}
	
	migration, exists := s.migrations[version]
	if !exists {
		return nil, fmt.Errorf("마이그레이션 버전 %s를 찾을 수 없습니다", version)
	}
	
	return migration, nil
}

// List 사용 가능한 모든 마이그레이션 버전 나열
func (s *FileSystemSource) List() ([]string, error) {
	if len(s.migrations) == 0 {
		_, err := s.Load()
		if err != nil {
			return nil, err
		}
	}
	
	return s.getSortedVersions(), nil
}

// scanMigrationFiles 마이그레이션 파일들을 스캔
func (s *FileSystemSource) scanMigrationFiles() ([]MigrationFile, error) {
	var files []MigrationFile
	
	err := fs.WalkDir(s.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		// 파일 패턴 매칭
		matched, err := filepath.Match(s.pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		
		if !matched && !s.isValidMigrationFile(path) {
			return nil
		}
		
		// 파일 파싱
		file, err := s.parseMigrationFile(path)
		if err != nil {
			return fmt.Errorf("파일 파싱 실패 %s: %w", path, err)
		}
		
		files = append(files, *file)
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return files, nil
}

// isValidMigrationFile 유효한 마이그레이션 파일인지 확인
func (s *FileSystemSource) isValidMigrationFile(path string) bool {
	base := filepath.Base(path)
	
	// SQL 파일 패턴: 001_name.up.sql, 001_name.down.sql
	sqlPattern := regexp.MustCompile(`^(\d{3,})_(.+)\.(up|down)\.sql$`)
	
	return sqlPattern.MatchString(base)
}

// parseMigrationFile 마이그레이션 파일 파싱
func (s *FileSystemSource) parseMigrationFile(path string) (*MigrationFile, error) {
	// 파일 내용 읽기
	content, err := fs.ReadFile(s.fs, path)
	if err != nil {
		return nil, fmt.Errorf("파일 읽기 실패: %w", err)
	}
	
	base := filepath.Base(path)
	
	// 파일명 파싱
	sqlPattern := regexp.MustCompile(`^(\d{3,})_(.+)\.(up|down)\.sql$`)
	matches := sqlPattern.FindStringSubmatch(base)
	
	if len(matches) != 4 {
		return nil, fmt.Errorf("파일명 형식이 잘못되었습니다: %s", base)
	}
	
	version := matches[1]
	description := strings.ReplaceAll(matches[2], "_", " ")
	directionStr := matches[3]
	
	var direction Direction
	switch directionStr {
	case "up":
		direction = DirectionUp
	case "down":
		direction = DirectionDown
	default:
		return nil, fmt.Errorf("지원하지 않는 방향: %s", directionStr)
	}
	
	return &MigrationFile{
		Path:        path,
		Version:     version,
		Description: description,
		Direction:   direction,
		Content:     string(content),
	}, nil
}

// createMigration 파일들로부터 마이그레이션 생성
func (s *FileSystemSource) createMigration(version string, files []MigrationFile) (*SQLMigration, error) {
	var upSQL, downSQL string
	var description string
	
	for _, file := range files {
		if file.Description != "" && description == "" {
			description = file.Description
		}
		
		switch file.Direction {
		case DirectionUp:
			upSQL = file.Content
		case DirectionDown:
			downSQL = file.Content
		}
	}
	
	if strings.TrimSpace(upSQL) == "" {
		return nil, fmt.Errorf("UP SQL이 없습니다 (버전 %s)", version)
	}
	
	if description == "" {
		description = fmt.Sprintf("Migration %s", version)
	}
	
	return NewSQLMigration(version, description, upSQL, downSQL), nil
}

// getSortedVersions 정렬된 버전 목록 반환
func (s *FileSystemSource) getSortedVersions() []string {
	versions := make([]string, 0, len(s.migrations))
	for version := range s.migrations {
		versions = append(versions, version)
	}
	
	sort.Strings(versions)
	return versions
}

// MigrationLoader 마이그레이션 로더
type MigrationLoader struct {
	sources map[string]MigrationSource
}

// NewMigrationLoader 새 마이그레이션 로더 생성
func NewMigrationLoader() *MigrationLoader {
	return &MigrationLoader{
		sources: make(map[string]MigrationSource),
	}
}

// AddSource 마이그레이션 소스 추가
func (l *MigrationLoader) AddSource(name string, source MigrationSource) {
	l.sources[name] = source
}

// LoadAll 모든 소스에서 마이그레이션 로드
func (l *MigrationLoader) LoadAll() (map[string][]Migration, error) {
	result := make(map[string][]Migration)
	
	for name, source := range l.sources {
		migrations, err := source.Load()
		if err != nil {
			return nil, fmt.Errorf("소스 %s에서 마이그레이션 로드 실패: %w", name, err)
		}
		result[name] = migrations
	}
	
	return result, nil
}

// LoadBySource 특정 소스에서 마이그레이션 로드
func (l *MigrationLoader) LoadBySource(sourceName string) ([]Migration, error) {
	source, exists := l.sources[sourceName]
	if !exists {
		return nil, fmt.Errorf("소스 %s를 찾을 수 없습니다", sourceName)
	}
	
	return source.Load()
}

// GetSources 등록된 소스 이름들 반환
func (l *MigrationLoader) GetSources() []string {
	sources := make([]string, 0, len(l.sources))
	for name := range l.sources {
		sources = append(sources, name)
	}
	
	sort.Strings(sources)
	return sources
}

// ValidateMigrations 마이그레이션 파일들의 유효성 검사
func (l *MigrationLoader) ValidateMigrations() error {
	allMigrations, err := l.LoadAll()
	if err != nil {
		return err
	}
	
	for sourceName, migrations := range allMigrations {
		if err := l.validateMigrationSequence(sourceName, migrations); err != nil {
			return err
		}
	}
	
	return nil
}

// validateMigrationSequence 마이그레이션 순서 검증
func (l *MigrationLoader) validateMigrationSequence(sourceName string, migrations []Migration) error {
	if len(migrations) == 0 {
		return nil
	}
	
	// 버전 정렬
	versions := make([]string, len(migrations))
	for i, migration := range migrations {
		versions[i] = migration.Version()
	}
	sort.Strings(versions)
	
	// 순서 검증
	for i := 1; i < len(versions); i++ {
		current := versions[i]
		previous := versions[i-1]
		
		if CompareVersions(current, previous) <= 0 {
			return fmt.Errorf("소스 %s: 마이그레이션 버전 순서가 잘못되었습니다. %s 다음에 %s", 
				sourceName, previous, current)
		}
	}
	
	return nil
}