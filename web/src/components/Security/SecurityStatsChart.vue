<template>
  <div class="security-stats-chart">
    <div
      ref="chartContainer"
      class="chart-container"
      :style="{ height: chartHeight + 'px' }"
    />
    <div v-if="loading" class="chart-loading">
      <NSpin size="medium" />
    </div>
    <div v-if="error" class="chart-error">
      <NResult
        status="error"
        title="차트 로딩 실패"
        :description="error"
        size="small"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { NResult, NSpin } from 'naive-ui'

interface Props {
  stats: any | null
  chartType: 'login-trends' | 'risk-distribution' | 'security-events' | 'device-analysis'
  height?: number
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  height: 300,
  loading: false,
})

// 상태 관리
const chartContainer = ref<HTMLElement>()
const error = ref<string>('')

// 계산된 속성
const chartHeight = computed(() => props.height)
</script>

<style scoped>
.security-stats-chart {
  position: relative;
  width: 100%;
}

.chart-container {
  width: 100%;
}

.chart-loading {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  align-items: center;
  justify-content: center;
}

.chart-error {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.9);
}
</style>