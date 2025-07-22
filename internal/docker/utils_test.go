package docker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal name",
			input:    "test-workspace",
			expected: "test-workspace",
		},
		{
			name:     "uppercase conversion",
			input:    "TEST-Workspace",
			expected: "test-workspace",
		},
		{
			name:     "space replacement",
			input:    "test workspace",
			expected: "test-workspace",
		},
		{
			name:     "underscore replacement",
			input:    "test_workspace",
			expected: "test-workspace",
		},
		{
			name:     "special characters removal",
			input:    "test@workspace#123",
			expected: "testworkspace123",
		},
		{
			name:     "leading/trailing hyphens",
			input:    "-test-workspace-",
			expected: "test-workspace",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "default",
		},
		{
			name:     "only special characters",
			input:    "@#$%",
			expected: "default",
		},
		{
			name:     "complex name",
			input:    "My_Test@Workspace#2023!",
			expected: "my-testworkspace2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateContainerName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "valid name",
			input:       "test-container",
			expectError: false,
		},
		{
			name:        "valid name with slash",
			input:       "/test-container",
			expectError: false,
		},
		{
			name:        "valid name with numbers",
			input:       "container-123",
			expectError: false,
		},
		{
			name:        "valid name with dots",
			input:       "test.container.name",
			expectError: false,
		},
		{
			name:        "empty name",
			input:       "",
			expectError: true,
		},
		{
			name:        "name too long",
			input:       string(make([]byte, 254)), // 254 characters, over limit
			expectError: true,
		},
		{
			name:        "invalid character",
			input:       "test@container",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContainerName(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateImageName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "simple image name",
			input:       "alpine",
			expectError: false,
		},
		{
			name:        "image with tag",
			input:       "alpine:latest",
			expectError: false,
		},
		{
			name:        "image with registry",
			input:       "docker.io/alpine:3.14",
			expectError: false,
		},
		{
			name:        "empty name",
			input:       "",
			expectError: true,
		},
		{
			name:        "empty tag",
			input:       "alpine:",
			expectError: true,
		},
		{
			name:        "multiple colons",
			input:       "alpine:latest:extra",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageName(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{
			name:     "bytes",
			input:    512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			input:    1536, // 1.5KB
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			input:    2097152, // 2MB
			expected: "2.0 MB",
		},
		{
			name:     "gigabytes",
			input:    3221225472, // 3GB
			expected: "3.0 GB",
		},
		{
			name:     "zero bytes",
			input:    0,
			expected: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{
			name:     "seconds",
			input:    30 * time.Second,
			expected: "30s",
		},
		{
			name:     "minutes and seconds",
			input:    90 * time.Second, // 1m30s
			expected: "1m30s",
		},
		{
			name:     "hours and minutes",
			input:    3900 * time.Second, // 1h5m
			expected: "1h5m",
		},
		{
			name:     "days and hours",
			input:    25 * time.Hour, // 1d1h
			expected: "1d1h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeLabels(t *testing.T) {
	labels1 := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	
	labels2 := map[string]string{
		"key2": "updated_value2", // override
		"key3": "value3",
	}
	
	labels3 := map[string]string{
		"key4": "value4",
	}

	result := MergeLabels(labels1, labels2, labels3)
	
	expected := map[string]string{
		"key1": "value1",
		"key2": "updated_value2", // overridden value
		"key3": "value3",
		"key4": "value4",
	}

	assert.Equal(t, expected, result)
}

func TestParseResourceLimits(t *testing.T) {
	tests := []struct {
		name           string
		cpu            string
		memory         string
		expectedCPU    float64
		expectedMemory int64
		expectError    bool
	}{
		{
			name:           "valid limits",
			cpu:            "1.5",
			memory:         "512M",
			expectedCPU:    1.5,
			expectedMemory: 512 * 1024 * 1024,
			expectError:    false,
		},
		{
			name:           "memory in GB",
			cpu:            "2.0",
			memory:         "2G",
			expectedCPU:    2.0,
			expectedMemory: 2 * 1024 * 1024 * 1024,
			expectError:    false,
		},
		{
			name:           "memory in KB",
			cpu:            "0.5",
			memory:         "1024K",
			expectedCPU:    0.5,
			expectedMemory: 1024 * 1024,
			expectError:    false,
		},
		{
			name:           "memory with B suffix",
			cpu:            "1.0",
			memory:         "512MB",
			expectedCPU:    1.0,
			expectedMemory: 512 * 1024 * 1024,
			expectError:    false,
		},
		{
			name:           "empty values",
			cpu:            "",
			memory:         "",
			expectedCPU:    0,
			expectedMemory: 0,
			expectError:    false,
		},
		{
			name:        "invalid CPU format",
			cpu:         "invalid",
			memory:      "512M",
			expectError: true,
		},
		{
			name:        "negative CPU",
			cpu:         "-1.0",
			memory:      "512M",
			expectError: true,
		},
		{
			name:        "invalid memory format",
			cpu:         "1.0",
			memory:      "invalid",
			expectError: true,
		},
		{
			name:        "negative memory",
			cpu:         "1.0",
			memory:      "-512M",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, memory, err := ParseResourceLimits(tt.cpu, tt.memory)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCPU, cpu)
				assert.Equal(t, tt.expectedMemory, memory)
			}
		})
	}
}

func TestClient_FilterByLabels(t *testing.T) {
	client := &Client{
		labelPrefix: "aicli",
	}

	labels := map[string]string{
		"aicli.managed":      "true",
		"aicli.workspace.id": "ws-123",
	}

	filters := client.FilterByLabels(labels)
	
	assert.NotNil(t, filters)
	assert.Contains(t, filters, "label")
	assert.Len(t, filters["label"], 2)
	
	expectedFilters := []string{
		"label=aicli.managed=true",
		"label=aicli.workspace.id=ws-123",
	}
	
	for _, expected := range expectedFilters {
		assert.Contains(t, filters["label"], expected)
	}
}

func TestClient_GetManagedResourceFilter(t *testing.T) {
	client := &Client{
		labelPrefix: "aicli",
	}

	filters := client.GetManagedResourceFilter()
	
	assert.NotNil(t, filters)
	assert.Contains(t, filters, "label")
	assert.Len(t, filters["label"], 1)
	assert.Equal(t, "aicli.managed=true", filters["label"][0])
}

func BenchmarkSanitizeName(b *testing.B) {
	testName := "My_Test@Workspace#2023!"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeName(testName)
	}
}

func BenchmarkFormatBytes(b *testing.B) {
	testBytes := int64(1536 * 1024 * 1024) // 1.5GB
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FormatBytes(testBytes)
	}
}

func BenchmarkMergeLabels(b *testing.B) {
	labels1 := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	
	labels2 := map[string]string{
		"key3": "value3",
		"key4": "value4",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MergeLabels(labels1, labels2)
	}
}