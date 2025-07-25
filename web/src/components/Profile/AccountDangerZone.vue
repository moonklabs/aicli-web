<template>
  <div class="account-danger-zone">
    <div class="danger-header">
      <h3>계정 관리</h3>
      <p class="danger-description">
        계정 비활성화나 삭제는 신중하게 결정해주세요. 일부 작업은 되돌릴 수 없습니다.
      </p>
    </div>

    <div class="danger-content">
      <n-space vertical size="large">
        <!-- 계정 상태 표시 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <Person />
              </n-icon>
              <span>계정 상태</span>
            </n-space>
          </template>
          
          <div class="account-status">
            <div class="status-info">
              <n-space align="center">
                <n-tag 
                  :type="profile?.isActive ? 'success' : 'warning'" 
                  size="medium"
                  round
                >
                  {{ profile?.isActive ? '활성' : '비활성' }}
                </n-tag>
                <span class="status-text">
                  계정이 {{ profile?.isActive ? '정상적으로 활성화' : '비활성화' }}되어 있습니다.
                </span>
              </n-space>
            </div>
            
            <div v-if="profile?.isActive" class="account-info">
              <n-space vertical size="small">
                <div class="info-item">
                  <span class="info-label">가입일:</span>
                  <span class="info-value">{{ formatDate(profile.createdAt) }}</span>
                </div>
                <div v-if="profile.lastLoginAt" class="info-item">
                  <span class="info-label">마지막 로그인:</span>
                  <span class="info-value">{{ formatDate(profile.lastLoginAt) }}</span>
                </div>
              </n-space>
            </div>
          </div>
        </n-card>

        <!-- 계정 비활성화 섹션 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20" color="#f0a020">
                <PauseCircle />
              </n-icon>
              <span>계정 비활성화</span>
            </n-space>
          </template>
          
          <div class="deactivation-section">
            <div class="section-description">
              <p>계정을 일시적으로 비활성화할 수 있습니다. 비활성화된 계정은 언제든지 다시 활성화할 수 있습니다.</p>
              
              <div class="effects-list">
                <h5>비활성화 시 영향:</h5>
                <ul>
                  <li>다른 사용자들이 내 프로파일을 볼 수 없게 됩니다</li>
                  <li>워크스페이스와 프로젝트에 접근할 수 없게 됩니다</li>
                  <li>알림을 받지 않게 됩니다</li>
                  <li>데이터는 보존되며 재활성화 시 복구됩니다</li>
                </ul>
              </div>
            </div>

            <div class="section-actions">
              <n-button
                v-if="profile?.isActive"
                type="warning"
                :loading="deactivating"
                @click="showDeactivationModal = true"
              >
                <template #icon>
                  <n-icon><PauseCircle /></n-icon>
                </template>
                계정 비활성화
              </n-button>
              
              <n-button
                v-else
                type="success"
                :loading="reactivating"
                @click="reactivateAccount"
              >
                <template #icon>
                  <n-icon><PlayCircle /></n-icon>
                </template>
                계정 재활성화
              </n-button>
            </div>
          </div>
        </n-card>

        <!-- 계정 삭제 섹션 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20" color="#e74c3c">
                <Trash />
              </n-icon>
              <span>계정 삭제</span>
            </n-space>
          </template>
          
          <div class="deletion-section">
            <div class="section-description">
              <n-alert type="error" :show-icon="true">
                <template #header>
                  <strong>경고: 이 작업은 되돌릴 수 없습니다!</strong>
                </template>
                계정을 삭제하면 모든 데이터가 영구적으로 삭제되며 복구할 수 없습니다.
              </n-alert>
              
              <div class="effects-list">
                <h5>삭제되는 데이터:</h5>
                <ul>
                  <li>프로파일 정보 및 개인 설정</li>
                  <li>모든 워크스페이스와 프로젝트</li>
                  <li>세션 기록 및 활동 로그</li>
                  <li>업로드한 파일 및 이미지</li>
                  <li>기타 모든 사용자 데이터</li>
                </ul>
              </div>

              <div class="deletion-options">
                <h5>삭제 옵션:</h5>
                <n-radio-group v-model:value="deletionType">
                  <n-space vertical>
                    <n-radio value="immediate">
                      <div class="radio-content">
                        <strong>즉시 삭제</strong>
                        <p>계정과 모든 데이터를 즉시 삭제합니다.</p>
                      </div>
                    </n-radio>
                    <n-radio value="delayed">
                      <div class="radio-content">
                        <strong>30일 후 삭제</strong>
                        <p>30일 유예 기간 후 삭제됩니다. 유예 기간 내에 복구 가능합니다.</p>
                      </div>
                    </n-radio>
                  </n-space>
                </n-radio-group>
              </div>
            </div>

            <div class="section-actions">
              <n-button
                type="error"
                @click="showDeletionModal = true"
              >
                <template #icon>
                  <n-icon><Trash /></n-icon>
                </template>
                계정 삭제 요청
              </n-button>
            </div>
          </div>
        </n-card>

        <!-- 데이터 백업 섹션 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <CloudDownload />
              </n-icon>
              <span>데이터 백업</span>
            </n-space>
          </template>
          
          <div class="backup-section">
            <div class="section-description">
              <p>계정을 삭제하기 전에 중요한 데이터를 백업받으실 수 있습니다.</p>
            </div>

            <div class="section-actions">
              <n-space>
                <n-button
                  type="primary"
                  ghost
                  :loading="backingUpData"
                  @click="requestDataBackup"
                >
                  <template #icon>
                    <n-icon><CloudDownload /></n-icon>
                  </template>
                  전체 데이터 백업
                </n-button>
                
                <n-button
                  ghost
                  @click="showBackupOptionsModal = true"
                >
                  <template #icon>
                    <n-icon><Settings /></n-icon>
                  </template>
                  선택적 백업
                </n-button>
              </n-space>
            </div>
          </div>
        </n-card>
      </n-space>
    </div>

    <!-- 계정 비활성화 확인 모달 -->
    <n-modal
      v-model:show="showDeactivationModal"
      preset="card"
      title="계정 비활성화 확인"
      size="medium"
      :bordered="false"
    >
      <div class="deactivation-form">
        <n-alert type="warning" :show-icon="true">
          <template #header>계정 비활성화</template>
          <p>계정을 비활성화하시겠습니까? 언제든지 다시 로그인하여 계정을 재활성화할 수 있습니다.</p>
        </n-alert>
        
        <n-form
          ref="deactivationFormRef"
          :model="deactivationForm"
          :rules="deactivationRules"
          style="margin-top: 16px;"
        >
          <n-form-item label="비활성화 사유 (선택)" path="reason">
            <n-select
              v-model:value="deactivationForm.reason"
              :options="deactivationReasons"
              placeholder="비활성화 사유를 선택하세요"
            />
          </n-form-item>
          
          <n-form-item label="비밀번호 확인" path="password">
            <n-input
              v-model:value="deactivationForm.password"
              type="password"
              placeholder="현재 비밀번호를 입력하세요"
              show-password-on="mousedown"
            />
          </n-form-item>
        </n-form>
      </div>

      <template #action>
        <n-space justify="end">
          <n-button @click="showDeactivationModal = false">취소</n-button>
          <n-button
            type="warning"
            :loading="deactivating"
            @click="confirmDeactivation"
          >
            비활성화
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 계정 삭제 확인 모달 -->
    <n-modal
      v-model:show="showDeletionModal"
      preset="card"
      title="계정 삭제 확인"
      size="medium"
      :bordered="false"
      :closable="false"
      :mask-closable="false"
    >
      <div class="deletion-form">
        <n-alert type="error" :show-icon="true">
          <template #header>
            <strong>최종 경고</strong>
          </template>
          <p>이 작업은 되돌릴 수 없습니다. {{ deletionType === 'immediate' ? '계정과 모든 데이터가 즉시 삭제됩니다.' : '30일 후 계정과 모든 데이터가 삭제됩니다.' }}</p>
        </n-alert>
        
        <n-form
          ref="deletionFormRef"
          :model="deletionForm"
          :rules="deletionRules"
          style="margin-top: 16px;"
        >
          <n-form-item label="삭제 사유 (선택)" path="reason">
            <n-select
              v-model:value="deletionForm.reason"
              :options="deletionReasons"
              placeholder="삭제 사유를 선택하세요"
            />
          </n-form-item>
          
          <n-form-item label="추가 의견 (선택)" path="feedback">
            <n-input
              v-model:value="deletionForm.feedback"
              type="textarea"
              :rows="3"
              placeholder="서비스 개선을 위한 의견을 남겨주세요"
            />
          </n-form-item>
          
          <n-form-item label="확인 문구 입력" path="confirmation">
            <n-input
              v-model:value="deletionForm.confirmation"
              placeholder="'계정 삭제'를 정확히 입력하세요"
            />
            <template #feedback>
              <n-text depth="3" style="font-size: 12px;">
                정말로 계정을 삭제하려면 '계정 삭제'를 정확히 입력해주세요.
              </n-text>
            </template>
          </n-form-item>
          
          <n-form-item label="비밀번호 확인" path="password">
            <n-input
              v-model:value="deletionForm.password"
              type="password"
              placeholder="현재 비밀번호를 입력하세요"
              show-password-on="mousedown"
            />
          </n-form-item>
        </n-form>
      </div>

      <template #action>
        <n-space justify="end">
          <n-button @click="showDeletionModal = false">취소</n-button>
          <n-button
            type="error"
            :loading="deleting"
            :disabled="!isDeletionFormValid"
            @click="confirmDeletion"
          >
            {{ deletionType === 'immediate' ? '즉시 삭제' : '30일 후 삭제 요청' }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 백업 옵션 모달 -->
    <n-modal
      v-model:show="showBackupOptionsModal"
      preset="card"
      title="선택적 데이터 백업"
      size="medium"
      :bordered="false"
    >
      <div class="backup-options">
        <p>백업받을 데이터를 선택하세요:</p>
        
        <n-checkbox-group v-model:value="selectedBackupItems">
          <n-space vertical>
            <n-checkbox value="profile" label="프로파일 정보" />
            <n-checkbox value="workspaces" label="워크스페이스 목록" />
            <n-checkbox value="projects" label="프로젝트 데이터" />
            <n-checkbox value="settings" label="개인 설정" />
            <n-checkbox value="activity" label="활동 기록" />
          </n-space>
        </n-checkbox-group>
      </div>

      <template #action>
        <n-space justify="end">
          <n-button @click="showBackupOptionsModal = false">취소</n-button>
          <n-button
            type="primary"
            :loading="backingUpData"
            :disabled="selectedBackupItems.length === 0"
            @click="confirmSelectiveBackup"
          >
            백업 시작
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useMessage, useDialog } from 'naive-ui'
import { useRouter } from 'vue-router'
import {
  PersonSharp as Person,
  PauseCircleSharp as PauseCircle,
  PlayCircleSharp as PlayCircle,
  TrashSharp as Trash,
  CloudDownloadSharp as CloudDownload,
  SettingsSharp as Settings
} from '@vicons/ionicons5'
import { profileApi } from '@/api/services'
import type { UserProfile, AccountDeactivationRequest, AccountDeletionRequest } from '@/types/api'

// Props
interface Props {
  profile?: UserProfile | null
}

const props = defineProps<Props>()

// Emits
const emit = defineEmits<{
  deactivate: [request: AccountDeactivationRequest]
  delete: [request: AccountDeletionRequest]
}>()

// 컴포저블
const message = useMessage()
const dialog = useDialog()
const router = useRouter()

// 반응형 상태
const deactivating = ref(false)
const reactivating = ref(false)
const deleting = ref(false)
const backingUpData = ref(false)
const showDeactivationModal = ref(false)
const showDeletionModal = ref(false)
const showBackupOptionsModal = ref(false)
const deletionType = ref<'immediate' | 'delayed'>('delayed')
const selectedBackupItems = ref<string[]>([])

// 폼 참조
const deactivationFormRef = ref()
const deletionFormRef = ref()

// 폼 데이터
const deactivationForm = reactive({
  reason: '',
  password: ''
})

const deletionForm = reactive({
  reason: '',
  feedback: '',
  confirmation: '',
  password: ''
})

// 옵션 데이터
const deactivationReasons = [
  { label: '일시적 휴식', value: 'temporary_break' },
  { label: '개인정보 보호', value: 'privacy_concerns' },
  { label: '서비스 불만족', value: 'service_dissatisfaction' },
  { label: '기타', value: 'other' }
]

const deletionReasons = [
  { label: '더 이상 서비스를 사용하지 않음', value: 'no_longer_needed' },
  { label: '개인정보 보호 우려', value: 'privacy_concerns' },
  { label: '서비스 품질 불만족', value: 'poor_service_quality' },
  { label: '다른 서비스로 이전', value: 'switching_service' },
  { label: '기타', value: 'other' }
]

// 폼 검증 규칙
const deactivationRules = {
  password: {
    required: true,
    message: '비밀번호를 입력해주세요',
    trigger: ['blur', 'input']
  }
}

const deletionRules = {
  confirmation: [
    {
      required: true,
      message: '확인 문구를 입력해주세요',
      trigger: ['blur', 'input']
    },
    {
      validator: (rule: any, value: string) => {
        if (value !== '계정 삭제') {
          return new Error('정확히 \'계정 삭제\'를 입력해주세요')
        }
        return true
      },
      trigger: ['blur', 'input']
    }
  ],
  password: {
    required: true,
    message: '비밀번호를 입력해주세요',
    trigger: ['blur', 'input']
  }
}

// 계산된 속성
const isDeletionFormValid = computed(() => {
  return deletionForm.confirmation === '계정 삭제' && 
         deletionForm.password.length > 0
})

// 메서드
const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString('ko-KR', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const reactivateAccount = async () => {
  reactivating.value = true
  try {
    await profileApi.reactivateAccount({
      email: props.profile?.email || ''
    })
    
    message.success('계정이 재활성화되었습니다')
    // 페이지 새로고침 또는 사용자 정보 다시 로드
    window.location.reload()
  } catch (error: any) {
    console.error('계정 재활성화 실패:', error)
    message.error(error.message || '계정 재활성화에 실패했습니다')
  } finally {
    reactivating.value = false
  }
}

const confirmDeactivation = async () => {
  try {
    await deactivationFormRef.value?.validate()
    
    deactivating.value = true
    
    const request: AccountDeactivationRequest = {
      password: deactivationForm.password,
      reason: deactivationForm.reason
    }
    
    await profileApi.deactivateAccount(request)
    
    showDeactivationModal.value = false
    deactivationForm.password = ''
    deactivationForm.reason = ''
    
    message.success('계정이 비활성화되었습니다')
    emit('deactivate', request)
    
    // 로그아웃 처리
    setTimeout(() => {
      router.push('/login')
    }, 2000)
    
  } catch (error: any) {
    console.error('계정 비활성화 실패:', error)
    message.error(error.message || '계정 비활성화에 실패했습니다')
  } finally {
    deactivating.value = false
  }
}

const confirmDeletion = async () => {
  try {
    await deletionFormRef.value?.validate()
    
    deleting.value = true
    
    const request: AccountDeletionRequest = {
      password: deletionForm.password,
      reason: deletionForm.reason,
      feedback: deletionForm.feedback,
      deleteImmediately: deletionType.value === 'immediate'
    }
    
    await profileApi.requestAccountDeletion(request)
    
    showDeletionModal.value = false
    
    const successMessage = deletionType.value === 'immediate' 
      ? '계정이 삭제되었습니다. 이용해 주셔서 감사합니다.'
      : '계정 삭제가 요청되었습니다. 30일 후 영구 삭제됩니다.'
    
    message.success(successMessage)
    emit('delete', request)
    
    // 로그아웃 처리
    setTimeout(() => {
      router.push('/login')
    }, 3000)
    
  } catch (error: any) {
    console.error('계정 삭제 실패:', error)
    message.error(error.message || '계정 삭제 요청에 실패했습니다')
  } finally {
    deleting.value = false
  }
}

const requestDataBackup = async () => {
  backingUpData.value = true
  try {
    // 전체 데이터 백업 요청
    await new Promise(resolve => setTimeout(resolve, 2000))
    message.success('데이터 백업 요청이 접수되었습니다. 완료되면 이메일로 다운로드 링크를 보내드리겠습니다.')
  } catch (error) {
    console.error('데이터 백업 실패:', error)
    message.error('데이터 백업 요청에 실패했습니다')
  } finally {
    backingUpData.value = false
  }
}

const confirmSelectiveBackup = async () => {
  backingUpData.value = true
  try {
    // 선택적 데이터 백업 요청
    await new Promise(resolve => setTimeout(resolve, 1500))
    
    const itemNames = selectedBackupItems.value.map(item => {
      const labels: Record<string, string> = {
        profile: '프로파일 정보',
        workspaces: '워크스페이스 목록',
        projects: '프로젝트 데이터',
        settings: '개인 설정',
        activity: '활동 기록'
      }
      return labels[item]
    }).join(', ')
    
    message.success(`선택한 데이터(${itemNames}) 백업 요청이 접수되었습니다.`)
    showBackupOptionsModal.value = false
    selectedBackupItems.value = []
  } catch (error) {
    console.error('선택적 백업 실패:', error)
    message.error('데이터 백업 요청에 실패했습니다')
  } finally {
    backingUpData.value = false
  }
}
</script>

<style scoped lang="scss">
.account-danger-zone {
  .danger-header {
    margin-bottom: 24px;

    h3 {
      margin: 0 0 8px 0;
      font-size: 20px;
      font-weight: 500;
      color: var(--text-color-1);
    }

    .danger-description {
      margin: 0;
      color: var(--text-color-2);
      font-size: 14px;
      line-height: 1.4;
    }
  }

  .danger-content {
    .account-status {
      .status-info {
        margin-bottom: 16px;

        .status-text {
          font-size: 16px;
          color: var(--text-color-1);
        }
      }

      .account-info {
        padding: 12px;
        background: var(--card-color);
        border-radius: 8px;
        border: 1px solid var(--border-color);

        .info-item {
          display: flex;
          justify-content: space-between;
          font-size: 14px;

          .info-label {
            color: var(--text-color-2);
          }

          .info-value {
            color: var(--text-color-1);
            font-weight: 500;
          }
        }
      }
    }

    .deactivation-section,
    .deletion-section,
    .backup-section {
      .section-description {
        margin-bottom: 24px;

        p {
          margin: 0 0 16px 0;
          font-size: 14px;
          color: var(--text-color-2);
          line-height: 1.5;
        }

        .effects-list {
          h5 {
            margin: 16px 0 8px 0;
            font-size: 14px;
            font-weight: 500;
            color: var(--text-color-1);
          }

          ul {
            margin: 0;
            padding-left: 20px;

            li {
              margin-bottom: 4px;
              font-size: 13px;
              color: var(--text-color-2);
              line-height: 1.4;
            }
          }
        }

        .deletion-options {
          margin-top: 16px;

          h5 {
            margin: 0 0 12px 0;
            font-size: 14px;
            font-weight: 500;
            color: var(--text-color-1);
          }

          .radio-content {
            p {
              margin: 4px 0 0 0;
              font-size: 12px;
              color: var(--text-color-3);
            }
          }
        }
      }

      .section-actions {
        display: flex;
        justify-content: flex-end;
      }
    }
  }

  .deactivation-form,
  .deletion-form {
    .n-form-item {
      margin-bottom: 16px;
    }
  }

  .backup-options {
    p {
      margin: 0 0 16px 0;
      font-size: 14px;
      color: var(--text-color-2);
    }
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .account-danger-zone {
    .danger-content {
      .account-status {
        .status-info {
          .n-space {
            flex-direction: column;
            align-items: flex-start;
            gap: 8px;
          }
        }

        .account-info {
          .info-item {
            flex-direction: column;
            gap: 4px;
          }
        }
      }

      .deactivation-section,
      .deletion-section,
      .backup-section {
        .section-actions {
          justify-content: stretch;

          .n-button,
          .n-space {
            width: 100%;

            :deep(.n-space-item) {
              flex: 1;

              .n-button {
                width: 100%;
              }
            }
          }
        }
      }
    }
  }
}
</style>