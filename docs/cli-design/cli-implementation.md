# CLI 도구 구현 가이드

## 🛠️ Terry CLI 구조

Go 언어로 구현된 사용자 친화적인 커맨드라인 인터페이스입니다.

## 📁 프로젝트 구조

```
terry/
├── cmd/
│   └── terry/
│       └── main.go          # 진입점
├── internal/
│   ├── cli/
│   │   ├── root.go         # 루트 명령
│   │   ├── workspace.go    # workspace 하위 명령
│   │   ├── task.go         # task 하위 명령
│   │   ├── logs.go         # logs 명령
│   │   └── config.go       # 설정 명령
│   ├── client/
│   │   ├── api.go          # API 클라이언트
│   │   └── websocket.go    # WebSocket 클라이언트
│   ├── config/
│   │   └── config.go       # 설정 관리
│   └── output/
│       ├── formatter.go    # 출력 포맷터
│       └── printer.go      # 출력 헬퍼
├── pkg/
│   └── version/
│       └── version.go      # 버전 정보
└── go.mod
```

## 🔧 핵심 구현

### 1. 메인 엔트리포인트

```go
// cmd/terry/main.go
package main

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/yourusername/terry/internal/cli"
    "github.com/yourusername/terry/pkg/version"
)

func main() {
    if err := cli.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

### 2. 루트 명령 구성

```go
// internal/cli/root.go
package cli

import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
    verbose bool
    output  string
)

var rootCmd = &cobra.Command{
    Use:   "terry",
    Short: "AI-powered code management CLI",
    Long: `Terry is a command-line interface for managing AI-powered coding tasks.
It allows you to create workspaces, run Claude AI tasks, and monitor progress.`,
    PersistentPreRun: func(cmd *cobra.Command, args []string) {
        initConfig()
    },
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    // 전역 플래그
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.terry.yaml)")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
    rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "output format (table|json|yaml)")
    
    // 하위 명령 추가
    rootCmd.AddCommand(
        NewWorkspaceCmd(),
        NewTaskCmd(),
        NewLogsCmd(),
        NewConfigCmd(),
        NewVersionCmd(),
    )
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        home, _ := os.UserHomeDir()
        viper.AddConfigPath(home)
        viper.SetConfigName(".terry")
    }
    
    viper.AutomaticEnv()
    viper.ReadInConfig()
}
```

### 3. Workspace 명령 구현

```go
// internal/cli/workspace.go
package cli

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/yourusername/terry/internal/client"
    "github.com/yourusername/terry/internal/output"
)

func NewWorkspaceCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "workspace",
        Short: "Manage workspaces",
        Aliases: []string{"ws"},
    }
    
    cmd.AddCommand(
        newWorkspaceListCmd(),
        newWorkspaceCreateCmd(),
        newWorkspaceDeleteCmd(),
        newWorkspaceInfoCmd(),
    )
    
    return cmd
}

func newWorkspaceListCmd() *cobra.Command {
    var (
        status string
        limit  int
    )
    
    cmd := &cobra.Command{
        Use:   "list",
        Short: "List all workspaces",
        RunE: func(cmd *cobra.Command, args []string) error {
            client := client.NewAPIClient()
            
            workspaces, err := client.ListWorkspaces(client.ListOptions{
                Status: status,
                Limit:  limit,
            })
            if err != nil {
                return fmt.Errorf("failed to list workspaces: %w", err)
            }
            
            // 출력 포맷팅
            formatter := output.NewFormatter(output)
            return formatter.PrintWorkspaces(workspaces)
        },
    }
    
    cmd.Flags().StringVarP(&status, "status", "s", "", "filter by status (active|archived)")
    cmd.Flags().IntVarP(&limit, "limit", "l", 10, "limit number of results")
    
    return cmd
}

func newWorkspaceCreateCmd() *cobra.Command {
    var (
        name        string
        description string
        path        string
        gitURL      string
        branch      string
    )
    
    cmd := &cobra.Command{
        Use:   "create",
        Short: "Create a new workspace",
        RunE: func(cmd *cobra.Command, args []string) error {
            if name == "" {
                return fmt.Errorf("workspace name is required")
            }
            
            // 경로가 지정되지 않았으면 현재 디렉토리 사용
            if path == "" {
                path, _ = os.Getwd()
            }
            
            client := client.NewAPIClient()
            workspace, err := client.CreateWorkspace(client.WorkspaceCreateRequest{
                Name:        name,
                Description: description,
                Path:        path,
                GitURL:      gitURL,
                Branch:      branch,
            })
            
            if err != nil {
                return fmt.Errorf("failed to create workspace: %w", err)
            }
            
            fmt.Printf("✅ Workspace '%s' created successfully!\n", workspace.Name)
            fmt.Printf("ID: %s\n", workspace.ID)
            return nil
        },
    }
    
    cmd.Flags().StringVarP(&name, "name", "n", "", "workspace name (required)")
    cmd.Flags().StringVarP(&description, "desc", "d", "", "workspace description")
    cmd.Flags().StringVarP(&path, "path", "p", "", "local path (default: current directory)")
    cmd.Flags().StringVar(&gitURL, "git-url", "", "git repository URL")
    cmd.Flags().StringVar(&branch, "branch", "main", "git branch")
    
    cmd.MarkFlagRequired("name")
    
    return cmd
}
```

### 4. Task 명령 구현

```go
// internal/cli/task.go
package cli

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    
    "github.com/spf13/cobra"
    "github.com/yourusername/terry/internal/client"
)

func NewTaskCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "task",
        Short: "Manage tasks",
        Aliases: []string{"t"},
    }
    
    cmd.AddCommand(
        newTaskCreateCmd(),
        newTaskListCmd(),
        newTaskStatusCmd(),
        newTaskCancelCmd(),
    )
    
    return cmd
}

func newTaskCreateCmd() *cobra.Command {
    var (
        workspace    string
        systemPrompt string
        maxTurns     int
        interactive  bool
        watch        bool
    )
    
    cmd := &cobra.Command{
        Use:   "create [prompt]",
        Short: "Create a new task",
        Args:  cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            var prompt string
            
            // 프롬프트 가져오기
            if len(args) > 0 {
                prompt = args[0]
            } else if interactive {
                prompt = getInteractivePrompt()
            } else {
                // 파이프에서 읽기
                scanner := bufio.NewScanner(os.Stdin)
                var lines []string
                for scanner.Scan() {
                    lines = append(lines, scanner.Text())
                }
                prompt = strings.Join(lines, "\n")
            }
            
            if prompt == "" {
                return fmt.Errorf("prompt is required")
            }
            
            // 워크스페이스 자동 감지
            if workspace == "" {
                workspace = detectWorkspace()
            }
            
            client := client.NewAPIClient()
            task, err := client.CreateTask(client.TaskCreateRequest{
                WorkspaceID:  workspace,
                Prompt:       prompt,
                SystemPrompt: systemPrompt,
                MaxTurns:     maxTurns,
            })
            
            if err != nil {
                return fmt.Errorf("failed to create task: %w", err)
            }
            
            fmt.Printf("✅ Task created: %s\n", task.ID)
            
            // 실시간 로그 보기
            if watch {
                return watchTaskLogs(task.ID)
            }
            
            return nil
        },
    }
    
    cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "workspace ID or name")
    cmd.Flags().StringVarP(&systemPrompt, "system", "s", "", "system prompt")
    cmd.Flags().IntVarP(&maxTurns, "max-turns", "m", 10, "maximum number of turns")
    cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "interactive prompt mode")
    cmd.Flags().BoolVarP(&watch, "watch", "f", false, "watch task logs")
    
    return cmd
}

func getInteractivePrompt() string {
    fmt.Println("Enter your prompt (Ctrl+D to finish):")
    scanner := bufio.NewScanner(os.Stdin)
    var lines []string
    
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    
    return strings.Join(lines, "\n")
}

func detectWorkspace() string {
    // .terry 파일에서 워크스페이스 ID 읽기
    if data, err := os.ReadFile(".terry"); err == nil {
        return strings.TrimSpace(string(data))
    }
    
    // 현재 디렉토리 이름 사용
    cwd, _ := os.Getwd()
    return filepath.Base(cwd)
}
```

### 5. 실시간 로그 스트리밍

```go
// internal/cli/logs.go
package cli

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    
    "github.com/fatih/color"
    "github.com/spf13/cobra"
    "github.com/yourusername/terry/internal/client"
)

func NewLogsCmd() *cobra.Command {
    var (
        follow bool
        tail   int
        format string
    )
    
    cmd := &cobra.Command{
        Use:   "logs [task-id]",
        Short: "View task logs",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            taskID := args[0]
            
            if follow {
                return streamLogs(taskID)
            }
            
            // 정적 로그 가져오기
            client := client.NewAPIClient()
            logs, err := client.GetTaskLogs(taskID, tail)
            if err != nil {
                return err
            }
            
            printLogs(logs, format)
            return nil
        },
    }
    
    cmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
    cmd.Flags().IntVarP(&tail, "tail", "n", 100, "number of lines to show")
    cmd.Flags().StringVar(&format, "format", "pretty", "log format (pretty|json|raw)")
    
    return cmd
}

func streamLogs(taskID string) error {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Ctrl+C 처리
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    go func() {
        <-sigCh
        cancel()
    }()
    
    client := client.NewAPIClient()
    logStream, err := client.StreamTaskLogs(ctx, taskID)
    if err != nil {
        return err
    }
    
    // 컬러 출력
    yellow := color.New(color.FgYellow).SprintFunc()
    red := color.New(color.FgRed).SprintFunc()
    green := color.New(color.FgGreen).SprintFunc()
    
    for log := range logStream {
        switch log.Level {
        case "error":
            fmt.Printf("[%s] %s\n", red("ERROR"), log.Message)
        case "warning":
            fmt.Printf("[%s] %s\n", yellow("WARN"), log.Message)
        case "info":
            fmt.Printf("[%s] %s\n", green("INFO"), log.Message)
        default:
            fmt.Println(log.Message)
        }
    }
    
    return nil
}
```

### 6. 설정 관리

```go
// internal/cli/config.go
package cli

import (
    "fmt"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

func NewConfigCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "config",
        Short: "Manage configuration",
    }
    
    cmd.AddCommand(
        newConfigGetCmd(),
        newConfigSetCmd(),
        newConfigListCmd(),
    )
    
    return cmd
}

func newConfigSetCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "set [key] [value]",
        Short: "Set configuration value",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            key, value := args[0], args[1]
            
            viper.Set(key, value)
            if err := viper.WriteConfig(); err != nil {
                return fmt.Errorf("failed to save config: %w", err)
            }
            
            fmt.Printf("✅ Set %s = %s\n", key, value)
            return nil
        },
    }
}

// 설정 키
const (
    ConfigAPIURL       = "api.url"
    ConfigAPIKey       = "api.key"
    ConfigDefaultWS    = "default.workspace"
    ConfigOutputFormat = "output.format"
    ConfigColorOutput  = "output.color"
)
```

### 7. 인터랙티브 기능

```go
// internal/cli/interactive.go
package cli

import (
    "github.com/AlecAivazis/survey/v2"
)

func selectWorkspace() (string, error) {
    client := client.NewAPIClient()
    workspaces, err := client.ListWorkspaces(client.ListOptions{})
    if err != nil {
        return "", err
    }
    
    options := make([]string, len(workspaces))
    for i, ws := range workspaces {
        options[i] = fmt.Sprintf("%s (%s)", ws.Name, ws.ID)
    }
    
    var selected string
    prompt := &survey.Select{
        Message: "Choose a workspace:",
        Options: options,
    }
    
    if err := survey.AskOne(prompt, &selected); err != nil {
        return "", err
    }
    
    // ID 추출
    for _, ws := range workspaces {
        if strings.Contains(selected, ws.ID) {
            return ws.ID, nil
        }
    }
    
    return "", fmt.Errorf("workspace not found")
}
```

### 8. 출력 포맷터

```go
// internal/output/formatter.go
package output

import (
    "encoding/json"
    "fmt"
    "os"
    
    "github.com/olekukonko/tablewriter"
    "gopkg.in/yaml.v2"
)

type Formatter struct {
    format string
}

func NewFormatter(format string) *Formatter {
    return &Formatter{format: format}
}

func (f *Formatter) PrintWorkspaces(workspaces []Workspace) error {
    switch f.format {
    case "json":
        return f.printJSON(workspaces)
    case "yaml":
        return f.printYAML(workspaces)
    case "table":
        return f.printTable(workspaces)
    default:
        return fmt.Errorf("unknown format: %s", f.format)
    }
}

func (f *Formatter) printTable(workspaces []Workspace) error {
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"ID", "Name", "Status", "Tasks", "Created"})
    table.SetAutoWrapText(false)
    table.SetAutoFormatHeaders(true)
    table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
    table.SetAlignment(tablewriter.ALIGN_LEFT)
    table.SetCenterSeparator("")
    table.SetColumnSeparator("")
    table.SetRowSeparator("")
    table.SetHeaderLine(false)
    table.SetBorder(false)
    table.SetTablePadding("\t")
    
    for _, ws := range workspaces {
        table.Append([]string{
            ws.ID[:8],
            ws.Name,
            ws.Status,
            fmt.Sprintf("%d", ws.TaskCount),
            ws.CreatedAt.Format("2006-01-02"),
        })
    }
    
    table.Render()
    return nil
}
```

## 🎨 사용자 경험 개선

### 1. 진행 표시기

```go
import "github.com/schollz/progressbar/v3"

func showProgress() {
    bar := progressbar.NewOptions(100,
        progressbar.OptionSetDescription("Processing..."),
        progressbar.OptionSetTheme(progressbar.Theme{
            Saucer:        "[green]=[reset]",
            SaucerHead:    "[green]>[reset]",
            SaucerPadding: " ",
            BarStart:      "[",
            BarEnd:        "]",
        }),
    )
    
    for i := 0; i < 100; i++ {
        bar.Add(1)
        time.Sleep(10 * time.Millisecond)
    }
}
```

### 2. 자동 완성

```bash
# Bash completion
terry completion bash > /etc/bash_completion.d/terry

# Zsh completion
terry completion zsh > "${fpath[1]}/_terry"

# Fish completion
terry completion fish > ~/.config/fish/completions/terry.fish
```

### 3. 별칭 및 단축키

```yaml
# ~/.terry.yaml
aliases:
  ws: workspace
  t: task
  l: logs

shortcuts:
  create-task: "task create -w default"
  watch-logs: "logs -f"
```

## 📦 배포 및 설치

### 1. 설치 스크립트

```bash
#!/bin/bash
# install.sh

VERSION=${VERSION:-latest}
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
fi

URL="https://github.com/yourusername/terry/releases/download/${VERSION}/terry-${OS}-${ARCH}"

echo "Downloading Terry CLI..."
curl -L "$URL" -o /usr/local/bin/terry
chmod +x /usr/local/bin/terry

echo "Terry CLI installed successfully!"
terry version
```

### 2. Homebrew Formula

```ruby
class Terry < Formula
  desc "AI-powered code management CLI"
  homepage "https://github.com/yourusername/terry"
  version "1.0.0"
  
  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/yourusername/terry/releases/download/v1.0.0/terry-darwin-arm64"
    sha256 "..."
  elsif OS.mac?
    url "https://github.com/yourusername/terry/releases/download/v1.0.0/terry-darwin-amd64"
    sha256 "..."
  elsif OS.linux?
    url "https://github.com/yourusername/terry/releases/download/v1.0.0/terry-linux-amd64"
    sha256 "..."
  end
  
  def install
    bin.install "terry"
  end
  
  test do
    system "#{bin}/terry", "version"
  end
end
```