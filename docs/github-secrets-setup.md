# GitHub Secrets 설정 가이드

## 개요

릴리스 자동화 파이프라인이 제대로 작동하려면 GitHub 저장소에 필요한 시크릿을 설정해야 합니다.

## 필요한 시크릿

### 1. Docker Hub 인증 (선택사항)

Docker 이미지를 자동으로 푸시하려면 다음 시크릿이 필요합니다:

- `DOCKERHUB_USERNAME`: Docker Hub 사용자명
- `DOCKERHUB_TOKEN`: Docker Hub 액세스 토큰

## 시크릿 설정 방법

### GitHub UI를 통한 설정

1. GitHub 저장소 페이지로 이동
2. **Settings** 탭 클릭
3. 왼쪽 사이드바에서 **Secrets and variables** → **Actions** 클릭
4. **New repository secret** 버튼 클릭
5. 시크릿 이름과 값 입력
6. **Add secret** 클릭

### Docker Hub 액세스 토큰 생성

1. [Docker Hub](https://hub.docker.com) 로그인
2. 우측 상단 프로필 → **Account Settings**
3. **Security** 탭 선택
4. **New Access Token** 클릭
5. 토큰 설명 입력 (예: "aicli-web GitHub Actions")
6. 권한 선택:
   - **Read** (public repos)
   - **Write** (push images)
   - **Delete** (선택사항)
7. **Generate** 클릭
8. 생성된 토큰을 복사하여 GitHub 시크릿에 저장

## 시크릿 검증

설정이 완료되면 다음 명령으로 테스트할 수 있습니다:

```yaml
# .github/workflows/test-secrets.yml
name: Test Secrets

on:
  workflow_dispatch:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Check Docker Hub credentials
        run: |
          if [ -n "${{ secrets.DOCKERHUB_USERNAME }}" ]; then
            echo "✅ DOCKERHUB_USERNAME is set"
          else
            echo "❌ DOCKERHUB_USERNAME is not set"
          fi
          
          if [ -n "${{ secrets.DOCKERHUB_TOKEN }}" ]; then
            echo "✅ DOCKERHUB_TOKEN is set"
          else
            echo "❌ DOCKERHUB_TOKEN is not set"
          fi
```

## 보안 모범 사례

1. **최소 권한 원칙**
   - 필요한 권한만 부여
   - 읽기 전용 토큰과 쓰기 토큰 분리

2. **정기적인 토큰 교체**
   - 3-6개월마다 토큰 재생성
   - 이전 토큰은 즉시 폐기

3. **액세스 로그 모니터링**
   - Docker Hub 보안 페이지에서 토큰 사용 내역 확인
   - 비정상적인 활동 감지 시 즉시 토큰 폐기

4. **시크릿 노출 방지**
   - 로그에 시크릿이 출력되지 않도록 주의
   - PR에서 시크릿 사용 제한

## 문제 해결

### "Error: Username and password required"

Docker Hub 시크릿이 제대로 설정되지 않았습니다:
- 시크릿 이름 확인 (대소문자 구분)
- 토큰이 만료되지 않았는지 확인

### "denied: requested access to the resource is denied"

권한 문제입니다:
- Docker Hub 토큰의 Write 권한 확인
- 저장소 이름이 올바른지 확인

## 시크릿 없이 릴리스하기

Docker 이미지가 필요하지 않은 경우:

1. `.github/workflows/release.yml`에서 `docker-release` job 제거
2. 또는 조건부 실행:
   ```yaml
   docker-release:
     if: ${{ secrets.DOCKERHUB_USERNAME != '' }}
   ```

## 추가 리소스

- [GitHub Actions 시크릿 문서](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [Docker Hub 액세스 토큰 문서](https://docs.docker.com/docker-hub/access-tokens/)