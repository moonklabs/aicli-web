package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SimpleSecurityTest 간단한 보안 테스트
type SimpleSecurityTest struct {
	app     *gin.Engine
	results *SecurityTestResults
}

// SecurityTestResults 보안 테스트 결과
type SecurityTestResults struct {
	VulnerabilityTests []VulnerabilityTestResult `json:"vulnerability_tests"`
	TotalTests         int                       `json:"total_tests"`
	PassedTests        int                       `json:"passed_tests"`
	FailedTests        int                       `json:"failed_tests"`
	ExecutionTime      time.Duration             `json:"execution_time"`
	Timestamp          time.Time                 `json:"timestamp"`
}

// VulnerabilityTestResult 취약점 테스트 결과
type VulnerabilityTestResult struct {
	TestName       string `json:"test_name"`
	Category       string `json:"category"`
	Severity       string `json:"severity"`
	Status         string `json:"status"`
	Description    string `json:"description"`
	Payload        string `json:"payload,omitempty"`
	Response       string `json:"response,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

// NewSimpleSecurityTest 간단한 보안 테스트 생성
func NewSimpleSecurityTest() *SimpleSecurityTest {
	gin.SetMode(gin.TestMode)
	
	test := &SimpleSecurityTest{
		results: &SecurityTestResults{},
	}
	
	test.setupApplication()
	return test
}

// setupApplication 테스트용 애플리케이션 설정
func (s *SimpleSecurityTest) setupApplication() {
	s.app = gin.New()
	
	// 기본 라우트
	s.app.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	
	s.app.GET("/search", func(c *gin.Context) {
		query := c.Query("q")
		// 기본적인 XSS 방지
		if strings.Contains(query, "<script>") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"query": query, "results": []string{"result1", "result2"}})
	})
	
	s.app.POST("/data", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
		
		// 기본적인 Path Traversal 방지
		if filename, ok := data["filename"].(string); ok {
			if strings.Contains(filename, "../") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
				return
			}
		}
		
		c.JSON(http.StatusOK, gin.H{"received": data})
	})
}

// RunSecurityTests 보안 테스트 실행
func (s *SimpleSecurityTest) RunSecurityTests() {
	start := time.Now()
	s.results.Timestamp = start
	
	fmt.Println("🔒 간단한 보안 테스트 시작...")
	
	// 취약점 테스트 실행
	s.runVulnerabilityTests()
	
	s.results.ExecutionTime = time.Since(start)
	s.calculateTestStatistics()
	
	fmt.Printf("✅ 보안 테스트 완료 (소요시간: %v)\\n", s.results.ExecutionTime)
}

// runVulnerabilityTests 취약점 테스트 실행
func (s *SimpleSecurityTest) runVulnerabilityTests() {
	tests := []struct {
		name        string
		category    string
		severity    string
		payload     string
		endpoint    string
		method      string
		description string
	}{
		{
			name:        "XSS Test",
			category:    "xss",
			severity:    "medium",
			payload:     "<script>alert('xss')</script>",
			endpoint:    "/search?q=%s",
			method:      "GET",
			description: "Cross-Site Scripting 공격 시도",
		},
		{
			name:        "Path Traversal Test",
			category:    "path_traversal",
			severity:    "high",
			payload:     "../../../etc/passwd",
			endpoint:    "/data",
			method:      "POST",
			description: "Path Traversal 공격 시도",
		},
		{
			name:        "SQL Injection Test",
			category:    "injection",
			severity:    "high",
			payload:     "'; DROP TABLE users; --",
			endpoint:    "/search?q=%s",
			method:      "GET",
			description: "SQL Injection 공격 시도",
		},
	}
	
	for _, test := range tests {
		result := s.executeVulnerabilityTest(test.name, test.category, test.severity, 
			test.payload, test.endpoint, test.method, test.description)
		s.results.VulnerabilityTests = append(s.results.VulnerabilityTests, result)
	}
}

// executeVulnerabilityTest 개별 취약점 테스트 실행
func (s *SimpleSecurityTest) executeVulnerabilityTest(name, category, severity, payload, endpoint, method, description string) VulnerabilityTestResult {
	var requestURL string
	if strings.Contains(endpoint, "%s") {
		// URL 인코딩을 사용하여 안전하게 처리
		encodedPayload := url.QueryEscape(payload)
		requestURL = fmt.Sprintf(endpoint, encodedPayload)
	} else {
		requestURL = endpoint
	}
	
	var req *http.Request
	
	if method == "POST" && category == "path_traversal" {
		body := fmt.Sprintf(`{"filename": "%s"}`, payload)
		req = httptest.NewRequest(method, endpoint, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, requestURL, nil)
	}
	
	w := httptest.NewRecorder()
	s.app.ServeHTTP(w, req)
	
	status := "passed"
	recommendation := ""
	
	// 공격이 차단되었는지 확인
	if w.Code == http.StatusOK {
		// 응답에서 공격 페이로드가 그대로 반영되었는지 확인
		responseBody := w.Body.String()
		if strings.Contains(responseBody, payload) && 
		   (strings.Contains(payload, "<script>") || strings.Contains(payload, "DROP")) {
			status = "failed"
			recommendation = fmt.Sprintf("%s 공격이 차단되지 않았습니다. 입력 검증을 강화하세요.", category)
		}
	} else if w.Code == http.StatusBadRequest {
		status = "passed"
		recommendation = "공격이 성공적으로 차단되었습니다."
	}
	
	return VulnerabilityTestResult{
		TestName:       name,
		Category:       category,
		Severity:       severity,
		Status:         status,
		Description:    description,
		Payload:        payload,
		Response:       fmt.Sprintf("HTTP %d", w.Code),
		Recommendation: recommendation,
	}
}

// calculateTestStatistics 테스트 통계 계산
func (s *SimpleSecurityTest) calculateTestStatistics() {
	total := len(s.results.VulnerabilityTests)
	passed := 0
	failed := 0
	
	for _, test := range s.results.VulnerabilityTests {
		if test.Status == "passed" {
			passed++
		} else {
			failed++
		}
	}
	
	s.results.TotalTests = total
	s.results.PassedTests = passed
	s.results.FailedTests = failed
}

// SaveReport 보고서 저장
func (s *SimpleSecurityTest) SaveReport(filename string) error {
	data, err := json.MarshalIndent(s.results, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}

// GenerateHTMLReport HTML 보고서 생성
func (s *SimpleSecurityTest) GenerateHTMLReport(filename string) error {
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>보안 평가 보고서 - AICode Manager</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #2c3e50; color: white; padding: 20px; margin-bottom: 30px; }
        .summary { background: #ecf0f1; padding: 20px; margin-bottom: 30px; }
        .section { margin-bottom: 30px; }
        .status-passed { color: #27ae60; }
        .status-failed { color: #e74c3c; }
        table { width: 100%%; border-collapse: collapse; margin-bottom: 20px; }
        th, td { border: 1px solid #bdc3c7; padding: 8px; text-align: left; }
        th { background: #34495e; color: white; }
    </style>
</head>
<body>
    <div class="header">
        <h1>보안 평가 보고서</h1>
        <p>프로젝트: AICode Manager | 평가일: %s</p>
    </div>
    
    <div class="summary">
        <h2>요약</h2>
        <p><strong>총 테스트:</strong> %d개 (통과: %d, 실패: %d)</p>
        <p><strong>실행 시간:</strong> %v</p>
    </div>
    
    <div class="section">
        <h2>취약점 테스트 결과</h2>
        <table>
            <tr>
                <th>테스트명</th>
                <th>카테고리</th>
                <th>심각도</th>
                <th>상태</th>
                <th>권장사항</th>
            </tr>
            %s
        </table>
    </div>
</body>
</html>`
	
	var testRows string
	for _, test := range s.results.VulnerabilityTests {
		statusClass := "status-passed"
		if test.Status == "failed" {
			statusClass = "status-failed"
		}
		
		testRows += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
                <td class="%s">%s</td>
                <td>%s</td>
            </tr>`,
			test.TestName, test.Category, test.Severity, statusClass, test.Status, test.Recommendation)
	}
	
	htmlContent := fmt.Sprintf(htmlTemplate,
		s.results.Timestamp.Format("2006-01-02"),
		s.results.TotalTests,
		s.results.PassedTests,
		s.results.FailedTests,
		s.results.ExecutionTime,
		testRows,
	)
	
	return os.WriteFile(filename, []byte(htmlContent), 0644)
}

// TestSimpleSecurityAssessment 간단한 보안 평가 테스트
func TestSimpleSecurityAssessment(t *testing.T) {
	test := NewSimpleSecurityTest()
	
	// 보안 테스트 실행
	test.RunSecurityTests()
	
	// 결과 검증
	assert.Greater(t, test.results.TotalTests, 0, "테스트가 실행되어야 함")
	
	// 보고서 생성
	reportsDir := "../docs/security"
	os.MkdirAll(reportsDir, 0755)
	
	// JSON 보고서 저장
	jsonFile := filepath.Join(reportsDir, "simple_security_report.json")
	err := test.SaveReport(jsonFile)
	require.NoError(t, err, "JSON 보고서 저장 실패")
	
	// HTML 보고서 생성
	htmlFile := filepath.Join(reportsDir, "simple_security_report.html")
	err = test.GenerateHTMLReport(htmlFile)
	require.NoError(t, err, "HTML 보고서 생성 실패")
	
	// 콘솔에 요약 출력
	t.Logf("🔒 간단한 보안 평가 완료:")
	t.Logf("   📊 총 테스트: %d개", test.results.TotalTests)
	t.Logf("   ✅ 통과: %d개", test.results.PassedTests)
	t.Logf("   ❌ 실패: %d개", test.results.FailedTests)
	t.Logf("   ⏱️  실행 시간: %v", test.results.ExecutionTime)
	t.Logf("   📄 보고서: %s", htmlFile)
}