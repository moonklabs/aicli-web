package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func BenchmarkHealthCheck(b *testing.B) {
	// 헬스체크 엔드포인트 성능 측정
	gin.SetMode(gin.ReleaseMode)
	s := New()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", w.Code)
		}
	}
}

func BenchmarkHealthCheckParallel(b *testing.B) {
	// 헬스체크 병렬 성능 측정
	gin.SetMode(gin.ReleaseMode)
	s := New()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			s.router.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				b.Fatalf("unexpected status code: %d", w.Code)
			}
		}
	})
}

func BenchmarkAPIWorkspaces(b *testing.B) {
	// 워크스페이스 목록 API 성능 측정
	gin.SetMode(gin.ReleaseMode)
	s := New()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/workspaces", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", w.Code)
		}
	}
}

func BenchmarkMiddlewareChain(b *testing.B) {
	// 미들웨어 체인 성능 측정
	gin.SetMode(gin.ReleaseMode)
	s := New()
	
	// 간단한 테스트 엔드포인트 추가
	s.router.GET("/bench", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/bench", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", w.Code)
		}
	}
}

func BenchmarkJSONResponse(b *testing.B) {
	// JSON 응답 성능 측정
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	
	// 큰 JSON 응답을 생성하는 핸들러
	router.GET("/json", func(c *gin.Context) {
		data := make([]map[string]interface{}, 100)
		for i := range data {
			data[i] = map[string]interface{}{
				"id":          i,
				"name":        "test",
				"description": "벤치마크 테스트용 데이터",
				"created_at":  "2025-01-20T12:00:00Z",
				"tags":        []string{"test", "benchmark"},
			}
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	})
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/json", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", w.Code)
		}
	}
}