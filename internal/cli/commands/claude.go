package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/olekukonko/tablewriter"

	"github.com/aicli/aicli-web/internal/claude"
	"github.com/aicli/aicli-web/internal/cli/output"
	"github.com/aicli/aicli-web/internal/storage"
)

// Claude 명령어 옵션
type ClaudeOptions struct {
	WorkspaceID  string
	SystemPrompt string
	MaxTurns     int
	Tools        []string
	Stream       bool
	Format       string
	SessionID    string
}

// Claude 인터랙티브 옵션
type InteractiveOptions struct {
	WorkspaceID string
	SessionName string
	Model       string
	MaxTurns    int
}

// NewClaudeCommand는 Claude CLI 관련 명령어를 반환합니다
func NewClaudeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Claude CLI 관련 명령어",
		Long: `Claude CLI와의 통합을 위한 명령어 모음입니다.

Claude CLI를 직접 실행하고, 세션을 관리하며, 인터랙티브 모드를 제공합니다.
모든 실행은 격리된 환경에서 이루어지며, 실시간 스트리밍을 지원합니다.

주요 기능:
  • Claude CLI 프로세스 실행 및 관리
  • 실시간 출력 스트리밍
  • 세션 기반 컨텍스트 관리
  • 인터랙티브 채팅 모드
  • 다양한 출력 형식 지원`,
		Example: `  # 단일 명령 실행
  aicli claude run "implement login feature" --workspace myproject

  # 인터랙티브 채팅 시작
  aicli claude chat --workspace myproject

  # 세션 목록 조회
  aicli claude session list

  # 특정 세션으로 명령 실행
  aicli claude run "continue implementation" --session abc123`,
	}

	// 하위 명령어 추가
	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newChatCommand())
	cmd.AddCommand(newSessionCommand())
	cmd.AddCommand(newStatusCommand())

	return cmd
}

// newRunCommand는 Claude run 명령어를 생성합니다
func newRunCommand() *cobra.Command {
	opts := &ClaudeOptions{}

	cmd := &cobra.Command{
		Use:   "run [prompt]",
		Short: "Claude에 단일 프롬프트 실행",
		Long: `Claude CLI에 단일 프롬프트를 전송하고 결과를 받습니다.

워크스페이스 또는 기존 세션 컨텍스트에서 실행할 수 있으며,
실시간 스트리밍으로 진행 상황을 확인할 수 있습니다.`,
		Args: cobra.ExactArgs(1),
		Example: `  # 기본 실행
  aicli claude run "implement user authentication"

  # 워크스페이스 지정
  aicli claude run "add tests" --workspace myproject

  # JSON 형식 출력
  aicli claude run "refactor code" --format json

  # 기존 세션 사용
  aicli claude run "continue work" --session abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClaude(cmd.Context(), opts, args[0])
		},
	}

	// 플래그 정의
	cmd.Flags().StringVarP(&opts.WorkspaceID, "workspace", "w", "", "워크스페이스 ID")
	cmd.Flags().StringVar(&opts.SystemPrompt, "system", "", "시스템 프롬프트")
	cmd.Flags().IntVar(&opts.MaxTurns, "max-turns", 10, "최대 턴 수")
	cmd.Flags().StringSliceVar(&opts.Tools, "tools", nil, "사용 가능한 도구")
	cmd.Flags().BoolVar(&opts.Stream, "stream", true, "실시간 스트리밍")
	cmd.Flags().StringVar(&opts.Format, "format", "text", "출력 형식 (text|json|markdown)")
	cmd.Flags().StringVarP(&opts.SessionID, "session", "s", "", "기존 세션 ID")

	// 플래그 자동완성
	cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json", "markdown"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("workspace", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// 워크스페이스 목록을 동적으로 가져오기
		store, err := storage.New()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		defer store.Close()

		workspaces, err := store.Workspace().List(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var names []string
		for _, ws := range workspaces {
			names = append(names, ws.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// newChatCommand는 인터랙티브 채팅 명령어를 생성합니다
func newChatCommand() *cobra.Command {
	opts := &InteractiveOptions{}

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Claude와 인터랙티브 채팅",
		Long: `Claude CLI와 인터랙티브 채팅 모드를 시작합니다.

연속적인 대화가 가능하며, 세션이 자동으로 관리됩니다.
특수 명령어를 통해 세션 제어 및 설정 변경이 가능합니다.`,
		Example: `  # 기본 채팅 시작
  aicli claude chat

  # 워크스페이스 지정
  aicli claude chat --workspace myproject

  # 세션 이름 지정
  aicli claude chat --session-name "feature-development"

  # 특수 명령어 (채팅 중 사용):
  /help    - 도움말 표시
  /exit    - 채팅 종료
  /clear   - 화면 지우기
  /session - 현재 세션 정보
  /save    - 세션 저장`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveChat(cmd.Context(), opts)
		},
	}

	// 플래그 정의
	cmd.Flags().StringVarP(&opts.WorkspaceID, "workspace", "w", "", "워크스페이스 ID")
	cmd.Flags().StringVar(&opts.SessionName, "session-name", "", "세션 이름")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Claude 모델 선택")
	cmd.Flags().IntVar(&opts.MaxTurns, "max-turns", 50, "최대 턴 수")

	return cmd
}

// newSessionCommand는 세션 관리 명령어를 생성합니다
func newSessionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Claude 세션 관리",
		Long: `Claude 세션의 생성, 조회, 관리를 위한 명령어입니다.

세션은 Claude CLI와의 지속적인 대화 컨텍스트를 제공하며,
여러 명령 실행 간에 컨텍스트를 유지할 수 있습니다.`,
	}

	cmd.AddCommand(newSessionListCommand())
	cmd.AddCommand(newSessionShowCommand())
	cmd.AddCommand(newSessionCloseCommand())
	cmd.AddCommand(newSessionLogsCommand())

	return cmd
}

// newStatusCommand는 Claude 상태 조회 명령어를 생성합니다
func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Claude CLI 상태 조회",
		Long: `현재 실행 중인 Claude 프로세스와 세션의 상태를 조회합니다.

활성 세션, 리소스 사용량, 성능 지표 등을 확인할 수 있습니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showClaudeStatus(cmd.Context())
		},
	}

	return cmd
}

// runClaude는 Claude CLI에 단일 명령을 실행합니다
func runClaude(ctx context.Context, opts *ClaudeOptions, prompt string) error {
	// 스토리지 초기화
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("스토리지 초기화 실패: %w", err)
	}
	defer store.Close()

	// 세션 매니저 초기화
	sessionManager := claude.NewSessionManager(store.Session())

	// 워크스페이스 ID가 지정되지 않은 경우 기본값 사용
	if opts.WorkspaceID == "" && opts.SessionID == "" {
		opts.WorkspaceID = "default"
	}

	var session *claude.Session
	var sessionID string

	// 기존 세션 사용 또는 새 세션 생성
	if opts.SessionID != "" {
		session, err = sessionManager.Get(ctx, opts.SessionID)
		if err != nil {
			return fmt.Errorf("세션 조회 실패: %w", err)
		}
		sessionID = opts.SessionID
	} else {
		// 새 세션 생성
		config := &claude.SessionConfig{
			WorkspaceID:  opts.WorkspaceID,
			SystemPrompt: opts.SystemPrompt,
			MaxTurns:     opts.MaxTurns,
			Model:        viper.GetString("claude.model"),
			Tools:        opts.Tools,
		}

		sessionID, err = sessionManager.Create(ctx, config)
		if err != nil {
			return fmt.Errorf("세션 생성 실패: %w", err)
		}

		session, err = sessionManager.Get(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("생성된 세션 조회 실패: %w", err)
		}
	}

	// 출력 포맷터 초기화
	formatter, err := createFormatter(opts.Format)
	if err != nil {
		return fmt.Errorf("포맷터 생성 실패: %w", err)
	}

	// 인터럽트 처리 설정
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\n실행이 중단되었습니다...")
		cancel()
	}()

	fmt.Printf("Claude 실행 중... (세션: %s)\n", sessionID)
	
	if opts.Stream {
		return executeWithStreaming(ctx, session, prompt, formatter)
	} else {
		return executeWithoutStreaming(ctx, session, prompt, formatter)
	}
}

// runInteractiveChat는 인터랙티브 채팅 모드를 실행합니다
func runInteractiveChat(ctx context.Context, opts *InteractiveOptions) error {
	// 스토리지 초기화
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("스토리지 초기화 실패: %w", err)
	}
	defer store.Close()

	// 세션 매니저 초기화
	sessionManager := claude.NewSessionManager(store.Session())

	// 워크스페이스 ID 기본값 설정
	if opts.WorkspaceID == "" {
		opts.WorkspaceID = "default"
	}

	// 새 세션 생성
	config := &claude.SessionConfig{
		WorkspaceID: opts.WorkspaceID,
		Name:        opts.SessionName,
		Model:       opts.Model,
		MaxTurns:    opts.MaxTurns,
	}

	sessionID, err := sessionManager.Create(ctx, config)
	if err != nil {
		return fmt.Errorf("세션 생성 실패: %w", err)
	}

	session, err := sessionManager.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("세션 조회 실패: %w", err)
	}

	// 인터랙티브 모드 시작
	return startInteractiveMode(ctx, session, sessionManager)
}

// startInteractiveMode는 실제 인터랙티브 채팅을 처리합니다
func startInteractiveMode(ctx context.Context, session *claude.Session, manager *claude.SessionManager) error {
	fmt.Printf("🤖 Claude 인터랙티브 모드 (세션: %s)\n", session.ID)
	fmt.Println("특수 명령어: /help, /exit, /clear, /session, /save")
	fmt.Println("Ctrl+C로 종료 가능합니다.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	formatter, err := createFormatter("text")
	if err != nil {
		return fmt.Errorf("포맷터 생성 실패: %w", err)
	}

	// 인터럽트 처리
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n👋 채팅을 종료합니다...")
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// 사용자 입력 받기
			fmt.Print("You> ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("입력 읽기 실패: %w", err)
			}

			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}

			// 특수 명령어 처리
			if strings.HasPrefix(input, "/") {
				if handled, err := handleSpecialCommand(ctx, input, session, manager); handled {
					if err != nil {
						fmt.Printf("❌ 오류: %v\n", err)
					}
					continue
				}
			}

			// Claude에게 메시지 전송
			fmt.Print("Claude> ")
			err = executeWithStreaming(ctx, session, input, formatter)
			if err != nil {
				fmt.Printf("❌ 오류: %v\n", err)
			}
			fmt.Println()
		}
	}
}

// handleSpecialCommand는 특수 명령어를 처리합니다
func handleSpecialCommand(ctx context.Context, command string, session *claude.Session, manager *claude.SessionManager) (bool, error) {
	switch command {
	case "/help":
		fmt.Println("사용 가능한 특수 명령어:")
		fmt.Println("  /help    - 이 도움말 표시")
		fmt.Println("  /exit    - 채팅 종료")
		fmt.Println("  /clear   - 화면 지우기")
		fmt.Println("  /session - 현재 세션 정보 표시")
		fmt.Println("  /save    - 세션 저장")
		return true, nil

	case "/exit":
		fmt.Println("👋 채팅을 종료합니다...")
		os.Exit(0)
		return true, nil

	case "/clear":
		fmt.Print("\033[2J\033[H") // ANSI 이스케이프 시퀀스로 화면 지우기
		return true, nil

	case "/session":
		fmt.Printf("세션 정보:\n")
		fmt.Printf("  ID: %s\n", session.ID)
		fmt.Printf("  워크스페이스: %s\n", session.Config.WorkspaceID)
		fmt.Printf("  상태: %s\n", session.State)
		fmt.Printf("  생성 시간: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
		return true, nil

	case "/save":
		fmt.Println("💾 세션이 자동으로 저장됩니다.")
		return true, nil

	default:
		return false, nil
	}
}

// 세션 목록 명령어
func newSessionListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "세션 목록 조회",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listSessions(cmd.Context())
		},
	}
	return cmd
}

// 세션 상세 정보 명령어
func newSessionShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [session-id]",
		Short: "세션 상세 정보 조회",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showSession(cmd.Context(), args[0])
		},
	}
	return cmd
}

// 세션 종료 명령어
func newSessionCloseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close [session-id]",
		Short: "세션 종료",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return closeSession(cmd.Context(), args[0])
		},
	}
	return cmd
}

// 세션 로그 명령어
func newSessionLogsCommand() *cobra.Command {
	var follow bool
	var lines int

	cmd := &cobra.Command{
		Use:   "logs [session-id]",
		Short: "세션 로그 조회",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showSessionLogs(cmd.Context(), args[0], follow, lines)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "실시간 로그 스트리밍")
	cmd.Flags().IntVarP(&lines, "lines", "n", 50, "출력할 로그 라인 수")

	return cmd
}

// 유틸리티 함수들
func createFormatter(format string) (output.ClaudeFormatter, error) {
	switch format {
	case "text":
		return output.NewTextFormatter(true, true), nil
	case "json":
		return output.NewJSONFormatter(true), nil
	case "markdown":
		return output.NewMarkdownFormatter(true), nil
	default:
		return nil, fmt.Errorf("지원하지 않는 출력 형식: %s", format)
	}
}

// executeWithStreaming은 스트리밍으로 Claude를 실행합니다
func executeWithStreaming(ctx context.Context, session *claude.Session, prompt string, formatter output.ClaudeFormatter) error {
	// 실제 Claude CLI 통합을 위한 시뮬레이션
	// TODO: 실제 ProcessManager와 통합 후 이 코드를 대체
	
	messages := []claude.Message{
		{
			Type:    "system",
			Content: "Claude CLI 실행을 시작합니다...",
			ID:      "msg_start",
			Meta:    map[string]interface{}{"status": "starting"},
		},
		{
			Type:    "text",
			Content: "이해했습니다. 요청하신 작업을 시작하겠습니다.",
			ID:      "msg_1",
			Meta:    map[string]interface{}{"timestamp": time.Now()},
		},
		{
			Type:    "text", 
			Content: "코드를 분석하고 있습니다...",
			ID:      "msg_2",
			Meta:    map[string]interface{}{"progress": 0.3},
		},
		{
			Type:    "text",
			Content: "구현 방안을 검토 중입니다.",
			ID:      "msg_3", 
			Meta:    map[string]interface{}{"progress": 0.7},
		},
		{
			Type:    "text",
			Content: "작업을 완료했습니다.",
			ID:      "msg_4",
			Meta:    map[string]interface{}{"progress": 1.0},
		},
		{
			Type:    "system",
			Content: "Claude CLI 실행이 완료되었습니다.",
			ID:      "msg_complete",
			Meta:    map[string]interface{}{"status": "completed"},
		},
	}

	for i, msg := range messages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 시뮬레이션 지연
			time.Sleep(500 * time.Millisecond)
			
			// 메시지 출력
			output := formatter.FormatMessage(&msg)
			fmt.Print(output)
			
			if i < len(messages)-1 {
				fmt.Print("\n")
			}
		}
	}
	
	return nil
}

// executeWithoutStreaming은 일괄 처리로 Claude를 실행합니다
func executeWithoutStreaming(ctx context.Context, session *claude.Session, prompt string, formatter output.ClaudeFormatter) error {
	// 스트리밍과 동일한 메시지를 생성하되, 모든 메시지를 수집한 후 출력
	// TODO: 실제 ProcessManager와 통합 후 이 코드를 대체
	
	messages := []claude.Message{
		{
			Type:    "system",
			Content: "Claude CLI 실행을 시작합니다...",
			ID:      "msg_start",
			Meta:    map[string]interface{}{"status": "starting"},
		},
		{
			Type:    "text",
			Content: "이해했습니다. 요청하신 작업을 시작하겠습니다.",
			ID:      "msg_1",
			Meta:    map[string]interface{}{"timestamp": time.Now()},
		},
		{
			Type:    "text", 
			Content: "코드를 분석하고 있습니다...",
			ID:      "msg_2",
			Meta:    map[string]interface{}{"progress": 0.3},
		},
		{
			Type:    "text",
			Content: "구현 방안을 검토 중입니다.",
			ID:      "msg_3", 
			Meta:    map[string]interface{}{"progress": 0.7},
		},
		{
			Type:    "text",
			Content: "작업을 완료했습니다.",
			ID:      "msg_4",
			Meta:    map[string]interface{}{"progress": 1.0},
		},
		{
			Type:    "system",
			Content: "Claude CLI 실행이 완료되었습니다.",
			ID:      "msg_complete",
			Meta:    map[string]interface{}{"status": "completed"},
		},
	}

	// 일괄 처리 시뮬레이션 (지연 없이 모든 메시지 처리)
	fmt.Println("비스트리밍 모드로 Claude 실행 중...")
	time.Sleep(2 * time.Second) // 전체 실행 시뮬레이션
	
	// 모든 메시지를 일괄 출력
	for i, msg := range messages {
		output := formatter.FormatMessage(&msg)
		fmt.Print(output)
		
		if i < len(messages)-1 {
			fmt.Print("\n")
		}
	}
	
	return nil
}

// listSessions는 세션 목록을 표시합니다
func listSessions(ctx context.Context) error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("스토리지 초기화 실패: %w", err)
	}
	defer store.Close()

	sessionManager := claude.NewSessionManager(store.Session())
	sessions, err := sessionManager.List(ctx)
	if err != nil {
		return fmt.Errorf("세션 목록 조회 실패: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("등록된 세션이 없습니다.")
		return nil
	}

	// 테이블 형식으로 출력
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Workspace", "State", "Created", "Last Active"})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, session := range sessions {
		lastActive := "Never"
		if !session.LastActiveAt.IsZero() {
			lastActive = session.LastActiveAt.Format("15:04:05")
		}

		table.Append([]string{
			session.ID[:8] + "...", // ID를 짧게 표시
			session.Config.WorkspaceID,
			string(session.State),
			session.CreatedAt.Format("2006-01-02 15:04"),
			lastActive,
		})
	}

	table.Render()
	return nil
}

// showSession은 세션 상세 정보를 표시합니다
func showSession(ctx context.Context, sessionID string) error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("스토리지 초기화 실패: %w", err)
	}
	defer store.Close()

	sessionManager := claude.NewSessionManager(store.Session())
	session, err := sessionManager.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("세션 조회 실패: %w", err)
	}

	fmt.Printf("세션 상세 정보:\n")
	fmt.Printf("  ID: %s\n", session.ID)
	fmt.Printf("  이름: %s\n", session.Config.Name)
	fmt.Printf("  워크스페이스: %s\n", session.Config.WorkspaceID)
	fmt.Printf("  상태: %s\n", session.State)
	fmt.Printf("  모델: %s\n", session.Config.Model)
	fmt.Printf("  최대 턴: %d\n", session.Config.MaxTurns)
	fmt.Printf("  생성 시간: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
	if !session.LastActiveAt.IsZero() {
		fmt.Printf("  마지막 활동: %s\n", session.LastActiveAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// closeSession은 세션을 종료합니다
func closeSession(ctx context.Context, sessionID string) error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("스토리지 초기화 실패: %w", err)
	}
	defer store.Close()

	sessionManager := claude.NewSessionManager(store.Session())
	err = sessionManager.Close(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("세션 종료 실패: %w", err)
	}

	fmt.Printf("✅ 세션 %s가 종료되었습니다.\n", sessionID)
	return nil
}

// showSessionLogs는 세션 로그를 표시합니다
func showSessionLogs(ctx context.Context, sessionID string, follow bool, lines int) error {
	// TODO: 실제 로그 조회 구현
	fmt.Printf("세션 %s의 로그 조회 (라인: %d, follow: %t)\n", sessionID, lines, follow)
	fmt.Println("로그 조회 기능은 아직 구현되지 않았습니다.")
	return nil
}

// showClaudeStatus는 Claude 상태를 표시합니다
func showClaudeStatus(ctx context.Context) error {
	fmt.Println("Claude CLI 상태:")
	fmt.Println("  버전: 확인 중...")
	fmt.Println("  활성 세션: 확인 중...")
	fmt.Println("  리소스 사용량: 확인 중...")
	fmt.Println()
	fmt.Println("상태 조회 기능은 아직 구현되지 않았습니다.")
	return nil
}