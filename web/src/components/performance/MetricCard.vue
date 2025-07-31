<template>
  <div class="metric-card" :class="status">
    <div class="metric-header">
      <h4 class="metric-title">{{ title }}</h4>
      <div class="metric-status-indicator" :class="status" :aria-label="getStatusLabel(status)">
        <span class="status-dot"></span>
      </div>
    </div>
    
    <div class="metric-value">
      <span class="value">{{ formattedValue }}</span>
      <span class="unit" v-if="unit">{{ unit }}</span>
    </div>
    
    <div class="metric-description">
      {{ description }}
    </div>
    
    <div class="metric-target" v-if="target !== null">
      <span class="target-label">목표:</span>
      <span class="target-value">{{ formatValue(target, unit) }}</span>
    </div>
    
    <!-- 성능 바 -->
    <div class="performance-bar" v-if="showBar">
      <div 
        class="performance-fill" 
        :style="{ width: progressPercentage + '%' }"
        :class="status"
      ></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  title: string
  description?: string
  value: number | null
  target: number | null
  unit?: string
  status?: 'good' | 'warning' | 'error' | 'unknown'
  showBar?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  description: '',
  target: null,
  unit: '',
  status: 'unknown',
  showBar: true,
})

// 값 포맷팅
const formatValue = (value: number | null, unit: string): string => {
  if (value === null) return 'N/A'
  
  switch (unit) {
    case 'ms':
      return `${Math.round(value)}ms`
    case 'KB':
      return `${Math.round(value / 1024)}KB`
    case 'MB':
      return `${(value / 1024 / 1024).toFixed(1)}MB`
    case '개':
      return `${value}개`
    case '':
      return value.toFixed(3)
    default:
      return value.toString()
  }
}

// 포맷된 값
const formattedValue = computed(() => formatValue(props.value, props.unit))

// 진행률 계산
const progressPercentage = computed(() => {
  if (props.value === null || props.target === null) return 0
  
  // CLS나 작은 값들의 경우 역방향 계산
  if (props.target <= 1) {
    const ratio = props.value / props.target
    return Math.min(100, Math.max(0, 100 - (ratio - 1) * 100))
  }
  
  // 시간 기반 메트릭의 경우
  const ratio = props.value / props.target
  return Math.min(100, Math.max(0, 100 - (ratio - 1) * 50))
})

// 상태 라벨
const getStatusLabel = (status: string): string => {
  const labels = {
    good: '양호',
    warning: '경고',
    error: '오류',
    unknown: '알 수 없음'
  }
  return labels[status as keyof typeof labels] || '알 수 없음'
}
</script>

<style scoped lang="scss">
.metric-card {
  padding: var(--spacing-md);
  background: var(--bg-elevated);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-primary);
  transition: var(--duration-normal) var(--ease-in-out);

  &:hover {
    border-color: var(--border-secondary);
    box-shadow: var(--shadow-sm);
  }

  &.good {
    --metric-color: var(--success-500);
    --metric-bg: var(--success-50);
  }

  &.warning {
    --metric-color: var(--warning-500);
    --metric-bg: var(--warning-50);
  }

  &.error {
    --metric-color: var(--error-500);
    --metric-bg: var(--error-50);
  }

  &.unknown {
    --metric-color: var(--gray-400);
    --metric-bg: var(--gray-50);
  }
}

.metric-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--spacing-sm);
}

.metric-title {
  margin: 0;
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  line-height: var(--leading-snug);
}

.metric-status-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--metric-bg);
  flex-shrink: 0;

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--metric-color);
  }
}

.metric-value {
  display: flex;
  align-items: baseline;
  gap: var(--spacing-xs);
  margin-bottom: var(--spacing-xs);

  .value {
    font-size: var(--text-xl);
    font-weight: var(--font-bold);
    color: var(--text-primary);
    line-height: 1;
  }

  .unit {
    font-size: var(--text-sm);
    font-weight: var(--font-medium);
    color: var(--text-secondary);
  }
}

.metric-description {
  font-size: var(--text-xs);
  color: var(--text-tertiary);
  margin-bottom: var(--spacing-sm);
  line-height: var(--leading-snug);
}

.metric-target {
  display: flex;
  gap: var(--spacing-xs);
  font-size: var(--text-xs);
  margin-bottom: var(--spacing-sm);

  .target-label {
    color: var(--text-tertiary);
  }

  .target-value {
    color: var(--text-secondary);
    font-weight: var(--font-medium);
  }
}

.performance-bar {
  width: 100%;
  height: 4px;
  background: var(--gray-200);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.performance-fill {
  height: 100%;
  background: var(--metric-color);
  border-radius: var(--radius-full);
  transition: width var(--duration-slow) var(--ease-out);

  &.good {
    background: var(--success-500);
  }

  &.warning {
    background: var(--warning-500);
  }

  &.error {
    background: var(--error-500);
  }

  &.unknown {
    background: var(--gray-400);
  }
}

// 다크 테마 지원
[data-theme="dark"] {
  .metric-card {
    &.good {
      --metric-bg: rgba(34, 197, 94, 0.1);
    }

    &.warning {
      --metric-bg: rgba(245, 158, 11, 0.1);
    }

    &.error {
      --metric-bg: rgba(239, 68, 68, 0.1);
    }

    &.unknown {
      --metric-bg: rgba(156, 163, 175, 0.1);
    }
  }

  .performance-bar {
    background: var(--gray-700);
  }
}

// 애니메이션 감소 설정
[data-motion-preference="reduce"] {
  .metric-card {
    transition: none !important;
  }

  .performance-fill {
    transition: none !important;
  }
}

// 접근성 향상
.metric-card:focus-within {
  outline: 2px solid var(--focus-ring-color);
  outline-offset: 2px;
}

// 모바일 최적화
@media (max-width: 480px) {
  .metric-card {
    padding: var(--spacing-sm);
  }

  .metric-value .value {
    font-size: var(--text-lg);
  }
}
</style>