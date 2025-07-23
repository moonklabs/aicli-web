<template>
  <div class="terminal-controls" :class="{ 'terminal-controls--compact': compact }">
    <div class="controls-section">
      <h4 class="section-title">연결</h4>

      <div class="control-group">
        <NButton
          type="primary"
          size="small"
          @click="$emit('connect')"
          :disabled="isConnected || isConnecting"
          :loading="isConnecting"
        >
          <template #icon>
            <Icon name="play" />
          </template>
          연결
        </NButton>

        <NButton
          size="small"
          @click="$emit('disconnect')"
          :disabled="!isConnected"
        >
          <template #icon>
            <Icon name="stop" />
          </template>
          연결 해제
        </NButton>

        <NButton
          size="small"
          @click="$emit('reconnect')"
          :disabled="isConnecting"
          :loading="isConnecting"
        >
          재연결
        </NButton>
      </div>
    </div>

    <div class="controls-section">
      <h4 class="section-title">터미널</h4>

      <div class="control-group">
        <NButton
          size="small"
          @click="$emit('clear')"
          :disabled="!hasLogs"
        >
          <template #icon>
            <Icon name="clear" />
          </template>
          클리어
        </NButton>

        <NButton
          size="small"
          @click="showExportModal = true"
          :disabled="!hasLogs"
        >
          <template #icon>
            <Icon name="download" />
          </template>
          내보내기
        </NButton>

        <NButton
          size="small"
          @click="toggleFullscreen"
        >
          <template #icon>
            <Icon :name="isFullscreen ? 'fullscreen-exit' : 'fullscreen'" />
          </template>
          {{ isFullscreen ? '전체화면 해제' : '전체화면' }}
        </NButton>
      </div>
    </div>

    <div class="controls-section">
      <h4 class="section-title">표시 옵션</h4>

      <div class="control-group vertical">
        <NCheckbox
          :checked="autoScroll"
          @update:checked="$emit('update:autoScroll', $event)"
        >
          자동 스크롤
        </NCheckbox>

        <NCheckbox
          :checked="showTimestamp"
          @update:checked="$emit('update:showTimestamp', $event)"
        >
          타임스탬프 표시
        </NCheckbox>

        <NCheckbox
          :checked="wrapText"
          @update:checked="$emit('update:wrapText', $event)"
        >
          텍스트 줄바꿈
        </NCheckbox>

        <NCheckbox
          :checked="enableVirtualScrolling"
          @update:checked="$emit('update:enableVirtualScrolling', $event)"
        >
          가상 스크롤링
        </NCheckbox>
      </div>
    </div>

    <div class="controls-section">
      <h4 class="section-title">폰트 설정</h4>

      <div class="control-group vertical">
        <div class="control-item">
          <label class="control-label">폰트 크기</label>
          <NSlider
            :value="fontSize"
            @update:value="$emit('update:fontSize', $event)"
            :min="8"
            :max="24"
            :step="1"
            style="flex: 1"
          />
          <span class="control-value">{{ fontSize }}px</span>
        </div>

        <div class="control-item">
          <label class="control-label">라인 높이</label>
          <NSlider
            :value="lineHeight"
            @update:value="$emit('update:lineHeight', $event)"
            :min="1.0"
            :max="2.0"
            :step="0.1"
            style="flex: 1"
          />
          <span class="control-value">{{ lineHeight.toFixed(1) }}</span>
        </div>
      </div>
    </div>

    <div class="controls-section">
      <h4 class="section-title">성능</h4>

      <div class="control-group vertical">
        <div class="control-item">
          <label class="control-label">최대 라인 수</label>
          <NInputNumber
            :value="maxLines"
            @update:value="(val) => $emit('update:maxLines', val || 1000)"
            :min="100"
            :max="10000"
            :step="100"
            size="small"
            style="width: 100px"
          />
        </div>

        <div class="control-item">
          <label class="control-label">버퍼 크기</label>
          <NInputNumber
            :value="bufferSize"
            @update:value="(val) => $emit('update:bufferSize', val || 50)"
            :min="10"
            :max="200"
            :step="10"
            size="small"
            style="width: 100px"
          />
        </div>
      </div>
    </div>

    <div v-if="performanceStats" class="controls-section">
      <h4 class="section-title">성능 통계</h4>

      <div class="stats-grid">
        <div class="stat-item">
          <span class="stat-label">총 라인</span>
          <span class="stat-value">{{ performanceStats.totalLogs.toLocaleString() }}</span>
        </div>

        <div class="stat-item">
          <span class="stat-label">메모리 사용량</span>
          <span class="stat-value">{{ formatFileSize(performanceStats.memoryUsage) }}</span>
        </div>

        <div class="stat-item">
          <span class="stat-label">렌더링 시간</span>
          <span class="stat-value">{{ performanceStats.renderTime.toFixed(1) }}ms</span>
        </div>

        <div class="stat-item">
          <span class="stat-label">배치 업데이트</span>
          <span class="stat-value">{{ performanceStats.batchedUpdates }}</span>
        </div>
      </div>

      <div class="performance-indicator">
        <div
          class="performance-bar"
          :class="getPerformanceClass()"
          :style="{ width: getPerformancePercentage() + '%' }"
        />
        <span class="performance-text">
          {{ getPerformanceText() }}
        </span>
      </div>
    </div>

    <!-- 내보내기 모달 -->
    <NModal v-model:show="showExportModal" title="로그 내보내기">
      <NCard style="width: 400px">
        <template #header>
          <span>내보내기 옵션</span>
        </template>

        <div class="export-options">
          <div class="export-format">
            <label class="control-label">포맷 선택</label>
            <NRadioGroup v-model:value="exportFormat">
              <NSpace vertical>
                <NRadio value="text">텍스트 (.txt)</NRadio>
                <NRadio value="html">HTML (.html)</NRadio>
                <NRadio value="json">JSON (.json)</NRadio>
              </NSpace>
            </NRadioGroup>
          </div>

          <div class="export-range">
            <label class="control-label">범위 선택</label>
            <NRadioGroup v-model:value="exportRange">
              <NSpace vertical>
                <NRadio value="all">전체 로그</NRadio>
                <NRadio value="visible">현재 보이는 로그</NRadio>
                <NRadio value="last">최근 {{ lastLinesCount }}줄</NRadio>
              </NSpace>
            </NRadioGroup>
          </div>

          <div v-if="exportRange === 'last'" class="last-lines-input">
            <label class="control-label">라인 수</label>
            <NInputNumber
              v-model:value="lastLinesCount"
              :min="1"
              :max="1000"
              size="small"
            />
          </div>
        </div>

        <template #action>
          <NSpace>
            <NButton @click="showExportModal = false">취소</NButton>
            <NButton type="primary" @click="handleExport">내보내기</NButton>
          </NSpace>
        </template>
      </NCard>
    </NModal>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import {
  NButton,
  NCard,
  NCheckbox,
  NInputNumber,
  NModal,
  NRadio,
  NRadioGroup,
  NSlider,
  NSpace,
} from 'naive-ui'
import Icon from '@/components/common/Icon.vue'
import { formatFileSize } from '@/utils/terminal-utils'

interface Props {
  isConnected?: boolean
  isConnecting?: boolean
  hasLogs?: boolean
  isFullscreen?: boolean
  autoScroll?: boolean
  showTimestamp?: boolean
  wrapText?: boolean
  enableVirtualScrolling?: boolean
  fontSize?: number
  lineHeight?: number
  maxLines?: number
  bufferSize?: number
  performanceStats?: {
    totalLogs: number
    memoryUsage: number
    renderTime: number
    batchedUpdates: number
    isOptimal?: boolean
  } | null
  compact?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isConnected: false,
  isConnecting: false,
  hasLogs: false,
  isFullscreen: false,
  autoScroll: true,
  showTimestamp: true,
  wrapText: true,
  enableVirtualScrolling: true,
  fontSize: 14,
  lineHeight: 1.4,
  maxLines: 1000,
  bufferSize: 50,
  performanceStats: null,
  compact: false,
})

const emit = defineEmits<{
  connect: []
  disconnect: []
  reconnect: []
  clear: []
  export: [format: string, range: string, count?: number]
  'update:autoScroll': [value: boolean]
  'update:showTimestamp': [value: boolean]
  'update:wrapText': [value: boolean]
  'update:enableVirtualScrolling': [value: boolean]
  'update:fontSize': [value: number]
  'update:lineHeight': [value: number]
  'update:maxLines': [value: number]
  'update:bufferSize': [value: number]
}>()

// 내보내기 모달 상태
const showExportModal = ref(false)
const exportFormat = ref('text')
const exportRange = ref('all')
const lastLinesCount = ref(100)

// 전체화면 토글
const toggleFullscreen = () => {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen?.()
  } else {
    document.exitFullscreen?.()
  }
}

// 성능 관련 메서드
const getPerformanceClass = () => {
  if (!props.performanceStats) return 'good'

  const { renderTime, memoryUsage, isOptimal } = props.performanceStats

  if (isOptimal) return 'excellent'
  if (renderTime < 32 && memoryUsage < 20 * 1024 * 1024) return 'good'
  if (renderTime < 50 && memoryUsage < 50 * 1024 * 1024) return 'fair'
  return 'poor'
}

const getPerformancePercentage = () => {
  if (!props.performanceStats) return 100

  const { renderTime } = props.performanceStats
  // 16ms = 60fps = 100%, 32ms = 30fps = 50%
  return Math.max(10, Math.min(100, 100 - (renderTime - 16) * 2))
}

const getPerformanceText = () => {
  const perfClass = getPerformanceClass()
  const percentage = getPerformancePercentage()

  const texts = {
    excellent: `최적 (${percentage}%)`,
    good: `양호 (${percentage}%)`,
    fair: `보통 (${percentage}%)`,
    poor: `저하 (${percentage}%)`,
  }

  return texts[perfClass as keyof typeof texts] || '알 수 없음'
}

// 내보내기 처리
const handleExport = () => {
  let count: number | undefined

  if (exportRange.value === 'last') {
    count = lastLinesCount.value
  }

  emit('export', exportFormat.value, exportRange.value, count)
  showExportModal.value = false
}
</script>

<style lang="scss" scoped>
.terminal-controls {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
  background: #2a2a2a;
  border-radius: 8px;
  color: #ffffff;
  width: 280px;
  max-height: 100%;
  overflow-y: auto;

  &--compact {
    width: 240px;
    gap: 12px;
    padding: 12px;

    .controls-section {
      gap: 8px;
    }

    .section-title {
      font-size: 12px;
      margin-bottom: 6px;
    }
  }

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: rgba(255, 255, 255, 0.1);
    border-radius: 3px;
  }

  &::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.3);
    border-radius: 3px;

    &:hover {
      background: rgba(255, 255, 255, 0.5);
    }
  }
}

.controls-section {
  display: flex;
  flex-direction: column;
  gap: 12px;

  + .controls-section {
    border-top: 1px solid rgba(255, 255, 255, 0.1);
    padding-top: 16px;
  }
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #ffffff;
  margin: 0 0 8px 0;
  padding-bottom: 4px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
}

.control-group {
  display: flex;
  gap: 8px;

  &.vertical {
    flex-direction: column;
    gap: 12px;
  }
}

.control-item {
  display: flex;
  align-items: center;
  gap: 8px;

  .control-label {
    font-size: 12px;
    color: rgba(255, 255, 255, 0.8);
    min-width: 70px;
    flex-shrink: 0;
  }

  .control-value {
    font-size: 12px;
    color: rgba(255, 255, 255, 0.6);
    min-width: 40px;
    text-align: right;
    flex-shrink: 0;
  }
}

.stats-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 2px;

  .stat-label {
    font-size: 11px;
    color: rgba(255, 255, 255, 0.6);
  }

  .stat-value {
    font-size: 13px;
    font-weight: 600;
    color: #ffffff;
  }
}

.performance-indicator {
  position: relative;
  width: 100%;
  height: 20px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  overflow: hidden;
  display: flex;
  align-items: center;

  .performance-bar {
    height: 100%;
    transition: width 0.3s ease;
    border-radius: 10px;

    &.excellent {
      background: linear-gradient(90deg, #00ff00, #32cd32);
    }

    &.good {
      background: linear-gradient(90deg, #32cd32, #ffa500);
    }

    &.fair {
      background: linear-gradient(90deg, #ffa500, #ff6b6b);
    }

    &.poor {
      background: linear-gradient(90deg, #ff6b6b, #dc143c);
    }
  }

  .performance-text {
    position: absolute;
    left: 50%;
    transform: translateX(-50%);
    font-size: 11px;
    font-weight: 600;
    color: #000000;
    text-shadow: 0 0 2px rgba(255, 255, 255, 0.8);
    z-index: 1;
  }
}

.export-options {
  display: flex;
  flex-direction: column;
  gap: 16px;

  .export-format,
  .export-range {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .last-lines-input {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
  }

  .control-label {
    font-weight: 600;
    color: #333;
  }
}

// 다크 테마 스타일 (NaiveUI 컴포넌트들)
:deep(.n-button) {
  --n-text-color: #ffffff;
  --n-text-color-hover: #ffffff;
  --n-text-color-pressed: #ffffff;
  --n-text-color-focus: #ffffff;
  --n-text-color-disabled: rgba(255, 255, 255, 0.3);
}

:deep(.n-checkbox) {
  --n-label-text-color: rgba(255, 255, 255, 0.9);
  --n-label-text-color-hover: #ffffff;
  --n-label-text-color-checked: #ffffff;
}

:deep(.n-slider) {
  --n-rail-color: rgba(255, 255, 255, 0.2);
  --n-rail-color-hover: rgba(255, 255, 255, 0.3);
  --n-fill-color: #18a058;
  --n-fill-color-hover: #36ad6a;
}

:deep(.n-input-number) {
  --n-text-color: #ffffff;
  --n-text-color-disabled: rgba(255, 255, 255, 0.3);
  --n-color: rgba(255, 255, 255, 0.1);
  --n-color-disabled: rgba(255, 255, 255, 0.05);
  --n-border-color: rgba(255, 255, 255, 0.2);
  --n-border-color-hover: rgba(255, 255, 255, 0.3);
  --n-border-color-focus: #18a058;
}

// 반응형
@media (max-width: 768px) {
  .terminal-controls {
    width: 100%;
    max-width: none;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .control-group:not(.vertical) {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>