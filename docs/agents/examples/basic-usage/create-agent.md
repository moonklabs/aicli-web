# 에이전트 생성 가이드

다양한 타입과 설정으로 에이전트를 생성하는 방법을 설명합니다.

## 🎯 기본 에이전트 생성

### 1. 표준 에이전트 생성

가장 기본적인 표준 타입 에이전트를 생성하는 예제입니다.

```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "standard-agent",
    "project_id": "my-project",
    "agent_type": "standard",
    "description": "표준 에이전트 예제"
  }'
```

**응답 예시:**
```json
{
  "id": "agent-std-001",
  "name": "standard-agent",
  "project_id": "my-project",
  "agent_type": "standard",
  "status": "created",
  "description": "표준 에이전트 예제",
  "created_at": "2025-07-30T15:30:00Z"
}
```

### 2. GPU 가속 에이전트 생성

GPU를 사용하는 특수한 작업을 위한 에이전트입니다.

```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "gpu-agent",
    "project_id": "ml-project",
    "agent_type": "gpu",
    "description": "머신러닝 작업용 GPU 에이전트",
    "config": {
      "resources": {
        "gpu": "1",
        "memory": "8Gi",
        "cpu": "4"
      }
    }
  }'
```

### 3. 메모리 최적화 에이전트 생성

대량의 메모리가 필요한 작업을 위한 에이전트입니다.

```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "memory-optimized-agent",
    "project_id": "data-processing",
    "agent_type": "memory_optimized",
    "description": "대용량 데이터 처리용 에이전트",
    "config": {
      "resources": {
        "memory": "16Gi",
        "cpu": "2"
      }
    }
  }'
```

## ⚙️ 고급 설정으로 에이전트 생성

### 1. 환경 변수가 있는 에이전트

특정 환경 변수가 설정된 에이전트를 생성합니다.

```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "env-configured-agent",
    "project_id": "webapp",
    "agent_type": "standard",
    "description": "환경 변수 설정된 에이전트",
    "config": {
      "environment": {
        "NODE_ENV": "production",
        "DATABASE_URL": "postgres://user:pass@db:5432/app",
        "API_KEY": "your-api-key",
        "DEBUG": "false",
        "WORKER_PROCESSES": "4"
      }
    }
  }'
```

### 2. Git 저장소 연동 에이전트

특정 Git 저장소와 브랜치를 지정한 에이전트입니다.

```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "git-integrated-agent",
    "project_id": "feature-development",
    "agent_type": "standard",
    "description": "Git 저장소 연동 에이전트",
    "config": {
      "git": {
        "repository": "https://github.com/your-org/your-repo.git",
        "branch": "feature/new-feature",
        "clone_depth": 1
      }
    }
  }'
```

### 3. 네트워크 격리 에이전트

보안이 중요한 작업을 위해 네트워크가 격리된 에이전트입니다.

```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "isolated-agent",
    "project_id": "secure-processing",
    "agent_type": "standard",
    "description": "네트워크 격리된 보안 에이전트",
    "config": {
      "security": {
        "network_isolation": true,
        "internet_access": false,
        "allowed_domains": ["internal.company.com"]
      },
      "resources": {
        "cpu": "1",
        "memory": "2Gi"
      }
    }
  }'
```

## 🔧 스크립트로 에이전트 생성

### Bash 스크립트 예제

여러 에이전트를 자동으로 생성하는 스크립트입니다.

```bash
#!/bin/bash
# create-agents.sh

API_BASE="http://localhost:8080/api/v1"
PROJECT_ID="batch-creation-demo"

# 에이전트 설정 배열
declare -a AGENTS=(
    "build-agent:standard:빌드 작업용 에이전트"
    "test-agent:standard:테스트 실행용 에이전트"
    "deploy-agent:standard:배포 작업용 에이전트"
    "monitor-agent:memory_optimized:모니터링용 에이전트"
)

echo "프로젝트 '$PROJECT_ID'에 에이전트들을 생성합니다..."

for agent_config in "${AGENTS[@]}"; do
    IFS=':' read -r name type description <<< "$agent_config"
    
    echo "에이전트 생성 중: $name ($type)"
    
    response=$(curl -s -X POST "$API_BASE/agents" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"$name\",
            \"project_id\": \"$PROJECT_ID\",
            \"agent_type\": \"$type\",
            \"description\": \"$description\"
        }")
    
    agent_id=$(echo "$response" | jq -r '.id')
    
    if [ "$agent_id" != "null" ] && [ "$agent_id" != "" ]; then
        echo "✅ 에이전트 생성 성공: $name (ID: $agent_id)"
        
        # 에이전트 시작
        start_response=$(curl -s -X POST "$API_BASE/agents/$agent_id/start")
        echo "🚀 에이전트 시작: $name"
    else
        echo "❌ 에이전트 생성 실패: $name"
        echo "응답: $response"
    fi
    
    sleep 2  # API 부하 방지를 위한 지연
done

echo "모든 에이전트 생성 완료!"

# 생성된 에이전트 목록 확인
echo "생성된 에이전트 목록:"
curl -s "$API_BASE/agents?project_id=$PROJECT_ID" | jq -r '.agents[] | "\(.name) (\(.id)) - \(.status)"'
```

### Python 스크립트 예제

Python을 사용한 에이전트 생성 스크립트입니다.

```python
#!/usr/bin/env python3
# create_agents.py

import requests
import json
import time
from typing import List, Dict

API_BASE = "http://localhost:8080/api/v1"
PROJECT_ID = "python-batch-demo"

class AgentCreator:
    def __init__(self, api_base: str):
        self.api_base = api_base
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})
    
    def create_agent(self, config: Dict) -> Dict:
        """에이전트 생성"""
        try:
            response = self.session.post(f"{self.api_base}/agents", json=config)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"❌ 에이전트 생성 실패: {e}")
            return {}
    
    def start_agent(self, agent_id: str) -> bool:
        """에이전트 시작"""
        try:
            response = self.session.post(f"{self.api_base}/agents/{agent_id}/start")
            response.raise_for_status()
            return True
        except requests.exceptions.RequestException as e:
            print(f"❌ 에이전트 시작 실패: {e}")
            return False
    
    def get_agent_status(self, agent_id: str) -> Dict:
        """에이전트 상태 확인"""
        try:
            response = self.session.get(f"{self.api_base}/agents/{agent_id}/status")
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"❌ 상태 확인 실패: {e}")
            return {}

def main():
    creator = AgentCreator(API_BASE)
    
    # 생성할 에이전트 설정
    agent_configs = [
        {
            "name": "web-server-agent",
            "project_id": PROJECT_ID,
            "agent_type": "standard",
            "description": "웹 서버 실행용 에이전트",
            "config": {
                "environment": {
                    "PORT": "8080",
                    "NODE_ENV": "production"
                },
                "resources": {
                    "cpu": "1",
                    "memory": "1Gi"
                }
            }
        },
        {
            "name": "database-agent",
            "project_id": PROJECT_ID,
            "agent_type": "memory_optimized",
            "description": "데이터베이스 실행용 에이전트",
            "config": {
                "environment": {
                    "POSTGRES_DB": "appdb",
                    "POSTGRES_USER": "dbuser"
                },
                "resources": {
                    "cpu": "2",
                    "memory": "4Gi"
                }
            }
        },
        {
            "name": "worker-agent",
            "project_id": PROJECT_ID,
            "agent_type": "standard",
            "description": "백그라운드 작업용 에이전트",
            "config": {
                "environment": {
                    "WORKER_TYPE": "background",
                    "QUEUE_NAME": "default"
                },
                "resources": {
                    "cpu": "0.5",
                    "memory": "512Mi"
                }
            }
        }
    ]
    
    created_agents = []
    
    print(f"프로젝트 '{PROJECT_ID}'에 {len(agent_configs)}개 에이전트를 생성합니다...")
    
    for config in agent_configs:
        print(f"\n에이전트 생성 중: {config['name']}")
        
        # 에이전트 생성
        agent = creator.create_agent(config)
        if not agent:
            continue
            
        agent_id = agent.get('id')
        print(f"✅ 에이전트 생성 성공: {config['name']} (ID: {agent_id})")
        
        # 에이전트 시작
        if creator.start_agent(agent_id):
            print(f"🚀 에이전트 시작: {config['name']}")
            created_agents.append(agent)
        
        time.sleep(2)  # API 부하 방지
    
    # 모든 에이전트 상태 확인
    print(f"\n생성된 에이전트 상태 확인:")
    for agent in created_agents:
        status = creator.get_agent_status(agent['id'])
        if status:
            print(f"📊 {agent['name']}: {status.get('status', 'unknown')}")
    
    print(f"\n총 {len(created_agents)}개 에이전트가 생성되었습니다!")

if __name__ == "__main__":
    main()
```

## 🧪 에이전트 생성 테스트

### 생성 성공 확인

```bash
# 에이전트 생성 후 ID 저장
AGENT_ID=$(curl -s -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-agent",
    "project_id": "test",
    "agent_type": "standard"
  }' | jq -r '.id')

echo "생성된 에이전트 ID: $AGENT_ID"

# 에이전트 존재 확인
curl -s http://localhost:8080/api/v1/agents/$AGENT_ID | jq '.name'
```

### 에이전트 상세 정보 확인

```bash
# 전체 에이전트 정보 조회
curl -s http://localhost:8080/api/v1/agents/$AGENT_ID | jq '.'

# 특정 필드만 확인
curl -s http://localhost:8080/api/v1/agents/$AGENT_ID | jq '{
  name: .name,
  status: .status,
  type: .agent_type,
  created: .created_at
}'
```

## 🚨 에러 처리

### 일반적인 생성 에러

#### 1. 중복 이름 에러
```bash
# 에러 응답 예시
{
  "error": "invalid_request",
  "message": "Agent name must be unique within project",
  "code": "DUPLICATE_NAME"
}

# 해결 방법: 고유한 이름 사용
TIMESTAMP=$(date +%s)
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"agent-${TIMESTAMP}\",
    \"project_id\": \"test\",
    \"agent_type\": \"standard\"
  }"
```

#### 2. 리소스 부족 에러
```bash
# 에러 응답 예시
{
  "error": "resource_unavailable",
  "message": "Insufficient resources to create agent",
  "code": "RESOURCE_LIMIT_EXCEEDED"
}

# 해결 방법: 리소스 요구사항 줄이기
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "lightweight-agent",
    "project_id": "test",
    "agent_type": "standard",
    "config": {
      "resources": {
        "cpu": "0.1",
        "memory": "128Mi"
      }
    }
  }'
```

## 📝 베스트 프랙티스

### 1. 명명 규칙

```bash
# 좋은 에이전트 이름 예시
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "build-webapp-main-20250730",
    "project_id": "webapp",
    "agent_type": "standard",
    "description": "메인 브랜치 웹앱 빌드용 에이전트"
  }'

# 패턴: {목적}-{프로젝트}-{브랜치}-{날짜}
```

### 2. 리소스 계획

```bash
# 작업 유형별 권장 리소스 설정

# 경량 작업 (린팅, 간단한 테스트)
{
  "resources": {
    "cpu": "0.5",
    "memory": "512Mi"
  }
}

# 일반 빌드 작업
{
  "resources": {
    "cpu": "2",
    "memory": "2Gi"
  }
}

# 대용량 빌드 (Docker, 큰 프로젝트)
{
  "resources": {
    "cpu": "4",
    "memory": "8Gi"
  }
}
```

### 3. 환경 변수 관리

```bash
# 보안 정보는 환경 변수로 전달하지 않기
# 대신 별도의 시크릿 관리 시스템 사용

# ❌ 좋지 않은 예시
{
  "environment": {
    "DATABASE_PASSWORD": "supersecret123",
    "API_KEY": "sk-1234567890abcdef"
  }
}

# ✅ 좋은 예시
{
  "environment": {
    "DATABASE_HOST": "db.internal.com",
    "API_ENDPOINT": "https://api.service.com",
    "LOG_LEVEL": "info"
  }
}
```

---

이제 에이전트 생성의 모든 측면을 이해했습니다. 다음 단계로 [생명주기 관리](manage-lifecycle.md)를 확인해보세요!