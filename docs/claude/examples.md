# Claude CLI 통합 예제 및 레시피

## 개요

이 문서는 AICode Manager의 Claude CLI 통합을 활용한 실용적인 예제와 레시피를 제공합니다. 다양한 사용 시나리오별로 구체적인 구현 방법을 안내합니다.

## 기본 사용 예제

### 1. 단일 프롬프트 실행

```bash
# 간단한 질문
aicli claude run "What is the capital of France?"

# 시스템 프롬프트와 함께
aicli claude run "Write a hello world function in Go" \
  --system "You are an expert Go developer"

# 특정 워크스페이스에서 실행
aicli claude run "Analyze this codebase structure" \
  --workspace ./my-project \
  --tools Read
```

### 2. 파일 기반 프롬프트

```bash
# 프롬프트를 파일에서 읽기
aicli claude run --file prompt.txt \
  --tools Read,Write \
  --workspace ./project

# 컨텍스트 파일과 함께
aicli claude run --file review-prompt.txt \
  --context main.go,utils.go \
  --system "You are a senior code reviewer"
```

### 3. 인터랙티브 세션

```bash
# 인터랙티브 채팅 시작
aicli claude chat

# 특정 설정으로 세션 시작
aicli claude chat \
  --session development \
  --system "You are my coding pair partner" \
  --tools Read,Write,Bash \
  --workspace ./project
```

## 코드 개발 레시피

### 1. Go 웹 서버 생성

```bash
#!/bin/bash
# create-go-webserver.sh

PROMPT="Create a complete Go web server with the following features:
1. REST API with CRUD operations for users
2. JWT authentication middleware  
3. Database integration with SQLite
4. Proper error handling and logging
5. Docker configuration
6. Comprehensive tests

Project structure should follow Go best practices."

aicli claude run "$PROMPT" \
  --tools Write,Read,Bash \
  --system "You are an expert Go backend developer" \
  --workspace ./new-webserver \
  --timeout 10m
```

### 2. 코드 리뷰 자동화

```bash
#!/bin/bash
# code-review.sh

# Git에서 변경된 파일 가져오기
CHANGED_FILES=$(git diff --name-only HEAD~1)

for file in $CHANGED_FILES; do
  if [[ $file == *.go ]]; then
    echo "Reviewing $file..."
    
    aicli claude run "
    Review this Go code for:
    1. Code quality and best practices
    2. Security vulnerabilities
    3. Performance issues
    4. Test coverage suggestions
    5. Documentation improvements
    
    File content:
    \$(cat $file)
    " \
    --system "You are a senior Go code reviewer with 10+ years experience" \
    --output "review-$file.md" \
    --format markdown
  fi
done
```

### 3. 테스트 코드 생성

```bash
#!/bin/bash
# generate-tests.sh

find . -name "*.go" -not -name "*_test.go" | while read -r file; do
  echo "Generating tests for $file..."
  
  aicli claude run "
  Generate comprehensive unit tests for this Go code:
  
  Requirements:
  1. Test all public functions and methods
  2. Include edge cases and error conditions
  3. Use table-driven tests where appropriate
  4. Mock external dependencies
  5. Achieve >90% code coverage
  
  Source code:
  \$(cat $file)
  " \
  --tools Write,Read \
  --system "You are an expert in Go testing and TDD" \
  --output "${file%.*}_test.go"
done
```

## 문서화 레시피

### 1. API 문서 자동 생성

```bash
#!/bin/bash
# generate-api-docs.sh

aicli claude run "
Analyze this Go web server code and generate comprehensive API documentation:

1. OpenAPI 3.0 specification
2. Endpoint descriptions with examples
3. Request/response schemas
4. Authentication requirements
5. Error codes and responses

Include interactive examples and curl commands.
" \
--tools Read,Write \
--workspace ./api-server \
--system "You are a technical writer specialized in API documentation" \
--output docs/api-reference.md
```

### 2. README 생성

```bash
#!/bin/bash
# generate-readme.sh

PROJECT_NAME=$(basename $PWD)

aicli claude run "
Create a comprehensive README.md for this $PROJECT_NAME project:

1. Project overview and purpose
2. Features and capabilities  
3. Installation instructions
4. Usage examples
5. API documentation links
6. Contributing guidelines
7. License information

Make it engaging and professional.
" \
--tools Read \
--workspace . \
--system "You are a technical writer creating developer documentation" \
--output README.md
```

### 3. 코드 주석 추가

```bash
#!/bin/bash
# add-comments.sh

find . -name "*.go" | while read -r file; do
  echo "Adding comments to $file..."
  
  aicli claude run "
  Add comprehensive Go documentation comments to this code:
  
  1. Package-level documentation
  2. Function and method comments
  3. Type definitions
  4. Exported constants and variables
  5. Follow Go documentation conventions
  
  Original code:
  \$(cat $file)
  " \
  --tools Write \
  --system "You are a Go documentation expert" \
  --output "$file.commented" \
  --workspace .
  
  # 원본 파일 백업 후 교체
  mv "$file" "$file.backup"
  mv "$file.commented" "$file"
done
```

## 데이터 분석 레시피

### 1. 로그 분석

```bash
#!/bin/bash
# analyze-logs.sh

aicli claude run "
Analyze this application log file and provide:

1. Error pattern analysis
2. Performance bottlenecks identification
3. Usage pattern insights
4. Anomaly detection
5. Recommendations for optimization

Log data:
\$(tail -n 1000 /var/log/app.log)
" \
--system "You are a DevOps engineer specialized in log analysis" \
--output log-analysis-report.md \
--format markdown
```

### 2. 성능 분석

```bash
#!/bin/bash
# performance-analysis.sh

aicli claude run "
Analyze this Go application's performance characteristics:

1. CPU profiling data analysis
2. Memory usage patterns
3. Goroutine behavior
4. Database query performance
5. Bottleneck identification
6. Optimization recommendations

Include:
- Profiling data: \$(cat cpu.prof)
- Memory profile: \$(cat mem.prof)  
- Trace data: \$(cat trace.out)
" \
--tools Read \
--system "You are a Go performance expert" \
--output performance-report.md
```

## 배치 처리 레시피

### 1. 다중 프로젝트 처리

```bash
#!/bin/bash
# batch-process-projects.sh

PROJECTS_DIR="/workspace/projects"

find $PROJECTS_DIR -name "*.go" -path "*/cmd/*/main.go" | \
  xargs dirname | \
  xargs dirname | \
  sort -u | \
while read -r project; do
  echo "Processing project: $project"
  
  aicli claude run "
  Analyze this Go project and provide:
  
  1. Architecture overview
  2. Code quality assessment
  3. Security vulnerability scan
  4. Performance optimization suggestions
  5. Testing strategy recommendations
  
  Focus on actionable insights.
  " \
  --workspace "$project" \
  --tools Read \
  --system "You are a senior software architect" \
  --output "$project/analysis-report.md" \
  --session "batch-analysis" &
  
  # 동시 실행 수 제한
  (($(jobs -r | wc -l) >= 5)) && wait
done

wait  # 모든 작업 완료 대기
```

### 2. 코드 마이그레이션

```bash
#!/bin/bash
# migrate-to-new-framework.sh

aicli claude run "
Migrate this Go HTTP server from net/http to Gin framework:

1. Convert route handlers to Gin handlers
2. Update middleware implementation
3. Migrate request/response handling
4. Update error handling patterns
5. Maintain existing API compatibility
6. Add Gin-specific optimizations

Original code structure:
\$(find . -name "*.go" -exec echo "=== {} ===" \; -exec cat {} \;)
" \
--tools Write,Read \
--system "You are an expert in Go web frameworks and migration" \
--workspace ./http-server \
--session migration \
--timeout 15m
```

## WebSocket & 실시간 처리

### 1. 실시간 로그 스트리밍

```javascript
// real-time-logs.js
const WebSocket = require('ws');

async function streamClaudeLogs() {
  // 세션 생성
  const session = await fetch('http://localhost:8080/api/v1/claude/sessions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${process.env.JWT_TOKEN}`
    },
    body: JSON.stringify({
      workspace_id: 'log-analysis',
      system_prompt: 'You are a log analysis expert',
      stream: true
    })
  }).then(r => r.json());

  // WebSocket 연결
  const ws = new WebSocket(`ws://localhost:8080/ws/executions/${session.id}`);
  
  ws.on('message', (data) => {
    const message = JSON.parse(data);
    
    switch (message.type) {
      case 'text':
        process.stdout.write(message.data.content);
        break;
      case 'tool_use':
        console.log(`\n[TOOL] ${message.data.tool}: ${message.data.parameters}`);
        break;
      case 'error':
        console.error(`\n[ERROR] ${message.data.message}`);
        break;
      case 'complete':
        console.log('\n[COMPLETE] Analysis finished');
        ws.close();
        break;
    }
  });
  
  // 프롬프트 전송
  await fetch('http://localhost:8080/api/v1/claude/execute', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${process.env.JWT_TOKEN}`
    },
    body: JSON.stringify({
      session_id: session.id,
      prompt: `Analyze these application logs in real-time:
        
        ${process.argv[2] || '/var/log/app.log'}
        
        Provide insights on:
        1. Error patterns
        2. Performance issues  
        3. Security events
        4. Usage trends`,
      stream: true
    })
  });
}

streamClaudeLogs().catch(console.error);
```

### 2. 대화형 개발 환경

```python
# interactive-dev.py
import asyncio
import websockets
import json
import requests

class ClaudeInteractiveDev:
    def __init__(self, base_url, token):
        self.base_url = base_url
        self.token = token
        self.session_id = None
        
    async def start_session(self):
        """개발 세션 시작"""
        response = requests.post(
            f"{self.base_url}/api/v1/claude/sessions",
            headers={
                "Authorization": f"Bearer {self.token}",
                "Content-Type": "application/json"
            },
            json={
                "workspace_id": "interactive-dev",
                "system_prompt": """You are my coding pair partner. 
                Help me develop software interactively:
                1. Suggest improvements
                2. Write code snippets
                3. Debug issues
                4. Review changes
                5. Explain concepts
                
                Be concise but thorough.""",
                "allowed_tools": ["Read", "Write", "Bash"],
                "max_turns": 100
            }
        )
        
        session = response.json()
        self.session_id = session['id']
        print(f"Session started: {self.session_id}")
        
    async def chat_loop(self):
        """대화형 루프"""
        ws_url = f"ws://localhost:8080/ws/sessions/{self.session_id}"
        
        async with websockets.connect(ws_url) as websocket:
            print("Connected to Claude. Type 'exit' to quit.\n")
            
            while True:
                user_input = input("You: ").strip()
                
                if user_input.lower() == 'exit':
                    break
                    
                # 프롬프트 전송
                await websocket.send(json.dumps({
                    "type": "prompt",
                    "content": user_input
                }))
                
                # 응답 수신
                print("Claude: ", end="", flush=True)
                async for message in websocket:
                    data = json.loads(message)
                    
                    if data['type'] == 'text':
                        print(data['content'], end="", flush=True)
                    elif data['type'] == 'complete':
                        print("\n")
                        break

# 사용 예제
async def main():
    dev = ClaudeInteractiveDev(
        base_url="http://localhost:8080",
        token=os.getenv("JWT_TOKEN")
    )
    
    await dev.start_session()
    await dev.chat_loop()

if __name__ == "__main__":
    asyncio.run(main())
```

## 통합 워크플로우

### 1. CI/CD 파이프라인 통합

```yaml
# .github/workflows/claude-integration.yml
name: Claude Code Review

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  claude-review:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
      with:
        fetch-depth: 0
        
    - name: Setup Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'
        
    - name: Setup AICLI
      run: |
        wget https://github.com/your-org/aicli-web/releases/latest/download/aicli-linux
        chmod +x aicli-linux
        sudo mv aicli-linux /usr/local/bin/aicli
        
    - name: Claude Code Review
      env:
        CLAUDE_CODE_OAUTH_TOKEN: ${{ secrets.CLAUDE_TOKEN }}
      run: |
        # 변경된 파일 가져오기
        CHANGED_FILES=$(git diff --name-only HEAD~1 | grep '\.go$' || true)
        
        if [ ! -z "$CHANGED_FILES" ]; then
          echo "$CHANGED_FILES" | while read file; do
            aicli claude run "
            Review this Go code changes for:
            1. Code quality and best practices
            2. Security vulnerabilities
            3. Performance issues
            4. Breaking changes
            5. Test coverage
            
            File: $file
            Changes:
            $(git diff HEAD~1 -- $file)
            " \
            --system "You are a senior Go code reviewer" \
            --output "review-$file.md"
          done
        fi
        
    - name: Post Review Comments
      if: always()
      uses: actions/github-script@v6
      with:
        script: |
          const fs = require('fs');
          const path = require('path');
          
          // 리뷰 파일들 찾기
          const reviewFiles = fs.readdirSync('.')
            .filter(file => file.startsWith('review-') && file.endsWith('.md'));
            
          for (const reviewFile of reviewFiles) {
            const content = fs.readFileSync(reviewFile, 'utf8');
            const filename = reviewFile.replace('review-', '').replace('.md', '');
            
            await github.rest.pulls.createReviewComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              pull_number: context.issue.number,
              body: `## Claude Code Review\n\n${content}`,
              path: filename,
              line: 1
            });
          }
```

### 2. 개발 환경 자동 설정

```bash
#!/bin/bash
# setup-dev-environment.sh

PROJECT_NAME=$1
if [ -z "$PROJECT_NAME" ]; then
  echo "Usage: $0 <project-name>"
  exit 1
fi

echo "Setting up development environment for $PROJECT_NAME..."

# 프로젝트 디렉토리 생성
mkdir -p "$PROJECT_NAME"
cd "$PROJECT_NAME"

# Claude를 통한 프로젝트 구조 생성
aicli claude run "
Create a complete Go project structure for a '$PROJECT_NAME' application:

1. Standard Go project layout
2. Go modules initialization  
3. Basic main.go with CLI structure
4. Makefile with common targets
5. Docker configuration
6. GitHub Actions CI/CD
7. README with setup instructions
8. Go dependencies for web development

Include best practices and modern Go patterns.
" \
--tools Write,Bash \
--system "You are a Go project architect" \
--workspace . \
--timeout 5m

echo "Project structure created!"

# Git 저장소 초기화
git init
git add .
git commit -m "Initial project setup by Claude"

echo "Development environment ready for $PROJECT_NAME"
echo "Next steps:"
echo "1. cd $PROJECT_NAME"
echo "2. make setup"
echo "3. make dev"
```

### 3. 멀티모달 문서화

```bash
#!/bin/bash
# generate-multimodal-docs.sh

aicli claude run "
Create comprehensive project documentation with multiple formats:

1. README.md - Project overview
2. ARCHITECTURE.md - System design with mermaid diagrams  
3. API.md - API documentation with OpenAPI spec
4. DEPLOYMENT.md - Deployment instructions
5. CONTRIBUTING.md - Contribution guidelines
6. docs/tutorials/ - Step-by-step tutorials
7. docs/examples/ - Code examples
8. docs/troubleshooting.md - Common issues

Include:
- Mermaid diagrams for architecture
- Code examples in multiple languages
- Interactive API documentation
- Docker compose examples
- Kubernetes manifests

Make it production-ready documentation.
" \
--tools Write,Read,Bash \
--system "You are a technical documentation expert" \
--workspace . \
--session documentation \
--timeout 10m

# 문서 검증 및 링크 확인
echo "Validating documentation..."
find docs -name "*.md" -exec markdown-link-check {} \;

# PDF 버전 생성 (optional)
if command -v pandoc &> /dev/null; then
  echo "Generating PDF documentation..."
  pandoc README.md ARCHITECTURE.md -o project-docs.pdf
fi

echo "Documentation generation complete!"
```

## 성능 및 모니터링

### 1. 성능 벤치마킹

```bash
#!/bin/bash
# claude-performance-benchmark.sh

echo "Claude CLI Integration Performance Benchmark"
echo "==========================================="

# 다양한 프롬프트 크기로 테스트
PROMPTS=(
  "Write a simple hello world function"
  "$(cat large-prompt.txt)"  # 큰 프롬프트
  "Analyze this codebase: $(find . -name '*.go' | head -10 | xargs cat)"
)

for i in "${!PROMPTS[@]}"; do
  echo "Test $((i+1)): Prompt size $(echo "${PROMPTS[$i]}" | wc -c) characters"
  
  time aicli claude run "${PROMPTS[$i]}" \
    --tools Read,Write \
    --timeout 2m \
    --format json > "benchmark-result-$((i+1)).json"
    
  echo "---"
done

# 결과 분석
aicli claude run "
Analyze these performance benchmark results and provide:

1. Response time analysis
2. Throughput metrics
3. Resource usage patterns
4. Performance recommendations
5. Bottleneck identification

Benchmark data:
$(for f in benchmark-result-*.json; do echo "=== $f ==="; cat "$f"; done)
" \
--system "You are a performance analysis expert" \
--output performance-analysis.md
```

### 2. 리소스 모니터링

```bash
#!/bin/bash
# monitor-claude-resources.sh

# 모니터링 시작
echo "Starting Claude resource monitoring..."

while true; do
  timestamp=$(date '+%Y-%m-%d %H:%M:%S')
  
  # CPU 및 메모리 사용량
  ps_output=$(ps aux | grep -E "(aicli|claude)" | grep -v grep)
  
  # 세션 정보
  session_count=$(aicli claude session list --format json | jq length)
  
  # 시스템 메트릭
  cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | sed 's/%us,//')
  mem_usage=$(free | grep Mem | awk '{printf "%.2f", $3/$2 * 100.0}')
  
  # 로그 출력
  echo "$timestamp,CPU:$cpu_usage%,Memory:$mem_usage%,Sessions:$session_count" \
    >> claude-metrics.csv
    
  echo "[$timestamp] CPU: $cpu_usage%, Memory: $mem_usage%, Sessions: $session_count"
  
  sleep 30
done
```

이러한 예제와 레시피를 통해 Claude CLI 통합의 다양한 활용 방법을 익히고, 실제 프로젝트에 적용할 수 있습니다. 각 예제는 실제 상황에서 바로 사용하거나 필요에 따라 수정하여 활용할 수 있도록 작성되었습니다.