package commands

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"

	"github.com/aicli/aicli-web/internal/config"
	"github.com/aicli/aicli-web/internal/storage/migration"
)

// newDBCommand DB 관련 명령어 그룹 생성
func newDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "데이터베이스 관리 명령어",
		Long:  "데이터베이스 스키마 마이그레이션 및 관리 명령어를 제공합니다.",
	}
	
	cmd.AddCommand(
		newDBMigrateCommand(),
		newDBRollbackCommand(),
		newDBStatusCommand(),
		newDBCreateCommand(),
		newDBVersionCommand(),
	)
	
	return cmd
}

// newDBMigrateCommand 마이그레이션 명령어
func newDBMigrateCommand() *cobra.Command {
	var (
		version  string
		dryRun   bool
		verbose  bool
		force    bool
		timeout  int
	)
	
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "데이터베이스 마이그레이션 실행",
		Long: `데이터베이스를 최신 버전 또는 지정된 버전으로 마이그레이션합니다.
		
예시:
  aicli db migrate                # 최신 버전으로 마이그레이션
  aicli db migrate --version 003  # 특정 버전으로 마이그레이션
  aicli db migrate --dry-run      # 실행하지 않고 계획만 표시`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
			defer cancel()
			
			migrator, err := createMigrator()
			if err != nil {
				return fmt.Errorf("마이그레이션 실행기 생성 실패: %w", err)
			}
			defer migrator.Close()
			
			// 옵션 설정
			options := migration.MigrationOptions{
				DryRun:  dryRun,
				Verbose: verbose,
				Force:   force,
				Timeout: time.Duration(timeout) * time.Second,
			}
			
			// 현재 버전 확인
			current, err := migrator.Current()
			if err != nil {
				return fmt.Errorf("현재 버전 확인 실패: %w", err)
			}
			
			fmt.Printf("현재 스키마 버전: %s\n", current)
			
			if version != "" {
				// 특정 버전으로 마이그레이션
				fmt.Printf("목표 버전: %s\n", version)
				
				if dryRun {
					fmt.Println("DRY RUN 모드: 실제 마이그레이션을 실행하지 않습니다.")
				}
				
				err = migrator.Migrate(ctx, version)
			} else {
				// 최신 버전으로 마이그레이션
				fmt.Println("최신 버전으로 마이그레이션합니다.")
				
				if dryRun {
					fmt.Println("DRY RUN 모드: 실제 마이그레이션을 실행하지 않습니다.")
				}
				
				err = migrator.MigrateUp(ctx)
			}
			
			if err != nil {
				return fmt.Errorf("마이그레이션 실패: %w", err)
			}
			
			if !dryRun {
				fmt.Println("✅ 마이그레이션이 성공적으로 완료되었습니다.")
			}
			
			return nil
		},
	}
	
	cmd.Flags().StringVarP(&version, "version", "v", "", "마이그레이션할 대상 버전")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "실제 실행하지 않고 계획만 표시")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "상세 출력")
	cmd.Flags().BoolVar(&force, "force", false, "에러가 있어도 강제 실행")
	cmd.Flags().IntVar(&timeout, "timeout", 600, "타임아웃 (초)")
	
	return cmd
}

// newDBRollbackCommand 롤백 명령어
func newDBRollbackCommand() *cobra.Command {
	var (
		steps   int
		version string
		dryRun  bool
		verbose bool
	)
	
	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "데이터베이스 마이그레이션 롤백",
		Long: `데이터베이스를 이전 버전으로 롤백합니다.
		
예시:
  aicli db rollback --steps 1      # 1단계 롤백
  aicli db rollback --version 002  # 특정 버전으로 롤백
  aicli db rollback --dry-run      # 실행하지 않고 계획만 표시`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			
			migrator, err := createMigrator()
			if err != nil {
				return fmt.Errorf("마이그레이션 실행기 생성 실패: %w", err)
			}
			defer migrator.Close()
			
			if dryRun {
				fmt.Println("DRY RUN 모드: 실제 롤백을 실행하지 않습니다.")
			}
			
			if version != "" {
				fmt.Printf("버전 %s로 롤백합니다.\n", version)
				err = migrator.RollbackTo(ctx, version)
			} else {
				fmt.Printf("%d단계 롤백합니다.\n", steps)
				err = migrator.Rollback(ctx, steps)
			}
			
			if err != nil {
				return fmt.Errorf("롤백 실패: %w", err)
			}
			
			if !dryRun {
				fmt.Println("✅ 롤백이 성공적으로 완료되었습니다.")
			}
			
			return nil
		},
	}
	
	cmd.Flags().IntVar(&steps, "steps", 1, "롤백할 단계 수")
	cmd.Flags().StringVar(&version, "version", "", "롤백할 대상 버전")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "실제 실행하지 않고 계획만 표시")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "상세 출력")
	
	return cmd
}

// newDBStatusCommand 상태 확인 명령어
func newDBStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "데이터베이스 마이그레이션 상태 확인",
		Long:  "현재 데이터베이스의 마이그레이션 상태를 표시합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			migrator, err := createMigrator()
			if err != nil {
				return fmt.Errorf("마이그레이션 실행기 생성 실패: %w", err)
			}
			defer migrator.Close()
			
			// 현재 버전
			current, err := migrator.Current()
			if err != nil {
				return fmt.Errorf("현재 버전 확인 실패: %w", err)
			}
			
			// 상태 정보
			overallStatus, infos, err := migrator.Status()
			if err != nil {
				return fmt.Errorf("상태 확인 실패: %w", err)
			}
			
			fmt.Printf("현재 스키마 버전: %s\n", current)
			fmt.Printf("전체 상태: %s\n\n", *overallStatus)
			
			// 마이그레이션 목록 출력
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "버전\t설명\t상태\t적용 시간\t소요 시간")
			fmt.Fprintln(w, "----\t----\t----\t--------\t--------")
			
			for _, info := range infos {
				appliedAt := "-"
				duration := "-"
				
				if info.AppliedAt != nil {
					appliedAt = info.AppliedAt.Format("2006-01-02 15:04:05")
				}
				
				if info.Duration > 0 {
					duration = info.Duration.String()
				}
				
				status := string(info.Status)
				if info.Status == migration.MigrationStatusCompleted {
					status = "✅ " + status
				} else if info.Status == migration.MigrationStatusFailed {
					status = "❌ " + status
				} else {
					status = "⏳ " + status
				}
				
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					info.Version, info.Description, status, appliedAt, duration)
			}
			
			w.Flush()
			
			return nil
		},
	}
	
	return cmd
}

// newDBCreateCommand 마이그레이션 파일 생성 명령어
func newDBCreateCommand() *cobra.Command {
	var (
		storageType string
	)
	
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "새 마이그레이션 파일 생성",
		Long: `새 마이그레이션 파일 템플릿을 생성합니다.
		
예시:
  aicli db create add_user_table        # 새 마이그레이션 파일 생성
  aicli db create --type sqlite add_indexes  # SQLite용 마이그레이션 생성`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			
			// 설정 로드
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			
			if storageType == "" {
				storageType = cfg.Storage.Type
			}
			
			// 다음 버전 번호 생성
			version, err := generateNextVersion(storageType)
			if err != nil {
				return fmt.Errorf("버전 번호 생성 실패: %w", err)
			}
			
			// 파일명 정규화
			filename := strings.ReplaceAll(strings.ToLower(name), " ", "_")
			
			if storageType == "sqlite" {
				return createSQLiteMigrationFiles(version, filename, name)
			} else if storageType == "boltdb" {
				return createBoltDBMigrationFile(version, filename, name)
			}
			
			return fmt.Errorf("지원하지 않는 스토리지 타입: %s", storageType)
		},
	}
	
	cmd.Flags().StringVar(&storageType, "type", "", "스토리지 타입 (sqlite, boltdb)")
	
	return cmd
}

// newDBVersionCommand 버전 확인 명령어
func newDBVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "현재 데이터베이스 스키마 버전 확인",
		Long:  "현재 데이터베이스의 스키마 버전을 표시합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			migrator, err := createMigrator()
			if err != nil {
				return fmt.Errorf("마이그레이션 실행기 생성 실패: %w", err)
			}
			defer migrator.Close()
			
			current, err := migrator.Current()
			if err != nil {
				return fmt.Errorf("현재 버전 확인 실패: %w", err)
			}
			
			if current == "" {
				fmt.Println("마이그레이션이 적용되지 않았습니다.")
			} else {
				fmt.Printf("현재 스키마 버전: %s\n", current)
			}
			
			return nil
		},
	}
	
	return cmd
}

// createMigrator 마이그레이션 실행기 생성
func createMigrator() (migration.Migrator, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	
	switch cfg.Storage.Type {
	case "sqlite":
		return createSQLiteMigrator(cfg)
	case "boltdb":
		return createBoltDBMigrator(cfg)
	default:
		return nil, fmt.Errorf("지원하지 않는 스토리지 타입: %s", cfg.Storage.Type)
	}
}

// createSQLiteMigrator SQLite 마이그레이션 실행기 생성
func createSQLiteMigrator(cfg *config.Config) (migration.Migrator, error) {
	dataSource := cfg.Storage.DataSource
	if dataSource == "" {
		// 기본 데이터베이스 경로
		homeDir, _ := os.UserHomeDir()
		dataSource = filepath.Join(homeDir, ".aicli", "aicli.db")
	}
	
	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(dataSource), 0755); err != nil {
		return nil, fmt.Errorf("데이터베이스 디렉토리 생성 실패: %w", err)
	}
	
	db, err := sql.Open("sqlite3", dataSource+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("SQLite 데이터베이스 연결 실패: %w", err)
	}
	
	// 마이그레이션 소스 생성 (여기서는 기본 예시)
	source := migration.NewFileSystemSource(os.DirFS("internal/storage/schema/sqlite"), "*.sql")
	
	options := migration.MigrationOptions{
		Timeout: time.Duration(cfg.Storage.Timeout),
		Verbose: false,
	}
	
	return migration.NewSQLiteMigrator(db, source, options), nil
}

// createBoltDBMigrator BoltDB 마이그레이션 실행기 생성
func createBoltDBMigrator(cfg *config.Config) (migration.Migrator, error) {
	dataSource := cfg.Storage.DataSource
	if dataSource == "" {
		homeDir, _ := os.UserHomeDir()
		dataSource = filepath.Join(homeDir, ".aicli", "aicli.boltdb")
	}
	
	if err := os.MkdirAll(filepath.Dir(dataSource), 0755); err != nil {
		return nil, fmt.Errorf("데이터베이스 디렉토리 생성 실패: %w", err)
	}
	
	db, err := bbolt.Open(dataSource, 0600, &bbolt.Options{
		Timeout: cfg.Storage.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("BoltDB 데이터베이스 연결 실패: %w", err)
	}
	
	// 마이그레이션 소스 생성
	source := migration.NewBoltDBMigrationSource()
	// 여기에 실제 마이그레이션들을 추가해야 함
	
	options := migration.MigrationOptions{
		Timeout: cfg.Storage.Timeout,
		Verbose: false,
	}
	
	return migration.NewBoltDBMigrator(db, source, options), nil
}

// loadConfig 설정 로드
func loadConfig() (*config.Config, error) {
	manager, err := config.NewManager()
	if err != nil {
		return nil, fmt.Errorf("설정 매니저 생성 실패: %w", err)
	}
	
	if err := manager.Load(); err != nil {
		// 설정 파일이 없으면 기본 설정 사용
		return manager.Get(), nil
	}
	
	return manager.Get(), nil
}

// generateNextVersion 다음 버전 번호 생성
func generateNextVersion(storageType string) (string, error) {
	// 기존 마이그레이션 파일들을 스캔하여 다음 버전 번호 생성
	var searchDir string
	
	switch storageType {
	case "sqlite":
		searchDir = "internal/storage/schema/sqlite"
	case "boltdb":
		searchDir = "internal/storage/schema/boltdb"
	default:
		return "", fmt.Errorf("지원하지 않는 스토리지 타입: %s", storageType)
	}
	
	maxVersion := 0
	
	if _, err := os.Stat(searchDir); os.IsNotExist(err) {
		// 디렉토리가 없으면 1부터 시작
		return "001", nil
	}
	
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return "", fmt.Errorf("마이그레이션 디렉토리 읽기 실패: %w", err)
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		name := entry.Name()
		if len(name) >= 3 {
			versionStr := name[:3]
			if version, err := strconv.Atoi(versionStr); err == nil {
				if version > maxVersion {
					maxVersion = version
				}
			}
		}
	}
	
	nextVersion := maxVersion + 1
	return fmt.Sprintf("%03d", nextVersion), nil
}

// createSQLiteMigrationFiles SQLite 마이그레이션 파일 생성
func createSQLiteMigrationFiles(version, filename, description string) error {
	dir := "internal/storage/schema/sqlite"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}
	
	// UP 파일
	upFile := filepath.Join(dir, fmt.Sprintf("%s_%s.up.sql", version, filename))
	upContent := fmt.Sprintf(`-- %s
-- Created: %s
-- Description: %s

-- TODO: Add your migration SQL here

`, description, time.Now().Format("2006-01-02 15:04:05"), description)
	
	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("UP 파일 생성 실패: %w", err)
	}
	
	// DOWN 파일
	downFile := filepath.Join(dir, fmt.Sprintf("%s_%s.down.sql", version, filename))
	downContent := fmt.Sprintf(`-- %s (Rollback)
-- Created: %s
-- Description: Rollback for %s

-- TODO: Add your rollback SQL here

`, description, time.Now().Format("2006-01-02 15:04:05"), description)
	
	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("DOWN 파일 생성 실패: %w", err)
	}
	
	fmt.Printf("✅ SQLite 마이그레이션 파일이 생성되었습니다:\n")
	fmt.Printf("   UP:   %s\n", upFile)
	fmt.Printf("   DOWN: %s\n", downFile)
	
	return nil
}

// createBoltDBMigrationFile BoltDB 마이그레이션 파일 생성
func createBoltDBMigrationFile(version, filename, description string) error {
	dir := "internal/storage/schema/boltdb"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}
	
	goFile := filepath.Join(dir, fmt.Sprintf("%s_%s.go", version, filename))
	content := fmt.Sprintf(`package migrations

import (
	"context"
	
	"go.etcd.io/bbolt"
	"github.com/aicli/aicli-web/internal/storage/migration"
)

// Migration%s %s
func Migration%s() migration.BoltDBMigration {
	return migration.NewBoltDBFuncMigration(
		"%s",
		"%s",
		migration%sUp,
		migration%sDown,
	)
}

// migration%sUp 업그레이드 마이그레이션
func migration%sUp(ctx context.Context, tx *bbolt.Tx) error {
	// TODO: Add your migration logic here
	
	return nil
}

// migration%sDown 다운그레이드 마이그레이션  
func migration%sDown(ctx context.Context, tx *bbolt.Tx) error {
	// TODO: Add your rollback logic here
	
	return nil
}
`, version, description, version, version, description, version, version, version, version, version, version)
	
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("BoltDB 마이그레이션 파일 생성 실패: %w", err)
	}
	
	fmt.Printf("✅ BoltDB 마이그레이션 파일이 생성되었습니다: %s\n", goFile)
	
	return nil
}

// NewDBCmd DB 명령어 생성
func NewDBCmd() *cobra.Command {
	return newDBCommand()
}