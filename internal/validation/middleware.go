package validation

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidationMiddleware Gin 검증 미들웨어
func ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 바인딩 에러를 검증 에러로 변환
		if len(c.Errors) > 0 {
			err := c.Errors[0]
			
			// validator.ValidationErrors 타입인지 확인
			if ve, ok := err.Err.(validator.ValidationErrors); ok {
				validationErrors := TranslateValidatorError(ve, "request")
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "VALIDATION_FAILED",
						"message": "요청 데이터 검증에 실패했습니다",
						"details": validationErrors,
					},
				})
				c.Abort()
				return
			}

			// BusinessValidationError 타입인지 확인
			if bve, ok := err.Err.(BusinessValidationError); ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error": gin.H{
						"code":    bve.Code,
						"message": bve.Message,
						"details": bve.Details,
						"field":   bve.Field,
						"value":   bve.Value,
					},
				})
				c.Abort()
				return
			}

			// ValidationErrors 타입인지 확인
			if vErrors, ok := err.Err.(ValidationErrors); ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "VALIDATION_FAILED",
						"message": "데이터 검증에 실패했습니다",
						"details": vErrors,
					},
				})
				c.Abort()
				return
			}
		}
	}
}

// ValidateRequestBody 요청 본문 검증 헬퍼 함수
func ValidateRequestBody(c *gin.Context, model interface{}) bool {
	// Gin의 기본 바인딩 (JSON 태그 기반)
	if err := c.ShouldBindJSON(model); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			validationErrors := TranslateValidatorError(ve, "request")
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_FAILED",
					"message": "요청 데이터 검증에 실패했습니다",
					"details": validationErrors,
				},
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_REQUEST",
					"message": "잘못된 요청 형식입니다",
					"details": err.Error(),
				},
			})
		}
		return false
	}

	// 추가 검증 (validate 태그 기반)
	if err := Validate(model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": "데이터 검증에 실패했습니다",
				"details": err,
			},
		})
		return false
	}

	return true
}

// ValidateBusinessRules 비즈니스 규칙 검증 헬퍼 함수
func ValidateBusinessRules(c *gin.Context, operation string, model interface{}, id ...string) bool {
	ctx := c.Request.Context()
	
	var err error
	switch operation {
	case "create":
		err = ValidateBusinessCreate(ctx, model)
	case "update":
		err = ValidateBusinessUpdate(ctx, model)
	case "delete":
		if len(id) > 0 {
			err = ValidateBusinessDelete(ctx, model, id[0])
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_ID",
					"message": "삭제 검증을 위한 ID가 필요합니다",
				},
			})
			return false
		}
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_OPERATION",
				"message": "유효하지 않은 검증 작업입니다",
			},
		})
		return false
	}

	if err != nil {
		if bve, ok := err.(BusinessValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    bve.Code,
					"message": bve.Message,
					"details": bve.Details,
					"field":   bve.Field,
					"value":   bve.Value,
				},
			})
		} else if vErrors, ok := err.(ValidationErrors); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_FAILED",
					"message": "비즈니스 규칙 검증에 실패했습니다",
					"details": vErrors,
				},
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "검증 중 오류가 발생했습니다",
					"details": err.Error(),
				},
			})
		}
		return false
	}

	return true
}

// ValidationService 검증 서비스 구조체
type ValidationService struct {
	manager *ValidationManager
}

// NewValidationService 새로운 검증 서비스 생성
func NewValidationService() *ValidationService {
	return &ValidationService{
		manager: GetGinValidator(),
	}
}

// ValidateStruct 구조체 검증
func (s *ValidationService) ValidateStruct(model interface{}) error {
	return s.manager.Validate(model)
}

// ValidateField 단일 필드 검증
func (s *ValidationService) ValidateField(field interface{}, tag string) error {
	return s.manager.ValidateVar(field, tag)
}

// ValidateBusinessCreate 생성 시 비즈니스 검증
func (s *ValidationService) ValidateBusinessCreate(ctx context.Context, model interface{}) error {
	return s.manager.ValidateBusinessCreate(ctx, model)
}

// ValidateBusinessUpdate 수정 시 비즈니스 검증
func (s *ValidationService) ValidateBusinessUpdate(ctx context.Context, model interface{}) error {
	return s.manager.ValidateBusinessUpdate(ctx, model)
}

// ValidateBusinessDelete 삭제 시 비즈니스 검증
func (s *ValidationService) ValidateBusinessDelete(ctx context.Context, model interface{}, id string) error {
	return s.manager.ValidateBusinessDelete(ctx, model, id)
}

// RegisterBusinessValidator 비즈니스 검증자 등록
func (s *ValidationService) RegisterBusinessValidator(model interface{}, validator BusinessValidator) {
	s.manager.RegisterBusinessValidator(model, validator)
}

// ValidationHandlerFactory 검증 핸들러 팩토리
type ValidationHandlerFactory struct {
	service *ValidationService
}

// NewValidationHandlerFactory 새로운 검증 핸들러 팩토리 생성
func NewValidationHandlerFactory(service *ValidationService) *ValidationHandlerFactory {
	return &ValidationHandlerFactory{
		service: service,
	}
}

// CreateHandler 생성 검증 핸들러
func (f *ValidationHandlerFactory) CreateHandler(modelType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 모델 타입에 따른 검증 로직
		switch strings.ToLower(modelType) {
		case "workspace":
			var req interface{}
			if !ValidateRequestBody(c, &req) {
				return
			}
			if !ValidateBusinessRules(c, "create", &req) {
				return
			}
		case "project":
			var req interface{}
			if !ValidateRequestBody(c, &req) {
				return
			}
			if !ValidateBusinessRules(c, "create", &req) {
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNKNOWN_MODEL_TYPE",
					"message": "알 수 없는 모델 타입입니다",
				},
			})
			return
		}

		c.Next()
	}
}

// UpdateHandler 수정 검증 핸들러
func (f *ValidationHandlerFactory) UpdateHandler(modelType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_ID",
					"message": "ID 파라미터가 필요합니다",
				},
			})
			return
		}

		// 모델 타입에 따른 검증 로직
		switch strings.ToLower(modelType) {
		case "workspace":
			var req interface{}
			if !ValidateRequestBody(c, &req) {
				return
			}
			if !ValidateBusinessRules(c, "update", &req) {
				return
			}
		case "project":
			var req interface{}
			if !ValidateRequestBody(c, &req) {
				return
			}
			if !ValidateBusinessRules(c, "update", &req) {
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNKNOWN_MODEL_TYPE",
					"message": "알 수 없는 모델 타입입니다",
				},
			})
			return
		}

		c.Next()
	}
}

// DeleteHandler 삭제 검증 핸들러
func (f *ValidationHandlerFactory) DeleteHandler(modelType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_ID",
					"message": "ID 파라미터가 필요합니다",
				},
			})
			return
		}

		// ID 형식 검증
		if err := ValidateVar(id, "uuid"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_ID_FORMAT",
					"message": "유효하지 않은 ID 형식입니다",
					"details": err,
				},
			})
			return
		}

		// 모델 타입에 따른 검증 로직
		var model interface{}
		switch strings.ToLower(modelType) {
		case "workspace":
			model = &struct{}{}
		case "project":
			model = &struct{}{}
		case "session":
			model = &struct{}{}
		case "task":
			model = &struct{}{}
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNKNOWN_MODEL_TYPE",
					"message": "알 수 없는 모델 타입입니다",
				},
			})
			return
		}

		if !ValidateBusinessRules(c, "delete", model, id) {
			return
		}

		c.Next()
	}
}

// ErrorHandler 검증 에러 전용 핸들러
func ErrorHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(BusinessValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    err.Code,
					"message": err.Message,
					"details": err.Details,
					"field":   err.Field,
					"value":   err.Value,
				},
			})
		} else if err, ok := recovered.(ValidationErrors); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_FAILED",
					"message": "데이터 검증에 실패했습니다",
					"details": err,
				},
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INTERNAL_SERVER_ERROR",
					"message": "서버 내부 오류가 발생했습니다",
				},
			})
		}
		c.Abort()
	})
}