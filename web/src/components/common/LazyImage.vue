<template>
  <div
    ref="containerRef"
    class="lazy-image"
    :class="{
      'lazy-image--loading': isLoading,
      'lazy-image--loaded': isLoaded,
      'lazy-image--error': hasError,
    }"
    :style="containerStyle"
  >
    <!-- 플레이스홀더 -->
    <div
      v-if="isLoading || hasError"
      class="lazy-image__placeholder"
      :style="placeholderStyle"
    >
      <slot v-if="hasError" name="error">
        <div class="lazy-image__error">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21,15 16,10 5,21"/>
          </svg>
          <span>이미지를 불러올 수 없습니다</span>
        </div>
      </slot>

      <slot v-else name="loading">
        <div class="lazy-image__loading">
          <div class="lazy-image__skeleton" />
        </div>
      </slot>
    </div>

    <!-- 실제 이미지 -->
    <img
      v-show="isLoaded"
      ref="imageRef"
      class="lazy-image__img"
      :src="currentSrc"
      :alt="alt"
      :sizes="sizes"
      :srcset="srcset"
      @load="handleLoad"
      @error="handleError"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

export interface LazyImageProps {
  // 이미지 소스
  src: string
  // 반응형 이미지
  srcset?: string
  sizes?: string
  // WebP 지원
  webpSrc?: string
  // 대체 텍스트
  alt?: string
  // 컨테이너 크기
  width?: string | number
  height?: string | number
  // 플레이스홀더
  placeholder?: string
  // 배경색
  backgroundColor?: string
  // 객체 핏
  objectFit?: 'fill' | 'contain' | 'cover' | 'none' | 'scale-down'
  // 지연 로딩
  lazy?: boolean
  // 임계값 (뷰포트로부터의 거리)
  threshold?: string
}

const props = withDefaults(defineProps<LazyImageProps>(), {
  lazy: true,
  threshold: '100px',
  objectFit: 'cover',
  backgroundColor: '#f5f5f5',
})

// 상태
const isLoading = ref(true)
const isLoaded = ref(false)
const hasError = ref(false)
const shouldLoad = ref(!props.lazy)

// 레퍼런스
const containerRef = ref<HTMLElement>()
const imageRef = ref<HTMLImageElement>()

// 현재 사용할 이미지 소스
const currentSrc = computed(() => {
  if (!shouldLoad.value) return ''

  // WebP 지원 확인
  if (props.webpSrc && supportsWebP.value) {
    return props.webpSrc
  }

  return props.src
})

// 컨테이너 스타일
const containerStyle = computed(() => ({
  width: typeof props.width === 'number' ? `${props.width}px` : props.width,
  height: typeof props.height === 'number' ? `${props.height}px` : props.height,
}))

// 플레이스홀더 스타일
const placeholderStyle = computed(() => ({
  backgroundColor: props.backgroundColor,
  backgroundImage: props.placeholder ? `url(${props.placeholder})` : undefined,
  backgroundSize: 'cover',
  backgroundPosition: 'center',
}))

// WebP 지원 감지
const supportsWebP = ref(false)

// WebP 지원 확인
function checkWebPSupport(): Promise<boolean> {
  return new Promise((resolve) => {
    const webP = new Image()
    webP.onload = webP.onerror = () => {
      resolve(webP.height === 2)
    }
    webP.src = 'data:image/webp;base64,UklGRjoAAABXRUJQVlA4IC4AAACyAgCdASoCAAIALmk0mk0iIiIiIgBoSygABc6WWgAA/veff/0PP8bA//LwYAAA'
  })
}

// 이미지 로드 처리
function handleLoad() {
  isLoading.value = false
  isLoaded.value = true
  hasError.value = false
}

// 이미지 에러 처리
function handleError() {
  isLoading.value = false
  isLoaded.value = false
  hasError.value = true
}

// 상태 초기화
function resetState() {
  isLoading.value = true
  isLoaded.value = false
  hasError.value = false
}

// Intersection Observer
let observer: IntersectionObserver | null = null

function createObserver() {
  if (!props.lazy || !containerRef.value) return

  observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          shouldLoad.value = true
          observer?.disconnect()
        }
      })
    },
    {
      rootMargin: props.threshold,
    },
  )

  observer.observe(containerRef.value)
}

function destroyObserver() {
  if (observer) {
    observer.disconnect()
    observer = null
  }
}

// 생명주기
onMounted(async () => {
  // WebP 지원 확인
  supportsWebP.value = await checkWebPSupport()

  // Intersection Observer 설정
  if (props.lazy) {
    createObserver()
  }
})

onUnmounted(() => {
  destroyObserver()
})

// src 변경 감지
watch(() => props.src, () => {
  resetState()

  if (props.lazy) {
    shouldLoad.value = false
    createObserver()
  }
})

// shouldLoad 변경 감지
watch(shouldLoad, (newValue) => {
  if (newValue && currentSrc.value) {
    // 이미지 미리 로드
    const img = new Image()
    img.onload = handleLoad
    img.onerror = handleError
    img.src = currentSrc.value
  }
})
</script>

<style scoped>
/* 컨테이너 */
.lazy-image {
  position: relative;
  display: inline-block;
  overflow: hidden;
  background-color: var(--placeholder-bg, #f5f5f5);
}

/* 플레이스홀더 */
.lazy-image__placeholder {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
}

/* 스켈레톤 로딩 */
.lazy-image__skeleton {
  width: 100%;
  height: 100%;
  background: linear-gradient(
    90deg,
    transparent,
    rgba(255, 255, 255, 0.4),
    transparent
  );
  background-size: 200% 100%;
  animation: skeleton-loading 1.5s infinite;
}

@keyframes skeleton-loading {
  0% {
    background-position: -200% 0;
  }
  100% {
    background-position: 200% 0;
  }
}

/* 로딩 상태 */
.lazy-image__loading {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-color-secondary, #9ca3af);
}

/* 에러 상태 */
.lazy-image__error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-color-secondary, #9ca3af);
  font-size: 12px;
  text-align: center;
  padding: 16px;
}

.lazy-image__error svg {
  width: 32px;
  height: 32px;
  opacity: 0.5;
}

/* 실제 이미지 */
.lazy-image__img {
  width: 100%;
  height: 100%;
  object-fit: v-bind('props.objectFit');
  transition: opacity 0.3s ease;
}

/* 상태별 스타일 */
.lazy-image--loading .lazy-image__img {
  opacity: 0;
}

.lazy-image--loaded .lazy-image__img {
  opacity: 1;
}

.lazy-image--error .lazy-image__img {
  opacity: 0;
}

/* 반응형 */
@media (max-width: 640px) {
  .lazy-image__error {
    padding: 12px;
    font-size: 11px;
  }

  .lazy-image__error svg {
    width: 24px;
    height: 24px;
  }
}

/* 다크 모드 */
@media (prefers-color-scheme: dark) {
  .lazy-image {
    --placeholder-bg: #374151;
    --text-color-secondary: #9ca3af;
  }
}
</style>