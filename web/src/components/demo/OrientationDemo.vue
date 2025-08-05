<template>
  <div class="orientation-demo component-responsive">
    <!-- 현재 방향 정보 -->
    <div class="orientation-status">
      <div class="status-card">
        <h2>현재 방향 상태</h2>
        <div class="status-grid">
          <div class="status-item">
            <span class="status-label">방향</span>
            <span class="status-value">
              {{ orientationState.current }}
              <span class="status-icon">
                {{ orientationState.current === 'portrait' ? '📱' : '📺' }}
              </span>
            </span>
          </div>
          
          <div class="status-item">
            <span class="status-label">각도</span>
            <span class="status-value">{{ orientationState.angle }}°</span>
          </div>
          
          <div class="status-item">
            <span class="status-label">종횡비</span>
            <span class="status-value">{{ orientationState.aspectRatio.toFixed(2) }}</span>
          </div>
          
          <div class="status-item">
            <span class="status-label">변경 중</span>
            <span class="status-value">
              {{ orientationState.isChanging ? '✅' : '❌' }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 뷰포트 정보 -->
    <div class="viewport-info">
      <div class="info-card">
        <h3>뷰포트 정보</h3>
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">너비</span>
            <span class="info-value">{{ viewport.width }}px</span>
          </div>
          <div class="info-item">
            <span class="info-label">높이</span>
            <span class="info-value">{{ viewport.height }}px</span>
          </div>
          <div class="info-item">
            <span class="info-label">사용 가능 너비</span>
            <span class="info-value">{{ viewport.availableWidth }}px</span>
          </div>
          <div class="info-item">
            <span class="info-label">사용 가능 높이</span>
            <span class="info-value">{{ viewport.availableHeight }}px</span>
          </div>
          <div class="info-item">
            <span class="info-label">기기 픽셀 비율</span>
            <span class="info-value">{{ viewport.devicePixelRatio }}x</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 안전 영역 정보 -->
    <div class="safe-area-info">
      <div class="info-card">
        <h3>안전 영역</h3>
        <div class="safe-area-visual">
          <div class="safe-area-box">
            <div class="safe-area-inset safe-area-top">
              상단: {{ safeAreaInsets.top }}
            </div>
            <div class="safe-area-content">
              <div class="safe-area-inset safe-area-left">
                좌: {{ safeAreaInsets.left }}
              </div>
              <div class="safe-area-center">
                콘텐츠 영역
              </div>
              <div class="safe-area-inset safe-area-right">
                우: {{ safeAreaInsets.right }}
              </div>
            </div>
            <div class="safe-area-inset safe-area-bottom">
              하단: {{ safeAreaInsets.bottom }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 방향 분류 정보 -->
    <div class="orientation-classification">
      <div class="info-card">
        <h3>방향 분류</h3>
        <div class="classification-list">
          <div class="classification-item" :class="{ active: isPortrait }">
            <span class="classification-icon">📱</span>
            <span class="classification-text">세로 모드</span>
          </div>
          <div class="classification-item" :class="{ active: isLandscape }">
            <span class="classification-icon">📺</span>
            <span class="classification-text">가로 모드</span>
          </div>
          <div class="classification-item" :class="{ active: isNarrowPortrait }">
            <span class="classification-icon">📲</span>
            <span class="classification-text">좁은 세로</span>
          </div>
          <div class="classification-item" :class="{ active: isWidePortrait }">
            <span class="classification-icon">📱➕</span>
            <span class="classification-text">넓은 세로</span>
          </div>
          <div class="classification-item" :class="{ active: isCompactLandscape }">
            <span class="classification-icon">📱📺</span>
            <span class="classification-text">컴팩트 가로</span>
          </div>
          <div class="classification-item" :class="{ active: isWideLandscape }">
            <span class="classification-icon">🖥️</span>
            <span class="classification-text">넓은 가로</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 레이아웃 테스트 -->
    <div class="layout-test">
      <div class="test-card">
        <h3>레이아웃 테스트</h3>
        
        <!-- 카드 그리드 -->
        <div class="card-grid">
          <div
            v-for="i in 6"
            :key="i"
            class="test-card-item card"
          >
            <div class="card-content">
              <h4>카드 {{ i }}</h4>
              <p>방향에 따라 레이아웃이 조정됩니다.</p>
            </div>
          </div>
        </div>
        
        <!-- 폼 테스트 -->
        <div class="form-test">
          <h4>폼 레이아웃</h4>
          <form class="form">
            <div class="form-row">
              <div class="form-field">
                <label>이름</label>
                <input type="text" placeholder="이름을 입력하세요" />
              </div>
              <div class="form-field">
                <label>이메일</label>
                <input type="email" placeholder="이메일을 입력하세요" />
              </div>
            </div>
            <div class="form-row">
              <div class="form-field">
                <label>전화번호</label>
                <input type="tel" placeholder="전화번호를 입력하세요" />
              </div>
              <div class="form-field">
                <label>회사</label>
                <input type="text" placeholder="회사명을 입력하세요" />
              </div>
            </div>
          </form>
        </div>
        
        <!-- 테이블 테스트 -->
        <div class="table-test">
          <h4>테이블 레이아웃</h4>
          <table class="table">
            <thead>
              <tr class="table-row">
                <th class="table-cell">이름</th>
                <th class="table-cell">역할</th>
                <th class="table-cell">상태</th>
                <th class="table-cell">액션</th>
              </tr>
            </thead>
            <tbody>
              <tr class="table-row">
                <td class="table-cell" data-label="이름">홍길동</td>
                <td class="table-cell" data-label="역할">개발자</td>
                <td class="table-cell" data-label="상태">활성</td>
                <td class="table-cell" data-label="액션">
                  <TouchButton size="sm" variant="primary">편집</TouchButton>
                </td>
              </tr>
              <tr class="table-row">
                <td class="table-cell" data-label="이름">김철수</td>
                <td class="table-cell" data-label="역할">디자이너</td>
                <td class="table-cell" data-label="상태">비활성</td>
                <td class="table-cell" data-label="액션">
                  <TouchButton size="sm" variant="secondary">보기</TouchButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- 방향 제어 -->
    <div class="orientation-controls">
      <div class="controls-card">
        <h3>방향 제어</h3>
        <div class="control-buttons">
          <TouchButton
            variant="primary"
            size="md"
            icon="RotateIcon"
            @click="forcePortrait"
          >
            세로 고정
          </TouchButton>
          <TouchButton
            variant="primary"
            size="md"
            icon="RotateIcon"
            @click="forceLandscape"
          >
            가로 고정
          </TouchButton>
          <TouchButton
            variant="secondary"
            size="md"
            icon="UnlockIcon"
            @click="unlockOrientation"
          >
            잠금 해제
          </TouchButton>
        </div>
        
        <div class="control-info">
          <p>
            <strong>참고:</strong> 방향 고정은 지원되는 브라우저에서만 작동합니다.
            대부분의 데스크톱 브라우저에서는 지원되지 않을 수 있습니다.
          </p>
        </div>
      </div>
    </div>

    <!-- 이벤트 로그 -->
    <div class="event-log">
      <div class="log-card">
        <h3>방향 변경 로그</h3>
        <div class="log-controls">
          <TouchButton
            variant="tertiary"
            size="sm"
            @click="clearLog"
          >
            로그 지우기
          </TouchButton>
        </div>
        <div class="log-entries">
          <div
            v-for="(entry, index) in eventLog"
            :key="index"
            class="log-entry"
          >
            <span class="log-time">{{ entry.timestamp }}</span>
            <span class="log-event">{{ entry.event }}</span>
            <span class="log-details">{{ entry.details }}</span>
          </div>
          <div v-if="eventLog.length === 0" class="log-empty">
            방향을 변경하면 로그가 표시됩니다.
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useOrientationAdaptation } from '@/composables/useOrientationAdaptation'
import TouchButton from '../ui/form/TouchButton.vue'

interface LogEntry {
  timestamp: string
  event: string
  details: string
}

// 방향 적응 사용
const {
  orientationState,
  viewport,
  isPortrait,
  isLandscape,
  isNarrowPortrait,
  isWidePortrait,
  isCompactLandscape,
  isWideLandscape,
  safeAreaInsets,
  forceOrientation,
  unlockOrientation,
  addOrientationChangeListener,
  removeOrientationChangeListener,
} = useOrientationAdaptation({
  enableAutoRotation: true,
  enableLayoutSwitching: true,
  enableContentReflow: true,
  enableTransitions: true,
  transitionDuration: 300,
})

// 이벤트 로그
const eventLog = ref<LogEntry[]>([])

// 방향 변경 리스너
const handleOrientationChange = (newOrientation: string, oldOrientation: string) => {
  const entry: LogEntry = {
    timestamp: new Date().toLocaleTimeString(),
    event: '방향 변경',
    details: `${oldOrientation} → ${newOrientation}`,
  }
  
  eventLog.value.unshift(entry)
  
  // 로그 크기 제한
  if (eventLog.value.length > 20) {
    eventLog.value = eventLog.value.slice(0, 20)
  }
}

// 방향 제어 함수들
const forcePortrait = async () => {
  try {
    await forceOrientation('portrait')
    addLogEntry('방향 고정', '세로 모드로 고정됨')
  } catch (error) {
    addLogEntry('방향 고정 실패', '세로 모드 고정 실패: ' + (error as Error).message)
  }
}

const forceLandscape = async () => {
  try {
    await forceOrientation('landscape')
    addLogEntry('방향 고정', '가로 모드로 고정됨')
  } catch (error) {
    addLogEntry('방향 고정 실패', '가로 모드 고정 실패: ' + (error as Error).message)
  }
}

const handleUnlockOrientation = () => {
  try {
    unlockOrientation()
    addLogEntry('방향 잠금 해제', '방향 잠금이 해제됨')
  } catch (error) {
    addLogEntry('잠금 해제 실패', '방향 잠금 해제 실패: ' + (error as Error).message)
  }
}

// 로그 유틸리티
const addLogEntry = (event: string, details: string) => {
  const entry: LogEntry = {
    timestamp: new Date().toLocaleTimeString(),
    event,
    details,
  }
  
  eventLog.value.unshift(entry)
  
  if (eventLog.value.length > 20) {
    eventLog.value = eventLog.value.slice(0, 20)
  }
}

const clearLog = () => {
  eventLog.value = []
}

// 라이프사이클
onMounted(() => {
  addOrientationChangeListener(handleOrientationChange)
  addLogEntry('컴포넌트 마운트', '방향 데모 컴포넌트가 초기화됨')
})

onBeforeUnmount(() => {
  removeOrientationChangeListener(handleOrientationChange)
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.orientation-demo {
  padding: $spacing-6;
  max-width: 1200px;
  margin: 0 auto;
  
  @include mobile {
    padding: $spacing-4;
  }
}

// 카드 기본 스타일
.status-card,
.info-card,
.test-card,
.controls-card,
.log-card {
  background: $light-bg-primary;
  border: 1px solid map-get($gray-colors, 200);
  border-radius: $border-radius-lg;
  padding: $spacing-6;
  margin-bottom: $spacing-6;
  box-shadow: $shadow-sm;
  
  .dark & {
    background: $dark-bg-secondary;
    border-color: $dark-bg-tertiary;
  }
  
  h2, h3, h4 {
    margin: 0 0 $spacing-4 0;
    color: $light-text-primary;
    
    .dark & {
      color: $dark-text-primary;
    }
  }
  
  h2 {
    font-size: $font-size-xl;
    font-weight: $font-weight-semibold;
  }
  
  h3 {
    font-size: $font-size-lg;
    font-weight: $font-weight-medium;
  }
  
  h4 {
    font-size: $font-size-base;
    font-weight: $font-weight-medium;
  }
}

// 상태 그리드
.status-grid,
.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: $spacing-4;
}

.status-item,
.info-item {
  display: flex;
  flex-direction: column;
  gap: $spacing-1;
  
  .status-label,
  .info-label {
    font-size: $font-size-sm;
    color: $light-text-secondary;
    font-weight: $font-weight-medium;
    
    .dark & {
      color: $dark-text-secondary;
    }
  }
  
  .status-value,
  .info-value {
    font-size: $font-size-lg;
    color: $light-text-primary;
    font-weight: $font-weight-semibold;
    
    .dark & {
      color: $dark-text-primary;
    }
  }
  
  .status-icon {
    margin-left: $spacing-2;
  }
}

// 안전 영역 시각화
.safe-area-visual {
  .safe-area-box {
    position: relative;
    background: map-get($gray-colors, 100);
    border: 2px solid map-get($primary-colors, 300);
    border-radius: $border-radius-md;
    min-height: 200px;
    
    .dark & {
      background: $dark-bg-primary;
      border-color: map-get($primary-colors, 500);
    }
  }
  
  .safe-area-inset {
    @include flex-center;
    font-size: $font-size-xs;
    font-weight: $font-weight-medium;
    color: $light-text-secondary;
    background: rgba(map-get($primary-colors, 500), 0.1);
    
    .dark & {
      color: $dark-text-secondary;
    }
    
    &.safe-area-top,
    &.safe-area-bottom {
      height: 30px;
    }
    
    &.safe-area-left,
    &.safe-area-right {
      writing-mode: vertical-lr;
      width: 40px;
    }
  }
  
  .safe-area-content {
    display: flex;
    flex: 1;
    min-height: 140px;
  }
  
  .safe-area-center {
    @include flex-center;
    flex: 1;
    font-weight: $font-weight-medium;
    color: $light-text-primary;
    
    .dark & {
      color: $dark-text-primary;
    }
  }
}

// 방향 분류
.classification-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: $spacing-3;
}

.classification-item {
  @include flex-center;
  flex-direction: column;
  gap: $spacing-2;
  padding: $spacing-3;
  border: 2px solid map-get($gray-colors, 200);
  border-radius: $border-radius-md;
  transition: all $transition-base;
  
  .dark & {
    border-color: $dark-bg-tertiary;
  }
  
  &.active {
    border-color: map-get($primary-colors, 500);
    background: map-get($primary-colors, 50);
    
    .dark & {
      background: rgba(map-get($primary-colors, 500), 0.1);
    }
  }
  
  .classification-icon {
    font-size: $font-size-xl;
  }
  
  .classification-text {
    font-size: $font-size-sm;
    font-weight: $font-weight-medium;
    text-align: center;
    color: $light-text-primary;
    
    .dark & {
      color: $dark-text-primary;
    }
  }
}

// 제어 버튼
.control-buttons {
  display: flex;
  gap: $spacing-3;
  margin-bottom: $spacing-4;
  
  @include mobile {
    flex-direction: column;
  }
}

.control-info {
  padding: $spacing-3;
  background: map-get($warning, 0.1);
  border: 1px solid rgba($warning, 0.3);
  border-radius: $border-radius-md;
  
  p {
    margin: 0;
    font-size: $font-size-sm;
    color: $light-text-secondary;
    
    .dark & {
      color: $dark-text-secondary;
    }
  }
}

// 로그
.log-controls {
  display: flex;
  justify-content: flex-end;
  margin-bottom: $spacing-4;
}

.log-entries {
  max-height: 300px;
  overflow-y: auto;
  @include scrollbar-thin;
}

.log-entry {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: $spacing-3;
  padding: $spacing-2;
  border-bottom: 1px solid map-get($gray-colors, 100);
  font-size: $font-size-sm;
  
  .dark & {
    border-bottom-color: $dark-bg-tertiary;
  }
  
  &:last-child {
    border-bottom: none;
  }
  
  .log-time {
    color: $light-text-secondary;
    font-family: monospace;
    
    .dark & {
      color: $dark-text-secondary;
    }
  }
  
  .log-event {
    font-weight: $font-weight-medium;
    color: $light-text-primary;
    
    .dark & {
      color: $dark-text-primary;
    }
  }
  
  .log-details {
    color: $light-text-secondary;
    text-align: right;
    
    .dark & {
      color: $dark-text-secondary;
    }
  }
}

.log-empty {
  @include flex-center;
  padding: $spacing-6;
  color: $light-text-secondary;
  font-style: italic;
  
  .dark & {
    color: $dark-text-secondary;
  }
}

// 폼 스타일
.form-test {
  margin-top: $spacing-6;
  
  .form-field {
    label {
      display: block;
      margin-bottom: $spacing-2;
      font-weight: $font-weight-medium;
      color: $light-text-primary;
      
      .dark & {
        color: $dark-text-primary;
      }
    }
    
    input {
      width: 100%;
      padding: $spacing-3;
      border: 1px solid map-get($gray-colors, 300);
      border-radius: $border-radius-md;
      font-size: $font-size-base;
      
      .dark & {
        background: $dark-bg-primary;
        border-color: $dark-bg-tertiary;
        color: $dark-text-primary;
      }
      
      &:focus {
        outline: none;
        border-color: map-get($primary-colors, 500);
        box-shadow: 0 0 0 3px rgba(map-get($primary-colors, 500), 0.1);
      }
    }
  }
}

// 테이블 스타일
.table-test {
  margin-top: $spacing-6;
  
  .table {
    width: 100%;
    border-collapse: collapse;
    
    .table-cell {
      padding: $spacing-3;
      border: 1px solid map-get($gray-colors, 200);
      text-align: left;
      
      .dark & {
        border-color: $dark-bg-tertiary;
      }
      
      &:first-child {
        font-weight: $font-weight-medium;
      }
    }
    
    thead .table-cell {
      background: map-get($gray-colors, 100);
      font-weight: $font-weight-semibold;
      
      .dark & {
        background: $dark-bg-tertiary;
      }
    }
  }
}
</style>