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
      settings: {
        preset: 'desktop',
        throttling: {
          cpuSlowdownMultiplier: 1,
        },
        // 추가 설정
        onlyCategories: ['performance', 'accessibility', 'best-practices', 'seo', 'pwa'],
      },
    },
    assert: {
      preset: 'lighthouse:recommended',
      assertions: {
        // Core Web Vitals 임계값 (Google 권장사항)
        'first-contentful-paint': ['warn', { maxNumericValue: 1800 }],
        'largest-contentful-paint': ['error', { maxNumericValue: 2500 }],
        'cumulative-layout-shift': ['error', { maxNumericValue: 0.1 }],
        'total-blocking-time': ['warn', { maxNumericValue: 300 }],
        'max-potential-fid': ['warn', { maxNumericValue: 100 }],
        
        // 카테고리별 최소 점수
        'categories:performance': ['error', { minScore: 0.85 }],
        'categories:accessibility': ['error', { minScore: 0.9 }],
        'categories:best-practices': ['warn', { minScore: 0.85 }],
        'categories:seo': ['warn', { minScore: 0.85 }],
        'categories:pwa': ['warn', { minScore: 0.8 }],
        
        // 번들 크기 제한
        'resource-summary:script:size': ['warn', { maxNumericValue: 250000 }], // 250KB
        'resource-summary:stylesheet:size': ['warn', { maxNumericValue: 150000 }], // 150KB
        'resource-summary:image:size': ['warn', { maxNumericValue: 500000 }], // 500KB
        'resource-summary:third-party:size': ['warn', { maxNumericValue: 200000 }], // 200KB
        'resource-summary:total:size': ['error', { maxNumericValue: 1000000 }], // 1MB
        
        // 추가 성능 메트릭
        'interactive': ['warn', { maxNumericValue: 3800 }],
        'speed-index': ['warn', { maxNumericValue: 3400 }],
        'bootup-time': ['warn', { maxNumericValue: 3500 }],
        'mainthread-work-breakdown': ['warn', { maxNumericValue: 4000 }],
        
        // 접근성 세부 사항
        'color-contrast': 'error',
        'heading-order': 'error',
        'aria-allowed-attr': 'error',
        'aria-required-attr': 'error',
        'aria-valid-attr': 'error',
        'aria-valid-attr-value': 'error',
        'button-name': 'error',
        'image-alt': 'error',
        'label': 'error',
        'link-name': 'error',
        'list': 'error',
        'listitem': 'error',
        
        // 보안 관련
        'csp-xss': 'warn',
        'is-on-https': 'error',
        'no-vulnerable-libraries': 'error',
        
        // PWA 관련
        'installable-manifest': 'warn',
        'service-worker': 'warn',
        'offline-start-url': 'warn',
        'themed-omnibox': 'warn',
        'maskable-icon': 'warn',
        
        // 모바일 친화성
        'viewport': 'error',
        'tap-targets': 'warn',
        'font-size': 'warn',
        
        // SEO 최적화
        'document-title': 'error',
        'meta-description': 'warn',
        'link-text': 'warn',
        'crawlable-anchors': 'error',
        'robots-txt': 'warn',
        'hreflang': 'warn',
        'canonical': 'warn',
      },
    },
    upload: {
      target: 'temporary-public-storage',
      reportFilenamePattern: '.lighthouseci/reports/%%PATHNAME%%-%%DATETIME%%-report.%%EXTENSION%%',
    },
    server: {},
    wizard: {},
  },
}