---
title: Monitoring System Technical Specifications
document_type: SPECS
milestone: M07
status: draft
last_updated: 2025-08-01 07:10
---

# Technical Specifications: Monitoring System

## Overview

AICode Manager의 실시간 모니터링 및 관찰성(Observability) 시스템을 구축합니다. 메트릭, 로그, 트레이스를 통합하여 시스템 상태를 완벽하게 파악하고 문제를 신속하게 해결할 수 있도록 합니다.

## Architecture

### 모니터링 시스템 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                     Applications                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   API       │  │   Worker    │  │     Agent           │ │
│  │  Server     │  │   Nodes     │  │   Containers        │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
│         │                 │                     │            │
│    [Exporters]      [Exporters]           [Exporters]       │
└─────────┴─────────────────┴───────────────────┴────────────┘
          │                 │                     │
          ▼                 ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                    Data Collection Layer                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │ Prometheus  │  │   Fluent    │  │      Jaeger         │ │
│  │  (Metrics)  │  │   (Logs)    │  │    (Traces)         │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
└─────────┴─────────────────┴───────────────────┴────────────┘
          │                 │                     │
          ▼                 ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                     Storage Layer                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   Cortex    │  │Elasticsearch│  │   Jaeger Storage    │ │
│  │   (TSDB)    │  │  (Logs DB)  │  │    (Trace DB)       │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
└─────────┴─────────────────┴───────────────────┴────────────┘
          │                 │                     │
          ▼                 ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                  Visualization Layer                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                      Grafana                         │   │
│  │  ┌─────────┐  ┌──────────┐  ┌─────────────────┐   │   │
│  │  │ Metrics │  │   Logs   │  │     Traces      │   │   │
│  │  │  Panels │  │  Panels  │  │     Panels      │   │   │
│  │  └─────────┘  └──────────┘  └─────────────────┘   │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Detailed Specifications

### 1. Metrics Collection

#### 1.1 Prometheus Integration

```go
// metrics/prometheus.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
    // HTTP 메트릭
    HTTPRequestsTotal    *prometheus.CounterVec
    HTTPRequestDuration  *prometheus.HistogramVec
    HTTPRequestsInFlight prometheus.Gauge
    
    // 비즈니스 메트릭
    ActiveWorkspaces    prometheus.Gauge
    AgentSessions       *prometheus.GaugeVec
    TasksProcessed      *prometheus.CounterVec
    TaskDuration        *prometheus.HistogramVec
    
    // 시스템 메트릭
    DBConnections       *prometheus.GaugeVec
    CacheHitRate        *prometheus.GaugeVec
    QueueDepth          *prometheus.GaugeVec
    
    // 에러 메트릭
    ErrorsTotal         *prometheus.CounterVec
    PanicsRecovered     prometheus.Counter
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
    m := &Metrics{
        HTTPRequestsTotal: promauto.With(reg).NewCounterVec(
            prometheus.CounterOpts{
                Name: "aicli_http_requests_total",
                Help: "Total number of HTTP requests",
            },
            []string{"method", "endpoint", "status"},
        ),
        
        HTTPRequestDuration: promauto.With(reg).NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "aicli_http_request_duration_seconds",
                Help:    "HTTP request latencies",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method", "endpoint"},
        ),
        
        HTTPRequestsInFlight: promauto.With(reg).NewGauge(
            prometheus.GaugeOpts{
                Name: "aicli_http_requests_in_flight",
                Help: "Current number of HTTP requests being served",
            },
        ),
        
        ActiveWorkspaces: promauto.With(reg).NewGauge(
            prometheus.GaugeOpts{
                Name: "aicli_active_workspaces",
                Help: "Number of active workspaces",
            },
        ),
        
        AgentSessions: promauto.With(reg).NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "aicli_agent_sessions",
                Help: "Number of agent sessions by type",
            },
            []string{"agent_type", "status"},
        ),
        
        TasksProcessed: promauto.With(reg).NewCounterVec(
            prometheus.CounterOpts{
                Name: "aicli_tasks_processed_total",
                Help: "Total number of tasks processed",
            },
            []string{"task_type", "status"},
        ),
        
        TaskDuration: promauto.With(reg).NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "aicli_task_duration_seconds",
                Help:    "Task processing duration",
                Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120, 300},
            },
            []string{"task_type"},
        ),
        
        ErrorsTotal: promauto.With(reg).NewCounterVec(
            prometheus.CounterOpts{
                Name: "aicli_errors_total",
                Help: "Total number of errors",
            },
            []string{"type", "component"},
        ),
    }
    
    return m
}
```

#### 1.2 Custom Metrics Collector

```go
// metrics/collector.go
package metrics

import (
    "context"
    "time"
)

type MetricsCollector struct {
    metrics   *Metrics
    services  *ServiceRegistry
    interval  time.Duration
}

func (mc *MetricsCollector) Start(ctx context.Context) {
    ticker := time.NewTicker(mc.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            mc.collect()
        }
    }
}

func (mc *MetricsCollector) collect() {
    // 워크스페이스 메트릭
    activeWorkspaces := mc.services.WorkspaceService.CountActive()
    mc.metrics.ActiveWorkspaces.Set(float64(activeWorkspaces))
    
    // 에이전트 세션 메트릭
    sessions := mc.services.SessionManager.GetSessionStats()
    for agentType, stats := range sessions {
        mc.metrics.AgentSessions.WithLabelValues(
            agentType, "active",
        ).Set(float64(stats.Active))
        mc.metrics.AgentSessions.WithLabelValues(
            agentType, "idle",
        ).Set(float64(stats.Idle))
    }
    
    // 데이터베이스 연결 메트릭
    dbStats := mc.services.Database.Stats()
    mc.metrics.DBConnections.WithLabelValues("open").Set(float64(dbStats.OpenConnections))
    mc.metrics.DBConnections.WithLabelValues("idle").Set(float64(dbStats.Idle))
    mc.metrics.DBConnections.WithLabelValues("in_use").Set(float64(dbStats.InUse))
    
    // 캐시 메트릭
    cacheStats := mc.services.Cache.Stats()
    hitRate := float64(cacheStats.Hits) / float64(cacheStats.Hits+cacheStats.Misses)
    mc.metrics.CacheHitRate.WithLabelValues("l1").Set(hitRate)
}
```

### 2. Logging System

#### 2.1 Structured Logging

```go
// logging/logger.go
package logging

import (
    "context"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

type Logger struct {
    *zap.Logger
    config LogConfig
}

type LogConfig struct {
    Level       string
    Format      string // json, console
    OutputPaths []string
    Sampling    SamplingConfig
}

type SamplingConfig struct {
    Initial    int
    Thereafter int
    Tick       time.Duration
}

func NewLogger(config LogConfig) (*Logger, error) {
    // Zap 설정 구성
    zapConfig := zap.Config{
        Level:            parseLevel(config.Level),
        Development:      config.Level == "debug",
        Encoding:         config.Format,
        OutputPaths:      config.OutputPaths,
        ErrorOutputPaths: []string{"stderr"},
        EncoderConfig:    newEncoderConfig(),
        Sampling: &zap.SamplingConfig{
            Initial:    config.Sampling.Initial,
            Thereafter: config.Sampling.Thereafter,
            Hook:       samplingHook,
        },
    }
    
    // 추가 필드 설정
    zapConfig.InitialFields = map[string]interface{}{
        "service": "aicli-web",
        "version": version.Version,
        "host":    hostname,
    }
    
    logger, err := zapConfig.Build(
        zap.AddCaller(),
        zap.AddStacktrace(zapcore.ErrorLevel),
    )
    if err != nil {
        return nil, err
    }
    
    return &Logger{Logger: logger, config: config}, nil
}

// 컨텍스트 기반 로깅
func (l *Logger) WithContext(ctx context.Context) *Logger {
    fields := []zap.Field{}
    
    // 요청 ID 추가
    if reqID := ctx.Value("request_id"); reqID != nil {
        fields = append(fields, zap.String("request_id", reqID.(string)))
    }
    
    // 사용자 ID 추가
    if userID := ctx.Value("user_id"); userID != nil {
        fields = append(fields, zap.String("user_id", userID.(string)))
    }
    
    // 워크스페이스 ID 추가
    if wsID := ctx.Value("workspace_id"); wsID != nil {
        fields = append(fields, zap.String("workspace_id", wsID.(string)))
    }
    
    return &Logger{Logger: l.With(fields...), config: l.config}
}
```

#### 2.2 Log Aggregation

```go
// logging/fluent.go
package logging

import (
    "github.com/fluent/fluent-logger-golang/fluent"
)

type FluentForwarder struct {
    client *fluent.Fluent
    config FluentConfig
}

type FluentConfig struct {
    Host         string
    Port         int
    Tag          string
    BufferLimit  int
    MaxRetry     int
    AsyncConnect bool
}

func NewFluentForwarder(config FluentConfig) (*FluentForwarder, error) {
    client, err := fluent.New(fluent.Config{
        FluentHost:   config.Host,
        FluentPort:   config.Port,
        BufferLimit:  config.BufferLimit,
        MaxRetry:     config.MaxRetry,
        AsyncConnect: config.AsyncConnect,
    })
    if err != nil {
        return nil, err
    }
    
    return &FluentForwarder{
        client: client,
        config: config,
    }, nil
}

func (ff *FluentForwarder) Forward(log LogEntry) error {
    data := map[string]interface{}{
        "timestamp": log.Timestamp,
        "level":     log.Level,
        "message":   log.Message,
        "fields":    log.Fields,
        "trace_id":  log.TraceID,
        "span_id":   log.SpanID,
    }
    
    return ff.client.Post(ff.config.Tag, data)
}
```

### 3. Distributed Tracing

#### 3.1 OpenTelemetry Integration

```go
// tracing/otel.go
package tracing

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type TracerProvider struct {
    provider *trace.TracerProvider
    config   TracingConfig
}

type TracingConfig struct {
    ServiceName  string
    JaegerURL    string
    SampleRate   float64
    MaxSpans     int
    BatchTimeout time.Duration
}

func NewTracerProvider(config TracingConfig) (*TracerProvider, error) {
    // Jaeger exporter 생성
    exporter, err := jaeger.New(
        jaeger.WithCollectorEndpoint(
            jaeger.WithEndpoint(config.JaegerURL),
        ),
    )
    if err != nil {
        return nil, err
    }
    
    // 리소스 정의
    res, err := resource.Merge(
        resource.Default(),
        resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(config.ServiceName),
            semconv.ServiceVersionKey.String(version.Version),
            attribute.String("environment", getEnvironment()),
        ),
    )
    if err != nil {
        return nil, err
    }
    
    // TracerProvider 생성
    provider := trace.NewTracerProvider(
        trace.WithBatcher(exporter,
            trace.WithBatchTimeout(config.BatchTimeout),
            trace.WithMaxExportBatchSize(config.MaxSpans),
        ),
        trace.WithResource(res),
        trace.WithSampler(trace.TraceIDRatioBased(config.SampleRate)),
    )
    
    otel.SetTracerProvider(provider)
    
    return &TracerProvider{
        provider: provider,
        config:   config,
    }, nil
}
```

#### 3.2 Trace Instrumentation

```go
// tracing/instrumentation.go
package tracing

import (
    "context"
    "fmt"
    "net/http"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("aicli-web")

// HTTP 미들웨어 계측
func TracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, span := tracer.Start(r.Context(), 
            fmt.Sprintf("%s %s", r.Method, r.URL.Path),
            trace.WithAttributes(
                attribute.String("http.method", r.Method),
                attribute.String("http.url", r.URL.String()),
                attribute.String("http.scheme", r.URL.Scheme),
                attribute.String("http.host", r.Host),
                attribute.String("http.user_agent", r.UserAgent()),
            ),
        )
        defer span.End()
        
        // Response writer 래핑
        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        // 다음 핸들러 실행
        next.ServeHTTP(rw, r.WithContext(ctx))
        
        // Response 속성 추가
        span.SetAttributes(
            attribute.Int("http.status_code", rw.statusCode),
            attribute.Int64("http.response_size", rw.size),
        )
        
        // 에러 상태 체크
        if rw.statusCode >= 400 {
            span.SetStatus(trace.Status{
                Code:        trace.StatusCodeError,
                Description: http.StatusText(rw.statusCode),
            })
        }
    })
}

// 데이터베이스 계측
func TraceDB(ctx context.Context, operation string, query string) (context.Context, trace.Span) {
    return tracer.Start(ctx, operation,
        trace.WithAttributes(
            attribute.String("db.system", "postgresql"),
            attribute.String("db.statement", query),
            attribute.String("db.operation", operation),
        ),
    )
}

// 캐시 계측
func TraceCache(ctx context.Context, operation string, key string) (context.Context, trace.Span) {
    return tracer.Start(ctx, fmt.Sprintf("cache.%s", operation),
        trace.WithAttributes(
            attribute.String("cache.operation", operation),
            attribute.String("cache.key", key),
        ),
    )
}

// 외부 API 호출 계측
func TraceHTTPClient(ctx context.Context, req *http.Request) (context.Context, trace.Span) {
    return tracer.Start(ctx, fmt.Sprintf("HTTP %s", req.URL.Host),
        trace.WithAttributes(
            attribute.String("http.method", req.Method),
            attribute.String("http.url", req.URL.String()),
            attribute.String("peer.service", req.URL.Host),
        ),
    )
}
```

### 4. Alerting System

#### 4.1 Alert Manager Integration

```go
// alerting/manager.go
package alerting

import (
    "context"
    "fmt"
)

type AlertManager struct {
    config   AlertConfig
    channels []AlertChannel
    rules    []AlertRule
}

type AlertConfig struct {
    WebhookURL    string
    DefaultChannel string
    Throttle      ThrottleConfig
}

type ThrottleConfig struct {
    Duration time.Duration
    MaxAlerts int
}

type Alert struct {
    ID          string
    Name        string
    Severity    Severity
    Description string
    Details     map[string]interface{}
    Timestamp   time.Time
    Labels      map[string]string
}

type AlertRule struct {
    Name       string
    Expression string
    Duration   time.Duration
    Severity   Severity
    Channels   []string
    Actions    []AlertAction
}

func (am *AlertManager) SendAlert(ctx context.Context, alert Alert) error {
    // 알림 중복 제거
    if am.isDuplicate(alert) {
        return nil
    }
    
    // 심각도별 채널 선택
    channels := am.selectChannels(alert.Severity)
    
    // 각 채널로 알림 전송
    for _, channel := range channels {
        if err := channel.Send(ctx, alert); err != nil {
            // 에러 로깅하지만 계속 진행
            log.Error("Failed to send alert", "channel", channel.Name(), "error", err)
        }
    }
    
    // 액션 실행
    am.executeActions(ctx, alert)
    
    return nil
}
```

#### 4.2 Alert Channels

```go
// alerting/channels.go
package alerting

type SlackChannel struct {
    webhookURL string
    channel    string
}

func (sc *SlackChannel) Send(ctx context.Context, alert Alert) error {
    payload := map[string]interface{}{
        "channel": sc.channel,
        "attachments": []map[string]interface{}{
            {
                "color":      getColorBySeverity(alert.Severity),
                "title":      alert.Name,
                "text":       alert.Description,
                "fields":     formatFields(alert.Details),
                "footer":     "AICode Manager Alert",
                "ts":         alert.Timestamp.Unix(),
            },
        },
    }
    
    return sendWebhook(ctx, sc.webhookURL, payload)
}

type EmailChannel struct {
    smtp     SMTPConfig
    from     string
    to       []string
    template string
}

func (ec *EmailChannel) Send(ctx context.Context, alert Alert) error {
    subject := fmt.Sprintf("[%s] %s", alert.Severity, alert.Name)
    body, err := renderTemplate(ec.template, alert)
    if err != nil {
        return err
    }
    
    return sendEmail(ec.smtp, ec.from, ec.to, subject, body)
}

type PagerDutyChannel struct {
    routingKey string
    client     *pagerduty.Client
}

func (pc *PagerDutyChannel) Send(ctx context.Context, alert Alert) error {
    event := &pagerduty.Event{
        RoutingKey: pc.routingKey,
        Action:     "trigger",
        DedupKey:   alert.ID,
        Payload: &pagerduty.Payload{
            Summary:   alert.Name,
            Severity:  string(alert.Severity),
            Source:    "aicli-web",
            Details:   alert.Details,
            Timestamp: alert.Timestamp,
        },
    }
    
    _, err := pc.client.CreateEvent(event)
    return err
}
```

### 5. Dashboard Configuration

#### 5.1 Grafana Dashboards

```json
{
  "dashboard": {
    "title": "AICode Manager Overview",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(aicli_http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Response Time (P95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(aicli_http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "{{endpoint}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Active Workspaces",
        "targets": [
          {
            "expr": "aicli_active_workspaces"
          }
        ],
        "type": "stat"
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(aicli_errors_total[5m])",
            "legendFormat": "{{type}} - {{component}}"
          }
        ],
        "type": "graph",
        "alert": {
          "conditions": [
            {
              "evaluator": {
                "params": [0.1],
                "type": "gt"
              }
            }
          ]
        }
      },
      {
        "title": "Database Connections",
        "targets": [
          {
            "expr": "aicli_db_connections",
            "legendFormat": "{{state}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Cache Hit Rate",
        "targets": [
          {
            "expr": "rate(aicli_cache_hits_total[5m]) / (rate(aicli_cache_hits_total[5m]) + rate(aicli_cache_misses_total[5m]))",
            "legendFormat": "{{cache_level}}"
          }
        ],
        "type": "gauge"
      }
    ]
  }
}
```

### 6. Health Checks

#### 6.1 Comprehensive Health Check

```go
// health/checker.go
package health

import (
    "context"
    "sync"
)

type HealthChecker struct {
    checks   map[string]Check
    mu       sync.RWMutex
}

type Check func(ctx context.Context) error

type HealthStatus struct {
    Status    string                 `json:"status"`
    Timestamp time.Time              `json:"timestamp"`
    Checks    map[string]CheckResult `json:"checks"`
}

type CheckResult struct {
    Status   string        `json:"status"`
    Message  string        `json:"message,omitempty"`
    Duration time.Duration `json:"duration"`
}

func (hc *HealthChecker) CheckHealth(ctx context.Context) HealthStatus {
    hc.mu.RLock()
    defer hc.mu.RUnlock()
    
    status := HealthStatus{
        Status:    "healthy",
        Timestamp: time.Now(),
        Checks:    make(map[string]CheckResult),
    }
    
    var wg sync.WaitGroup
    var mu sync.Mutex
    
    for name, check := range hc.checks {
        wg.Add(1)
        go func(n string, c Check) {
            defer wg.Done()
            
            start := time.Now()
            err := c(ctx)
            duration := time.Since(start)
            
            result := CheckResult{
                Status:   "healthy",
                Duration: duration,
            }
            
            if err != nil {
                result.Status = "unhealthy"
                result.Message = err.Error()
                mu.Lock()
                status.Status = "unhealthy"
                mu.Unlock()
            }
            
            mu.Lock()
            status.Checks[n] = result
            mu.Unlock()
        }(name, check)
    }
    
    wg.Wait()
    return status
}

// 기본 헬스 체크들
func DefaultChecks(deps Dependencies) map[string]Check {
    return map[string]Check{
        "database": func(ctx context.Context) error {
            return deps.DB.PingContext(ctx)
        },
        "redis": func(ctx context.Context) error {
            return deps.Redis.Ping(ctx).Err()
        },
        "docker": func(ctx context.Context) error {
            _, err := deps.Docker.Ping(ctx)
            return err
        },
        "disk_space": func(ctx context.Context) error {
            usage, err := disk.Usage("/")
            if err != nil {
                return err
            }
            if usage.UsedPercent > 90 {
                return fmt.Errorf("disk usage too high: %.2f%%", usage.UsedPercent)
            }
            return nil
        },
        "memory": func(ctx context.Context) error {
            v, err := mem.VirtualMemory()
            if err != nil {
                return err
            }
            if v.UsedPercent > 90 {
                return fmt.Errorf("memory usage too high: %.2f%%", v.UsedPercent)
            }
            return nil
        },
    }
}
```

### 7. Configuration

```yaml
# monitoring.yaml
monitoring:
  metrics:
    enabled: true
    port: 9090
    path: /metrics
    collection_interval: 10s
    
  logging:
    level: info
    format: json
    outputs:
      - stdout
      - /var/log/aicli/app.log
    sampling:
      initial: 100
      thereafter: 100
    fluent:
      enabled: true
      host: localhost
      port: 24224
      tag: aicli.logs
      
  tracing:
    enabled: true
    service_name: aicli-web
    jaeger_url: http://localhost:14268/api/traces
    sample_rate: 0.1
    max_spans: 1000
    batch_timeout: 5s
    
  alerting:
    enabled: true
    channels:
      - type: slack
        webhook_url: ${SLACK_WEBHOOK_URL}
        channel: "#alerts"
      - type: email
        smtp:
          host: smtp.gmail.com
          port: 587
        from: alerts@aicli.com
        to:
          - ops@aicli.com
      - type: pagerduty
        routing_key: ${PAGERDUTY_ROUTING_KEY}
        
  health:
    enabled: true
    port: 8080
    path: /health
    interval: 30s
    timeout: 5s
```

## Alert Rules

### Critical Alerts
1. **Service Down**: 서비스 응답 없음 (1분 이상)
2. **High Error Rate**: 에러율 > 5% (5분 동안)
3. **Database Down**: DB 연결 실패
4. **Disk Full**: 디스크 사용률 > 95%

### Warning Alerts
1. **High Response Time**: P95 > 1초 (10분 동안)
2. **High Memory Usage**: 메모리 사용률 > 80%
3. **High CPU Usage**: CPU 사용률 > 80% (5분 동안)
4. **Low Cache Hit Rate**: 캐시 히트율 < 80%

## SLI/SLO Definitions

### Service Level Indicators (SLI)
1. **Availability**: 성공한 요청 / 전체 요청
2. **Latency**: P95 응답 시간
3. **Error Rate**: 에러 응답 / 전체 응답
4. **Throughput**: 초당 처리 요청 수

### Service Level Objectives (SLO)
1. **Availability SLO**: 99.9% (월간)
2. **Latency SLO**: P95 < 100ms
3. **Error Rate SLO**: < 0.1%
4. **Throughput SLO**: > 1000 RPS

## Implementation Timeline

### Week 1: Foundation
- Prometheus 및 Grafana 설정
- 기본 메트릭 수집 구현
- 구조화된 로깅 시스템

### Week 2: Advanced Monitoring
- 분산 추적 구현
- 커스텀 메트릭 추가
- 알림 시스템 구축

### Week 3: Optimization
- 대시보드 최적화
- 알림 규칙 튜닝
- 문서화 및 교육