import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AppButton from '../AppButton.vue'

describe('AppButton', () => {
  describe('기본 렌더링', () => {
    it('기본 프롭스로 올바르게 렌더링된다', () => {
      const wrapper = mount(AppButton, {
        slots: { default: '버튼 텍스트' },
      })

      const button = wrapper.find('button')
      expect(button.exists()).toBe(true)
      expect(button.text()).toBe('버튼 텍스트')
      expect(button.classes()).toContain('app-button')
      expect(button.classes()).toContain('app-button--medium')
      expect(button.classes()).toContain('app-button--solid')
      expect(button.classes()).toContain('app-button--default')
    })

    it('슬롯 내용이 올바르게 렌더링된다', () => {
      const wrapper = mount(AppButton, {
        slots: {
          default: '<span>커스텀 내용</span>',
          icon: '<svg>아이콘</svg>',
          suffix: '<span>접미사</span>',
        },
      })

      expect(wrapper.find('.app-button__content').html()).toContain('<span>커스텀 내용</span>')
      expect(wrapper.find('.app-button__icon').html()).toContain('<svg>아이콘</svg>')
      expect(wrapper.find('.app-button__suffix').html()).toContain('<span>접미사</span>')
    })
  })

  describe('프롭스', () => {
    it('type 프롭이 올바른 클래스를 적용한다', () => {
      const types = ['default', 'primary', 'success', 'warning', 'error', 'info'] as const

      types.forEach(type => {
        const wrapper = mount(AppButton, {
          props: { type },
          slots: { default: '버튼' },
        })

        expect(wrapper.find('button').classes()).toContain(`app-button--${type}`)
      })
    })

    it('size 프롭이 올바른 클래스를 적용한다', () => {
      const sizes = ['small', 'medium', 'large'] as const

      sizes.forEach(size => {
        const wrapper = mount(AppButton, {
          props: { size },
          slots: { default: '버튼' },
        })

        expect(wrapper.find('button').classes()).toContain(`app-button--${size}`)
      })
    })

    it('variant 프롭이 올바른 클래스를 적용한다', () => {
      const variants = ['solid', 'outline', 'ghost', 'text'] as const

      variants.forEach(variant => {
        const wrapper = mount(AppButton, {
          props: { variant },
          slots: { default: '버튼' },
        })

        expect(wrapper.find('button').classes()).toContain(`app-button--${variant}`)
      })
    })

    it('disabled 프롭이 올바르게 작동한다', () => {
      const wrapper = mount(AppButton, {
        props: { disabled: true },
        slots: { default: '버튼' },
      })

      const button = wrapper.find('button')
      expect(button.attributes('disabled')).toBeDefined()
      expect(button.classes()).toContain('app-button--disabled')
    })

    it('loading 프롭이 올바르게 작동한다', () => {
      const wrapper = mount(AppButton, {
        props: { loading: true },
        slots: { default: '버튼' },
      })

      const button = wrapper.find('button')
      expect(button.attributes('disabled')).toBeDefined()
      expect(button.classes()).toContain('app-button--loading')
      expect(wrapper.find('.app-button__loading').exists()).toBe(true)
    })

    it('block 프롭이 올바른 클래스를 적용한다', () => {
      const wrapper = mount(AppButton, {
        props: { block: true },
        slots: { default: '버튼' },
      })

      expect(wrapper.find('button').classes()).toContain('app-button--block')
    })

    it('round 프롭이 올바른 클래스를 적용한다', () => {
      const wrapper = mount(AppButton, {
        props: { round: true },
        slots: { default: '버튼' },
      })

      expect(wrapper.find('button').classes()).toContain('app-button--round')
    })

    it('circle 프롭이 올바른 클래스를 적용한다', () => {
      const wrapper = mount(AppButton, {
        props: { circle: true },
        slots: { default: '버튼' },
      })

      expect(wrapper.find('button').classes()).toContain('app-button--circle')
    })

    it('htmlType 프롭이 올바른 type 속성을 설정한다', () => {
      const types = ['button', 'submit', 'reset'] as const

      types.forEach(htmlType => {
        const wrapper = mount(AppButton, {
          props: { htmlType },
          slots: { default: '버튼' },
        })

        expect(wrapper.find('button').attributes('type')).toBe(htmlType)
      })
    })
  })

  describe('이벤트', () => {
    it('클릭 이벤트가 올바르게 발생한다', async () => {
      const wrapper = mount(AppButton, {
        slots: { default: '버튼' },
      })

      await wrapper.find('button').trigger('click')
      expect(wrapper.emitted('click')).toHaveLength(1)
    })

    it('disabled 상태에서 클릭 이벤트가 발생하지 않는다', async () => {
      const clickHandler = vi.fn()
      const wrapper = mount(AppButton, {
        props: {
          disabled: true,
          onClick: clickHandler,
        },
        slots: { default: '버튼' },
      })

      await wrapper.find('button').trigger('click')
      expect(clickHandler).not.toHaveBeenCalled()
    })

    it('loading 상태에서 클릭 이벤트가 발생하지 않는다', async () => {
      const clickHandler = vi.fn()
      const wrapper = mount(AppButton, {
        props: {
          loading: true,
          onClick: clickHandler,
        },
        slots: { default: '버튼' },
      })

      await wrapper.find('button').trigger('click')
      expect(clickHandler).not.toHaveBeenCalled()
    })

    it('키보드 이벤트가 올바르게 처리된다', async () => {
      const wrapper = mount(AppButton, {
        slots: { default: '버튼' },
      })

      // Enter 키
      await wrapper.find('button').trigger('keydown', { key: 'Enter' })
      expect(wrapper.emitted('click')).toHaveLength(1)

      // Space 키
      await wrapper.find('button').trigger('keydown', { key: ' ' })
      expect(wrapper.emitted('click')).toHaveLength(2)
    })

    it('포커스 이벤트가 올바르게 발생한다', async () => {
      const wrapper = mount(AppButton, {
        slots: { default: '버튼' },
      })

      await wrapper.find('button').trigger('focus')
      expect(wrapper.emitted('focus')).toHaveLength(1)

      await wrapper.find('button').trigger('blur')
      expect(wrapper.emitted('blur')).toHaveLength(1)
    })
  })

  describe('접근성', () => {
    it('ARIA 속성이 올바르게 설정된다', () => {
      const wrapper = mount(AppButton, {
        props: {
          ariaLabel: '커스텀 라벨',
          ariaDescribedby: 'description-id',
          ariaExpanded: true,
          ariaPressed: true,
        },
        slots: { default: '버튼' },
      })

      const button = wrapper.find('button')
      expect(button.attributes('aria-label')).toBe('커스텀 라벨')
      expect(button.attributes('aria-describedby')).toBe('description-id')
      expect(button.attributes('aria-expanded')).toBe('true')
      expect(button.attributes('aria-pressed')).toBe('true')
    })

    it('tabindex가 올바르게 설정된다', () => {
      // 기본 상태
      const wrapper1 = mount(AppButton, {
        slots: { default: '버튼' },
      })
      expect(wrapper1.find('button').attributes('tabindex')).toBe('0')

      // disabled 상태
      const wrapper2 = mount(AppButton, {
        props: { disabled: true },
        slots: { default: '버튼' },
      })
      expect(wrapper2.find('button').attributes('tabindex')).toBe('-1')

      // 커스텀 tabindex
      const wrapper3 = mount(AppButton, {
        props: { tabindex: 5 },
        slots: { default: '버튼' },
      })
      expect(wrapper3.find('button').attributes('tabindex')).toBe('5')
    })
  })

  describe('아이콘 전용 버튼', () => {
    it('아이콘만 있는 버튼이 올바르게 식별된다', () => {
      const wrapper = mount(AppButton, {
        slots: { icon: '<svg>아이콘</svg>' },
      })

      expect(wrapper.find('button').classes()).toContain('app-button--icon-only')
    })

    it('circle 프롭이 있으면 아이콘 전용으로 처리된다', () => {
      const wrapper = mount(AppButton, {
        props: { circle: true },
        slots: {
          default: '텍스트',
          icon: '<svg>아이콘</svg>',
        },
      })

      expect(wrapper.find('button').classes()).toContain('app-button--icon-only')
    })
  })

  describe('로딩 상태', () => {
    it('로딩 상태에서 스피너가 표시된다', () => {
      const wrapper = mount(AppButton, {
        props: { loading: true },
        slots: { default: '버튼' },
      })

      expect(wrapper.find('.app-button__loading').exists()).toBe(true)
      expect(wrapper.findComponent({ name: 'AppSpinner' }).exists()).toBe(true)
    })

    it('로딩 상태에서 텍스트가 숨겨진다', () => {
      const wrapper = mount(AppButton, {
        props: { loading: true },
        slots: { default: '버튼 텍스트' },
      })

      const content = wrapper.find('.app-button__content')
      expect(content.classes()).toContain('app-button__content--hidden')
    })

    it('로딩 상태에서 아이콘이 숨겨진다', () => {
      const wrapper = mount(AppButton, {
        props: { loading: true },
        slots: {
          default: '버튼',
          icon: '<svg>아이콘</svg>',
        },
      })

      expect(wrapper.find('.app-button__icon').exists()).toBe(false)
    })
  })

  describe('스피너 설정', () => {
    it('버튼 크기에 따라 스피너 크기가 조정된다', () => {
      const sizes = [
        { size: 'small', expectedSpinnerSize: 'small' },
        { size: 'medium', expectedSpinnerSize: 'small' },
        { size: 'large', expectedSpinnerSize: 'medium' },
      ] as const

      sizes.forEach(({ size, expectedSpinnerSize }) => {
        const wrapper = mount(AppButton, {
          props: {
            loading: true,
            size,
          },
          slots: { default: '버튼' },
        })

        const spinner = wrapper.findComponent({ name: 'AppSpinner' })
        expect(spinner.props('size')).toBe(expectedSpinnerSize)
      })
    })

    it('버튼 타입에 따라 스피너 색상이 조정된다', () => {
      const wrapper = mount(AppButton, {
        props: {
          loading: true,
          type: 'primary',
          variant: 'outline',
        },
        slots: { default: '버튼' },
      })

      const spinner = wrapper.findComponent({ name: 'AppSpinner' })
      expect(spinner.props('variant')).toBe('primary')
    })
  })

  describe('Custom 속성', () => {
    it('커스텀 클래스가 올바르게 적용된다', () => {
      const wrapper = mount(AppButton, {
        props: { class: 'custom-class' },
        slots: { default: '버튼' },
      })

      expect(wrapper.find('button').classes()).toContain('custom-class')
    })

    it('커스텀 스타일이 올바르게 적용된다', () => {
      const wrapper = mount(AppButton, {
        props: { style: 'background-color: red;' },
        slots: { default: '버튼' },
      })

      expect(wrapper.find('button').attributes('style')).toContain('background-color: red')
    })

    it('데이터 속성이 올바르게 전달된다', () => {
      const wrapper = mount(AppButton, {
        attrs: { 'data-testid': 'test-button' },
        slots: { default: '버튼' },
      })

      expect(wrapper.find('button').attributes('data-testid')).toBe('test-button')
    })
  })
})