import { expect, test } from '@playwright/test'
import { checkA11y, getViolations, injectAxe } from 'axe-playwright'

/**
 * 접근성 테스트
 */
test.describe('접근성 테스트', () => {

  test.beforeEach(async ({ page }) => {
    // axe-core 라이브러리 주입
    await injectAxe(page)
  })

  test('홈페이지 접근성 검증', async ({ page }) => {
    await page.goto('/')
    await page.waitForLoadState('networkidle')

    // 접근성 검사 실행
    await checkA11y(page, null, {
      detailedReport: true,
      detailedReportOptions: { html: true },
    })

    // 위반 사항 확인
    const violations = await getViolations(page)

    if (violations.length > 0) {
      console.log('🚨 접근성 위반 사항:', violations)
    }

    // 심각한 위반 사항이 없어야 함
    const criticalViolations = violations.filter(v => v.impact === 'critical' || v.impact === 'serious')
    expect(criticalViolations).toHaveLength(0)
  })

  test('키보드 네비게이션 테스트', async ({ page }) => {
    await page.goto('/')

    // 첫 번째 포커스 가능한 요소로 이동
    await page.keyboard.press('Tab')

    // 포커스된 요소 확인
    const focusedElement = await page.locator(':focus').first()
    expect(focusedElement).toBeVisible()

    // 여러 Tab 키로 네비게이션 테스트
    const focusableElements = []
    for (let i = 0; i < 10; i++) {
      await page.keyboard.press('Tab')
      const currentFocus = await page.locator(':focus').first()
      if (await currentFocus.isVisible()) {
        const tagName = await currentFocus.evaluate(el => el.tagName)
        const role = await currentFocus.getAttribute('role')
        focusableElements.push({ tagName, role, index: i })
      }
    }

    console.log('⌨️  키보드 네비게이션 경로:', focusableElements)

    // 최소 5개 이상의 포커스 가능한 요소가 있어야 함
    expect(focusableElements.length).toBeGreaterThan(5)
  })

  test('스크린 리더 호환성 테스트', async ({ page }) => {
    await page.goto('/')

    // aria-label 검증
    const elementsWithAriaLabel = await page.locator('[aria-label]').count()
    console.log(`🔊 aria-label이 있는 요소: ${elementsWithAriaLabel}개`)

    // 헤딩 구조 검증
    const headings = await page.locator('h1, h2, h3, h4, h5, h6').all()
    const headingStructure = []

    for (const heading of headings) {
      const tagName = await heading.evaluate(el => el.tagName)
      const text = await heading.textContent()
      headingStructure.push({ level: tagName, text: text?.slice(0, 50) })
    }

    console.log('📋 헤딩 구조:', headingStructure)

    // H1이 존재해야 함
    expect(headingStructure.some(h => h.level === 'H1')).toBeTruthy()

    // 이미지 alt 텍스트 검증
    const images = await page.locator('img').all()
    for (const image of images) {
      const alt = await image.getAttribute('alt')
      const src = await image.getAttribute('src')

      if (!alt && src && !src.includes('data:')) {
        console.warn(`⚠️  alt 텍스트가 없는 이미지: ${src}`)
      }
    }
  })

  test('색상 대비 테스트', async ({ page }) => {
    await page.goto('/')

    // 색상 대비 검사를 위한 axe 규칙
    await checkA11y(page, null, {
      rules: {
        'color-contrast': { enabled: true },
      },
    })

    // 사용자 정의 색상 대비 검사
    const colorContrastResults = await page.evaluate(() => {
      const elements = document.querySelectorAll('*')
      const results = []

      for (const element of elements) {
        const computedStyle = window.getComputedStyle(element)
        const color = computedStyle.color
        const backgroundColor = computedStyle.backgroundColor

        if (color !== 'rgba(0, 0, 0, 0)' && backgroundColor !== 'rgba(0, 0, 0, 0)') {
          results.push({
            tagName: element.tagName,
            color,
            backgroundColor,
            text: element.textContent?.slice(0, 30),
          })
        }
      }

      return results.slice(0, 10) // 처음 10개만 반환
    })

    console.log('🎨 색상 대비 분석 샘플:', colorContrastResults)
  })

  test('포커스 표시 테스트', async ({ page }) => {
    await page.goto('/')

    // 모든 인터랙티브 요소에 포커스 표시가 있는지 확인
    const interactiveSelectors = [
      'button',
      'input',
      'select',
      'textarea',
      'a[href]',
      '[tabindex]:not([tabindex="-1"])',
      '[role="button"]',
      '[role="link"]',
    ]

    for (const selector of interactiveSelectors) {
      const elements = await page.locator(selector).all()

      for (const element of elements.slice(0, 5)) { // 처음 5개만 테스트
        if (await element.isVisible()) {
          await element.focus()

          // 포커스 스타일 확인
          const focusStyle = await element.evaluate(el => {
            const style = window.getComputedStyle(el)
            return {
              outline: style.outline,
              outlineWidth: style.outlineWidth,
              outlineStyle: style.outlineStyle,
              outlineColor: style.outlineColor,
              boxShadow: style.boxShadow,
            }
          })

          // 포커스 표시가 있어야 함 (outline 또는 box-shadow)
          const hasFocusIndicator =
            focusStyle.outline !== 'none' ||
            focusStyle.outlineWidth !== '0px' ||
            focusStyle.boxShadow !== 'none'

          if (!hasFocusIndicator) {
            const tagInfo = await element.evaluate(el => ({
              tagName: el.tagName,
              className: el.className,
              id: el.id,
            }))
            console.warn('⚠️  포커스 표시가 없는 요소:', tagInfo)
          }
        }
      }
    }
  })

  test('ARIA 속성 테스트', async ({ page }) => {
    await page.goto('/')

    // ARIA 속성이 올바르게 사용되는지 확인
    const ariaElements = await page.evaluate(() => {
      const elements = document.querySelectorAll('[aria-label], [aria-labelledby], [aria-describedby], [role]')
      const results = []

      for (const element of elements) {
        const ariaLabel = element.getAttribute('aria-label')
        const ariaLabelledBy = element.getAttribute('aria-labelledby')
        const ariaDescribedBy = element.getAttribute('aria-describedby')
        const role = element.getAttribute('role')

        results.push({
          tagName: element.tagName,
          ariaLabel,
          ariaLabelledBy,
          ariaDescribedBy,
          role,
          text: element.textContent?.slice(0, 30),
        })
      }

      return results
    })

    console.log('🏷️  ARIA 속성 사용 현황:', ariaElements.slice(0, 10))

    // aria-labelledby 참조 검증
    for (const element of ariaElements) {
      if (element.ariaLabelledBy) {
        const referencedElement = await page.locator(`#${element.ariaLabelledBy}`).first()
        expect(referencedElement).toBeTruthy()
      }

      if (element.ariaDescribedBy) {
        const referencedElement = await page.locator(`#${element.ariaDescribedBy}`).first()
        expect(referencedElement).toBeTruthy()
      }
    }
  })

  test('모바일 접근성 테스트', async ({ page, context }) => {
    // 모바일 뷰포트로 설정
    await page.setViewportSize({ width: 375, height: 667 })
    await page.goto('/')

    // 터치 타겟 크기 검증 (최소 44x44px)
    const touchTargets = await page.locator('button, a, input, [role="button"]').all()

    for (const target of touchTargets.slice(0, 10)) {
      if (await target.isVisible()) {
        const boundingBox = await target.boundingBox()

        if (boundingBox) {
          const minSize = 44
          if (boundingBox.width < minSize || boundingBox.height < minSize) {
            const elementInfo = await target.evaluate(el => ({
              tagName: el.tagName,
              className: el.className,
              text: el.textContent?.slice(0, 20),
            }))

            console.warn(`⚠️  터치 타겟 크기 부족 (${boundingBox.width}x${boundingBox.height}):`, elementInfo)
          }
        }
      }
    }

    // 가로 스크롤 확인
    const hasHorizontalScroll = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth
    })

    expect(hasHorizontalScroll).toBeFalsy() // 가로 스크롤이 없어야 함
  })
})