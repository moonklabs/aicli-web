-- AI CLI Manager 테스트 데이터베이스 초기화 스크립트
-- 통합 테스트를 위한 기본 스키마 및 테스트 데이터 생성

-- 확장 설치
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 테스트용 사용자 및 역할 생성
CREATE ROLE test_admin WITH LOGIN SUPERUSER;
CREATE ROLE test_user WITH LOGIN;
CREATE ROLE test_readonly WITH LOGIN;

-- 기본 테이블 생성 (기존 스키마 기반)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    avatar_url VARCHAR(512),
    is_active BOOLEAN DEFAULT true,
    is_admin BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id VARCHAR(255) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    state JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    path VARCHAR(512) NOT NULL,
    git_url VARCHAR(512),
    git_branch VARCHAR(255) DEFAULT 'main',
    docker_image VARCHAR(255),
    config JSONB DEFAULT '{}',
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    priority VARCHAR(20) DEFAULT 'medium',
    config JSONB DEFAULT '{}',
    result JSONB DEFAULT '{}',
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'idle',
    config JSONB DEFAULT '{}',
    metrics JSONB DEFAULT '{}',
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    container_id VARCHAR(255),
    image_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_heartbeat TIMESTAMP WITH TIME ZONE
);

-- 인덱스 생성
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_workspace_id ON sessions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_workspaces_owner_id ON workspaces(owner_id);
CREATE INDEX IF NOT EXISTS idx_tasks_workspace_id ON tasks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_tasks_session_id ON tasks(session_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_agents_workspace_id ON agents(workspace_id);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);

-- 테스트 데이터 삽입
INSERT INTO users (id, username, email, password_hash, full_name, is_admin) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'testadmin', 'admin@test.com', crypt('testpassword', gen_salt('bf')), 'Test Administrator', true),
    ('550e8400-e29b-41d4-a716-446655440002', 'testuser1', 'user1@test.com', crypt('userpassword', gen_salt('bf')), 'Test User 1', false),
    ('550e8400-e29b-41d4-a716-446655440003', 'testuser2', 'user2@test.com', crypt('userpassword', gen_salt('bf')), 'Test User 2', false),
    ('550e8400-e29b-41d4-a716-446655440004', 'readonly', 'readonly@test.com', crypt('readonly', gen_salt('bf')), 'Read Only User', false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO workspaces (id, name, description, path, git_url, owner_id) VALUES
    ('660e8400-e29b-41d4-a716-446655440001', 'test-workspace-1', 'Primary test workspace', '/test/workspace1', 'https://github.com/test/repo1.git', '550e8400-e29b-41d4-a716-446655440001'),
    ('660e8400-e29b-41d4-a716-446655440002', 'test-workspace-2', 'Secondary test workspace', '/test/workspace2', 'https://github.com/test/repo2.git', '550e8400-e29b-41d4-a716-446655440002'),
    ('660e8400-e29b-41d4-a716-446655440003', 'performance-test-workspace', 'Performance testing workspace', '/test/perf-workspace', 'https://github.com/test/perf-repo.git', '550e8400-e29b-41d4-a716-446655440001')
ON CONFLICT (id) DO NOTHING;

INSERT INTO sessions (id, user_id, workspace_id, config, state, status) VALUES
    ('770e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', '{"max_turns": 10, "tools": ["Read", "Write"]}', '{"status": "active", "turn_count": 0}', 'active'),
    ('770e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440002', '{"max_turns": 5, "tools": ["Read"]}', '{"status": "idle", "turn_count": 0}', 'idle')
ON CONFLICT (id) DO NOTHING;

INSERT INTO tasks (id, name, description, type, status, workspace_id) VALUES
    ('880e8400-e29b-41d4-a716-446655440001', 'Test Task 1', 'First test task', 'integration', 'pending', '660e8400-e29b-41d4-a716-446655440001'),
    ('880e8400-e29b-41d4-a716-446655440002', 'Test Task 2', 'Second test task', 'unit', 'running', '660e8400-e29b-41d4-a716-446655440001'),
    ('880e8400-e29b-41d4-a716-446655440003', 'Performance Test Task', 'Load testing task', 'performance', 'pending', '660e8400-e29b-41d4-a716-446655440003')
ON CONFLICT (id) DO NOTHING;

INSERT INTO agents (id, name, type, status, workspace_id, image_name) VALUES
    ('990e8400-e29b-41d4-a716-446655440001', 'test-agent-1', 'claude-cli', 'idle', '660e8400-e29b-41d4-a716-446655440001', 'aicli/claude-agent:test'),
    ('990e8400-e29b-41d4-a716-446655440002', 'test-agent-2', 'claude-cli', 'running', '660e8400-e29b-41d4-a716-446655440002', 'aicli/claude-agent:test'),
    ('990e8400-e29b-41d4-a716-446655440003', 'perf-agent-1', 'claude-cli', 'idle', '660e8400-e29b-41d4-a716-446655440003', 'aicli/claude-agent:test')
ON CONFLICT (id) DO NOTHING;

-- 권한 설정
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO test_admin;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO test_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO test_readonly;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO test_user;

-- 테스트 완료 로그
INSERT INTO tasks (name, description, type, status, config) VALUES
    ('Database Initialization', 'Test database initialized successfully', 'system', 'completed', '{"timestamp": "' || NOW() || '"}');

-- 통계 정보 업데이트
ANALYZE;