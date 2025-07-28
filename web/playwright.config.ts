import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright 설정 파일
 * E2E 테스트 및 성능 테스트를 위한 설정
 */
export default defineConfig({
  testDir: './src/test/e2e',

  /* 병렬 실행 설정 */
  fullyParallel: true,

  /* CI에서 실패한 테스트 재시도 금지 */
  forbidOnly: !!process.env.CI,

  /* CI에서 재시도 설정 */
  retries: process.env.CI ? 2 : 0,

  /* 병렬 실행 워커 수 */
  workers: process.env.CI ? 1 : undefined,

  /* 리포터 설정 */
  reporter: [
    ['html'],
    ['json', { outputFile: 'test-results/e2e-results.json' }],
    ['junit', { outputFile: 'test-results/e2e-results.xml' }],
  ],

  /* 공통 테스트 설정 */
  use: {
    /* 기본 URL */
    baseURL: 'http://localhost:4173',

    /* 실패 시 스크린샷 촬영 */
    screenshot: 'only-on-failure',

    /* 실패 시 비디오 녹화 */
    video: 'retain-on-failure',

    /* 브라우저 추적 */
    trace: 'on-first-retry',

    /* 테스트 타임아웃 */
    actionTimeout: 30000,
    navigationTimeout: 30000,
  },

  /* 테스트 프로젝트 설정 */
  projects: [
    /* 데스크톱 브라우저 */
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    /* 모바일 브라우저 */
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
    },

    /* 태블릿 테스트 */
    {
      name: 'Tablet',
      use: { ...devices['iPad Pro'] },
    },

    /* 성능 테스트 전용 프로젝트 */
    {
      name: 'performance',
      use: {
        ...devices['Desktop Chrome'],
        // 성능 측정을 위한 설정
        launchOptions: {
          args: ['--enable-precise-memory-info'],
        },
      },
      testMatch: '**/performance/*.spec.ts',
    },
  ],

  /* 로컬 개발 서버 설정 */
  webServer: {
    command: 'npm run preview',
    url: 'http://localhost:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },

  /* 출력 디렉토리 */
  outputDir: 'test-results',

  /* 테스트 타임아웃 */
  timeout: 60000,
  expect: {
    timeout: 10000,
  },

  /* 글로벌 설정 */
  globalSetup: './src/test/global-setup.ts',
  globalTeardown: './src/test/global-teardown.ts',
})