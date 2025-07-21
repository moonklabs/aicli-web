package models

// ErrorResponse 에러 응답 구조체
// swagger:model ErrorResponse
type ErrorResponse struct {
	// 성공 여부
	// example: false
	Success bool `json:"success"`
	
	// 에러 정보
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 에러 상세 정보
// swagger:model ErrorDetail
type ErrorDetail struct {
	// 에러 코드
	// example: INVALID_REQUEST
	Code string `json:"code"`
	
	// 에러 메시지
	// example: Invalid request body
	Message string `json:"message"`
	
	// 에러 상세 정보 (선택적)
	// example: field 'username' is required
	Details string `json:"details,omitempty"`
}

// SuccessResponse 성공 응답 구조체
// swagger:model SuccessResponse
type SuccessResponse struct {
	// 성공 여부
	// example: true
	Success bool `json:"success"`
	
	// 응답 데이터
	Data interface{} `json:"data,omitempty"`
	
	// 메시지 (선택적)
	// example: Operation completed successfully
	Message string `json:"message,omitempty"`
}

// PaginationMeta 페이지네이션 메타 정보
// swagger:model PaginationMeta
type PaginationMeta struct {
	// 현재 페이지
	// example: 1
	CurrentPage int `json:"current_page"`
	
	// 페이지당 항목 수
	// example: 10
	PerPage int `json:"per_page"`
	
	// 전체 항목 수
	// example: 100
	Total int `json:"total"`
	
	// 전체 페이지 수
	// example: 10
	TotalPages int `json:"total_pages"`
}