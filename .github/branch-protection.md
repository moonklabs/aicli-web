# GitHub Branch Protection 설정 가이드

이 문서는 GitHub 저장소의 브랜치 보호 규칙 설정 방법을 안내합니다.

## main 브랜치 보호 규칙

### 1. Settings > Branches 접속

### 2. Add rule 클릭 후 다음 설정 적용:

#### Branch name pattern
- `main`

#### Protect matching branches 설정:

- ✅ **Require a pull request before merging**
  - ✅ Require approvals (최소 1명)
  - ✅ Dismiss stale pull request approvals when new commits are pushed
  - ✅ Require review from CODEOWNERS

- ✅ **Require status checks to pass before merging**
  - ✅ Require branches to be up to date before merging
  - 필수 상태 체크 추가:
    - `Lint Code`
    - `Run Tests`
    - `Build Binary`
    - `Security Scan`
    - `PR Status Check`

- ✅ **Require conversation resolution before merging**

- ✅ **Include administrators**

- ✅ **Restrict who can push to matching branches** (선택사항)
  - 관리자 또는 특정 팀만 직접 푸시 가능하도록 설정

### 3. Create 클릭하여 규칙 생성

## develop 브랜치 보호 규칙

develop 브랜치에도 유사한 규칙을 적용하되, 다음 사항을 조정할 수 있습니다:

- Require approvals: 0명 (빠른 개발을 위해)
- Include administrators: 해제 (관리자는 직접 푸시 가능)

## CI 워크플로우 최적화

현재 CI 파이프라인은 다음과 같이 최적화되어 있습니다:

1. **병렬 실행**: lint와 security-scan이 동시에 실행됨
2. **의존성 관리**: test는 lint 성공 후, build는 test 성공 후 실행
3. **캐싱**: Go 모듈 캐시를 활용하여 빌드 시간 단축
4. **멀티플랫폼 빌드**: matrix 전략으로 여러 OS/아키텍처 동시 빌드

## 실행 시간 목표

- Lint: ~1분
- Test: ~2분  
- Build: ~1분 (플랫폼당)
- Security Scan: ~1분
- **총 실행 시간: 5분 이내**

## 추가 권장사항

1. **코드 커버리지 임계값 설정**
   - Codecov 통합 후 최소 70% 커버리지 요구

2. **자동 머지 설정**
   - Mergify 또는 GitHub의 auto-merge 기능 활용

3. **알림 설정**
   - CI 실패 시 Slack/Discord 알림
   - PR 리뷰 요청 알림

4. **정기적인 의존성 업데이트**
   - Dependabot 설정으로 자동 업데이트 PR 생성