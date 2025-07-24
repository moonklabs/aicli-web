package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	apierrors "github.com/aicli/aicli-web/internal/errors"
)

// ErrorResponse는 표준 에러 응답 구조체입니다.
type ErrorResponse struct {
	Success bool  `json:"success"`
	Error   Error `json:"error"`
}

// Error는 에러 정보 구조체입니다.
type Error struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// 에러 코드 상수
const (
	ErrValidation   = "ERR_VALIDATION"
	ErrNotFound     = "ERR_NOT_FOUND"
	ErrUnauthorized = "ERR_UNAUTHORIZED"
	ErrForbidden    = "ERR_FORBIDDEN"
	ErrConflict     = "ERR_CONFLICT"
	ErrInternal     = "ERR_INTERNAL"
	ErrBadRequest   = "ERR_BAD_REQUEST"
	ErrTimeout      = "ERR_TIMEOUT"
)

// ErrorHandler는 표준화된 에러 처리 미들웨어입니다.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 에러가 있는 경우 처리
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			requestID := GetRequestID(c)

			// 에러 타입에 따른 응답 생성
			response := createErrorResponse(err, requestID)
			
			// 이미 응답이 작성된 경우 처리하지 않음
			if c.Writer.Written() {
				return
			}

			// HTTP 상태 코드 결정
			statusCode := determineStatusCode(err)
			
			c.JSON(statusCode, response)
			c.Abort()
		}
	}
}

// createErrorResponse는 에러로부터 표준 응답을 생성합니다.
func createErrorResponse(err *gin.Error, requestID string) ErrorResponse {
	// 에러 코드와 메시지 결정
	code := ErrInternal
	message := "서버 내부 오류가 발생했습니다"
	var details interface{}

	// Gin의 에러 타입에 따른 처리
	switch err.Type {
	case gin.ErrorTypeBind:
		code = ErrValidation
		message = "요청 데이터가 올바르지 않습니다"
		details = err.Error()
	case gin.ErrorTypePublic:
		code = ErrBadRequest
		message = err.Error()
	case gin.ErrorTypePrivate:
		code = ErrInternal
		message = "서버 내부 오류가 발생했습니다"
		// 개발 환경에서만 상세 정보 제공
		if gin.Mode() == gin.DebugMode {
			details = err.Error()
		}
	default:
		// 기본 처리
		if err.Error() != "" {
			message = err.Error()
		}
	}

	return ErrorResponse{
		Success: false,
		Error: Error{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: requestID,
		},
	}
}

// determineStatusCode는 에러로부터 HTTP 상태 코드를 결정합니다.
func determineStatusCode(err *gin.Error) int {
	switch err.Type {
	case gin.ErrorTypeBind:
		return http.StatusBadRequest
	case gin.ErrorTypePublic:
		return http.StatusBadRequest
	case gin.ErrorTypePrivate:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewAPIError는 새로운 API 에러를 생성합니다.
func NewAPIError(code string, message string, details interface{}) *gin.Error {
	var detailsMap map[string]interface{}
	if details != nil {
		if dm, ok := details.(map[string]interface{}); ok {
			detailsMap = dm
		}
	}
	
	return &gin.Error{
		Err: &APIError{
			Code:      code,
			Message:   message,
			Details:   detailsMap,
			TraceID:   "",
			Timestamp: time.Now(),
		},
		Type: gin.ErrorTypePublic,
	}
}


// AbortWithError는 에러와 함께 요청을 중단합니다.
func AbortWithError(c *gin.Context, statusCode int, code string, message string, details interface{}) {
	requestID := GetRequestID(c)
	
	response := ErrorResponse{
		Success: false,
		Error: Error{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: requestID,
		},
	}
	
	c.JSON(statusCode, response)
	c.Abort()
}

// ValidationError는 유효성 검사 에러를 처리합니다.
func ValidationError(c *gin.Context, message string, details interface{}) {
	AbortWithError(c, http.StatusBadRequest, ErrValidation, message, details)
}

// NotFoundError는 리소스를 찾을 수 없는 에러를 처리합니다.
func NotFoundError(c *gin.Context, message string) {
	AbortWithError(c, http.StatusNotFound, ErrNotFound, message, nil)
}

// UnauthorizedError는 인증 실패 에러를 처리합니다.
func UnauthorizedError(c *gin.Context, message string) {
	AbortWithError(c, http.StatusUnauthorized, ErrUnauthorized, message, nil)
}

// ForbiddenError는 권한 부족 에러를 처리합니다.
func ForbiddenError(c *gin.Context, message string) {
	AbortWithError(c, http.StatusForbidden, ErrForbidden, message, nil)
}

// ConflictError는 리소스 충돌 에러를 처리합니다.
func ConflictError(c *gin.Context, message string) {
	AbortWithError(c, http.StatusConflict, "CONFLICT", message, nil)
}

// InternalError는 서버 내부 에러를 처리합니다.
func InternalError(c *gin.Context, message string, details interface{}) {
	AbortWithError(c, http.StatusInternalServerError, ErrInternal, message, details)
}

// HandleServiceError는 서비스 계층의 에러를 적절한 HTTP 응답으로 변환합니다.
func HandleServiceError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	
	// WorkspaceError 타입 확인
	var workspaceErr *apierrors.WorkspaceError
	if errors.As(err, &workspaceErr) {
		statusCode := getHTTPStatusFromWorkspaceError(workspaceErr)
		AbortWithError(c, statusCode, workspaceErr.Code, workspaceErr.Message, nil)
		return
	}
	
	// 일반적인 서비스 에러 처리
	switch err {
	case apierrors.ErrWorkspaceNotFound:
		NotFoundError(c, "워크스페이스를 찾을 수 없습니다")
	case apierrors.ErrInvalidWorkspaceName:
		ValidationError(c, "워크스페이스 이름이 유효하지 않습니다", nil)
	case apierrors.ErrInvalidProjectPath:
		ValidationError(c, "프로젝트 경로가 유효하지 않습니다", nil)
	case apierrors.ErrWorkspaceExists:
		ConflictError(c, "이미 존재하는 워크스페이스입니다")
	case apierrors.ErrUnauthorized:
		UnauthorizedError(c, "접근 권한이 없습니다")
	case apierrors.ErrInvalidRequest:
		ValidationError(c, "잘못된 요청입니다", nil)
	case apierrors.ErrInvalidWorkspaceStatus:
		ValidationError(c, "워크스페이스 상태가 유효하지 않습니다", nil)
	case apierrors.ErrWorkspaceNotActive:
		AbortWithError(c, http.StatusBadRequest, "WORKSPACE_NOT_ACTIVE", "워크스페이스가 활성 상태가 아닙니다", nil)
	case apierrors.ErrWorkspaceArchived:
		AbortWithError(c, http.StatusBadRequest, "WORKSPACE_ARCHIVED", "아카이브된 워크스페이스입니다", nil)
	case apierrors.ErrInsufficientPermissions:
		ForbiddenError(c, "권한이 부족합니다")
	case apierrors.ErrOwnershipRequired:
		ForbiddenError(c, "소유자 권한이 필요합니다")
	case apierrors.ErrMaxWorkspacesReached:
		AbortWithError(c, http.StatusBadRequest, "MAX_WORKSPACES_REACHED", "최대 워크스페이스 수에 도달했습니다", nil)
	case apierrors.ErrResourceBusy:
		AbortWithError(c, http.StatusConflict, "RESOURCE_BUSY", "리소스가 사용 중입니다", nil)
	case apierrors.ErrDependencyExists:
		AbortWithError(c, http.StatusConflict, "DEPENDENCY_EXISTS", "의존성이 존재합니다", nil)
	default:
		// 알 수 없는 에러는 내부 서버 에러로 처리
		InternalError(c, "서버 내부 오류가 발생했습니다", err.Error())
	}
}

// getHTTPStatusFromWorkspaceError는 WorkspaceError의 코드를 HTTP 상태 코드로 변환합니다.
func getHTTPStatusFromWorkspaceError(err *apierrors.WorkspaceError) int {
	switch err.Code {
	case apierrors.ErrCodeNotFound:
		return http.StatusNotFound
	case apierrors.ErrCodeInvalidName, apierrors.ErrCodeInvalidPath, apierrors.ErrCodeInvalidRequest, apierrors.ErrCodeInvalidStatus:
		return http.StatusBadRequest
	case apierrors.ErrCodeAlreadyExists:
		return http.StatusConflict
	case apierrors.ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case apierrors.ErrCodeInsufficientPerm, apierrors.ErrCodeOwnershipRequired:
		return http.StatusForbidden
	case apierrors.ErrCodeMaxWorkspaces, apierrors.ErrCodeNotActive, apierrors.ErrCodeArchived:
		return http.StatusBadRequest
	case apierrors.ErrCodeResourceBusy, apierrors.ErrCodeDependencyExists:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}