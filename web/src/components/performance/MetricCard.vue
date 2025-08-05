<template>
  <div class="metric-card" :class="`status-${status}`">
    <div class="metric-header">
      <div class="metric-icon">
        <component :is="getIcon(icon)" />
      </div>
      <div class="metric-title">
        <h4>{{ title }}</h4>
        <p>{{ subtitle }}</p>
      </div>
    </div>
    
    <div class="metric-value">
      {{ value }}
    </div>
    
    <div class="metric-status">
      <span class="status-indicator"></span>
      <span class="status-text">{{ getStatusText(status) }}</span>
    </div>
    
    <div v-if="threshold" class="metric-thresholds">
      <div class="threshold good">
        <span class="label">Good</span>
        <span class="value">{{ formatThreshold(threshold.good) }}</span>
      </div>
      <div class="threshold poor">
        <span class="label">Poor</span>
        <span class="value">{{ formatThreshold(threshold.poor) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import IconPaint from '@/components/icons/IconPaint.vue'
import IconCursor from '@/components/icons/IconCursor.vue'
import IconLayout from '@/components/icons/IconLayout.vue'
import IconEye from '@/components/icons/IconEye.vue'
import IconClick from '@/components/icons/IconClick.vue'
import IconServer from '@/components/icons/IconServer.vue'

interface Props {
  title: string
  subtitle: string
  value: string
  status: 'good' | 'needs-improvement' | 'poor' | 'unknown'
  threshold?: {
    good: number
    poor: number
  }
  icon: string
}

const props = defineProps<Props>()

// 아이콘 컴포넌트 매핑
const iconMap = {
  paint: IconPaint,
  cursor: IconCursor,
  layout: IconLayout,
  eye: IconEye,
  click: IconClick,
  server: IconServer,
}

const getIcon = (iconName: string) => {
  return iconMap[iconName as keyof typeof iconMap] || IconEye
}

// 상태 텍스트
const getStatusText = (status: string) => {
  switch (status) {
    case 'good':
      return '양호'
    case 'needs-improvement':
      return '개선 필요'
    case 'poor':
      return '불량'
    default:
      return '측정 중'
  }
}

// 임계값 포맷팅
const formatThreshold = (value: number) => {
  if (props.title === 'CLS') {
    return value.toFixed(2)
  }
  return value >= 1000 ? `${(value / 1000).toFixed(1)}s` : `${value}ms`
}
</script>

<style scoped>
.metric-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
  transition: all var(--duration-normal);
}

.metric-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.metric-header {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-md);
  margin-bottom: var(--spacing-lg);
}

.metric-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  color: var(--text-accent);
}

.metric-icon svg {
  width: 24px;
  height: 24px;
}

.metric-title h4 {
  font-size: var(--text-lg);
  font-weight: var(--font-bold);
  color: var(--text-primary);
  margin: 0;
}

.metric-title p {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  margin: var(--spacing-xs) 0 0 0;
}

.metric-value {
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  color: var(--text-primary);
  margin-bottom: var(--spacing-md);
}

.metric-status {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-lg);
}

.status-indicator {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: currentColor;
}

.status-text {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
}

/* 상태별 색상 */
.status-good .status-indicator,
.status-good .status-text {
  color: var(--success-500);
}

.status-needs-improvement .status-indicator,
.status-needs-improvement .status-text {
  color: var(--warning-500);
}

.status-poor .status-indicator,
.status-poor .status-text {
  color: var(--error-500);
}

.status-unknown .status-indicator,
.status-unknown .status-text {
  color: var(--gray-400);
}

.metric-thresholds {
  display: flex;
  gap: var(--spacing-lg);
  padding-top: var(--spacing-md);
  border-top: 1px solid var(--border-primary);
}

.threshold {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.threshold .label {
  font-size: var(--text-xs);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-tertiary);
}

.threshold .value {
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
}

.threshold.good .value {
  color: var(--success-600);
}

.threshold.poor .value {
  color: var(--error-600);
}

/* 상태별 카드 스타일 */
.status-good {
  border-color: var(--success-200);
  background: linear-gradient(
    to bottom,
    var(--bg-secondary),
    rgba(34, 197, 94, 0.05)
  );
}

.status-needs-improvement {
  border-color: var(--warning-200);
  background: linear-gradient(
    to bottom,
    var(--bg-secondary),
    rgba(245, 158, 11, 0.05)
  );
}

.status-poor {
  border-color: var(--error-200);
  background: linear-gradient(
    to bottom,
    var(--bg-secondary),
    rgba(239, 68, 68, 0.05)
  );
}

/* 다크 모드 */
[data-theme='dark'] .status-good {
  border-color: var(--success-800);
  background: linear-gradient(
    to bottom,
    var(--bg-secondary),
    rgba(34, 197, 94, 0.1)
  );
}

[data-theme='dark'] .status-needs-improvement {
  border-color: var(--warning-800);
  background: linear-gradient(
    to bottom,
    var(--bg-secondary),
    rgba(245, 158, 11, 0.1)
  );
}

[data-theme='dark'] .status-poor {
  border-color: var(--error-800);
  background: linear-gradient(
    to bottom,
    var(--bg-secondary),
    rgba(239, 68, 68, 0.1)
  );
}

/* 반응형 */
@media (max-width: 768px) {
  .metric-value {
    font-size: var(--text-2xl);
  }
  
  .metric-thresholds {
    font-size: var(--text-xs);
  }
}
</style>