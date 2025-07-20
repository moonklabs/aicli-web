# 릴리스 프로세스 가이드

## 개요

AICode Manager는 Git 태그 기반의 자동화된 릴리스 시스템을 사용합니다. 시맨틱 버저닝을 따르며, 태그가 푸시되면 자동으로 릴리스가 생성됩니다.

## 릴리스 워크플로우

### 1. 버전 태그 생성

```bash
# 새로운 버전 태그 생성
git tag -a v1.0.0 -m "Release version 1.0.0"

# 프리릴리스 태그 (베타, RC 등)
git tag -a v1.0.0-beta.1 -m "Beta release 1.0.0-beta.1"
git tag -a v1.0.0-rc.1 -m "Release candidate 1.0.0-rc.1"

# 태그 푸시
git push origin v1.0.0
```

### 2. 자동화 프로세스

태그가 푸시되면 다음 단계가 자동으로 실행됩니다:

1. **사전 검증**
   - 전체 테스트 스위트 실행
   - 코드 린팅
   - 보안 스캔
   - 버전 태그 형식 검증

2. **멀티 플랫폼 빌드**
   - Linux (amd64, arm64)
   - macOS (amd64, arm64)
   - Windows (amd64, arm64)

3. **릴리스 생성**
   - 바이너리 수집
   - SHA256 체크섬 생성
   - 릴리스 노트 자동 생성
   - GitHub Release 생성

4. **Docker 이미지** (선택사항)
   - 멀티 아키텍처 이미지 빌드
   - Docker Hub 푸시

## 버전 관리 전략

### 시맨틱 버저닝

```
v{MAJOR}.{MINOR}.{PATCH}[-{PRERELEASE}][+{BUILD}]
```

- **MAJOR**: 호환되지 않는 API 변경
- **MINOR**: 하위 호환 기능 추가
- **PATCH**: 하위 호환 버그 수정
- **PRERELEASE**: alpha, beta, rc 등
- **BUILD**: 빌드 메타데이터

### 버전 예시

```
v1.0.0          # 정식 릴리스
v1.0.0-alpha.1  # 알파 릴리스
v1.0.0-beta.2   # 베타 릴리스
v1.0.0-rc.1     # 릴리스 후보
v1.0.0+20130313144700  # 빌드 메타데이터 포함
```

## 릴리스 노트 작성

### 자동 생성

릴리스 노트는 이전 태그 이후의 커밋을 기반으로 자동 생성됩니다:

- `feat:` → ✨ 새로운 기능
- `fix:` → 🐛 버그 수정
- `docs:` → 📚 문서 개선
- `chore:` → 🔧 기타 변경사항
- `refactor:` → ♻️ 코드 개선

### 수동 편집

필요한 경우 GitHub에서 릴리스를 직접 편집할 수 있습니다:

1. [Releases 페이지](https://github.com/drumcap/aicli-web/releases) 접속
2. 해당 릴리스의 "Edit" 클릭
3. 릴리스 노트 수정
4. "Update release" 클릭

## 체크섬 검증

### 다운로드 후 검증

```bash
# 전체 체크섬 파일 다운로드
curl -L https://github.com/drumcap/aicli-web/releases/latest/download/checksums.txt -o checksums.txt

# 바이너리 다운로드
curl -L https://github.com/drumcap/aicli-web/releases/latest/download/aicli-v1.0.0-linux-amd64 -o aicli

# 체크섬 검증
sha256sum -c checksums.txt

# 또는 개별 검증
sha256sum aicli
```

### 검증 스크립트 사용

```bash
# 검증 스크립트 다운로드
curl -L https://github.com/drumcap/aicli-web/releases/latest/download/verify-checksums.sh -o verify-checksums.sh
chmod +x verify-checksums.sh

# 검증 실행
./verify-checksums.sh
```

## 문제 해결

### 릴리스 실패 시

1. **Actions 탭에서 로그 확인**
   ```
   https://github.com/drumcap/aicli-web/actions
   ```

2. **일반적인 문제**
   - 태그 형식 오류: 시맨틱 버저닝 형식 확인
   - 테스트 실패: 모든 테스트가 통과하는지 확인
   - 권한 오류: GitHub 토큰 권한 확인

3. **재시도**
   ```bash
   # 태그 삭제 (로컬)
   git tag -d v1.0.0
   
   # 태그 삭제 (원격)
   git push origin :refs/tags/v1.0.0
   
   # 문제 해결 후 다시 태그 생성
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin v1.0.0
   ```

### 수동 릴리스

자동화가 실패한 경우 수동으로 릴리스할 수 있습니다:

```bash
# 로컬에서 빌드
make release

# 체크섬 생성
./scripts/generate-checksums.sh ./dist

# GitHub에서 수동으로 릴리스 생성
# 1. https://github.com/drumcap/aicli-web/releases/new
# 2. 태그 선택
# 3. 바이너리 및 체크섬 파일 업로드
# 4. 릴리스 노트 작성
# 5. "Publish release" 클릭
```

## 릴리스 체크리스트

릴리스 전 확인사항:

- [ ] 모든 테스트가 통과하는가?
- [ ] 코드 리뷰가 완료되었는가?
- [ ] CHANGELOG.md가 업데이트되었는가?
- [ ] 버전 번호가 적절한가?
- [ ] 문서가 최신 상태인가?
- [ ] 이전 버전과의 호환성을 확인했는가?

## 릴리스 후 작업

1. **릴리스 확인**
   - GitHub Releases 페이지 확인
   - 바이너리 다운로드 테스트
   - 체크섬 검증

2. **공지사항**
   - 프로젝트 README 업데이트
   - 사용자에게 알림 (필요한 경우)

3. **다음 버전 준비**
   ```bash
   # 개발 브랜치로 전환
   git checkout develop
   
   # 버전 업데이트 (예: 1.0.1-dev)
   # version.go 파일 수정
   ```

## 보안 고려사항

- 릴리스 바이너리는 항상 체크섬과 함께 제공
- HTTPS를 통해서만 다운로드
- 서명된 커밋 사용 권장
- 민감한 정보가 바이너리에 포함되지 않도록 주의

## 참고 링크

- [시맨틱 버저닝](https://semver.org/lang/ko/)
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [GitHub Actions](https://docs.github.com/en/actions)