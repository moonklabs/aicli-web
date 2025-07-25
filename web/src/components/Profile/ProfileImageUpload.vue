<template>
  <div class="profile-image-upload">
    <div class="upload-area">
      <!-- 현재 이미지 표시 -->
      <div class="current-image">
        <n-avatar
          :size="120"
          :src="currentImage"
          :fallback-src="defaultAvatar"
          round
          class="avatar"
        >
          <template #placeholder>
            <n-icon size="40" :depth="3">
              <Person />
            </n-icon>
          </template>
        </n-avatar>

        <!-- 업로드 오버레이 -->
        <div
          v-if="!uploading"
          class="upload-overlay"
          @click="triggerFileInput"
          @dragover.prevent
          @dragenter.prevent
          @drop.prevent="handleFileDrop"
        >
          <n-icon size="24" color="white">
            <Camera />
          </n-icon>
          <span>변경</span>
        </div>

        <!-- 업로드 중 오버레이 -->
        <div v-else class="uploading-overlay">
          <n-spin size="medium" stroke="white" />
          <span>업로드 중...</span>
        </div>
      </div>

      <!-- 설명 텍스트 -->
      <div class="upload-info">
        <p class="info-text">클릭하거나 파일을 드래그하여 이미지를 업로드하세요</p>
        <p class="info-detail">
          JPG, PNG, GIF 파일을 지원합니다. 최대 파일 크기: 5MB
        </p>
      </div>
    </div>

    <!-- 액션 버튼들 -->
    <div class="action-buttons">
      <n-space>
        <n-button
          type="primary"
          ghost
          :loading="uploading"
          @click="triggerFileInput"
        >
          <template #icon>
            <n-icon><CloudUpload /></n-icon>
          </template>
          이미지 업로드
        </n-button>

        <n-button
          v-if="currentImage"
          type="error"
          ghost
          :disabled="uploading"
          @click="confirmDelete"
        >
          <template #icon>
            <n-icon><Trash /></n-icon>
          </template>
          삭제
        </n-button>
      </n-space>
    </div>

    <!-- 숨겨진 파일 입력 -->
    <input
      ref="fileInputRef"
      type="file"
      accept="image/*"
      style="display: none"
      @change="handleFileSelect"
    />

    <!-- 이미지 크롭 모달 -->
    <n-modal
      v-model:show="showCropModal"
      preset="card"
      title="이미지 크롭"
      size="large"
      :bordered="false"
      :closable="false"
      :mask-closable="false"
    >
      <div class="crop-container">
        <div class="crop-preview">
          <img
            ref="cropImageRef"
            :src="cropImageSrc"
            alt="크롭할 이미지"
            style="max-width: 100%; max-height: 400px;"
          />
        </div>

        <div class="crop-controls">
          <n-space vertical>
            <div class="aspect-ratio-controls">
              <span class="control-label">비율:</span>
              <n-radio-group v-model:value="cropAspectRatio" size="small">
                <n-radio value="1">1:1 (정사각형)</n-radio>
                <n-radio value="free">자유</n-radio>
              </n-radio-group>
            </div>

            <div class="crop-preview-section">
              <span class="control-label">미리보기:</span>
              <div class="preview-container">
                <div class="preview-circle">
                  <canvas ref="previewCanvasRef" width="80" height="80"></canvas>
                </div>
              </div>
            </div>
          </n-space>
        </div>
      </div>

      <template #action>
        <n-space justify="end">
          <n-button @click="cancelCrop">취소</n-button>
          <n-button
            type="primary"
            :loading="uploading"
            @click="confirmCrop"
          >
            적용
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 삭제 확인 모달 -->
    <n-modal
      v-model:show="showDeleteModal"
      preset="dialog"
      title="이미지 삭제"
      content="프로파일 이미지를 삭제하시겠습니까?"
      positive-text="삭제"
      negative-text="취소"
      @positive-click="handleDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useMessage } from 'naive-ui'
import {
  CameraSharp as Camera,
  CloudUploadSharp as CloudUpload,
  PersonSharp as Person,
  TrashSharp as Trash,
} from '@vicons/ionicons5'

// Props
interface Props {
  currentImage?: string
  uploading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  uploading: false,
})

// Emits
const emit = defineEmits<{
  upload: [file: File, cropData?: CropData]
  delete: []
}>()

// 크롭 데이터 인터페이스
interface CropData {
  x: number
  y: number
  width: number
  height: number
  rotate?: number
  scaleX?: number
  scaleY?: number
}

// 컴포저블
const message = useMessage()

// 반응형 상태
const showCropModal = ref(false)
const showDeleteModal = ref(false)
const cropImageSrc = ref('')
const cropAspectRatio = ref('1')
const selectedFile = ref<File | null>(null)

// 참조
const fileInputRef = ref<HTMLInputElement>()
const cropImageRef = ref<HTMLImageElement>()
const previewCanvasRef = ref<HTMLCanvasElement>()

// 기본 아바타 이미지
const defaultAvatar = '/default-avatar.png'

// 파일 검증
const validateFile = (file: File): boolean => {
  // 파일 크기 체크 (5MB)
  const maxSize = 5 * 1024 * 1024
  if (file.size > maxSize) {
    message.error('파일 크기는 5MB를 초과할 수 없습니다')
    return false
  }

  // 파일 타입 체크
  const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/gif']
  if (!allowedTypes.includes(file.type)) {
    message.error('JPG, PNG, GIF 파일만 업로드 가능합니다')
    return false
  }

  return true
}

// 파일 입력 트리거
const triggerFileInput = () => {
  fileInputRef.value?.click()
}

// 파일 선택 처리
const handleFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]

  if (file && validateFile(file)) {
    processFile(file)
  }

  // 입력 초기화
  target.value = ''
}

// 파일 드롭 처리
const handleFileDrop = (event: DragEvent) => {
  const files = event.dataTransfer?.files
  const file = files?.[0]

  if (file && validateFile(file)) {
    processFile(file)
  }
}

// 파일 처리
const processFile = (file: File) => {
  selectedFile.value = file

  // 파일을 데이터 URL로 변환
  const reader = new FileReader()
  reader.onload = (e) => {
    cropImageSrc.value = e.target?.result as string
    showCropModal.value = true

    // 다음 틱에서 크롭 초기화
    nextTick(() => {
      initializeCrop()
    })
  }
  reader.readAsDataURL(file)
}

// 크롭 초기화
const initializeCrop = () => {
  // 여기서 실제 크롭 라이브러리를 초기화해야 함
  // 예: Cropper.js, vue-cropper 등
  // 현재는 기본 구현만 제공
  console.log('크롭 초기화')
}

// 크롭 미리보기 업데이트
const updateCropPreview = () => {
  const canvas = previewCanvasRef.value
  const img = cropImageRef.value

  if (!canvas || !img) return

  const ctx = canvas.getContext('2d')
  if (!ctx) return

  // 기본 미리보기 구현
  ctx.clearRect(0, 0, 80, 80)
  ctx.drawImage(img, 0, 0, 80, 80)
}

// 크롭 확인
const confirmCrop = async () => {
  if (!selectedFile.value) return

  try {
    // 여기서 실제 크롭 데이터를 가져와야 함
    // 현재는 기본 크롭 데이터 사용
    const cropData: CropData = {
      x: 0,
      y: 0,
      width: 100,
      height: 100,
    }

    emit('upload', selectedFile.value, cropData)
    showCropModal.value = false
    selectedFile.value = null
    cropImageSrc.value = ''
  } catch (error) {
    console.error('크롭 처리 실패:', error)
    message.error('이미지 처리에 실패했습니다')
  }
}

// 크롭 취소
const cancelCrop = () => {
  showCropModal.value = false
  selectedFile.value = null
  cropImageSrc.value = ''
}

// 삭제 확인
const confirmDelete = () => {
  showDeleteModal.value = true
}

// 삭제 처리
const handleDelete = () => {
  emit('delete')
  showDeleteModal.value = false
}
</script>

<style scoped lang="scss">
.profile-image-upload {
  .upload-area {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;

    .current-image {
      position: relative;
      cursor: pointer;

      .avatar {
        border: 3px solid var(--border-color);
        transition: all 0.3s ease;

        &:hover {
          border-color: var(--primary-color);
        }
      }

      .upload-overlay,
      .uploading-overlay {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        border-radius: 50%;
        background: rgba(0, 0, 0, 0.6);
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 4px;
        opacity: 0;
        transition: opacity 0.3s ease;
        color: white;
        font-size: 12px;
        font-weight: 500;

        &:hover {
          opacity: 1;
        }
      }

      .uploading-overlay {
        opacity: 1;
      }

      &:hover .upload-overlay {
        opacity: 1;
      }
    }

    .upload-info {
      text-align: center;
      max-width: 280px;

      .info-text {
        margin: 0 0 4px 0;
        font-size: 14px;
        color: var(--text-color-1);
      }

      .info-detail {
        margin: 0;
        font-size: 12px;
        color: var(--text-color-3);
        line-height: 1.4;
      }
    }
  }

  .action-buttons {
    margin-top: 16px;
    display: flex;
    justify-content: center;
  }
}

.crop-container {
  .crop-preview {
    margin-bottom: 24px;
    text-align: center;
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 16px;
    background: var(--card-color);
  }

  .crop-controls {
    .control-label {
      font-size: 14px;
      font-weight: 500;
      color: var(--text-color-1);
    }

    .aspect-ratio-controls {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .crop-preview-section {
      display: flex;
      align-items: center;
      gap: 12px;

      .preview-container {
        .preview-circle {
          width: 80px;
          height: 80px;
          border-radius: 50%;
          border: 2px solid var(--border-color);
          overflow: hidden;
          background: var(--card-color);

          canvas {
            width: 100%;
            height: 100%;
            border-radius: 50%;
          }
        }
      }
    }
  }
}

// 반응형 디자인
@media (max-width: 480px) {
  .profile-image-upload {
    .upload-area {
      .current-image {
        .avatar {
          width: 100px !important;
          height: 100px !important;
        }
      }

      .upload-info {
        .info-text {
          font-size: 13px;
        }

        .info-detail {
          font-size: 11px;
        }
      }
    }

    .action-buttons {
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

  .crop-container {
    .crop-controls {
      .aspect-ratio-controls,
      .crop-preview-section {
        flex-direction: column;
        align-items: flex-start;
        gap: 8px;
      }
    }
  }
}
</style>