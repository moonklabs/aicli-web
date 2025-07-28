/**
 * 테마 관리 컴포저블
 */
import { computed, onMounted, readonly, ref, watch } from 'vue'
import type { AccessibilitySettings, AccessibilityTheme, FontSizePreference, MotionPreference, ThemeMode } from '@/types/ui'

// 로컬 스토리지 키
const THEME_STORAGE_KEY = 'aicli-theme-mode'
const ACCESSIBILITY_STORAGE_KEY = 'aicli-accessibility-settings'

// 전역 테마 상태
const themeMode = ref<ThemeMode>('auto')
const isDark = ref(false)

// 접근성 설정 상태
const accessibilitySettings = ref<AccessibilitySettings>({
  // 시각적 접근성
  highContrast: false,
  reducedMotion: false,
  reducedTransparency: false,
  forceFocusVisible: false,
  colorBlindnessFilter: 'default',
  fontSize: 'medium',

  // 키보드 네비게이션
  keyboardNavigation: true,
  skipLinks: true,
  tabTrapEnabled: true,

  // 스크린 리더
  announcePageChanges: true,
  announceFormErrors: true,
  announceLiveRegions: true,

  // 타이밍 설정
  extendedTimeouts: false,
  pauseAnimations: false,
})

/**
 * 시스템 다크 모드 감지
 */
const getSystemTheme = (): 'light' | 'dark' => {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

/**
 * 로컬 스토리지에서 테마 설정 로드
 */
const loadThemeFromStorage = (): ThemeMode => {
  if (typeof window === 'undefined') return 'auto'

  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY)
    if (stored && ['light', 'dark', 'auto'].includes(stored)) {
      return stored as ThemeMode
    }
  } catch (error) {
    console.warn('Failed to load theme from localStorage:', error)
  }

  return 'auto'
}

/**
 * 로컬 스토리지에 테마 설정 저장
 */
const saveThemeToStorage = (mode: ThemeMode): void => {
  if (typeof window === 'undefined') return

  try {
    localStorage.setItem(THEME_STORAGE_KEY, mode)
  } catch (error) {
    console.warn('Failed to save theme to localStorage:', error)
  }
}

/**
 * 접근성 설정 로드
 */
const loadAccessibilitySettings = (): AccessibilitySettings => {
  if (typeof window === 'undefined') return accessibilitySettings.value

  try {
    const stored = localStorage.getItem(ACCESSIBILITY_STORAGE_KEY)
    if (stored) {
      const parsed = JSON.parse(stored)
      return { ...accessibilitySettings.value, ...parsed }
    }
  } catch (error) {
    console.warn('Failed to load accessibility settings from localStorage:', error)
  }

  return accessibilitySettings.value
}

/**
 * 접근성 설정 저장
 */
const saveAccessibilitySettings = (settings: AccessibilitySettings): void => {
  if (typeof window === 'undefined') return

  try {
    localStorage.setItem(ACCESSIBILITY_STORAGE_KEY, JSON.stringify(settings))
  } catch (error) {
    console.warn('Failed to save accessibility settings to localStorage:', error)
  }
}

/**
 * 시스템 접근성 설정 감지
 */
const detectSystemAccessibilitySettings = (): Partial<AccessibilitySettings> => {
  if (typeof window === 'undefined') return {}

  const settings: Partial<AccessibilitySettings> = {}

  // 고대비 모드 감지
  if (window.matchMedia('(prefers-contrast: high)').matches) {
    settings.highContrast = true
  }

  // 모션 감소 감지
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    settings.reducedMotion = true
    settings.pauseAnimations = true
  }

  // 투명도 감소 감지
  if (window.matchMedia('(prefers-reduced-transparency: reduce)').matches) {
    settings.reducedTransparency = true
  }

  return settings
}

/**
 * DOM에 테마 적용
 */
const applyThemeToDOM = (dark: boolean): void => {
  if (typeof document === 'undefined') return

  const root = document.documentElement
  const theme = dark ? 'dark' : 'light'

  root.setAttribute('data-theme', theme)
  root.style.colorScheme = theme

  // 메타 테마 컬러 업데이트 (모바일 브라우저용)
  const metaThemeColor = document.querySelector('meta[name="theme-color"]')
  if (metaThemeColor) {
    metaThemeColor.setAttribute('content', dark ? '#1a1a1a' : '#ffffff')
  }
}

/**
 * DOM에 접근성 설정 적용
 */
const applyAccessibilitySettingsToDOM = (settings: AccessibilitySettings): void => {
  if (typeof document === 'undefined') return

  const root = document.documentElement

  // 접근성 속성 설정
  root.setAttribute('data-accessibility-theme', settings.colorBlindnessFilter)
  root.setAttribute('data-motion-preference', settings.reducedMotion ? 'reduce' : 'auto')
  root.setAttribute('data-reduced-transparency', settings.reducedTransparency.toString())
  root.setAttribute('data-force-focus-visible', settings.forceFocusVisible.toString())
  root.setAttribute('data-font-size', settings.fontSize)

  // 고대비 모드
  if (settings.highContrast) {
    root.setAttribute('data-accessibility-theme', 'high-contrast')
  }
}

/**
 * 실제 테마 계산 (auto 모드 처리)
 */
const resolvedTheme = computed(() => {
  if (themeMode.value === 'auto') {
    return getSystemTheme()
  }
  return themeMode.value
})

/**
 * 테마 관리 컴포저블
 */
export function useTheme() {
  /**
   * 테마 모드 설정
   */
  const setThemeMode = (mode: ThemeMode): void => {
    themeMode.value = mode
    saveThemeToStorage(mode)
  }

  /**
   * 테마 토글 (라이트 <-> 다크)
   */
  const toggleTheme = (): void => {
    const newMode = resolvedTheme.value === 'dark' ? 'light' : 'dark'
    setThemeMode(newMode)
  }

  /**
   * 접근성 설정 업데이트
   */
  const updateAccessibilitySettings = (newSettings: Partial<AccessibilitySettings>): void => {
    accessibilitySettings.value = { ...accessibilitySettings.value, ...newSettings }
    saveAccessibilitySettings(accessibilitySettings.value)
    applyAccessibilitySettingsToDOM(accessibilitySettings.value)
  }

  /**
   * 고대비 모드 토글
   */
  const toggleHighContrast = (): void => {
    updateAccessibilitySettings({ highContrast: !accessibilitySettings.value.highContrast })
  }

  /**
   * 모션 감소 토글
   */
  const toggleReducedMotion = (): void => {
    updateAccessibilitySettings({
      reducedMotion: !accessibilitySettings.value.reducedMotion,
      pauseAnimations: !accessibilitySettings.value.reducedMotion,
    })
  }

  /**
   * 색맹 필터 설정
   */
  const setColorBlindnessFilter = (filter: AccessibilityTheme): void => {
    updateAccessibilitySettings({ colorBlindnessFilter: filter })
  }

  /**
   * 폰트 크기 설정
   */
  const setFontSize = (size: FontSizePreference): void => {
    updateAccessibilitySettings({ fontSize: size })
  }

  /**
   * 시스템 테마 변경 감지
   */
  const watchSystemTheme = (): void => {
    if (typeof window === 'undefined') return

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

    const handleChange = (e: MediaQueryListEvent): void => {
      if (themeMode.value === 'auto') {
        isDark.value = e.matches
      }
    }

    // 초기 설정
    if (themeMode.value === 'auto') {
      isDark.value = mediaQuery.matches
    }

    // 리스너 등록
    mediaQuery.addEventListener('change', handleChange)

    // 정리 함수 반환
    return () => {
      mediaQuery.removeEventListener('change', handleChange)
    }
  }

  /**
   * 초기화
   */
  const initTheme = (): void => {
    // 저장된 테마 설정 로드
    themeMode.value = loadThemeFromStorage()
    accessibilitySettings.value = loadAccessibilitySettings()

    // 시스템 접근성 설정 감지 및 마이그레이션
    const systemSettings = detectSystemAccessibilitySettings()
    if (Object.keys(systemSettings).length > 0) {
      accessibilitySettings.value = { ...accessibilitySettings.value, ...systemSettings }
    }

    // 초기 다크 모드 상태 설정
    isDark.value = resolvedTheme.value === 'dark'

    // DOM에 테마 및 접근성 설정 적용
    applyThemeToDOM(isDark.value)
    applyAccessibilitySettingsToDOM(accessibilitySettings.value)

    // 시스템 테마 변경 감지 시작
    watchSystemTheme()
  }

  /**
   * 테마 변경 감지 및 적용
   */
  watch(
    () => resolvedTheme.value,
    (newTheme) => {
      isDark.value = newTheme === 'dark'
      applyThemeToDOM(isDark.value)
    },
    { immediate: true },
  )

  /**
   * 테마 모드별 CSS 클래스 계산
   */
  const themeClasses = computed(() => ({
    'theme-light': resolvedTheme.value === 'light',
    'theme-dark': resolvedTheme.value === 'dark',
    'theme-auto': themeMode.value === 'auto',
  }))

  /**
   * 현재 테마 색상 값 가져오기
   */
  const getThemeColor = (colorVar: string): string => {
    if (typeof document === 'undefined') return ''

    return getComputedStyle(document.documentElement)
      .getPropertyValue(colorVar)
      .trim()
  }

  /**
   * 테마 색상 동적 설정
   */
  const setThemeColor = (colorVar: string, value: string): void => {
    if (typeof document === 'undefined') return

    document.documentElement.style.setProperty(colorVar, value)
  }

  /**
   * 컴포넌트가 마운트될 때 초기화
   */
  onMounted(() => {
    initTheme()
  })

  return {
    // 상태
    themeMode: readonly(themeMode),
    isDark: readonly(isDark),
    resolvedTheme,
    themeClasses,
    accessibilitySettings: readonly(accessibilitySettings),

    // 테마 메서드
    setThemeMode,
    toggleTheme,
    initTheme,
    getThemeColor,
    setThemeColor,

    // 접근성 메서드
    updateAccessibilitySettings,
    toggleHighContrast,
    toggleReducedMotion,
    setColorBlindnessFilter,
    setFontSize,

    // 유틸리티
    isLight: computed(() => !isDark.value),
    themeIcon: computed(() => isDark.value ? '🌙' : '☀️'),
    themeLabel: computed(() => {
      switch (themeMode.value) {
        case 'light': return '라이트 모드'
        case 'dark': return '다크 모드'
        case 'auto': return '시스템 설정'
        default: return '알 수 없음'
      }
    }),

    // 접근성 상태 컴퓨티드
    isHighContrast: computed(() => accessibilitySettings.value.highContrast),
    isReducedMotion: computed(() => accessibilitySettings.value.reducedMotion),
    currentFontSize: computed(() => accessibilitySettings.value.fontSize),
    colorBlindnessFilter: computed(() => accessibilitySettings.value.colorBlindnessFilter),
  }
}

/**
 * 전역 테마 상태 (싱글톤)
 */
export const globalTheme = {
  mode: themeMode,
  isDark,
  resolvedTheme,
  accessibilitySettings,
}