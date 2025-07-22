# Advanced Session Pool Management

고급 세션 풀 관리 시스템은 대규모 환경에서 Claude AI 세션의 생명주기를 효율적으로 관리하여 최적의 성능과 리소스 활용을 제공합니다.

## 📋 목차

- [개요](#개요)
- [아키텍처](#아키텍처)
- [구성 옵션](#구성-옵션)
- [동적 스케일링](#동적-스케일링)
- [로드 밸런싱](#로드-밸런싱)
- [세션 재사용](#세션-재사용)
- [모니터링](#모니터링)
- [문제 해결](#문제-해결)

## 🎯 개요

### 주요 기능

- **동적 스케일링**: 부하에 따른 자동 세션 풀 크기 조정
- **지능형 로드 밸런싱**: 여러 전략을 통한 최적 세션 할당
- **세션 재사용**: 리소스 효율성을 위한 유휴 세션 재활용
- **건강성 모니터링**: 실시간 세션 상태 추적 및 복구
- **리소스 제한**: 메모리 및 CPU 사용량 제어

### 성능 지표

```bash
# 목표 성능
Session Creation Rate: 100+ sessions/sec
Session Reuse Rate: 80%+
Average Session Latency: <50ms
Resource Efficiency: <1MB per session
```

## 🏗️ 아키텍처

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client App    │────│   Load Balancer  │────│  Session Pool   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                         │
                                │                         │
                       ┌──────────────────┐    ┌─────────────────┐
                       │   Auto Scaler    │────│  Pool Monitor   │
                       └──────────────────┘    └─────────────────┘
                                │                         │
                                │                         │
                       ┌──────────────────┐    ┌─────────────────┐
                       │ Health Checker   │────│  Metrics Store  │
                       └──────────────────┘    └─────────────────┘
```

### 핵심 컴포넌트

1. **AdvancedSessionPool**: 메인 세션 풀 관리자
2. **AutoScaler**: 동적 스케일링 엔진
3. **LoadBalancer**: 세션 할당 로드 밸런서
4. **PoolMonitor**: 실시간 모니터링
5. **HealthChecker**: 세션 건강성 검사
6. **MetricsCollector**: 성능 메트릭 수집

## ⚙️ 구성 옵션

### 기본 설정

```yaml
# config/session-pool.yaml
session_pool:
  # 풀 크기 설정
  min_size: 10              # 최소 세션 수
  max_size: 100             # 최대 세션 수
  initial_size: 20          # 초기 세션 수
  
  # 자동 스케일링
  auto_scaling:
    enabled: true
    scale_up_threshold: 0.8   # 80% 사용률에서 확장
    scale_down_threshold: 0.3 # 30% 사용률에서 축소
    scale_up_increment: 5     # 확장 시 증가량
    scale_down_increment: 2   # 축소 시 감소량
    cooldown_period: 30s      # 스케일링 간격
  
  # 로드 밸런싱
  load_balancing:
    strategy: "weighted_round_robin"  # round_robin, least_connections, weighted_round_robin
    health_check_interval: 30s
    circuit_breaker_enabled: true
  
  # 세션 관리
  session_management:
    enable_reuse: true
    idle_timeout: 300s        # 5분 유휴 시간 후 재사용 풀로 이동
    max_lifetime: 3600s       # 1시간 후 강제 종료
    cleanup_interval: 60s     # 정리 작업 간격
  
  # 리소스 제한
  resource_limits:
    max_memory_per_session: 100MB
    max_total_memory: 1GB
    max_concurrent_requests: 1000
```

### 환경 변수

```bash
# 환경 변수로 설정 가능
export AICLI_SESSION_POOL_MIN_SIZE=10
export AICLI_SESSION_POOL_MAX_SIZE=100
export AICLI_SESSION_POOL_AUTO_SCALING=true
export AICLI_SESSION_POOL_LOAD_BALANCING=weighted_round_robin
```

## 📈 동적 스케일링

### 스케일링 트리거

```go
// 자동 스케일링 조건
type ScalingCondition struct {
    Metric    string  // "cpu_usage", "memory_usage", "request_rate", "queue_length"
    Threshold float64 // 임계값
    Duration  time.Duration // 지속 시간
}

// 예제: CPU 사용률 80% 이상 30초 지속 시 스케일 업
scaleUpCondition := ScalingCondition{
    Metric:    "cpu_usage",
    Threshold: 0.8,
    Duration:  30 * time.Second,
}
```

### 스케일링 정책

```yaml
scaling_policies:
  # CPU 기반 스케일링
  - name: "cpu_scaling"
    metric: "cpu_usage"
    scale_up:
      threshold: 0.8
      increment: 5
    scale_down:
      threshold: 0.3
      decrement: 2
  
  # 메모리 기반 스케일링
  - name: "memory_scaling"
    metric: "memory_usage"
    scale_up:
      threshold: 0.7
      increment: 3
    scale_down:
      threshold: 0.2
      decrement: 1
  
  # 요청률 기반 스케일링
  - name: "request_rate_scaling"
    metric: "request_rate"
    scale_up:
      threshold: 90  # requests/sec
      increment: 10
    scale_down:
      threshold: 30
      decrement: 5
```

### 프로그래매틱 제어

```go
// 수동 스케일링
poolManager := claude.NewAdvancedSessionPool(config)

// 스케일 업
err := poolManager.ScaleUp(5)
if err != nil {
    log.Printf("Scale up failed: %v", err)
}

// 스케일 다운
err = poolManager.ScaleDown(3)
if err != nil {
    log.Printf("Scale down failed: %v", err)
}

// 목표 크기 설정
err = poolManager.SetTargetSize(50)
if err != nil {
    log.Printf("Set target size failed: %v", err)
}
```

## ⚖️ 로드 밸런싱

### 로드 밸런싱 전략

#### 1. Round Robin
```go
// 순차적으로 세션 할당
strategy := claude.LoadBalancingRoundRobin
poolManager.SetLoadBalancingStrategy(strategy)
```

#### 2. Least Connections
```go
// 가장 적은 연결 수를 가진 세션 선택
strategy := claude.LoadBalancingLeastConnections
poolManager.SetLoadBalancingStrategy(strategy)
```

#### 3. Weighted Round Robin
```go
// 가중치 기반 순차 할당
strategy := claude.LoadBalancingWeightedRoundRobin
poolManager.SetLoadBalancingStrategy(strategy)

// 세션별 가중치 설정
poolManager.SetSessionWeight("session-1", 1.0)
poolManager.SetSessionWeight("session-2", 2.0)  // 2배 더 많은 요청 처리
```

#### 4. Performance-based
```go
// 성능 지표 기반 할당
strategy := claude.LoadBalancingPerformanceBased
poolManager.SetLoadBalancingStrategy(strategy)

// 성능 메트릭 가중치 설정
weights := claude.PerformanceWeights{
    Latency:    0.4,  // 40% - 응답 시간
    Throughput: 0.3,  // 30% - 처리량
    ErrorRate:  0.2,  // 20% - 오류율
    Resource:   0.1,  // 10% - 리소스 사용률
}
poolManager.SetPerformanceWeights(weights)
```

### 건강성 검사

```yaml
health_check:
  enabled: true
  interval: 30s
  timeout: 5s
  failure_threshold: 3      # 3회 연속 실패 시 unhealthy
  recovery_threshold: 2     # 2회 연속 성공 시 healthy
  
  checks:
    - type: "ping"
      endpoint: "/health"
    - type: "response_time"
      max_latency: 100ms
    - type: "memory_usage"
      max_usage: 80%
```

## 🔄 세션 재사용

### 재사용 정책

```go
type ReusePolicy struct {
    EnableReuse     bool          // 재사용 활성화
    MaxIdleTime     time.Duration // 최대 유휴 시간
    MaxLifetime     time.Duration // 최대 생존 시간
    MaxReuseCount   int           // 최대 재사용 횟수
    ContextReset    bool          // 컨텍스트 초기화 여부
}

policy := ReusePolicy{
    EnableReuse:   true,
    MaxIdleTime:   5 * time.Minute,
    MaxLifetime:   1 * time.Hour,
    MaxReuseCount: 100,
    ContextReset:  true,
}

poolManager.SetReusePolicy(policy)
```

### 세션 생명주기

```
┌─────────────┐    create    ┌─────────────┐    assign    ┌─────────────┐
│   Pool      │─────────────→│   Ready     │─────────────→│   Active    │
│  (Empty)    │              │  (Idle)     │              │ (In Use)    │
└─────────────┘              └─────────────┘              └─────────────┘
                                     ↑                             │
                                     │ release                     │ complete/timeout
                                     │                             ↓
┌─────────────┐    cleanup   ┌─────────────┐    validate  ┌─────────────┐
│  Destroyed  │←─────────────│   Expired   │←─────────────│  Candidate  │
│             │              │             │              │ (Returning) │
└─────────────┘              └─────────────┘              └─────────────┘
```

### 성능 최적화

```go
// 세션 풀 워밍업
func (p *AdvancedSessionPool) WarmUp(ctx context.Context, targetSize int) error {
    var wg sync.WaitGroup
    errChan := make(chan error, targetSize)
    
    for i := 0; i < targetSize; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            session, err := p.createSession(ctx)
            if err != nil {
                errChan <- err
                return
            }
            
            p.addToPool(session)
        }()
    }
    
    wg.Wait()
    close(errChan)
    
    // 오류 수집
    var errors []error
    for err := range errChan {
        errors = append(errors, err)
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("warmup failed with %d errors", len(errors))
    }
    
    return nil
}
```

## 📊 모니터링

### 핵심 메트릭

```prometheus
# 세션 풀 크기
aicli_session_pool_size{type="total"} 45
aicli_session_pool_size{type="active"} 32
aicli_session_pool_size{type="idle"} 13
aicli_session_pool_size{type="ready"} 8

# 세션 처리량
aicli_session_pool_operations_total{operation="create"} 1250
aicli_session_pool_operations_total{operation="acquire"} 2100
aicli_session_pool_operations_total{operation="release"} 2050
aicli_session_pool_operations_total{operation="destroy"} 45

# 재사용 메트릭
aicli_session_pool_reuse_rate 0.85
aicli_session_pool_reuse_count_total 1680

# 성능 메트릭
aicli_session_pool_latency_histogram{operation="acquire",le="10"} 856
aicli_session_pool_latency_histogram{operation="acquire",le="50"} 1980
aicli_session_pool_latency_histogram{operation="acquire",le="100"} 2100

# 리소스 사용률
aicli_session_pool_resource_usage{resource="memory"} 0.65
aicli_session_pool_resource_usage{resource="cpu"} 0.42
```

### 대시보드 설정

```json
{
  "dashboard": {
    "title": "Session Pool Management",
    "panels": [
      {
        "title": "Pool Size",
        "type": "graph",
        "targets": [
          {
            "expr": "aicli_session_pool_size",
            "legendFormat": "{{type}}"
          }
        ]
      },
      {
        "title": "Session Operations Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(aicli_session_pool_operations_total[5m])",
            "legendFormat": "{{operation}}/sec"
          }
        ]
      },
      {
        "title": "Reuse Rate",
        "type": "singlestat",
        "targets": [
          {
            "expr": "aicli_session_pool_reuse_rate",
            "format": "percent"
          }
        ]
      }
    ]
  }
}
```

### 알람 규칙

```yaml
groups:
  - name: session_pool_alerts
    rules:
      - alert: SessionPoolHighUtilization
        expr: aicli_session_pool_size{type="active"} / aicli_session_pool_size{type="total"} > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Session pool utilization is high"
          description: "Session pool utilization is {{ $value | humanizePercentage }}"
      
      - alert: SessionPoolLowReuseRate
        expr: aicli_session_pool_reuse_rate < 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Session reuse rate is low"
          description: "Session reuse rate is {{ $value | humanizePercentage }}"
      
      - alert: SessionCreationFailure
        expr: rate(aicli_session_pool_operations_total{operation="create",result="error"}[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High session creation failure rate"
          description: "Session creation failure rate is {{ $value | humanize }}/sec"
```

## 🛠️ 문제 해결

### 일반적인 문제

#### 1. 세션 풀 고갈

**증상**: 새로운 세션 요청이 지연되거나 실패

**원인 분석**:
```bash
# 현재 풀 상태 확인
curl http://localhost:8080/api/v1/session-pool/status

# 메트릭 확인
curl http://localhost:8080/api/v1/metrics | grep session_pool
```

**해결 방법**:
```yaml
# 풀 크기 증가
session_pool:
  max_size: 200  # 기존 100에서 증가
  
# 더 빠른 스케일링
auto_scaling:
  scale_up_threshold: 0.7  # 기존 0.8에서 감소
  scale_up_increment: 10   # 기존 5에서 증가
```

#### 2. 메모리 누수

**증상**: 시간이 지남에 따라 메모리 사용량 지속 증가

**진단**:
```bash
# 메모리 프로파일 수집
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# 고루틴 누수 확인
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

**해결**:
```go
// 세션 정리 강화
config.SessionManagement.CleanupInterval = 30 * time.Second
config.SessionManagement.MaxLifetime = 30 * time.Minute

// 리소스 제한 강화
config.ResourceLimits.MaxMemoryPerSession = 50 * 1024 * 1024 // 50MB
```

#### 3. 로드 밸런싱 불균형

**증상**: 일부 세션에 요청이 집중되고 다른 세션은 유휴 상태

**확인**:
```bash
# 세션별 요청 분산 확인
curl http://localhost:8080/api/v1/session-pool/sessions | jq '.sessions[] | {id: .id, request_count: .request_count}'
```

**해결**:
```yaml
load_balancing:
  strategy: "least_connections"  # round_robin에서 변경
  health_check_interval: 15s     # 더 빈번한 건강성 검사
```

### 디버깅 도구

#### 세션 풀 상태 조회
```bash
#!/bin/bash
# session-pool-debug.sh

echo "=== Session Pool Status ==="
curl -s http://localhost:8080/api/v1/session-pool/status | jq '.'

echo -e "\n=== Session Details ==="
curl -s http://localhost:8080/api/v1/session-pool/sessions | jq '.sessions[] | {id: .id, status: .status, created_at: .created_at, request_count: .request_count}'

echo -e "\n=== Resource Usage ==="
curl -s http://localhost:8080/api/v1/metrics | grep -E "(memory|cpu|goroutine)" | head -10
```

#### 성능 테스트
```go
// 부하 테스트
func BenchmarkSessionPoolPerformance(b *testing.B) {
    pool := setupTestPool()
    defer pool.Close()
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            session, err := pool.AcquireSession(context.Background())
            if err != nil {
                b.Error(err)
                continue
            }
            
            // 작업 시뮬레이션
            time.Sleep(10 * time.Millisecond)
            
            pool.ReleaseSession(session)
        }
    })
}
```

---

**다음 단계**: [웹 인터페이스 통합](./web-interface-integration.md)에서 실시간 웹 통신 기능을 알아보세요.