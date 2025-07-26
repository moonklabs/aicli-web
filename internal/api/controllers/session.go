package controllers

import (
	"net/http"
	"strconv"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/aicli/aicli-web/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SessionController 세션 관리 컨트롤러
type SessionController struct {
	sessionService *services.SessionService
	logger         *zap.Logger
}

// NewSessionController 새 세션 컨트롤러 생성
func NewSessionController(sessionService *services.SessionService) *SessionController {
	logger, _ := zap.NewProduction()
	return &SessionController{
		sessionService: sessionService,
		logger:         logger,
	}
}

// Create 새 세션 생성
// @Summary 새 Claude 세션 생성
// @Description 프로젝트에 대한 새로운 Claude CLI 세션을 생성합니다
// @Tags sessions
// @Accept json
// @Produce json
// @Param project_id path string true "프로젝트 ID"
// @Param request body models.SessionCreateRequest true "세션 생성 요청"
// @Success 201 {object} models.SessionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /projects/{project_id}/sessions [post]
func (c *SessionController) Create(ctx *gin.Context) {
	projectID := ctx.Param("id")
	if projectID == "" {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "MISSING_PROJECT_ID",
				Message: "프로젝트 ID가 필요합니다",
			},
		})
		return
	}

	var req models.SessionCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "잘못된 요청 형식",
				Details: err.Error(),
			},
		})
		return
	}

	// 요청에 프로젝트 ID 설정
	req.ProjectID = projectID

	session, err := c.sessionService.Create(ctx, &req)
	if err != nil {
		c.logger.Error("세션 생성 실패",
			zap.String("project_id", projectID),
			zap.Error(err),
		)
		
		statusCode := http.StatusInternalServerError
		if err.Error() == "프로젝트를 찾을 수 없습니다" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "최대 동시 세션 수 초과" {
			statusCode = http.StatusTooManyRequests
		}
		
		ctx.JSON(statusCode, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "SESSION_CREATE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	ctx.JSON(http.StatusCreated, session.ToResponse())
}

// List 세션 목록 조회
// @Summary 세션 목록 조회
// @Description 활성 세션 목록을 조회합니다
// @Tags sessions
// @Accept json
// @Produce json
// @Param project_id query string false "프로젝트 ID"
// @Param status query string false "세션 상태"
// @Param active query boolean false "활성 세션만 조회"
// @Param page query int false "페이지 번호" default(1)
// @Param limit query int false "페이지 크기" default(20)
// @Success 200 {object} models.PagingResponse[models.SessionResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /sessions [get]
func (c *SessionController) List(ctx *gin.Context) {
	filter := &models.SessionFilter{
		ProjectID: ctx.Query("project_id"),
	}
	
	if status := ctx.Query("status"); status != "" {
		filter.Status = models.SessionStatus(status)
	}
	
	if activeStr := ctx.Query("active"); activeStr != "" {
		active, err := strconv.ParseBool(activeStr)
		if err == nil {
			filter.Active = &active
		}
	}
	
	paging := &models.PagingRequest{
		Page:  1,
		Limit: 20,
	}
	
	if page, err := strconv.Atoi(ctx.Query("page")); err == nil && page > 0 {
		paging.Page = page
	}
	
	if limit, err := strconv.Atoi(ctx.Query("limit")); err == nil && limit > 0 && limit <= 100 {
		paging.Limit = limit
	}
	
	result, err := c.sessionService.List(ctx, filter, paging)
	if err != nil {
		c.logger.Error("세션 목록 조회 실패", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "SESSION_LIST_FAILED",
				Message: "세션 목록 조회 실패",
				Details: err.Error(),
			},
		})
		return
	}
	
	// Response 변환
	sessions := result.Data.([]*models.Session)
	responses := make([]*models.SessionResponse, len(sessions))
	for i, session := range sessions {
		responses[i] = session.ToResponse()
	}
	
	ctx.JSON(http.StatusOK, models.PaginationResponse{
		Data: responses,
		Meta: result.Meta,
	})
}

// GetByID ID로 세션 조회
// @Summary 세션 정보 조회
// @Description ID로 세션 정보를 조회합니다
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "세션 ID"
// @Success 200 {object} models.SessionResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /sessions/{id} [get]
func (c *SessionController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "MISSING_SESSION_ID",
				Message: "세션 ID가 필요합니다",
			},
		})
		return
	}
	
	session, err := c.sessionService.GetByID(ctx, id)
	if err != nil {
		if err.Error() == "not found" {
			ctx.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    "SESSION_NOT_FOUND",
					Message: "세션을 찾을 수 없습니다",
				},
			})
			return
		}
		
		c.logger.Error("세션 조회 실패",
			zap.String("session_id", id),
			zap.Error(err),
		)
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "SESSION_GET_FAILED",
				Message: "세션 조회 실패",
				Details: err.Error(),
			},
		})
		return
	}
	
	ctx.JSON(http.StatusOK, session.ToResponse())
}

// Terminate 세션 종료
// @Summary 세션 종료
// @Description Claude 세션을 종료합니다
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "세션 ID"
// @Success 204
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /sessions/{id} [delete]
func (c *SessionController) Terminate(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "MISSING_SESSION_ID",
				Message: "세션 ID가 필요합니다",
			},
		})
		return
	}
	
	err := c.sessionService.Terminate(ctx, id)
	if err != nil {
		if err.Error() == "not found" {
			ctx.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    "SESSION_NOT_FOUND",
					Message: "세션을 찾을 수 없습니다",
				},
			})
			return
		}
		
		c.logger.Error("세션 종료 실패",
			zap.String("session_id", id),
			zap.Error(err),
		)
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "SESSION_TERMINATE_FAILED",
				Message: "세션 종료 실패",
				Details: err.Error(),
			},
		})
		return
	}
	
	ctx.Status(http.StatusNoContent)
}

// UpdateActivity 세션 활동 업데이트
// @Summary 세션 활동 업데이트
// @Description 세션의 마지막 활동 시간을 업데이트합니다
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "세션 ID"
// @Success 204
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /sessions/{id}/activity [put]
func (c *SessionController) UpdateActivity(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "MISSING_SESSION_ID",
				Message: "세션 ID가 필요합니다",
			},
		})
		return
	}
	
	err := c.sessionService.UpdateActivity(ctx, id)
	if err != nil {
		if err.Error() == "세션을 찾을 수 없습니다" {
			ctx.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    "SESSION_NOT_FOUND",
					Message: err.Error(),
				},
			})
			return
		}
		
		c.logger.Error("세션 활동 업데이트 실패",
			zap.String("session_id", id),
			zap.Error(err),
		)
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    "SESSION_UPDATE_ACTIVITY_FAILED",
				Message: "세션 활동 업데이트 실패",
				Details: err.Error(),
			},
		})
		return
	}
	
	ctx.Status(http.StatusNoContent)
}

// GetActiveSessions 활성 세션 목록 조회
// @Summary 활성 세션 목록 조회
// @Description 현재 활성화된 모든 세션을 조회합니다
// @Tags sessions
// @Accept json
// @Produce json
// @Success 200 {array} models.SessionResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /sessions/active [get]
func (c *SessionController) GetActiveSessions(ctx *gin.Context) {
	sessions := c.sessionService.GetActiveSessions()
	
	responses := make([]*models.SessionResponse, len(sessions))
	for i, session := range sessions {
		responses[i] = session.ToResponse()
	}
	
	ctx.JSON(http.StatusOK, responses)
}