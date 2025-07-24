package boltdb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	
	"github.com/aicli/aicli-web/internal/models"
)

// BatchProcessor 배치 처리기
type BatchProcessor struct {
	storage      *Storage
	logger       *zap.Logger
	batchSize    int
	flushTimeout time.Duration
	
	// 배치 큐
	writeQueue  chan BatchOperation
	closeOnce   sync.Once
	closed      bool
	mu          sync.RWMutex
}

// BatchOperation 배치 연산
type BatchOperation struct {
	Type      BatchOpType
	Bucket    string
	Key       string
	Value     []byte
	IndexOps  []IndexUpdate
	Callback  func(error)
}

// BatchOpType 배치 연산 타입
type BatchOpType string

const (
	BatchOpPut    BatchOpType = "put"
	BatchOpDelete BatchOpType = "delete"
	BatchOpUpdate BatchOpType = "update"
)

// BatchConfig 배치 설정
type BatchConfig struct {
	BatchSize     int
	FlushTimeout  time.Duration
	QueueSize     int
	Workers       int
	Logger        *zap.Logger
}

// DefaultBatchConfig 기본 배치 설정
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		BatchSize:    1000,      // 1000개 단위로 배치 처리
		FlushTimeout: time.Second, // 1초마다 강제 플러시
		QueueSize:    10000,     // 큐 크기
		Workers:      2,         // 워커 수
		Logger:       zap.NewNop(),
	}
}

// NewBatchProcessor 새 배치 처리기 생성
func NewBatchProcessor(storage *Storage, config BatchConfig) *BatchProcessor {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	
	bp := &BatchProcessor{
		storage:      storage,
		logger:       config.Logger,
		batchSize:    config.BatchSize,
		flushTimeout: config.FlushTimeout,
		writeQueue:   make(chan BatchOperation, config.QueueSize),
	}
	
	// 배치 처리 워커 시작
	for i := 0; i < config.Workers; i++ {
		go bp.processBatch()
	}
	
	return bp
}

// processBatch 배치 처리 워커
func (bp *BatchProcessor) processBatch() {
	var batch []BatchOperation
	flushTimer := time.NewTicker(bp.flushTimeout)
	defer flushTimer.Stop()
	
	for {
		select {
		case op, ok := <-bp.writeQueue:
			if !ok {
				// 채널이 닫힌 경우, 남은 배치 처리 후 종료
				if len(batch) > 0 {
					bp.executeBatch(batch)
				}
				return
			}
			
			batch = append(batch, op)
			
			// 배치 크기 도달 시 즉시 처리
			if len(batch) >= bp.batchSize {
				bp.executeBatch(batch)
				batch = nil
			}
			
		case <-flushTimer.C:
			// 타임아웃 시 현재 배치 처리
			if len(batch) > 0 {
				bp.executeBatch(batch)
				batch = nil
			}
		}
	}
}

// executeBatch 배치 실행
func (bp *BatchProcessor) executeBatch(batch []BatchOperation) {
	if len(batch) == 0 {
		return
	}
	
	start := time.Now()
	
	err := bp.storage.db.Update(func(tx *bbolt.Tx) error {
		for _, op := range batch {
			if err := bp.executeOperation(tx, op); err != nil {
				bp.logger.Error("배치 연산 실행 실패",
					zap.String("type", string(op.Type)),
					zap.String("bucket", op.Bucket),
					zap.String("key", op.Key),
					zap.Error(err),
				)
				// 개별 연산 실패 시 콜백 호출
				if op.Callback != nil {
					go op.Callback(err)
				}
				continue
			}
			
			// 성공 시 콜백 호출
			if op.Callback != nil {
				go op.Callback(nil)
			}
		}
		return nil
	})
	
	duration := time.Since(start)
	
	if err != nil {
		bp.logger.Error("배치 트랜잭션 실패",
			zap.Int("batch_size", len(batch)),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
	} else {
		bp.logger.Debug("배치 처리 완료",
			zap.Int("batch_size", len(batch)),
			zap.Duration("duration", duration),
		)
	}
}

// executeOperation 개별 연산 실행
func (bp *BatchProcessor) executeOperation(tx *bbolt.Tx, op BatchOperation) error {
	bucket := tx.Bucket([]byte(op.Bucket))
	if bucket == nil {
		return fmt.Errorf("버킷 '%s'이 존재하지 않습니다", op.Bucket)
	}
	
	switch op.Type {
	case BatchOpPut:
		if err := bucket.Put([]byte(op.Key), op.Value); err != nil {
			return fmt.Errorf("PUT 연산 실패: %w", err)
		}
		
	case BatchOpDelete:
		if err := bucket.Delete([]byte(op.Key)); err != nil {
			return fmt.Errorf("DELETE 연산 실패: %w", err)
		}
		
	case BatchOpUpdate:
		// UPDATE는 PUT과 동일하지만, 기존 값 존재 확인
		if existing := bucket.Get([]byte(op.Key)); existing == nil {
			return fmt.Errorf("업데이트할 키 '%s'가 존재하지 않습니다", op.Key)
		}
		
		if err := bucket.Put([]byte(op.Key), op.Value); err != nil {
			return fmt.Errorf("UPDATE 연산 실패: %w", err)
		}
		
	default:
		return fmt.Errorf("알 수 없는 배치 연산 타입: %s", op.Type)
	}
	
	// 인덱스 업데이트
	if len(op.IndexOps) > 0 {
		indexMgr := newIndexManager(bp.storage)
		if err := indexMgr.BatchUpdate(tx, op.IndexOps); err != nil {
			return fmt.Errorf("인덱스 업데이트 실패: %w", err)
		}
	}
	
	return nil
}

// SubmitBatch 배치 연산 제출
func (bp *BatchProcessor) SubmitBatch(op BatchOperation) error {
	bp.mu.RLock()
	if bp.closed {
		bp.mu.RUnlock()
		return fmt.Errorf("배치 처리기가 종료되었습니다")
	}
	bp.mu.RUnlock()
	
	select {
	case bp.writeQueue <- op:
		return nil
	default:
		return fmt.Errorf("배치 큐가 가득 찼습니다")
	}
}

// SubmitBatchSync 동기식 배치 연산 제출
func (bp *BatchProcessor) SubmitBatchSync(op BatchOperation, timeout time.Duration) error {
	result := make(chan error, 1)
	
	op.Callback = func(err error) {
		result <- err
	}
	
	if err := bp.SubmitBatch(op); err != nil {
		return err
	}
	
	select {
	case err := <-result:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("배치 연산 타임아웃")
	}
}

// Close 배치 처리기 종료
func (bp *BatchProcessor) Close() error {
	bp.closeOnce.Do(func() {
		bp.mu.Lock()
		bp.closed = true
		bp.mu.Unlock()
		
		close(bp.writeQueue)
		
		bp.logger.Info("배치 처리기가 종료되었습니다")
	})
	
	return nil
}

// BatchWriter 배치 라이터
type BatchWriter struct {
	processor  *BatchProcessor
	bucketName string
	serializer interface{}
}

// NewBatchWriter 새 배치 라이터 생성
func (bp *BatchProcessor) NewBatchWriter(bucketName string, serializer interface{}) *BatchWriter {
	return &BatchWriter{
		processor:  bp,
		bucketName: bucketName,
		serializer: serializer,
	}
}

// WriteWorkspace 워크스페이스 배치 쓰기
func (bw *BatchWriter) WriteWorkspace(workspace *models.Workspace, indexOps []IndexUpdate) error {
	serializer, ok := bw.serializer.(*WorkspaceSerializer)
	if !ok {
		return fmt.Errorf("워크스페이스 직렬화기가 아닙니다")
	}
	
	data, err := serializer.Marshal(workspace)
	if err != nil {
		return fmt.Errorf("워크스페이스 직렬화 실패: %w", err)
	}
	
	op := BatchOperation{
		Type:     BatchOpPut,
		Bucket:   bw.bucketName,
		Key:      workspace.ID,
		Value:    data,
		IndexOps: indexOps,
	}
	
	return bw.processor.SubmitBatch(op)
}

// WriteProject 프로젝트 배치 쓰기
func (bw *BatchWriter) WriteProject(project *models.Project, indexOps []IndexUpdate) error {
	serializer, ok := bw.serializer.(*ProjectSerializer)
	if !ok {
		return fmt.Errorf("프로젝트 직렬화기가 아닙니다")
	}
	
	data, err := serializer.Marshal(project)
	if err != nil {
		return fmt.Errorf("프로젝트 직렬화 실패: %w", err)
	}
	
	op := BatchOperation{
		Type:     BatchOpPut,
		Bucket:   bw.bucketName,
		Key:      project.ID,
		Value:    data,
		IndexOps: indexOps,
	}
	
	return bw.processor.SubmitBatch(op)
}

// WriteSession 세션 배치 쓰기
func (bw *BatchWriter) WriteSession(session *models.Session, indexOps []IndexUpdate) error {
	serializer, ok := bw.serializer.(*SessionSerializer)
	if !ok {
		return fmt.Errorf("세션 직렬화기가 아닙니다")
	}
	
	data, err := serializer.Marshal(session)
	if err != nil {
		return fmt.Errorf("세션 직렬화 실패: %w", err)
	}
	
	op := BatchOperation{
		Type:     BatchOpPut,
		Bucket:   bw.bucketName,
		Key:      session.ID,
		Value:    data,
		IndexOps: indexOps,
	}
	
	return bw.processor.SubmitBatch(op)
}

// WriteTask 태스크 배치 쓰기
func (bw *BatchWriter) WriteTask(task *models.Task, indexOps []IndexUpdate) error {
	serializer, ok := bw.serializer.(*TaskSerializer)
	if !ok {
		return fmt.Errorf("태스크 직렬화기가 아닙니다")
	}
	
	data, err := serializer.Marshal(task)
	if err != nil {
		return fmt.Errorf("태스크 직렬화 실패: %w", err)
	}
	
	op := BatchOperation{
		Type:     BatchOpPut,
		Bucket:   bw.bucketName,
		Key:      task.ID,
		Value:    data,
		IndexOps: indexOps,
	}
	
	return bw.processor.SubmitBatch(op)
}

// Delete 배치 삭제
func (bw *BatchWriter) Delete(key string, indexOps []IndexUpdate) error {
	op := BatchOperation{
		Type:     BatchOpDelete,
		Bucket:   bw.bucketName,
		Key:      key,
		IndexOps: indexOps,
	}
	
	return bw.processor.SubmitBatch(op)
}

// BulkImporter 대량 데이터 가져오기
type BulkImporter struct {
	storage     *Storage
	logger      *zap.Logger
	batchSize   int
	parallelism int
}

// BulkImportConfig 대량 가져오기 설정
type BulkImportConfig struct {
	BatchSize   int
	Parallelism int
	Logger      *zap.Logger
}

// DefaultBulkImportConfig 기본 대량 가져오기 설정
func DefaultBulkImportConfig() BulkImportConfig {
	return BulkImportConfig{
		BatchSize:   5000,  // 5000개 단위로 처리
		Parallelism: 4,     // 4개 고루틴으로 병렬 처리
		Logger:      zap.NewNop(),
	}
}

// NewBulkImporter 새 대량 가져오기 생성
func NewBulkImporter(storage *Storage, config BulkImportConfig) *BulkImporter {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	
	return &BulkImporter{
		storage:     storage,
		logger:      config.Logger,
		batchSize:   config.BatchSize,
		parallelism: config.Parallelism,
	}
}

// ImportWorkspaces 워크스페이스 대량 가져오기
func (bi *BulkImporter) ImportWorkspaces(ctx context.Context, workspaces []*models.Workspace) error {
	total := len(workspaces)
	if total == 0 {
		return nil
	}
	
	bi.logger.Info("워크스페이스 대량 가져오기 시작",
		zap.Int("total_count", total),
		zap.Int("batch_size", bi.batchSize),
	)
	
	start := time.Now()
	
	// 배치로 나누어 처리
	for i := 0; i < total; i += bi.batchSize {
		end := i + bi.batchSize
		if end > total {
			end = total
		}
		
		batch := workspaces[i:end]
		
		err := bi.storage.db.Update(func(tx *bbolt.Tx) error {
			bucket := tx.Bucket([]byte(BucketWorkspaces))
			if bucket == nil {
				return fmt.Errorf("워크스페이스 버킷이 존재하지 않습니다")
			}
			
			serializer := &WorkspaceSerializer{}
			indexMgr := newIndexManager(bi.storage)
			
			for _, workspace := range batch {
				// 직렬화
				data, err := serializer.Marshal(workspace)
				if err != nil {
					bi.logger.Error("워크스페이스 직렬화 실패",
						zap.String("id", workspace.ID),
						zap.Error(err),
					)
					continue
				}
				
				// 저장
				if err := bucket.Put([]byte(workspace.ID), data); err != nil {
					bi.logger.Error("워크스페이스 저장 실패",
						zap.String("id", workspace.ID),
						zap.Error(err),
					)
					continue
				}
				
				// 인덱스 업데이트
				indexOps := []IndexUpdate{
					{
						Index:     IndexWorkspaceOwner,
						Operation: IndexOpAdd,
						Key:       workspace.OwnerID,
						Value:     workspace.ID,
					},
					{
						Index:     IndexWorkspaceName,
						Operation: IndexOpAdd,
						Key:       fmt.Sprintf("%s:%s", workspace.OwnerID, workspace.Name),
						Value:     workspace.ID,
					},
				}
				
				if err := indexMgr.BatchUpdate(tx, indexOps); err != nil {
					bi.logger.Error("워크스페이스 인덱스 업데이트 실패",
						zap.String("id", workspace.ID),
						zap.Error(err),
					)
				}
			}
			
			return nil
		})
		
		if err != nil {
			bi.logger.Error("워크스페이스 배치 처리 실패",
				zap.Int("batch_start", i),
				zap.Int("batch_end", end),
				zap.Error(err),
			)
			return err
		}
		
		bi.logger.Debug("워크스페이스 배치 처리 완료",
			zap.Int("batch_start", i),
			zap.Int("batch_end", end),
		)
		
		// 컨텍스트 취소 확인
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	
	duration := time.Since(start)
	
	bi.logger.Info("워크스페이스 대량 가져오기 완료",
		zap.Int("total_count", total),
		zap.Duration("duration", duration),
		zap.Float64("items_per_second", float64(total)/duration.Seconds()),
	)
	
	return nil
}

// GetBatchStats 배치 처리 통계
func (bp *BatchProcessor) GetBatchStats() BatchStats {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	
	return BatchStats{
		QueueLength: len(bp.writeQueue),
		BatchSize:   bp.batchSize,
		Closed:      bp.closed,
	}
}

// BatchStats 배치 통계
type BatchStats struct {
	QueueLength int  `json:"queue_length"`
	BatchSize   int  `json:"batch_size"`
	Closed      bool `json:"closed"`
}

// FlushAll 모든 대기 중인 배치 즉시 처리
func (bp *BatchProcessor) FlushAll(timeout time.Duration) error {
	start := time.Now()
	
	// 현재 큐 길이 확인
	initialLength := len(bp.writeQueue)
	
	// 타임아웃 설정
	deadline := start.Add(timeout)
	
	for time.Now().Before(deadline) {
		currentLength := len(bp.writeQueue)
		
		// 큐가 비었으면 완료
		if currentLength == 0 {
			bp.logger.Info("모든 배치 처리 완료",
				zap.Int("initial_queue_length", initialLength),
				zap.Duration("flush_duration", time.Since(start)),
			)
			return nil
		}
		
		// 짧은 시간 대기
		time.Sleep(10 * time.Millisecond)
	}
	
	// 타임아웃 시 에러 반환
	remainingLength := len(bp.writeQueue)
	return fmt.Errorf("배치 플러시 타임아웃: %d개 항목이 남아있음", remainingLength)
}