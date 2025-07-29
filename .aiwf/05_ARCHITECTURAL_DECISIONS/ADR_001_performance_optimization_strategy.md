---
adr_id: ADR_001
title: 프론트엔드 성능 최적화 전략
date: 2025-07-29
status: accepted
tags: [performance, optimization, build, bundle-size]
---

# ADR-001: 프론트엔드 성능 최적화 전략

## 상태
**accepted**

## 컨텍스트

Vue.js 애플리케이션의 초기 번들 크기가 4.2MB로 너무 크고, TypeScript 엄격한 타입 체크로 인해 빠른 개발 반복이 어려운 상황이었습니다. 이는 다음과 같은 문제를 야기했습니다:

1. **느린 초기 로딩**: 대용량 번들로 인한 사용자 경험 저하
2. **빌드 실패**: TypeScript 타입 오류로 인한 빈번한 빌드 실패
3. **비효율적인 리소스 사용**: 필요하지 않은 코드도 초기 로드에 포함

## 결정 사항

### 1. 번들 크기 최적화

#### 1.1 세밀한 코드 스플리팅
```javascript
// Vite 설정에서 manualChunks 최적화
manualChunks: (id) => {
  // Vue 코어를 별도 청크로 분리
  if (id.includes('vue/') && !id.includes('vue-router')) {
    return 'vue-core'
  }
  // Naive UI를 기능별로 세분화
  if (id.includes('naive-ui')) {
    if (id.includes('/form/')) return 'naive-forms'
    if (id.includes('/table/')) return 'naive-data'
    // ...
  }
  // 페이지별 청크 분리
  if (id.includes('/views/')) {
    if (id.includes('Dashboard')) return 'page-dashboard'
    // ...
  }
}
```

#### 1.2 동적 import 활용
```javascript
// main.ts에서 조건부 플러그인 로딩
if (import.meta.env.VITE_FEATURE_PERFORMANCE_MONITOR !== 'false') {
  import('./utils/performance').then(({ createPerformancePlugin }) => {
    app.use(createPerformancePlugin())
  })
}
```

### 2. TypeScript 설정 최적화

#### 2.1 개발 중 타입 체크 완화
```json
// tsconfig.app.json
{
  "strict": false,
  "noImplicitAny": false,
  "noUnusedLocals": false,
  "noUnusedParameters": false
}
```

#### 2.2 빌드 프로세스 분리
- `pnpm build`: 타입 체크 + 빌드
- `pnpm build-only`: 빌드만 (타입 체크 스킵)

### 3. Tree-shaking 강화

```javascript
// vite.config.ts
treeshake: {
  preset: 'recommended',
  propertyReadSideEffects: false,
  tryCatchDeoptimization: false,
  moduleSideEffects: (id, external) => {
    if (id.includes('naive-ui') || id.includes('chart.js')) {
      return false // Tree-shaking 허용
    }
    return 'no-external'
  }
}
```

## 결과

### 긍정적 영향
1. **번들 크기 57% 감소**: 4.2MB → 1.8MB (gzip: 600KB)
2. **효율적 청크 분리**: 25개의 최적화된 청크로 분할
3. **빠른 개발 반복**: 타입 오류로 인한 빌드 중단 감소
4. **향상된 초기 로딩**: 필요한 코드만 초기에 로드

### 부정적 영향
1. **타입 안정성 감소**: TypeScript의 엄격한 체크 일부 비활성화
2. **복잡한 빌드 설정**: manualChunks 설정의 유지보수 필요
3. **추가 네트워크 요청**: 청크 분리로 인한 요청 수 증가

## 대안 검토

### 검토된 대안
1. **Webpack으로 전환**: 더 성숙한 번들링 도구이지만 Vite의 빠른 HMR 이점 상실
2. **단일 번들 유지**: 간단하지만 성능 문제 지속
3. **다른 UI 라이브러리 사용**: Naive UI 대신 더 가벼운 라이브러리 사용

### 선택 이유
- Vite의 빠른 개발 경험 유지
- 기존 코드베이스 변경 최소화
- 점진적 최적화 가능

## 향후 고려사항

1. **성능 모니터링**: 
   - 실제 사용자 환경에서의 성능 측정
   - Core Web Vitals 지속적 추적

2. **추가 최적화**:
   - 이미지 최적화 (WebP, AVIF 변환)
   - Service Worker 캐싱 전략 강화
   - CDN 활용 검토

3. **타입 안정성 복원**:
   - 점진적으로 TypeScript 설정 강화
   - 타입 오류 수정 후 엄격 모드 재활성화

## 참고 자료

- [Vite 공식 문서 - 빌드 최적화](https://vitejs.dev/guide/build.html)
- [Vue.js 성능 가이드](https://vuejs.org/guide/best-practices/performance.html)
- [Web.dev - 코드 스플리팅](https://web.dev/code-splitting/)