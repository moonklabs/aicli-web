# Claude CLI 통합 가이드

## 🤖 개요

이 문서는 Claude CLI를 Python subprocess로 래핑하여 웹 플랫폼에 통합하는 방법을 설명합니다. Claude Max 구독을 활용하여 API 비용 없이 Claude의 모든 기능을 사용할 수 있습니다.

## 🔑 Claude Max 인증 설정

### 1. OAuth 토큰 생성
```bash
# Claude CLI 설치
npm install -g @anthropic-ai/claude-code

# OAuth 토큰 생성 (브라우저에서 로그인 필요)
claude setup-token

# 토큰이 ~/.config/claude/config.json에 저장됨
```

### 2. 환경 변수 설정
```python
import os

# API 키 환경 변수 제거 (Max 구독 인증 활성화)
if 'ANTHROPIC_API_KEY' in os.environ:
    del os.environ['ANTHROPIC_API_KEY']

# OAuth 토큰 설정
os.environ['CLAUDE_CODE_OAUTH_TOKEN'] = 'your-oauth-token'
```

## 📦 Python Claude 래퍼 구현

### 1. 기본 래퍼 클래스
```python
import asyncio
import json
import logging
from typing import AsyncIterator, Optional, Dict, Any
from dataclasses import dataclass
import aiofiles

logger = logging.getLogger(__name__)

@dataclass
class ClaudeSession:
    """Claude 세션 정보"""
    session_id: str
    workspace_id: str
    process: Optional[asyncio.subprocess.Process] = None
    created_at: float = None
    
class ClaudeWrapper:
    """Claude CLI Python 래퍼"""
    
    def __init__(self, max_concurrent_sessions: int = 5):
        self.sessions: Dict[str, ClaudeSession] = {}
        self.max_concurrent_sessions = max_concurrent_sessions
        self._semaphore = asyncio.Semaphore(max_concurrent_sessions)
    
    async def create_session(
        self, 
        session_id: str, 
        workspace_id: str,
        working_dir: str
    ) -> ClaudeSession:
        """새로운 Claude 세션 생성"""
        async with self._semaphore:
            session = ClaudeSession(
                session_id=session_id,
                workspace_id=workspace_id,
                created_at=asyncio.get_event_loop().time()
            )
            self.sessions[session_id] = session
            return session
    
    async def execute_command(
        self,
        session_id: str,
        prompt: str,
        working_dir: str,
        system_prompt: Optional[str] = None,
        max_turns: int = 10
    ) -> AsyncIterator[Dict[str, Any]]:
        """Claude 명령 실행 및 스트림 반환"""
        
        session = self.sessions.get(session_id)
        if not session:
            raise ValueError(f"Session {session_id} not found")
        
        # Claude CLI 명령 구성
        cmd = [
            "claude",
            "chat",
            "--stream-json",
            f"--max-turns={max_turns}",
            "--permission-mode=auto",
            "--allowed-tools=Read,Write,Bash,Edit,Search"
        ]
        
        if system_prompt:
            cmd.extend(["--system-prompt", system_prompt])
        
        # 프롬프트 추가
        cmd.append(prompt)
        
        # 프로세스 생성
        process = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
            cwd=working_dir,
            env={**os.environ, "CLAUDE_CODE_OAUTH_TOKEN": self._get_oauth_token()}
        )
        
        session.process = process
        
        # 스트림 처리
        async for line in self._read_stream(process.stdout):
            try:
                data = json.loads(line)
                yield data
            except json.JSONDecodeError:
                logger.warning(f"Failed to parse JSON: {line}")
                continue
        
        # 프로세스 종료 대기
        await process.wait()
        
        # stderr 처리
        if process.returncode != 0:
            stderr = await process.stderr.read()
            logger.error(f"Claude process failed: {stderr.decode()}")
            yield {
                "type": "error",
                "content": stderr.decode()
            }
    
    async def _read_stream(self, stream) -> AsyncIterator[str]:
        """스트림에서 줄 단위로 읽기"""
        while True:
            line = await stream.readline()
            if not line:
                break
            yield line.decode().strip()
    
    def _get_oauth_token(self) -> str:
        """OAuth 토큰 가져오기"""
        # 환경 변수에서 먼저 확인
        token = os.environ.get('CLAUDE_CODE_OAUTH_TOKEN')
        if token:
            return token
        
        # 설정 파일에서 읽기
        config_path = os.path.expanduser('~/.config/claude/config.json')
        if os.path.exists(config_path):
            with open(config_path, 'r') as f:
                config = json.load(f)
                return config.get('oauth_token', '')
        
        raise ValueError("Claude OAuth token not found")
    
    async def terminate_session(self, session_id: str):
        """세션 종료"""
        session = self.sessions.get(session_id)
        if session and session.process:
            session.process.terminate()
            await session.process.wait()
        
        if session_id in self.sessions:
            del self.sessions[session_id]
```

### 2. 병렬 실행 관리자
```python
import uuid
from concurrent.futures import ThreadPoolExecutor
from typing import List, Callable, Awaitable

class ParallelClaudeManager:
    """여러 Claude 인스턴스를 병렬로 관리"""
    
    def __init__(self, max_workers: int = 5):
        self.wrapper = ClaudeWrapper(max_concurrent_sessions=max_workers)
        self.executor = ThreadPoolExecutor(max_workers=max_workers)
        self.active_tasks: Dict[str, asyncio.Task] = {}
    
    async def create_parallel_tasks(
        self,
        workspace_id: str,
        tasks: List[Dict[str, Any]]
    ) -> List[str]:
        """여러 작업을 병렬로 생성"""
        task_ids = []
        
        for task in tasks:
            task_id = str(uuid.uuid4())
            task_ids.append(task_id)
            
            # 비동기 태스크 생성
            asyncio_task = asyncio.create_task(
                self._execute_task(task_id, workspace_id, task)
            )
            self.active_tasks[task_id] = asyncio_task
        
        return task_ids
    
    async def _execute_task(
        self,
        task_id: str,
        workspace_id: str,
        task_config: Dict[str, Any]
    ):
        """개별 작업 실행"""
        session = await self.wrapper.create_session(
            session_id=task_id,
            workspace_id=workspace_id,
            working_dir=task_config['working_dir']
        )
        
        # 작업 상태 콜백
        on_update = task_config.get('on_update')
        
        try:
            async for message in self.wrapper.execute_command(
                session_id=task_id,
                prompt=task_config['prompt'],
                working_dir=task_config['working_dir'],
                system_prompt=task_config.get('system_prompt'),
                max_turns=task_config.get('max_turns', 10)
            ):
                if on_update:
                    await on_update(task_id, message)
        
        finally:
            await self.wrapper.terminate_session(task_id)
            if task_id in self.active_tasks:
                del self.active_tasks[task_id]
    
    async def get_task_status(self, task_id: str) -> Dict[str, Any]:
        """작업 상태 확인"""
        if task_id in self.active_tasks:
            task = self.active_tasks[task_id]
            return {
                'status': 'running' if not task.done() else 'completed',
                'done': task.done(),
                'cancelled': task.cancelled()
            }
        return {'status': 'not_found'}
    
    async def cancel_task(self, task_id: str) -> bool:
        """작업 취소"""
        if task_id in self.active_tasks:
            task = self.active_tasks[task_id]
            task.cancel()
            await self.wrapper.terminate_session(task_id)
            return True
        return False
```

### 3. 스트림 처리 및 메시지 파싱
```python
class ClaudeMessageParser:
    """Claude 출력 메시지 파싱"""
    
    @staticmethod
    def parse_message(data: Dict[str, Any]) -> Dict[str, Any]:
        """Claude 메시지를 표준 포맷으로 변환"""
        message_type = data.get('type', '')
        
        if message_type == 'text':
            return {
                'type': 'text',
                'content': data.get('text', ''),
                'timestamp': asyncio.get_event_loop().time()
            }
        
        elif message_type == 'tool_use':
            return {
                'type': 'tool_use',
                'tool': data.get('name', ''),
                'parameters': data.get('input', {}),
                'timestamp': asyncio.get_event_loop().time()
            }
        
        elif message_type == 'tool_result':
            return {
                'type': 'tool_result',
                'tool': data.get('tool_name', ''),
                'result': data.get('content', ''),
                'timestamp': asyncio.get_event_loop().time()
            }
        
        elif message_type == 'error':
            return {
                'type': 'error',
                'error': data.get('error', ''),
                'timestamp': asyncio.get_event_loop().time()
            }
        
        else:
            return {
                'type': 'unknown',
                'data': data,
                'timestamp': asyncio.get_event_loop().time()
            }

class StreamBuffer:
    """스트림 버퍼링 및 청킹"""
    
    def __init__(self, chunk_size: int = 1024):
        self.buffer = []
        self.chunk_size = chunk_size
    
    async def add(self, data: str):
        """버퍼에 데이터 추가"""
        self.buffer.append(data)
        
        if len(''.join(self.buffer)) >= self.chunk_size:
            return await self.flush()
        return None
    
    async def flush(self) -> Optional[str]:
        """버퍼 내용 반환"""
        if self.buffer:
            content = ''.join(self.buffer)
            self.buffer = []
            return content
        return None
```

## 🚀 고급 기능

### 1. Git 워크트리를 활용한 격리
```python
import tempfile
import shutil
from pathlib import Path

class GitWorktreeManager:
    """Git worktree를 사용한 작업 격리"""
    
    async def create_worktree(
        self,
        repo_path: str,
        branch_name: str
    ) -> str:
        """새로운 worktree 생성"""
        worktree_dir = tempfile.mkdtemp(prefix="claude_worktree_")
        
        # worktree 생성
        proc = await asyncio.create_subprocess_exec(
            "git", "worktree", "add", worktree_dir, "-b", branch_name,
            cwd=repo_path,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        
        await proc.wait()
        
        if proc.returncode != 0:
            stderr = await proc.stderr.read()
            raise RuntimeError(f"Failed to create worktree: {stderr.decode()}")
        
        return worktree_dir
    
    async def remove_worktree(self, worktree_path: str):
        """worktree 제거"""
        # Git에서 worktree 제거
        proc = await asyncio.create_subprocess_exec(
            "git", "worktree", "remove", worktree_path,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        
        await proc.wait()
        
        # 디렉토리 강제 삭제 (실패해도 무시)
        try:
            shutil.rmtree(worktree_path)
        except:
            pass
```

### 2. 리소스 모니터링
```python
import psutil
import resource

class ResourceMonitor:
    """Claude 프로세스 리소스 모니터링"""
    
    @staticmethod
    async def monitor_process(pid: int) -> Dict[str, Any]:
        """프로세스 리소스 사용량 모니터링"""
        try:
            process = psutil.Process(pid)
            
            return {
                'cpu_percent': process.cpu_percent(interval=1),
                'memory_mb': process.memory_info().rss / 1024 / 1024,
                'num_threads': process.num_threads(),
                'open_files': len(process.open_files()),
                'status': process.status()
            }
        except psutil.NoSuchProcess:
            return {'status': 'terminated'}
    
    @staticmethod
    def set_resource_limits():
        """프로세스 리소스 제한 설정"""
        # CPU 시간 제한 (초)
        resource.setrlimit(resource.RLIMIT_CPU, (300, 600))
        
        # 메모리 제한 (바이트)
        resource.setrlimit(resource.RLIMIT_AS, (2 * 1024**3, 4 * 1024**3))
        
        # 파일 크기 제한
        resource.setrlimit(resource.RLIMIT_FSIZE, (100 * 1024**2, 200 * 1024**2))
```

## 🔧 통합 예제

### FastAPI 엔드포인트
```python
from fastapi import FastAPI, WebSocket, HTTPException
from fastapi.responses import StreamingResponse

app = FastAPI()
claude_manager = ParallelClaudeManager(max_workers=5)

@app.post("/api/claude/execute")
async def execute_claude(request: ClaudeExecuteRequest):
    """Claude 명령 실행"""
    task_ids = await claude_manager.create_parallel_tasks(
        workspace_id=request.workspace_id,
        tasks=[{
            'prompt': request.prompt,
            'working_dir': request.working_dir,
            'system_prompt': request.system_prompt,
            'max_turns': request.max_turns,
            'on_update': lambda task_id, msg: None  # WebSocket으로 전달
        }]
    )
    
    return {"task_id": task_ids[0]}

@app.websocket("/ws/claude/{task_id}")
async def claude_websocket(websocket: WebSocket, task_id: str):
    """Claude 실시간 출력 스트리밍"""
    await websocket.accept()
    
    try:
        # 작업 상태 확인
        status = await claude_manager.get_task_status(task_id)
        
        if status['status'] == 'not_found':
            await websocket.send_json({"error": "Task not found"})
            return
        
        # 실시간 업데이트 전송
        # (실제 구현에서는 메시지 큐나 이벤트 버스 사용)
        
    except Exception as e:
        await websocket.send_json({"error": str(e)})
    finally:
        await websocket.close()
```

## 📊 성능 최적화 팁

1. **프로세스 풀 사용**: 자주 사용되는 워크스페이스는 프로세스를 재사용
2. **출력 버퍼링**: 대용량 출력은 청킹하여 전송
3. **비동기 I/O**: 모든 I/O 작업은 비동기로 처리
4. **리소스 제한**: 각 프로세스에 적절한 리소스 제한 설정
5. **캐싱**: 자주 사용되는 명령 결과는 캐싱

## 🐛 문제 해결

### 일반적인 문제들
1. **인증 실패**: OAuth 토큰 만료 시 재생성 필요
2. **프로세스 중단**: 타임아웃 설정 및 강제 종료 메커니즘
3. **메모리 누수**: 완료된 세션 정리 및 가비지 컬렉션
4. **동시성 문제**: 세마포어로 동시 실행 수 제한