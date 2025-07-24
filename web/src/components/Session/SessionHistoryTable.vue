<template>
  <div class="session-history">
    <!-- 헤더 영역 -->
    <div class="history-header">
      <div class="title-section">
        <h3>보안 이벤트 히스토리</h3>
        <p class="subtitle">
          로그인, 로그아웃 및 보안 관련 활동 기록을 확인할 수 있습니다.
        </p>
      </div>
      
      <div class="filter-section">
        <n-space>
          <!-- 이벤트 타입 필터 -->
          <n-select
            v-model:value="filters.eventType"
            placeholder="이벤트 타입"
            :options="eventTypeOptions"
            clearable
            style="width: 150px"
            @update:value="handleFilterChange"
          />
          
          <!-- 심각도 필터 -->
          <n-select
            v-model:value="filters.severity"
            placeholder="심각도"
            :options="severityOptions"
            clearable
            style="width: 120px"
            @update:value="handleFilterChange"
          />
          
          <!-- 날짜 범위 필터 -->
          <n-date-picker
            v-model:value="dateRange"
            type="daterange"
            clearable
            :shortcuts="dateShortcuts"
            @update:value="handleDateRangeChange"
          />
          
          <!-- 새로고침 버튼 -->
          <n-button
            type="primary"
            ghost
            :loading="loading"
            @click="handleRefresh"
          >
            <template #icon>
              <n-icon><Refresh /></n-icon>
            </template>
            새로고침
          </n-button>
        </n-space>
      </div>
    </div>

    <!-- 데이터 테이블 -->
    <n-data-table
      :columns="columns"
      :data="events"
      :loading="loading"
      :pagination="paginationConfig"
      :bordered="false"
      :single-line="false"
      :scroll-x="1200"
      size="medium"
      flex-height
      style="min-height: 400px"
    />

    <!-- 이벤트 상세 모달 -->
    <n-modal
      v-model:show="showEventDetail"
      preset="card"
      title="보안 이벤트 상세정보"
      size="large"
      :bordered="false"
      :segmented="true"
    >
      <div v-if="selectedEvent" class="event-detail">
        <n-descriptions :column="2" bordered>
          <n-descriptions-item label="이벤트 ID">
            <n-text code>{{ selectedEvent.id }}</n-text>
          </n-descriptions-item>
          <n-descriptions-item label="세션 ID">
            <n-text code>{{ selectedEvent.sessionId }}</n-text>
          </n-descriptions-item>
          <n-descriptions-item label="이벤트 타입">
            <n-tag :type="getEventTypeTagType(selectedEvent.eventType)" size="small">
              {{ getEventTypeLabel(selectedEvent.eventType) }}
            </n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="심각도">
            <n-tag :type="getSeverityTagType(selectedEvent.severity)" size="small">
              {{ getSeverityLabel(selectedEvent.severity) }}
            </n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="설명" :span="2">
            {{ selectedEvent.description }}
          </n-descriptions-item>
          <n-descriptions-item label="IP 주소">
            {{ selectedEvent.ipAddress }}
          </n-descriptions-item>
          <n-descriptions-item label="발생 시간">
            {{ formatDateTime(selectedEvent.createdAt) }}
          </n-descriptions-item>
          <n-descriptions-item label="사용자 에이전트" :span="2">
            <n-text style="word-break: break-all; font-size: 12px;">
              {{ selectedEvent.userAgent }}
            </n-text>
          </n-descriptions-item>
          <n-descriptions-item v-if="selectedEvent.metadata && Object.keys(selectedEvent.metadata).length > 0" label="추가 정보" :span="2">
            <n-code :code="JSON.stringify(selectedEvent.metadata, null, 2)" language="json" />
          </n-descriptions-item>
        </n-descriptions>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted, watch } from 'vue'
import { NButton, NTag, NIcon, useMessage } from 'naive-ui'
import { 
  RefreshSharp as Refresh, 
  EyeSharp as Eye, 
  ShieldSharp as Shield, 
  WarningSharp as AlertTriangle, 
  InformationCircleSharp as Info, 
  CloseCircleSharp as XCircle,
  CheckmarkCircleSharp as CheckCircle,
  LogInSharp as Login,
  LogOutSharp as Logout,
  PersonCheckmarkSharp as UserCheck,
  KeySharp as Key,
  PhonePortraitSharp as Smartphone,
  LocationSharp as MapPin
} from '@vicons/ionicons5'
// date-fns 대신 내장 함수 사용
import type { SessionSecurityEvent } from '@/types/api'

// Props
interface Props {
  events: SessionSecurityEvent[]
  loading?: boolean
  pagination: {
    page: number
    limit: number
    total: number
    totalPages: number
  }
}

const props = withDefaults(defineProps<Props>(), {
  loading: false
})

// Emits
const emit = defineEmits<{
  refresh: []
  pageChange: [page: number]
}>()

// 컴포저블
const message = useMessage()

// 반응형 상태
const showEventDetail = ref(false)
const selectedEvent = ref<SessionSecurityEvent | null>(null)
const dateRange = ref<[number, number] | null>(null)

// 필터 상태
const filters = ref({
  eventType: null as string | null,
  severity: null as string | null
})

// 이벤트 타입 옵션
const eventTypeOptions = [
  { label: '로그인', value: 'login' },
  { label: '로그아웃', value: 'logout' },
  { label: '의심스러운 활동', value: 'suspicious_activity' },
  { label: '비밀번호 변경', value: 'password_change' },
  { label: '디바이스 변경', value: 'device_change' },
  { label: '위치 변경', value: 'location_change' }
]

// 심각도 옵션
const severityOptions = [
  { label: '낮음', value: 'low' },
  { label: '보통', value: 'medium' },
  { label: '높음', value: 'high' },
  { label: '위험', value: 'critical' }
]

// 날짜 단축키
const dateShortcuts = {
  '오늘': () => {
    const today = new Date()
    const startOfDay = new Date(today.getFullYear(), today.getMonth(), today.getDate())
    const endOfDay = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 23, 59, 59)
    return [startOfDay.getTime(), endOfDay.getTime()]
  },
  '어제': () => {
    const yesterday = new Date()
    yesterday.setDate(yesterday.getDate() - 1)
    const startOfDay = new Date(yesterday.getFullYear(), yesterday.getMonth(), yesterday.getDate())
    const endOfDay = new Date(yesterday.getFullYear(), yesterday.getMonth(), yesterday.getDate(), 23, 59, 59)
    return [startOfDay.getTime(), endOfDay.getTime()]
  },
  '최근 7일': () => {
    const today = new Date()
    const weekAgo = new Date()
    weekAgo.setDate(weekAgo.getDate() - 6)
    const startOfDay = new Date(weekAgo.getFullYear(), weekAgo.getMonth(), weekAgo.getDate())
    const endOfDay = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 23, 59, 59)
    return [startOfDay.getTime(), endOfDay.getTime()]
  },
  '최근 30일': () => {
    const today = new Date()
    const monthAgo = new Date()
    monthAgo.setDate(monthAgo.getDate() - 29)
    const startOfDay = new Date(monthAgo.getFullYear(), monthAgo.getMonth(), monthAgo.getDate())
    const endOfDay = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 23, 59, 59)
    return [startOfDay.getTime(), endOfDay.getTime()]
  }
}

// 테이블 컬럼 정의
const columns = [
  {
    title: '이벤트',
    key: 'eventType',
    width: 120,
    render: (row: SessionSecurityEvent) => {
      return h(NTag, {
        type: getEventTypeTagType(row.eventType),
        size: 'small'
      }, {
        icon: () => h(NIcon, { size: 14 }, { default: () => getEventTypeIcon(row.eventType) }),
        default: () => getEventTypeLabel(row.eventType)
      })
    }
  },
  {
    title: '심각도',
    key: 'severity',
    width: 100,
    render: (row: SessionSecurityEvent) => {
      return h(NTag, {
        type: getSeverityTagType(row.severity),
        size: 'small'
      }, {
        default: () => getSeverityLabel(row.severity)
      })
    }
  },
  {
    title: '설명',
    key: 'description',
    minWidth: 200,
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: 'IP 주소',
    key: 'ipAddress',
    width: 140
  },
  {
    title: '발생 시간',
    key: 'createdAt',
    width: 160,
    render: (row: SessionSecurityEvent) => {
      return new Date(row.createdAt).toLocaleString('ko-KR', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      })
    }
  },
  {
    title: '작업',
    key: 'actions',
    width: 100,
    render: (row: SessionSecurityEvent) => {
      return h(NButton, {
        size: 'small',
        type: 'primary',
        ghost: true,
        onClick: () => showEventDetails(row)
      }, {
        icon: () => h(NIcon, {}, { default: () => h(Eye) }),
        default: () => '상세'
      })
    }
  }
]

// 계산된 속성
const paginationConfig = computed(() => ({
  page: props.pagination.page,
  pageSize: props.pagination.limit,
  itemCount: props.pagination.total,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  showQuickJumper: true,
  prefix: ({ itemCount }: { itemCount: number }) => `총 ${itemCount}개`,
  onUpdatePage: (page: number) => emit('pageChange', page),
  onUpdatePageSize: (pageSize: number) => {
    // 페이지 크기 변경 시 첫 페이지로 이동
    emit('pageChange', 1)
  }
}))

// 메서드
const getEventTypeIcon = (eventType: string) => {
  switch (eventType) {
    case 'login': return h(Login)
    case 'logout': return h(Logout)
    case 'suspicious_activity': return h(AlertTriangle)
    case 'password_change': return h(Key)
    case 'device_change': return h(Smartphone)
    case 'location_change': return h(MapPin)
    default: return h(Info)
  }
}

const getEventTypeLabel = (eventType: string) => {
  const option = eventTypeOptions.find(opt => opt.value === eventType)
  return option?.label || eventType
}

const getEventTypeTagType = (eventType: string): 'success' | 'warning' | 'error' | 'info' | 'default' => {
  switch (eventType) {
    case 'login': return 'success'
    case 'logout': return 'info'
    case 'suspicious_activity': return 'error'
    case 'password_change': return 'warning'
    case 'device_change': return 'warning'
    case 'location_change': return 'warning'
    default: return 'default'
  }
}

const getSeverityLabel = (severity: string) => {
  const option = severityOptions.find(opt => opt.value === severity)
  return option?.label || severity
}

const getSeverityTagType = (severity: string): 'success' | 'warning' | 'error' | 'info' | 'default' => {
  switch (severity) {
    case 'low': return 'success'
    case 'medium': return 'info'
    case 'high': return 'warning'
    case 'critical': return 'error'
    default: return 'default'
  }
}

const formatDateTime = (dateString: string) => {
  return new Date(dateString).toLocaleString('ko-KR', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const showEventDetails = (event: SessionSecurityEvent) => {
  selectedEvent.value = event
  showEventDetail.value = true
}

const handleRefresh = () => {
  emit('refresh')
}

const handleFilterChange = () => {
  // 필터 변경 시 첫 페이지로 이동하면서 새로고침
  emit('pageChange', 1)
}

const handleDateRangeChange = (value: [number, number] | null) => {
  dateRange.value = value
  handleFilterChange()
}

// 와처
watch([
  () => filters.value.eventType,
  () => filters.value.severity,
  () => dateRange.value
], () => {
  // 실제 구현에서는 부모 컴포넌트에서 필터 파라미터를 받아 API 호출
  console.log('필터 변경:', {
    eventType: filters.value.eventType,
    severity: filters.value.severity,
    dateRange: dateRange.value
  })
}, { deep: true })

// 라이프사이클
onMounted(() => {
  // 컴포넌트 마운트 시 초기 데이터 로드는 부모에서 처리
})
</script>

<style scoped lang="scss">
.session-history {
  .history-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 24px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border-color);

    .title-section {
      h3 {
        margin: 0 0 8px 0;
        font-size: 20px;
        font-weight: 500;
        color: var(--text-color-1);
      }

      .subtitle {
        margin: 0;
        color: var(--text-color-2);
        font-size: 14px;
        line-height: 1.4;
      }
    }

    .filter-section {
      flex-shrink: 0;
    }
  }

  .event-detail {
    .n-descriptions {
      --n-th-color: var(--card-color);
    }
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .session-history {
    .history-header {
      flex-direction: column;
      gap: 16px;
      align-items: stretch;

      .filter-section {
        .n-space {
          flex-wrap: wrap;
          
          :deep(.n-space-item) {
            flex: 1;
            min-width: 120px;
          }
        }
      }
    }

    :deep(.n-data-table) {
      .n-data-table-wrapper {
        overflow-x: auto;
      }
    }
  }
}

@media (max-width: 480px) {
  .session-history {
    .history-header {
      .filter-section {
        .n-space {
          :deep(.n-space-item) {
            width: 100%;
            
            .n-select,
            .n-date-picker,
            .n-button {
              width: 100%;
            }
          }
        }
      }
    }
  }
}
</style>