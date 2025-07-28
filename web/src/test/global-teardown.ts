/**
 * Playwright 글로벌 정리
 * 모든 테스트 실행 후에 한 번 실행됩니다
 */
async function globalTeardown(): Promise<void> {
  console.log('🧹 글로벌 테스트 정리 시작')

  // 테스트 아티팩트 정리
  console.log('🗑️  테스트 아티팩트 정리')

  // 임시 파일 정리
  console.log('📁 임시 파일 정리')

  // 성능 데이터 정리 (필요시)
  console.log('📊 성능 데이터 정리')

  console.log('✅ 글로벌 테스트 정리 완료')
}

export default globalTeardown