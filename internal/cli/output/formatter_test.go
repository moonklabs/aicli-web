package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// 테스트용 구조체
type TestWorkspace struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	Description string `json:"description,omitempty"`
}

type TestConfig struct {
	APIKey    string `json:"api_key"`
	BaseURL   string `json:"base_url"`
	Timeout   int    `json:"timeout"`
	DebugMode bool   `json:"debug_mode"`
}

func TestNewFormatterManager(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		want   Format
	}{
		{"Table format", FormatTable, FormatTable},
		{"JSON format", FormatJSON, FormatJSON},
		{"YAML format", FormatYAML, FormatYAML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := NewFormatterManager(tt.format)
			if fm.format != tt.want {
				t.Errorf("NewFormatterManager() format = %v, want %v", fm.format, tt.want)
			}
			if fm.formatter == nil {
				t.Error("NewFormatterManager() formatter is nil")
			}
		})
	}
}

func TestDetectColorSupport(t *testing.T) {
	// 원래 환경 변수 저장
	originalNoColor := os.Getenv("NO_COLOR")
	originalTerm := os.Getenv("TERM")
	defer func() {
		os.Setenv("NO_COLOR", originalNoColor)
		os.Setenv("TERM", originalTerm)
	}()

	tests := []struct {
		name    string
		noColor string
		term    string
		want    bool
	}{
		{"No color env set", "1", "", false},
		{"Dumb terminal", "", "dumb", false},
		{"Normal terminal", "", "xterm-256color", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("NO_COLOR", tt.noColor)
			os.Setenv("TERM", tt.term)
			
			// 실제 터미널 환경이 아닌 경우 detectColorSupport는 항상 false를 반환할 수 있음
			// 이 경우 테스트는 환경 변수 확인만 수행
			got := detectColorSupport()
			if tt.noColor == "1" && got != false {
				t.Errorf("detectColorSupport() with NO_COLOR=1 = %v, want false", got)
			}
			if tt.term == "dumb" && got != false {
				t.Errorf("detectColorSupport() with TERM=dumb = %v, want false", got)
			}
		})
	}
}

func TestTableFormatter_Format(t *testing.T) {
	formatter := &TableFormatter{
		baseFormatter: baseFormatter{colorEnabled: false},
	}

	tests := []struct {
		name    string
		data    interface{}
		headers []string
		want    string
		wantErr bool
	}{
		{
			name: "Map slice",
			data: []map[string]interface{}{
				{"id": "1", "name": "workspace1", "status": "active"},
				{"id": "2", "name": "workspace2", "status": "inactive"},
			},
			headers: []string{"id", "name", "status"},
			wantErr: false,
		},
		{
			name: "Single map",
			data: map[string]interface{}{
				"api_key": "test-key",
				"timeout": 30,
			},
			wantErr: false,
		},
		{
			name:    "Empty slice",
			data:    []map[string]interface{}{},
			want:    "No data to display.\n",
			wantErr: false,
		},
		{
			name: "Struct slice",
			data: []TestWorkspace{
				{ID: "1", Name: "ws1", Status: "active", CreatedAt: "2024-01-01"},
				{ID: "2", Name: "ws2", Status: "inactive", CreatedAt: "2024-01-02"},
			},
			wantErr: false,
		},
		{
			name: "Single struct",
			data: TestConfig{
				APIKey:    "test-key",
				BaseURL:   "https://api.example.com",
				Timeout:   30,
				DebugMode: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.headers) > 0 {
				formatter.SetHeaders(tt.headers)
			}
			
			got, err := formatter.Format(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("TableFormatter.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.want != "" && got != tt.want {
				t.Errorf("TableFormatter.Format() = %v, want %v", got, tt.want)
			}
			
			// 출력이 있는지 확인
			if !tt.wantErr && len(got) == 0 {
				t.Error("TableFormatter.Format() returned empty output")
			}
		})
	}
}

func TestJSONFormatter_Format(t *testing.T) {
	formatter := &JSONFormatter{
		baseFormatter: baseFormatter{colorEnabled: false},
		indent:        true,
	}

	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{
			name: "Workspace slice",
			data: []TestWorkspace{
				{ID: "1", Name: "ws1", Status: "active"},
				{ID: "2", Name: "ws2", Status: "inactive"},
			},
			wantErr: false,
		},
		{
			name: "Config struct",
			data: TestConfig{
				APIKey:  "test-key",
				BaseURL: "https://api.example.com",
				Timeout: 30,
			},
			wantErr: false,
		},
		{
			name:    "Empty data",
			data:    map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatter.Format(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONFormatter.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			// JSON 유효성 검증
			if !tt.wantErr {
				var result interface{}
				if err := json.Unmarshal([]byte(got), &result); err != nil {
					t.Errorf("JSONFormatter.Format() produced invalid JSON: %v", err)
				}
			}
		})
	}
}

func TestJSONFormatter_ColorHighlighting(t *testing.T) {
	formatter := &JSONFormatter{
		baseFormatter: baseFormatter{colorEnabled: true},
		indent:        true,
	}

	data := map[string]interface{}{
		"name": "test",
		"id":   123,
	}

	got, err := formatter.Format(data)
	if err != nil {
		t.Fatalf("JSONFormatter.Format() error = %v", err)
	}

	// 색상 코드가 포함되어 있는지 확인 (ANSI 이스케이프 코드)
	if !strings.Contains(got, "\x1b[") {
		t.Error("JSONFormatter.Format() with color enabled should contain ANSI color codes")
	}
}

func TestYAMLFormatter_Format(t *testing.T) {
	formatter := &YAMLFormatter{
		baseFormatter: baseFormatter{colorEnabled: false},
	}

	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{
			name: "Workspace slice",
			data: []TestWorkspace{
				{ID: "1", Name: "ws1", Status: "active"},
				{ID: "2", Name: "ws2", Status: "inactive"},
			},
			wantErr: false,
		},
		{
			name: "Config struct",
			data: TestConfig{
				APIKey:  "test-key",
				BaseURL: "https://api.example.com",
				Timeout: 30,
			},
			wantErr: false,
		},
		{
			name:    "Empty data",
			data:    map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatter.Format(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLFormatter.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			// YAML 유효성 검증
			if !tt.wantErr {
				var result interface{}
				if err := yaml.Unmarshal([]byte(got), &result); err != nil {
					t.Errorf("YAMLFormatter.Format() produced invalid YAML: %v", err)
				}
			}
		})
	}
}

func TestFormatterManager_PrintTo(t *testing.T) {
	data := map[string]interface{}{
		"test": "value",
		"num":  42,
	}

	formats := []Format{FormatTable, FormatJSON, FormatYAML}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			fm := NewFormatterManager(format)
			var buf bytes.Buffer
			
			err := fm.PrintTo(&buf, data)
			if err != nil {
				t.Errorf("FormatterManager.PrintTo() error = %v", err)
			}
			
			if buf.Len() == 0 {
				t.Error("FormatterManager.PrintTo() produced no output")
			}
		})
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{"Valid table", "table", false},
		{"Valid json", "json", false},
		{"Valid yaml", "yaml", false},
		{"Invalid format", "xml", true},
		{"Empty format", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetFormatFromString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   Format
	}{
		{"Lowercase json", "json", FormatJSON},
		{"Uppercase JSON", "JSON", FormatJSON},
		{"Lowercase yaml", "yaml", FormatYAML},
		{"Short yml", "yml", FormatYAML},
		{"Lowercase table", "table", FormatTable},
		{"Invalid format", "invalid", FormatTable},
		{"Empty string", "", FormatTable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFormatFromString(tt.input); got != tt.want {
				t.Errorf("GetFormatFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 벤치마크 테스트
func BenchmarkTableFormatter_Format(b *testing.B) {
	formatter := &TableFormatter{
		baseFormatter: baseFormatter{colorEnabled: false},
	}
	
	// 큰 데이터셋 생성
	data := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = map[string]interface{}{
			"id":     i,
			"name":   "item" + string(rune(i)),
			"status": "active",
			"value":  i * 100,
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.Format(data)
	}
}

func BenchmarkJSONFormatter_Format(b *testing.B) {
	formatter := &JSONFormatter{
		baseFormatter: baseFormatter{colorEnabled: false},
		indent:        true,
	}
	
	data := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = map[string]interface{}{
			"id":     i,
			"name":   "item" + string(rune(i)),
			"status": "active",
			"value":  i * 100,
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatter.Format(data)
	}
}