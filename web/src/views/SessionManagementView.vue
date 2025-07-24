<template>
  <div class="session-management">
    <!-- 헤더 섹션 -->
    <div class="header">
      <div class="title-section">
        <h1>세션 관리</h1>
        <p class="subtitle">활성 세션을 모니터링하고 보안 설정을 관리합니다</p>
      </div>
      
      <div class="stats-section">
        <n-space>
          <n-statistic
            v-if="sessionStats"
            label="활성 세션"
            :value="sessionStats.totalActiveSessions"
            :show-indicator="true"
          />
          <n-statistic
            v-if="sessionStats"
            label="연결된 디바이스"
            :value="sessionStats.currentDevices"
          />
          <n-statistic
            v-if="sessionStats && sessionStats.suspiciousActivities > 0"
            label="의심스러운 활동"
            :value="sessionStats.suspiciousActivities"
            :value-style="{ color: '#e74c3c' }"
          />
        </n-space>
      </div>
    </div>

    <!-- 메인 콘텐츠 -->
    <div class="content">
      <n-tabs v-model:value="activeTab" type="line" animated>
        <!-- 활성 세션 탭 -->
        <n-tab-pane name="sessions" tab="활성 세션">
          <div class="sessions-section">
            <div class="section-header">
              <h3>활성 세션 목록</h3>
              <n-space>
                <n-button
                  type="error"
                  ghost
                  :disabled="!hasOtherSessions"
                  :loading="terminatingAllSessions"
                  @click="handleTerminateAllSessions"
                >
                  <template #icon>
                    <n-icon><LogOut /></n-icon>
                  </template>
                  다른 모든 세션 종료
                </n-button>
                <n-button
                  type="primary"
                  ghost
                  :loading="refreshingSessions"
                  @click="refreshSessionData"
                >
                  <template #icon>
                    <n-icon><Refresh /></n-icon>
                  </template>
                  새로고침
                </n-button>
              </n-space>
            </div>

            <!-- 세션 목록 -->
            <div class="sessions-grid">
              <ActiveSessionCard
                v-for="session in activeSessions"
                :key="session.id"
                :session="session"
                @terminate="handleTerminateSession"
                @report-suspicious="handleReportSuspicious"
              />
            </div>

            <!-- 세션이 없을 때 -->
            <n-empty 
              v-if="!loadingSessions && activeSessions.length === 0"
              description="활성 세션이 없습니다"
            />

            <!-- 로딩 상태 -->
            <div v-if="loadingSessions" class="loading-state">
              <n-spin size="large" />
            </div>
          </div>
        </n-tab-pane>

        <!-- 보안 설정 탭 -->
        <n-tab-pane name="security" tab="보안 설정">
          <SessionSecuritySettings
            :settings="securitySettings"
            :loading="loadingSettings"
            @update="handleUpdateSettings"
          />
        </n-tab-pane>

        <!-- 보안 히스토리 탭 -->
        <n-tab-pane name="history" tab="보안 히스토리">
          <SessionHistoryTable
            :events="securityEvents"
            :loading="loadingEvents"
            :pagination="eventsPagination"
            @refresh="refreshSecurityEvents"
            @page-change="handlePageChange"
          />
        </n-tab-pane>
      </n-tabs>
    </div>

    <!-- 세션 종료 확인 모달 -->
    <n-modal
      v-model:show="showTerminateModal"
      preset="dialog"
      title="세션 종료 확인"
      content="이 세션을 종료하시겠습니까? 해당 디바이스에서 자동으로 로그아웃됩니다."
      positive-text="종료"
      negative-text="취소"
      @positive-click="confirmTerminateSession"
    />

    <!-- 전체 세션 종료 확인 모달 -->
    <n-modal
      v-model:show="showTerminateAllModal"
      preset="dialog"
      title="모든 세션 종료 확인"
      content="현재 세션을 제외한 모든 세션을 종료하시겠습니까? 다른 디바이스에서 자동으로 로그아웃됩니다."
      positive-text="모두 종료"
      negative-text="취소"
      type="warning"
      @positive-click="confirmTerminateAllSessions"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { LogOutSharp as LogOut, RefreshSharp as Refresh } from '@vicons/ionicons5'
import { sessionApi } from '@/api/services'
import { useWebSocket } from '@/composables/useWebSocket'
import type { 
  UserSession, 
  SessionSecurityEvent, 
  SessionSecuritySettings as SessionSettings,
  SessionStatsResponse,
  SessionUpdateMessage
} from '@/types/api'
import ActiveSessionCard from '@/components/Session/ActiveSessionCard.vue'
import SessionSecuritySettings from '@/components/Session/SessionSecuritySettings.vue'
import SessionHistoryTable from '@/components/Session/SessionHistoryTable.vue'

// 컴포저블
const message = useMessage()

// 반응형 상태
const activeTab = ref('sessions')
const loadingSessions = ref(false)
const loadingSettings = ref(false)
const loadingEvents = ref(false)
const refreshingSessions = ref(false)
const terminatingAllSessions = ref(false)

// 데이터 상태
const activeSessions = ref<UserSession[]>([])
const sessionStats = ref<SessionStatsResponse | null>(null)
const securitySettings = ref<SessionSettings | null>(null)
const securityEvents = ref<SessionSecurityEvent[]>([])
const eventsPagination = ref({
  page: 1,
  limit: 20,
  total: 0,
  totalPages: 0
})

// 모달 상태
const showTerminateModal = ref(false)
const showTerminateAllModal = ref(false)
const selectedSessionId = ref<string>('')

// 계산된 속성
const hasOtherSessions = computed(() => {
  return activeSessions.value.some(session => !session.isCurrentSession)
})

// WebSocket 연결
const { connect, disconnect } = useWebSocket()

// 메서드
const loadSessionData = async () => {
  loadingSessions.value = true
  try {
    const [sessions, stats] = await Promise.all([
      sessionApi.getActiveSessions(),
      sessionApi.getSessionStats()
    ])
    
    activeSessions.value = sessions
    sessionStats.value = stats
  } catch (error) {
    console.error('세션 데이터 로드 실패:', error)
    message.error('세션 데이터를 불러오는데 실패했습니다')
  } finally {
    loadingSessions.value = false
  }
}

const loadSecuritySettings = async () => {
  loadingSettings.value = true
  try {
    securitySettings.value = await sessionApi.getSecuritySettings()
  } catch (error) {
    console.error('보안 설정 로드 실패:', error)
    message.error('보안 설정을 불러오는데 실패했습니다')
  } finally {
    loadingSettings.value = false
  }
}

const loadSecurityEvents = async (page = 1) => {
  loadingEvents.value = true
  try {
    const response = await sessionApi.getSecurityEvents({
      page,
      limit: eventsPagination.value.limit
    })
    
    securityEvents.value = response.items
    eventsPagination.value = {
      page: response.page,
      limit: response.limit,
      total: response.total,
      totalPages: response.totalPages
    }
  } catch (error) {
    console.error('보안 이벤트 로드 실패:', error)
    message.error('보안 이벤트를 불러오는데 실패했습니다')
  } finally {
    loadingEvents.value = false
  }
}

const refreshSessionData = async () => {
  refreshingSessions.value = true
  try {
    await loadSessionData()
    message.success('세션 데이터를 새로고침했습니다')
  } finally {
    refreshingSessions.value = false
  }
}

const refreshSecurityEvents = () => {
  loadSecurityEvents(eventsPagination.value.page)
}

const handlePageChange = (page: number) => {
  loadSecurityEvents(page)
}

const handleTerminateSession = (sessionId: string) => {
  selectedSessionId.value = sessionId
  showTerminateModal.value = true
}

const confirmTerminateSession = async () => {
  try {
    await sessionApi.terminateSession({
      sessionId: selectedSessionId.value,
      reason: '사용자 요청'
    })
    
    message.success('세션을 종료했습니다')
    await loadSessionData()
  } catch (error) {
    console.error('세션 종료 실패:', error)
    message.error('세션 종료에 실패했습니다')
  }
}

const handleTerminateAllSessions = () => {
  showTerminateAllModal.value = true
}

const confirmTerminateAllSessions = async () => {
  terminatingAllSessions.value = true
  try {
    await sessionApi.terminateAllSessions({
      excludeCurrentSession: true,
      reason: '사용자 요청 - 모든 세션 종료'
    })
    
    message.success('다른 모든 세션을 종료했습니다')
    await loadSessionData()
  } catch (error) {
    console.error('전체 세션 종료 실패:', error)
    message.error('세션 종료에 실패했습니다')
  } finally {
    terminatingAllSessions.value = false
  }
}

const handleUpdateSettings = async (newSettings: Partial<SessionSettings>) => {
  try {
    securitySettings.value = await sessionApi.updateSecuritySettings(newSettings)
    message.success('보안 설정을 업데이트했습니다')
  } catch (error) {
    console.error('보안 설정 업데이트 실패:', error)
    message.error('보안 설정 업데이트에 실패했습니다')
  }
}

const handleReportSuspicious = async (sessionId: string, reason: string) => {
  try {
    await sessionApi.reportSuspiciousActivity(sessionId, reason)
    message.success('의심스러운 활동을 신고했습니다')
    await refreshSecurityEvents()
  } catch (error) {
    console.error('의심스러운 활동 신고 실패:', error)
    message.error('신고에 실패했습니다')
  }
}

// WebSocket 메시지 핸들러
const handleWebSocketMessage = (message: any) => {
  if (message.type === 'session_update') {
    const sessionMessage = message as SessionUpdateMessage
    const { type } = sessionMessage.payload
    
    switch (type) {
      case 'session_created':
      case 'session_terminated':
      case 'session_activity':
        // 세션 목록 새로고침
        loadSessionData()
        break
      case 'security_event':
        // 보안 이벤트 새로고침
        if (activeTab.value === 'history') {
          refreshSecurityEvents()
        }
        break
    }
  }
}

// 라이프사이클
onMounted(async () => {
  await Promise.all([
    loadSessionData(),
    loadSecuritySettings(),
    loadSecurityEvents()
  ])
  
  // WebSocket 연결
  connect({
    onMessage: handleWebSocketMessage,
    onError: (error: any) => {
      console.error('WebSocket 에러:', error)
    }
  })
})

// 탭 변경 시 데이터 로드
watch(activeTab, (newTab) => {
  if (newTab === 'history' && securityEvents.value.length === 0) {
    loadSecurityEvents()
  }
})

// 컴포넌트 언마운트 시 WebSocket 연결 해제
onUnmounted(() => {
  disconnect()
})
</script>

<style scoped lang="scss">
.session-management {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;

  .header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 32px;
    padding-bottom: 24px;
    border-bottom: 1px solid var(--border-color);

    .title-section {
      h1 {
        margin: 0 0 8px 0;
        font-size: 28px;
        font-weight: 600;
        color: var(--text-color-1);
      }

      .subtitle {
        margin: 0;
        color: var(--text-color-2);
        font-size: 16px;
      }
    }

    .stats-section {
      min-width: 400px;
    }
  }

  .content {
    .sessions-section {
      .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 24px;

        h3 {
          margin: 0;
          font-size: 20px;
          font-weight: 500;
          color: var(--text-color-1);
        }
      }

      .sessions-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
        gap: 16px;
        margin-bottom: 24px;
      }

      .loading-state {
        display: flex;
        justify-content: center;
        align-items: center;
        min-height: 200px;
      }
    }
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .session-management {
    padding: 16px;

    .header {
      flex-direction: column;
      gap: 16px;

      .stats-section {
        min-width: auto;
        width: 100%;
      }
    }

    .content {
      .sessions-section {
        .section-header {
          flex-direction: column;
          align-items: flex-start;
          gap: 12px;

          h3 {
            font-size: 18px;
          }
        }

        .sessions-grid {
          grid-template-columns: 1fr;
        }
      }
    }
  }
}
</style>