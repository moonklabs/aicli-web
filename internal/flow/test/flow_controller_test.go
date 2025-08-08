package flow_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/aicli/internal/flow"
)

func TestFlowController_NewFlowController(t *testing.T) {
	config := flow.DefaultFlowControlConfig()
	fc, err := flow.NewFlowController(config)
	
	require.NoError(t, err)
	assert.NotNil(t, fc)
}

func TestFlowController_RegisterConnection(t *testing.T) {
	fc, err := flow.NewFlowController(nil)
	require.NoError(t, err)
	
	ctx := context.Background()
	err = fc.Start(ctx)
	require.NoError(t, err)
	defer fc.Stop()
	
	// 연결 등록
	err = fc.RegisterConnection("conn1", "session1")
	assert.NoError(t, err)
	
	// 중복 등록 시도
	err = fc.RegisterConnection("conn1", "session1")
	assert.Error(t, err)
	
	// 상태 확인
	state, err := fc.GetConnectionState("conn1")
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, "conn1", state.ConnectionID)
	assert.Equal(t, "session1", state.SessionID)
}

func TestFlowController_ProcessMessage(t *testing.T) {
	fc, err := flow.NewFlowController(nil)
	require.NoError(t, err)
	
	ctx := context.Background()
	err = fc.Start(ctx)
	require.NoError(t, err)
	defer fc.Stop()
	
	// 연결 등록
	err = fc.RegisterConnection("conn1", "session1")
	require.NoError(t, err)
	
	// 메시지 처리
	data := []byte("test message")
	err = fc.ProcessMessage("conn1", data, flow.PriorityNormal)
	assert.NoError(t, err)
	
	// 통계 확인
	stats := fc.GetStatistics()
	assert.NotNil(t, stats)
	assert.Equal(t, uint64(1), stats.TotalMessages)
}

func TestFlowController_BackpressureDetection(t *testing.T) {
	config := flow.DefaultFlowControlConfig()
	config.Thresholds.BufferUtilizationLow = 0.3
	config.Thresholds.BufferUtilizationMedium = 0.5
	config.Thresholds.BufferUtilizationHigh = 0.7
	
	fc, err := flow.NewFlowController(config)
	require.NoError(t, err)
	
	ctx := context.Background()
	err = fc.Start(ctx)
	require.NoError(t, err)
	defer fc.Stop()
	
	// 연결 등록
	err = fc.RegisterConnection("conn1", "session1")
	require.NoError(t, err)
	
	// 백프레셔 감지
	level, err := fc.DetectBackpressure("conn1")
	assert.NoError(t, err)
	assert.Equal(t, flow.BackpressureNone, level)
	
	// 여러 메시지 처리하여 백프레셔 유발
	largeData := make([]byte, 1024*10) // 10KB
	for i := 0; i < 10; i++ {
		fc.ProcessMessage("conn1", largeData, flow.PriorityNormal)
	}
	
	// 백프레셔 레벨 확인
	level, err = fc.DetectBackpressure("conn1")
	assert.NoError(t, err)
	// 백프레셔가 감지되어야 함
	assert.True(t, level >= flow.BackpressureNone)
}

func TestFlowController_UnregisterConnection(t *testing.T) {
	fc, err := flow.NewFlowController(nil)
	require.NoError(t, err)
	
	ctx := context.Background()
	err = fc.Start(ctx)
	require.NoError(t, err)
	defer fc.Stop()
	
	// 연결 등록
	err = fc.RegisterConnection("conn1", "session1")
	require.NoError(t, err)
	
	// 연결 해제
	err = fc.UnregisterConnection("conn1")
	assert.NoError(t, err)
	
	// 상태 확인
	_, err = fc.GetConnectionState("conn1")
	assert.Error(t, err)
	
	// 없는 연결 해제 시도
	err = fc.UnregisterConnection("conn1")
	assert.Error(t, err)
}

func TestFlowController_GlobalMetrics(t *testing.T) {
	fc, err := flow.NewFlowController(nil)
	require.NoError(t, err)
	
	ctx := context.Background()
	err = fc.Start(ctx)
	require.NoError(t, err)
	defer fc.Stop()
	
	// 여러 연결 등록
	for i := 0; i < 5; i++ {
		connID := fmt.Sprintf("conn%d", i)
		sessionID := fmt.Sprintf("session%d", i)
		err = fc.RegisterConnection(connID, sessionID)
		require.NoError(t, err)
	}
	
	// 글로벌 메트릭 확인
	metrics := fc.GetGlobalMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int32(5), metrics.ActiveConnections)
	assert.Equal(t, int32(5), metrics.TotalConnections)
	
	// 일부 연결 해제
	for i := 0; i < 2; i++ {
		connID := fmt.Sprintf("conn%d", i)
		err = fc.UnregisterConnection(connID)
		require.NoError(t, err)
	}
	
	// 메트릭 재확인
	time.Sleep(100 * time.Millisecond) // 메트릭 업데이트 대기
	metrics = fc.GetGlobalMetrics()
	assert.Equal(t, int32(3), metrics.ActiveConnections)
	assert.Equal(t, int32(5), metrics.TotalConnections)
}

func TestFlowController_ConcurrentOperations(t *testing.T) {
	fc, err := flow.NewFlowController(nil)
	require.NoError(t, err)
	
	ctx := context.Background()
	err = fc.Start(ctx)
	require.NoError(t, err)
	defer fc.Stop()
	
	// 동시 연결 등록
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			connID := fmt.Sprintf("conn%d", id)
			sessionID := fmt.Sprintf("session%d", id)
			fc.RegisterConnection(connID, sessionID)
			
			// 메시지 처리
			for j := 0; j < 10; j++ {
				data := []byte(fmt.Sprintf("message %d-%d", id, j))
				fc.ProcessMessage(connID, data, flow.PriorityNormal)
			}
			
			done <- true
		}(i)
	}
	
	// 모든 고루틴 완료 대기
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// 통계 확인
	stats := fc.GetStatistics()
	assert.True(t, stats.TotalMessages > 0)
	
	metrics := fc.GetGlobalMetrics()
	assert.Equal(t, int32(10), metrics.ActiveConnections)
}