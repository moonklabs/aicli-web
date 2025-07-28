<template>
  <div
    ref="itemRef"
    class="touch-list-item"
    :class="{
      'touch-list-item--selected': selected,
      'touch-list-item--disabled': disabled,
      'touch-list-item--dense': dense,
      'touch-list-item--pressed': isPressed,
    }"
    role="button"
    :tabindex="disabled ? -1 : 0"
    :aria-selected="selected"
    :aria-disabled="disabled"
    @click="handleClick"
    @keydown="handleKeydown"
    @focus="handleFocus"
    @blur="handleBlur"
  >
    <!-- 선택 체크박스 (다중 선택 모드) -->
    <div v-if="showCheckbox" class="touch-list-item__checkbox">
      <input
        type="checkbox"
        :checked="selected"
        :disabled="disabled"
        @change="handleCheckboxChange"
        @click.stop
      />
    </div>

    <!-- 아이템 아바타/아이콘 -->
    <div v-if="$slots.avatar || avatar" class="touch-list-item__avatar">
      <slot name="avatar">
        <img
          v-if="typeof avatar === 'string'"
          :src="avatar"
          :alt="item.title || item.label"
          class="touch-list-item__avatar-img"
        />
        <component
          v-else-if="avatar"
          :is="avatar"
          class="touch-list-item__avatar-icon"
        />
      </slot>
    </div>

    <!-- 메인 콘텐츠 -->
    <div class="touch-list-item__content">
      <slot :item="item" :index="index">
        <div class="touch-list-item__main">
          <h3 v-if="item.title" class="touch-list-item__title">
            {{ item.title }}
          </h3>
          <h3 v-else-if="item.label" class="touch-list-item__title">
            {{ item.label }}
          </h3>

          <p v-if="item.subtitle" class="touch-list-item__subtitle">
            {{ item.subtitle }}
          </p>
          <p v-else-if="item.description" class="touch-list-item__subtitle">
            {{ item.description }}
          </p>
        </div>

        <!-- 메타 정보 -->
        <div v-if="item.meta || item.time || item.status" class="touch-list-item__meta">
          <span v-if="item.time" class="touch-list-item__time">
            {{ formatTime(item.time) }}
          </span>
          <span
            v-if="item.status"
            class="touch-list-item__status"
            :class="`touch-list-item__status--${item.status}`"
          >
            {{ item.status }}
          </span>
          <span v-if="item.meta" class="touch-list-item__meta-text">
            {{ item.meta }}
          </span>
        </div>
      </slot>
    </div>

    <!-- 액션 버튼들 -->
    <div v-if="$slots.actions || showChevron" class="touch-list-item__actions">
      <slot name="actions" :item="item">
        <svg
          v-if="showChevron"
          class="touch-list-item__chevron"
          viewBox="0 0 16 16"
          fill="currentColor"
        >
          <path
            d="M6.22 3.22a.75.75 0 011.06 0l4.25 4.25a.75.75 0 010 1.06l-4.25 4.25a.75.75 0 01-1.06-1.06L9.94 8 6.22 4.28a.75.75 0 010-1.06z"
          />
        </svg>
      </slot>
    </div>

    <!-- 활성 인디케이터 -->
    <div
      v-if="selected"
      class="touch-list-item__indicator"
    ></div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useTouchGestures } from '@/composables/useTouchGestures'

interface ListItem {
  id: string | number
  title?: string
  label?: string
  subtitle?: string
  description?: string
  meta?: string
  time?: string | Date
  status?: string
  avatar?: string
  [key: string]: any
}

interface Props {
  item: ListItem
  index: number
  selected?: boolean
  disabled?: boolean
  dense?: boolean
  showCheckbox?: boolean
  showChevron?: boolean
  avatar?: string | any
}

const props = withDefaults(defineProps<Props>(), {
  selected: false,
  disabled: false,
  dense: false,
  showCheckbox: false,
  showChevron: true,
})

const emit = defineEmits<{
  click: [item: ListItem, index: number]
  'long-press': [item: ListItem, index: number]
  'touch-start': [item: ListItem, index: number]
  'touch-end': [item: ListItem, index: number]
  select: [item: ListItem, selected: boolean]
}>()

const itemRef = ref<HTMLElement>()
const isPressed = ref(false)
const longPressTimer = ref<NodeJS.Timeout>()

// 터치 제스처 설정
const { on: onGesture } = useTouchGestures(itemRef, {
  enableTap: true,
  enableLongPress: true,
  enablePan: false,
  enablePinch: false,
  longPressThreshold: 500,
})

// 터치 제스처 이벤트
onGesture('tap', () => {
  if (!props.disabled) {
    handleClick()
  }
})

onGesture('longpress', () => {
  if (!props.disabled) {
    handleLongPress()
  }
})

// 시간 포맷팅
const formatTime = (time: string | Date) => {
  if (!time) return ''

  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  // 1분 미만
  if (diff < 60000) {
    return '방금 전'
  }

  // 1시간 미만
  if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    return `${minutes}분 전`
  }

  // 24시간 미만
  if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000)
    return `${hours}시간 전`
  }

  // 그 이상은 날짜 표시
  return date.toLocaleDateString('ko-KR', {
    month: 'short',
    day: 'numeric',
  })
}

// 이벤트 핸들러
const handleClick = (event?: Event) => {
  if (props.disabled) return

  emit('click', props.item, props.index)
}

const handleKeydown = (event: KeyboardEvent) => {
  if (props.disabled) return

  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    handleClick()
  }
}

const handleFocus = () => {
  if (props.disabled) return
  // 포커스 시 접근성을 위한 처리
}

const handleBlur = () => {
  isPressed.value = false
  if (longPressTimer.value) {
    clearTimeout(longPressTimer.value)
  }
}

const handleLongPress = () => {
  if (props.disabled) return

  emit('long-press', props.item, props.index)
}

const handleCheckboxChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  emit('select', props.item, target.checked)
}

// 터치 이벤트 처리
const handleTouchStart = () => {
  if (props.disabled) return

  isPressed.value = true
  emit('touch-start', props.item, props.index)

  // 롱 프레스 타이머 시작
  longPressTimer.value = setTimeout(() => {
    handleLongPress()
  }, 500)
}

const handleTouchEnd = () => {
  isPressed.value = false
  emit('touch-end', props.item, props.index)

  if (longPressTimer.value) {
    clearTimeout(longPressTimer.value)
  }
}

// 리스너 정리
onBeforeUnmount(() => {
  if (longPressTimer.value) {
    clearTimeout(longPressTimer.value)
  }
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.touch-list-item {
  @include touch-target(56px);
  @include touch-feedback;
  display: flex;
  align-items: center;
  gap: $spacing-3;
  padding: $spacing-3 $spacing-4;
  border-bottom: 1px solid map-get($gray-colors, 100);
  background: $light-bg-primary;
  cursor: pointer;
  transition: $transition-base;
  position: relative;

  .dark & {
    background: $dark-bg-secondary;
    border-bottom-color: $dark-bg-tertiary;
  }

  &:last-child {
    border-bottom: none;
  }

  &:hover {
    @include no-touch {
      background: map-get($gray-colors, 50);

      .dark & {
        background: lighten($dark-bg-secondary, 5%);
      }
    }
  }

  &:focus {
    outline: none;
    background: map-get($gray-colors, 50);

    .dark & {
      background: lighten($dark-bg-secondary, 5%);
    }
  }

  &--selected {
    background: map-get($primary-colors, 50);

    .dark & {
      background: rgba(map-get($primary-colors, 500), 0.1);
    }

    &:hover {
      background: map-get($primary-colors, 100);

      .dark & {
        background: rgba(map-get($primary-colors, 500), 0.2);
      }
    }
  }

  &--disabled {
    opacity: 0.5;
    cursor: not-allowed;
    pointer-events: none;
  }

  &--dense {
    @include touch-target(40px);
    padding: $spacing-2 $spacing-3;
  }

  &--pressed {
    background: map-get($gray-colors, 100);
    transform: scale(0.98);

    .dark & {
      background: lighten($dark-bg-secondary, 10%);
    }
  }

  &__checkbox {
    display: flex;
    align-items: center;

    input[type="checkbox"] {
      @include touch-target(20px);
      width: 20px;
      height: 20px;
      border-radius: $border-radius-base;
      cursor: pointer;
    }
  }

  &__avatar {
    flex-shrink: 0;
    width: 40px;
    height: 40px;
    border-radius: $border-radius-full;
    overflow: hidden;
    background: map-get($gray-colors, 100);
    @include flex-center;

    .touch-list-item--dense & {
      width: 32px;
      height: 32px;
    }

    .dark & {
      background: $dark-bg-tertiary;
    }
  }

  &__avatar-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  &__avatar-icon {
    width: 20px;
    height: 20px;
    color: map-get($gray-colors, 500);

    .dark & {
      color: $dark-text-secondary;
    }
  }

  &__content {
    flex: 1;
    min-width: 0;
  }

  &__main {
    margin-bottom: $spacing-1;
  }

  &__title {
    font-size: $font-size-base;
    font-weight: $font-weight-medium;
    color: $light-text-primary;
    margin: 0 0 2px 0;
    @include text-ellipsis;

    .touch-list-item--dense & {
      font-size: $font-size-sm;
    }

    .dark & {
      color: $dark-text-primary;
    }
  }

  &__subtitle {
    font-size: $font-size-sm;
    color: $light-text-secondary;
    margin: 0;
    @include text-clamp(2);

    .touch-list-item--dense & {
      font-size: $font-size-xs;
      @include text-clamp(1);
    }

    .dark & {
      color: $dark-text-secondary;
    }
  }

  &__meta {
    display: flex;
    align-items: center;
    gap: $spacing-2;
    margin-top: $spacing-1;
  }

  &__time,
  &__meta-text {
    font-size: $font-size-xs;
    color: $light-text-tertiary;

    .dark & {
      color: $dark-text-tertiary;
    }
  }

  &__status {
    font-size: $font-size-xs;
    font-weight: $font-weight-medium;
    padding: 2px 6px;
    border-radius: $border-radius-full;
    text-transform: uppercase;

    &--active {
      background: lighten($success, 35%);
      color: darken($success, 10%);
    }

    &--pending {
      background: lighten($warning, 35%);
      color: darken($warning, 10%);
    }

    &--error {
      background: lighten($error, 35%);
      color: darken($error, 10%);
    }

    &--inactive {
      background: lighten(map-get($gray-colors, 500), 35%);
      color: darken(map-get($gray-colors, 500), 10%);
    }
  }

  &__actions {
    display: flex;
    align-items: center;
    gap: $spacing-2;
    flex-shrink: 0;
  }

  &__chevron {
    width: 16px;
    height: 16px;
    color: map-get($gray-colors, 400);
    transition: $transition-base;

    .touch-list-item:hover & {
      color: map-get($gray-colors, 600);
    }

    .dark & {
      color: $dark-text-tertiary;
    }

    .dark .touch-list-item:hover & {
      color: $dark-text-secondary;
    }
  }

  &__indicator {
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 3px;
    background: map-get($primary-colors, 500);

    .dark & {
      background: map-get($primary-colors, 400);
    }
  }
}

// 모바일 최적화
@include mobile {
  .touch-list-item {
    @include touch-target(60px);
    padding: $spacing-4;

    &--dense {
      @include touch-target(48px);
      padding: $spacing-3;
    }

    &__avatar {
      width: 48px;
      height: 48px;

      .touch-list-item--dense & {
        width: 36px;
        height: 36px;
      }
    }
  }
}

// 접근성 개선
@include reduce-motion {
  .touch-list-item {
    transition: none;

    &--pressed {
      transform: none;
    }
  }
}

// 포커스 표시
.touch-list-item:focus-visible {
  outline: 2px solid map-get($primary-colors, 500);
  outline-offset: -2px;
}
</style>