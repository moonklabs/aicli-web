# 보안 가이드

## 🔐 개요

AICode Manager는 다층 보안 아키텍처를 통해 사용자 데이터와 시스템을 보호합니다. 이 문서는 보안 구현 방법과 모범 사례를 설명합니다.

## 🔑 인증 및 권한 관리

### Supabase Auth 통합

#### 1. Supabase 프로젝트 설정
```bash
# Supabase 프로젝트 생성 후 환경 변수 설정
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key
SUPABASE_SERVICE_KEY=your-service-key
```

#### 2. 인증 미들웨어
```python
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from supabase import create_client, Client
import jwt

security = HTTPBearer()

class AuthService:
    def __init__(self):
        self.supabase: Client = create_client(
            supabase_url=os.getenv("SUPABASE_URL"),
            supabase_key=os.getenv("SUPABASE_SERVICE_KEY")
        )
    
    async def verify_token(
        self, 
        credentials: HTTPAuthorizationCredentials = Depends(security)
    ) -> dict:
        """JWT 토큰 검증"""
        token = credentials.credentials
        
        try:
            # Supabase JWT 검증
            payload = jwt.decode(
                token,
                options={"verify_signature": False}  # Supabase가 검증
            )
            
            # 사용자 정보 조회
            user = self.supabase.auth.get_user(token)
            if not user:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Invalid authentication credentials"
                )
            
            return user
            
        except Exception as e:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=str(e)
            )

# 의존성 주입
auth_service = AuthService()

@app.get("/api/protected")
async def protected_route(user=Depends(auth_service.verify_token)):
    return {"user": user}
```

### 역할 기반 접근 제어 (RBAC)

#### 1. 사용자 역할 정의
```python
from enum import Enum

class UserRole(Enum):
    ADMIN = "admin"
    USER = "user"
    VIEWER = "viewer"

class Permission(Enum):
    CREATE_WORKSPACE = "create_workspace"
    DELETE_WORKSPACE = "delete_workspace"
    EXECUTE_TASK = "execute_task"
    VIEW_LOGS = "view_logs"
    MANAGE_USERS = "manage_users"

# 역할별 권한 매핑
ROLE_PERMISSIONS = {
    UserRole.ADMIN: [
        Permission.CREATE_WORKSPACE,
        Permission.DELETE_WORKSPACE,
        Permission.EXECUTE_TASK,
        Permission.VIEW_LOGS,
        Permission.MANAGE_USERS
    ],
    UserRole.USER: [
        Permission.CREATE_WORKSPACE,
        Permission.EXECUTE_TASK,
        Permission.VIEW_LOGS
    ],
    UserRole.VIEWER: [
        Permission.VIEW_LOGS
    ]
}
```

#### 2. 권한 확인 데코레이터
```python
from functools import wraps

def require_permission(permission: Permission):
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, user=None, **kwargs):
            if not user:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Authentication required"
                )
            
            user_role = UserRole(user.get("role", "viewer"))
            if permission not in ROLE_PERMISSIONS.get(user_role, []):
                raise HTTPException(
                    status_code=status.HTTP_403_FORBIDDEN,
                    detail="Insufficient permissions"
                )
            
            return await func(*args, user=user, **kwargs)
        return wrapper
    return decorator

# 사용 예
@app.post("/api/workspaces")
@require_permission(Permission.CREATE_WORKSPACE)
async def create_workspace(
    workspace_data: WorkspaceCreate,
    user=Depends(auth_service.verify_token)
):
    # 워크스페이스 생성 로직
    pass
```

## 🛡️ 컨테이너 보안

### 1. Docker 보안 설정

#### 보안 강화된 Dockerfile
```dockerfile
# 비특권 사용자 실행
FROM python:3.11-slim

# 보안 업데이트
RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# 비특권 사용자 생성
RUN groupadd -r appuser && useradd -r -g appuser appuser

# 애플리케이션 디렉토리
WORKDIR /app
RUN chown appuser:appuser /app

# 의존성 설치 (루트 권한)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 애플리케이션 코드 복사
COPY --chown=appuser:appuser . .

# 비특권 사용자로 전환
USER appuser

# 읽기 전용 파일시스템
RUN chmod -R 555 /app

# 헬스체크
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8000/health')"

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

#### Docker Compose 보안 설정
```yaml
services:
  api:
    security_opt:
      - no-new-privileges:true
      - apparmor:docker-default
      - seccomp:unconfined
    cap_drop:
      - ALL
    cap_add:
      - CHOWN
      - SETUID
      - SETGID
    read_only: true
    tmpfs:
      - /tmp
      - /run
    volumes:
      - ./app:/app:ro
    environment:
      - PYTHONDONTWRITEBYTECODE=1
```

### 2. 컨테이너 격리

#### 네트워크 격리
```yaml
networks:
  frontend:
    driver: bridge
    internal: false
  backend:
    driver: bridge
    internal: true
  database:
    driver: bridge
    internal: true

services:
  nginx:
    networks:
      - frontend
      - backend
  
  api:
    networks:
      - backend
      - database
  
  postgres:
    networks:
      - database
```

#### 리소스 제한
```yaml
services:
  workspace:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
    ulimits:
      nproc: 65535
      nofile:
        soft: 20000
        hard: 40000
```

## 🔒 데이터 보안

### 1. 암호화

#### 환경 변수 암호화
```python
from cryptography.fernet import Fernet
import base64
import os

class SecretManager:
    def __init__(self):
        # 마스터 키는 환경 변수 또는 KMS에서 가져옴
        master_key = os.getenv("MASTER_ENCRYPTION_KEY")
        if not master_key:
            raise ValueError("Master encryption key not found")
        
        self.cipher = Fernet(master_key.encode())
    
    def encrypt_secret(self, plaintext: str) -> str:
        """비밀 정보 암호화"""
        encrypted = self.cipher.encrypt(plaintext.encode())
        return base64.urlsafe_b64encode(encrypted).decode()
    
    def decrypt_secret(self, ciphertext: str) -> str:
        """비밀 정보 복호화"""
        encrypted = base64.urlsafe_b64decode(ciphertext.encode())
        decrypted = self.cipher.decrypt(encrypted)
        return decrypted.decode()

# 사용 예
secret_manager = SecretManager()

# Claude OAuth 토큰 암호화 저장
encrypted_token = secret_manager.encrypt_secret(oauth_token)
store_in_database(encrypted_token)

# 사용 시 복호화
decrypted_token = secret_manager.decrypt_secret(encrypted_token)
```

#### 데이터베이스 암호화
```sql
-- PostgreSQL 투명한 데이터 암호화 (TDE)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 민감한 데이터 암호화
CREATE TABLE user_secrets (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    secret_name VARCHAR(255),
    secret_value BYTEA,  -- 암호화된 데이터
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 데이터 삽입 시 암호화
INSERT INTO user_secrets (user_id, secret_name, secret_value)
VALUES (
    'user-uuid',
    'api_key',
    pgp_sym_encrypt('actual-secret-value', 'encryption-password')
);

-- 데이터 조회 시 복호화
SELECT 
    secret_name,
    pgp_sym_decrypt(secret_value, 'encryption-password') as decrypted_value
FROM user_secrets
WHERE user_id = 'user-uuid';
```

### 2. 민감 정보 관리

#### Secrets 스캐닝
```python
import re
from typing import List, Dict

class SecretScanner:
    """코드에서 민감 정보 검출"""
    
    PATTERNS = {
        'aws_access_key': r'AKIA[0-9A-Z]{16}',
        'aws_secret_key': r'[0-9a-zA-Z/+=]{40}',
        'api_key': r'api[_-]?key[_-]?=[\'"]\w+[\'"]',
        'password': r'password[_-]?=[\'"]\w+[\'"]',
        'jwt_token': r'eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*',
        'private_key': r'-----BEGIN (RSA|EC|DSA) PRIVATE KEY-----'
    }
    
    @classmethod
    def scan_content(cls, content: str) -> List[Dict[str, str]]:
        """콘텐츠에서 비밀 정보 검출"""
        findings = []
        
        for secret_type, pattern in cls.PATTERNS.items():
            matches = re.finditer(pattern, content, re.IGNORECASE)
            for match in matches:
                findings.append({
                    'type': secret_type,
                    'match': match.group(0)[:20] + '...',  # 일부만 표시
                    'position': match.start()
                })
        
        return findings

# Git pre-commit hook 예제
def pre_commit_secret_check(files: List[str]):
    scanner = SecretScanner()
    
    for file_path in files:
        with open(file_path, 'r') as f:
            content = f.read()
            findings = scanner.scan_content(content)
            
            if findings:
                print(f"⚠️  Secrets detected in {file_path}:")
                for finding in findings:
                    print(f"  - {finding['type']} at position {finding['position']}")
                
                return False  # 커밋 차단
    
    return True
```

## 🚨 보안 모니터링

### 1. 감사 로깅

```python
import json
from datetime import datetime
from typing import Any, Dict

class AuditLogger:
    """보안 이벤트 감사 로깅"""
    
    def __init__(self, logger):
        self.logger = logger
    
    def log_event(
        self,
        event_type: str,
        user_id: str,
        resource_type: str,
        resource_id: str,
        action: str,
        result: str,
        metadata: Dict[str, Any] = None
    ):
        """감사 이벤트 기록"""
        event = {
            'timestamp': datetime.utcnow().isoformat(),
            'event_type': event_type,
            'user_id': user_id,
            'resource_type': resource_type,
            'resource_id': resource_id,
            'action': action,
            'result': result,
            'metadata': metadata or {},
            'ip_address': self._get_client_ip(),
            'user_agent': self._get_user_agent()
        }
        
        self.logger.info(f"AUDIT: {json.dumps(event)}")
        
        # 데이터베이스에도 저장
        self._store_audit_event(event)
    
    def log_security_event(
        self,
        severity: str,
        event_type: str,
        description: str,
        metadata: Dict[str, Any] = None
    ):
        """보안 이벤트 기록"""
        event = {
            'timestamp': datetime.utcnow().isoformat(),
            'severity': severity,  # critical, high, medium, low
            'event_type': event_type,
            'description': description,
            'metadata': metadata or {}
        }
        
        self.logger.warning(f"SECURITY: {json.dumps(event)}")
        
        # 심각도가 높은 경우 알림
        if severity in ['critical', 'high']:
            self._send_security_alert(event)

# 사용 예
audit_logger = AuditLogger(logger)

# API 호출 감사
@app.post("/api/workspaces/{workspace_id}/tasks")
async def create_task(workspace_id: str, user=Depends(get_current_user)):
    try:
        # 작업 생성 로직
        task = create_task_logic(workspace_id)
        
        # 감사 로그
        audit_logger.log_event(
            event_type='task_creation',
            user_id=user['id'],
            resource_type='workspace',
            resource_id=workspace_id,
            action='create_task',
            result='success',
            metadata={'task_id': task.id}
        )
        
        return task
    
    except Exception as e:
        audit_logger.log_event(
            event_type='task_creation',
            user_id=user['id'],
            resource_type='workspace',
            resource_id=workspace_id,
            action='create_task',
            result='failure',
            metadata={'error': str(e)}
        )
        raise
```

### 2. 침입 탐지

```python
from collections import defaultdict
from datetime import datetime, timedelta

class IntrusionDetector:
    """비정상 행동 탐지"""
    
    def __init__(self):
        self.failed_attempts = defaultdict(list)
        self.api_calls = defaultdict(list)
    
    def check_brute_force(self, user_id: str, ip_address: str) -> bool:
        """무차별 대입 공격 탐지"""
        key = f"{user_id}:{ip_address}"
        current_time = datetime.utcnow()
        
        # 최근 5분간의 실패 시도
        recent_attempts = [
            attempt for attempt in self.failed_attempts[key]
            if current_time - attempt < timedelta(minutes=5)
        ]
        
        # 5분에 5회 이상 실패 시 차단
        if len(recent_attempts) >= 5:
            audit_logger.log_security_event(
                severity='high',
                event_type='brute_force_detected',
                description=f'Multiple failed login attempts from {ip_address}',
                metadata={'user_id': user_id, 'attempts': len(recent_attempts)}
            )
            return True
        
        return False
    
    def check_rate_limit_abuse(self, user_id: str, endpoint: str) -> bool:
        """API 남용 탐지"""
        key = f"{user_id}:{endpoint}"
        current_time = datetime.utcnow()
        
        # 최근 1분간의 API 호출
        recent_calls = [
            call for call in self.api_calls[key]
            if current_time - call < timedelta(minutes=1)
        ]
        
        # 1분에 100회 이상 호출 시 의심
        if len(recent_calls) >= 100:
            audit_logger.log_security_event(
                severity='medium',
                event_type='rate_limit_abuse',
                description=f'Excessive API calls to {endpoint}',
                metadata={'user_id': user_id, 'calls': len(recent_calls)}
            )
            return True
        
        return False
```

## 🔥 보안 모범 사례

### 1. 최소 권한 원칙
- 각 서비스는 필요한 최소한의 권한만 부여
- 정기적인 권한 검토 및 정리

### 2. 보안 업데이트
```bash
# 자동 보안 업데이트 스크립트
#!/bin/bash
set -e

# Python 패키지 업데이트
pip list --outdated --format=json | \
  jq -r '.[] | .name' | \
  xargs -n1 pip install -U

# Docker 이미지 업데이트
docker-compose pull
docker-compose build --no-cache
docker-compose up -d

# 시스템 패키지 업데이트
apt-get update && apt-get upgrade -y
```

### 3. 보안 헤더 설정
```python
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from starlette.middleware.base import BaseHTTPMiddleware

class SecurityHeadersMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request, call_next):
        response = await call_next(request)
        
        # 보안 헤더 추가
        response.headers['X-Content-Type-Options'] = 'nosniff'
        response.headers['X-Frame-Options'] = 'DENY'
        response.headers['X-XSS-Protection'] = '1; mode=block'
        response.headers['Strict-Transport-Security'] = 'max-age=31536000; includeSubDomains'
        response.headers['Content-Security-Policy'] = "default-src 'self'"
        response.headers['Referrer-Policy'] = 'strict-origin-when-cross-origin'
        response.headers['Permissions-Policy'] = 'geolocation=(), microphone=(), camera=()'
        
        return response

app.add_middleware(SecurityHeadersMiddleware)
```

### 4. 정기 보안 점검
- 주간 취약점 스캔
- 월간 침투 테스트
- 분기별 보안 감사

## 📊 보안 대시보드

실시간 보안 모니터링을 위한 Grafana 대시보드 구성:

1. **인증 지표**
   - 로그인 성공/실패율
   - 활성 세션 수
   - 비정상 로그인 시도

2. **API 보안**
   - Rate limit 위반
   - 권한 거부 횟수
   - 의심스러운 API 패턴

3. **시스템 보안**
   - 컨테이너 보안 이벤트
   - 파일 시스템 변경
   - 네트워크 이상 감지