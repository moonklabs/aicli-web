package storage

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionState 연결 상태
type ConnectionState int

const (
	// ConnectionStateIdle 유휴 상태
	ConnectionStateIdle ConnectionState = iota
	
	// ConnectionStateActive 활성 상태
	ConnectionStateActive
	
	// ConnectionStateClosed 닫힌 상태
	ConnectionStateClosed
	
	// ConnectionStateError 에러 상태
	ConnectionStateError
)

// String 연결 상태 문자열 반환
func (cs ConnectionState) String() string {
	switch cs {
	case ConnectionStateIdle:
		return "idle"
	case ConnectionStateActive:
		return "active"
	case ConnectionStateClosed:
		return "closed"
	case ConnectionStateError:
		return "error"
	default:
		return "unknown"
	}
}

// Connection 연결 인터페이스
type Connection interface {
	// ID 연결 고유 ID
	ID() string
	
	// State 현재 상태
	State() ConnectionState
	
	// LastUsed 마지막 사용 시간
	LastUsed() time.Time
	
	// CreatedAt 생성 시간
	CreatedAt() time.Time
	
	// Close 연결 닫기
	Close() error
	
	// Ping 연결 상태 확인
	Ping(ctx context.Context) error
	
	// IsExpired 만료 여부 확인
	IsExpired(maxLifetime time.Duration) bool
	
	// MarkUsed 사용 표시
	MarkUsed()
}

// BaseConnection 기본 연결 구현
type BaseConnection struct {
	id        string
	state     int32 // atomic으로 관리
	createdAt time.Time
	lastUsed  int64 // atomic으로 관리
	mu        sync.RWMutex
}

// NewBaseConnection 기본 연결 생성
func NewBaseConnection(id string) *BaseConnection {
	now := time.Now()
	return &BaseConnection{
		id:        id,
		state:     int32(ConnectionStateIdle),
		createdAt: now,
		lastUsed:  now.UnixNano(),
	}
}

// ID 연결 ID 반환
func (c *BaseConnection) ID() string {
	return c.id
}

// State 현재 상태 반환
func (c *BaseConnection) State() ConnectionState {
	return ConnectionState(atomic.LoadInt32(&c.state))
}

// setState 상태 설정
func (c *BaseConnection) setState(state ConnectionState) {
	atomic.StoreInt32(&c.state, int32(state))
}

// LastUsed 마지막 사용 시간 반환
func (c *BaseConnection) LastUsed() time.Time {
	nanos := atomic.LoadInt64(&c.lastUsed)
	return time.Unix(0, nanos)
}

// CreatedAt 생성 시간 반환
func (c *BaseConnection) CreatedAt() time.Time {
	return c.createdAt
}

// Close 연결 닫기
func (c *BaseConnection) Close() error {
	c.setState(ConnectionStateClosed)
	return nil
}

// Ping 연결 상태 확인 (기본 구현)
func (c *BaseConnection) Ping(ctx context.Context) error {
	if c.State() == ConnectionStateClosed {
		return ErrStorageClosed
	}
	return nil
}

// IsExpired 만료 여부 확인
func (c *BaseConnection) IsExpired(maxLifetime time.Duration) bool {
	if maxLifetime <= 0 {
		return false
	}
	return time.Since(c.createdAt) > maxLifetime
}

// MarkUsed 사용 표시
func (c *BaseConnection) MarkUsed() {
	atomic.StoreInt64(&c.lastUsed, time.Now().UnixNano())
}

// ConnectionPool 연결 풀 인터페이스
type ConnectionPool interface {
	// Get 연결 가져오기
	Get(ctx context.Context) (Connection, error)
	
	// Put 연결 반환
	Put(conn Connection) error
	
	// Close 풀 닫기
	Close() error
	
	// Stats 풀 통계 반환
	Stats() ConnectionPoolStats
	
	// HealthCheck 연결 상태 확인
	HealthCheck(ctx context.Context) error
}

// ConnectionPoolStats 연결 풀 통계
type ConnectionPoolStats struct {
	// TotalConnections 총 연결 수
	TotalConnections int
	
	// IdleConnections 유휴 연결 수
	IdleConnections int
	
	// ActiveConnections 활성 연결 수
	ActiveConnections int
	
	// WaitingRequests 대기 중인 요청 수
	WaitingRequests int
	
	// CreatedConnections 생성된 연결 총수
	CreatedConnections int64
	
	// ClosedConnections 닫힌 연결 총수
	ClosedConnections int64
}

// ConnectionPoolConfig 연결 풀 설정
type ConnectionPoolConfig struct {
	// MaxOpenConns 최대 열린 연결 수
	MaxOpenConns int
	
	// MaxIdleConns 최대 유휴 연결 수
	MaxIdleConns int
	
	// ConnMaxLifetime 연결 최대 생명 시간
	ConnMaxLifetime time.Duration
	
	// ConnMaxIdleTime 연결 최대 유휴 시간
	ConnMaxIdleTime time.Duration
	
	// HealthCheckInterval 헬스체크 간격
	HealthCheckInterval time.Duration
	
	// CleanupInterval 정리 간격
	CleanupInterval time.Duration
}

// DefaultConnectionPoolConfig 기본 연결 풀 설정
func DefaultConnectionPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MaxOpenConns:        10,
		MaxIdleConns:        5,
		ConnMaxLifetime:     time.Hour,
		ConnMaxIdleTime:     time.Minute * 10,
		HealthCheckInterval: time.Minute * 5,
		CleanupInterval:     time.Minute,
	}
}

// BaseConnectionPool 기본 연결 풀 구현
type BaseConnectionPool struct {
	config   ConnectionPoolConfig
	creator  func() (Connection, error)
	
	mu          sync.RWMutex
	connections []Connection
	waiters     []chan Connection
	
	stats struct {
		created int64
		closed  int64
	}
	
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	closed bool
}

// NewBaseConnectionPool 기본 연결 풀 생성
func NewBaseConnectionPool(config ConnectionPoolConfig, creator func() (Connection, error)) *BaseConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &BaseConnectionPool{
		config:      config,
		creator:     creator,
		connections: make([]Connection, 0, config.MaxIdleConns),
		waiters:     make([]chan Connection, 0),
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// 백그라운드 작업 시작
	pool.wg.Add(2)
	go pool.cleanupWorker()
	go pool.healthCheckWorker()
	
	return pool
}

// Get 연결 가져오기
func (p *BaseConnectionPool) Get(ctx context.Context) (Connection, error) {
	p.mu.Lock()
	
	// 사용 가능한 연결 찾기
	for i, conn := range p.connections {
		if conn.State() == ConnectionStateIdle {
			// 연결을 슬라이스에서 제거
			p.connections = append(p.connections[:i], p.connections[i+1:]...)
			conn.MarkUsed()
			p.mu.Unlock()
			return conn, nil
		}
	}
	
	// 새 연결 생성 가능한지 확인
	totalConns := len(p.connections)
	if totalConns < p.config.MaxOpenConns {
		p.mu.Unlock()
		
		// 새 연결 생성
		conn, err := p.creator()
		if err != nil {
			return nil, WrapError(err, "create_connection", "pool")
		}
		
		atomic.AddInt64(&p.stats.created, 1)
		conn.MarkUsed()
		return conn, nil
	}
	
	// 대기 채널 생성
	waiter := make(chan Connection, 1)
	p.waiters = append(p.waiters, waiter)
	p.mu.Unlock()
	
	// 연결 대기
	select {
	case conn := <-waiter:
		conn.MarkUsed()
		return conn, nil
	case <-ctx.Done():
		// 대기 취소 시 대기자 목록에서 제거
		p.mu.Lock()
		for i, w := range p.waiters {
			if w == waiter {
				p.waiters = append(p.waiters[:i], p.waiters[i+1:]...)
				break
			}
		}
		p.mu.Unlock()
		close(waiter)
		return nil, ctx.Err()
	}
}

// Put 연결 반환
func (p *BaseConnectionPool) Put(conn Connection) error {
	if conn == nil {
		return nil
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// 풀이 닫혔거나 연결이 닫힌 경우
	if p.closed || conn.State() == ConnectionStateClosed || conn.State() == ConnectionStateError {
		conn.Close()
		atomic.AddInt64(&p.stats.closed, 1)
		return nil
	}
	
	// 연결이 만료된 경우
	if conn.IsExpired(p.config.ConnMaxLifetime) {
		conn.Close()
		atomic.AddInt64(&p.stats.closed, 1)
		return nil
	}
	
	// 대기자가 있는 경우
	if len(p.waiters) > 0 {
		waiter := p.waiters[0]
		p.waiters = p.waiters[1:]
		
		select {
		case waiter <- conn:
			// 대기자에게 전달 성공
		default:
			// 대기자 채널이 닫힌 경우 연결을 풀에 반환
			if len(p.connections) < p.config.MaxIdleConns {
				p.connections = append(p.connections, conn)
			} else {
				conn.Close()
				atomic.AddInt64(&p.stats.closed, 1)
			}
		}
		return nil
	}
	
	// 유휴 연결 풀에 추가
	if len(p.connections) < p.config.MaxIdleConns {
		p.connections = append(p.connections, conn)
	} else {
		conn.Close()
		atomic.AddInt64(&p.stats.closed, 1)
	}
	
	return nil
}

// Close 풀 닫기
func (p *BaseConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.closed {
		return nil
	}
	
	p.closed = true
	p.cancel()
	
	// 모든 연결 닫기
	for _, conn := range p.connections {
		conn.Close()
	}
	p.connections = nil
	
	// 모든 대기자에게 닫힘 신호
	for _, waiter := range p.waiters {
		close(waiter)
	}
	p.waiters = nil
	
	// 백그라운드 작업 종료 대기
	p.wg.Wait()
	
	return nil
}

// Stats 풀 통계 반환
func (p *BaseConnectionPool) Stats() ConnectionPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	idleCount := 0
	for _, conn := range p.connections {
		if conn.State() == ConnectionStateIdle {
			idleCount++
		}
	}
	
	return ConnectionPoolStats{
		TotalConnections:   len(p.connections),
		IdleConnections:    idleCount,
		ActiveConnections:  len(p.connections) - idleCount,
		WaitingRequests:    len(p.waiters),
		CreatedConnections: atomic.LoadInt64(&p.stats.created),
		ClosedConnections:  atomic.LoadInt64(&p.stats.closed),
	}
}

// HealthCheck 연결 상태 확인
func (p *BaseConnectionPool) HealthCheck(ctx context.Context) error {
	p.mu.RLock()
	connections := make([]Connection, len(p.connections))
	copy(connections, p.connections)
	p.mu.RUnlock()
	
	var firstErr error
	for _, conn := range connections {
		if err := conn.Ping(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	
	return firstErr
}

// cleanupWorker 정리 작업자
func (p *BaseConnectionPool) cleanupWorker() {
	defer p.wg.Done()
	
	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.cleanup()
		case <-p.ctx.Done():
			return
		}
	}
}

// cleanup 만료된 연결 정리
func (p *BaseConnectionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	var activeConns []Connection
	now := time.Now()
	
	for _, conn := range p.connections {
		// 만료된 연결이나 오래 유휴 상태인 연결 제거
		if conn.IsExpired(p.config.ConnMaxLifetime) ||
		   now.Sub(conn.LastUsed()) > p.config.ConnMaxIdleTime {
			conn.Close()
			atomic.AddInt64(&p.stats.closed, 1)
		} else {
			activeConns = append(activeConns, conn)
		}
	}
	
	p.connections = activeConns
}

// healthCheckWorker 헬스체크 작업자
func (p *BaseConnectionPool) healthCheckWorker() {
	defer p.wg.Done()
	
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.performHealthCheck()
		case <-p.ctx.Done():
			return
		}
	}
}

// performHealthCheck 헬스체크 수행
func (p *BaseConnectionPool) performHealthCheck() {
	ctx, cancel := context.WithTimeout(p.ctx, time.Second*10)
	defer cancel()
	
	p.mu.RLock()
	connections := make([]Connection, len(p.connections))
	copy(connections, p.connections)
	p.mu.RUnlock()
	
	for _, conn := range connections {
		if err := conn.Ping(ctx); err != nil {
			// 헬스체크 실패한 연결 제거
			p.mu.Lock()
			for i, c := range p.connections {
				if c == conn {
					p.connections = append(p.connections[:i], p.connections[i+1:]...)
					break
				}
			}
			p.mu.Unlock()
			
			conn.Close()
			atomic.AddInt64(&p.stats.closed, 1)
		}
	}
}