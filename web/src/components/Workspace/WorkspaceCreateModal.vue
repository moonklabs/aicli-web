<template>
  <n-modal v-model:show="showModal" preset="card" style="width: 600px" title="새 워크스페이스 생성">
    <n-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-placement="top"
      require-mark-placement="right-hanging"
    >
      <n-form-item label="워크스페이스 이름" path="name">
        <n-input
          v-model:value="formData.name"
          placeholder="워크스페이스의 이름을 입력하세요"
          :maxlength="50"
          show-count
        />
      </n-form-item>

      <n-form-item label="프로젝트 경로" path="path">
        <n-input-group>
          <n-input
            v-model:value="formData.path"
            placeholder="/workspace/my-project"
            :maxlength="200"
          />
          <n-button @click="selectDirectory">
            <template #icon>
              <n-icon><FolderOpen /></n-icon>
            </template>
            찾아보기
          </n-button>
        </n-input-group>
        <n-text depth="3" style="font-size: 12px; margin-top: 4px">
          Docker 컨테이너에 마운트될 로컬 디렉토리 경로
        </n-text>
      </n-form-item>

      <n-form-item label="설명 (선택사항)" path="description">
        <n-input
          v-model:value="formData.description"
          type="textarea"
          placeholder="워크스페이스에 대한 간단한 설명을 입력하세요"
          :maxlength="200"
          show-count
          :rows="3"
        />
      </n-form-item>

      <n-divider />

      <n-form-item label="Docker 설정">
        <n-space vertical style="width: 100%">
          <n-form-item label="베이스 이미지" path="config.baseImage">
            <n-select
              v-model:value="formData.config.baseImage"
              :options="baseImageOptions"
              placeholder="Docker 베이스 이미지를 선택하세요"
            />
          </n-form-item>

          <n-form-item label="작업 디렉토리" path="config.workingDir">
            <n-input
              v-model:value="formData.config.workingDir"
              placeholder="/workspace"
            />
          </n-form-item>

          <n-form-item label="환경 변수">
            <div class="env-vars">
              <div
                v-for="(env, index) in formData.config.environment"
                :key="index"
                class="env-var-row"
              >
                <n-input
                  v-model:value="env.key"
                  placeholder="변수명"
                  style="flex: 1"
                />
                <span>=</span>
                <n-input
                  v-model:value="env.value"
                  placeholder="값"
                  style="flex: 2"
                />
                <n-button
                  quaternary
                  circle
                  type="error"
                  @click="removeEnvVar(index)"
                >
                  <template #icon>
                    <n-icon><X /></n-icon>
                  </template>
                </n-button>
              </div>
              <n-button dashed block @click="addEnvVar">
                <template #icon>
                  <n-icon><Plus /></n-icon>
                </template>
                환경 변수 추가
              </n-button>
            </div>
          </n-form-item>

          <n-form-item label="포트 매핑">
            <div class="port-mappings">
              <div
                v-for="(port, index) in formData.config.ports"
                :key="index"
                class="port-row"
              >
                <n-input-number
                  v-model:value="port.host"
                  placeholder="호스트 포트"
                  :min="1"
                  :max="65535"
                  style="flex: 1"
                />
                <span>:</span>
                <n-input-number
                  v-model:value="port.container"
                  placeholder="컨테이너 포트"
                  :min="1"
                  :max="65535"
                  style="flex: 1"
                />
                <n-button
                  quaternary
                  circle
                  type="error"
                  @click="removePort(index)"
                >
                  <template #icon>
                    <n-icon><X /></n-icon>
                  </template>
                </n-button>
              </div>
              <n-button dashed block @click="addPort">
                <template #icon>
                  <n-icon><Plus /></n-icon>
                </template>
                포트 매핑 추가
              </n-button>
            </div>
          </n-form-item>
        </n-space>
      </n-form-item>
    </n-form>

    <template #footer>
      <n-space justify="end">
        <n-button @click="handleCancel">취소</n-button>
        <n-button type="primary" :loading="creating" @click="handleCreate">
          생성
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  NModal,
  NForm,
  NFormItem,
  NInput,
  NInputGroup,
  NInputNumber,
  NSelect,
  NButton,
  NSpace,
  NIcon,
  NText,
  NDivider,
  useMessage,
  type FormInst
} from 'naive-ui'
import { FolderOpen, Plus, X } from '@vicons/lucide'
import { useWorkspaceStore } from '@/stores/workspace'
import type { Workspace } from '@/stores/workspace'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'update:show', value: boolean): void
  (e: 'created', workspace: Workspace): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const workspaceStore = useWorkspaceStore()
const message = useMessage()
const formRef = ref<FormInst | null>(null)

// 모달 표시 상태 (양방향 바인딩)
const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

// 폼 데이터
const formData = ref({
  name: '',
  path: '',
  description: '',
  config: {
    baseImage: 'node:18-alpine',
    workingDir: '/workspace',
    environment: [] as Array<{ key: string; value: string }>,
    ports: [] as Array<{ host: number | null; container: number | null }>
  }
})

const creating = ref(false)

// 베이스 이미지 옵션
const baseImageOptions = [
  { label: 'Node.js 18 (Alpine)', value: 'node:18-alpine' },
  { label: 'Node.js 20 (Alpine)', value: 'node:20-alpine' },
  { label: 'Python 3.11', value: 'python:3.11-slim' },
  { label: 'Python 3.12', value: 'python:3.12-slim' },
  { label: 'Ubuntu 22.04', value: 'ubuntu:22.04' },
  { label: 'Debian 12', value: 'debian:12-slim' },
  { label: 'Alpine Linux', value: 'alpine:latest' },
  { label: '사용자 정의', value: 'custom' }
]

// 유효성 검사 규칙
const rules = {
  name: [
    { required: true, message: '워크스페이스 이름을 입력하세요' },
    { min: 2, max: 50, message: '이름은 2-50자 사이여야 합니다' },
    { 
      pattern: /^[a-zA-Z0-9가-힣\-_\s]+$/, 
      message: '이름에는 영문, 숫자, 한글, 하이픈, 언더스코어만 사용할 수 있습니다' 
    }
  ],
  path: [
    { required: true, message: '프로젝트 경로를 입력하세요' },
    { 
      pattern: /^\/[^\s]*$/, 
      message: '절대 경로를 입력하세요 (예: /workspace/my-project)' 
    }
  ],
  description: [
    { max: 200, message: '설명은 200자 이하여야 합니다' }
  ]
}

// 환경 변수 관리
const addEnvVar = (): void => {
  formData.value.config.environment.push({ key: '', value: '' })
}

const removeEnvVar = (index: number): void => {
  formData.value.config.environment.splice(index, 1)
}

// 포트 매핑 관리
const addPort = (): void => {
  formData.value.config.ports.push({ host: null, container: null })
}

const removePort = (index: number): void => {
  formData.value.config.ports.splice(index, 1)
}

// 디렉토리 선택
const selectDirectory = (): void => {
  // TODO: 파일 시스템 탐색기 구현
  message.info('디렉토리 탐색기는 준비 중입니다')
}

// 폼 처리
const handleCreate = async (): Promise<void> => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    creating.value = true

    // 환경 변수를 객체로 변환
    const environment: Record<string, string> = {}
    formData.value.config.environment.forEach(env => {
      if (env.key && env.value) {
        environment[env.key] = env.value
      }
    })

    // 포트 배열 정리
    const ports = formData.value.config.ports
      .filter(port => port.host && port.container)
      .map(port => port.host as number)

    const workspaceData = {
      name: formData.value.name,
      path: formData.value.path,
      description: formData.value.description || undefined,
      config: {
        baseImage: formData.value.config.baseImage,
        workingDir: formData.value.config.workingDir,
        environment,
        ports
      }
    }

    const workspace = await workspaceStore.createWorkspace(workspaceData)
    if (workspace) {
      emit('created', workspace)
      resetForm()
    }
  } catch (error) {
    console.error('Workspace creation error:', error)
    if (error instanceof Error) {
      message.error(`워크스페이스 생성 실패: ${error.message}`)
    }
  } finally {
    creating.value = false
  }
}

const handleCancel = (): void => {
  showModal.value = false
  resetForm()
}

const resetForm = (): void => {
  formData.value = {
    name: '',
    path: '',
    description: '',
    config: {
      baseImage: 'node:18-alpine',
      workingDir: '/workspace',
      environment: [],
      ports: []
    }
  }
}

// 모달이 닫힐 때 폼 리셋
watch(showModal, (newValue) => {
  if (!newValue) {
    resetForm()
  }
})
</script>

<style scoped>
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
</style>