<template>
  <div class="gesture-demo">
    <div class="demo-header">
      <h2>고급 제스처 데모</h2>
      <p>다양한 터치 제스처를 테스트해보세요</p>
    </div>

    <!-- 제스처 테스트 영역 -->
    <div
      ref="gestureArea"
      class="gesture-test-area"
      :class="{ 'active': isGesturing }"
    >
      <div class="gesture-info">
        <div v-if="!currentGesture" class="placeholder">
          이 영역에서 다양한 제스처를 시도해보세요
        </div>
        <div v-else class="current-gesture">
          <h3>{{ gestureInfo.title }}</h3>
          <p>{{ gestureInfo.description }}</p>
          <div class="gesture-data">
            <span v-if="gestureInfo.fingerCount">
              손가락: {{ gestureInfo.fingerCount }}개
            </span>
            <span v-if="gestureInfo.velocity">
              속도: {{ gestureInfo.velocity.toFixed(2) }}
            </span>
            <span v-if="gestureInfo.rotation">
              회전: {{ gestureInfo.rotation.toFixed(2) }}°
            </span>
          </div>
        </div>
      </div>

      <!-- 시각적 피드백 -->
      <div
        v-for="touchPoint in activeTouches"
        :key="touchPoint.id"
        class="touch-indicator"
        :style="{
          left: `${touchPoint.x}px`,
          top: `${touchPoint.y}px`,
        }"
      />

      <!-- 중심점 표시 -->
      <div
        v-if="centroid"
        class="centroid-indicator"
        :style="{
          left: `${centroid.x}px`,
          top: `${centroid.y}px`,
        }"
      />
    </div>

    <!-- 제스처 로그 -->
    <div class="gesture-log">
      <h3>제스처 로그</h3>
      <div class="log-entries">
        <div
          v-for="(entry, index) in gestureLog"
          :key="index"
          class="log-entry"
          :class="`log-entry--${entry.type}`"
        >
          <span class="log-time">{{ entry.timestamp }}</span>
          <span class="log-type">{{ entry.name }}</span>
          <span class="log-details">{{ entry.details }}</span>
        </div>
      </div>
    </div>

    <!-- 제스처 설정 -->
    <div class="gesture-settings">
      <h3>제스처 설정</h3>
      <div class="settings-grid">
        <label class="setting-item">
          <input
            v-model="settings.enableRotation"
            type="checkbox"
          />
          회전 제스처
        </label>
        <label class="setting-item">
          <input
            v-model="settings.enableMultiFingerGestures"
            type="checkbox"
          />
          멀티 핑거
        </label>
        <label class="setting-item">
          <input
            v-model="settings.enableVelocityGestures"
            type="checkbox"
          />
          속도 기반 제스처
        </label>
        <label class="setting-item">
          <input
            v-model="settings.enableInertia"
            type="checkbox"
          />
          관성 효과
        </label>
        <label class="setting-item">
          <input
            v-model="settings.enableHapticFeedback"
            type="checkbox"
          />
          햅틱 피드백
        </label>
        <label class="setting-item">
          <input
            v-model="settings.enableSoundFeedback"
            type="checkbox"
          />
          사운드 피드백
        </label>
      </div>
    </div>

    <!-- 제스처 가이드 -->
    <div class="gesture-guide">
      <h3>제스처 가이드</h3>
      <div class="guide-grid">
        <div class="guide-item">
          <div class="guide-icon">👆</div>
          <div class="guide-text">
            <strong>탭</strong><br />
            한 번 터치
          </div>
        </div>
        <div class="guide-item">
          <div class="guide-icon">👆👆</div>
          <div class="guide-text">
            <strong>더블 탭</strong><br />
            빠르게 두 번 터치
          </div>
        </div>
        <div class="guide-item">
          <div class="guide-icon">✋</div>
          <div class="guide-text">
            <strong>롱 프레스</strong><br />
            길게 누르기
          </div>
        </div>
        <div class="guide-item">
          <div class="guide-icon">👈</div>
          <div class="guide-text">
            <strong>스와이프</strong><br />
            빠르게 밀기
          </div>
        </div>
        <div class="guide-item">
          <div class="guide-icon">🤏</div>
          <div class="guide-text">
            <strong>핀치</strong><br />
            두 손가락으로 확대/축소
          </div>
        </div>
        <div class="guide-item">
          <div class="guide-icon">🔄</div>
          <div class="guide-text">
            <strong>회전</strong><br />
            두 손가락으로 회전
          </div>
        </div>
        <div class="guide-item">
          <div class="guide-icon">👋</div>
          <div class="guide-text">
            <strong>3핑거</strong><br />
            세 손가락 제스처
          </div>
        </div>
        <div class="guide-item">
          <div class="guide-icon">⚡</div>
          <div class="guide-text">
            <strong>빠른 스와이프</strong><br />
            고속 제스처
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useAdvancedGestures } from '@/composables/useAdvancedGestures'

interface TouchPoint {
  id: number
  x: number
  y: number
}

interface GestureLogEntry {
  name: string
  type: 'basic' | 'advanced' | 'custom'
  details: string
  timestamp: string
}

interface GestureInfo {
  title: string
  description: string
  fingerCount?: number
  velocity?: number
  rotation?: number
}

// 참조
const gestureArea = ref<HTMLElement>()

// 상태
const currentGesture = ref<string>('')
const activeTouches = ref<TouchPoint[]>([])
const centroid = ref<{ x: number; y: number } | null>(null)
const gestureLog = ref<GestureLogEntry[]>([])

// 설정
const settings = reactive({
  enableRotation: true,
  enableMultiFingerGestures: true,
  enableVelocityGestures: true,
  enableInertia: true,
  enableHapticFeedback: true,
  enableSoundFeedback: false,
})

// 고급 제스처 설정
const advancedGestures = useAdvancedGestures(gestureArea, {
  enableRotation: settings.enableRotation,
  enableMultiFingerGestures: settings.enableMultiFingerGestures,
  enableVelocityGestures: settings.enableVelocityGestures,
  enableInertia: settings.enableInertia,
  enableHapticFeedback: settings.enableHapticFeedback,
  enableSoundFeedback: settings.enableSoundFeedback,
})

// 현재 제스처 정보
const gestureInfo = computed<GestureInfo>(() => {
  const gestureMap: Record<string, GestureInfo> = {
    tap: {
      title: '탭',
      description: '화면을 한 번 터치했습니다',
      fingerCount: 1,
    },
    doubletap: {
      title: '더블 탭',
      description: '빠르게 두 번 터치했습니다',
      fingerCount: 1,
    },
    longpress: {
      title: '롱 프레스',
      description: '화면을 길게 눌렀습니다',
      fingerCount: 1,
    },
    swipe: {
      title: '스와이프',
      description: '화면을 밀었습니다',
      fingerCount: 1,
    },
    fastswipe: {
      title: '빠른 스와이프',
      description: '화면을 빠르게 밀었습니다',
      fingerCount: 1,
    },
    pan: {
      title: '팬',
      description: '화면을 드래그했습니다',
      fingerCount: 1,
    },
    pinch: {
      title: '핀치',
      description: '두 손가락으로 확대/축소했습니다',
      fingerCount: 2,
    },
    rotation: {
      title: '회전',
      description: '두 손가락으로 회전했습니다',
      fingerCount: 2,
    },
    threefinger: {
      title: '3핑거 제스처',
      description: '세 손가락 제스처를 사용했습니다',
      fingerCount: 3,
    },
    fourfinger: {
      title: '4핑거 제스처',
      description: '네 손가락 제스처를 사용했습니다',
      fingerCount: 4,
    },
    fivefinger: {
      title: '5핑거 제스처',
      description: '다섯 손가락 제스처를 사용했습니다',
      fingerCount: 5,
    },
    pattern_circle: {
      title: '원형 패턴',
      description: '원을 그렸습니다',
      fingerCount: 1,
    },
    pattern_zigzag: {
      title: '지그재그 패턴',
      description: '지그재그를 그렸습니다',
      fingerCount: 1,
    },
  }

  return gestureMap[currentGesture.value] || {
    title: '알 수 없는 제스처',
    description: '감지된 제스처 정보가 없습니다',
  }
})

// 제스처 중인지 여부
const isGesturing = computed(() => advancedGestures.isGesturing.value)

// 로그 추가 함수
const addLogEntry = (name: string, type: 'basic' | 'advanced' | 'custom', details: string) => {
  gestureLog.value.unshift({
    name,
    type,
    details,
    timestamp: new Date().toLocaleTimeString(),
  })

  // 로그 크기 제한
  if (gestureLog.value.length > 20) {
    gestureLog.value = gestureLog.value.slice(0, 20)
  }
}

// 기본 제스처 이벤트 리스너
advancedGestures.on('tap', (event) => {
  currentGesture.value = 'tap'
  addLogEntry('탭', 'basic', `위치: (${event.centroid.x.toFixed(0)}, ${event.centroid.y.toFixed(0)})`)
})

advancedGestures.on('longpress', (event) => {
  currentGesture.value = 'longpress'
  addLogEntry('롱 프레스', 'basic', `위치: (${event.centroid.x.toFixed(0)}, ${event.centroid.y.toFixed(0)})`)
})

advancedGestures.on('swipe', (event) => {
  currentGesture.value = 'swipe'
  addLogEntry('스와이프', 'basic', `속도: ${event.velocity.magnitude.toFixed(2)}`)
})

advancedGestures.on('fastswipe', (event) => {
  currentGesture.value = 'fastswipe'
  addLogEntry('빠른 스와이프', 'advanced', `고속 제스처 - 속도: ${event.velocity.magnitude.toFixed(2)}`)
})

advancedGestures.on('pan', (event) => {
  currentGesture.value = 'pan'
  centroid.value = event.centroid
})

advancedGestures.on('pinch', (event) => {
  currentGesture.value = 'pinch'
  centroid.value = event.centroid
  addLogEntry('핀치', 'basic', `손가락: ${event.fingerCount}개`)
})

advancedGestures.on('rotation', (event) => {
  currentGesture.value = 'rotation'
  centroid.value = event.centroid
  if (event.rotation) {
    addLogEntry('회전', 'advanced', `각도: ${(event.rotation * 180 / Math.PI).toFixed(1)}°`)
  }
})

// 멀티 핑거 제스처
advancedGestures.on('threefinger', (event) => {
  currentGesture.value = 'threefinger'
  centroid.value = event.centroid
  addLogEntry('3핑거 제스처', 'advanced', `손가락: ${event.fingerCount}개`)
})

advancedGestures.on('fourfinger', (event) => {
  currentGesture.value = 'fourfinger'
  centroid.value = event.centroid
  addLogEntry('4핑거 제스처', 'advanced', `손가락: ${event.fingerCount}개`)
})

advancedGestures.on('fivefinger', (event) => {
  currentGesture.value = 'fivefinger'
  centroid.value = event.centroid
  addLogEntry('5핑거 제스처', 'advanced', `손가락: ${event.fingerCount}개`)
})

// 커스텀 패턴
advancedGestures.on('pattern_circle', (event) => {
  currentGesture.value = 'pattern_circle'
  addLogEntry('원형 패턴', 'custom', '원을 그렸습니다')
})

advancedGestures.on('pattern_zigzag', (event) => {
  currentGesture.value = 'pattern_zigzag'
  addLogEntry('지그재그 패턴', 'custom', '지그재그를 그렸습니다')
})

// 관성 이벤트
advancedGestures.on('inertiaframe', (event) => {
  if (event.inertia) {
    addLogEntry('관성', 'advanced', `속도: ${Math.sqrt(event.inertia.x ** 2 + event.inertia.y ** 2).toFixed(2)}`)
  }
})

// 제스처 종료
advancedGestures.on('gestureend', () => {
  setTimeout(() => {
    currentGesture.value = ''
    activeTouches.value = []
    centroid.value = null
  }, 1000)
})

// 설정 변경 감지
watch(settings, (newSettings) => {
  // 고급 제스처 설정 업데이트 (실제로는 재초기화 필요)
  console.log('제스처 설정 변경:', newSettings)
}, { deep: true })
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.gesture-demo {
  padding: $spacing-6;
  max-width: 800px;
  margin: 0 auto;

  @include mobile {
    padding: $spacing-4;
  }
}

.demo-header {
  text-align: center;
  margin-bottom: $spacing-8;

  h2 {
    font-size: $font-size-2xl;
    color: $light-text-primary;
    margin-bottom: $spacing-2;

    .dark & {
      color: $dark-text-primary;
    }
  }

  p {
    color: $light-text-secondary;
    font-size: $font-size-base;

    .dark & {
      color: $dark-text-secondary;
    }
  }
}

.gesture-test-area {
  position: relative;
  min-height: 300px;
  border: 2px dashed map-get($gray-colors, 300);
  border-radius: $border-radius-lg;
  background: map-get($gray-colors, 50);
  margin-bottom: $spacing-6;
  overflow: hidden;
  user-select: none;
  touch-action: none;

  .dark & {
    border-color: $dark-bg-tertiary;
    background: $dark-bg-secondary;
  }

  &.active {
    border-color: map-get($primary-colors, 400);
    background: map-get($primary-colors, 50);

    .dark & {
      border-color: map-get($primary-colors, 500);
      background: rgba(map-get($primary-colors, 500), 0.1);
    }
  }

  .gesture-info {
    @include flex-center;
    height: 100%;
    padding: $spacing-6;
    text-align: center;

    .placeholder {
      color: $light-text-secondary;
      font-style: italic;

      .dark & {
        color: $dark-text-secondary;
      }
    }

    .current-gesture {
      h3 {
        font-size: $font-size-xl;
        color: map-get($primary-colors, 600);
        margin-bottom: $spacing-2;

        .dark & {
          color: map-get($primary-colors, 400);
        }
      }

      p {
        color: $light-text-secondary;
        margin-bottom: $spacing-3;

        .dark & {
          color: $dark-text-secondary;
        }
      }

      .gesture-data {
        display: flex;
        gap: $spacing-4;
        justify-content: center;
        font-size: $font-size-sm;

        span {
          padding: $spacing-1 $spacing-2;
          background: map-get($gray-colors, 100);
          border-radius: $border-radius-sm;
          color: $light-text-primary;

          .dark & {
            background: $dark-bg-tertiary;
            color: $dark-text-primary;
          }
        }
      }
    }
  }

  .touch-indicator {
    position: absolute;
    width: 40px;
    height: 40px;
    border: 2px solid map-get($primary-colors, 500);
    border-radius: 50%;
    background: rgba(map-get($primary-colors, 500), 0.2);
    transform: translate(-50%, -50%);
    pointer-events: none;
    animation: pulse 1s infinite;
  }

  .centroid-indicator {
    position: absolute;
    width: 20px;
    height: 20px;
    background: $warning;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    pointer-events: none;
    box-shadow: 0 0 0 4px rgba($warning, 0.3);
  }
}

.gesture-log {
  margin-bottom: $spacing-6;

  h3 {
    font-size: $font-size-lg;
    margin-bottom: $spacing-4;
    color: $light-text-primary;

    .dark & {
      color: $dark-text-primary;
    }
  }

  .log-entries {
    max-height: 200px;
    overflow-y: auto;
    border: 1px solid map-get($gray-colors, 200);
    border-radius: $border-radius-md;
    background: map-get($gray-colors, 50);

    .dark & {
      border-color: $dark-bg-tertiary;
      background: $dark-bg-secondary;
    }
  }

  .log-entry {
    display: grid;
    grid-template-columns: auto 1fr auto;
    gap: $spacing-3;
    padding: $spacing-2 $spacing-3;
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

    .log-type {
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

    &--basic {
      .log-type {
        color: map-get($primary-colors, 600);

        .dark & {
          color: map-get($primary-colors, 400);
        }
      }
    }

    &--advanced {
      .log-type {
        color: $warning;
      }
    }

    &--custom {
      .log-type {
        color: $success;
      }
    }
  }
}

.gesture-settings {
  margin-bottom: $spacing-6;

  h3 {
    font-size: $font-size-lg;
    margin-bottom: $spacing-4;
    color: $light-text-primary;

    .dark & {
      color: $dark-text-primary;
    }
  }

  .settings-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: $spacing-3;
  }

  .setting-item {
    @include flex-center;
    gap: $spacing-2;
    padding: $spacing-3;
    border: 1px solid map-get($gray-colors, 200);
    border-radius: $border-radius-md;
    background: map-get($gray-colors, 50);
    cursor: pointer;
    transition: $transition-base;

    .dark & {
      border-color: $dark-bg-tertiary;
      background: $dark-bg-secondary;
    }

    &:hover {
      border-color: map-get($primary-colors, 300);
      background: map-get($primary-colors, 50);

      .dark & {
        border-color: map-get($primary-colors, 500);
        background: rgba(map-get($primary-colors, 500), 0.1);
      }
    }

    input[type="checkbox"] {
      width: 18px;
      height: 18px;
    }
  }
}

.gesture-guide {
  h3 {
    font-size: $font-size-lg;
    margin-bottom: $spacing-4;
    color: $light-text-primary;

    .dark & {
      color: $dark-text-primary;
    }
  }

  .guide-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: $spacing-4;
  }

  .guide-item {
    text-align: center;
    padding: $spacing-4;
    border: 1px solid map-get($gray-colors, 200);
    border-radius: $border-radius-md;
    background: map-get($gray-colors, 50);

    .dark & {
      border-color: $dark-bg-tertiary;
      background: $dark-bg-secondary;
    }

    .guide-icon {
      font-size: $font-size-2xl;
      margin-bottom: $spacing-2;
    }

    .guide-text {
      font-size: $font-size-sm;
      color: $light-text-secondary;

      .dark & {
        color: $dark-text-secondary;
      }

      strong {
        color: $light-text-primary;
        display: block;
        margin-bottom: $spacing-1;

        .dark & {
          color: $dark-text-primary;
        }
      }
    }
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
  50% {
    opacity: 0.7;
    transform: translate(-50%, -50%) scale(1.1);
  }
}
</style>