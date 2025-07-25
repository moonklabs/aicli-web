<template>
  <div class="privacy-settings">
    <div class="settings-header">
      <h3>개인정보 및 프라이버시</h3>
      <p class="settings-description">
        개인정보 공개 범위와 프라이버시 설정을 관리할 수 있습니다.
      </p>
    </div>

    <div class="settings-content">
      <n-space vertical size="large">
        <!-- 프로파일 공개 설정 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <Eye />
              </n-icon>
              <span>프로파일 공개 설정</span>
            </n-space>
          </template>

          <div class="privacy-section">
            <n-space vertical size="medium">
              <div class="privacy-item">
                <div class="item-info">
                  <h5>프로파일 공개 범위</h5>
                  <p>다른 사용자들이 내 프로파일을 볼 수 있는 범위를 설정합니다.</p>
                </div>
                <div class="item-action">
                  <n-select
                    v-model:value="localSettings.profileVisibility"
                    :options="profileVisibilityOptions"
                    style="width: 150px"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <div class="visibility-description">
                <n-alert :type="visibilityAlertType" :show-icon="true" size="small">
                  <template #header>
                    {{ visibilityTitle }}
                  </template>
                  {{ visibilityDescription }}
                </n-alert>
              </div>
            </n-space>
          </div>
        </n-card>

        <!-- 개인정보 노출 설정 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <Shield />
              </n-icon>
              <span>개인정보 노출 설정</span>
            </n-space>
          </template>

          <div class="privacy-section">
            <n-space vertical size="medium">
              <div class="privacy-item">
                <div class="item-info">
                  <h5>이메일 주소 공개</h5>
                  <p>다른 사용자들이 내 이메일 주소를 볼 수 있도록 허용합니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.showEmail"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <div class="privacy-item">
                <div class="item-info">
                  <h5>전화번호 공개</h5>
                  <p>다른 사용자들이 내 전화번호를 볼 수 있도록 허용합니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.showPhone"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <div class="privacy-item">
                <div class="item-info">
                  <h5>온라인 상태 표시</h5>
                  <p>다른 사용자들이 내가 온라인인지 오프라인인지 볼 수 있도록 허용합니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.showOnlineStatus"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>
            </n-space>
          </div>
        </n-card>

        <!-- 소통 및 상호작용 설정 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <ChatBubbles />
              </n-icon>
              <span>소통 및 상호작용</span>
            </n-space>
          </template>

          <div class="privacy-section">
            <n-space vertical size="medium">
              <div class="privacy-item">
                <div class="item-info">
                  <h5>다이렉트 메시지 허용</h5>
                  <p>다른 사용자들이 나에게 개인 메시지를 보낼 수 있도록 허용합니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.allowDirectMessages"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <div class="privacy-item">
                <div class="item-info">
                  <h5>친구 요청 허용</h5>
                  <p>다른 사용자들이 나에게 친구 요청을 보낼 수 있도록 허용합니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.allowFriendRequests"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>
            </n-space>
          </div>
        </n-card>

        <!-- 데이터 및 개인정보 처리 동의 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <Document />
              </n-icon>
              <span>데이터 처리 동의</span>
            </n-space>
          </template>

          <div class="privacy-section">
            <n-space vertical size="medium">
              <div class="privacy-item">
                <div class="item-info">
                  <h5>개인정보 처리 동의</h5>
                  <p>서비스 제공을 위한 개인정보 수집 및 처리에 동의합니다. (필수)</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.dataProcessingConsent"
                    :disabled="true"
                  />
                </div>
              </div>

              <div class="privacy-item">
                <div class="item-info">
                  <h5>마케팅 정보 수신 동의</h5>
                  <p>신규 서비스, 이벤트, 프로모션 등 마케팅 정보 수신에 동의합니다. (선택)</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.marketingConsent"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <div class="privacy-item">
                <div class="item-info">
                  <h5>분석 데이터 수집 동의</h5>
                  <p>서비스 개선을 위한 사용 패턴 분석 데이터 수집에 동의합니다. (선택)</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.analyticsConsent"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <!-- 동의 관련 안내 -->
              <div class="consent-notice">
                <n-alert type="info" :show-icon="true" size="small">
                  <template #header>개인정보 처리 방침</template>
                  <p>개인정보 처리에 대한 자세한 내용은 개인정보 처리방침을 확인해주세요.</p>
                  <template #action>
                    <n-button
                      text
                      type="primary"
                      size="small"
                      @click="openPrivacyPolicy"
                    >
                      처리방침 보기
                    </n-button>
                  </template>
                </n-alert>
              </div>
            </n-space>
          </div>
        </n-card>

        <!-- 데이터 다운로드 및 삭제 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <Download />
              </n-icon>
              <span>내 데이터 관리</span>
            </n-space>
          </template>

          <div class="privacy-section">
            <n-space vertical size="medium">
              <div class="privacy-item">
                <div class="item-info">
                  <h5>내 데이터 다운로드</h5>
                  <p>계정과 관련된 모든 데이터를 다운로드할 수 있습니다.</p>
                </div>
                <div class="item-action">
                  <n-button
                    type="primary"
                    ghost
                    :loading="downloadingData"
                    @click="requestDataDownload"
                  >
                    데이터 요청
                  </n-button>
                </div>
              </div>

              <div class="privacy-item">
                <div class="item-info">
                  <h5>계정 및 데이터 삭제</h5>
                  <p>계정과 관련된 모든 데이터를 영구적으로 삭제할 수 있습니다.</p>
                </div>
                <div class="item-action">
                  <n-button
                    type="error"
                    ghost
                    @click="showDeleteAccountModal"
                  >
                    계정 삭제
                  </n-button>
                </div>
              </div>

              <!-- 데이터 다운로드 요청 상태 -->
              <div v-if="dataDownloadRequests.length > 0" class="download-requests">
                <h6>데이터 다운로드 요청 기록</h6>
                <div class="requests-list">
                  <div
                    v-for="request in dataDownloadRequests"
                    :key="request.id"
                    class="request-item"
                  >
                    <div class="request-info">
                      <span class="request-date">{{ formatDate(request.createdAt) }}</span>
                      <n-tag :type="getRequestStatusType(request.status)" size="small">
                        {{ getRequestStatusText(request.status) }}
                      </n-tag>
                    </div>
                    <div class="request-actions">
                      <n-button
                        v-if="request.status === 'completed' && request.downloadUrl"
                        size="small"
                        type="primary"
                        ghost
                        @click="downloadData(request.downloadUrl)"
                      >
                        다운로드
                      </n-button>
                    </div>
                  </div>
                </div>
              </div>
            </n-space>
          </div>
        </n-card>
      </n-space>
    </div>

    <!-- 개인정보 처리방침 모달 -->
    <n-modal
      v-model:show="showPrivacyPolicyModal"
      preset="card"
      title="개인정보 처리방침"
      size="large"
      :bordered="false"
    >
      <div class="privacy-policy-content">
        <!-- 실제 구현에서는 정적 파일에서 내용을 로드하거나 API에서 가져옴 -->
        <div class="policy-section">
          <h4>1. 개인정보의 수집 및 이용 목적</h4>
          <p>회사는 다음의 목적을 위하여 개인정보를 처리합니다...</p>
        </div>

        <div class="policy-section">
          <h4>2. 처리하는 개인정보 항목</h4>
          <p>회사는 다음의 개인정보 항목을 처리하고 있습니다...</p>
        </div>

        <div class="policy-section">
          <h4>3. 개인정보의 처리 및 보유 기간</h4>
          <p>개인정보는 원칙적으로 개인정보의 수집 및 이용목적이 달성되면 지체없이 파기합니다...</p>
        </div>

        <!-- 더 많은 내용... -->
      </div>
    </n-modal>

    <!-- 계정 삭제 확인 모달 -->
    <n-modal
      v-model:show="showDeleteModal"
      preset="dialog"
      title="계정 삭제"
      type="error"
      :closable="false"
      :mask-closable="false"
    >
      <div class="delete-warning">
        <p><strong>주의: 이 작업은 되돌릴 수 없습니다!</strong></p>
        <p>계정을 삭제하면 다음 데이터가 영구적으로 삭제됩니다:</p>
        <ul>
          <li>프로파일 정보 및 설정</li>
          <li>워크스페이스 및 프로젝트</li>
          <li>세션 기록 및 활동 로그</li>
          <li>기타 모든 사용자 데이터</li>
        </ul>
        <p>계속하려면 비밀번호를 입력하고 확인 버튼을 클릭하세요.</p>
      </div>

      <template #action>
        <n-space justify="end">
          <n-button @click="showDeleteModal = false">취소</n-button>
          <n-button
            type="error"
            @click="goToAccountDangerZone"
          >
            계정 삭제로 이동
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useRouter } from 'vue-router'
import {
  ChatbubblesSharp as ChatBubbles,
  DocumentTextSharp as Document,
  DownloadSharp as Download,
  EyeSharp as Eye,
  ShieldSharp as Shield,
} from '@vicons/ionicons5'
import { profileApi } from '@/api/services'
import type { PrivacySettings, UpdatePrivacySettingsRequest } from '@/types/api'

// Props
interface Props {
  settings?: PrivacySettings | null
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
})

// Emits
const emit = defineEmits<{
  update: [settings: UpdatePrivacySettingsRequest]
}>()

// 데이터 다운로드 요청 인터페이스
interface DataDownloadRequest {
  id: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  createdAt: string
  completedAt?: string
  downloadUrl?: string
}

// 컴포저블
const message = useMessage()
const router = useRouter()

// 반응형 상태
const downloadingData = ref(false)
const showPrivacyPolicyModal = ref(false)
const showDeleteModal = ref(false)
const dataDownloadRequests = ref<DataDownloadRequest[]>([])

// 로컬 설정
const localSettings = reactive<PrivacySettings>({
  userId: '',
  profileVisibility: 'private',
  showEmail: false,
  showPhone: false,
  showOnlineStatus: true,
  allowDirectMessages: true,
  allowFriendRequests: true,
  dataProcessingConsent: true,
  marketingConsent: false,
  analyticsConsent: false,
  updatedAt: '',
})

// 옵션 데이터
const profileVisibilityOptions = [
  { label: '공개', value: 'public' },
  { label: '연락처만', value: 'contacts' },
  { label: '비공개', value: 'private' },
]

// 계산된 속성
const visibilityAlertType = computed(() => {
  switch (localSettings.profileVisibility) {
    case 'public': return 'warning'
    case 'contacts': return 'info'
    case 'private': return 'success'
    default: return 'info'
  }
})

const visibilityTitle = computed(() => {
  switch (localSettings.profileVisibility) {
    case 'public': return '공개 프로파일'
    case 'contacts': return '제한적 공개'
    case 'private': return '비공개 프로파일'
    default: return '알 수 없음'
  }
})

const visibilityDescription = computed(() => {
  switch (localSettings.profileVisibility) {
    case 'public':
      return '모든 사용자가 내 프로파일과 기본 정보를 볼 수 있습니다.'
    case 'contacts':
      return '연락처에 등록된 사용자만 내 프로파일을 볼 수 있습니다.'
    case 'private':
      return '다른 사용자들은 내 프로파일을 볼 수 없습니다.'
    default:
      return ''
  }
})

// 메서드
const handleSettingChange = () => {
  // 디바운스를 위한 지연
  clearTimeout(window.privacyUpdateTimeout)
  window.privacyUpdateTimeout = setTimeout(() => {
    updateSettings()
  }, 500)
}

const updateSettings = () => {
  const updateData: UpdatePrivacySettingsRequest = {
    profileVisibility: localSettings.profileVisibility,
    showEmail: localSettings.showEmail,
    showPhone: localSettings.showPhone,
    showOnlineStatus: localSettings.showOnlineStatus,
    allowDirectMessages: localSettings.allowDirectMessages,
    allowFriendRequests: localSettings.allowFriendRequests,
    dataProcessingConsent: localSettings.dataProcessingConsent,
    marketingConsent: localSettings.marketingConsent,
    analyticsConsent: localSettings.analyticsConsent,
  }

  emit('update', updateData)
}

const openPrivacyPolicy = () => {
  showPrivacyPolicyModal.value = true
}

const requestDataDownload = async () => {
  downloadingData.value = true
  try {
    // 실제 구현에서는 데이터 다운로드 요청 API 호출
    await new Promise(resolve => setTimeout(resolve, 1000))

    // 새로운 요청 추가
    dataDownloadRequests.value.unshift({
      id: Date.now().toString(),
      status: 'pending',
      createdAt: new Date().toISOString(),
    })

    message.success('데이터 다운로드 요청이 접수되었습니다. 처리 완료 시 이메일로 알려드리겠습니다.')
  } catch (error) {
    console.error('데이터 다운로드 요청 실패:', error)
    message.error('데이터 다운로드 요청에 실패했습니다')
  } finally {
    downloadingData.value = false
  }
}

const downloadData = (url: string) => {
  // 실제 구현에서는 안전한 다운로드 링크 처리
  window.open(url, '_blank')
  message.success('데이터 다운로드를 시작합니다')
}

const showDeleteAccountModal = () => {
  showDeleteModal.value = true
}

const goToAccountDangerZone = () => {
  showDeleteModal.value = false
  // ProfileEditView의 계정 관리 탭으로 이동
  router.push('/profile?tab=account')
}

const getRequestStatusType = (status: string) => {
  switch (status) {
    case 'completed': return 'success'
    case 'failed': return 'error'
    case 'processing': return 'warning'
    default: return 'info'
  }
}

const getRequestStatusText = (status: string) => {
  switch (status) {
    case 'pending': return '대기 중'
    case 'processing': return '처리 중'
    case 'completed': return '완료'
    case 'failed': return '실패'
    default: return '알 수 없음'
  }
}

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString('ko-KR', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

// 설정 동기화
watch(() => props.settings, (newSettings) => {
  if (newSettings) {
    Object.assign(localSettings, newSettings)
  }
}, { immediate: true, deep: true })

// 전역 타입 확장
declare global {
  interface Window {
    privacyUpdateTimeout: number
  }
}
</script>

<style scoped lang="scss">
.privacy-settings {
  .settings-header {
    margin-bottom: 24px;

    h3 {
      margin: 0 0 8px 0;
      font-size: 20px;
      font-weight: 500;
      color: var(--text-color-1);
    }

    .settings-description {
      margin: 0;
      color: var(--text-color-2);
      font-size: 14px;
      line-height: 1.4;
    }
  }

  .settings-content {
    .privacy-section {
      .privacy-item {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
        padding: 16px 0;
        border-bottom: 1px solid var(--border-color);

        &:last-child {
          border-bottom: none;
        }

        .item-info {
          flex: 1;

          h5 {
            margin: 0 0 4px 0;
            font-size: 16px;
            font-weight: 500;
            color: var(--text-color-1);
          }

          p {
            margin: 0;
            font-size: 14px;
            color: var(--text-color-2);
            line-height: 1.4;
          }
        }

        .item-action {
          flex-shrink: 0;
          display: flex;
          align-items: center;
        }
      }

      .visibility-description {
        margin-top: 8px;
      }

      .consent-notice {
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid var(--border-color);
      }

      .download-requests {
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid var(--border-color);

        h6 {
          margin: 0 0 12px 0;
          font-size: 14px;
          font-weight: 500;
          color: var(--text-color-1);
        }

        .requests-list {
          .request-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 8px 0;
            border-bottom: 1px solid var(--border-color);

            &:last-child {
              border-bottom: none;
            }

            .request-info {
              display: flex;
              align-items: center;
              gap: 8px;

              .request-date {
                font-size: 12px;
                color: var(--text-color-3);
              }
            }

            .request-actions {
              flex-shrink: 0;
            }
          }
        }
      }
    }
  }

  .privacy-policy-content {
    max-height: 400px;
    overflow-y: auto;

    .policy-section {
      margin-bottom: 24px;

      h4 {
        margin: 0 0 8px 0;
        font-size: 16px;
        font-weight: 500;
        color: var(--text-color-1);
      }

      p {
        margin: 0;
        font-size: 14px;
        color: var(--text-color-2);
        line-height: 1.6;
      }
    }
  }

  .delete-warning {
    p {
      margin: 0 0 12px 0;
      font-size: 14px;
      line-height: 1.5;
    }

    ul {
      margin: 12px 0;
      padding-left: 20px;

      li {
        margin-bottom: 4px;
        font-size: 14px;
        color: var(--text-color-2);
      }
    }
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .privacy-settings {
    .settings-content {
      .privacy-section {
        .privacy-item {
          flex-direction: column;
          align-items: flex-start;
          gap: 12px;

          .item-action {
            width: 100%;
            justify-content: flex-end;

            .n-select {
              width: 100% !important;
            }
          }
        }

        .download-requests {
          .requests-list {
            .request-item {
              flex-direction: column;
              align-items: flex-start;
              gap: 8px;

              .request-actions {
                width: 100%;
                display: flex;
                justify-content: flex-end;
              }
            }
          }
        }
      }
    }
  }
}

@media (max-width: 480px) {
  .privacy-settings {
    .settings-content {
      .privacy-section {
        .privacy-item {
          .item-action {
            justify-content: flex-start;
          }
        }
      }
    }
  }
}
</style>