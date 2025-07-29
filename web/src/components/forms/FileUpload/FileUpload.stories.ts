import type { Meta, StoryObj } from '@storybook/vue3'
// action 함수를 간단한 console.log로 대체
const action = (name: string) => (...args: any[]) => {
  console.log(`[${name}]`, ...args)
}
import { ref } from 'vue'
import FileUpload from './FileUpload.vue'

const meta: Meta<typeof FileUpload> = {
  title: 'Forms/FileUpload',
  component: FileUpload,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
### FileUpload 컴포넌트

드래그 앤 드롭을 지원하는 고급 파일 업로드 컴포넌트입니다.

#### 주요 기능
- 드래그 앤 드롭 파일 업로드
- 다중 파일 선택 지원
- 파일 크기 제한
- 파일 타입 필터링
- 업로드 진행률 표시
- 파일 미리보기 (이미지)
- 파일 삭제 기능
- 배치 작업 (전체 삭제, 재시도)
- 자동 업로드 옵션
- 접근성 표준 준수
        `,
      },
    },
  },
  argTypes: {
    modelValue: {
      control: { type: 'object' },
      description: '선택된 파일 배열',
    },
    accept: {
      control: { type: 'text' },
      description: '허용되는 파일 타입',
    },
    multiple: {
      control: { type: 'boolean' },
      description: '다중 파일 선택 가능 여부',
    },
    maxSize: {
      control: { type: 'number' },
      description: '최대 파일 크기 (bytes)',
    },
    maxFiles: {
      control: { type: 'number' },
      description: '최대 파일 개수',
    },
    disabled: {
      control: { type: 'boolean' },
      description: '비활성화 여부',
    },
    autoUpload: {
      control: { type: 'boolean' },
      description: '자동 업로드 여부',
    },
    showBatchActions: {
      control: { type: 'boolean' },
      description: '배치 작업 버튼 표시 여부',
    },
    primaryText: {
      control: { type: 'text' },
      description: '메인 안내 텍스트',
    },
    secondaryText: {
      control: { type: 'text' },
      description: '보조 안내 텍스트',
    },
  },
  args: {
    multiple: true,
    maxSize: 10 * 1024 * 1024, // 10MB
    maxFiles: 10,
    autoUpload: false,
    showBatchActions: true,
    primaryText: 'Click to upload or drag and drop',
    secondaryText: 'Maximum file size: 10MB',
  },
}

export default meta
type Story = StoryObj<typeof meta>

// 기본 스토리
export const Default: Story = {
  args: {
    modelValue: [],
  },
  render: (args) => ({
    components: { FileUpload },
    setup() {
      const files = ref<File[]>(args.modelValue)
      
      return {
        args,
        files,
        onUpdate: (value: File[]) => {
          files.value = value
          action('update:modelValue')(value)
        },
        onChange: action('change'),
        onRemove: action('remove'),
        onError: action('error'),
      }
    },
    template: `
      <div style="width: 600px;">
        <FileUpload
          v-bind="args"
          v-model="files"
          @update:modelValue="onUpdate"
          @change="onChange"
          @remove="onRemove"
          @error="onError"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          선택된 파일 수: {{ files.length }}
        </div>
      </div>
    `,
  }),
}

// 이미지 전용
export const ImageOnly: Story = {
  args: {
    modelValue: [],
    accept: 'image/*',
    primaryText: '이미지를 업로드하세요',
    secondaryText: '지원 형식: JPG, PNG, GIF, WebP (최대 5MB)',
    maxSize: 5 * 1024 * 1024,
  },
  render: (args) => ({
    components: { FileUpload },
    setup() {
      const files = ref<File[]>(args.modelValue)
      
      return {
        args,
        files,
        onUpdate: (value: File[]) => {
          files.value = value
          action('update:modelValue')(value)
        },
      }
    },
    template: `
      <div style="width: 600px;">
        <FileUpload
          v-bind="args"
          v-model="files"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 단일 파일만
export const SingleFile: Story = {
  args: {
    modelValue: [],
    multiple: false,
    primaryText: '파일 하나를 선택하세요',
    secondaryText: '한 번에 하나의 파일만 업로드할 수 있습니다',
  },
  render: (args) => ({
    components: { FileUpload },
    setup() {
      const files = ref<File[]>(args.modelValue)
      
      return {
        args,
        files,
        onUpdate: (value: File[]) => {
          files.value = value
          action('update:modelValue')(value)
        },
      }
    },
    template: `
      <div style="width: 600px;">
        <FileUpload
          v-bind="args"
          v-model="files"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 문서 파일 전용
export const DocumentsOnly: Story = {
  args: {
    modelValue: [],
    accept: '.pdf,.doc,.docx,.txt,.csv,.xlsx',
    primaryText: '문서를 업로드하세요',
    secondaryText: '지원 형식: PDF, Word, Excel, TXT, CSV',
  },
  render: (args) => ({
    components: { FileUpload },
    setup() {
      const files = ref<File[]>(args.modelValue)
      
      return {
        args,
        files,
        onUpdate: (value: File[]) => {
          files.value = value
        },
      }
    },
    template: `
      <div style="width: 600px;">
        <FileUpload
          v-bind="args"
          v-model="files"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 자동 업로드 시뮬레이션
export const WithAutoUpload: Story = {
  args: {
    modelValue: [],
    autoUpload: true,
    uploadFunction: async (file: File, onProgress: (progress: number) => void, controller: AbortController) => {
      // 업로드 시뮬레이션
      for (let i = 0; i <= 100; i += 10) {
        await new Promise(resolve => setTimeout(resolve, 200))
        onProgress(i)
        
        // 취소 확인
        if (controller.signal.aborted) {
          throw new Error('Upload cancelled')
        }
      }
      
      // 가상의 업로드 결과 반환
      return {
        url: `https://example.com/uploads/${file.name}`,
        id: Math.random().toString(36).substr(2, 9),
      }
    },
  },
  render: (args) => ({
    components: { FileUpload },
    setup() {
      const files = ref<File[]>(args.modelValue)
      
      return {
        args,
        files,
        onUpdate: (value: File[]) => {
          files.value = value
          action('update:modelValue')(value)
        },
        onUploadComplete: action('upload-complete'),
        onUploadError: action('upload-error'),
      }
    },
    template: `
      <div style="width: 600px;">
        <FileUpload
          v-bind="args"
          v-model="files"
          @update:modelValue="onUpdate"
          @upload-complete="onUploadComplete"
          @upload-error="onUploadError"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          <p>파일을 선택하면 자동으로 업로드가 시작됩니다.</p>
        </div>
      </div>
    `,
  }),
}

// 파일 크기 제한
export const WithSizeLimit: Story = {
  args: {
    modelValue: [],
    maxSize: 1024 * 1024, // 1MB
    primaryText: '작은 파일을 업로드하세요',
    secondaryText: '최대 파일 크기: 1MB',
  },
  render: (args) => ({
    components: { FileUpload },
    setup() {
      const files = ref<File[]>(args.modelValue)
      const errors = ref<string[]>([])
      
      return {
        args,
        files,
        errors,
        onUpdate: (value: File[]) => {
          files.value = value
        },
        onError: (error: any) => {
          errors.value.push(error.message)
          action('error')(error)
        },
      }
    },
    template: `
      <div style="width: 600px;">
        <FileUpload
          v-bind="args"
          v-model="files"
          @update:modelValue="onUpdate"
          @error="onError"
        />
        <div v-if="errors.length > 0" style="margin-top: 16px; color: red; font-size: 14px;">
          <div v-for="(error, index) in errors" :key="index">{{ error }}</div>
        </div>
      </div>
    `,
  }),
}

// 파일 개수 제한
export const WithFileLimit: Story = {
  args: {
    modelValue: [],
    maxFiles: 3,
    primaryText: '최대 3개의 파일을 업로드하세요',
    secondaryText: '파일 개수 제한: 3개',
  },
  render: (args) => ({
    components: { FileUpload },
    setup() {
      const files = ref<File[]>(args.modelValue)
      
      return {
        args,
        files,
        onUpdate: (value: File[]) => {
          files.value = value
        },
        onError: action('error'),
      }
    },
    template: `
      <div style="width: 600px;">
        <FileUpload
          v-bind="args"
          v-model="files"
          @update:modelValue="onUpdate"
          @error="onError"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          선택된 파일: {{ files.length }} / {{ args.maxFiles }}
        </div>
      </div>
    `,
  }),
}

// 배치 작업 숨기기
export const WithoutBatchActions: Story = {
  args: {
    modelValue: [],
    showBatchActions: false,
  },
  render: (args) => ({
    components: { FileUpload },
    setup() {
      const files = ref<File[]>(args.modelValue)
      
      return {
        args,
        files,
        onUpdate: (value: File[]) => {
          files.value = value
        },
      }
    },
    template: `
      <div style="width: 600px;">
        <FileUpload
          v-bind="args"
          v-model="files"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 비활성화된 상태
export const Disabled: Story = {
  args: {
    modelValue: [],
    disabled: true,
    primaryText: '파일 업로드가 비활성화되었습니다',
    secondaryText: '현재 파일을 업로드할 수 없습니다',
  },
  render: (args) => ({
    components: { FileUpload },
    setup() {
      const files = ref<File[]>(args.modelValue)
      
      return {
        args,
        files,
      }
    },
    template: `
      <div style="width: 600px;">
        <FileUpload
          v-bind="args"
          v-model="files"
        />
      </div>
    `,
  }),
}

// 커스텀 업로드 함수 (에러 시뮬레이션)
export const WithUploadError: Story = {
  args: {
    modelValue: [],
    autoUpload: true,
    uploadFunction: async (file: File, onProgress: (progress: number) => void) => {
      // 업로드 진행 시뮬레이션
      for (let i = 0; i <= 50; i += 10) {
        await new Promise(resolve => setTimeout(resolve, 100))
        onProgress(i)
      }
      
      // 에러 발생
      throw new Error(`Failed to upload ${file.name}`)
    },
  },
  render: (args) => ({
    components: { FileUpload },
    setup() {
      const files = ref<File[]>(args.modelValue)
      
      return {
        args,
        files,
        onUpdate: (value: File[]) => {
          files.value = value
        },
        onUploadError: action('upload-error'),
      }
    },
    template: `
      <div style="width: 600px;">
        <FileUpload
          v-bind="args"
          v-model="files"
          @update:modelValue="onUpdate"
          @upload-error="onUploadError"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          업로드 중 에러가 발생하도록 시뮬레이션되었습니다.
        </div>
      </div>
    `,
  }),
}