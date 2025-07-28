<template>
  <div
    class="file-upload"
    :class="{
      'file-upload--disabled': disabled,
      'file-upload--dragging': isDragging
    }"
  >
    <!-- 드롭 영역 -->
    <div
      ref="dropZoneRef"
      class="file-upload__drop-zone"
      :class="{
        'file-upload__drop-zone--active': isDragging,
        'file-upload__drop-zone--disabled': disabled
      }"
      @drop.prevent="handleDrop"
      @dragover.prevent="handleDragOver"
      @dragleave.prevent="handleDragLeave"
      @click="handleClick"
      role="button"
      :tabindex="disabled ? -1 : 0"
      :aria-label="dropZoneAriaLabel"
      :aria-disabled="disabled"
      @keydown.enter.space.prevent="handleClick"
    >
      <input
        ref="fileInputRef"
        type="file"
        class="file-upload__input"
        :multiple="multiple"
        :accept="accept"
        :disabled="disabled"
        @change="handleFileSelect"
        :aria-describedby="ariaDescribedby"
      />

      <div class="file-upload__drop-content">
        <slot name="drop-zone" :is-dragging="isDragging">
          <svg class="file-upload__icon" viewBox="0 0 48 48" fill="none">
            <path d="M24 32V16m0 0l-6 6m6-6l6 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M12 40h24M12 8h24a4 4 0 014 4v24a4 4 0 01-4 4H12a4 4 0 01-4-4V12a4 4 0 014-4z" stroke="currentColor" stroke-width="2"/>
          </svg>
          <p class="file-upload__text">
            <span class="file-upload__text-primary">
              {{ primaryText || 'Click to upload or drag and drop' }}
            </span>
            <span v-if="secondaryText" class="file-upload__text-secondary">
              {{ secondaryText }}
            </span>
          </p>
        </slot>
      </div>
    </div>

    <!-- 파일 목록 -->
    <TransitionGroup
      v-if="fileList.length > 0"
      name="file-upload-item"
      tag="div"
      class="file-upload__list"
    >
      <div
        v-for="file in fileList"
        :key="file.id"
        class="file-upload__item"
        :class="{
          'file-upload__item--error': file.status === 'error',
          'file-upload__item--success': file.status === 'success'
        }"
      >
        <!-- 파일 미리보기 -->
        <div class="file-upload__preview">
          <img
            v-if="file.preview && isImageFile(file)"
            :src="file.preview"
            :alt="file.name"
            class="file-upload__preview-image"
          />
          <div v-else class="file-upload__preview-icon">
            <svg viewBox="0 0 24 24" fill="currentColor">
              <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8l-6-6z"/>
              <path d="M14 2v6h6M10 12h4M10 16h4" stroke="white" stroke-width="1.5"/>
            </svg>
            <span class="file-upload__preview-ext">{{ getFileExtension(file.name) }}</span>
          </div>
        </div>

        <!-- 파일 정보 -->
        <div class="file-upload__info">
          <p class="file-upload__name">{{ file.name }}</p>
          <p class="file-upload__meta">
            <span>{{ formatFileSize(file.size) }}</span>
            <span v-if="file.status === 'error'" class="file-upload__error">
              {{ file.error }}
            </span>
          </p>

          <!-- 진행률 바 -->
          <div v-if="file.status === 'uploading'" class="file-upload__progress">
            <div
              class="file-upload__progress-bar"
              :style="{ width: `${file.progress}%` }"
            />
          </div>
        </div>

        <!-- 액션 버튼 -->
        <div class="file-upload__actions">
          <button
            v-if="file.status === 'pending' || file.status === 'uploading'"
            @click="cancelUpload(file)"
            class="file-upload__action"
            :aria-label="`Cancel upload of ${file.name}`"
          >
            <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
              <path d="M10 2a8 8 0 100 16 8 8 0 000-16zM7.172 7.172a.5.5 0 01.656 0L10 9.344l2.172-2.172a.5.5 0 11.656.656L10.656 10l2.172 2.172a.5.5 0 01-.656.656L10 10.656l-2.172 2.172a.5.5 0 01-.656-.656L9.344 10 7.172 7.828a.5.5 0 010-.656z"/>
            </svg>
          </button>

          <button
            v-else
            @click="removeFile(file)"
            class="file-upload__action file-upload__action--remove"
            :aria-label="`Remove ${file.name}`"
          >
            <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
              <path d="M9 2a1 1 0 00-1 1v1H4a1 1 0 000 2h1v10a2 2 0 002 2h6a2 2 0 002-2V6h1a1 1 0 100-2h-4V3a1 1 0 00-1-1H9zm0 2h2v1H9V4zm1 5a1 1 0 011 1v5a1 1 0 11-2 0v-5a1 1 0 011-1z"/>
            </svg>
          </button>
        </div>
      </div>
    </TransitionGroup>

    <!-- 일괄 액션 -->
    <div v-if="fileList.length > 0 && showBatchActions" class="file-upload__batch-actions">
      <button
        @click="uploadAll"
        :disabled="!hasUploadableFiles"
        class="file-upload__batch-button file-upload__batch-button--primary"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
          <path d="M7.646 2.646a.5.5 0 01.708 0l3 3a.5.5 0 01-.708.708L8.5 4.207V11.5a.5.5 0 01-1 0V4.207L5.354 6.354a.5.5 0 11-.708-.708l3-3z"/>
          <path d="M2 13.5a.5.5 0 01.5-.5h11a.5.5 0 010 1h-11a.5.5 0 01-.5-.5z"/>
        </svg>
        Upload All
      </button>

      <button
        @click="clearAll"
        class="file-upload__batch-button file-upload__batch-button--secondary"
      >
        Clear All
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Ref } from 'vue'

interface FileItem {
  id: string
  file: File
  name: string
  size: number
  type: string
  status: 'pending' | 'uploading' | 'success' | 'error'
  progress: number
  preview?: string
  error?: string
  uploadController?: AbortController
}

interface Props {
  modelValue: File[]
  accept?: string
  multiple?: boolean
  maxSize?: number // in bytes
  maxFiles?: number
  disabled?: boolean
  autoUpload?: boolean
  showBatchActions?: boolean
  primaryText?: string
  secondaryText?: string
  dropZoneAriaLabel?: string
  ariaDescribedby?: string
  uploadFunction?: (file: File, onProgress: (progress: number) => void, controller: AbortController) => Promise<any>
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: () => [],
  multiple: true,
  maxSize: 10 * 1024 * 1024, // 10MB
  maxFiles: 10,
  disabled: false,
  autoUpload: false,
  showBatchActions: true,
  dropZoneAriaLabel: 'File upload drop zone',
})

const emit = defineEmits<{
  'update:modelValue': [files: File[]]
  'upload': [file: FileItem]
  'remove': [file: FileItem]
  'error': [error: string, file?: FileItem]
  'change': [files: FileItem[]]
}>()

// State
const fileList = ref<FileItem[]>([])
const isDragging = ref(false)
const dragCounter = ref(0)

// Refs
const dropZoneRef = ref<HTMLDivElement>()
const fileInputRef = ref<HTMLInputElement>()

// Computed
const hasUploadableFiles = computed(() => {
  return fileList.value.some(f => f.status === 'pending')
})

// Methods
const generateFileId = (): string => {
  return `file-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
}

const isImageFile = (file: FileItem): boolean => {
  return file.type.startsWith('image/')
}

const getFileExtension = (filename: string): string => {
  const parts = filename.split('.')
  return parts.length > 1 ? parts[parts.length - 1].toUpperCase() : 'FILE'
}

const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

const validateFile = (file: File): string | null => {
  // Check file size
  if (file.size > props.maxSize) {
    return `File size exceeds ${formatFileSize(props.maxSize)}`
  }

  // Check file type
  if (props.accept) {
    const acceptedTypes = props.accept.split(',').map(t => t.trim())
    const fileType = file.type
    const fileExtension = `.${file.name.split('.').pop()}`

    const isAccepted = acceptedTypes.some(type => {
      if (type.startsWith('.')) {
        return fileExtension.toLowerCase() === type.toLowerCase()
      }
      if (type.endsWith('/*')) {
        return fileType.startsWith(type.replace('/*', ''))
      }
      return fileType === type
    })

    if (!isAccepted) {
      return 'File type not accepted'
    }
  }

  return null
}

const createFileItem = async (file: File): Promise<FileItem> => {
  const fileItem: FileItem = {
    id: generateFileId(),
    file,
    name: file.name,
    size: file.size,
    type: file.type,
    status: 'pending',
    progress: 0,
  }

  // Create preview for images
  if (file.type.startsWith('image/')) {
    try {
      fileItem.preview = await createImagePreview(file)
    } catch (error) {
      console.error('Failed to create preview:', error)
    }
  }

  return fileItem
}

const createImagePreview = (file: File): Promise<string> => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = (e) => resolve(e.target?.result as string)
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

const addFiles = async (files: File[]) => {
  // Check max files limit
  const availableSlots = props.maxFiles - fileList.value.length
  if (availableSlots <= 0) {
    emit('error', `Maximum ${props.maxFiles} files allowed`)
    return
  }

  const filesToAdd = files.slice(0, availableSlots)
  const fileItems: FileItem[] = []

  for (const file of filesToAdd) {
    const error = validateFile(file)
    if (error) {
      const errorItem = await createFileItem(file)
      errorItem.status = 'error'
      errorItem.error = error
      fileItems.push(errorItem)
    } else {
      fileItems.push(await createFileItem(file))
    }
  }

  fileList.value.push(...fileItems)
  updateModelValue()
  emit('change', fileList.value)

  // Auto upload if enabled
  if (props.autoUpload) {
    fileItems
      .filter(f => f.status === 'pending')
      .forEach(f => uploadFile(f))
  }
}

const updateModelValue = () => {
  const validFiles = fileList.value
    .filter(f => f.status !== 'error')
    .map(f => f.file)
  emit('update:modelValue', validFiles)
}

const uploadFile = async (fileItem: FileItem) => {
  if (!props.uploadFunction || fileItem.status !== 'pending') return

  fileItem.status = 'uploading'
  fileItem.uploadController = new AbortController()

  try {
    await props.uploadFunction(
      fileItem.file,
      (progress) => {
        fileItem.progress = progress
      },
      fileItem.uploadController,
    )
    fileItem.status = 'success'
    fileItem.progress = 100
    emit('upload', fileItem)
  } catch (error) {
    if (error instanceof Error && error.name === 'AbortError') {
      fileItem.status = 'pending'
      fileItem.progress = 0
    } else {
      fileItem.status = 'error'
      fileItem.error = error instanceof Error ? error.message : 'Upload failed'
      emit('error', fileItem.error, fileItem)
    }
  }
}

const cancelUpload = (fileItem: FileItem) => {
  if (fileItem.uploadController) {
    fileItem.uploadController.abort()
  }
}

const removeFile = (fileItem: FileItem) => {
  const index = fileList.value.indexOf(fileItem)
  if (index > -1) {
    if (fileItem.uploadController) {
      fileItem.uploadController.abort()
    }
    fileList.value.splice(index, 1)
    updateModelValue()
    emit('remove', fileItem)
    emit('change', fileList.value)
  }
}

const uploadAll = () => {
  fileList.value
    .filter(f => f.status === 'pending')
    .forEach(f => uploadFile(f))
}

const clearAll = () => {
  fileList.value.forEach(f => {
    if (f.uploadController) {
      f.uploadController.abort()
    }
  })
  fileList.value = []
  updateModelValue()
  emit('change', fileList.value)
}

// Event handlers
const handleClick = () => {
  if (!props.disabled) {
    fileInputRef.value?.click()
  }
}

const handleFileSelect = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    addFiles(Array.from(input.files))
    input.value = '' // Reset input
  }
}

const handleDrop = (event: DragEvent) => {
  isDragging.value = false
  dragCounter.value = 0

  if (props.disabled || !event.dataTransfer) return

  const files = Array.from(event.dataTransfer.files)
  if (files.length > 0) {
    addFiles(files)
  }
}

const handleDragOver = (event: DragEvent) => {
  if (!props.disabled && event.dataTransfer) {
    event.dataTransfer.dropEffect = 'copy'
    isDragging.value = true
  }
}

const handleDragLeave = () => {
  dragCounter.value--
  if (dragCounter.value === 0) {
    isDragging.value = false
  }
}
</script>

<style lang="scss" scoped>
.file-upload {
  @apply w-full;

  &--disabled {
    @apply opacity-60 cursor-not-allowed;
  }

  &__drop-zone {
    @apply relative;
    @apply border-2 border-dashed border-gray-300 rounded-lg;
    @apply p-6 text-center;
    @apply cursor-pointer;
    @apply transition-all duration-200;
    @apply hover:border-gray-400;

    &:focus {
      @apply outline-none ring-2 ring-blue-500 ring-opacity-25;
      @apply border-blue-500;
    }

    &--active {
      @apply border-blue-500 bg-blue-50;
    }

    &--disabled {
      @apply cursor-not-allowed hover:border-gray-300;
    }
  }

  &__input {
    @apply absolute inset-0 w-full h-full opacity-0 cursor-pointer;

    &:disabled {
      @apply cursor-not-allowed;
    }
  }

  &__drop-content {
    @apply pointer-events-none;
  }

  &__icon {
    @apply w-12 h-12 mx-auto mb-4;
    @apply text-gray-400;
  }

  &__text {
    @apply space-y-1;
  }

  &__text-primary {
    @apply block text-base text-gray-700;
  }

  &__text-secondary {
    @apply block text-sm text-gray-500;
  }

  &__list {
    @apply mt-4 space-y-2;
  }

  &__item {
    @apply flex items-center gap-3;
    @apply p-3 bg-white border border-gray-200 rounded-lg;
    @apply transition-all duration-200;

    &--error {
      @apply border-red-300 bg-red-50;
    }

    &--success {
      @apply border-green-300 bg-green-50;
    }
  }

  &__preview {
    @apply flex-shrink-0 w-12 h-12;
  }

  &__preview-image {
    @apply w-full h-full object-cover rounded;
  }

  &__preview-icon {
    @apply relative w-full h-full;
    @apply flex items-center justify-center;
    @apply bg-gray-100 rounded;
    @apply text-gray-600;

    svg {
      @apply w-8 h-8;
    }
  }

  &__preview-ext {
    @apply absolute bottom-0 right-0;
    @apply text-xs font-bold text-gray-500;
    @apply bg-white px-1 rounded;
  }

  &__info {
    @apply flex-1 min-w-0;
  }

  &__name {
    @apply text-sm font-medium text-gray-900;
    @apply truncate;
  }

  &__meta {
    @apply text-xs text-gray-500;
    @apply flex items-center gap-2;
  }

  &__error {
    @apply text-red-600;
  }

  &__progress {
    @apply mt-1 w-full h-1 bg-gray-200 rounded-full overflow-hidden;
  }

  &__progress-bar {
    @apply h-full bg-blue-500;
    @apply transition-all duration-300 ease-out;
  }

  &__actions {
    @apply flex-shrink-0;
  }

  &__action {
    @apply p-1 rounded;
    @apply text-gray-400 hover:text-gray-600;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply transition-colors duration-150;

    &--remove {
      @apply hover:text-red-600;
    }
  }

  &__batch-actions {
    @apply mt-4 flex gap-2;
  }

  &__batch-button {
    @apply px-4 py-2;
    @apply text-sm font-medium;
    @apply border rounded-md;
    @apply inline-flex items-center gap-2;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply transition-colors duration-150;

    &--primary {
      @apply bg-blue-600 text-white border-blue-600;
      @apply hover:bg-blue-700;

      &:disabled {
        @apply opacity-50 cursor-not-allowed;
        @apply hover:bg-blue-600;
      }
    }

    &--secondary {
      @apply bg-white text-gray-700 border-gray-300;
      @apply hover:bg-gray-50;
    }
  }
}

// Dark mode
.dark .file-upload {
  &__drop-zone {
    @apply border-gray-600;
    @apply hover:border-gray-500;

    &--active {
      @apply border-blue-500 bg-blue-900 bg-opacity-20;
    }
  }

  &__icon {
    @apply text-gray-500;
  }

  &__text-primary {
    @apply text-gray-200;
  }

  &__text-secondary {
    @apply text-gray-400;
  }

  &__item {
    @apply bg-gray-800 border-gray-700;

    &--error {
      @apply border-red-700 bg-red-900 bg-opacity-20;
    }

    &--success {
      @apply border-green-700 bg-green-900 bg-opacity-20;
    }
  }

  &__preview-icon {
    @apply bg-gray-700 text-gray-400;
  }

  &__preview-ext {
    @apply text-gray-400 bg-gray-800;
  }

  &__name {
    @apply text-gray-100;
  }

  &__meta {
    @apply text-gray-400;
  }

  &__error {
    @apply text-red-400;
  }

  &__progress {
    @apply bg-gray-700;
  }

  &__action {
    @apply text-gray-500 hover:text-gray-300;

    &--remove {
      @apply hover:text-red-400;
    }
  }

  &__batch-button {
    &--primary {
      @apply bg-blue-600 text-white border-blue-600;
      @apply hover:bg-blue-700;
    }

    &--secondary {
      @apply bg-gray-800 text-gray-300 border-gray-600;
      @apply hover:bg-gray-700;
    }
  }
}

// Animations
.file-upload-item-enter-active,
.file-upload-item-leave-active {
  @apply transition-all duration-300;
}

.file-upload-item-enter-from,
.file-upload-item-leave-to {
  @apply opacity-0 transform scale-95;
}

.file-upload-item-move {
  @apply transition-transform duration-300;
}
</style>