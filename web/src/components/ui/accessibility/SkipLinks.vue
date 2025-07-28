<template>
  <nav class="skip-links" aria-label="스킵 링크">
    <ul class="skip-links-list" role="list">
      <li v-for="link in skipLinks" :key="link.id" role="listitem">
        <a
          :href="`#${link.target}`"
          class="skip-link"
          :class="skipLinkClasses"
          @click="handleSkipLinkClick(link)"
          @keydown="handleSkipLinkKeydown"
        >
          {{ link.label }}
        </a>
      </li>
    </ul>
  </nav>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useAccessibilityNavigation } from '@/composables/useAccessibilityNavigation'
import { useAriaLive } from '@/composables/useAriaLive'
import { useTheme } from '@/composables/useTheme'

interface SkipLink {
  id: string
  target: string
  label: string
  description?: string
}

interface Props {
  customLinks?: SkipLink[]
  autoDetect?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  customLinks: () => [],
  autoDetect: true,
})

// 컴포저블
const { skipToTarget, registerSkipTarget } = useAccessibilityNavigation()
const { announce } = useAriaLive('skip-links')
const { accessibilitySettings } = useTheme()

// 기본 스킵 링크
const defaultSkipLinks: SkipLink[] = [
  {
    id: 'main-content',
    target: 'main-content',
    label: '본문으로 바로가기',
    description: '페이지의 주요 내용으로 이동합니다',
  },
  {
    id: 'navigation',
    target: 'navigation',
    label: '네비게이션으로 바로가기',
    description: '사이트 메뉴로 이동합니다',
  },
  {
    id: 'search',
    target: 'search',
    label: '검색으로 바로가기',
    description: '검색 영역으로 이동합니다',
  },
  {
    id: 'footer',
    target: 'footer',
    label: '푸터로 바로가기',
    description: '페이지 하단 정보로 이동합니다',
  },
]

// 컴퓨티드 스킵 링크 목록
const skipLinks = computed(() => {
  const allLinks = [...defaultSkipLinks, ...props.customLinks]

  if (!props.autoDetect) {
    return allLinks
  }

  // DOM에서 실제 존재하는 대상만 필터링
  return allLinks.filter(link => {
    const target = document.getElementById(link.target)
    return target !== null
  })
})

// 스킵 링크 클래스
const skipLinkClasses = computed(() => [
  'skip-link',
  {
    'skip-link--high-contrast': accessibilitySettings.value.highContrast,
    'skip-link--large-font': accessibilitySettings.value.fontSize === 'large' || accessibilitySettings.value.fontSize === 'extra-large',
  },
])

// 이벤트 핸들러
const handleSkipLinkClick = (link: SkipLink): void => {
  const target = document.getElementById(link.target)

  if (!target) {
    announce(`${link.label} 대상을 찾을 수 없습니다.`, { politeness: 'assertive' })
    return
  }

  // 스킵 링크 실행
  skipToTarget(link.target)

  // 접근성 알림
  if (accessibilitySettings.value.announcePageChanges) {
    const description = link.description || `${link.label} 영역`
    announce(`${description}으로 이동했습니다.`, { politeness: 'polite' })
  }
}

const handleSkipLinkKeydown = (event: KeyboardEvent): void => {
  // Enter나 Space로 스킵 링크 활성화
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    ;(event.target as HTMLElement).click()
  }
}

// 동적 스킵 링크 대상 감지
const detectSkipTargets = (): void => {
  const selectors = [
    { id: 'main-content', selectors: ['main', '[role="main"]', '#main', '#content', '.main-content'] },
    { id: 'navigation', selectors: ['nav', '[role="navigation"]', '#nav', '#navigation', '.navigation'] },
    { id: 'search', selectors: ['[role="search"]', '#search', '.search', 'form[action*="search"]'] },
    { id: 'footer', selectors: ['footer', '[role="contentinfo"]', '#footer', '.footer'] },
    { id: 'sidebar', selectors: ['aside', '[role="complementary"]', '#sidebar', '.sidebar'] },
    { id: 'breadcrumb', selectors: ['[role="breadcrumb"]', '.breadcrumb', '.breadcrumbs'] },
  ]

  selectors.forEach(({ id, selectors: selectorList }) => {
    for (const selector of selectorList) {
      const element = document.querySelector(selector) as HTMLElement
      if (element) {
        // ID가 없으면 자동으로 추가
        if (!element.id) {
          element.id = id
        }

        // 스킵 대상으로 등록
        const label = getDefaultLabel(id)
        if (label) {
          registerSkipTarget(element.id, label, element)
        }
        break
      }
    }
  })
}

// 기본 라벨 반환
const getDefaultLabel = (id: string): string => {
  const labelMap: Record<string, string> = {
    'main-content': '본문',
    'navigation': '네비게이션',
    'search': '검색',
    'footer': '푸터',
    'sidebar': '사이드바',
    'breadcrumb': '경로',
  }
  return labelMap[id] || ''
}

// 접근성 향상을 위한 추가 기능
const enhanceAccessibility = (): void => {
  // 스킵 링크 컨테이너에 랜드마크 role 추가
  const skipContainer = document.querySelector('.skip-links')
  if (skipContainer && !skipContainer.getAttribute('role')) {
    skipContainer.setAttribute('role', 'navigation')
    skipContainer.setAttribute('aria-label', '스킵 네비게이션')
  }

  // 스킵 대상에 접근성 속성 추가
  skipLinks.value.forEach(link => {
    const target = document.getElementById(link.target)
    if (target) {
      // tabindex 추가 (포커스 가능하도록)
      if (!target.hasAttribute('tabindex')) {
        target.tabIndex = -1
      }

      // 스크린 리더용 설명 추가
      if (link.description && !target.getAttribute('aria-describedby')) {
        const descId = `skip-desc-${link.id}`
        const descElement = document.createElement('span')
        descElement.id = descId
        descElement.textContent = link.description
        descElement.style.display = 'none'
        target.appendChild(descElement)
        target.setAttribute('aria-describedby', descId)
      }
    }
  })
}

// 생명주기
onMounted(() => {
  // 스킵 대상 자동 감지
  if (props.autoDetect) {
    detectSkipTargets()
  }

  // 스킵 링크 등록
  skipLinks.value.forEach(link => {
    const target = document.getElementById(link.target)
    if (target) {
      registerSkipTarget(link.id, link.label, target)
    }
  })

  // 접근성 향상
  enhanceAccessibility()
})
</script>

<style scoped lang="scss">
.skip-links {
  position: fixed;
  top: 0;
  left: 0;
  z-index: var(--z-toast);
  padding: var(--spacing-sm);

  // 기본적으로 화면 밖에 위치
  transform: translateY(-100%);
}

.skip-links-list {
  display: flex;
  gap: var(--spacing-xs);
  margin: 0;
  padding: 0;
  list-style: none;
}

.skip-link {
  display: inline-block;
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--bg-elevated);
  color: var(--text-primary);
  text-decoration: none;
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  border: 2px solid var(--border-focus);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  white-space: nowrap;
  transition: var(--duration-fast) var(--ease-out);

  // 포커스 시에만 표시
  &:focus,
  &:focus-visible {
    transform: translateY(0);
    outline: none;
  }

  &:hover {
    background: var(--state-hover);
    transform: translateY(0) scale(1.05);
  }

  &:active {
    background: var(--state-active);
    transform: translateY(0) scale(0.98);
  }

  // 고대비 모드 스타일
  &--high-contrast {
    border-width: 3px;
    font-weight: var(--font-bold);
    background: var(--bg-primary);
    color: var(--text-primary);

    &:focus,
    &:focus-visible {
      background: var(--text-primary);
      color: var(--bg-primary);
    }
  }

  // 큰 폰트 모드
  &--large-font {
    font-size: var(--text-base);
    padding: var(--spacing-md) var(--spacing-lg);
  }
}

// 애니메이션 감소 설정
[data-motion-preference="reduce"] {
  .skip-link {
    transition: none !important;

    &:hover {
      transform: translateY(0) !important;
    }

    &:active {
      transform: translateY(0) !important;
    }
  }
}

// 모바일 대응
@media (max-width: 640px) {
  .skip-links-list {
    flex-direction: column;
    gap: var(--spacing-xs);
  }

  .skip-link {
    width: 100%;
    text-align: center;
  }
}

// 인쇄 시 숨김
@media print {
  .skip-links {
    display: none !important;
  }
}

// 키보드 전용 사용자를 위한 표시 (선택적)
.skip-links:focus-within {
  .skip-link {
    transform: translateY(0);
  }
}

// 터치 디바이스에서 실수로 터치하는 것 방지
@media (pointer: coarse) {
  .skip-link {
    &:hover {
      transform: translateY(0); // 스케일 효과 제거
    }
  }
}
</style>