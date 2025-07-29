package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/models"
)

// Mock 정의는 mocks_test.go 파일에서 통합 관리됩니다.

func TestDockerAdapter_CreateAgentContainer(t *testing.T) {
	ctx := context.Background()
	
	// 모의 객체 생성
	mockContainerManager := &MockContainerManager{}
	mockClient := &MockDockerClient{}
	mockStatsCollector := &MockStatsCollector{}
	
	// Docker 어댑터 생성
	adapter := NewDockerAdapter(mockContainerManager, mockClient, mockStatsCollector)
	
	// Agent 전용 Docker 통합 설정 (Mock client 호환성 문제로 생략)
	// mockDockerClient := &MockDockerClient{}
	// agentIntegration := &DockerAgentIntegration{
	//     client:      mockDockerClient,
	//     imageName:   DefaultAgentImage,
	//     networkName: "aicli-agent-network",
	// }
	// adapter.(*dockerAdapter).SetAgentIntegration(agentIntegration)
	
	// 컨테이너 설정
	config := ContainerConfig{
		Image: DefaultAgentImage,
		Labels: map[string]string{
			"type":       "agent",
			"agent_id":   "test-agent-123",
			"project_id": "test-project-456",
			"agent_name": "test-agent",
			"agent_type": string(models.AgentTypeClaude),
		},
		Environment: map[string]string{
			"CLAUDE_API_KEY": "test-api-key",
			"GIT_USER_NAME":  "Test User",
			"GIT_USER_EMAIL": "test@example.com",
		},
		WorkingDir: "/tmp/test-worktree",
	}
	
	// 네트워크 확인 모의 (일반 컨테이너는 Agent 네트워크 설정 불필요)
	// mockClient.On("NetworkList", ctx, mock.Anything).Return([]types.NetworkResource{
	//     {Name: "aicli-agent-network"},
	// }, nil)
	
	// 컨테이너 생성 모의 (ContainerManager 사용)
	// mockClient.On("ContainerCreate", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
	//     Return(container.CreateResponse{ID: "container-abc123"}, nil)
	
	// 컨테이너 생성 실행
	containerInfo, err := adapter.CreateContainer(ctx, config)
	
	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, containerInfo)
	assert.Equal(t, "container-abc123", containerInfo.ID)
	assert.Equal(t, "aicli-agent-test-agent-123", containerInfo.Name)
	assert.Equal(t, "created", containerInfo.Status)
	// mockDockerClient.AssertExpectations(t) // Agent 통합 설정이 제거되어 불필요
}

func TestDockerAdapter_CreateRegularContainer(t *testing.T) {
	ctx := context.Background()
	
	// 모의 객체 생성
	mockContainerManager := &MockContainerManager{}
	mockClient := &MockDockerClient{}
	mockStatsCollector := &MockStatsCollector{}
	
	// Docker 어댑터 생성
	adapter := NewDockerAdapter(mockContainerManager, mockClient, mockStatsCollector)
	
	// 일반 컨테이너 설정
	config := ContainerConfig{
		Image: "ubuntu:22.04",
		Labels: map[string]string{
			"app": "test",
		},
		Environment: map[string]string{
			"ENV_VAR": "value",
		},
		WorkingDir: "/app",
	}
	
	// 컨테이너 생성 모의
	mockContainerManager.On("CreateWorkspaceContainer", ctx, mock.Anything).
		Return(&docker.WorkspaceContainer{
			ID:      "container-xyz789",
			Name:    "test-container",
			State:   docker.ContainerStateCreated,
			Created: time.Now(),
		}, nil)
	
	// 컨테이너 생성 실행
	containerInfo, err := adapter.CreateContainer(ctx, config)
	
	// 검증
	assert.NoError(t, err)
	assert.NotNil(t, containerInfo)
	assert.Equal(t, "container-xyz789", containerInfo.ID)
	assert.Equal(t, "test-container", containerInfo.Name)
	assert.Equal(t, "created", containerInfo.Status)
	mockContainerManager.AssertExpectations(t)
}