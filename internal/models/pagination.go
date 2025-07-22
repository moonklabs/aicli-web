package models

import (
	"math"
)

// PaginationRequest 페이지네이션 요청 파라미터
type PaginationRequest struct {
	// 페이지 번호 (1부터 시작)
	Page int `form:"page,default=1" binding:"min=1"`
	
	// 페이지당 항목 수
	Limit int `form:"limit,default=20" binding:"min=1,max=100"`
	
	// 정렬 필드
	Sort string `form:"sort,default=created_at"`
	
	// 정렬 순서 (asc, desc)
	Order string `form:"order,default=desc" binding:"oneof=asc desc"`
}

// PaginationResponse 페이지네이션 응답
type PaginationResponse struct {
	// 데이터 배열
	Data interface{} `json:"data"`
	
	// 페이지네이션 메타 정보
	Meta PaginationMeta `json:"meta"`
}

// GetOffset 오프셋 계산
func (p *PaginationRequest) GetOffset() int {
	return (p.Page - 1) * p.Limit
}

// Normalize 기본값 설정
func (p *PaginationRequest) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if p.Order != "asc" && p.Order != "desc" {
		p.Order = "desc"
	}
}

// NewPaginationMeta 페이지네이션 메타 정보 생성
func NewPaginationMeta(page, limit, total int) PaginationMeta {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	
	return PaginationMeta{
		CurrentPage: page,
		PerPage:     limit,
		Total:       total,
		TotalPages:  totalPages,
	}
}

// HasMore 다음 페이지 존재 여부
func (m *PaginationMeta) HasMore() bool {
	return m.CurrentPage < m.TotalPages
}

// HasPrev 이전 페이지 존재 여부
func (m *PaginationMeta) HasPrev() bool {
	return m.CurrentPage > 1
}

// 별칭 타입 정의 (호환성을 위한)
type PagingRequest = PaginationRequest
type PagingResponse = PaginationResponse