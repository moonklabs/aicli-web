<template>
  <div class="login-history-table">
    <!-- 필터 및 검색 -->
    <div class="table-header">
      <div class="search-section">
        <n-input
          v-model:value="searchKeyword"
          placeholder="IP 주소 또는 위치로 검색..."
          clearable
          @update:value="handleSearch"
        >
          <template #prefix>
            <n-icon><Search /></n-icon>
          </template>
        </n-input>
      </div>
      
      <div class="filter-section">
        <n-space>
          <n-select
            v-model:value="statusFilter"
            placeholder="상태 필터"
            :options="statusOptions"
            clearable
            style="width: 140px"
            @update:value="handleFilterChange"
          />
          
          <n-select
            v-model:value="methodFilter"
            placeholder="로그인 방법"
            :options="methodOptions"
            clearable
            style="width: 140px"
            @update:value="handleFilterChange"
          />
          
          <n-date-picker
            v-model:value="dateRange"
            type="daterange"
            placeholder="날짜 범위"
            clearable
            @update:value="handleFilterChange"
          />
          
          <n-button
            type="primary"
            ghost
            :loading="loading"
            @click="handleExport"
          >
            <template #icon>
              <n-icon><Download /></n-icon>
            </template>
            내보내기
          </n-button>
        </n-space>
      </div>
    </div>

    <!-- 데이터 테이블 -->
    <n-data-table
      :columns="columns"
      :data="filteredData"
      :loading="loading"
      :pagination="paginationConfig"
      :row-class-name="getRowClassName"
      :scroll-x="1200"
      @update:page="handlePageChange"
    />

    <!-- 로그인 상세 모달 -->
    <n-modal
      v-model:show="showDetailModal"
      preset="card"
      title="로그인 상세 정보"
      style="width: 600px"
    >
      <LoginDetailModal
        v-if="selectedLogin"
        :login="selectedLogin"
      />
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { 
  NDataTable, 
  NInput, 
  NSelect, 
  NDatePicker,
  NButton, 
  NIcon, 
  NSpace,
  NModal,
  NTag,
  NTooltip,
  useMessage,
  type DataTableColumns
} from 'naive-ui'
import { Search, Download, MapPin, AlertTriangle, Eye } from '@vicons/tabler'
import { authApi } from '@/api/services/auth'
import { formatDistanceToNow, format } from 'date-fns'
import { ko } from 'date-fns/locale'
import LoginDetailModal from './LoginDetailModal.vue'
import type { LoginHistory, SecurityEventFilter, LogExportRequest } from '@/types/api'

interface Props {
  data?: LoginHistory[]
  loading?: boolean
  limit?: number
  showPagination?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  data: () => [],
  loading: false,
  limit: 50,
  showPagination: true
})

const emit = defineEmits<{
  refresh: []
}>()

const message = useMessage()

// 상태 관리
const searchKeyword = ref('')
const statusFilter = ref<string | null>(null)
const methodFilter = ref<string | null>(null)
const dateRange = ref<[number, number] | null>(null)
const currentPage = ref(1)
const pageSize = ref(20)
const selectedLogin = ref<LoginHistory | null>(null)
const showDetailModal = ref(false)

// 필터 옵션
const statusOptions = [
  { label: '성공', value: 'success' },
  { label: '실패', value: 'failure' },
  { label: '차단됨', value: 'blocked' }
]

const methodOptions = [
  { label: '비밀번호', value: 'password' },
  { label: 'OAuth', value: 'oauth' },
  { label: 'SSO', value: 'sso' },
  { label: '토큰', value: 'token' }
]

// 계산된 속성
const filteredData = computed(() => {
  let result = props.data || []

  // 검색 필터
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    result = result.filter(item => 
      item.ipAddress.toLowerCase().includes(keyword) ||
      item.location?.country?.toLowerCase().includes(keyword) ||
      item.location?.city?.toLowerCase().includes(keyword) ||
      item.deviceInfo.browser.toLowerCase().includes(keyword) ||
      item.deviceInfo.os.toLowerCase().includes(keyword)
    )
  }

  // 상태 필터
  if (statusFilter.value) {
    result = result.filter(item => item.status === statusFilter.value)
  }

  // 로그인 방법 필터
  if (methodFilter.value) {
    result = result.filter(item => item.loginMethod === methodFilter.value)
  }

  // 날짜 범위 필터
  if (dateRange.value) {
    const [startTime, endTime] = dateRange.value
    result = result.filter(item => {
      const itemTime = new Date(item.createdAt).getTime()
      return itemTime >= startTime && itemTime <= endTime
    })
  }

  return result
})

const paginationConfig = computed(() => {
  if (!props.showPagination) return false
  
  return {
    page: currentPage.value,
    pageSize: pageSize.value,
    itemCount: filteredData.value.length,
    showSizePicker: true,
    pageSizes: [10, 20, 50, 100],
    onUpdatePage: (page: number) => {
      currentPage.value = page
    },
    onUpdatePageSize: (size: number) => {
      pageSize.value = size
      currentPage.value = 1
    }
  }
})

// 테이블 컬럼 정의
const columns: DataTableColumns<LoginHistory> = [
  {
    title: '시간',
    key: 'createdAt',
    width: 160,
    render: (row) => {
      const date = new Date(row.createdAt)
      return h('div', { class: 'timestamp-cell' }, [
        h('div', { class: 'date' }, format(date, 'MM/dd HH:mm', { locale: ko })),
        h('div', { class: 'relative-time' }, formatDistanceToNow(date, { addSuffix: true, locale: ko }))
      ])
    }
  },
  {
    title: '상태',
    key: 'status',
    width: 100,
    render: (row) => {
      const statusConfig = {
        success: { type: 'success', text: '성공' },
        failure: { type: 'error', text: '실패' },
        blocked: { type: 'warning', text: '차단됨' }
      }
      const config = statusConfig[row.status] || { type: 'default', text: row.status }
      
      return h(NTag, { type: config.type, size: 'small' }, { default: () => config.text })
    }
  },
  {
    title: '위험도',
    key: 'riskScore',
    width: 100,
    render: (row) => {
      const getRiskConfig = (score: number) => {
        if (score >= 80) return { type: 'error', text: '높음', color: '#e74c3c' }
        if (score >= 60) return { type: 'warning', text: '보통', color: '#f39c12' }
        if (score >= 40) return { type: 'info', text: '낮음', color: '#3498db' }
        return { type: 'success', text: '안전', color: '#27ae60' }
      }
      
      const config = getRiskConfig(row.riskScore)
      return h('div', { class: 'risk-score-cell' }, [
        h(NTag, { 
          type: config.type, 
          size: 'small',
          style: { backgroundColor: config.color + '20', borderColor: config.color }
        }, { default: () => `${config.text} (${row.riskScore})` }),
        row.isSuspicious && h(NIcon, { 
          component: AlertTriangle, 
          style: { color: '#e74c3c', marginLeft: '4px' } 
        })
      ])
    }
  },
  {
    title: 'IP 주소',
    key: 'ipAddress',
    width: 140,
    render: (row) => {
      return h('div', { class: 'ip-address-cell' }, [
        h('code', { class: 'ip-address' }, row.ipAddress)
      ])
    }
  },
  {
    title: '위치',
    key: 'location',
    width: 150,
    render: (row) => {
      if (!row.location) return h('span', { class: 'no-data' }, '-')
      
      const location = [row.location.city, row.location.country].filter(Boolean).join(', ')
      return h('div', { class: 'location-cell' }, [
        h(NIcon, { component: MapPin, size: 14, style: { marginRight: '4px' } }),
        h('span', location || '알 수 없음')
      ])
    }
  },
  {
    title: '디바이스',
    key: 'deviceInfo',
    width: 200,
    render: (row) => {
      return h('div', { class: 'device-cell' }, [
        h('div', { class: 'browser' }, `${row.deviceInfo.browser} / ${row.deviceInfo.os}`),
        h('div', { class: 'device-type' }, row.deviceInfo.device)
      ])
    }
  },
  {
    title: '로그인 방법',
    key: 'loginMethod',
    width: 120,
    render: (row) => {
      const methodConfig = {
        password: { type: 'default', text: '비밀번호' },
        oauth: { type: 'info', text: 'OAuth' },
        sso: { type: 'warning', text: 'SSO' },
        token: { type: 'success', text: '토큰' }
      }
      const config = methodConfig[row.loginMethod] || { type: 'default', text: row.loginMethod }
      
      return h(NTag, { type: config.type, size: 'small' }, { 
        default: () => row.provider ? `${config.text} (${row.provider})` : config.text
      })
    }
  },
  {
    title: '작업',
    key: 'actions',
    width: 80,
    fixed: 'right',
    render: (row) => {
      return h(NButton, {
        size: 'small',
        type: 'primary',
        ghost: true,
        onClick: () => handleViewDetails(row)
      }, {
        default: () => '상세',
        icon: () => h(NIcon, { component: Eye })
      })
    }
  }
]

// 메소드
const getRowClassName = (row: LoginHistory) => {
  const classes = ['login-history-row']
  if (row.isSuspicious) classes.push('suspicious-row')
  if (row.status === 'failure') classes.push('failed-row')
  if (row.status === 'blocked') classes.push('blocked-row')
  return classes.join(' ')
}

const handleSearch = () => {
  currentPage.value = 1
}

const handleFilterChange = () => {
  currentPage.value = 1
}

const handlePageChange = (page: number) => {
  currentPage.value = page
}

const handleViewDetails = (login: LoginHistory) => {
  selectedLogin.value = login
  showDetailModal.value = true
}

const handleExport = async () => {
  try {
    const filters: SecurityEventFilter = {
      limit: 10000 // 최대 개수
    }
    
    if (statusFilter.value) {
      filters.status = [statusFilter.value]
    }
    
    if (dateRange.value) {
      filters.startDate = new Date(dateRange.value[0]).toISOString()
      filters.endDate = new Date(dateRange.value[1]).toISOString()
    }

    const exportRequest: LogExportRequest = {
      type: 'login_history',
      format: 'csv',
      filters,
      includeMetadata: true
    }

    const response = await authApi.exportSecurityLogs(exportRequest)
    
    // 다운로드 링크 생성
    const link = document.createElement('a')
    link.href = response.downloadUrl
    link.download = response.filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    
    message.success(`로그인 이력 ${response.recordCount}건이 내보내기되었습니다`)
  } catch (error) {
    message.error('내보내기에 실패했습니다')
    console.error('Export error:', error)
  }
}

// 생명주기
onMounted(() => {
  if (!props.data?.length) {
    emit('refresh')
  }
})

// 검색어 변경 감지
watch(searchKeyword, () => {
  handleSearch()
}, { debounce: 300 })
</script>

<style scoped>
.login-history-table {
  width: 100%;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 16px;
}

.search-section {
  min-width: 280px;
}

.filter-section {
  flex: 1;
  display: flex;
  justify-content: flex-end;
}

.timestamp-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.date {
  font-weight: 500;
  font-size: 13px;
}

.relative-time {
  font-size: 11px;
  color: var(--text-color-3);
}

.risk-score-cell {
  display: flex;
  align-items: center;
  gap: 4px;
}

.ip-address-cell .ip-address {
  background: var(--code-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  font-family: 'Fira Code', monospace;
}

.location-cell {
  display: flex;
  align-items: center;
  font-size: 13px;
}

.device-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.browser {
  font-size: 13px;
  font-weight: 500;
}

.device-type {
  font-size: 11px;
  color: var(--text-color-3);
}

.no-data {
  color: var(--text-color-3);
  font-style: italic;
}

/* 행 스타일 */
:deep(.suspicious-row) {
  background-color: rgba(231, 76, 60, 0.05);
}

:deep(.failed-row) {
  background-color: rgba(231, 76, 60, 0.03);
}

:deep(.blocked-row) {
  background-color: rgba(243, 156, 18, 0.05);
}

:deep(.suspicious-row:hover),
:deep(.failed-row:hover),
:deep(.blocked-row:hover) {
  background-color: rgba(231, 76, 60, 0.08);
}

@media (max-width: 768px) {
  .table-header {
    flex-direction: column;
    align-items: stretch;
  }
  
  .filter-section {
    justify-content: flex-start;
  }
  
  .filter-section > * {
    flex-wrap: wrap;
  }
}
</style>