package websocket

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aicli/aicli-web/internal/storage"
)

// FileManager는 세션 파일 관리를 담당합니다
type FileManager struct {
	storage     storage.Storage
	uploadsPath string
	mutex       sync.RWMutex
	
	// 파일 캐시 및 메타데이터
	fileCache    map[string]*FileMetadata
	cacheMutex   sync.RWMutex
	
	// 업로드 진행 상태 추적
	uploadProgress map[string]*UploadProgress
	progressMutex  sync.RWMutex
	
	// 설정
	config FileManagerConfig
}

// FileMetadata는 파일 메타데이터입니다
type FileMetadata struct {
	FileID        string                 `json:"file_id"`
	OriginalName  string                 `json:"original_name"`
	StoredName    string                 `json:"stored_name"`
	SessionID     string                 `json:"session_id"`
	UploaderID    string                 `json:"uploader_id"`
	UploaderName  string                 `json:"uploader_name"`
	MimeType      string                 `json:"mime_type"`
	Size          int64                  `json:"size"`
	Checksum      string                 `json:"checksum"`
	UploadedAt    time.Time              `json:"uploaded_at"`
	LastAccessed  time.Time              `json:"last_accessed"`
	IsTemporary   bool                   `json:"is_temporary"`
	ExpiresAt     *time.Time             `json:"expires_at,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	DownloadCount int64                  `json:"download_count"`
	Status        FileStatus             `json:"status"`
}

// UploadProgress는 업로드 진행 상태입니다
type UploadProgress struct {
	FileID         string    `json:"file_id"`
	SessionID      string    `json:"session_id"`
	UserID         string    `json:"user_id"`
	FileName       string    `json:"file_name"`
	TotalSize      int64     `json:"total_size"`
	UploadedSize   int64     `json:"uploaded_size"`
	Progress       float64   `json:"progress"`
	Status         string    `json:"status"`
	StartTime      time.Time `json:"start_time"`
	LastUpdate     time.Time `json:"last_update"`
	EstimatedTime  time.Duration `json:"estimated_time"`
	Error          string    `json:"error,omitempty"`
}

// FileManagerConfig는 파일 매니저 설정입니다
type FileManagerConfig struct {
	UploadsPath       string            `json:"uploads_path"`
	MaxFileSize       int64             `json:"max_file_size"`
	MaxFilesPerSession int              `json:"max_files_per_session"`
	AllowedMimeTypes  []string          `json:"allowed_mime_types"`
	BlockedExtensions []string          `json:"blocked_extensions"`
	EnableVirusScanning bool            `json:"enable_virus_scanning"`
	EnableCompression bool              `json:"enable_compression"`
	CleanupInterval   time.Duration     `json:"cleanup_interval"`
	TempFileExpiry    time.Duration     `json:"temp_file_expiry"`
}

// FileStatus는 파일 상태입니다
type FileStatus int

const (
	FileStatusUploading FileStatus = iota
	FileStatusProcessing
	FileStatusReady
	FileStatusError
	FileStatusDeleted
	FileStatusExpired
)

// FileUploadRequest는 파일 업로드 요청입니다
type FileUploadRequest struct {
	SessionID   string            `json:"session_id"`
	FileName    string            `json:"file_name"`
	FileSize    int64             `json:"file_size"`
	MimeType    string            `json:"mime_type"`
	Checksum    string            `json:"checksum,omitempty"`
	IsTemporary bool              `json:"is_temporary,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FileUploadResponse는 파일 업로드 응답입니다
type FileUploadResponse struct {
	FileID      string `json:"file_id"`
	UploadURL   string `json:"upload_url"`
	Message     string `json:"message"`
	MaxChunkSize int   `json:"max_chunk_size"`
}

// FileDownloadRequest는 파일 다운로드 요청입니다
type FileDownloadRequest struct {
	FileID    string `json:"file_id"`
	SessionID string `json:"session_id"`
	Range     string `json:"range,omitempty"`
}

// FileListRequest는 파일 목록 요청입니다
type FileListRequest struct {
	SessionID    string   `json:"session_id"`
	FileTypes    []string `json:"file_types,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	SortBy       string   `json:"sort_by,omitempty"`
	SortOrder    string   `json:"sort_order,omitempty"`
	Limit        int      `json:"limit,omitempty"`
	Offset       int      `json:"offset,omitempty"`
}

// ChunkUploadRequest는 청크 업로드 요청입니다
type ChunkUploadRequest struct {
	FileID      string `json:"file_id"`
	ChunkIndex  int    `json:"chunk_index"`
	ChunkSize   int    `json:"chunk_size"`
	ChunkHash   string `json:"chunk_hash"`
	IsLastChunk bool   `json:"is_last_chunk"`
}

// DefaultFileManagerConfig는 기본 파일 매니저 설정을 반환합니다
func DefaultFileManagerConfig() FileManagerConfig {
	return FileManagerConfig{
		UploadsPath:        "./uploads",
		MaxFileSize:        100 * 1024 * 1024, // 100MB
		MaxFilesPerSession: 50,
		AllowedMimeTypes: []string{
			"text/plain",
			"text/markdown",
			"text/csv",
			"application/json",
			"application/pdf",
			"image/png",
			"image/jpeg",
			"image/gif",
			"application/zip",
			"application/x-tar",
			"application/gzip",
		},
		BlockedExtensions: []string{
			".exe", ".bat", ".cmd", ".com", ".scr",
			".vbs", ".vbe", ".js", ".jar", ".py",
		},
		EnableVirusScanning: false, // 실제 환경에서는 활성화
		EnableCompression:   true,
		CleanupInterval:     24 * time.Hour,
		TempFileExpiry:      7 * 24 * time.Hour, // 7일
	}
}

// NewFileManager는 새로운 파일 매니저를 생성합니다
func NewFileManager(storage storage.Storage, config FileManagerConfig) (*FileManager, error) {
	// 업로드 디렉토리 생성
	if err := os.MkdirAll(config.UploadsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create uploads directory: %w", err)
	}

	fm := &FileManager{
		storage:        storage,
		uploadsPath:    config.UploadsPath,
		fileCache:      make(map[string]*FileMetadata),
		uploadProgress: make(map[string]*UploadProgress),
		config:         config,
	}

	// 정리 작업 고루틴 시작
	go fm.cleanupRoutine()

	return fm, nil
}

// InitiateUpload는 파일 업로드를 시작합니다
func (fm *FileManager) InitiateUpload(request FileUploadRequest, userID, userName string) (*FileUploadResponse, error) {
	// 요청 유효성 검사
	if err := fm.validateUploadRequest(request); err != nil {
		return nil, fmt.Errorf("invalid upload request: %w", err)
	}

	// 세션별 파일 수 제한 확인
	if err := fm.checkSessionFileLimit(request.SessionID); err != nil {
		return nil, err
	}

	// 파일 ID 생성
	fileID := fm.generateFileID(request.FileName, userID)

	// 저장될 파일명 생성
	storedName := fm.generateStoredFileName(fileID, request.FileName)

	// 파일 메타데이터 생성
	metadata := &FileMetadata{
		FileID:       fileID,
		OriginalName: request.FileName,
		StoredName:   storedName,
		SessionID:    request.SessionID,
		UploaderID:   userID,
		UploaderName: userName,
		MimeType:     request.MimeType,
		Size:         request.FileSize,
		Checksum:     request.Checksum,
		UploadedAt:   time.Now(),
		LastAccessed: time.Now(),
		IsTemporary:  request.IsTemporary,
		Tags:         request.Tags,
		Metadata:     request.Metadata,
		Status:       FileStatusUploading,
	}

	// 임시 파일인 경우 만료 시간 설정
	if request.IsTemporary {
		expiry := time.Now().Add(fm.config.TempFileExpiry)
		metadata.ExpiresAt = &expiry
	}

	// 업로드 진행 상태 초기화
	progress := &UploadProgress{
		FileID:      fileID,
		SessionID:   request.SessionID,
		UserID:      userID,
		FileName:    request.FileName,
		TotalSize:   request.FileSize,
		Status:      "initiated",
		StartTime:   time.Now(),
		LastUpdate:  time.Now(),
	}

	// 캐시에 저장
	fm.cacheMutex.Lock()
	fm.fileCache[fileID] = metadata
	fm.cacheMutex.Unlock()

	fm.progressMutex.Lock()
	fm.uploadProgress[fileID] = progress
	fm.progressMutex.Unlock()

	// 스토리지에 메타데이터 저장
	if err := fm.saveFileMetadata(metadata); err != nil {
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}

	return &FileUploadResponse{
		FileID:       fileID,
		UploadURL:    fmt.Sprintf("/api/files/upload/%s", fileID),
		Message:      "파일 업로드가 초기화되었습니다",
		MaxChunkSize: 1024 * 1024, // 1MB chunks
	}, nil
}

// UploadChunk는 파일 청크를 업로드합니다
func (fm *FileManager) UploadChunk(fileID string, chunkIndex int, chunkData []byte, isLastChunk bool) error {
	// 업로드 진행 상태 확인
	fm.progressMutex.RLock()
	progress, exists := fm.uploadProgress[fileID]
	fm.progressMutex.RUnlock()

	if !exists {
		return fmt.Errorf("upload session not found: %s", fileID)
	}

	if progress.Status == "error" {
		return fmt.Errorf("upload session is in error state")
	}

	// 파일 메타데이터 확인
	fm.cacheMutex.RLock()
	metadata, exists := fm.fileCache[fileID]
	fm.cacheMutex.RUnlock()

	if !exists {
		return fmt.Errorf("file metadata not found: %s", fileID)
	}

	// 청크를 임시 파일에 저장
	tempChunkPath := filepath.Join(fm.uploadsPath, "temp", fmt.Sprintf("%s_chunk_%d", fileID, chunkIndex))
	if err := os.MkdirAll(filepath.Dir(tempChunkPath), 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	if err := os.WriteFile(tempChunkPath, chunkData, 0644); err != nil {
		return fmt.Errorf("failed to write chunk: %w", err)
	}

	// 진행 상태 업데이트
	fm.progressMutex.Lock()
	progress.UploadedSize += int64(len(chunkData))
	progress.Progress = float64(progress.UploadedSize) / float64(progress.TotalSize) * 100
	progress.LastUpdate = time.Now()
	
	elapsed := time.Since(progress.StartTime)
	if progress.UploadedSize > 0 {
		estimatedTotal := elapsed * time.Duration(progress.TotalSize) / time.Duration(progress.UploadedSize)
		progress.EstimatedTime = estimatedTotal - elapsed
	}
	fm.progressMutex.Unlock()

	// 마지막 청크인 경우 파일 조립
	if isLastChunk {
		if err := fm.assembleFile(fileID, metadata); err != nil {
			fm.markUploadError(fileID, err.Error())
			return fmt.Errorf("failed to assemble file: %w", err)
		}

		// 업로드 완료 처리
		fm.completeUpload(fileID)
	}

	return nil
}

// GetFile은 파일 정보를 조회합니다
func (fm *FileManager) GetFile(fileID, sessionID string) (*FileMetadata, error) {
	// 캐시에서 먼저 확인
	fm.cacheMutex.RLock()
	if metadata, exists := fm.fileCache[fileID]; exists {
		fm.cacheMutex.RUnlock()
		
		// 세션 접근 권한 확인
		if metadata.SessionID != sessionID {
			return nil, fmt.Errorf("access denied to file")
		}
		
		// 마지막 접근 시간 업데이트
		metadata.LastAccessed = time.Now()
		return metadata, nil
	}
	fm.cacheMutex.RUnlock()

	// 스토리지에서 조회
	metadata, err := fm.loadFileMetadata(fileID)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// 세션 접근 권한 확인
	if metadata.SessionID != sessionID {
		return nil, fmt.Errorf("access denied to file")
	}

	// 캐시에 저장
	fm.cacheMutex.Lock()
	fm.fileCache[fileID] = metadata
	fm.cacheMutex.Unlock()

	return metadata, nil
}

// DownloadFile은 파일을 다운로드합니다
func (fm *FileManager) DownloadFile(fileID, sessionID string) (io.ReadCloser, *FileMetadata, error) {
	// 파일 메타데이터 조회
	metadata, err := fm.GetFile(fileID, sessionID)
	if err != nil {
		return nil, nil, err
	}

	// 파일 상태 확인
	if metadata.Status != FileStatusReady {
		return nil, nil, fmt.Errorf("file is not ready for download")
	}

	// 파일 경로 구성
	filePath := filepath.Join(fm.uploadsPath, metadata.StoredName)

	// 파일 열기
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	// 다운로드 카운트 증가
	metadata.DownloadCount++
	metadata.LastAccessed = time.Now()

	// 메타데이터 업데이트
	fm.saveFileMetadata(metadata)

	return file, metadata, nil
}

// ListFiles는 세션의 파일 목록을 조회합니다
func (fm *FileManager) ListFiles(request FileListRequest) ([]*FileMetadata, error) {
	// 스토리지에서 세션 파일 목록 조회
	files, err := fm.loadSessionFiles(request.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session files: %w", err)
	}

	// 필터링 적용
	filteredFiles := fm.filterFiles(files, request)

	// 정렬 적용
	sortedFiles := fm.sortFiles(filteredFiles, request.SortBy, request.SortOrder)

	// 페이징 적용
	paginatedFiles := fm.paginateFiles(sortedFiles, request.Offset, request.Limit)

	return paginatedFiles, nil
}

// DeleteFile은 파일을 삭제합니다
func (fm *FileManager) DeleteFile(fileID, sessionID, userID string) error {
	// 파일 메타데이터 조회
	metadata, err := fm.GetFile(fileID, sessionID)
	if err != nil {
		return err
	}

	// 삭제 권한 확인 (업로더 또는 관리자만 삭제 가능)
	if metadata.UploaderID != userID {
		// 실제 구현에서는 세션 관리자 권한도 확인
		return fmt.Errorf("access denied: only uploader can delete file")
	}

	// 파일 시스템에서 삭제
	filePath := filepath.Join(fm.uploadsPath, metadata.StoredName)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// 메타데이터 상태 업데이트
	metadata.Status = FileStatusDeleted

	// 캐시에서 제거
	fm.cacheMutex.Lock()
	delete(fm.fileCache, fileID)
	fm.cacheMutex.Unlock()

	// 스토리지에서 삭제
	return fm.deleteFileMetadata(fileID)
}

// GetUploadProgress는 업로드 진행 상태를 조회합니다
func (fm *FileManager) GetUploadProgress(fileID string) (*UploadProgress, error) {
	fm.progressMutex.RLock()
	defer fm.progressMutex.RUnlock()

	progress, exists := fm.uploadProgress[fileID]
	if !exists {
		return nil, fmt.Errorf("upload progress not found")
	}

	// 복사본 반환
	progressCopy := *progress
	return &progressCopy, nil
}

// 내부 메서드들

func (fm *FileManager) validateUploadRequest(request FileUploadRequest) error {
	// 파일 크기 검사
	if request.FileSize > fm.config.MaxFileSize {
		return fmt.Errorf("file size exceeds limit: %d > %d", request.FileSize, fm.config.MaxFileSize)
	}

	if request.FileSize <= 0 {
		return fmt.Errorf("invalid file size: %d", request.FileSize)
	}

	// MIME 타입 검사
	if !fm.isAllowedMimeType(request.MimeType) {
		return fmt.Errorf("mime type not allowed: %s", request.MimeType)
	}

	// 파일 확장자 검사
	ext := strings.ToLower(filepath.Ext(request.FileName))
	if fm.isBlockedExtension(ext) {
		return fmt.Errorf("file extension not allowed: %s", ext)
	}

	// 파일명 검사
	if strings.TrimSpace(request.FileName) == "" {
		return fmt.Errorf("file name is required")
	}

	return nil
}

func (fm *FileManager) checkSessionFileLimit(sessionID string) error {
	files, err := fm.loadSessionFiles(sessionID)
	if err != nil {
		return err
	}

	activeFiles := 0
	for _, file := range files {
		if file.Status != FileStatusDeleted && file.Status != FileStatusExpired {
			activeFiles++
		}
	}

	if activeFiles >= fm.config.MaxFilesPerSession {
		return fmt.Errorf("session file limit exceeded: %d >= %d", activeFiles, fm.config.MaxFilesPerSession)
	}

	return nil
}

func (fm *FileManager) generateFileID(fileName, userID string) string {
	data := fmt.Sprintf("%s_%s_%d", fileName, userID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // 32자리 ID
}

func (fm *FileManager) generateStoredFileName(fileID, originalName string) string {
	ext := filepath.Ext(originalName)
	return fmt.Sprintf("%s%s", fileID, ext)
}

func (fm *FileManager) assembleFile(fileID string, metadata *FileMetadata) error {
	// 최종 파일 경로
	finalPath := filepath.Join(fm.uploadsPath, metadata.StoredName)

	// 최종 파일 생성
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return fmt.Errorf("failed to create final file: %w", err)
	}
	defer finalFile.Close()

	// 청크 디렉토리
	tempDir := filepath.Join(fm.uploadsPath, "temp")

	// 청크들을 순서대로 조립
	chunkIndex := 0
	for {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("%s_chunk_%d", fileID, chunkIndex))
		
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			if os.IsNotExist(err) {
				break // 더 이상 청크가 없음
			}
			return fmt.Errorf("failed to open chunk %d: %w", chunkIndex, err)
		}

		// 청크 데이터 복사
		if _, err := io.Copy(finalFile, chunkFile); err != nil {
			chunkFile.Close()
			return fmt.Errorf("failed to copy chunk %d: %w", chunkIndex, err)
		}

		chunkFile.Close()

		// 청크 파일 삭제
		os.Remove(chunkPath)

		chunkIndex++
	}

	// 체크섬 검증 (제공된 경우)
	if metadata.Checksum != "" {
		if err := fm.verifyChecksum(finalPath, metadata.Checksum); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	return nil
}

func (fm *FileManager) completeUpload(fileID string) {
	// 파일 상태를 완료로 변경
	fm.cacheMutex.Lock()
	if metadata, exists := fm.fileCache[fileID]; exists {
		metadata.Status = FileStatusReady
		fm.saveFileMetadata(metadata)
	}
	fm.cacheMutex.Unlock()

	// 업로드 진행 상태 업데이트
	fm.progressMutex.Lock()
	if progress, exists := fm.uploadProgress[fileID]; exists {
		progress.Status = "completed"
		progress.Progress = 100.0
		progress.LastUpdate = time.Now()
	}
	fm.progressMutex.Unlock()
}

func (fm *FileManager) markUploadError(fileID, errorMsg string) {
	fm.progressMutex.Lock()
	if progress, exists := fm.uploadProgress[fileID]; exists {
		progress.Status = "error"
		progress.Error = errorMsg
		progress.LastUpdate = time.Now()
	}
	fm.progressMutex.Unlock()

	fm.cacheMutex.Lock()
	if metadata, exists := fm.fileCache[fileID]; exists {
		metadata.Status = FileStatusError
	}
	fm.cacheMutex.Unlock()
}

func (fm *FileManager) verifyChecksum(filePath, expectedChecksum string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

func (fm *FileManager) isAllowedMimeType(mimeType string) bool {
	if len(fm.config.AllowedMimeTypes) == 0 {
		return true // 제한 없음
	}

	for _, allowed := range fm.config.AllowedMimeTypes {
		if mimeType == allowed {
			return true
		}
	}

	return false
}

func (fm *FileManager) isBlockedExtension(ext string) bool {
	for _, blocked := range fm.config.BlockedExtensions {
		if ext == blocked {
			return true
		}
	}
	return false
}

func (fm *FileManager) filterFiles(files []*FileMetadata, request FileListRequest) []*FileMetadata {
	var filtered []*FileMetadata

	for _, file := range files {
		// 파일 타입 필터
		if len(request.FileTypes) > 0 {
			found := false
			for _, fileType := range request.FileTypes {
				if strings.Contains(file.MimeType, fileType) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// 태그 필터
		if len(request.Tags) > 0 {
			found := false
			for _, reqTag := range request.Tags {
				for _, fileTag := range file.Tags {
					if fileTag == reqTag {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, file)
	}

	return filtered
}

func (fm *FileManager) sortFiles(files []*FileMetadata, sortBy, sortOrder string) []*FileMetadata {
	// 실제 구현에서는 정렬 로직 추가
	return files
}

func (fm *FileManager) paginateFiles(files []*FileMetadata, offset, limit int) []*FileMetadata {
	if limit == 0 {
		limit = 20 // 기본값
	}

	if offset >= len(files) {
		return []*FileMetadata{}
	}

	end := offset + limit
	if end > len(files) {
		end = len(files)
	}

	return files[offset:end]
}

func (fm *FileManager) cleanupRoutine() {
	ticker := time.NewTicker(fm.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		fm.cleanupExpiredFiles()
		fm.cleanupOldUploadProgress()
	}
}

func (fm *FileManager) cleanupExpiredFiles() {
	now := time.Now()

	fm.cacheMutex.RLock()
	var expiredFiles []string
	for fileID, metadata := range fm.fileCache {
		if metadata.ExpiresAt != nil && now.After(*metadata.ExpiresAt) {
			expiredFiles = append(expiredFiles, fileID)
		}
	}
	fm.cacheMutex.RUnlock()

	// 만료된 파일들 정리
	for _, fileID := range expiredFiles {
		fm.markFileExpired(fileID)
	}
}

func (fm *FileManager) cleanupOldUploadProgress() {
	cutoff := time.Now().Add(-24 * time.Hour) // 24시간 이전

	fm.progressMutex.Lock()
	for fileID, progress := range fm.uploadProgress {
		if progress.LastUpdate.Before(cutoff) {
			delete(fm.uploadProgress, fileID)
		}
	}
	fm.progressMutex.Unlock()
}

func (fm *FileManager) markFileExpired(fileID string) {
	fm.cacheMutex.Lock()
	if metadata, exists := fm.fileCache[fileID]; exists {
		metadata.Status = FileStatusExpired
		fm.saveFileMetadata(metadata)
	}
	fm.cacheMutex.Unlock()
}

// 스토리지 인터페이스 메서드들 (실제 구현에서는 storage 패키지와 연동)

func (fm *FileManager) saveFileMetadata(metadata *FileMetadata) error {
	// 실제 구현에서는 storage.Storage 인터페이스 사용
	return nil
}

func (fm *FileManager) loadFileMetadata(fileID string) (*FileMetadata, error) {
	// 실제 구현에서는 storage.Storage 인터페이스 사용
	return nil, fmt.Errorf("not implemented")
}

func (fm *FileManager) deleteFileMetadata(fileID string) error {
	// 실제 구현에서는 storage.Storage 인터페이스 사용
	return nil
}

func (fm *FileManager) loadSessionFiles(sessionID string) ([]*FileMetadata, error) {
	// 실제 구현에서는 storage.Storage 인터페이스 사용
	return []*FileMetadata{}, nil
}