-- AICode Manager SQLite Schema
-- Version: 001_initial
-- Created: 2025-07-21

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Workspace 테이블
CREATE TABLE IF NOT EXISTS workspaces (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    project_path VARCHAR(500) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    owner_id VARCHAR(50) NOT NULL,
    claude_key TEXT, -- 암호화된 API 키
    active_tasks INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    version INTEGER NOT NULL DEFAULT 1
);

-- Project 테이블
CREATE TABLE IF NOT EXISTS projects (
    id CHAR(36) PRIMARY KEY,
    workspace_id CHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    path VARCHAR(500) NOT NULL,
    description TEXT,
    git_url VARCHAR(500),
    git_branch VARCHAR(100),
    language VARCHAR(50),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    config TEXT, -- JSON 데이터
    git_info TEXT, -- JSON 데이터
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    version INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);

-- Session 테이블
CREATE TABLE IF NOT EXISTS sessions (
    id CHAR(36) PRIMARY KEY,
    project_id CHAR(36) NOT NULL,
    process_id INTEGER,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'idle', 'ending', 'ended', 'error')),
    started_at DATETIME,
    ended_at DATETIME,
    last_active DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT, -- JSON 데이터
    command_count BIGINT NOT NULL DEFAULT 0,
    bytes_in BIGINT NOT NULL DEFAULT 0,
    bytes_out BIGINT NOT NULL DEFAULT 0,
    error_count BIGINT NOT NULL DEFAULT 0,
    max_idle_time BIGINT NOT NULL DEFAULT 1800000000000, -- 30분 (나노초)
    max_lifetime BIGINT NOT NULL DEFAULT 14400000000000, -- 4시간 (나노초)
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Task 테이블
CREATE TABLE IF NOT EXISTS tasks (
    id CHAR(36) PRIMARY KEY,
    session_id CHAR(36) NOT NULL,
    command TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    output TEXT,
    error TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    bytes_in BIGINT NOT NULL DEFAULT 0,
    bytes_out BIGINT NOT NULL DEFAULT 0,
    duration BIGINT NOT NULL DEFAULT 0, -- 밀리초
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- 트리거: updated_at 자동 업데이트
CREATE TRIGGER IF NOT EXISTS update_workspaces_updated_at 
AFTER UPDATE ON workspaces
BEGIN
    UPDATE workspaces SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_projects_updated_at 
AFTER UPDATE ON projects
BEGIN
    UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_sessions_updated_at 
AFTER UPDATE ON sessions
BEGIN
    UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at 
AFTER UPDATE ON tasks
BEGIN
    UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;