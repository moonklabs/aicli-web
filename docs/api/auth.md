# 인증 API 문서

## 개요

AICode Manager API는 JWT(JSON Web Token) 기반 인증을 사용합니다. 모든 보호된 엔드포인트에 접근하려면 유효한 액세스 토큰이 필요합니다.

## 인증 플로우

1. `/api/v1/auth/login` 엔드포인트로 로그인하여 액세스 토큰과 리프레시 토큰을 받습니다.
2. 보호된 엔드포인트에 요청할 때 `Authorization: Bearer {access_token}` 헤더를 포함합니다.
3. 액세스 토큰이 만료되면 `/api/v1/auth/refresh` 엔드포인트로 새 액세스 토큰을 받습니다.
4. 로그아웃 시 `/api/v1/auth/logout` 엔드포인트를 호출하여 토큰을 무효화합니다.

## 엔드포인트

### 1. 로그인

사용자 자격증명으로 로그인하여 JWT 토큰을 받습니다.

**엔드포인트:** `POST /api/v1/auth/login`

**요청 본문:**
```json
{
  "username": "string",
  "password": "string"
}
```

**성공 응답 (200 OK):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

**오류 응답 (401 Unauthorized):**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid username or password"
  }
}
```

**오류 응답 (400 Bad Request):**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid request body",
    "details": "Error details..."
  }
}
```

### 2. 토큰 갱신

리프레시 토큰을 사용하여 새 액세스 토큰을 받습니다.

**엔드포인트:** `POST /api/v1/auth/refresh`

**요청 본문:**
```json
{
  "refresh_token": "string"
}
```

**성공 응답 (200 OK):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

**오류 응답 (401 Unauthorized):**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_REFRESH_TOKEN",
    "message": "Invalid or expired refresh token",
    "details": "Error details..."
  }
}
```

**오류 응답 (401 Unauthorized - 블랙리스트):**
```json
{
  "success": false,
  "error": {
    "code": "TOKEN_BLACKLISTED",
    "message": "Refresh token has been revoked"
  }
}
```

### 3. 로그아웃

현재 액세스 토큰을 무효화합니다.

**엔드포인트:** `POST /api/v1/auth/logout`

**헤더:**
```
Authorization: Bearer {access_token}
```

**성공 응답 (200 OK):**
```json
{
  "success": true,
  "message": "Logged out successfully"
}
```

**오류 응답 (400 Bad Request):**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_TOKEN",
    "message": "Invalid authorization header",
    "details": "Error details..."
  }
}
```

**오류 응답 (401 Unauthorized):**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_TOKEN",
    "message": "Invalid or expired token"
  }
}
```

## 보호된 엔드포인트 사용

모든 보호된 엔드포인트는 Authorization 헤더가 필요합니다:

```
Authorization: Bearer {access_token}
```

**예시 요청:**
```bash
curl -X GET https://api.aicli.dev/api/v1/workspaces \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

**인증 실패 응답 (401 Unauthorized):**
```json
{
  "success": false,
  "error": {
    "code": "MISSING_AUTH_HEADER",
    "message": "Authorization header is required"
  }
}
```

## 토큰 정보

### 액세스 토큰
- **유효 기간**: 15분
- **용도**: API 요청 인증
- **갱신**: 리프레시 토큰으로 갱신 가능

### 리프레시 토큰
- **유효 기간**: 7일
- **용도**: 새 액세스 토큰 발급
- **보관**: 안전하게 보관 필요

## 역할 기반 접근 제어

일부 엔드포인트는 특정 역할이 필요합니다:

- `admin`: 모든 리소스에 대한 전체 접근 권한
- `user`: 기본 사용자 권한

**권한 부족 응답 (403 Forbidden):**
```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_PERMISSIONS",
    "message": "You don't have permission to access this resource",
    "details": {
      "required_roles": ["admin"],
      "user_role": "user"
    }
  }
}
```

## 보안 권장사항

1. **HTTPS 사용**: 모든 API 요청은 HTTPS를 통해 전송되어야 합니다.
2. **토큰 저장**: 액세스 토큰은 메모리에, 리프레시 토큰은 안전한 저장소에 보관하세요.
3. **토큰 노출 방지**: 토큰을 URL 파라미터나 로그에 포함하지 마세요.
4. **정기적인 갱신**: 액세스 토큰은 짧은 수명을 가지므로 정기적으로 갱신하세요.
5. **로그아웃**: 사용 후에는 반드시 로그아웃하여 토큰을 무효화하세요.

## 임시 사용자 계정

개발 및 테스트 목적으로 다음 계정을 사용할 수 있습니다:

- **관리자**: username: `admin`, password: `admin123`
- **일반 사용자**: username: `user`, password: `user123`
- **테스트 사용자**: username: `test`, password: `test123`

**주의**: 프로덕션 환경에서는 반드시 실제 사용자 인증 시스템을 구현해야 합니다.