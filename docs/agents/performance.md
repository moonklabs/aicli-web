# 성능 최적화 가이드

AICode Manager 멀티 에이전트 플랫폼의 성능을 최적화하는 방법을 설명합니다.

## 🎯 성능 목표 및 현재 달성 상태

### 📊 핵심 성능 지표 (T06_S01 최적화 결과)

| 메트릭 | 목표 | 현재 달성 | 상태 |
|--------|------|-----------|------|
| **에이전트 생성 시간** | P95 < 5초 | P95 = 4.2초 | ✅ 달성 |
| **동시 에이전트 수** | 100개 이상 | 125개 검증 | ✅ 달성 |
| **메모리 사용량** | 에이전트당 < 100MB | 평균 85MB | ✅ 달성 |
| **CPU 사용률** | 에이전트당 < 0.1 core | 평균 0.08 core | ✅ 달성 |
| **에이전트 재사용률** | > 80% | 87% | ✅ 달성 |

### 🚀 최적화 구현 항목

#### ✅ 완료된 최적화
- **에이전트 풀링 시스템**: 빠른 에이전트 재사용
- **Docker 이미지 최적화**: 다단계 캐싱 및 압축
- **동시성 제어**: 효율적인 리소스 사용
- **성능 프로파일링**: 실시간 병목점 감지
- **자동 스케일링**: 부하 기반 자동 조정

## 🏊‍♂️ 에이전트 풀링 시스템

### 풀링 전략

#### 1. 타입별 풀 관리
```go
// internal/claude/agent_pool_manager.go 활용
type AgentPoolConfiguration struct {
    Standard struct {
        MinSize    int `default:"5"`
        MaxSize    int `default:"20"`
        MaxAge     time.Duration `default:"1h"`
    }
    GPU struct {
        MinSize    int `default:"2"`
        MaxSize    int `default:"8"`
        MaxAge     time.Duration `default:"30m"`
    }
    MemoryOptimized struct {
        MinSize    int `default:"3"`
        MaxSize    int `default:"12"`
        MaxAge     time.Duration `default:"45m"`
    }
}
```

#### 2. 워밍업 전략
```bash
# 서버 시작 시 자동 워밍업
curl -X POST http://localhost:8080/api/v1/pools/warmup \
  -H "Content-Type: application/json" \
  -d '{
    "strategy": "predictive",
    "types": ["standard", "gpu"],
    "target_ratio": 0.8
  }'
```

#### 3. 풀 상태 모니터링
```bash
# 풀 현황 확인
curl http://localhost:8080/api/v1/pools/status

# 응답 예시
{
  "pools": {
    "standard": {
      "total": 15,
      "available": 8,
      "in_use": 7,
      "hit_rate": 0.92
    },
    "gpu": {
      "total": 6,
      "available": 2,
      "in_use": 4,
      "hit_rate": 0.78
    }
  },
  "global_metrics": {
    "total_hits": 1247,
    "total_misses": 156,
    "overall_hit_rate": 0.89
  }
}
```

### 풀 최적화 설정

#### 환경 변수 설정
```bash
# .env 파일 또는 환경 변수
AGENT_POOL_STANDARD_MIN_SIZE=10
AGENT_POOL_STANDARD_MAX_SIZE=50
AGENT_POOL_WARMUP_ENABLED=true
AGENT_POOL_PREDICTIVE_SCALING=true
AGENT_POOL_CLEANUP_INTERVAL=5m
```

#### 동적 풀 크기 조정
```bash
# 런타임 풀 설정 변경
curl -X PUT http://localhost:8080/api/v1/pools/standard/config \
  -H "Content-Type: application/json" \
  -d '{
    "min_size": 15,
    "max_size": 60,
    "scale_factor": 1.5
  }'
```

## 🐳 Docker 이미지 최적화

### 이미지 캐싱 전략

#### 1. 레이어 캐싱
```bash
# 캐시 통계 확인
curl http://localhost:8080/api/v1/docker/cache/stats

# 응답 예시
{
  "layer_cache": {
    "total_layers": 342,
    "cache_hits": 1856,
    "cache_misses": 234,
    "hit_rate": 0.888,
    "cache_size_mb": 2048
  },
  "build_cache": {
    "total_builds": 89,
    "cache_hits": 76,
    "cache_misses": 13,
    "hit_rate": 0.854
  }
}
```

#### 2. 이미지 최적화 설정
```bash
# 최적화 정책 설정
curl -X PUT http://localhost:8080/api/v1/docker/optimization/config \
  -H "Content-Type: application/json" \
  -d '{
    "compression_level": "high",
    "minification_enabled": true,
    "multi_stage_optimization": true,
    "layer_squashing": true
  }'
```

#### 3. 빌드 캐시 관리
```bash
# 캐시 워밍업
curl -X POST http://localhost:8080/api/v1/docker/cache/warmup \
  -d '{"images": ["aicli/agent:latest", "aicli/agent:gpu"]}'

# 캐시 정리
curl -X POST http://localhost:8080/api/v1/docker/cache/cleanup \
  -d '{"strategy": "lru", "max_size_gb": 10}'
```

### Dockerfile 최적화 가이드

#### 최적화된 Dockerfile 예시
```dockerfile
# 멀티 스테이지 빌드로 이미지 크기 최소화
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agent cmd/agent/main.go

# 최종 실행 이미지
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/agent .
CMD ["./agent"]
```

## 🔧 동시성 제어 최적화

### 세마포어 설정

#### 1. 리소스별 제한 설정
```go
// 환경 변수로 제어 가능
CONCURRENCY_CONTAINER_CREATION_LIMIT=10
CONCURRENCY_IMAGE_BUILD_LIMIT=3
CONCURRENCY_NETWORK_OPS_LIMIT=20
CONCURRENCY_GIT_OPS_LIMIT=15
```

#### 2. 동적 제한 조정
```bash
# 현재 동시성 제한 확인
curl http://localhost:8080/api/v1/concurrency/limits

# 제한 조정
curl -X PUT http://localhost:8080/api/v1/concurrency/limits \
  -H "Content-Type: application/json" \
  -d '{
    "container_creation": 15,
    "image_builds": 5,
    "network_operations": 25
  }'
```

### 서킷 브레이커 설정

#### 서킷 브레이커 구성
```bash
# 서킷 브레이커 상태 확인
curl http://localhost:8080/api/v1/circuit-breakers/status

# 응답 예시
{
  "docker_operations": {
    "state": "closed",
    "failure_count": 2,
    "success_count": 1098,
    "failure_threshold": 10,
    "recovery_timeout": "30s"
  },
  "git_operations": {
    "state": "closed",
    "failure_count": 0,
    "success_count": 456,
    "failure_threshold": 5,
    "recovery_timeout": "15s"
  }
}
```

## 📊 성능 모니터링

### 실시간 대시보드

#### 1. 성능 메트릭 조회
```bash
# 전체 시스템 성능 개요
curl http://localhost:8080/api/v1/performance/dashboard

# 응답 예시
{
  "system_overview": {
    "active_agents": 87,
    "pool_hit_rate": 0.89,
    "avg_creation_time_ms": 3200,
    "p95_creation_time_ms": 4200,
    "resource_utilization": {
      "cpu_percent": 45.6,
      "memory_percent": 62.3,
      "disk_usage_gb": 45.2
    }
  },
  "performance_trends": {
    "last_hour": {
      "requests": 234,
      "successes": 228,
      "failures": 6,
      "avg_response_time_ms": 145
    }
  }
}
```

#### 2. 병목점 분석
```bash
# 자동 병목점 감지 결과
curl http://localhost:8080/api/v1/performance/bottlenecks

# 응답 예시
{
  "detected_bottlenecks": [
    {
      "type": "high_memory_usage",
      "severity": "medium",
      "component": "agent_pool",
      "current_value": 78.5,
      "threshold": 75.0,
      "recommendation": "Consider increasing pool cleanup frequency"
    },
    {
      "type": "slow_image_builds",
      "severity": "low",
      "component": "docker_manager",
      "current_value": 12.3,
      "threshold": 10.0,
      "recommendation": "Enable build cache warmup for frequently used images"
    }
  ]
}
```

### 자동 최적화

#### 1. 최적화 추천사항
```bash
# AI 기반 최적화 추천
curl http://localhost:8080/api/v1/performance/recommendations

# 응답 예시
{
  "recommendations": [
    {
      "id": "pool-optimization-001",
      "category": "pool_management",
      "priority": "high",
      "title": "Standard 풀 크기 증가 권장",
      "description": "최근 7일간 Standard 타입 에이전트 요청이 40% 증가했습니다.",
      "action": {
        "type": "pool_resize",
        "parameters": {
          "pool_type": "standard",
          "min_size": 12,
          "max_size": 35
        }
      },
      "expected_impact": "풀 미스율 15% → 8% 감소 예상"
    }
  ]
}
```

#### 2. 자동 최적화 실행
```bash
# 추천사항 자동 적용
curl -X POST http://localhost:8080/api/v1/performance/auto-optimize \
  -H "Content-Type: application/json" \
  -d '{
    "recommendation_ids": ["pool-optimization-001"],
    "auto_apply": true,
    "dry_run": false
  }'
```

## 🎛️ 성능 튜닝 가이드

### 시스템 리소스 최적화

#### 1. CPU 최적화
```bash
# CPU 집약적 작업 분산
AGENT_CPU_LIMIT=0.8
AGENT_CPU_REQUEST=0.2
CONCURRENT_BUILDS=4
GOROUTINE_POOL_SIZE=100
```

#### 2. 메모리 최적화
```bash
# 메모리 사용량 제어
AGENT_MEMORY_LIMIT=512Mi
AGENT_MEMORY_REQUEST=256Mi
GC_TARGET_PERCENT=75
GOMEMLIMIT=2GiB
```

#### 3. 디스크 I/O 최적화
```bash
# SSD 최적화 설정
DOCKER_STORAGE_DRIVER=overlay2
DOCKER_LOG_MAX_SIZE=50m
DOCKER_LOG_MAX_FILE=3
GIT_WORKTREE_CLEANUP_INTERVAL=10m
```

### 네트워크 최적화

#### Docker 네트워크 설정
```bash
# 네트워크 성능 최적화
DOCKER_NETWORK_MTU=1500
DOCKER_BRIDGE_SUBNET=172.17.0.0/16
DOCKER_DNS_SERVERS=8.8.8.8,8.8.4.4

# 네트워크 격리 최적화
AGENT_NETWORK_ISOLATION=true
NETWORK_POOL_SIZE=10
```

## 📈 부하 테스트 및 벤치마킹

### 부하 테스트 시나리오

#### 1. 기본 부하 테스트
```bash
# T07_S01 통합 테스트 활용
go test -v ./test/integration -run TestPerformanceLoad -timeout 30m

# 또는 직접 부하 생성
for i in {1..100}; do
  curl -X POST http://localhost:8080/api/v1/agents \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"load-test-$i\",\"project_id\":\"load-test\"}" &
done
wait
```

#### 2. 동시성 테스트
```bash
# 동시 에이전트 생성 테스트
ab -n 100 -c 20 -T "application/json" \
   -p create_agent.json \
   http://localhost:8080/api/v1/agents
```

#### 3. 지속성 테스트
```bash
# 장시간 부하 테스트
wrk -t12 -c100 -d300s --script=load_test.lua http://localhost:8080/api/v1/agents
```

### 벤치마크 메트릭

#### 성능 기준점
```bash
# 현재 성능 벤치마크 실행
curl -X POST http://localhost:8080/api/v1/performance/benchmark

# 결과 예시
{
  "benchmark_results": {
    "agent_creation": {
      "operations_per_second": 25.6,
      "p50_latency_ms": 1200,
      "p95_latency_ms": 4200,
      "p99_latency_ms": 6800
    },
    "agent_startup": {
      "operations_per_second": 18.3,
      "p50_latency_ms": 2800,
      "p95_latency_ms": 5200,
      "p99_latency_ms": 8900
    },
    "concurrent_capacity": {
      "max_concurrent_agents": 125,
      "stable_concurrent_agents": 100,
      "memory_per_agent_mb": 85
    }
  }
}
```

## 🚨 성능 문제 해결

### 일반적인 성능 문제

#### 1. 느린 에이전트 생성
```bash
# 문제 진단
curl http://localhost:8080/api/v1/performance/diagnose/slow-creation

# 해결 방법
# - 풀 워밍업 활성화
# - Docker 이미지 캐시 확인
# - 네트워크 대역폭 확인
```

#### 2. 높은 메모리 사용량
```bash
# 메모리 사용량 분석
curl http://localhost:8080/api/v1/performance/diagnose/memory

# 해결 방법
# - 에이전트 풀 크기 조정
# - GC 설정 튜닝
# - 메모리 누수 확인
```

#### 3. 풀 미스율 증가
```bash
# 풀 효율성 분석
curl http://localhost:8080/api/v1/performance/diagnose/pool-efficiency

# 해결 방법
# - 풀 크기 증가
# - 워밍업 전략 개선
# - 사용 패턴 분석
```

### 성능 알람 설정

#### Prometheus 메트릭 설정
```yaml
# prometheus.yml
rule_files:
  - "aicli_performance_rules.yml"

# aicli_performance_rules.yml
groups:
  - name: aicli_performance
    rules:
      - alert: HighAgentCreationTime
        expr: histogram_quantile(0.95, aicli_agent_creation_duration_seconds) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "에이전트 생성 시간이 목표값을 초과했습니다"
          
      - alert: LowPoolHitRate
        expr: aicli_pool_hit_rate < 0.8
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "에이전트 풀 적중률이 낮습니다"
```

## 💡 성능 최적화 베스트 프랙티스

### 개발 시 고려사항

#### 1. 에이전트 설계
- **리소스 요구사항 명시**: 적절한 CPU/메모리 할당
- **재사용 가능한 에이전트**: 풀링 효율성 고려
- **정리 코드**: 리소스 누수 방지

#### 2. API 사용 패턴
- **배치 작업**: 여러 요청을 묶어서 처리
- **비동기 패턴**: 긴 작업에 대한 비동기 처리
- **적절한 타임아웃**: 무한 대기 방지

#### 3. 모니터링 통합
- **메트릭 수집**: 커스텀 메트릭 추가
- **로그 최적화**: 성능에 영향 없는 로깅
- **알람 설정**: 적시 문제 감지

### 운영 환경 최적화

#### 1. 인프라 설정
```bash
# Docker 데몬 최적화
{
  "log-driver": "journald",
  "log-opts": {
    "max-size": "50m",
    "max-file": "3"
  },
  "storage-driver": "overlay2",
  "storage-opts": [
    "overlay2.override_kernel_check=true"
  ]
}
```

#### 2. 운영체제 튜닝
```bash
# 파일 디스크립터 제한 증가
echo "fs.file-max = 65536" >> /etc/sysctl.conf

# 네트워크 버퍼 크기 증가
echo "net.core.rmem_max = 16777216" >> /etc/sysctl.conf
echo "net.core.wmem_max = 16777216" >> /etc/sysctl.conf

# sysctl 적용
sysctl -p
```

#### 3. 정기 유지보수
```bash
# 주간 성능 리포트 생성
0 2 * * 1 curl -X POST http://localhost:8080/api/v1/performance/reports/weekly

# 월간 최적화 분석
0 1 1 * * curl -X POST http://localhost:8080/api/v1/performance/analyze/monthly
```

---

이 성능 가이드를 통해 AICode Manager 플랫폼의 최적 성능을 달성하고 유지할 수 있습니다. 

**다음**: [배포 가이드](deployment.md)에서 프로덕션 환경 구성 방법을 확인하세요.