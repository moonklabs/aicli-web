package pty_streaming

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/flow"
	"github.com/aicli/aicli-web/internal/pty"
	"github.com/aicli/aicli-web/internal/terminal"
	"github.com/aicli/aicli-web/internal/websocket"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	gorilla "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "integration-test")

// IntegrationTestFramework provides comprehensive testing framework for PTY streaming
type IntegrationTestFramework struct {
	testServer      *TestServer
	dockerManager   *TestDockerManager
	clients         map[string]*TestClient
	metrics         *TestMetrics
	config          *TestConfig
	cleanup         []func() error
	mutex           sync.RWMutex
}

// TestServer represents a test server instance
type TestServer struct {
	server          *http.Server
	ptyManager      *pty.SessionManager
	wsManager       *websocket.StreamManager
	dockerPTY       *docker.DockerPTYIntegration
	flowController  *flow.FlowController
	terminalManager *terminal.SnapshotManager
	port            int
	running         bool
	mutex           sync.RWMutex
}

// TestClient represents a test WebSocket client
type TestClient struct {
	clientID     string
	wsConn       *gorilla.Conn
	sessionID    string
	containerID  string
	connected    bool
	lastMessage  time.Time
	messageCount int64
	errorCount   int64
	mutex        sync.RWMutex
}

// TestConfig contains test configuration
type TestConfig struct {
	ServerPort        int
	ContainerImages   []string
	MaxSessions       int
	TestDuration      time.Duration
	MessageRate       int
	EnableMetrics     bool
	LogLevel          string
	DockerHost        string
}

// TestDockerManager manages Docker containers for testing
type TestDockerManager struct {
	client     *client.Client
	containers map[string]*TestContainer
	mutex      sync.RWMutex
}

// TestContainer represents a test container
type TestContainer struct {
	ID        string
	Image     string
	Name      string
	CreatedAt time.Time
	Status    string
}

// NewIntegrationTestFramework creates a new integration test framework
func NewIntegrationTestFramework(config *TestConfig) (*IntegrationTestFramework, error) {
	if config == nil {
		config = &TestConfig{
			ServerPort:      8080,
			ContainerImages: []string{"alpine:latest"},
			MaxSessions:     100,
			TestDuration:    5 * time.Minute,
			MessageRate:     10,
			EnableMetrics:   true,
			LogLevel:        "info",
			DockerHost:      "unix:///var/run/docker.sock",
		}
	}

	framework := &IntegrationTestFramework{
		clients: make(map[string]*TestClient),
		config:  config,
		cleanup: make([]func() error, 0),
	}

	// 테스트 서버 설정
	if err := framework.setupTestServer(); err != nil {
		return nil, fmt.Errorf("failed to setup test server: %w", err)
	}

	// Docker 테스트 환경 설정
	if err := framework.setupDockerEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to setup docker environment: %w", err)
	}

	// 메트릭 수집 설정
	if config.EnableMetrics {
		framework.setupMetrics()
	}

	return framework, nil
}

// setupTestServer initializes the test server
func (itf *IntegrationTestFramework) setupTestServer() error {
	// PTY 매니저 초기화
	ptyConfig := &pty.SessionConfig{
		MaxSessions:     itf.config.MaxSessions,
		CleanupInterval: 30 * time.Second,
	}
	ptyManager := pty.NewSessionManager(ptyConfig)

	// WebSocket 매니저 초기화
	wsConfig := &websocket.StreamConfig{
		BufferSize:        4096,
		MaxConnections:    itf.config.MaxSessions,
		PingInterval:      30 * time.Second,
		EnableCompression: true,
	}
	wsManager := websocket.NewStreamManager(wsConfig)

	// Docker 클라이언트 생성
	dockerClient, err := client.NewClientWithOpts(
		client.WithHost(itf.config.DockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}

	// Docker PTY 통합 초기화
	dockerPTYConfig := &docker.DockerPTYConfig{
		MaxSessions:         itf.config.MaxSessions,
		MonitorInterval:     5 * time.Second,
		ReconnectTimeout:    30 * time.Second,
		HealthCheckInterval: 5 * time.Second,
	}
	dockerPTY := docker.NewDockerPTYIntegration(dockerClient, dockerPTYConfig)

	// 플로우 제어 초기화
	flowConfig := flow.DefaultFlowControlConfig()
	flowController, err := flow.NewFlowController(flowConfig)
	if err != nil {
		return fmt.Errorf("failed to create flow controller: %w", err)
	}

	// 터미널 스냅샷 매니저 초기화
	terminalManager := terminal.NewSnapshotManager(&terminal.SnapshotConfig{
		MaxSnapshots:     100,
		CompressionLevel: 6,
	})

	itf.testServer = &TestServer{
		ptyManager:      ptyManager,
		wsManager:       wsManager,
		dockerPTY:       dockerPTY,
		flowController:  flowController,
		terminalManager: terminalManager,
		port:            itf.config.ServerPort,
	}

	return nil
}

// setupDockerEnvironment initializes Docker test environment
func (itf *IntegrationTestFramework) setupDockerEnvironment() error {
	dockerClient, err := client.NewClientWithOpts(
		client.WithHost(itf.config.DockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}

	itf.dockerManager = &TestDockerManager{
		client:     dockerClient,
		containers: make(map[string]*TestContainer),
	}

	// 테스트 이미지 풀
	for _, image := range itf.config.ContainerImages {
		if err := itf.pullTestImage(image); err != nil {
			log.Warnf("Failed to pull test image %s: %v", image, err)
		}
	}

	return nil
}

// setupMetrics initializes metrics collection
func (itf *IntegrationTestFramework) setupMetrics() {
	itf.metrics = NewTestMetrics()
	itf.metrics.Start()
}

// Start starts the test framework
func (itf *IntegrationTestFramework) Start() error {
	// 서버 시작
	if err := itf.startTestServer(); err != nil {
		return fmt.Errorf("failed to start test server: %w", err)
	}

	// Docker PTY 시작
	itf.testServer.dockerPTY.Start()

	// WebSocket 매니저 시작 (실제 구현에서는 Start 메서드가 없으므로 주석 처리)

	return nil
}

// Stop stops the test framework
func (itf *IntegrationTestFramework) Stop() error {
	// 클라이언트 종료
	itf.mutex.Lock()
	for _, client := range itf.clients {
		if client.wsConn != nil {
			client.wsConn.Close()
		}
	}
	itf.mutex.Unlock()

	// 서버 종료
	if itf.testServer != nil {
		if itf.testServer.dockerPTY != nil {
			itf.testServer.dockerPTY.Stop()
		}
		// WebSocket 매니저 정지 (실제 구현에서는 Stop 메서드가 없으므로 주석 처리)
		// if itf.testServer.wsManager != nil {
		//		itf.testServer.wsManager.Stop()
		// }
		if itf.testServer.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			itf.testServer.server.Shutdown(ctx)
		}
	}

	// 메트릭 종료
	if itf.metrics != nil {
		itf.metrics.Stop()
	}

	return nil
}

// Cleanup performs cleanup operations
func (itf *IntegrationTestFramework) Cleanup() error {
	// 역순으로 정리 작업 수행
	for i := len(itf.cleanup) - 1; i >= 0; i-- {
		if err := itf.cleanup[i](); err != nil {
			log.Errorf("Cleanup error: %v", err)
		}
	}

	// 모든 컨테이너 제거
	if itf.dockerManager != nil {
		itf.dockerManager.RemoveAllContainers()
	}

	// 프레임워크 종료
	return itf.Stop()
}

// CreateTestContainer creates a test container
func (itf *IntegrationTestFramework) CreateTestContainer(image string) (string, error) {
	return itf.dockerManager.CreateContainer(image)
}

// RemoveContainer removes a container
func (itf *IntegrationTestFramework) RemoveContainer(containerID string) error {
	return itf.dockerManager.RemoveContainer(containerID)
}

// CreatePTYSession creates a PTY session
func (itf *IntegrationTestFramework) CreatePTYSession(containerID string) (string, error) {
	config := &docker.PTYSessionConfig{
		Shell:        "/bin/sh",
		Term:         "xterm-256color",
		Rows:         24,
		Cols:         80,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	ctx := context.Background()
	session, err := itf.testServer.dockerPTY.ConnectContainer(ctx, containerID, config)
	if err != nil {
		return "", err
	}

	return session.SessionID, nil
}

// ClosePTYSession closes a PTY session
func (itf *IntegrationTestFramework) ClosePTYSession(sessionID string) error {
	return itf.testServer.dockerPTY.DisconnectContainer(sessionID)
}

// ConnectWebSocket creates a WebSocket connection
func (itf *IntegrationTestFramework) ConnectWebSocket(sessionID string) (*TestClient, error) {
	url := fmt.Sprintf("ws://localhost:%d/ws/%s", itf.config.ServerPort, sessionID)

	conn, _, err := gorilla.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect websocket: %w", err)
	}

	client := &TestClient{
		clientID:    fmt.Sprintf("test-client-%d", time.Now().UnixNano()),
		wsConn:      conn,
		sessionID:   sessionID,
		connected:   true,
		lastMessage: time.Now(),
	}

	itf.mutex.Lock()
	itf.clients[client.clientID] = client
	itf.mutex.Unlock()

	return client, nil
}

// MeasureLatencies measures message latencies
func (itf *IntegrationTestFramework) MeasureLatencies(clients []*TestClient, messageCount int) []time.Duration {
	latencies := make([]time.Duration, 0, len(clients)*messageCount)

	for _, client := range clients {
		for i := 0; i < messageCount; i++ {
			start := time.Now()
			cmd := fmt.Sprintf("echo 'latency test %d'", i)

			if err := client.SendCommand(cmd); err != nil {
				continue
			}

			if _, err := client.WaitForOutput(fmt.Sprintf("latency test %d", i), 5*time.Second); err != nil {
				continue
			}

			latencies = append(latencies, time.Since(start))
		}
	}

	return latencies
}

// GetCurrentMemoryUsage returns current memory usage
func (itf *IntegrationTestFramework) GetCurrentMemoryUsage() int64 {
	if itf.metrics != nil {
		return itf.metrics.GetMemoryUsage()
	}
	return 0
}

// StartResourceMonitoring starts resource monitoring for a container
func (itf *IntegrationTestFramework) StartResourceMonitoring(containerID string) *ResourceMonitor {
	return NewResourceMonitor(itf.dockerManager.client, containerID)
}

// startTestServer starts the HTTP test server
func (itf *IntegrationTestFramework) startTestServer() error {
	mux := http.NewServeMux()

	// WebSocket 엔드포인트
	mux.HandleFunc("/ws/", itf.handleWebSocket)

	// 헬스 체크 엔드포인트
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	itf.testServer.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", itf.testServer.port),
		Handler: mux,
	}

	go func() {
		if err := itf.testServer.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Errorf("Test server error: %v", err)
		}
	}()

	// 서버 시작 대기
	time.Sleep(1 * time.Second)
	itf.testServer.running = true

	return nil
}

// handleWebSocket handles WebSocket connections
func (itf *IntegrationTestFramework) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := gorilla.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Failed to upgrade websocket: %v", err)
		return
	}

	// WebSocket 연결 처리
	go itf.handleWebSocketConnection(conn)
}

// handleWebSocketConnection handles individual WebSocket connections
func (itf *IntegrationTestFramework) handleWebSocketConnection(conn *gorilla.Conn) {
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if gorilla.IsUnexpectedCloseError(err, gorilla.CloseGoingAway, gorilla.CloseAbnormalClosure) {
				log.Errorf("WebSocket error: %v", err)
			}
			break
		}

		// 에코 백
		if err := conn.WriteMessage(messageType, message); err != nil {
			log.Errorf("Failed to write message: %v", err)
			break
		}
	}
}

// pullTestImage pulls a Docker image for testing
func (itf *IntegrationTestFramework) pullTestImage(image string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	_, err := itf.dockerManager.client.ImagePull(ctx, image, types.ImagePullOptions{})
	return err
}

// getRandomPort returns a random available port
func getRandomPort() int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 8080
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}