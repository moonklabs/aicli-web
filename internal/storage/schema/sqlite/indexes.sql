-- AICode Manager SQLite Indexes
-- Version: 001_initial
-- Created: 2025-07-21

-- Workspace 인덱스
CREATE INDEX IF NOT EXISTS idx_workspace_owner_id ON workspaces(owner_id);
CREATE INDEX IF NOT EXISTS idx_workspace_status ON workspaces(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_workspace_deleted_at ON workspaces(deleted_at);

-- Project 인덱스
CREATE INDEX IF NOT EXISTS idx_project_workspace_id ON projects(workspace_id);
CREATE INDEX IF NOT EXISTS idx_project_status ON projects(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_project_workspace_status ON projects(workspace_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_project_deleted_at ON projects(deleted_at);

-- Session 인덱스
CREATE INDEX IF NOT EXISTS idx_session_project_id ON sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_session_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_session_project_status ON sessions(project_id, status);
CREATE INDEX IF NOT EXISTS idx_session_last_active ON sessions(last_active) WHERE status IN ('active', 'idle');

-- Task 인덱스
CREATE INDEX IF NOT EXISTS idx_task_session_id ON tasks(session_id);
CREATE INDEX IF NOT EXISTS idx_task_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_task_session_status ON tasks(session_id, status);
CREATE INDEX IF NOT EXISTS idx_task_created_at ON tasks(created_at);

-- 복합 인덱스 (자주 사용되는 쿼리 패턴용)
CREATE INDEX IF NOT EXISTS idx_workspace_owner_status ON workspaces(owner_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_session_active ON sessions(project_id, status, last_active) WHERE status IN ('active', 'idle');
CREATE INDEX IF NOT EXISTS idx_task_active ON tasks(session_id, status) WHERE status IN ('pending', 'running');