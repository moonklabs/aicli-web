<template>
  <div
    v-if="isDevelopment && showChecker"
    class="accessibility-checker"
    :class="checkerClasses"
  >
    <!-- 토글 버튼 -->
    <button
      class="checker-toggle"
      :aria-label="isExpanded ? '접근성 검사 도구 접기' : '접근성 검사 도구 펼치기'"
      :aria-expanded="isExpanded"
      @click="toggleExpanded"
    >
      <span class="toggle-icon" :aria-hidden="true">
        {{ isExpanded ? '🔽' : '▶️' }}
      </span>
      <span class="toggle-text">접근성 검사</span>
      <span class="issue-count" v-if="totalIssues > 0">{{ totalIssues }}</span>
    </button>

    <!-- 검사 결과 패널 -->
    <div v-if="isExpanded" class="checker-panel">
      <!-- 검사 컨트롤 -->
      <div class="checker-controls">
        <button
          class="btn btn-primary"
          @click="runAccessibilityCheck"
          :disabled="isChecking"
        >
          {{ isChecking ? '검사 중...' : '접근성 검사 실행' }}
        </button>

        <button class="btn btn-secondary" @click="clearResults">
          결과 지우기
        </button>

        <button class="btn btn-secondary" @click="exportResults">
          결과 내보내기
        </button>
      </div>

      <!-- 검사 결과 요약 -->
      <div v-if="checkResults" class="results-summary">
        <h3>검사 결과 요약</h3>
        <div class="summary-stats">
          <div class="stat-item error">
            <span class="stat-label">오류</span>
            <span class="stat-value">{{ checkResults.errors.length }}</span>
          </div>
          <div class="stat-item warning">
            <span class="stat-label">경고</span>
            <span class="stat-value">{{ checkResults.warnings.length }}</span>
          </div>
          <div class="stat-item info">
            <span class="stat-label">정보</span>
            <span class="stat-value">{{ checkResults.info.length }}</span>
          </div>
          <div class="stat-item success">
            <span class="stat-label">통과</span>
            <span class="stat-value">{{ checkResults.passed.length }}</span>
          </div>
        </div>
      </div>

      <!-- 이슈 목록 -->
      <div v-if="checkResults" class="issues-list">
        <!-- 오류 -->
        <div v-if="checkResults.errors.length > 0" class="issue-section">
          <h4 class="issue-section-title error">오류 ({{ checkResults.errors.length }})</h4>
          <div class="issue-items">
            <div
              v-for="issue in checkResults.errors"
              :key="issue.id"
              class="issue-item error"
              @click="highlightElement(issue.element)"
            >
              <div class="issue-header">
                <span class="issue-rule">{{ issue.rule }}</span>
                <span class="issue-impact">{{ getImpactLabel(issue.impact) }}</span>
              </div>
              <div class="issue-description">{{ issue.description }}</div>
              <div class="issue-help">{{ issue.help }}</div>
              <div class="issue-selector">{{ issue.selector }}</div>
            </div>
          </div>
        </div>

        <!-- 경고 -->
        <div v-if="checkResults.warnings.length > 0" class="issue-section">
          <h4 class="issue-section-title warning">경고 ({{ checkResults.warnings.length }})</h4>
          <div class="issue-items">
            <div
              v-for="issue in checkResults.warnings"
              :key="issue.id"
              class="issue-item warning"
              @click="highlightElement(issue.element)"
            >
              <div class="issue-header">
                <span class="issue-rule">{{ issue.rule }}</span>
                <span class="issue-impact">{{ getImpactLabel(issue.impact) }}</span>
              </div>
              <div class="issue-description">{{ issue.description }}</div>
              <div class="issue-help">{{ issue.help }}</div>
              <div class="issue-selector">{{ issue.selector }}</div>
            </div>
          </div>
        </div>

        <!-- 정보 -->
        <div v-if="checkResults.info.length > 0" class="issue-section">
          <h4 class="issue-section-title info">정보 ({{ checkResults.info.length }})</h4>
          <div class="issue-items">
            <div
              v-for="issue in checkResults.info"
              :key="issue.id"
              class="issue-item info"
              @click="highlightElement(issue.element)"
            >
              <div class="issue-header">
                <span class="issue-rule">{{ issue.rule }}</span>
                <span class="issue-impact">{{ getImpactLabel(issue.impact) }}</span>
              </div>
              <div class="issue-description">{{ issue.description }}</div>
              <div class="issue-help">{{ issue.help }}</div>
              <div class="issue-selector">{{ issue.selector }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 실시간 모니터링 정보 -->
      <div class="live-monitoring" v-if="liveMonitoring">
        <h4>실시간 모니터링</h4>
        <div class="monitoring-stats">
          <div class="monitoring-item">
            <span class="monitoring-label">포커스 이벤트</span>
            <span class="monitoring-value">{{ focusEvents }}</span>
          </div>
          <div class="monitoring-item">
            <span class="monitoring-label">키보드 이벤트</span>
            <span class="monitoring-value">{{ keyboardEvents }}</span>
          </div>
          <div class="monitoring-item">
            <span class="monitoring-label">ARIA 업데이트</span>
            <span class="monitoring-value">{{ ariaUpdates }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'

interface AccessibilityIssue {
  id: string
  rule: string
  impact: 'critical' | 'serious' | 'moderate' | 'minor'
  description: string
  help: string
  selector: string
  element: HTMLElement | null
}

interface CheckResults {
  errors: AccessibilityIssue[]
  warnings: AccessibilityIssue[]
  info: AccessibilityIssue[]
  passed: AccessibilityIssue[]
}

interface Props {
  autoCheck?: boolean
  liveMonitoring?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  autoCheck: false,
  liveMonitoring: true,
})

// 개발 환경 체크
const isDevelopment = computed(() => import.meta.env.DEV)

// 상태
const showChecker = ref(true)
const isExpanded = ref(false)
const isChecking = ref(false)
const checkResults = ref<CheckResults | null>(null)
const highlightedElement = ref<HTMLElement | null>(null)

// 실시간 모니터링 카운터
const focusEvents = ref(0)
const keyboardEvents = ref(0)
const ariaUpdates = ref(0)

// 컴퓨티드
const totalIssues = computed(() => {
  if (!checkResults.value) return 0
  return checkResults.value.errors.length +
         checkResults.value.warnings.length +
         checkResults.value.info.length
})

const checkerClasses = computed(() => [
  'accessibility-checker',
  {
    'accessibility-checker--expanded': isExpanded.value,
    'accessibility-checker--has-issues': totalIssues.value > 0,
  },
])

// 메서드
const toggleExpanded = (): void => {
  isExpanded.value = !isExpanded.value
}

const runAccessibilityCheck = async (): Promise<void> => {
  isChecking.value = true

  try {
    // 기본 접근성 검사 수행
    const results = await performAccessibilityChecks()
    checkResults.value = results
  } catch (error) {
    console.error('접근성 검사 중 오류:', error)
  } finally {
    isChecking.value = false
  }
}

const performAccessibilityChecks = async (): Promise<CheckResults> => {
  const results: CheckResults = {
    errors: [],
    warnings: [],
    info: [],
    passed: [],
  }

  // 기본 접근성 규칙 검사
  const checks = [
    checkMissingAltText,
    checkMissingLabels,
    checkColorContrast,
    checkHeadingStructure,
    checkLandmarks,
    checkFocusManagement,
    checkAriaUsage,
    checkKeyboardAccessibility,
  ]

  for (const check of checks) {
    const issues = await check()
    issues.forEach(issue => {
      switch (issue.impact) {
        case 'critical':
        case 'serious':
          results.errors.push(issue)
          break
        case 'moderate':
          results.warnings.push(issue)
          break
        case 'minor':
          results.info.push(issue)
          break
      }
    })
  }

  return results
}

// 개별 검사 함수들
const checkMissingAltText = async (): Promise<AccessibilityIssue[]> => {
  const issues: AccessibilityIssue[] = []
  const images = document.querySelectorAll('img')

  images.forEach((img, index) => {
    if (!img.alt && !img.getAttribute('aria-label') && !img.getAttribute('aria-labelledby')) {
      issues.push({
        id: `missing-alt-${index}`,
        rule: 'image-alt',
        impact: 'serious',
        description: '이미지에 대체 텍스트가 없습니다.',
        help: 'img 요소에 의미 있는 alt 속성을 추가하세요.',
        selector: getElementSelector(img),
        element: img,
      })
    }
  })

  return issues
}

const checkMissingLabels = async (): Promise<AccessibilityIssue[]> => {
  const issues: AccessibilityIssue[] = []
  const inputs = document.querySelectorAll('input, select, textarea')

  inputs.forEach((input, index) => {
    const hasLabel = input.getAttribute('aria-label') ||
                    input.getAttribute('aria-labelledby') ||
                    document.querySelector(`label[for="${input.id}"]`)

    if (!hasLabel) {
      issues.push({
        id: `missing-label-${index}`,
        rule: 'label',
        impact: 'serious',
        description: '폼 요소에 라벨이 없습니다.',
        help: '모든 폼 요소는 접근 가능한 라벨을 가져야 합니다.',
        selector: getElementSelector(input),
        element: input as HTMLElement,
      })
    }
  })

  return issues
}

const checkColorContrast = async (): Promise<AccessibilityIssue[]> => {
  const issues: AccessibilityIssue[] = []

  // 간단한 색상 대비 검사 (실제로는 더 복잡한 알고리즘 필요)
  const textElements = document.querySelectorAll('p, span, div, h1, h2, h3, h4, h5, h6, a, button')

  textElements.forEach((element, index) => {
    const styles = window.getComputedStyle(element)
    const backgroundColor = styles.backgroundColor
    const color = styles.color

    // 간단한 대비 검사 (실제 구현에서는 WCAG 대비 비율 계산 필요)
    if (backgroundColor === color) {
      issues.push({
        id: `color-contrast-${index}`,
        rule: 'color-contrast',
        impact: 'serious',
        description: '텍스트와 배경색의 대비가 부족합니다.',
        help: 'WCAG AA 기준을 만족하는 색상 대비를 사용하세요.',
        selector: getElementSelector(element),
        element: element as HTMLElement,
      })
    }
  })

  return issues
}

const checkHeadingStructure = async (): Promise<AccessibilityIssue[]> => {
  const issues: AccessibilityIssue[] = []
  const headings = document.querySelectorAll('h1, h2, h3, h4, h5, h6')

  let previousLevel = 0
  headings.forEach((heading, index) => {
    const level = parseInt(heading.tagName.charAt(1))

    if (level - previousLevel > 1) {
      issues.push({
        id: `heading-skip-${index}`,
        rule: 'heading-order',
        impact: 'moderate',
        description: '제목 레벨이 순서대로 배치되지 않았습니다.',
        help: '제목은 순차적으로 배치되어야 합니다 (h1, h2, h3...).',
        selector: getElementSelector(heading),
        element: heading as HTMLElement,
      })
    }

    previousLevel = level
  })

  return issues
}

const checkLandmarks = async (): Promise<AccessibilityIssue[]> => {
  const issues: AccessibilityIssue[] = []

  const hasMain = document.querySelector('main, [role="main"]')
  const hasNav = document.querySelector('nav, [role="navigation"]')

  if (!hasMain) {
    issues.push({
      id: 'missing-main',
      rule: 'landmark-main-is-top-level',
      impact: 'moderate',
      description: 'main 랜드마크가 없습니다.',
      help: '페이지에는 main 요소 또는 role="main"이 있어야 합니다.',
      selector: 'document',
      element: null,
    })
  }

  if (!hasNav) {
    issues.push({
      id: 'missing-nav',
      rule: 'landmark-no-duplicate-main',
      impact: 'minor',
      description: 'navigation 랜드마크가 없습니다.',
      help: '페이지에는 nav 요소 또는 role="navigation"이 있는 것이 좋습니다.',
      selector: 'document',
      element: null,
    })
  }

  return issues
}

const checkFocusManagement = async (): Promise<AccessibilityIssue[]> => {
  const issues: AccessibilityIssue[] = []
  const focusableElements = document.querySelectorAll('a, button, input, select, textarea, [tabindex]:not([tabindex="-1"])')

  focusableElements.forEach((element, index) => {
    const tabIndex = element.getAttribute('tabindex')

    if (tabIndex && parseInt(tabIndex) > 0) {
      issues.push({
        id: `positive-tabindex-${index}`,
        rule: 'tabindex',
        impact: 'moderate',
        description: '양수 tabindex가 사용되었습니다.',
        help: '양수 tabindex 사용을 피하고 자연스러운 탭 순서를 유지하세요.',
        selector: getElementSelector(element),
        element: element as HTMLElement,
      })
    }
  })

  return issues
}

const checkAriaUsage = async (): Promise<AccessibilityIssue[]> => {
  const issues: AccessibilityIssue[] = []
  const elementsWithAria = document.querySelectorAll('[aria-labelledby], [aria-describedby]')

  elementsWithAria.forEach((element, index) => {
    const labelledBy = element.getAttribute('aria-labelledby')
    const describedBy = element.getAttribute('aria-describedby')

    if (labelledBy && !document.getElementById(labelledBy)) {
      issues.push({
        id: `aria-labelledby-missing-${index}`,
        rule: 'aria-valid-attr-value',
        impact: 'serious',
        description: 'aria-labelledby가 참조하는 요소가 존재하지 않습니다.',
        help: 'aria-labelledby는 존재하는 요소의 ID를 참조해야 합니다.',
        selector: getElementSelector(element),
        element: element as HTMLElement,
      })
    }

    if (describedBy && !document.getElementById(describedBy)) {
      issues.push({
        id: `aria-describedby-missing-${index}`,
        rule: 'aria-valid-attr-value',
        impact: 'serious',
        description: 'aria-describedby가 참조하는 요소가 존재하지 않습니다.',
        help: 'aria-describedby는 존재하는 요소의 ID를 참조해야 합니다.',
        selector: getElementSelector(element),
        element: element as HTMLElement,
      })
    }
  })

  return issues
}

const checkKeyboardAccessibility = async (): Promise<AccessibilityIssue[]> => {
  const issues: AccessibilityIssue[] = []
  const interactiveElements = document.querySelectorAll('div[onclick], span[onclick], [role="button"]:not(button)')

  interactiveElements.forEach((element, index) => {
    const isKeyboardAccessible = element.hasAttribute('tabindex') ||
                                element.hasAttribute('onkeydown') ||
                                element.hasAttribute('onkeyup')

    if (!isKeyboardAccessible) {
      issues.push({
        id: `keyboard-access-${index}`,
        rule: 'keyboard',
        impact: 'serious',
        description: '키보드로 접근할 수 없는 인터랙티브 요소입니다.',
        help: 'onclick이 있는 요소는 키보드 이벤트도 처리해야 합니다.',
        selector: getElementSelector(element),
        element: element as HTMLElement,
      })
    }
  })

  return issues
}

const getElementSelector = (element: Element): string => {
  if (element.id) return `#${element.id}`
  if (element.className) return `.${element.className.split(' ')[0]}`
  return element.tagName.toLowerCase()
}

const getImpactLabel = (impact: string): string => {
  const labels: Record<string, string> = {
    critical: '치명적',
    serious: '심각',
    moderate: '보통',
    minor: '경미',
  }
  return labels[impact] || impact
}

const highlightElement = (element: HTMLElement | null): void => {
  // 이전 하이라이트 제거
  if (highlightedElement.value) {
    highlightedElement.value.classList.remove('accessibility-highlight')
  }

  if (element) {
    element.classList.add('accessibility-highlight')
    element.scrollIntoView({ behavior: 'smooth', block: 'center' })
    highlightedElement.value = element

    // 3초 후 하이라이트 제거
    setTimeout(() => {
      element.classList.remove('accessibility-highlight')
      highlightedElement.value = null
    }, 3000)
  }
}

const clearResults = (): void => {
  checkResults.value = null
  if (highlightedElement.value) {
    highlightedElement.value.classList.remove('accessibility-highlight')
    highlightedElement.value = null
  }
}

const exportResults = (): void => {
  if (!checkResults.value) return

  const data = {
    timestamp: new Date().toISOString(),
    url: window.location.href,
    summary: {
      errors: checkResults.value.errors.length,
      warnings: checkResults.value.warnings.length,
      info: checkResults.value.info.length,
      passed: checkResults.value.passed.length,
    },
    issues: checkResults.value,
  }

  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `accessibility-report-${Date.now()}.json`
  link.click()
  URL.revokeObjectURL(url)
}

// 실시간 모니터링
const setupLiveMonitoring = (): void => {
  if (!props.liveMonitoring) return

  // 포커스 이벤트 모니터링
  const handleFocus = (): void => {
    focusEvents.value++
  }

  // 키보드 이벤트 모니터링
  const handleKeyboard = (): void => {
    keyboardEvents.value++
  }

  // ARIA 속성 변경 모니터링
  const observer = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      if (mutation.type === 'attributes' && mutation.attributeName?.startsWith('aria-')) {
        ariaUpdates.value++
      }
    })
  })

  document.addEventListener('focus', handleFocus, true)
  document.addEventListener('keydown', handleKeyboard)
  observer.observe(document.body, {
    attributes: true,
    attributeFilter: ['aria-live', 'aria-label', 'aria-expanded', 'aria-selected'],
    subtree: true,
  })

  return () => {
    document.removeEventListener('focus', handleFocus, true)
    document.removeEventListener('keydown', handleKeyboard)
    observer.disconnect()
  }
}

// 생명주기
let cleanupMonitoring: (() => void) | undefined

onMounted(() => {
  cleanupMonitoring = setupLiveMonitoring()

  if (props.autoCheck) {
    setTimeout(runAccessibilityCheck, 1000)
  }
})

onUnmounted(() => {
  if (cleanupMonitoring) {
    cleanupMonitoring()
  }
})
</script>

<style scoped lang="scss">
.accessibility-checker {
  position: fixed;
  bottom: var(--spacing-lg);
  right: var(--spacing-lg);
  z-index: var(--z-modal);
  max-width: 400px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  font-size: var(--text-sm);

  &--has-issues {
    border-color: var(--error-500);
  }
}

.checker-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-md);
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  cursor: pointer;
  border-radius: var(--radius-xl);
  transition: var(--duration-normal) var(--ease-in-out);

  &:hover {
    background: var(--state-hover);
  }
}

.toggle-icon {
  font-size: var(--text-base);
}

.toggle-text {
  flex: 1;
  text-align: left;
}

.issue-count {
  background: var(--error-500);
  color: white;
  padding: 2px 6px;
  border-radius: var(--radius-full);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  min-width: 20px;
  text-align: center;
}

.checker-panel {
  padding: var(--spacing-md);
  border-top: 1px solid var(--border-primary);
  max-height: 70vh;
  overflow-y: auto;
}

.checker-controls {
  display: flex;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-md);
  flex-wrap: wrap;
}

.btn {
  padding: var(--spacing-xs) var(--spacing-sm);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  font-size: var(--text-xs);
  cursor: pointer;
  transition: var(--duration-normal) var(--ease-in-out);

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  &.btn-primary {
    background: var(--primary-500);
    color: white;
    border-color: var(--primary-500);

    &:hover:not(:disabled) {
      background: var(--primary-600);
    }
  }

  &.btn-secondary {
    background: var(--bg-secondary);
    color: var(--text-primary);

    &:hover:not(:disabled) {
      background: var(--state-hover);
    }
  }
}

.results-summary {
  margin-bottom: var(--spacing-md);

  h3 {
    margin: 0 0 var(--spacing-sm) 0;
    font-size: var(--text-sm);
    font-weight: var(--font-semibold);
  }
}

.summary-stats {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--spacing-xs);
}

.stat-item {
  display: flex;
  justify-content: space-between;
  padding: var(--spacing-xs);
  border-radius: var(--radius-md);

  &.error {
    background: var(--error-50);
    color: var(--error-700);
  }

  &.warning {
    background: var(--warning-50);
    color: var(--warning-700);
  }

  &.info {
    background: var(--info-50);
    color: var(--info-700);
  }

  &.success {
    background: var(--success-50);
    color: var(--success-700);
  }
}

.stat-label {
  font-size: var(--text-xs);
}

.stat-value {
  font-weight: var(--font-semibold);
}

.issues-list {
  max-height: 40vh;
  overflow-y: auto;
}

.issue-section {
  margin-bottom: var(--spacing-md);
}

.issue-section-title {
  margin: 0 0 var(--spacing-sm) 0;
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);

  &.error {
    color: var(--error-600);
  }

  &.warning {
    color: var(--warning-600);
  }

  &.info {
    color: var(--info-600);
  }
}

.issue-items {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.issue-item {
  padding: var(--spacing-sm);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: var(--duration-normal) var(--ease-in-out);

  &:hover {
    background: var(--state-hover);
  }

  &.error {
    border-left: 3px solid var(--error-500);
    background: var(--error-50);
  }

  &.warning {
    border-left: 3px solid var(--warning-500);
    background: var(--warning-50);
  }

  &.info {
    border-left: 3px solid var(--info-500);
    background: var(--info-50);
  }
}

.issue-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--spacing-xs);
}

.issue-rule {
  font-weight: var(--font-semibold);
  font-size: var(--text-xs);
}

.issue-impact {
  font-size: var(--text-xs);
  padding: 1px 4px;
  border-radius: var(--radius-sm);
  background: var(--bg-tertiary);
}

.issue-description {
  font-size: var(--text-xs);
  margin-bottom: var(--spacing-xs);
}

.issue-help {
  font-size: var(--text-xs);
  color: var(--text-secondary);
  margin-bottom: var(--spacing-xs);
}

.issue-selector {
  font-size: var(--text-xs);
  font-family: monospace;
  color: var(--text-tertiary);
}

.live-monitoring {
  margin-top: var(--spacing-md);
  padding-top: var(--spacing-md);
  border-top: 1px solid var(--border-primary);

  h4 {
    margin: 0 0 var(--spacing-sm) 0;
    font-size: var(--text-sm);
    font-weight: var(--font-semibold);
  }
}

.monitoring-stats {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.monitoring-item {
  display: flex;
  justify-content: space-between;
  font-size: var(--text-xs);
}

.monitoring-label {
  color: var(--text-secondary);
}

.monitoring-value {
  font-weight: var(--font-semibold);
}

// 하이라이트 스타일 (전역)
:global(.accessibility-highlight) {
  outline: 3px solid var(--error-500) !important;
  outline-offset: 2px !important;
  background: var(--error-100) !important;
  animation: accessibility-pulse 1s ease-in-out;
}

@keyframes accessibility-pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

// 모바일 대응
@media (max-width: 640px) {
  .accessibility-checker {
    bottom: var(--spacing-md);
    right: var(--spacing-md);
    left: var(--spacing-md);
    max-width: none;
  }

  .summary-stats {
    grid-template-columns: 1fr;
  }
}
</style>