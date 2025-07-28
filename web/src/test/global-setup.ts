import type { FullConfig } from '@playwright/test'

/**
 * Playwright 글로벌 설정
 * 모든 테스트 실행 전에 한 번 실행됩니다
 */
async function globalSetup(config: FullConfig): Promise<void> {
  console.log('🚀 글로벌 테스트 설정 시작')

  // 환경 변수 검증
  const requiredEnvVars = [
    'VITE_API_BASE_URL',
    'VITE_WS_BASE_URL',
  ]

  for (const envVar of requiredEnvVars) {
    if (!process.env[envVar]) {
      console.warn(`⚠️  환경 변수 ${envVar}가 설정되지 않았습니다`)
    }
  }

  // 테스트 데이터베이스 초기화 (필요시)
  console.log('📊 테스트 환경 준비 중...')

  // 성능 측정 기준점 설정
  console.log('⚡ 성능 측정 기준점 설정')

  console.log('✅ 글로벌 테스트 설정 완료')
}

export default globalSetup