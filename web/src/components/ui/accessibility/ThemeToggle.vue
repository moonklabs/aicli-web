<template>
  <div class="theme-toggle-container">
    <!-- 메인 테마 토글 버튼 -->
    <button
      ref="themeToggleButton"
      :class="themeToggleClasses"
      :aria-label="themeToggleLabel"
      :aria-pressed="isDark"
      role="switch"
      @click="handleThemeToggle"
      @keydown="handleKeydown"
    >
      <span class="theme-toggle-icon" :aria-hidden="true">
        {{ themeIcon }}
      </span>
      <span class="theme-toggle-text">
        {{ themeLabel }}
      </span>
    </button>

    <!-- 고급 접근성 설정 메뉴 -->
    <div class="accessibility-menu" v-if="showAdvancedSettings">
      <button
        :class="accessibilityMenuClasses"
        :aria-label="'접근성 설정 메뉴 ' + (showMenu ? '닫기' : '열기')"
        :aria-expanded="showMenu"
        :aria-controls="menuId"
        @click="toggleMenu"
        @keydown="handleMenuKeydown"
      >
        <span class="sr-only">접근성 설정</span>
        <svg class="accessibility-icon" viewBox="0 0 24 24" :aria-hidden="true">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
        </svg>
      </button>

      <!-- 드롭다운 메뉴 -->
      <div
        v-if="showMenu"
        :id="menuId"
        class="accessibility-dropdown"
        role="menu"
        :aria-labelledby="menuButtonId"
      >
        <!-- 테마 모드 선택 -->
        <div class="menu-section" role="group" aria-labelledby="theme-section">
          <h3 id="theme-section" class="menu-section-title">테마 모드</h3>
          <div class="radio-group" role="radiogroup" aria-labelledby="theme-section">
            <label
              v-for="mode in themeModes"
              :key="mode.value"
              class="radio-option"
              :class="{ 'selected': themeMode === mode.value }"
            >
              <input
                type="radio"
                :name="'theme-mode-' + menuId"
                :value="mode.value"
                :checked="themeMode === mode.value"
                @change="setThemeMode(mode.value)"
              />
              <span class="radio-label">{{ mode.label }}</span>
            </label>
          </div>
        </div>

        <!-- 접근성 옵션 -->
        <div class="menu-section" role="group" aria-labelledby="accessibility-section">
          <h3 id="accessibility-section" class="menu-section-title">접근성 설정</h3>

          <!-- 고대비 모드 -->
          <label class="checkbox-option">
            <input
              type="checkbox"
              :checked="isHighContrast"
              @change="toggleHighContrast"
              role="switch"
              :aria-checked="isHighContrast"
            />
            <span class="checkbox-label">고대비 모드</span>
          </label>

          <!-- 모션 감소 -->
          <label class="checkbox-option">
            <input
              type="checkbox"
              :checked="isReducedMotion"
              @change="toggleReducedMotion"
              role="switch"
              :aria-checked="isReducedMotion"
            />
            <span class="checkbox-label">모션 감소</span>
          </label>

          <!-- 투명도 감소 -->
          <label class="checkbox-option">
            <input
              type="checkbox"
              :checked="accessibilitySettings.reducedTransparency"
              @change="updateAccessibilitySettings({ reducedTransparency: !accessibilitySettings.reducedTransparency })"
              role="switch"
              :aria-checked="accessibilitySettings.reducedTransparency"
            />
            <span class="checkbox-label">투명도 감소</span>
          </label>

          <!-- 포커스 강제 표시 -->
          <label class="checkbox-option">
            <input
              type="checkbox"
              :checked="accessibilitySettings.forceFocusVisible"
              @change="updateAccessibilitySettings({ forceFocusVisible: !accessibilitySettings.forceFocusVisible })"
              role="switch"
              :aria-checked="accessibilitySettings.forceFocusVisible"
            />
            <span class="checkbox-label">포커스 강제 표시</span>
          </label>
        </div>

        <!-- 폰트 크기 설정 -->
        <div class="menu-section" role="group" aria-labelledby="font-section">
          <h3 id="font-section" class="menu-section-title">폰트 크기</h3>
          <div class="radio-group" role="radiogroup" aria-labelledby="font-section">
            <label
              v-for="size in fontSizes"
              :key="size.value"
              class="radio-option"
              :class="{ 'selected': currentFontSize === size.value }"
            >
              <input
                type="radio"
                :name="'font-size-' + menuId"
                :value="size.value"
                :checked="currentFontSize === size.value"
                @change="setFontSize(size.value)"
              />
              <span class="radio-label">{{ size.label }}</span>
            </label>
          </div>
        </div>

        <!-- 색맹 필터 -->
        <div class="menu-section" role="group" aria-labelledby="colorblind-section">
          <h3 id="colorblind-section" class="menu-section-title">색맹 필터</h3>
          <div class="radio-group" role="radiogroup" aria-labelledby="colorblind-section">
            <label
              v-for="filter in colorBlindnessFilters"
              :key="filter.value"
              class="radio-option"
              :class="{ 'selected': colorBlindnessFilter === filter.value }"
            >
              <input
                type="radio"
                :name="'colorblind-filter-' + menuId"
                :value="filter.value"
                :checked="colorBlindnessFilter === filter.value"
                @change="setColorBlindnessFilter(filter.value)"
              />
              <span class="radio-label">{{ filter.label }}</span>
            </label>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useTheme } from '@/composables/useTheme'
import { useAriaLive } from '@/composables/useAriaLive'
import type { AccessibilityTheme, FontSizePreference, ThemeMode } from '@/types/ui'

interface Props {
  showAdvancedSettings?: boolean
  compact?: boolean
  showLabels?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showAdvancedSettings: true,
  compact: false,
  showLabels: true,
})

// 테마 관리
const {
  themeMode,
  isDark,
  themeIcon,
  themeLabel,
  isHighContrast,
  isReducedMotion,
  currentFontSize,
  colorBlindnessFilter,
  accessibilitySettings,
  setThemeMode,
  toggleTheme,
  toggleHighContrast,
  toggleReducedMotion,
  setColorBlindnessFilter,
  setFontSize,
  updateAccessibilitySettings,
} = useTheme()

// ARIA Live 알림
const { announceSuccess, announceInfo } = useAriaLive('theme-settings')

// 메뉴 상태
const showMenu = ref(false)
const themeToggleButton = ref<HTMLElement>()
const menuId = computed(() => `accessibility-menu-${Math.random().toString(36).substr(2, 9)}`)
const menuButtonId = computed(() => `accessibility-button-${Math.random().toString(36).substr(2, 9)}`)

// 옵션 정의
const themeModes = [
  { value: 'light' as ThemeMode, label: '라이트 모드' },
  { value: 'dark' as ThemeMode, label: '다크 모드' },
  { value: 'auto' as ThemeMode, label: '시스템 설정' },
]

const fontSizes = [
  { value: 'small' as FontSizePreference, label: '작게' },
  { value: 'medium' as FontSizePreference, label: '보통' },
  { value: 'large' as FontSizePreference, label: '크게' },
  { value: 'extra-large' as FontSizePreference, label: '매우 크게' },
]

const colorBlindnessFilters = [
  { value: 'default' as AccessibilityTheme, label: '기본' },
  { value: 'protanopia' as AccessibilityTheme, label: '적색맹 (Protanopia)' },
  { value: 'deuteranopia' as AccessibilityTheme, label: '녹색맹 (Deuteranopia)' },
  { value: 'tritanopia' as AccessibilityTheme, label: '청색맹 (Tritanopia)' },
  { value: 'monochrome' as AccessibilityTheme, label: '단색 모드' },
]

// 컴퓨티드 클래스
const themeToggleClasses = computed(() => [
  'theme-toggle',
  {
    'theme-toggle--compact': props.compact,
    'theme-toggle--dark': isDark.value,
    'theme-toggle--high-contrast': isHighContrast.value,
  },
])

const accessibilityMenuClasses = computed(() => [
  'accessibility-menu-button',
  {
    'accessibility-menu-button--active': showMenu.value,
    'accessibility-menu-button--high-contrast': isHighContrast.value,
  },
])

const themeToggleLabel = computed(() => {
  const action = isDark.value ? '라이트 모드로 전환' : '다크 모드로 전환'
  return `현재 ${themeLabel.value}. ${action}`
})

// 이벤트 핸들러
const handleThemeToggle = async (): Promise<void> => {
  const previousTheme = themeLabel.value
  toggleTheme()

  await nextTick()
  announceSuccess(`테마가 ${previousTheme}에서 ${themeLabel.value}(으)로 변경되었습니다.`)
}

const toggleMenu = (): void => {
  showMenu.value = !showMenu.value

  if (showMenu.value) {
    announceInfo('접근성 설정 메뉴가 열렸습니다.')
  }
}

const handleKeydown = (event: KeyboardEvent): void => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    handleThemeToggle()
  }
}

const handleMenuKeydown = (event: KeyboardEvent): void => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    toggleMenu()
  } else if (event.key === 'Escape' && showMenu.value) {
    showMenu.value = false
    announceInfo('접근성 설정 메뉴가 닫혔습니다.')
  }
}

// 외부 클릭으로 메뉴 닫기
const handleClickOutside = (event: Event): void => {
  const target = event.target as Element
  if (!target.closest('.accessibility-menu')) {
    showMenu.value = false
  }
}

// 생명주기
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped lang="scss">
.theme-toggle-container {
  position: relative;
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.theme-toggle {
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  cursor: pointer;
  transition: var(--duration-normal) var(--ease-in-out);
  min-height: var(--min-touch-target);

  &:hover {
    background: var(--state-hover);
    border-color: var(--border-secondary);
  }

  &:focus-visible {
    outline: var(--focus-ring-width) var(--focus-ring-style) var(--focus-ring-color);
    outline-offset: var(--focus-ring-offset);
  }

  &:active {
    background: var(--state-active);
  }

  &--compact {
    padding: var(--spacing-xs) var(--spacing-sm);

    .theme-toggle-text {
      @media (max-width: 640px) {
        display: none;
      }
    }
  }

  &--dark {
    background: var(--bg-tertiary);
  }

  &--high-contrast {
    border-width: 2px;
    font-weight: var(--font-semibold);
  }
}

.theme-toggle-icon {
  font-size: var(--text-lg);
  line-height: 1;
}

.theme-toggle-text {
  white-space: nowrap;
}

.accessibility-menu {
  position: relative;
}

.accessibility-menu-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: var(--min-touch-target);
  height: var(--min-touch-target);
  padding: var(--spacing-sm);
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  color: var(--text-secondary);
  cursor: pointer;
  transition: var(--duration-normal) var(--ease-in-out);

  &:hover {
    background: var(--state-hover);
    border-color: var(--border-secondary);
    color: var(--text-primary);
  }

  &:focus-visible {
    outline: var(--focus-ring-width) var(--focus-ring-style) var(--focus-ring-color);
    outline-offset: var(--focus-ring-offset);
  }

  &--active {
    background: var(--bg-accent);
    border-color: var(--border-accent);
    color: var(--text-accent);
  }

  &--high-contrast {
    border-width: 2px;
  }
}

.accessibility-icon {
  width: 20px;
  height: 20px;
  fill: currentColor;
}

.accessibility-dropdown {
  position: absolute;
  top: 100%;
  right: 0;
  z-index: var(--z-dropdown);
  min-width: 280px;
  max-width: 400px;
  margin-top: var(--spacing-xs);
  padding: var(--spacing-md);
  background: var(--bg-elevated);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);

  @media (max-width: 480px) {
    left: 0;
    right: 0;
    min-width: auto;
  }
}

.menu-section {
  & + & {
    margin-top: var(--spacing-lg);
    padding-top: var(--spacing-lg);
    border-top: 1px solid var(--border-primary);
  }
}

.menu-section-title {
  margin: 0 0 var(--spacing-sm) 0;
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--text-secondary);
}

.radio-group,
.checkbox-group {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.radio-option,
.checkbox-option {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-xs) var(--spacing-sm);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: var(--duration-normal) var(--ease-in-out);

  &:hover {
    background: var(--state-hover);
  }

  &.selected {
    background: var(--bg-accent);
    color: var(--text-accent);
  }

  input[type="radio"],
  input[type="checkbox"] {
    margin: 0;
    width: 16px;
    height: 16px;
    accent-color: var(--primary-500);

    &:focus-visible {
      outline: var(--focus-ring-width) var(--focus-ring-style) var(--focus-ring-color);
      outline-offset: var(--focus-ring-offset);
    }
  }
}

.radio-label,
.checkbox-label {
  flex: 1;
  font-size: var(--text-sm);
  line-height: var(--leading-snug);
}

// 스크린 리더 전용 텍스트
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

// 고대비 모드에서 추가 스타일링
[data-accessibility-theme="high-contrast"] {
  .theme-toggle,
  .accessibility-menu-button {
    border-width: 3px;
    font-weight: var(--font-bold);
  }

  .accessibility-dropdown {
    border-width: 2px;
  }

  .radio-option,
  .checkbox-option {
    &.selected {
      border: 2px solid var(--text-accent);
    }
  }
}

// 애니메이션 감소 설정
[data-motion-preference="reduce"] {
  .theme-toggle,
  .accessibility-menu-button,
  .radio-option,
  .checkbox-option {
    transition: none !important;
  }
}
</style>