module.exports = {
  ci: {
    collect: {
      // 정적 사이트 모드로 설정
      staticDistDir: './dist',
      // 테스트할 URL들
      url: [
        'http://localhost:4173/',
        'http://localhost:4173/workspace',
        'http://localhost:4173/dashboard',
        'http://localhost:4173/terminal',
      ],
      // 빌드 후 서버 시작 명령어
      startServerCommand: 'npm run preview',
      startServerReadyPattern: 'Local:',
      startServerReadyTimeout: 30000,
      numberOfRuns: 3, // 안정적인 결과를 위해 3회 실행
    },
    assert: {
      // 성능 기준 설정 (T09_S01 요구사항: 95점 이상)
      assertions: {
        'categories:performance': ['error', { minScore: 0.95 }],
        'categories:accessibility': ['error', { minScore: 0.95 }],
        'categories:best-practices': ['error', { minScore: 0.95 }],
        'categories:seo': ['error', { minScore: 0.95 }],
        'categories:pwa': ['warn', { minScore: 0.8 }],

        // Core Web Vitals (더 엄격한 기준)
        'largest-contentful-paint': ['error', { maxNumericValue: 2500 }],
        'first-contentful-paint': ['error', { maxNumericValue: 1500 }],
        'cumulative-layout-shift': ['error', { maxNumericValue: 0.1 }],
        'total-blocking-time': ['error', { maxNumericValue: 300 }],
        'speed-index': ['warn', { maxNumericValue: 3000 }],
        'interactive': ['error', { maxNumericValue: 3000 }],

        // 접근성 관련
        'color-contrast': 'error',
        'heading-order': 'warn',
        'aria-allowed-attr': 'error',

        // 보안 관련
        'csp-xss': 'warn',
        'is-on-https': 'warn',

        // PWA 관련
        'installable-manifest': 'warn',
        'service-worker': 'warn',
        'offline-start-url': 'warn',
      },
    },
    upload: {
      target: 'temporary-public-storage',
    },
    server: {},
    wizard: {},
  },
}