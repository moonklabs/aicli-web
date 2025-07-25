/**
 * 테마 관리 컴포저블
 */
import { ref, computed, watch, onMounted } from 'vue';
import type { ThemeMode } from '@/types/ui';

// 로컬 스토리지 키
const THEME_STORAGE_KEY = 'aicli-theme-mode';

// 전역 테마 상태
const themeMode = ref<ThemeMode>('auto');
const isDark = ref(false);

/**
 * 시스템 다크 모드 감지
 */
const getSystemTheme = (): 'light' | 'dark' => {
  if (typeof window === 'undefined') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
};

/**
 * 로컬 스토리지에서 테마 설정 로드
 */
const loadThemeFromStorage = (): ThemeMode => {
  if (typeof window === 'undefined') return 'auto';
  
  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY);
    if (stored && ['light', 'dark', 'auto'].includes(stored)) {
      return stored as ThemeMode;
    }
  } catch (error) {
    console.warn('Failed to load theme from localStorage:', error);
  }
  
  return 'auto';
};

/**
 * 로컬 스토리지에 테마 설정 저장
 */
const saveThemeToStorage = (mode: ThemeMode): void => {
  if (typeof window === 'undefined') return;
  
  try {
    localStorage.setItem(THEME_STORAGE_KEY, mode);
  } catch (error) {
    console.warn('Failed to save theme to localStorage:', error);
  }
};

/**
 * DOM에 테마 적용
 */
const applyThemeToDOM = (dark: boolean): void => {
  if (typeof document === 'undefined') return;
  
  const root = document.documentElement;
  const theme = dark ? 'dark' : 'light';
  
  root.setAttribute('data-theme', theme);
  root.style.colorScheme = theme;
  
  // 메타 테마 컬러 업데이트 (모바일 브라우저용)
  const metaThemeColor = document.querySelector('meta[name="theme-color"]');
  if (metaThemeColor) {
    metaThemeColor.setAttribute('content', dark ? '#1a1a1a' : '#ffffff');
  }
};

/**
 * 실제 테마 계산 (auto 모드 처리)
 */
const resolvedTheme = computed(() => {
  if (themeMode.value === 'auto') {
    return getSystemTheme();
  }
  return themeMode.value;
});

/**
 * 테마 관리 컴포저블
 */
export function useTheme() {
  /**
   * 테마 모드 설정
   */
  const setThemeMode = (mode: ThemeMode): void => {
    themeMode.value = mode;
    saveThemeToStorage(mode);
  };

  /**
   * 테마 토글 (라이트 <-> 다크)
   */
  const toggleTheme = (): void => {
    const newMode = resolvedTheme.value === 'dark' ? 'light' : 'dark';
    setThemeMode(newMode);
  };

  /**
   * 시스템 테마 변경 감지
   */
  const watchSystemTheme = (): void => {
    if (typeof window === 'undefined') return;
    
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    
    const handleChange = (e: MediaQueryListEvent): void => {
      if (themeMode.value === 'auto') {
        isDark.value = e.matches;
      }
    };
    
    // 초기 설정
    if (themeMode.value === 'auto') {
      isDark.value = mediaQuery.matches;
    }
    
    // 리스너 등록
    mediaQuery.addEventListener('change', handleChange);
    
    // 정리 함수 반환
    return () => {
      mediaQuery.removeEventListener('change', handleChange);
    };
  };

  /**
   * 초기화
   */
  const initTheme = (): void => {
    // 저장된 테마 설정 로드
    themeMode.value = loadThemeFromStorage();
    
    // 초기 다크 모드 상태 설정
    isDark.value = resolvedTheme.value === 'dark';
    
    // DOM에 테마 적용
    applyThemeToDOM(isDark.value);
    
    // 시스템 테마 변경 감지 시작
    watchSystemTheme();
  };

  /**
   * 테마 변경 감지 및 적용
   */
  watch(
    () => resolvedTheme.value,
    (newTheme) => {
      isDark.value = newTheme === 'dark';
      applyThemeToDOM(isDark.value);
    },
    { immediate: true }
  );

  /**
   * 테마 모드별 CSS 클래스 계산
   */
  const themeClasses = computed(() => ({
    'theme-light': resolvedTheme.value === 'light',
    'theme-dark': resolvedTheme.value === 'dark',
    'theme-auto': themeMode.value === 'auto'
  }));

  /**
   * 현재 테마 색상 값 가져오기
   */
  const getThemeColor = (colorVar: string): string => {
    if (typeof document === 'undefined') return '';
    
    return getComputedStyle(document.documentElement)
      .getPropertyValue(colorVar)
      .trim();
  };

  /**
   * 테마 색상 동적 설정
   */
  const setThemeColor = (colorVar: string, value: string): void => {
    if (typeof document === 'undefined') return;
    
    document.documentElement.style.setProperty(colorVar, value);
  };

  /**
   * 컴포넌트가 마운트될 때 초기화
   */
  onMounted(() => {
    initTheme();
  });

  return {
    // 상태
    themeMode: readonly(themeMode),
    isDark: readonly(isDark),
    resolvedTheme,
    themeClasses,
    
    // 메서드
    setThemeMode,
    toggleTheme,
    initTheme,
    getThemeColor,
    setThemeColor,
    
    // 유틸리티
    isLight: computed(() => !isDark.value),
    themeIcon: computed(() => isDark.value ? '🌙' : '☀️'),
    themeLabel: computed(() => {
      switch (themeMode.value) {
        case 'light': return '라이트 모드';
        case 'dark': return '다크 모드';
        case 'auto': return '시스템 설정';
        default: return '알 수 없음';
      }
    })
  };
}

/**
 * 전역 테마 상태 (싱글톤)
 */
export const globalTheme = {
  mode: themeMode,
  isDark,
  resolvedTheme
};