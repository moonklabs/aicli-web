<template>
  <div
    class="touch-list"
    :class="{
      'touch-list--virtual': virtual,
      'touch-list--dense': dense,
      'touch-list--bordered': bordered,
    }"
  >
    <!-- 검색 필터 (선택사항) -->
    <div v-if="searchable" class="touch-list__search">
      <AppInput
        v-model="searchQuery"
        placeholder="검색..."
        clearable
        :prefix-icon="SearchIcon"
        class="touch-list__search-input"
      />
    </div>

    <!-- 가상 스크롤 리스트 -->
    <div
      v-if="virtual"
      ref="virtualContainerRef"
      class="touch-list__virtual-container"
      @scroll="handleVirtualScroll"
    >
      <div
        class="touch-list__virtual-content"
        :style="{ height: `${totalHeight}px` }"
      >
        <div
          class="touch-list__virtual-items"
          :style="{ transform: `translateY(${startOffset}px)` }"
        >
          <TouchListItem
            v-for="(item, index) in visibleItems"
            :key="getItemKey(item, index)"
            :item="item"
            :index="startIndex + index"
            :selected="isSelected(item)"
            :disabled="isDisabled(item)"
            :dense="dense"
            @click="handleItemClick"
            @touch-start="handleItemTouchStart"
            @touch-end="handleItemTouchEnd"
            @long-press="handleItemLongPress"
          >
            <template #default="{ item: slotItem, index: slotIndex }">
              <slot :item="slotItem" :index="slotIndex" />
            </template>

            <template v-if="$slots.actions" #actions="{ item: slotItem }">
              <slot name="actions" :item="slotItem" />
            </template>
          </TouchListItem>
        </div>
      </div>
    </div>

    <!-- 일반 리스트 -->
    <div v-else class="touch-list__container">
      <TouchListItem
        v-for="(item, index) in filteredItems"
        :key="getItemKey(item, index)"
        :item="item"
        :index="index"
        :selected="isSelected(item)"
        :disabled="isDisabled(item)"
        :dense="dense"
        @click="handleItemClick"
        @touch-start="handleItemTouchStart"
        @touch-end="handleItemTouchEnd"
        @long-press="handleItemLongPress"
      >
        <template #default="{ item: slotItem, index: slotIndex }">
          <slot :item="slotItem" :index="slotIndex" />
        </template>

        <template v-if="$slots.actions" #actions="{ item: slotItem }">
          <slot name="actions" :item="slotItem" />
        </template>
      </TouchListItem>
    </div>

    <!-- 로딩 상태 -->
    <div v-if="loading" class="touch-list__loading">
      <AppSpinner size="medium" />
      <span class="touch-list__loading-text">{{ loadingText }}</span>
    </div>

    <!-- 빈 상태 -->
    <div v-else-if="filteredItems.length === 0" class="touch-list__empty">
      <slot name="empty">
        <div class="touch-list__empty-content">
          <svg class="touch-list__empty-icon" viewBox="0 0 24 24" fill="none">
            <path
              d="M20 6L9 17l-5-5"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          <p class="touch-list__empty-title">{{ emptyTitle }}</p>
          <p class="touch-list__empty-message">{{ emptyMessage }}</p>
        </div>
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useTouchGestures } from '@/composables/useTouchGestures'
import AppInput from './AppInput.vue'
import AppSpinner from '../feedback/AppSpinner.vue'
import TouchListItem from './TouchListItem.vue'

interface ListItem {
  id: string | number
  [key: string]: any
}

interface Props {
  items: ListItem[]
  selectedItems?: (string | number)[]
  disabledItems?: (string | number)[]
  virtual?: boolean
  itemHeight?: number
  searchable?: boolean
  dense?: boolean
  bordered?: boolean
  loading?: boolean
  loadingText?: string
  emptyTitle?: string
  emptyMessage?: string
  keyField?: string
  searchFields?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  items: () => [],
  selectedItems: () => [],
  disabledItems: () => [],
  virtual: false,
  itemHeight: 56,
  searchable: false,
  dense: false,
  bordered: false,
  loading: false,
  loadingText: '로딩 중...',
  emptyTitle: '항목이 없습니다',
  emptyMessage: '표시할 항목이 없습니다',
  keyField: 'id',
  searchFields: () => ['title', 'label', 'name'],
})

const emit = defineEmits<{
  'item-click': [item: ListItem, index: number]
  'item-select': [item: ListItem, selected: boolean]
  'item-long-press': [item: ListItem, index: number]
  'scroll': [scrollTop: number]
  'scroll-end': []
}>()

// 검색 쿼리
const searchQuery = ref('')

// 가상 스크롤 관련 상태
const virtualContainerRef = ref<HTMLElement>()
const scrollTop = ref(0)
const containerHeight = ref(0)
const startIndex = ref(0)
const endIndex = ref(0)

// 가상 스크롤 계산
const visibleCount = computed(() => {
  return Math.ceil(containerHeight.value / props.itemHeight) + 2
})

const totalHeight = computed(() => {
  return filteredItems.value.length * props.itemHeight
})

const startOffset = computed(() => {
  return startIndex.value * props.itemHeight
})

const visibleItems = computed(() => {
  return filteredItems.value.slice(startIndex.value, endIndex.value)
})

// 검색 필터링
const filteredItems = computed(() => {
  if (!props.searchable || !searchQuery.value.trim()) {
    return props.items
  }

  const query = searchQuery.value.toLowerCase()
  return props.items.filter(item => {
    return props.searchFields.some(field => {
      const value = item[field]
      return value && String(value).toLowerCase().includes(query)
    })
  })
})

// 아이템 키 생성
const getItemKey = (item: ListItem, index: number) => {
  return item[props.keyField] ?? index
}

// 선택 상태 확인
const isSelected = (item: ListItem) => {
  const key = item[props.keyField]
  return props.selectedItems.includes(key)
}

// 비활성화 상태 확인
const isDisabled = (item: ListItem) => {
  const key = item[props.keyField]
  return props.disabledItems.includes(key)
}

// 가상 스크롤 업데이트
const updateVirtualScroll = () => {
  if (!props.virtual) return

  const container = virtualContainerRef.value
  if (!container) return

  scrollTop.value = container.scrollTop
  containerHeight.value = container.clientHeight

  startIndex.value = Math.floor(scrollTop.value / props.itemHeight)
  endIndex.value = Math.min(
    startIndex.value + visibleCount.value,
    filteredItems.value.length,
  )

  emit('scroll', scrollTop.value)

  // 스크롤 끝 감지
  const isAtEnd = container.scrollTop + container.clientHeight >= container.scrollHeight - 10
  if (isAtEnd) {
    emit('scroll-end')
  }
}

// 가상 스크롤 핸들러
const handleVirtualScroll = () => {
  updateVirtualScroll()
}

// 아이템 이벤트 핸들러
const handleItemClick = (item: ListItem, index: number) => {
  if (isDisabled(item)) return

  emit('item-click', item, index)
}

const handleItemTouchStart = (item: ListItem, index: number) => {
  // 터치 시작 시 햅틱 피드백
  if ('vibrate' in navigator) {
    navigator.vibrate(10)
  }
}

const handleItemTouchEnd = (item: ListItem, index: number) => {
  // 터치 종료 처리
}

const handleItemLongPress = (item: ListItem, index: number) => {
  if (isDisabled(item)) return

  // 롱 프레스 햅틱 피드백
  if ('vibrate' in navigator) {
    navigator.vibrate([50, 50, 50])
  }

  emit('item-long-press', item, index)
}

// 스크롤 위치 복원
const scrollToIndex = (index: number) => {
  if (!props.virtual) return

  const container = virtualContainerRef.value
  if (!container) return

  const scrollPosition = index * props.itemHeight
  container.scrollTop = scrollPosition
}

// 스크롤 맨 위로
const scrollToTop = () => {
  const container = virtualContainerRef.value
  if (container) {
    container.scrollTop = 0
  }
}

// 스크롤 맨 아래로
const scrollToBottom = () => {
  const container = virtualContainerRef.value
  if (container) {
    container.scrollTop = container.scrollHeight
  }
}

// 컴포넌트 마운트 시 가상 스크롤 초기화
onMounted(() => {
  if (props.virtual) {
    nextTick(() => {
      updateVirtualScroll()
    })

    // 리사이즈 이벤트 리스너
    const handleResize = () => {
      updateVirtualScroll()
    }
    window.addEventListener('resize', handleResize)

    onBeforeUnmount(() => {
      window.removeEventListener('resize', handleResize)
    })
  }
})

// 아이템 변경 시 가상 스크롤 업데이트
watch(
  () => [filteredItems.value.length, props.itemHeight],
  () => {
    if (props.virtual) {
      nextTick(() => {
        updateVirtualScroll()
      })
    }
  },
)

// 공개 메서드
defineExpose({
  scrollToIndex,
  scrollToTop,
  scrollToBottom,
  updateVirtualScroll,
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.touch-list {
  @include card-base;
  overflow: hidden;

  &--bordered {
    border: 1px solid map-get($gray-colors, 200);

    .dark & {
      border-color: $dark-bg-tertiary;
    }
  }

  &--dense {
    .touch-list-item {
      min-height: 40px;
    }
  }

  &__search {
    padding: $spacing-3;
    border-bottom: 1px solid map-get($gray-colors, 200);
    background: map-get($gray-colors, 50);

    .dark & {
      border-bottom-color: $dark-bg-tertiary;
      background: $dark-bg-tertiary;
    }
  }

  &__search-input {
    width: 100%;
  }

  &__container {
    max-height: 400px;
    overflow-y: auto;
    @include smooth-scroll;
    @include scrollbar-thin;
  }

  &__virtual-container {
    height: 400px;
    overflow-y: auto;
    @include smooth-scroll;
    @include scrollbar-thin;
  }

  &__virtual-content {
    position: relative;
  }

  &__virtual-items {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
  }

  &__loading {
    @include flex-center;
    flex-direction: column;
    gap: $spacing-3;
    padding: $spacing-8;
    color: $light-text-secondary;

    .dark & {
      color: $dark-text-secondary;
    }
  }

  &__loading-text {
    font-size: $font-size-sm;
  }

  &__empty {
    padding: $spacing-8;
    text-align: center;
  }

  &__empty-content {
    @include flex-column-center;
    gap: $spacing-4;
    max-width: 300px;
    margin: 0 auto;
  }

  &__empty-icon {
    width: 48px;
    height: 48px;
    color: map-get($gray-colors, 400);

    .dark & {
      color: $dark-text-tertiary;
    }
  }

  &__empty-title {
    font-size: $font-size-lg;
    font-weight: $font-weight-semibold;
    color: $light-text-primary;
    margin: 0;

    .dark & {
      color: $dark-text-primary;
    }
  }

  &__empty-message {
    font-size: $font-size-sm;
    color: $light-text-secondary;
    margin: 0;

    .dark & {
      color: $dark-text-secondary;
    }
  }
}

// 모바일 최적화
@include mobile {
  .touch-list {
    &__container,
    &__virtual-container {
      max-height: 60vh;
    }

    &__search {
      padding: $spacing-4;
    }

    &__loading,
    &__empty {
      padding: $spacing-6;
    }
  }
}
</style>