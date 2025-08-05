<template>
  <component
    :is="terminalComponent"
    v-bind="terminalProps"
    @close="emit('close')"
    @execute-command="emit('executeCommand', $event)"
    @stop-execution="emit('stopExecution')"
    @clear-logs="emit('clearLogs')"
    @export-logs="emit('exportLogs', $event)"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import TerminalEmulator from './TerminalEmulator.vue'
import MobileTerminal from '../ui/mobile/MobileTerminal.vue'

interface TerminalLog {
  id: string
  content: string
  type: 'input' | 'output' | 'error' | 'system'
  timestamp: string
}

interface Props {
  sessionId: string
  sessionName?: string
  logs?: TerminalLog[]
  isConnected?: boolean
  isExecuting?: boolean
  lastActivity?: string
  commandHistory?: string[]
  // Desktop specific
  useVirtualScrolling?: boolean
  lineHeight?: number
  maxLines?: number
  autoScroll?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  sessionName: 'Terminal',
  logs: () => [],
  isConnected: false,
  isExecuting: false,
  commandHistory: () => [],
  useVirtualScrolling: true,
  lineHeight: 24,
  maxLines: 1000,
  autoScroll: true,
})

const emit = defineEmits<{
  close: []
  executeCommand: [command: string]
  stopExecution: []
  clearLogs: []
  exportLogs: [format?: 'text' | 'html' | 'json']
}>()

// 모바일 여부 확인
const isMobile = useMediaQuery('(max-width: 768px)')
const isTouch = useMediaQuery('(pointer: coarse)')

// 터미널 컴포넌트 선택
const terminalComponent = computed(() => {
  // 모바일이거나 터치 디바이스인 경우 모바일 터미널 사용
  return isMobile.value || isTouch.value ? MobileTerminal : TerminalEmulator
})

// Props 매핑
const terminalProps = computed(() => {
  if (terminalComponent.value === MobileTerminal) {
    // 모바일 터미널에 필요한 props만 전달
    return {
      sessionName: props.sessionName,
      logs: props.logs,
      isConnected: props.isConnected,
      isExecuting: props.isExecuting,
      commandHistory: props.commandHistory,
    }
  }
  
  // 데스크톱 터미널에는 모든 props 전달
  return props
})
</script>