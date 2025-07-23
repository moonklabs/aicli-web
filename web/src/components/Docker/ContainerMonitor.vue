<template>
  <div class="container-monitor">
    <div class="monitor-header">
      <div class="header-left">
        <h3 class="monitor-title">Docker 컨테이너</h3>
        <n-badge :value="totalContainers" type="info" />
      </div>
      
      <div class="header-actions">
        <n-switch
          v-model:value="isMonitoringEnabled"
          @update:value="toggleMonitoring"
          :loading="connectionStatus === 'connecting'"
        >
          <template #checked>실시간 모니터링</template>
          <template #unchecked>정적 표시</template>
        </n-switch>
        
        <n-button @click="refreshContainers" :loading="isLoading">
          <template #icon>
            <n-icon><RefreshCw /></n-icon>
          </template>
          새로고침
        </n-button>
      </div>
    </div>

    <!-- 연결 상태 표시 -->
    <div v-if="showConnectionStatus" class="connection-status">
      <n-alert
        :type="getConnectionAlertType(connectionStatus)"
        :show-icon="true"
        closable
        @close="showConnectionStatus = false"
      >
        <template #icon>
          <n-icon>
            <Wifi v-if="connectionStatus === 'connected'" />
            <WifiOff v-else />
          </n-icon>
        </template>
        {{ getConnectionStatusText(connectionStatus) }}
      </n-alert>
    </div>

    <!-- 컨테이너 목록 -->
    <div v-if="containerList.length > 0" class="container-list">
      <div
        v-for="container in containerList"
        :key="container.id"
        class="container-item"
        :class="{ 'is-running': container.status === 'running' }"
      >
        <div class="container-info">
          <div class="container-header">
            <h4 class="container-name">{{ container.name }}</h4>
            <n-tag
              :type="getStatusType(container.status)"
              size="small"
              class="container-status-tag"
            >
              <template #icon>
                <n-icon>
                  <Play v-if="container.status === 'running'" />
                  <Square v-else-if="container.status === 'stopped'" />
                  <AlertCircle v-else-if="container.status === 'dead'" />
                  <Pause v-else-if="container.status === 'paused'" />
                  <RotateCw v-else-if="container.status === 'restarting'" />
                </n-icon>
              </template>
              {{ getStatusText(container.status) }}
            </n-tag>
          </div>
          
          <div class="container-details">
            <p class="container-image">
              <n-icon><Box /></n-icon>
              {{ container.image }}
            </p>
            <p v-if="container.workspaceId" class="container-workspace">
              <n-icon><Folder /></n-icon>
              워크스페이스: {{ getWorkspaceName(container.workspaceId) }}
            </p>
          </div>
        </div>

        <!-- 컨테이너 통계 (실행 중인 경우만) -->
        <div v-if="container.status === 'running' && getStats(container.id)" class="container-stats">
          <div class="stats-grid">
            <div class="stat-item">
              <div class="stat-label">CPU</div>
              <n-progress
                :percentage="getStats(container.id)?.cpuPercent || 0"
                type="line"
                :show-indicator="false"
                :height="6"
                :color="getProgressColor(getStats(container.id)?.cpuPercent || 0)"
              />
              <span class="stat-value">{{ (getStats(container.id)?.cpuPercent || 0).toFixed(1) }}%</span>
            </div>
            
            <div class="stat-item">
              <div class="stat-label">메모리</div>
              <n-progress
                :percentage="getStats(container.id)?.memoryPercent || 0"
                type="line"
                :show-indicator="false"
                :height="6"
                :color="getProgressColor(getStats(container.id)?.memoryPercent || 0)"
              />
              <span class="stat-value">
                {{ formatBytes(getStats(container.id)?.memoryUsage || 0) }} / 
                {{ formatBytes(getStats(container.id)?.memoryLimit || 0) }}
              </span>
            </div>
            
            <div class="stat-item">
              <div class="stat-label">네트워크</div>
              <div class="network-stats">
                <span class="network-stat">
                  <n-icon><ArrowDown /></n-icon>
                  {{ formatBytes(getStats(container.id)?.networkRx || 0) }}
                </span>
                <span class="network-stat">
                  <n-icon><ArrowUp /></n-icon>
                  {{ formatBytes(getStats(container.id)?.networkTx || 0) }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 포트 매핑 정보 -->
        <div v-if="container.ports.length > 0" class="container-ports">
          <div class="ports-label">포트:</div>
          <n-space>
            <n-tag
              v-for="port in container.ports"
              :key="`${port.privatePort}-${port.type}`"
              size="small"
              type="info"
            >
              {{ port.publicPort || 'none' }}:{{ port.privatePort }}/{{ port.type }}
            </n-tag>
          </n-space>
        </div>

        <!-- 컨테이너 액션 -->
        <div class="container-actions">
          <n-button-group size="small">
            <n-button
              v-if="container.status === 'stopped'"
              type="primary"
              @click="startContainer(container.id)"
              :loading="isLoading"
            >
              <template #icon>
                <n-icon><Play /></n-icon>
              </template>
              시작
            </n-button>
            
            <n-button
              v-if="container.status === 'running'"
              type="warning"
              @click="stopContainer(container.id)"
              :loading="isLoading"
            >
              <template #icon>
                <n-icon><Square /></n-icon>
              </template>
              중지
            </n-button>
            
            <n-button
              v-if="['running', 'stopped'].includes(container.status)"
              @click="restartContainer(container.id)"
              :loading="isLoading"
            >
              <template #icon>
                <n-icon><RotateCw /></n-icon>
              </template>
              재시작
            </n-button>
            
            <n-button @click="viewLogs(container.id)">
              <template #icon>
                <n-icon><FileText /></n-icon>
              </template>
              로그
            </n-button>
            
            <n-button @click="openTerminal(container.id)">
              <template #icon>
                <n-icon><Terminal /></n-icon>
              </template>
              터미널
            </n-button>
            
            <n-popconfirm
              @positive-click="removeContainer(container.id)"
              negative-text="취소"
              positive-text="삭제"
            >
              <template #trigger>
                <n-button type="error" :loading="isLoading">
                  <template #icon>
                    <n-icon><Trash2 /></n-icon>
                  </template>
                  삭제
                </n-button>
              </template>
              <p>정말로 <strong>{{ container.name }}</strong> 컨테이너를 삭제하시겠습니까?</p>
              <p style="color: var(--n-color-error); font-size: 12px;">
                이 작업은 되돌릴 수 없습니다.
              </p>
            </n-popconfirm>
          </n-button-group>
        </div>
      </div>
    </div>

    <!-- 빈 상태 -->
    <n-empty
      v-else-if="!isLoading"
      description="실행 중인 컨테이너가 없습니다"
      class="container-empty"
    >
      <template #icon>
        <n-icon size="48"><Box /></n-icon>
      </template>
      <template #extra>
        <n-button type="primary" @click="refreshContainers">
          컨테이너 목록 새로고침
        </n-button>
      </template>
    </n-empty>

    <!-- 로딩 상태 -->
    <div v-if="isLoading" class="loading-container">
      <n-spin size="large">
        <template #description>
          컨테이너 정보를 불러오는 중...
        </template>
      </n-spin>
    </div>

    <!-- 컨테이너 로그 모달 -->
    <container-logs-modal
      v-model:show="showLogsModal"
      :container-id="selectedContainerId"
      :container-name="selectedContainerName"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  NButton,
  NButtonGroup,
  NIcon,
  NTag,
  NSpace,
  NSwitch,
  NBadge,
  NProgress,
  NPopconfirm,
  NEmpty,
  NSpin,
  NAlert
} from 'naive-ui'
import {
  RefreshCw,
  Play,
  Square,
  Pause,
  RotateCw,
  AlertCircle,
  Box,
  Folder,
  ArrowDown,
  ArrowUp,
  FileText,
  Terminal,
  Trash2,
  Wifi,
  WifiOff
} from '@vicons/lucide'

import { useDockerStore } from '@/stores/docker'
import { useWorkspaceStore } from '@/stores/workspace'
import ContainerLogsModal from './ContainerLogsModal.vue'

interface Props {
  workspaceId?: string
}

const props = defineProps<Props>()

const dockerStore = useDockerStore()
const workspaceStore = useWorkspaceStore()
const router = useRouter()
const message = useMessage()

// 로컬 상태
const isMonitoringEnabled = ref(false)
const showConnectionStatus = ref(false)
const showLogsModal = ref(false)
const selectedContainerId = ref<string>('')
const selectedContainerName = ref<string>('')

// 계산된 속성
const containerList = computed(() => {
  if (props.workspaceId) {
    return dockerStore.containersByWorkspace(props.workspaceId)
  }
  return dockerStore.containerList
})

const totalContainers = computed(() => containerList.value.length)
const isLoading = computed(() => dockerStore.isLoading)
const isMonitoring = computed(() => dockerStore.isMonitoring)
const connectionStatus = computed(() => dockerStore.connectionStatus)

// 통계 데이터 가져오기
const getStats = (containerId: string) => {
  return dockerStore.stats.get(containerId)
}

// 워크스페이스 이름 가져오기
const getWorkspaceName = (workspaceId: string): string => {
  const workspace = workspaceStore.workspaceById(workspaceId)
  return workspace?.name || workspaceId
}

// 상태별 타입 결정
const getStatusType = (status: string): 'success' | 'warning' | 'error' | 'info' => {
  switch (status) {
    case 'running': return 'success'
    case 'paused': return 'warning'
    case 'restarting': return 'info'
    case 'dead': return 'error'
    default: return 'info'
  }
}

const getStatusText = (status: string): string => {
  switch (status) {
    case 'running': return '실행중'
    case 'stopped': return '중지'
    case 'paused': return '일시정지'
    case 'restarting': return '재시작중'
    case 'dead': return '죽음'
    case 'created': return '생성됨'
    default: return status
  }
}

// 연결 상태 관련
const getConnectionAlertType = (status: string): 'success' | 'warning' | 'error' | 'info' => {
  switch (status) {
    case 'connected': return 'success'
    case 'connecting': return 'info'
    case 'error': return 'error'
    default: return 'warning'
  }
}

const getConnectionStatusText = (status: string): string => {
  switch (status) {
    case 'connected': return 'WebSocket 연결됨 - 실시간 모니터링 활성화'
    case 'connecting': return 'WebSocket 연결 중...'
    case 'error': return 'WebSocket 연결 오류 - 실시간 모니터링 비활성화'
    case 'disconnected': return 'WebSocket 연결 끊김'
    default: return '알 수 없는 상태'
  }
}

// 프로그레스 바 색상
const getProgressColor = (percentage: number): string => {
  if (percentage < 50) return '#52c41a'
  if (percentage < 80) return '#faad14'
  return '#ff4d4f'
}

// 바이트 포맷팅
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

// 이벤트 핸들러
const toggleMonitoring = async (enabled: boolean): Promise<void> => {
  try {
    if (enabled) {
      await dockerStore.startMonitoring(props.workspaceId)
      showConnectionStatus.value = true
      message.success('실시간 모니터링이 시작되었습니다')
    } else {
      dockerStore.stopMonitoring()
      showConnectionStatus.value = false
      message.info('실시간 모니터링이 중지되었습니다')
    }
  } catch (error) {
    message.error('모니터링 설정을 변경할 수 없습니다')
    isMonitoringEnabled.value = !enabled // 원래 상태로 되돌림
  }
}

const refreshContainers = async (): Promise<void> => {
  try {
    await dockerStore.refreshContainers()
    message.success('컨테이너 목록이 새로고침되었습니다')
  } catch (error) {
    message.error('컨테이너 목록 새로고침에 실패했습니다')
  }
}

const startContainer = async (containerId: string): Promise<void> => {
  try {
    const success = await dockerStore.startContainer(containerId)
    if (success) {
      message.success('컨테이너가 시작되었습니다')
    }
  } catch (error) {
    message.error('컨테이너 시작에 실패했습니다')
  }
}

const stopContainer = async (containerId: string): Promise<void> => {
  try {
    const success = await dockerStore.stopContainer(containerId)
    if (success) {
      message.success('컨테이너가 중지되었습니다')
    }
  } catch (error) {
    message.error('컨테이너 중지에 실패했습니다')
  }
}

const restartContainer = async (containerId: string): Promise<void> => {
  try {
    const success = await dockerStore.restartContainer(containerId)
    if (success) {
      message.success('컨테이너가 재시작되었습니다')
    }
  } catch (error) {
    message.error('컨테이너 재시작에 실패했습니다')
  }
}

const removeContainer = async (containerId: string): Promise<void> => {
  try {
    const success = await dockerStore.removeContainerById(containerId)
    if (success) {
      message.success('컨테이너가 삭제되었습니다')
    }
  } catch (error) {
    message.error('컨테이너 삭제에 실패했습니다')
  }
}

const viewLogs = (containerId: string): void => {
  const container = dockerStore.containerById(containerId)
  if (container) {
    selectedContainerId.value = containerId
    selectedContainerName.value = container.name
    showLogsModal.value = true
  }
}

const openTerminal = (containerId: string): void => {
  // 컨테이너 터미널 열기
  router.push(`/terminal/${containerId}`)
}

// 모니터링 상태 동기화
watch(isMonitoring, (newValue) => {
  isMonitoringEnabled.value = newValue
})

watch(connectionStatus, (newStatus) => {
  if (newStatus === 'error') {
    showConnectionStatus.value = true
  }
})

// 생명주기
onMounted(async () => {
  await refreshContainers()
})

onUnmounted(() => {
  if (isMonitoring.value) {
    dockerStore.stopMonitoring()
  }
})
</script>

<style scoped>
.container-monitor {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.monitor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background-color: var(--n-color-hover);
  border-radius: 8px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.monitor-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--n-text-color);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.connection-status {
  margin-bottom: 16px;
}

.container-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.container-item {
  border: 1px solid var(--n-border-color);
  border-radius: 12px;
  padding: 20px;
  background-color: var(--n-card-color);
  transition: all 0.3s ease;
}

.container-item:hover {
  border-color: var(--n-color-primary);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.container-item.is-running {
  border-left: 4px solid var(--n-color-success);
}

.container-info {
  margin-bottom: 16px;
}

.container-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.container-name {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--n-text-color);
}

.container-status-tag {
  display: flex;
  align-items: center;
  gap: 4px;
}

.container-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.container-image,
.container-workspace {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--n-text-color-2);
}

.container-stats {
  margin: 16px 0;
  padding: 16px;
  background-color: var(--n-color-hover);
  border-radius: 8px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stat-label {
  font-size: 12px;
  color: var(--n-text-color-3);
  font-weight: 500;
}

.stat-value {
  font-size: 12px;
  color: var(--n-text-color-2);
  font-weight: 500;
}

.network-stats {
  display: flex;
  gap: 16px;
}

.network-stat {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--n-text-color-2);
}

.container-ports {
  margin: 16px 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.ports-label {
  font-size: 14px;
  color: var(--n-text-color-2);
  font-weight: 500;
}

.container-actions {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--n-border-color);
}

.container-empty {
  min-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.loading-container {
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 반응형 디자인 */
@media (max-width: 768px) {
  .monitor-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .header-actions {
    flex-direction: column;
    gap: 12px;
  }

  .container-header {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }

  .stats-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .network-stats {
    flex-direction: column;
    gap: 8px;
  }

  .container-ports {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
}
</style>