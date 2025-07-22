# 프로젝트 디렉토리 마운트 시스템

이 패키지는 로컬 프로젝트 디렉토리를 Docker 컨테이너로 안전하게 마운트하는 시스템을 제공합니다.

## 📋 주요 기능

### 1. 안전한 경로 검증
- 프로젝트 디렉토리 존재 및 접근 권한 확인
- 시스템 디렉토리 및 민감한 경로 차단
- 심볼릭 링크 보안 검사
- 플랫폼별 경로 처리 (Windows/Unix)

### 2. 유연한 마운트 구성
- 읽기 전용/읽기-쓰기 모드 설정
- 사용자/그룹 ID 매핑
- 동기화 모드 최적화 (cached, delegated)
- 제외 패턴을 통한 파일 필터링

### 3. 실시간 파일 모니터링
- 파일 변경 사항 실시간 감지
- 설정 가능한 제외 패턴
- 백그라운드 감시 및 이벤트 콜백
- 성능 최적화된 스캐닝

### 4. Docker 통합
- Docker Mount 객체 자동 변환
- 컨테이너 생명주기와 연동
- 마운트 상태 모니터링
- 에러 복구 및 진단

## 🏗️ 아키텍처

```
mount/
├── validator.go    # 경로 검증 및 보안 검사
├── manager.go      # 마운트 설정 관리
├── sync.go         # 파일 동기화 및 모니터링
└── types.go        # 공통 타입 정의
```

### 핵심 컴포넌트

1. **Validator**: 경로 유효성 및 보안 검증
2. **Manager**: 마운트 설정 생성 및 관리
3. **Syncer**: 파일 변경 감시 및 동기화

## 📝 사용 예제

### 기본 워크스페이스 마운트

```go
package main

import (
    "context"
    "github.com/aicli/aicli-web/internal/docker/mount"
    "github.com/aicli/aicli-web/internal/models"
)

func main() {
    manager := mount.NewManager()
    
    workspace := &models.Workspace{
        ID:          "my-workspace",
        ProjectPath: "/home/user/project",
    }
    
    // 마운트 설정 생성
    config, err := manager.CreateWorkspaceMount(workspace)
    if err != nil {
        panic(err)
    }
    
    // Docker Mount 객체로 변환
    dockerMount, err := manager.ToDockerMount(config)
    if err != nil {
        panic(err)
    }
    
    // 컨테이너 생성 시 사용
    // containerConfig.HostConfig.Mounts = []mount.Mount{dockerMount}
}
```

### 사용자 정의 마운트

```go
func customMount() {
    manager := mount.NewManager()
    
    req := &mount.CreateMountRequest{
        WorkspaceID:     "my-workspace",
        SourcePath:      "/home/user/data",
        TargetPath:      "/data",
        ReadOnly:        true,
        SyncMode:        mount.SyncModeCached,
        ExcludePatterns: []string{"*.tmp", "node_modules"},
    }
    
    config, err := manager.CreateCustomMount(req)
    if err != nil {
        panic(err)
    }
    
    dockerMount, _ := manager.ToDockerMount(config)
}
```

### 파일 변경 감시

```go
func watchFiles() {
    manager := mount.NewManager()
    ctx := context.Background()
    
    callback := func(changedFiles []string) {
        for _, file := range changedFiles {
            fmt.Printf("Changed: %s\n", file)
        }
    }
    
    err := manager.StartFileWatcher(
        ctx,
        "/home/user/project",
        []string{"*.tmp", ".git"},
        callback,
    )
    if err != nil {
        panic(err)
    }
    
    // 작업 완료 후 정리
    defer manager.StopFileWatcher("/home/user/project")
}
```

### 마운트 상태 확인

```go
func checkMountStatus() {
    manager := mount.NewManager()
    ctx := context.Background()
    
    status, err := manager.GetMountStatus(ctx, config)
    if err != nil {
        panic(err)
    }
    
    if !status.Available {
        fmt.Printf("Mount not available: %s\n", status.Error)
        return
    }
    
    fmt.Printf("Disk usage: %d/%d bytes\n", 
        status.DiskUsage.Used, status.DiskUsage.Total)
}
```

## ⚙️ 설정 옵션

### MountConfig 필드

| 필드 | 타입 | 설명 |
|------|------|------|
| `SourcePath` | `string` | 로컬 프로젝트 경로 |
| `TargetPath` | `string` | 컨테이너 내부 경로 |
| `ReadOnly` | `bool` | 읽기 전용 모드 |
| `UserID` | `int` | 마운트 소유자 UID |
| `GroupID` | `int` | 마운트 그룹 GID |
| `SyncMode` | `SyncMode` | 동기화 모드 |
| `ExcludePatterns` | `[]string` | 제외 패턴 목록 |
| `NoExec` | `bool` | 실행 권한 제거 |
| `NoSuid` | `bool` | SUID 비트 무시 |
| `NoDev` | `bool` | 디바이스 파일 차단 |

### 동기화 모드

- `SyncModeNative`: 기본 Docker 마운트 (기본값)
- `SyncModeOptimized`: 파일시스템별 성능 최적화
- `SyncModeCached`: 호스트 우선 캐싱
- `SyncModeDelegated`: 컨테이너 우선 캐싱

### 기본 제외 패턴

```go
[]string{
    ".git", ".svn", ".hg",           // VCS 디렉토리
    ".vscode", ".idea",              // IDE 설정
    "node_modules", "dist", "build", // 빌드 결과물
    "*.log", "*.tmp",                // 로그 및 임시 파일
    ".DS_Store", "Thumbs.db",        // OS 메타파일
}
```

## 🔒 보안 고려사항

### 차단되는 경로

**Unix 계열:**
- 시스템 루트 디렉토리 (`/`, `/etc`, `/usr`, `/bin` 등)
- Docker 소켓 (`/var/run/docker.sock`)
- 루트 홈 디렉토리 (`/root`)

**Windows:**
- 시스템 디렉토리 (`C:\Windows`, `C:\Program Files` 등)
- 시스템 볼륨 정보

### 컨테이너 내부 민감한 경로

- `/etc`, `/usr`, `/bin`, `/sbin` 등 시스템 디렉토리
- `/var/run` 런타임 디렉토리

## 📊 성능 특성

### 벤치마크 결과 (참고용)

- **경로 검증**: < 1ms per path
- **마운트 설정 생성**: < 5ms per config  
- **파일 스캔**: ~1000 files/second
- **메모리 사용량**: < 10MB for typical workloads

### 최적화 팁

1. **제외 패턴 사용**: 불필요한 파일 스캔 방지
2. **적절한 동기화 모드**: 워크로드에 맞는 모드 선택
3. **감시 간격 조정**: 실시간성 vs 성능 트레이드오프

## 🧪 테스트

### 단위 테스트 실행

```bash
make test-mount
```

### 통합 테스트 실행

```bash
make test-mount-integration
```

### Docker 환경 테스트

```bash
DOCKER_INTEGRATION_TEST=1 go test -v ./internal/docker/mount_manager_integration_test.go
```

## 🐛 문제 해결

### 일반적인 오류

**"project directory does not exist"**
- 경로가 올바른지 확인
- 디렉토리 접근 권한 확인

**"cannot mount system directory"**
- 시스템 디렉토리 마운트 시도
- 안전한 사용자 디렉토리 사용

**"security check failed"**
- 심볼릭 링크가 안전하지 않은 위치를 가리킴
- 실제 경로 확인 및 수정

### 디버깅 도구

```go
// 마운트 상태 진단
status, _ := manager.GetMountStatus(ctx, config)
fmt.Printf("Available: %v, Error: %s\n", status.Available, status.Error)

// 파일 통계 조회
stats, _ := manager.GetFileStats(ctx, sourcePath, excludePatterns)
fmt.Printf("Files: %d, Size: %d bytes\n", stats.FileCount, stats.TotalSize)

// 활성 watcher 확인
watchers := manager.GetActiveWatchers()
fmt.Printf("Active watchers: %v\n", watchers)
```

## 🤝 기여 방법

1. 새로운 기능이나 버그 수정 전 이슈 생성
2. 테스트 커버리지 유지 (>90%)
3. 보안 검증 강화
4. 문서 업데이트

## 📚 관련 문서

- [Docker Bind Mounts](https://docs.docker.com/storage/bind-mounts/)
- [Container Security](https://docs.docker.com/engine/security/)
- [File System Monitoring](https://pkg.go.dev/path/filepath#Walk)

## 📄 라이센스

프로젝트 루트의 LICENSE 파일 참조