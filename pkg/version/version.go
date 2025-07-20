package version

import (
	"fmt"
	"runtime"
)

var (
	// Version은 현재 릴리스의 버전입니다.
	// 빌드 시 -ldflags로 설정됩니다.
	Version = "dev"

	// BuildTime은 빌드된 시간입니다.
	// 빌드 시 -ldflags로 설정됩니다.
	BuildTime = "unknown"

	// GitCommit은 빌드 시점의 git commit hash입니다.
	// 빌드 시 -ldflags로 설정됩니다.
	GitCommit = "unknown"

	// GitBranch는 빌드 시점의 git branch입니다.
	// 빌드 시 -ldflags로 설정됩니다.
	GitBranch = "unknown"
)

// Info는 버전 정보를 담는 구조체입니다.
type Info struct {
	Version   string `json:"version"`
	BuildTime string `json:"buildTime"`
	GitCommit string `json:"gitCommit"`
	GitBranch string `json:"gitBranch"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// Get은 현재 버전 정보를 반환합니다.
func Get() Info {
	return Info{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String은 버전 정보를 문자열로 반환합니다.
func (i Info) String() string {
	return fmt.Sprintf(`AICode Manager CLI
Version:     %s
Build Time:  %s
Git Commit:  %s
Git Branch:  %s
Go Version:  %s
Platform:    %s`,
		i.Version,
		i.BuildTime,
		i.GitCommit,
		i.GitBranch,
		i.GoVersion,
		i.Platform,
	)
}