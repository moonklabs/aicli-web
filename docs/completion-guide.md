# AICode Manager CLI 자동완성 설치 가이드

이 문서는 `aicli` 명령어의 쉘 자동완성 기능을 설치하고 사용하는 방법을 안내합니다.

## 지원하는 쉘

- Bash
- Zsh
- Fish
- PowerShell

## 자동완성 스크립트 생성

모든 쉘에서 자동완성 스크립트를 생성하는 기본 명령어는 다음과 같습니다:

```bash
aicli completion [shell]
```

## Bash 설치 가이드

### 시스템 전역 설치 (권장)

```bash
# 자동완성 스크립트 생성 및 저장
sudo aicli completion bash > /etc/bash_completion.d/aicli
```

### 사용자별 설치

```bash
# 1. 자동완성 스크립트 생성
aicli completion bash > ~/.aicli-completion.bash

# 2. ~/.bashrc에 추가
echo "source ~/.aicli-completion.bash" >> ~/.bashrc

# 3. 현재 세션에 적용
source ~/.bashrc
```

## Zsh 설치 가이드

### Oh My Zsh 사용자

```bash
# 1. 자동완성 스크립트 생성
aicli completion zsh > ~/.oh-my-zsh/completions/_aicli

# 2. 새 쉘 세션 시작 또는 재로드
source ~/.zshrc
```

### 일반 Zsh 사용자

```bash
# 1. fpath 확인
echo $fpath

# 2. 자동완성 스크립트를 fpath 디렉토리에 저장
aicli completion zsh > "${fpath[1]}/_aicli"

# 또는 사용자별 설치
aicli completion zsh > ~/.aicli-completion.zsh
echo "source ~/.aicli-completion.zsh" >> ~/.zshrc

# 3. 재로드
source ~/.zshrc
```

## Fish 설치 가이드

Fish는 가장 간단한 설치 과정을 제공합니다:

```bash
# 자동완성 스크립트 생성 및 저장
aicli completion fish > ~/.config/fish/completions/aicli.fish
```

새 쉘 세션을 시작하면 자동으로 적용됩니다.

## PowerShell 설치 가이드

```powershell
# 1. 자동완성 스크립트 생성
aicli completion powershell > aicli.ps1

# 2. PowerShell 프로필에 추가
Add-Content $PROFILE ". ./aicli.ps1"

# 3. 새 PowerShell 세션 시작
```

## 자동완성 기능 테스트

설치가 완료되면 다음과 같이 테스트할 수 있습니다:

```bash
# 명령어 자동완성
aicli [TAB]

# 서브커맨드 자동완성
aicli workspace [TAB]

# 플래그 자동완성
aicli --[TAB]

# 동적 자동완성 (워크스페이스 이름)
aicli workspace delete [TAB]
```

## 동적 자동완성 기능

`aicli`는 다음과 같은 동적 자동완성을 지원합니다:

- **워크스페이스 이름**: `workspace delete`, `workspace info` 명령어에서 사용
- **태스크 ID**: `task` 관련 명령어에서 사용
- **출력 형식**: `--output` 플래그에서 `table`, `json`, `yaml`, `csv` 중 선택
- **설정 키**: `config` 명령어에서 사용 가능한 설정 키 목록

## 문제 해결

### Bash에서 자동완성이 작동하지 않는 경우

1. bash-completion 패키지가 설치되어 있는지 확인:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install bash-completion
   
   # CentOS/RHEL
   sudo yum install bash-completion
   
   # macOS
   brew install bash-completion
   ```

2. 쉘을 재시작하거나 다음 명령어 실행:
   ```bash
   source /etc/bash_completion
   ```

### Zsh에서 자동완성이 작동하지 않는 경우

1. compinit이 활성화되어 있는지 확인:
   ```bash
   # ~/.zshrc에 추가
   autoload -U compinit && compinit
   ```

2. 캐시 재생성:
   ```bash
   rm -f ~/.zcompdump
   compinit
   ```

### 권한 문제

시스템 전역 설치 시 권한 문제가 발생하면 사용자별 설치를 사용하세요.

## 자동완성 제거

### Bash
```bash
# 시스템 전역
sudo rm /etc/bash_completion.d/aicli

# 사용자별
rm ~/.aicli-completion.bash
# ~/.bashrc에서 source 라인 제거
```

### Zsh
```bash
rm "${fpath[1]}/_aicli"
# 또는
rm ~/.aicli-completion.zsh
# ~/.zshrc에서 source 라인 제거
```

### Fish
```bash
rm ~/.config/fish/completions/aicli.fish
```

### PowerShell
```powershell
# $PROFILE에서 관련 라인 제거
```

## 추가 도움말

더 자세한 정보는 다음 명령어를 참조하세요:

```bash
aicli completion --help
aicli completion bash --help
aicli completion zsh --help
aicli completion fish --help
aicli completion powershell --help
```