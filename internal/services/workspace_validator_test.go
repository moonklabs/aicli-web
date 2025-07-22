package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/aicli/aicli-web/internal/models"
)

func TestWorkspaceValidator_ValidateCreate(t *testing.T) {
	validator := NewWorkspaceValidator()
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *models.CreateWorkspaceRequest
		wantErr bool
		errCode string
	}{
		{
			name: "유효한 생성 요청",
			req: &models.CreateWorkspaceRequest{
				Name:        "valid-workspace",
				ProjectPath: "/tmp/test",
				ClaudeKey:   "sk-ant-test123456789012345678901234567890123456789012",
			},
			wantErr: false,
		},
		{
			name: "빈 이름",
			req: &models.CreateWorkspaceRequest{
				Name:        "",
				ProjectPath: "/tmp/test",
			},
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name: "빈 프로젝트 경로",
			req: &models.CreateWorkspaceRequest{
				Name:        "valid-workspace",
				ProjectPath: "",
			},
			wantErr: true,
			errCode: ErrCodeInvalidPath,
		},
		{
			name: "너무 긴 이름",
			req: &models.CreateWorkspaceRequest{
				Name:        "a" + "b"*99, // 100자 초과
				ProjectPath: "/tmp/test",
			},
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name: "금지된 문자가 포함된 이름",
			req: &models.CreateWorkspaceRequest{
				Name:        "workspace/with/slash",
				ProjectPath: "/tmp/test",
			},
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name: "예약어 이름",
			req: &models.CreateWorkspaceRequest{
				Name:        "admin",
				ProjectPath: "/tmp/test",
			},
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name: "잘못된 Claude API 키 형식",
			req: &models.CreateWorkspaceRequest{
				Name:        "valid-workspace",
				ProjectPath: "/tmp/test",
				ClaudeKey:   "invalid-key",
			},
			wantErr: true,
			errCode: ErrCodeInvalidRequest,
		},
		{
			name: "너무 짧은 Claude API 키",
			req: &models.CreateWorkspaceRequest{
				Name:        "valid-workspace",
				ProjectPath: "/tmp/test",
				ClaudeKey:   "sk-ant-short",
			},
			wantErr: true,
			errCode: ErrCodeInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateCreate(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				var workspaceErr *WorkspaceError
				if assert.ErrorAs(t, err, &workspaceErr) {
					assert.Equal(t, tt.errCode, workspaceErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkspaceValidator_ValidateUpdate(t *testing.T) {
	validator := NewWorkspaceValidator()
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *models.UpdateWorkspaceRequest
		wantErr bool
		errCode string
	}{
		{
			name: "유효한 업데이트 요청",
			req: &models.UpdateWorkspaceRequest{
				Name:        "updated-workspace",
				ProjectPath: "/tmp/updated",
				Status:      models.WorkspaceStatusActive,
			},
			wantErr: false,
		},
		{
			name: "빈 필드들 (업데이트 안함)",
			req: &models.UpdateWorkspaceRequest{
				Name:        "",
				ProjectPath: "",
				ClaudeKey:   "",
			},
			wantErr: false,
		},
		{
			name: "잘못된 이름 (업데이트 시)",
			req: &models.UpdateWorkspaceRequest{
				Name: "workspace/with/slash",
			},
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name: "잘못된 상태",
			req: &models.UpdateWorkspaceRequest{
				Status: models.WorkspaceStatus("invalid"),
			},
			wantErr: true,
			errCode: ErrCodeInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateUpdate(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				var workspaceErr *WorkspaceError
				if assert.ErrorAs(t, err, &workspaceErr) {
					assert.Equal(t, tt.errCode, workspaceErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkspaceValidator_ValidateWorkspace(t *testing.T) {
	validator := NewWorkspaceValidator()
	ctx := context.Background()

	tests := []struct {
		name      string
		workspace *models.Workspace
		wantErr   bool
		errCode   string
	}{
		{
			name: "유효한 워크스페이스",
			workspace: &models.Workspace{
				ID:          "ws123",
				Name:        "valid-workspace",
				ProjectPath: "/tmp/test",
				Status:      models.WorkspaceStatusActive,
				OwnerID:     "user123",
			},
			wantErr: false,
		},
		{
			name:      "nil 워크스페이스",
			workspace: nil,
			wantErr:   true,
			errCode:   ErrCodeInvalidRequest,
		},
		{
			name: "빈 소유자 ID",
			workspace: &models.Workspace{
				ID:          "ws123",
				Name:        "valid-workspace",
				ProjectPath: "/tmp/test",
				Status:      models.WorkspaceStatusActive,
				OwnerID:     "",
			},
			wantErr: true,
			errCode: ErrCodeInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWorkspace(ctx, tt.workspace)

			if tt.wantErr {
				assert.Error(t, err)
				var workspaceErr *WorkspaceError
				if assert.ErrorAs(t, err, &workspaceErr) {
					assert.Equal(t, tt.errCode, workspaceErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkspaceValidator_validateName(t *testing.T) {
	validator := NewWorkspaceValidator()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errCode  string
	}{
		{
			name:    "유효한 이름",
			input:   "valid-workspace",
			wantErr: false,
		},
		{
			name:    "빈 이름",
			input:   "",
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name:    "앞뒤 공백이 있는 이름",
			input:   " workspace ",
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name:    "연속된 공백이 있는 이름",
			input:   "work  space",
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name:    "슬래시가 포함된 이름",
			input:   "work/space",
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name:    "백슬래시가 포함된 이름",
			input:   "work\\space",
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name:    "콜론이 포함된 이름",
			input:   "work:space",
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name:    "예약어 (대소문자 구분 없음)",
			input:   "ADMIN",
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name:    "너무 긴 이름",
			input:   "a" + "b"*99, // 100자 초과
			wantErr: true,
			errCode: ErrCodeInvalidName,
		},
		{
			name:    "최대 길이 이름",
			input:   "a" + "b"*98, // 정확히 100자
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateName(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				var workspaceErr *WorkspaceError
				if assert.ErrorAs(t, err, &workspaceErr) {
					assert.Equal(t, tt.errCode, workspaceErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkspaceValidator_validateClaudeKey(t *testing.T) {
	validator := NewWorkspaceValidator()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		errCode string
	}{
		{
			name:    "빈 키 (선택적)",
			input:   "",
			wantErr: false,
		},
		{
			name:    "유효한 Claude API 키",
			input:   "sk-ant-test123456789012345678901234567890123456789012",
			wantErr: false,
		},
		{
			name:    "잘못된 프리픽스",
			input:   "sk-openai-test123456789012345678901234567890123456789012",
			wantErr: true,
			errCode: ErrCodeInvalidRequest,
		},
		{
			name:    "너무 짧은 키",
			input:   "sk-ant-short",
			wantErr: true,
			errCode: ErrCodeInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateClaudeKey(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				var workspaceErr *WorkspaceError
				if assert.ErrorAs(t, err, &workspaceErr) {
					assert.Equal(t, tt.errCode, workspaceErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkspaceValidator_validateStatus(t *testing.T) {
	validator := NewWorkspaceValidator()

	tests := []struct {
		name    string
		input   models.WorkspaceStatus
		wantErr bool
		errCode string
	}{
		{
			name:    "활성 상태",
			input:   models.WorkspaceStatusActive,
			wantErr: false,
		},
		{
			name:    "비활성 상태",
			input:   models.WorkspaceStatusInactive,
			wantErr: false,
		},
		{
			name:    "아카이브 상태",
			input:   models.WorkspaceStatusArchived,
			wantErr: false,
		},
		{
			name:    "잘못된 상태",
			input:   models.WorkspaceStatus("invalid"),
			wantErr: true,
			errCode: ErrCodeInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateStatus(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				var workspaceErr *WorkspaceError
				if assert.ErrorAs(t, err, &workspaceErr) {
					assert.Equal(t, tt.errCode, workspaceErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkspaceValidator_CanCreateWorkspace(t *testing.T) {
	validator := NewWorkspaceValidator()
	ctx := context.Background()

	tests := []struct {
		name         string
		userID       string
		currentCount int
		wantErr      bool
		errCode      string
	}{
		{
			name:         "생성 가능 (제한 내)",
			userID:       "user123",
			currentCount: 10,
			wantErr:      false,
		},
		{
			name:         "생성 가능 (제한 경계)",
			userID:       "user123",
			currentCount: 49,
			wantErr:      false,
		},
		{
			name:         "생성 불가능 (제한 도달)",
			userID:       "user123",
			currentCount: 50,
			wantErr:      true,
			errCode:      ErrCodeMaxWorkspaces,
		},
		{
			name:         "생성 불가능 (제한 초과)",
			userID:       "user123",
			currentCount: 55,
			wantErr:      true,
			errCode:      ErrCodeMaxWorkspaces,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.CanCreateWorkspace(ctx, tt.userID, tt.currentCount)

			if tt.wantErr {
				assert.Error(t, err)
				var workspaceErr *WorkspaceError
				if assert.ErrorAs(t, err, &workspaceErr) {
					assert.Equal(t, tt.errCode, workspaceErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkspaceValidator_CanActivateWorkspace(t *testing.T) {
	validator := NewWorkspaceValidator()
	ctx := context.Background()

	tests := []struct {
		name      string
		workspace *models.Workspace
		wantErr   bool
		errCode   string
	}{
		{
			name: "활성화 가능 (비활성 상태)",
			workspace: &models.Workspace{
				Status: models.WorkspaceStatusInactive,
			},
			wantErr: false,
		},
		{
			name: "활성화 가능 (이미 활성 상태)",
			workspace: &models.Workspace{
				Status: models.WorkspaceStatusActive,
			},
			wantErr: false,
		},
		{
			name: "활성화 불가능 (아카이브 상태)",
			workspace: &models.Workspace{
				Status: models.WorkspaceStatusArchived,
			},
			wantErr: true,
			errCode: ErrCodeArchived,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.CanActivateWorkspace(ctx, tt.workspace)

			if tt.wantErr {
				assert.Error(t, err)
				var workspaceErr *WorkspaceError
				if assert.ErrorAs(t, err, &workspaceErr) {
					assert.Equal(t, tt.errCode, workspaceErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkspaceValidator_CanDeactivateWorkspace(t *testing.T) {
	validator := NewWorkspaceValidator()
	ctx := context.Background()

	tests := []struct {
		name      string
		workspace *models.Workspace
		wantErr   bool
		errCode   string
	}{
		{
			name: "비활성화 가능 (활성 태스크 없음)",
			workspace: &models.Workspace{
				ActiveTasks: 0,
			},
			wantErr: false,
		},
		{
			name: "비활성화 불가능 (활성 태스크 존재)",
			workspace: &models.Workspace{
				ActiveTasks: 3,
			},
			wantErr: true,
			errCode: ErrCodeResourceBusy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.CanDeactivateWorkspace(ctx, tt.workspace)

			if tt.wantErr {
				assert.Error(t, err)
				var workspaceErr *WorkspaceError
				if assert.ErrorAs(t, err, &workspaceErr) {
					assert.Equal(t, tt.errCode, workspaceErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkspaceValidator_CanDeleteWorkspace(t *testing.T) {
	validator := NewWorkspaceValidator()
	ctx := context.Background()

	tests := []struct {
		name      string
		workspace *models.Workspace
		wantErr   bool
		errCode   string
	}{
		{
			name: "삭제 가능 (활성 태스크 없음)",
			workspace: &models.Workspace{
				ActiveTasks: 0,
			},
			wantErr: false,
		},
		{
			name: "삭제 불가능 (활성 태스크 존재)",
			workspace: &models.Workspace{
				ActiveTasks: 2,
			},
			wantErr: true,
			errCode: ErrCodeResourceBusy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.CanDeleteWorkspace(ctx, tt.workspace)

			if tt.wantErr {
				assert.Error(t, err)
				var workspaceErr *WorkspaceError
				if assert.ErrorAs(t, err, &workspaceErr) {
					assert.Equal(t, tt.errCode, workspaceErr.Code)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}