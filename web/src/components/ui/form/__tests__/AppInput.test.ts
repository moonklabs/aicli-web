import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import AppInput from '../AppInput.vue'

describe('AppInput', () => {
  describe('기본 렌더링', () => {
    it('기본 프롭스로 올바르게 렌더링된다', () => {
      const wrapper = mount(AppInput)

      const container = wrapper.find('.app-input')
      const input = wrapper.find('.app-input__field')

      expect(container.exists()).toBe(true)
      expect(input.exists()).toBe(true)
      expect(container.classes()).toContain('app-input--medium')
      expect(container.classes()).toContain('app-input--default')
      expect(input.attributes('type')).toBe('text')
    })

    it('modelValue가 올바르게 바인딩된다', async () => {
      const wrapper = mount(AppInput, {
        props: { modelValue: 'test value' },
      })

      const input = wrapper.find('input')
      expect(input.element.value).toBe('test value')

      // 값 변경 테스트
      await wrapper.setProps({ modelValue: 'new value' })
      expect(input.element.value).toBe('new value')
    })
  })

  describe('프롭스', () => {
    it('type 프롭이 올바른 input type을 설정한다', () => {
      const types = ['text', 'password', 'email', 'number', 'tel', 'url'] as const

      types.forEach(type => {
        const wrapper = mount(AppInput, {
          props: { type },
        })

        expect(wrapper.find('input').attributes('type')).toBe(type)
      })
    })

    it('size 프롭이 올바른 클래스를 적용한다', () => {
      const sizes = ['small', 'medium', 'large'] as const

      sizes.forEach(size => {
        const wrapper = mount(AppInput, {
          props: { size },
        })

        expect(wrapper.find('.app-input').classes()).toContain(`app-input--${size}`)
      })
    })

    it('status 프롭이 올바른 클래스를 적용한다', () => {
      const statuses = ['default', 'success', 'warning', 'error'] as const

      statuses.forEach(status => {
        const wrapper = mount(AppInput, {
          props: { status },
        })

        expect(wrapper.find('.app-input').classes()).toContain(`app-input--${status}`)
      })
    })

    it('placeholder가 올바르게 설정된다', () => {
      const wrapper = mount(AppInput, {
        props: { placeholder: '텍스트를 입력하세요' },
      })

      expect(wrapper.find('input').attributes('placeholder')).toBe('텍스트를 입력하세요')
    })

    it('disabled 프롭이 올바르게 작동한다', () => {
      const wrapper = mount(AppInput, {
        props: { disabled: true },
      })

      const container = wrapper.find('.app-input')
      const input = wrapper.find('input')

      expect(container.classes()).toContain('app-input--disabled')
      expect(input.attributes('disabled')).toBeDefined()
    })

    it('readonly 프롭이 올바르게 작동한다', () => {
      const wrapper = mount(AppInput, {
        props: { readonly: true },
      })

      const container = wrapper.find('.app-input')
      const input = wrapper.find('input')

      expect(container.classes()).toContain('app-input--readonly')
      expect(input.attributes('readonly')).toBeDefined()
    })

    it('maxlength가 올바르게 설정된다', () => {
      const wrapper = mount(AppInput, {
        props: { maxlength: 100 },
      })

      expect(wrapper.find('input').attributes('maxlength')).toBe('100')
    })

    it('round 프롭이 올바른 클래스를 적용한다', () => {
      const wrapper = mount(AppInput, {
        props: { round: true },
      })

      expect(wrapper.find('.app-input').classes()).toContain('app-input--round')
    })
  })

  describe('clearable 기능', () => {
    it('clearable이 true이고 값이 있을 때 클리어 버튼이 표시된다', async () => {
      const wrapper = mount(AppInput, {
        props: {
          clearable: true,
          modelValue: 'test value',
        },
      })

      expect(wrapper.find('.app-input__clear').exists()).toBe(true)
    })

    it('clearable이 true이지만 값이 없을 때 클리어 버튼이 숨겨진다', () => {
      const wrapper = mount(AppInput, {
        props: {
          clearable: true,
          modelValue: '',
        },
      })

      expect(wrapper.find('.app-input__clear').exists()).toBe(false)
    })

    it('클리어 버튼 클릭 시 값이 초기화된다', async () => {
      const wrapper = mount(AppInput, {
        props: {
          clearable: true,
          modelValue: 'test value',
        },
      })

      await wrapper.find('.app-input__clear').trigger('click')

      expect(wrapper.emitted('update:value')).toHaveLength(1)
      expect(wrapper.emitted('update:value')[0]).toEqual([''])
      expect(wrapper.emitted('clear')).toHaveLength(1)
    })

    it('Escape 키로 클리어가 가능하다', async () => {
      const wrapper = mount(AppInput, {
        props: {
          clearable: true,
          modelValue: 'test value',
        },
      })

      await wrapper.find('input').trigger('keydown', { key: 'Escape' })

      expect(wrapper.emitted('update:value')).toHaveLength(1)
      expect(wrapper.emitted('update:value')[0]).toEqual([''])
      expect(wrapper.emitted('clear')).toHaveLength(1)
    })
  })

  describe('비밀번호 토글', () => {
    it('password 타입일 때 토글 버튼이 표시된다', () => {
      const wrapper = mount(AppInput, {
        props: { type: 'password' },
      })

      expect(wrapper.find('.app-input__password-toggle').exists()).toBe(true)
    })

    it('showPasswordOn이 false일 때 토글 버튼이 숨겨진다', () => {
      const wrapper = mount(AppInput, {
        props: {
          type: 'password',
          showPasswordOn: false,
        },
      })

      expect(wrapper.find('.app-input__password-toggle').exists()).toBe(false)
    })

    it('토글 버튼 클릭 시 비밀번호 가시성이 변경된다', async () => {
      const wrapper = mount(AppInput, {
        props: { type: 'password' },
      })

      const input = wrapper.find('input')
      const toggleBtn = wrapper.find('.app-input__password-toggle')

      // 초기 상태: password 타입
      expect(input.attributes('type')).toBe('password')

      // 토글 클릭 후: text 타입
      await toggleBtn.trigger('click')
      expect(input.attributes('type')).toBe('text')

      // 다시 토글 클릭 후: password 타입
      await toggleBtn.trigger('click')
      expect(input.attributes('type')).toBe('password')
    })
  })

  describe('문자 수 표시', () => {
    it('showCount가 true일 때 문자 수가 표시된다', () => {
      const wrapper = mount(AppInput, {
        props: {
          showCount: true,
          modelValue: 'test',
        },
      })

      expect(wrapper.find('.app-input__count').exists()).toBe(true)
      expect(wrapper.find('.app-input__count').text()).toBe('4')
    })

    it('maxlength가 있을 때 형식이 올바르게 표시된다', () => {
      const wrapper = mount(AppInput, {
        props: {
          showCount: true,
          maxlength: 10,
          modelValue: 'test',
        },
      })

      expect(wrapper.find('.app-input__count').text()).toBe('4/10')
    })

    it('문자 수가 비어있을 때 0으로 표시된다', () => {
      const wrapper = mount(AppInput, {
        props: {
          showCount: true,
          modelValue: '',
        },
      })

      expect(wrapper.find('.app-input__count').text()).toBe('0')
    })
  })

  describe('prefix/suffix 슬롯', () => {
    it('prefix 슬롯이 올바르게 렌더링된다', () => {
      const wrapper = mount(AppInput, {
        slots: {
          prefix: '<span>$</span>',
        },
      })

      expect(wrapper.find('.app-input__prefix').exists()).toBe(true)
      expect(wrapper.find('.app-input__prefix').html()).toContain('<span>$</span>')
      expect(wrapper.find('.app-input').classes()).toContain('app-input--with-prefix')
    })

    it('suffix 슬롯이 올바르게 렌더링된다', () => {
      const wrapper = mount(AppInput, {
        slots: {
          suffix: '<span>원</span>',
        },
      })

      expect(wrapper.find('.app-input__suffix-content').exists()).toBe(true)
      expect(wrapper.find('.app-input__suffix-content').html()).toContain('<span>원</span>')
      expect(wrapper.find('.app-input').classes()).toContain('app-input--with-suffix')
    })
  })

  describe('이벤트', () => {
    it('input 이벤트가 올바르게 발생한다', async () => {
      const wrapper = mount(AppInput)
      const input = wrapper.find('input')

      await input.setValue('test value')

      expect(wrapper.emitted('update:value')).toHaveLength(1)
      expect(wrapper.emitted('update:value')[0]).toEqual(['test value'])
      expect(wrapper.emitted('input')).toHaveLength(1)
    })

    it('change 이벤트가 올바르게 발생한다', async () => {
      const wrapper = mount(AppInput)
      const input = wrapper.find('input')

      await input.trigger('change')

      expect(wrapper.emitted('change')).toHaveLength(1)
    })

    it('focus/blur 이벤트가 올바르게 발생한다', async () => {
      const wrapper = mount(AppInput)
      const input = wrapper.find('input')

      await input.trigger('focus')
      expect(wrapper.emitted('focus')).toHaveLength(1)
      expect(wrapper.find('.app-input').classes()).toContain('app-input--focused')

      await input.trigger('blur')
      expect(wrapper.emitted('blur')).toHaveLength(1)
      expect(wrapper.find('.app-input').classes()).not.toContain('app-input--focused')
    })

    it('keydown/keyup 이벤트가 올바르게 발생한다', async () => {
      const wrapper = mount(AppInput)
      const input = wrapper.find('input')

      await input.trigger('keydown', { key: 'Enter' })
      expect(wrapper.emitted('keydown')).toHaveLength(1)

      await input.trigger('keyup', { key: 'Enter' })
      expect(wrapper.emitted('keyup')).toHaveLength(1)
    })
  })

  describe('number 타입', () => {
    it('number 타입일 때 숫자 값을 올바르게 처리한다', async () => {
      const wrapper = mount(AppInput, {
        props: { type: 'number' },
      })

      const input = wrapper.find('input')
      await input.setValue('123')

      expect(wrapper.emitted('update:value')).toHaveLength(1)
      expect(wrapper.emitted('update:value')[0]).toEqual([123])
    })

    it('number 타입에서 클리어 시 0으로 설정된다', async () => {
      const wrapper = mount(AppInput, {
        props: {
          type: 'number',
          clearable: true,
          modelValue: 123,
        },
      })

      await wrapper.find('.app-input__clear').trigger('click')

      expect(wrapper.emitted('update:value')).toHaveLength(1)
      expect(wrapper.emitted('update:value')[0]).toEqual([0])
    })
  })

  describe('접근성', () => {
    it('ARIA 속성이 올바르게 설정된다', () => {
      const wrapper = mount(AppInput, {
        props: {
          ariaLabel: '사용자명 입력',
          ariaDescribedby: 'help-text',
          required: true,
          status: 'error',
        },
      })

      const input = wrapper.find('input')
      expect(input.attributes('aria-label')).toBe('사용자명 입력')
      expect(input.attributes('aria-describedby')).toBe('help-text')
      expect(input.attributes('aria-required')).toBe('true')
      expect(input.attributes('aria-invalid')).toBe('true')
    })

    it('disabled 상태에서 tabindex가 -1로 설정된다', () => {
      const wrapper = mount(AppInput, {
        props: { disabled: true },
      })

      expect(wrapper.find('input').attributes('tabindex')).toBe('-1')
    })

    it('커스텀 tabindex가 올바르게 설정된다', () => {
      const wrapper = mount(AppInput, {
        props: { tabindex: 5 },
      })

      expect(wrapper.find('input').attributes('tabindex')).toBe('5')
    })
  })

  describe('노출된 메서드', () => {
    it('focus 메서드가 올바르게 작동한다', () => {
      const wrapper = mount(AppInput)
      const focusSpy = vi.spyOn(wrapper.find('input').element, 'focus')

      wrapper.vm.focus()
      expect(focusSpy).toHaveBeenCalled()
    })

    it('blur 메서드가 올바르게 작동한다', () => {
      const wrapper = mount(AppInput)
      const blurSpy = vi.spyOn(wrapper.find('input').element, 'blur')

      wrapper.vm.blur()
      expect(blurSpy).toHaveBeenCalled()
    })

    it('select 메서드가 올바르게 작동한다', () => {
      const wrapper = mount(AppInput)
      const selectSpy = vi.spyOn(wrapper.find('input').element, 'select')

      wrapper.vm.select()
      expect(selectSpy).toHaveBeenCalled()
    })
  })

  describe('v-model 양방향 바인딩', () => {
    it('v-model이 올바르게 작동한다', async () => {
      const wrapper = mount({
        components: { AppInput },
        template: '<AppInput v-model:value="value" />',
        data() {
          return { value: 'initial' }
        },
      })

      // 초기 값 확인
      expect(wrapper.find('input').element.value).toBe('initial')

      // 입력 값 변경
      await wrapper.find('input').setValue('changed')
      expect(wrapper.vm.value).toBe('changed')

      // 프로그래밍 방식 값 변경
      wrapper.vm.value = 'programmatic'
      await nextTick()
      expect(wrapper.find('input').element.value).toBe('programmatic')
    })
  })
})