-- RBAC 테이블 스키마 정의
-- 마이그레이션 버전: 002
-- 설명: 역할 기반 접근 제어(RBAC) 시스템을 위한 테이블 생성

-- 1. 역할(Role) 테이블
CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    parent_id TEXT,
    level INTEGER DEFAULT 0 CHECK (level >= 0 AND level <= 10),
    is_system BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    version INTEGER DEFAULT 1,
    FOREIGN KEY (parent_id) REFERENCES roles(id) ON DELETE SET NULL
);

-- 2. 권한(Permission) 테이블
CREATE TABLE IF NOT EXISTS permissions (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    resource_type TEXT NOT NULL CHECK (resource_type IN ('workspace', 'project', 'session', 'task', 'user', 'system')),
    action TEXT NOT NULL CHECK (action IN ('create', 'read', 'update', 'delete', 'execute', 'manage')),
    effect TEXT NOT NULL DEFAULT 'allow' CHECK (effect IN ('allow', 'deny')),
    conditions TEXT, -- JSON 형태의 조건
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    version INTEGER DEFAULT 1
);

-- 3. 리소스(Resource) 테이블  
CREATE TABLE IF NOT EXISTS resources (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('workspace', 'project', 'session', 'task', 'user', 'system')),
    identifier TEXT NOT NULL, -- 리소스 고유 식별자
    parent_id TEXT,
    path TEXT, -- 계층 경로
    attributes TEXT, -- JSON 형태의 추가 속성
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    version INTEGER DEFAULT 1,
    FOREIGN KEY (parent_id) REFERENCES resources(id) ON DELETE SET NULL,
    UNIQUE(type, identifier)
);

-- 4. 사용자 그룹(UserGroup) 테이블
CREATE TABLE IF NOT EXISTS user_groups (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL,
    description TEXT,
    parent_id TEXT,
    type TEXT NOT NULL CHECK (type IN ('organization', 'department', 'team', 'project')),
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    version INTEGER DEFAULT 1,
    FOREIGN KEY (parent_id) REFERENCES user_groups(id) ON DELETE SET NULL
);

-- 5. 역할-권한 연결(RolePermission) 테이블
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id TEXT NOT NULL,
    permission_id TEXT NOT NULL,
    effect TEXT NOT NULL DEFAULT 'allow' CHECK (effect IN ('allow', 'deny')),
    conditions TEXT, -- 역할별 추가 조건
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- 6. 사용자-역할 연결(UserRole) 테이블  
CREATE TABLE IF NOT EXISTS user_roles (
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    assigned_by TEXT NOT NULL,
    resource_id TEXT, -- 특정 리소스에 대한 역할
    expires_at DATETIME,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id, COALESCE(resource_id, '')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_by) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE
);

-- 7. 사용자 그룹 멤버(UserGroupMember) 테이블
CREATE TABLE IF NOT EXISTS user_group_members (
    user_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('member', 'admin', 'owner')),
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, group_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES user_groups(id) ON DELETE CASCADE
);

-- 8. 그룹-역할 연결(GroupRole) 테이블
CREATE TABLE IF NOT EXISTS group_roles (
    group_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    resource_id TEXT, -- 특정 리소스에 대한 그룹 역할
    assigned_by TEXT NOT NULL,
    expires_at DATETIME,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, role_id, COALESCE(resource_id, '')),
    FOREIGN KEY (group_id) REFERENCES user_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_by) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE
);

-- 인덱스 생성
CREATE INDEX IF NOT EXISTS idx_roles_parent_id ON roles(parent_id);
CREATE INDEX IF NOT EXISTS idx_roles_level ON roles(level);
CREATE INDEX IF NOT EXISTS idx_roles_is_active ON roles(is_active);
CREATE INDEX IF NOT EXISTS idx_roles_is_system ON roles(is_system);

CREATE INDEX IF NOT EXISTS idx_permissions_resource_type ON permissions(resource_type);
CREATE INDEX IF NOT EXISTS idx_permissions_action ON permissions(action);
CREATE INDEX IF NOT EXISTS idx_permissions_effect ON permissions(effect);
CREATE INDEX IF NOT EXISTS idx_permissions_is_active ON permissions(is_active);

CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(type);
CREATE INDEX IF NOT EXISTS idx_resources_identifier ON resources(identifier);
CREATE INDEX IF NOT EXISTS idx_resources_parent_id ON resources(parent_id);
CREATE INDEX IF NOT EXISTS idx_resources_is_active ON resources(is_active);

CREATE INDEX IF NOT EXISTS idx_user_groups_parent_id ON user_groups(parent_id);
CREATE INDEX IF NOT EXISTS idx_user_groups_type ON user_groups(type);
CREATE INDEX IF NOT EXISTS idx_user_groups_is_active ON user_groups(is_active);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions(permission_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_effect ON role_permissions(effect);

CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_resource_id ON user_roles(resource_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_is_active ON user_roles(is_active);
CREATE INDEX IF NOT EXISTS idx_user_roles_expires_at ON user_roles(expires_at);

CREATE INDEX IF NOT EXISTS idx_user_group_members_user_id ON user_group_members(user_id);
CREATE INDEX IF NOT EXISTS idx_user_group_members_group_id ON user_group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_user_group_members_role ON user_group_members(role);
CREATE INDEX IF NOT EXISTS idx_user_group_members_is_active ON user_group_members(is_active);

CREATE INDEX IF NOT EXISTS idx_group_roles_group_id ON group_roles(group_id);
CREATE INDEX IF NOT EXISTS idx_group_roles_role_id ON group_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_group_roles_resource_id ON group_roles(resource_id);
CREATE INDEX IF NOT EXISTS idx_group_roles_is_active ON group_roles(is_active);
CREATE INDEX IF NOT EXISTS idx_group_roles_expires_at ON group_roles(expires_at);

-- 업데이트 트리거 생성 (updated_at 자동 갱신)
CREATE TRIGGER IF NOT EXISTS update_roles_updated_at
    AFTER UPDATE ON roles
BEGIN
    UPDATE roles SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_permissions_updated_at
    AFTER UPDATE ON permissions
BEGIN
    UPDATE permissions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_resources_updated_at
    AFTER UPDATE ON resources
BEGIN
    UPDATE resources SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_user_groups_updated_at
    AFTER UPDATE ON user_groups
BEGIN
    UPDATE user_groups SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_role_permissions_updated_at
    AFTER UPDATE ON role_permissions
BEGIN
    UPDATE role_permissions SET updated_at = CURRENT_TIMESTAMP 
    WHERE role_id = NEW.role_id AND permission_id = NEW.permission_id;
END;

CREATE TRIGGER IF NOT EXISTS update_user_roles_updated_at
    AFTER UPDATE ON user_roles
BEGIN
    UPDATE user_roles SET updated_at = CURRENT_TIMESTAMP 
    WHERE user_id = NEW.user_id AND role_id = NEW.role_id AND COALESCE(resource_id, '') = COALESCE(NEW.resource_id, '');
END;

CREATE TRIGGER IF NOT EXISTS update_user_group_members_updated_at
    AFTER UPDATE ON user_group_members
BEGIN
    UPDATE user_group_members SET updated_at = CURRENT_TIMESTAMP 
    WHERE user_id = NEW.user_id AND group_id = NEW.group_id;
END;

CREATE TRIGGER IF NOT EXISTS update_group_roles_updated_at
    AFTER UPDATE ON group_roles
BEGIN
    UPDATE group_roles SET updated_at = CURRENT_TIMESTAMP 
    WHERE group_id = NEW.group_id AND role_id = NEW.role_id AND COALESCE(resource_id, '') = COALESCE(NEW.resource_id, '');
END;

-- 시스템 기본 역할 및 권한 데이터 삽입
INSERT OR IGNORE INTO roles (id, name, description, level, is_system, is_active) VALUES
    ('role-super-admin', 'super_admin', '시스템 최고 관리자', 0, TRUE, TRUE),
    ('role-admin', 'admin', '관리자', 1, TRUE, TRUE),
    ('role-user', 'user', '일반 사용자', 2, TRUE, TRUE),
    ('role-viewer', 'viewer', '읽기 전용 사용자', 3, TRUE, TRUE),
    ('role-workspace-owner', 'workspace_owner', '워크스페이스 소유자', 2, TRUE, TRUE),
    ('role-project-manager', 'project_manager', '프로젝트 관리자', 3, TRUE, TRUE);

-- 기본 권한 정의
INSERT OR IGNORE INTO permissions (id, name, description, resource_type, action, effect, is_active) VALUES
    -- 시스템 권한
    ('perm-system-manage', 'system_manage', '시스템 전체 관리', 'system', 'manage', 'allow', TRUE),
    ('perm-system-read', 'system_read', '시스템 정보 조회', 'system', 'read', 'allow', TRUE),
    
    -- 사용자 권한
    ('perm-user-create', 'user_create', '사용자 생성', 'user', 'create', 'allow', TRUE),
    ('perm-user-read', 'user_read', '사용자 정보 조회', 'user', 'read', 'allow', TRUE),
    ('perm-user-update', 'user_update', '사용자 정보 수정', 'user', 'update', 'allow', TRUE),
    ('perm-user-delete', 'user_delete', '사용자 삭제', 'user', 'delete', 'allow', TRUE),
    ('perm-user-manage', 'user_manage', '사용자 전체 관리', 'user', 'manage', 'allow', TRUE),
    
    -- 워크스페이스 권한
    ('perm-workspace-create', 'workspace_create', '워크스페이스 생성', 'workspace', 'create', 'allow', TRUE),
    ('perm-workspace-read', 'workspace_read', '워크스페이스 조회', 'workspace', 'read', 'allow', TRUE),
    ('perm-workspace-update', 'workspace_update', '워크스페이스 수정', 'workspace', 'update', 'allow', TRUE),
    ('perm-workspace-delete', 'workspace_delete', '워크스페이스 삭제', 'workspace', 'delete', 'allow', TRUE),
    ('perm-workspace-manage', 'workspace_manage', '워크스페이스 관리', 'workspace', 'manage', 'allow', TRUE),
    
    -- 프로젝트 권한
    ('perm-project-create', 'project_create', '프로젝트 생성', 'project', 'create', 'allow', TRUE),
    ('perm-project-read', 'project_read', '프로젝트 조회', 'project', 'read', 'allow', TRUE),
    ('perm-project-update', 'project_update', '프로젝트 수정', 'project', 'update', 'allow', TRUE),
    ('perm-project-delete', 'project_delete', '프로젝트 삭제', 'project', 'delete', 'allow', TRUE),
    ('perm-project-execute', 'project_execute', '프로젝트 실행', 'project', 'execute', 'allow', TRUE),
    ('perm-project-manage', 'project_manage', '프로젝트 관리', 'project', 'manage', 'allow', TRUE),
    
    -- 세션 권한
    ('perm-session-create', 'session_create', '세션 생성', 'session', 'create', 'allow', TRUE),
    ('perm-session-read', 'session_read', '세션 조회', 'session', 'read', 'allow', TRUE),
    ('perm-session-update', 'session_update', '세션 수정', 'session', 'update', 'allow', TRUE),
    ('perm-session-delete', 'session_delete', '세션 삭제', 'session', 'delete', 'allow', TRUE),
    ('perm-session-execute', 'session_execute', '세션 실행', 'session', 'execute', 'allow', TRUE),
    
    -- 태스크 권한  
    ('perm-task-create', 'task_create', '태스크 생성', 'task', 'create', 'allow', TRUE),
    ('perm-task-read', 'task_read', '태스크 조회', 'task', 'read', 'allow', TRUE),
    ('perm-task-update', 'task_update', '태스크 수정', 'task', 'update', 'allow', TRUE),
    ('perm-task-delete', 'task_delete', '태스크 삭제', 'task', 'delete', 'allow', TRUE),
    ('perm-task-execute', 'task_execute', '태스크 실행', 'task', 'execute', 'allow', TRUE);

-- 역할별 권한 할당
-- super_admin: 모든 권한
INSERT OR IGNORE INTO role_permissions (role_id, permission_id, effect) 
SELECT 'role-super-admin', id, 'allow' FROM permissions WHERE is_active = TRUE;

-- admin: 사용자 관리 및 시스템 조회 권한
INSERT OR IGNORE INTO role_permissions (role_id, permission_id, effect) VALUES
    ('role-admin', 'perm-system-read', 'allow'),
    ('role-admin', 'perm-user-create', 'allow'),
    ('role-admin', 'perm-user-read', 'allow'),
    ('role-admin', 'perm-user-update', 'allow'),
    ('role-admin', 'perm-user-delete', 'allow'),
    ('role-admin', 'perm-workspace-read', 'allow'),
    ('role-admin', 'perm-workspace-manage', 'allow'),
    ('role-admin', 'perm-project-read', 'allow'),
    ('role-admin', 'perm-project-manage', 'allow');

-- user: 기본 사용자 권한
INSERT OR IGNORE INTO role_permissions (role_id, permission_id, effect) VALUES
    ('role-user', 'perm-workspace-create', 'allow'),
    ('role-user', 'perm-workspace-read', 'allow'),
    ('role-user', 'perm-workspace-update', 'allow'),
    ('role-user', 'perm-project-create', 'allow'),
    ('role-user', 'perm-project-read', 'allow'),
    ('role-user', 'perm-project-update', 'allow'),
    ('role-user', 'perm-project-execute', 'allow'),
    ('role-user', 'perm-session-create', 'allow'),
    ('role-user', 'perm-session-read', 'allow'),
    ('role-user', 'perm-session-update', 'allow'),
    ('role-user', 'perm-session-execute', 'allow'),
    ('role-user', 'perm-task-create', 'allow'),
    ('role-user', 'perm-task-read', 'allow'),
    ('role-user', 'perm-task-update', 'allow'),
    ('role-user', 'perm-task-execute', 'allow');

-- viewer: 읽기 전용 권한
INSERT OR IGNORE INTO role_permissions (role_id, permission_id, effect) VALUES
    ('role-viewer', 'perm-workspace-read', 'allow'),
    ('role-viewer', 'perm-project-read', 'allow'),
    ('role-viewer', 'perm-session-read', 'allow'),
    ('role-viewer', 'perm-task-read', 'allow');

-- workspace_owner: 워크스페이스 소유자 권한
INSERT OR IGNORE INTO role_permissions (role_id, permission_id, effect) VALUES
    ('role-workspace-owner', 'perm-workspace-read', 'allow'),
    ('role-workspace-owner', 'perm-workspace-update', 'allow'),
    ('role-workspace-owner', 'perm-workspace-delete', 'allow'),
    ('role-workspace-owner', 'perm-workspace-manage', 'allow'),
    ('role-workspace-owner', 'perm-project-create', 'allow'),
    ('role-workspace-owner', 'perm-project-read', 'allow'),
    ('role-workspace-owner', 'perm-project-update', 'allow'),
    ('role-workspace-owner', 'perm-project-delete', 'allow'),
    ('role-workspace-owner', 'perm-project-execute', 'allow'),
    ('role-workspace-owner', 'perm-project-manage', 'allow');

-- project_manager: 프로젝트 관리자 권한  
INSERT OR IGNORE INTO role_permissions (role_id, permission_id, effect) VALUES
    ('role-project-manager', 'perm-project-read', 'allow'),
    ('role-project-manager', 'perm-project-update', 'allow'),
    ('role-project-manager', 'perm-project-execute', 'allow'),
    ('role-project-manager', 'perm-project-manage', 'allow'),
    ('role-project-manager', 'perm-session-create', 'allow'),
    ('role-project-manager', 'perm-session-read', 'allow'),
    ('role-project-manager', 'perm-session-update', 'allow'),
    ('role-project-manager', 'perm-session-delete', 'allow'),
    ('role-project-manager', 'perm-session-execute', 'allow'),
    ('role-project-manager', 'perm-task-create', 'allow'),
    ('role-project-manager', 'perm-task-read', 'allow'),
    ('role-project-manager', 'perm-task-update', 'allow'),
    ('role-project-manager', 'perm-task-delete', 'allow'),
    ('role-project-manager', 'perm-task-execute', 'allow');