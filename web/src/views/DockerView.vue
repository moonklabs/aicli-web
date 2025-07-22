<template>
  <div class="docker-view">
    <header class="docker-header">
      <h1 class="docker-title">Docker 관리</h1>
      <p class="docker-description">
        컨테이너, 이미지, 네트워크를 관리할 수 있습니다.
      </p>
    </header>

    <div class="docker-content">
      <!-- Docker 통계 -->
      <div class="docker-stats">
        <NCard class="stat-card">
          <NStatistic
            label="전체 컨테이너"
            :value="totalContainers"
          />
        </NCard>
        <NCard class="stat-card">
          <NStatistic
            label="실행 중인 컨테이너"
            :value="runningContainers.length"
          />
        </NCard>
        <NCard class="stat-card">
          <NStatistic
            label="중지된 컨테이너"
            :value="stoppedContainers.length"
          />
        </NCard>
        <NCard class="stat-card">
          <NStatistic
            label="이미지"
            :value="totalImages"
          />
        </NCard>
      </div>

      <!-- 탭 네비게이션 -->
      <NTabs v-model:value="activeTab" type="line" animated>
        <NTabPane name="containers" tab="컨테이너">
          <div class="containers-section">
            <div class="section-header">
              <NSpace>
                <NButton @click="refreshContainers" :loading="isLoading">
                  <template #icon>
                    <NIcon><RefreshIcon /></NIcon>
                  </template>
                  새로고침
                </NButton>
                <NButton type="error" @click="cleanupContainers">
                  정리
                </NButton>
              </NSpace>
            </div>

            <NDataTable
              :columns="containerColumns"
              :data="containers"
              :loading="isLoading"
              :scroll-x="1200"
              size="small"
            />
          </div>
        </NTabPane>

        <NTabPane name="images" tab="이미지">
          <div class="images-section">
            <div class="section-header">
              <NSpace>
                <NButton @click="refreshImages" :loading="isLoading">
                  <template #icon>
                    <NIcon><RefreshIcon /></NIcon>
                  </template>
                  새로고침
                </NButton>
                <NButton type="error" @click="cleanupImages">
                  사용하지 않는 이미지 정리
                </NButton>
              </NSpace>
            </div>

            <NDataTable
              :columns="imageColumns"
              :data="images"
              :loading="isLoading"
              :scroll-x="1000"
              size="small"
            />
          </div>
        </NTabPane>

        <NTabPane name="networks" tab="네트워크">
          <div class="networks-section">
            <div class="section-header">
              <NSpace>
                <NButton @click="refreshNetworks" :loading="isLoading">
                  <template #icon>
                    <NIcon><RefreshIcon /></NIcon>
                  </template>
                  새로고침
                </NButton>
              </NSpace>
            </div>

            <NDataTable
              :columns="networkColumns"
              :data="networks"
              :loading="isLoading"
              size="small"
            />
          </div>
        </NTabPane>
      </NTabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import {
  type DataTableColumns,
  NButton,
  NCard,
  NDataTable,
  NIcon,
  NSpace,
  NStatistic,
  NTabPane,
  NTabs,
  NTag,
} from 'naive-ui'
import { useDockerStore } from '@/stores/docker'
import { formatBytes, formatDate } from '@/utils/format'

// 아이콘
const RefreshIcon = {
  render: () => '↻',
}

const message = useMessage()
const dockerStore = useDockerStore()

// 상태
const activeTab = ref('containers')

// 계산된 속성
const containers = computed(() => dockerStore.containers)
const runningContainers = computed(() => dockerStore.runningContainers)
const stoppedContainers = computed(() => dockerStore.stoppedContainers)
const totalContainers = computed(() => dockerStore.totalContainers)
const images = computed(() => dockerStore.images)
const totalImages = computed(() => dockerStore.totalImages)
const networks = computed(() => dockerStore.networks)
const isLoading = computed(() => dockerStore.isLoading)

// 테이블 컬럼 정의
const containerColumns: DataTableColumns<any> = [
  {
    title: '이름',
    key: 'name',
    width: 200,
    ellipsis: true,
  },
  {
    title: '상태',
    key: 'status',
    width: 120,
    render: (row: any) => {
      const statusType = row.status === 'running' ? 'success' : 'default'
      const statusText = row.status === 'running' ? '실행 중' : '중지됨'
      return h(NTag, { type: statusType }, () => statusText)
    },
  },
  {
    title: '이미지',
    key: 'image',
    width: 200,
    ellipsis: true,
  },
  {
    title: '포트',
    key: 'ports',
    width: 150,
    render: (row: any) => row.ports?.join(', ') || '-',
  },
  {
    title: '생성일',
    key: 'created',
    width: 180,
    render: (row: any) => formatDate(row.created),
  },
  {
    title: '액션',
    key: 'actions',
    width: 200,
    render: (row: any) => {
      const actions: any[] = []

      if (row.status === 'running') {
        actions.push(
          h(
            NButton,
            {
              size: 'small',
              type: 'warning',
              onClick: () => stopContainer(row.id),
            },
            () => '중지',
          ),
        )
        actions.push(
          h(
            NButton,
            {
              size: 'small',
              type: 'info',
              onClick: () => restartContainer(row.id),
            },
            () => '재시작',
          ),
        )
      } else {
        actions.push(
          h(
            NButton,
            {
              size: 'small',
              type: 'primary',
              onClick: () => startContainer(row.id),
            },
            () => '시작',
          ),
        )
      }

      actions.push(
        h(
          NButton,
          {
            size: 'small',
            type: 'error',
            onClick: () => removeContainer(row.id),
          },
          () => '삭제',
        ),
      )

      return h(NSpace, { size: 'small' }, () => actions)
    },
  },
]

const imageColumns: DataTableColumns<any> = [
  {
    title: '저장소',
    key: 'repository',
    width: 200,
    ellipsis: true,
  },
  {
    title: '태그',
    key: 'tag',
    width: 120,
  },
  {
    title: '이미지 ID',
    key: 'id',
    width: 150,
    ellipsis: true,
    render: (row: any) => row.id.substring(0, 12),
  },
  {
    title: '크기',
    key: 'size',
    width: 120,
    render: (row: any) => formatBytes(row.size),
  },
  {
    title: '생성일',
    key: 'created',
    width: 180,
    render: (row: any) => formatDate(row.created),
  },
  {
    title: '액션',
    key: 'actions',
    width: 120,
    render: (row: any) => {
      return h(
        NButton,
        {
          size: 'small',
          type: 'error',
          onClick: () => removeImage(row.id),
        },
        () => '삭제',
      )
    },
  },
]

const networkColumns: DataTableColumns<any> = [
  {
    title: '이름',
    key: 'name',
    width: 200,
  },
  {
    title: '드라이버',
    key: 'driver',
    width: 120,
  },
  {
    title: '범위',
    key: 'scope',
    width: 120,
  },
  {
    title: '연결된 컨테이너',
    key: 'containers',
    width: 120,
    render: (row: any) => Object.keys(row.containers || {}).length,
  },
  {
    title: '생성일',
    key: 'created',
    width: 180,
    render: (row: any) => formatDate(row.created),
  },
]

// 메서드
const refreshContainers = async () => {
  try {
    await dockerStore.refreshContainers()
    message.success('컨테이너 목록이 새로고침되었습니다')
  } catch (_error) {
    message.error('컨테이너 목록 새로고침에 실패했습니다')
  }
}

const refreshImages = async () => {
  try {
    await dockerStore.refreshImages()
    message.success('이미지 목록이 새로고침되었습니다')
  } catch (_error) {
    message.error('이미지 목록 새로고침에 실패했습니다')
  }
}

const refreshNetworks = async () => {
  try {
    await dockerStore.refreshNetworks()
    message.success('네트워크 목록이 새로고침되었습니다')
  } catch (_error) {
    message.error('네트워크 목록 새로고침에 실패했습니다')
  }
}

const startContainer = async (id: string) => {
  try {
    await dockerStore.startContainer(id)
    message.success('컨테이너가 시작되었습니다')
  } catch (_error) {
    message.error('컨테이너 시작에 실패했습니다')
  }
}

const stopContainer = async (id: string) => {
  try {
    await dockerStore.stopContainer(id)
    message.success('컨테이너가 중지되었습니다')
  } catch (_error) {
    message.error('컨테이너 중지에 실패했습니다')
  }
}

const restartContainer = async (id: string) => {
  try {
    await dockerStore.restartContainer(id)
    message.success('컨테이너가 재시작되었습니다')
  } catch (_error) {
    message.error('컨테이너 재시작에 실패했습니다')
  }
}

const removeContainer = async (id: string) => {
  try {
    await dockerStore.removeContainer(id)
    message.success('컨테이너가 삭제되었습니다')
  } catch (_error) {
    message.error('컨테이너 삭제에 실패했습니다')
  }
}

const removeImage = async (id: string) => {
  try {
    await dockerStore.removeImage(id)
    message.success('이미지가 삭제되었습니다')
  } catch (_error) {
    message.error('이미지 삭제에 실패했습니다')
  }
}

const cleanupContainers = async () => {
  try {
    await dockerStore.cleanupContainers()
    message.success('중지된 컨테이너가 정리되었습니다')
  } catch (_error) {
    message.error('컨테이너 정리에 실패했습니다')
  }
}

const cleanupImages = async () => {
  try {
    await dockerStore.cleanupImages()
    message.success('사용하지 않는 이미지가 정리되었습니다')
  } catch (_error) {
    message.error('이미지 정리에 실패했습니다')
  }
}

// 라이프사이클
onMounted(() => {
  refreshContainers()
  refreshImages()
  refreshNetworks()
})
</script>

<style lang="scss" scoped>
.docker-view {
  padding: 1.5rem;
  max-width: 1400px;
  margin: 0 auto;
}

.docker-header {
  margin-bottom: 2rem;
  text-align: center;

  .docker-title {
    font-size: 2rem;
    font-weight: 600;
    margin-bottom: 0.5rem;
    color: var(--text-color-1);
  }

  .docker-description {
    color: var(--text-color-2);
    font-size: 1.1rem;
  }
}

.docker-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;

  .stat-card {
    text-align: center;
  }
}

.section-header {
  margin-bottom: 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>