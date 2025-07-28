<template>
  <NModal
    v-model:show="showModal"
    preset="card"
    style="width: 700px"
    :title="`${workspace?.name} 설정`"
  >
    <div v-if="workspace" class="workspace-settings">
      <NTabs type="line" animated>
        <NTabPane name="general" tab="일반 설정">
          <NForm
            ref="generalFormRef"
            :model="generalForm"
            :rules="generalRules"
            label-placement="top"
          >
            <NFormItem label="워크스페이스 이름" path="name">
              <NInput
                v-model:value="generalForm.name"
                placeholder="워크스페이스의 이름을 입력하세요"
                :maxlength="50"
                show-count
              />
            </NFormItem>

            <NFormItem label="설명" path="description">
              <NInput
                v-model:value="generalForm.description"
                type="textarea"
                placeholder="워크스페이스에 대한 간단한 설명을 입력하세요"
                :maxlength="200"
                show-count
                :rows="3"
              />
            </NFormItem>

            <NFormItem label="프로젝트 경로">
              <NInput
                :value="workspace.path"
                disabled
                placeholder="프로젝트 경로는 변경할 수 없습니다"
              />
              <NText depth="3" style="font-size: 12px; margin-top: 4px">
                보안상의 이유로 프로젝트 경로는 워크스페이스 생성 후 변경할 수 없습니다
              </NText>
            </NFormItem>
          </NForm>
        </NTabPane>

        <NTabPane name="docker" tab="Docker 설정">
          <NForm
            ref="dockerFormRef"
            :model="dockerForm"
            label-placement="top"
          >
            <NAlert type="warning" style="margin-bottom: 16px">
              <template #icon>
                <span>⚠️</span>
              </template>
              Docker 설정 변경 시 컨테이너가 재시작됩니다
            </NAlert>

            <NFormItem label="베이스 이미지">
              <NSelect
                v-model:value="dockerForm.baseImage"
                :options="baseImageOptions"
                placeholder="Docker 베이스 이미지를 선택하세요"
              />
            </NFormItem>

            <NFormItem label="작업 디렉토리">
              <NInput
                v-model:value="dockerForm.workingDir"
                placeholder="/workspace"
              />
            </NFormItem>

            <NFormItem label="환경 변수">
              <div class="env-vars">
                <div
                  v-for="(env, index) in dockerForm.environment"
                  :key="index"
                  class="env-var-row"
                >
                  <NInput
                    v-model:value="env.key"
                    placeholder="변수명"
                    style="flex: 1"
                  />
                  <span>=</span>
                  <NInput
                    v-model:value="env.value"
                    placeholder="값"
                    style="flex: 2"
                  />
                  <NButton
                    quaternary
                    circle
                    type="error"
                    @click="removeEnvVar(index)"
                  >
                    <template #icon>
                      <span>✕</span>
                    </template>
                  </NButton>
                </div>
                <NButton dashed block @click="addEnvVar">
                  <template #icon>
                    <span>+</span>
                  </template>
                  환경 변수 추가
                </NButton>
              </div>
            </NFormItem>

            <NFormItem label="포트 매핑">
              <div class="port-mappings">
                <div
                  v-for="(port, index) in dockerForm.ports"
                  :key="index"
                  class="port-row"
                >
                  <NInputNumber
                    v-model:value="port.host"
                    placeholder="호스트 포트"
                    :min="1"
                    :max="65535"
                    style="flex: 1"
                  />
                  <span>:</span>
                  <NInputNumber
                    v-model:value="port.container"
                    placeholder="컨테이너 포트"
                    :min="1"
                    :max="65535"
                    style="flex: 1"
                  />
                  <NButton
                    quaternary
                    circle
                    type="error"
                    @click="removePort(index)"
                  >
                    <template #icon>
                      <span>✕</span>
                    </template>
                  </NButton>
                </div>
                <NButton dashed block @click="addPort">
                  <template #icon>
                    <span>+</span>
                  </template>
                  포트 매핑 추가
                </NButton>
              </div>
            </NFormItem>
          </NForm>
        </NTabPane>

        <NTabPane name="info" tab="정보">
          <div class="workspace-info">
            <NDescriptions :column="2" bordered>
              <NDescriptionsItem label="ID">
                {{ workspace.id }}
              </NDescriptionsItem>
              <NDescriptionsItem label="상태">
                <NTag :type="getWorkspaceStatusType(workspace.status)">
                  {{ getWorkspaceStatusText(workspace.status) }}
                </NTag>
              </NDescriptionsItem>
              <NDescriptionsItem label="생성일">
                {{ formatDate(workspace.createdAt) }}
              </NDescriptionsItem>
              <NDescriptionsItem label="수정일">
                {{ formatDate(workspace.updatedAt) }}
              </NDescriptionsItem>
              <NDescriptionsItem label="컨테이너 ID">
                <NText code v-if="workspace.containerId">
                  {{ workspace.containerId }}
                </NText>
                <NText depth="3" v-else>없음</NText>
              </NDescriptionsItem>
              <NDescriptionsItem label="Git 브랜치">
                <NText v-if="workspace.git?.branch">
                  {{ workspace.git.branch }}
                  <span v-if="workspace.git.hasChanges" style="color: var(--n-color-warning)">
                    (변경사항 있음)
                  </span>
                </NText>
                <NText depth="3" v-else>Git 연결 안됨</NText>
              </NDescriptionsItem>
              <NDescriptionsItem label="파일 수">
                {{ workspace.stats?.fileCount || 0 }}
              </NDescriptionsItem>
              <NDescriptionsItem label="코드 라인 수">
                {{ workspace.stats?.lineCount || 0 }}
              </NDescriptionsItem>
            </NDescriptions>

            <NDivider />

            <div class="danger-zone">
              <h4>위험한 작업</h4>
              <NSpace vertical>
                <NAlert type="error">
                  <template #icon>
                    <span>⚠️</span>
                  </template>
                  아래 작업들은 되돌릴 수 없습니다. 신중하게 실행하세요.
                </NAlert>
                <NButton type="error" ghost @click="showResetConfirm = true">
                  워크스페이스 초기화
                </NButton>
              </NSpace>
            </div>
          </div>
        </NTabPane>
      </NTabs>
    </div>

    <template #footer>
      <NSpace justify="end">
        <NButton @click="handleCancel">취소</NButton>
        <NButton type="primary" :loading="saving" @click="handleSave">
          저장
        </NButton>
      </NSpace>
    </template>

    <!-- 초기화 확인 대화상자 -->
    <NModal
      v-model:show="showResetConfirm"
      preset="dialog"
      type="error"
      title="워크스페이스 초기화"
      positive-text="초기화"
      negative-text="취소"
      @positive-click="handleReset"
    >
      <p>
        정말로 <strong>{{ workspace?.name }}</strong> 워크스페이스를 초기화하시겠습니까?
      </p>
      <p class="reset-warning">
        모든 컨테이너가 중지되고 삭제되며, 임시 데이터가 모두 손실됩니다.
        (프로젝트 파일은 보존됩니다)
      </p>
    </NModal>
  </NModal>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import {
  type FormInst,
  NAlert,
  NButton,
  NDescriptions,
  NDescriptionsItem,
  NDivider,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  NTabPane,
  NTabs,
  NTag,
  NText,
  useMessage,
} from 'naive-ui'
// import { Plus, X, AlertTriangle } from '@vicons/lucide' // 이모지로 교체
import type { Workspace } from '@/stores/workspace'

interface Props {
  show: boolean
  workspace: Workspace | null
}

interface Emits {
  (e: 'update:show', value: boolean): void
  (e: 'updated', workspace: Workspace): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const message = useMessage()
const generalFormRef = ref<FormInst | null>(null)
const dockerFormRef = ref<FormInst | null>(null)

// 모달 표시 상태
const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value),
})

const showResetConfirm = ref(false)
const saving = ref(false)

// 폼 데이터
const generalForm = reactive({
  name: '',
  description: '',
})

const dockerForm = reactive({
  baseImage: '',
  workingDir: '',
  environment: [] as Array<{ key: string; value: string }>,
  ports: [] as Array<{ host: number | null; container: number | null }>,
})

// 베이스 이미지 옵션
const baseImageOptions = [
  { label: 'Node.js 18 (Alpine)', value: 'node:18-alpine' },
  { label: 'Node.js 20 (Alpine)', value: 'node:20-alpine' },
  { label: 'Python 3.11', value: 'python:3.11-slim' },
  { label: 'Python 3.12', value: 'python:3.12-slim' },
  { label: 'Ubuntu 22.04', value: 'ubuntu:22.04' },
  { label: 'Debian 12', value: 'debian:12-slim' },
  { label: 'Alpine Linux', value: 'alpine:latest' },
]

// 유효성 검사 규칙
const generalRules = {
  name: [
    { required: true, message: '워크스페이스 이름을 입력하세요' },
    { min: 2, max: 50, message: '이름은 2-50자 사이여야 합니다' },
  ],
  description: [
    { max: 200, message: '설명은 200자 이하여야 합니다' },
  ],
}

// 환경 변수 관리
const addEnvVar = (): void => {
  dockerForm.environment.push({ key: '', value: '' })
}

const removeEnvVar = (index: number): void => {
  dockerForm.environment.splice(index, 1)
}

// 포트 매핑 관리
const addPort = (): void => {
  dockerForm.ports.push({ host: null, container: null })
}

const removePort = (index: number): void => {
  dockerForm.ports.splice(index, 1)
}

// 상태 변환 함수들
const getWorkspaceStatusType = (status: string): 'success' | 'warning' | 'error' | 'info' => {
  switch (status) {
    case 'active': return 'success'
    case 'loading': return 'warning'
    case 'error': return 'error'
    default: return 'info'
  }
}

const getWorkspaceStatusText = (status: string): string => {
  switch (status) {
    case 'active': return '활성'
    case 'idle': return '대기'
    case 'loading': return '로딩중'
    case 'error': return '오류'
    case 'creating': return '생성중'
    case 'deleting': return '삭제중'
    default: return status
  }
}

const formatDate = (dateString: string): string => {
  return new Date(dateString).toLocaleString('ko-KR')
}

// 폼 초기화
const initializeForms = (): void => {
  if (!props.workspace) return

  // 일반 설정 폼
  generalForm.name = props.workspace.name
  generalForm.description = props.workspace.description || ''

  // Docker 설정 폼 (더미 데이터)
  dockerForm.baseImage = 'node:18-alpine'
  dockerForm.workingDir = '/workspace'
  dockerForm.environment = [
    { key: 'NODE_ENV', value: 'development' },
    { key: 'PORT', value: '3000' },
  ]
  dockerForm.ports = [
    { host: 3000, container: 3000 },
    { host: 8080, container: 80 },
  ]
}

// 폼 처리
const handleSave = async (): Promise<void> => {
  if (!props.workspace) return

  try {
    saving.value = true

    // TODO: 실제 API 호출로 업데이트
    const updatedWorkspace: Workspace = {
      ...props.workspace,
      name: generalForm.name,
      description: generalForm.description || undefined,
      updatedAt: new Date().toISOString(),
    }

    emit('updated', updatedWorkspace)
    message.success('워크스페이스 설정이 저장되었습니다')
  } catch (error) {
    message.error('설정 저장에 실패했습니다')
  } finally {
    saving.value = false
  }
}

const handleCancel = (): void => {
  showModal.value = false
}

const handleReset = async (): Promise<void> => {
  if (!props.workspace) return

  try {
    // TODO: 워크스페이스 초기화 API 호출
    message.success('워크스페이스가 초기화되었습니다')
    showResetConfirm.value = false
    showModal.value = false
  } catch (error) {
    message.error('워크스페이스 초기화에 실패했습니다')
  }
}

// 워크스페이스 변경 시 폼 초기화
watch(() => props.workspace, () => {
  if (props.workspace) {
    initializeForms()
  }
}, { immediate: true })

// 모달이 열릴 때 폼 초기화
watch(showModal, (newValue) => {
  if (newValue && props.workspace) {
    initializeForms()
  }
})
</script>

<style scoped>
.workspace-settings {
  min-height: 400px;
}

.env-vars {
  width: 100%;
}

.env-var-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.port-mappings {
  width: 100%;
}

.port-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.port-row span {
  color: var(--n-text-color-3);
  font-weight: bold;
}

.workspace-info {
  padding: 0;
}

.danger-zone {
  margin-top: 24px;
  padding: 16px;
  border: 1px solid var(--n-color-error);
  border-radius: 8px;
  background-color: var(--n-color-error-hover);
}

.danger-zone h4 {
  margin: 0 0 12px 0;
  color: var(--n-color-error);
  font-size: 16px;
  font-weight: 600;
}

.reset-warning {
  color: var(--n-color-error);
  font-size: 12px;
  margin-top: 8px;
}
</style>