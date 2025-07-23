<template>
  <div
    ref="scrollerRef"
    class="virtual-scroller"
    :style="containerStyle"
    @scroll="onScroll"
  >
    <!-- 전체 높이를 설정하는 플레이스홀더 -->
    <div :style="{ height: totalHeight + 'px', position: 'relative' }">
      <!-- 실제 렌더링되는 아이템들 -->
      <div
        :style="{
          transform: `translateY(${offsetY}px)`,
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
        }"
      >
        <div
          v-for="(item, index) in visibleItems"
          :key="item.id || item.key || visibleStartIndex + index"
          :style="{ height: itemHeight + 'px' }"
          :class="[
            'virtual-item',
            { 'virtual-item--selected': isSelected(visibleStartIndex + index) }
          ]"
          :data-index="visibleStartIndex + index"
          @mousedown="startSelection(visibleStartIndex + index, $event)"
          @mouseover="updateSelection(visibleStartIndex + index, $event)"
          @mouseup="endSelection"
        >
          <slot
            :item="item"
            :index="visibleStartIndex + index"
            :is-visible="true"
          >
            {{ item }}
          </slot>
        </div>
      </div>
    </div>

    <!-- 스크롤 표시기 -->
    <div
      v-if="showScrollIndicator && !isAtBottom"
      class="scroll-indicator"
      @click="scrollToBottom"
    >
      <NIcon size="16">
        <svg viewBox="0 0 24 24">
          <path d="M7 14l5 5 5-5z" fill="currentColor" />
        </svg>
      </NIcon>
      <span>{{ newItemsCount }}개의 새 항목</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { NIcon } from 'naive-ui'
import { throttle } from '@/utils/terminal-utils'

interface Props {
  items: any[]
  itemHeight?: number
  overscan?: number
  height?: number | string
  enableSelection?: boolean
  showScrollIndicator?: boolean
  autoScroll?: boolean
}

interface Emits {
  (e: 'selection-change', selection: { start: number; end: number } | null): void
  (e: 'scroll', scrollInfo: { top: number; isAtBottom: boolean }): void
}

const props = withDefaults(defineProps<Props>(), {
  itemHeight: 20,
  overscan: 5,
  height: '100%',
  enableSelection: true,
  showScrollIndicator: true,
  autoScroll: true,
})

const emit = defineEmits<Emits>()

// 템플릿 refs
const scrollerRef = ref<HTMLElement>()

// 상태
const scrollTop = ref(0)
const containerHeight = ref(0)
const isUserScrolling = ref(false)
const selection = ref<{ start: number; end: number } | null>(null)
const isSelecting = ref(false)
const newItemsCount = ref(0)
const lastScrollTime = ref(0)

// 계산된 속성
const totalHeight = computed(() => props.items.length * props.itemHeight)

const visibleStartIndex = computed(() => {
  const start = Math.floor(scrollTop.value / props.itemHeight) - props.overscan
  return Math.max(0, start)
})

const visibleEndIndex = computed(() => {
  const itemsInView = Math.ceil(containerHeight.value / props.itemHeight)
  const end = visibleStartIndex.value + itemsInView + props.overscan * 2
  return Math.min(props.items.length - 1, end)
})

const visibleItems = computed(() => {
  return props.items.slice(visibleStartIndex.value, visibleEndIndex.value + 1)
})

const offsetY = computed(() => visibleStartIndex.value * props.itemHeight)

const containerStyle = computed(() => ({
  height: typeof props.height === 'number' ? `${props.height}px` : props.height,
  overflow: 'auto',
  position: 'relative' as const,
}))

const isAtBottom = computed(() => {
  if (!scrollerRef.value) return true
  const { scrollTop, scrollHeight, clientHeight } = scrollerRef.value
  return scrollTop + clientHeight >= scrollHeight - 5 // 5px 여유
})

// 스크롤 이벤트 처리 (스로틀링 적용)
const onScroll = throttle((event: Event) => {
  const target = event.target as HTMLElement
  scrollTop.value = target.scrollTop
  lastScrollTime.value = Date.now()

  // 사용자가 스크롤 중임을 표시
  isUserScrolling.value = true

  // 일정 시간 후 사용자 스크롤 상태 해제
  setTimeout(() => {
    if (Date.now() - lastScrollTime.value >= 150) {
      isUserScrolling.value = false
    }
  }, 150)

  // 스크롤 정보 emit
  emit('scroll', {
    top: scrollTop.value,
    isAtBottom: isAtBottom.value,
  })

  // 새 항목 표시기 업데이트
  if (isAtBottom.value) {
    newItemsCount.value = 0
  }
}, 16) // 60fps

// 컨테이너 크기 계산
const updateContainerHeight = () => {
  if (scrollerRef.value) {
    containerHeight.value = scrollerRef.value.clientHeight
  }
}

// 맨 아래로 스크롤
const scrollToBottom = () => {
  if (scrollerRef.value) {
    scrollerRef.value.scrollTop = scrollerRef.value.scrollHeight
    newItemsCount.value = 0
  }
}

// 특정 인덱스로 스크롤
const scrollToIndex = (index: number, behavior: ScrollBehavior = 'smooth') => {
  if (scrollerRef.value) {
    const targetScrollTop = index * props.itemHeight
    scrollerRef.value.scrollTo({
      top: targetScrollTop,
      behavior,
    })
  }
}

// 선택 관련 메서드
const isSelected = (index: number): boolean => {
  if (!selection.value || !props.enableSelection) return false
  const { start, end } = selection.value
  return index >= Math.min(start, end) && index <= Math.max(start, end)
}

const startSelection = (index: number, event: MouseEvent) => {
  if (!props.enableSelection) return

  event.preventDefault()
  isSelecting.value = true
  selection.value = { start: index, end: index }
  emit('selection-change', selection.value)
}

const updateSelection = (index: number, event: MouseEvent) => {
  if (!props.enableSelection || !isSelecting.value || !selection.value) return

  selection.value.end = index
  emit('selection-change', selection.value)
}

const endSelection = () => {
  isSelecting.value = false
}

const clearSelection = () => {
  selection.value = null
  emit('selection-change', null)
}

// 키보드 이벤트 처리
const onKeyDown = (event: KeyboardEvent) => {
  if (!props.enableSelection) return

  switch (event.key) {
    case 'Escape':
      clearSelection()
      break
    case 'a':
      if (event.ctrlKey || event.metaKey) {
        event.preventDefault()
        selection.value = { start: 0, end: props.items.length - 1 }
        emit('selection-change', selection.value)
      }
      break
  }
}

// ResizeObserver로 컨테이너 크기 변화 감지
let resizeObserver: ResizeObserver | null = null

// 생명주기
onMounted(() => {
  updateContainerHeight()

  // ResizeObserver 설정
  if (scrollerRef.value && window.ResizeObserver) {
    resizeObserver = new ResizeObserver(() => {
      updateContainerHeight()
    })
    resizeObserver.observe(scrollerRef.value)
  }

  // 키보드 이벤트 리스너 등록
  document.addEventListener('keydown', onKeyDown)
  document.addEventListener('mouseup', endSelection)
})

onUnmounted(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
  }
  document.removeEventListener('keydown', onKeyDown)
  document.removeEventListener('mouseup', endSelection)
})

// 아이템 추가 감지 및 자동 스크롤
let previousItemCount = 0
watch(() => props.items.length, (newLength) => {
  const itemsAdded = newLength - previousItemCount
  previousItemCount = newLength

  if (itemsAdded > 0) {
    // 새 항목이 추가된 경우
    if (props.autoScroll && isAtBottom.value && !isUserScrolling.value) {
      // 자동 스크롤이 활성화되고 맨 아래에 있으면 스크롤
      nextTick(() => {
        scrollToBottom()
      })
    } else if (!isAtBottom.value) {
      // 맨 아래에 있지 않으면 새 항목 수 증가
      newItemsCount.value += itemsAdded
    }
  }
}, { immediate: true })

// 노출할 메서드들
defineExpose({
  scrollToBottom,
  scrollToIndex,
  clearSelection,
  getSelection: () => selection.value,
  getVisibleRange: () => ({
    start: visibleStartIndex.value,
    end: visibleEndIndex.value,
  }),
})
</script>

<style scoped>
.virtual-scroller {
  position: relative;
  background-color: var(--terminal-bg, #1a1a1a);
  color: var(--terminal-fg, #e5e5e5);
  font-family: 'Fira Code', 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  line-height: 1.4;
}

.virtual-item {
  display: flex;
  align-items: center;
  padding: 0 8px;
  box-sizing: border-box;
  user-select: text;
  cursor: text;
  white-space: pre-wrap;
  word-break: break-all;
}

.virtual-item--selected {
  background-color: rgba(64, 128, 255, 0.2);
}

.virtual-item:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.scroll-indicator {
  position: absolute;
  bottom: 16px;
  right: 16px;
  background: rgba(0, 0, 0, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 6px;
  padding: 8px 12px;
  color: #e5e5e5;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  transition: all 0.2s;
  backdrop-filter: blur(4px);
  z-index: 10;
}

.scroll-indicator:hover {
  background: rgba(0, 0, 0, 0.9);
  border-color: rgba(64, 128, 255, 0.5);
  transform: translateY(-1px);
}

.scroll-indicator svg {
  opacity: 0.8;
}

/* 스크롤바 스타일링 */
.virtual-scroller::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.virtual-scroller::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
}

.virtual-scroller::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 4px;
  transition: background 0.2s;
}

.virtual-scroller::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

/* Firefox */
.virtual-scroller {
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.2) rgba(255, 255, 255, 0.05);
}

/* 반응형 대응 */
@media (max-width: 768px) {
  .virtual-item {
    padding: 0 4px;
    font-size: 12px;
  }

  .scroll-indicator {
    bottom: 8px;
    right: 8px;
    padding: 6px 8px;
    font-size: 11px;
  }
}
</style>