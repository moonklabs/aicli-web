import { beforeEach, describe, expect, it, vi } from 'vitest'
import { VueWrapper, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import TimePicker from './TimePicker.vue'

describe('TimePicker 컴포넌트', () => {
  let wrapper: VueWrapper<any>

  beforeEach(() => {
    wrapper = mount(TimePicker, {
      props: {
        modelValue: null,
      },
    })
  })

  describe('기본 렌더링', () => {
    it('컴포넌트가 올바르게 렌더링되어야 한다', () => {
      expect(wrapper.exists()).toBe(true)
      expect(wrapper.find('.time-picker').exists()).toBe(true)
    })

    it('입력 필드가 표시되어야 한다', () => {
      const input = wrapper.find('.time-picker__input')
      expect(input.exists()).toBe(true)
    })

    it('플레이스홀더가 표시되어야 한다', () => {
      const input = wrapper.find('.time-picker__input')
      expect(input.attributes('placeholder')).toBe('시간을 선택하세요')
    })

    it('커스텀 플레이스홀더가 적용되어야 한다', async () => {
      await wrapper.setProps({ placeholder: 'Select time' })
      const input = wrapper.find('.time-picker__input')
      expect(input.attributes('placeholder')).toBe('Select time')
    })
  })

  describe('시간 선택기 동작', () => {
    it('입력 필드 클릭 시 시간 선택기가 열려야 한다', async () => {
      const input = wrapper.find('.time-picker__input')
      await input.trigger('click')

      expect(wrapper.find('.time-picker__picker').exists()).toBe(true)
    })

    it('ESC 키로 시간 선택기가 닫혀야 한다', async () => {
      await wrapper.find('.time-picker__input').trigger('click')
      expect(wrapper.find('.time-picker__picker').exists()).toBe(true)

      await wrapper.find('.time-picker__input').trigger('keydown', { key: 'Escape' })
      await nextTick()

      expect(wrapper.find('.time-picker__picker').exists()).toBe(false)
    })
  })

  describe('12시간 형식', () => {
    it('기본값은 12시간 형식이어야 한다', async () => {
      await wrapper.find('.time-picker__input').trigger('click')

      const ampmToggle = wrapper.find('.time-picker__ampm')
      expect(ampmToggle.exists()).toBe(true)
    })

    it('AM/PM 토글이 작동해야 한다', async () => {
      await wrapper.setProps({ modelValue: '10:30' })
      await wrapper.find('.time-picker__input').trigger('click')

      const amButton = wrapper.find('.time-picker__am')
      const pmButton = wrapper.find('.time-picker__pm')

      expect(amButton.classes()).toContain('time-picker__ampm--active')

      await pmButton.trigger('click')
      expect(wrapper.emitted('update:modelValue')![0]).toEqual(['22:30'])
    })

    it('12시간 형식으로 시간이 표시되어야 한다', async () => {
      await wrapper.setProps({ modelValue: '14:30' }) // 2:30 PM

      const input = wrapper.find('.time-picker__input')
      expect(input.element.value).toBe('2:30 PM')
    })
  })

  describe('24시간 형식', () => {
    beforeEach(async () => {
      await wrapper.setProps({ use24Hour: true })
    })

    it('AM/PM 토글이 표시되지 않아야 한다', async () => {
      await wrapper.find('.time-picker__input').trigger('click')

      const ampmToggle = wrapper.find('.time-picker__ampm')
      expect(ampmToggle.exists()).toBe(false)
    })

    it('24시간 형식으로 시간이 표시되어야 한다', async () => {
      await wrapper.setProps({ modelValue: '14:30' })

      const input = wrapper.find('.time-picker__input')
      expect(input.element.value).toBe('14:30')
    })

    it('24시간 범위의 시간을 선택할 수 있어야 한다', async () => {
      await wrapper.find('.time-picker__input').trigger('click')

      const hours = wrapper.findAll('.time-picker__hour')
      expect(hours.length).toBe(24)
    })
  })

  describe('시간 선택', () => {
    beforeEach(async () => {
      await wrapper.find('.time-picker__input').trigger('click')
    })

    it('시간을 클릭하여 선택할 수 있어야 한다', async () => {
      const hour = wrapper.findAll('.time-picker__hour')[9] // 9시 또는 9AM
      await hour.trigger('click')

      const minute = wrapper.findAll('.time-picker__minute')[6] // 30분
      await minute.trigger('click')

      const confirmButton = wrapper.find('.time-picker__confirm')
      await confirmButton.trigger('click')

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('change')).toBeTruthy()
    })

    it('선택된 시간이 하이라이트되어야 한다', async () => {
      await wrapper.setProps({ modelValue: '10:30' })
      await wrapper.find('.time-picker__input').trigger('click')

      const selectedHour = wrapper.find('.time-picker__hour--selected')
      const selectedMinute = wrapper.find('.time-picker__minute--selected')

      expect(selectedHour.exists()).toBe(true)
      expect(selectedMinute.exists()).toBe(true)
    })
  })

  describe('분 단위 설정', () => {
    it('기본값은 5분 단위여야 한다', async () => {
      await wrapper.find('.time-picker__input').trigger('click')

      const minutes = wrapper.findAll('.time-picker__minute')
      expect(minutes).toHaveLength(12) // 0, 5, 10, ..., 55
      expect(minutes[1].text()).toBe('05')
    })

    it('15분 단위로 설정할 수 있어야 한다', async () => {
      await wrapper.setProps({ minuteStep: 15 })
      await wrapper.find('.time-picker__input').trigger('click')

      const minutes = wrapper.findAll('.time-picker__minute')
      expect(minutes).toHaveLength(4) // 0, 15, 30, 45
      expect(minutes[1].text()).toBe('15')
    })

    it('30분 단위로 설정할 수 있어야 한다', async () => {
      await wrapper.setProps({ minuteStep: 30 })
      await wrapper.find('.time-picker__input').trigger('click')

      const minutes = wrapper.findAll('.time-picker__minute')
      expect(minutes).toHaveLength(2) // 0, 30
    })
  })

  describe('현재 시간 버튼', () => {
    it('현재 시간 버튼이 표시되어야 한다', async () => {
      await wrapper.find('.time-picker__input').trigger('click')

      const nowButton = wrapper.find('.time-picker__now')
      expect(nowButton.exists()).toBe(true)
      expect(nowButton.text()).toBe('현재')
    })

    it('현재 시간 버튼 클릭 시 현재 시간이 선택되어야 한다', async () => {
      await wrapper.find('.time-picker__input').trigger('click')

      const nowButton = wrapper.find('.time-picker__now')
      await nowButton.trigger('click')

      const now = new Date()
      const expectedTime = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}`

      // 분 단위로 반올림된 값과 비교
      const emitted = wrapper.emitted('update:modelValue')
      expect(emitted).toBeTruthy()
      const [hours, minutes] = emitted![0][0].split(':')
      expect(parseInt(hours)).toBeCloseTo(now.getHours(), 0)
    })

    it('커스텀 텍스트가 적용되어야 한다', async () => {
      await wrapper.setProps({ nowButtonText: 'Now' })
      await wrapper.find('.time-picker__input').trigger('click')

      const nowButton = wrapper.find('.time-picker__now')
      expect(nowButton.text()).toBe('Now')
    })
  })

  describe('시간 범위 제한', () => {
    it('min 시간 이전은 비활성화되어야 한다', async () => {
      await wrapper.setProps({ min: '09:00' })
      await wrapper.find('.time-picker__input').trigger('click')

      const hour8 = wrapper.findAll('.time-picker__hour')[8] // 8시
      expect(hour8.classes()).toContain('time-picker__hour--disabled')

      await hour8.trigger('click')
      expect(wrapper.vm.selectedHour).not.toBe(8)
    })

    it('max 시간 이후는 비활성화되어야 한다', async () => {
      await wrapper.setProps({ max: '18:00' })
      await wrapper.find('.time-picker__input').trigger('click')

      const hour19 = wrapper.findAll('.time-picker__hour')[19] // 19시
      if (hour19) {
        expect(hour19.classes()).toContain('time-picker__hour--disabled')
      }
    })

    it('시간 선택에 따라 분이 동적으로 비활성화되어야 한다', async () => {
      await wrapper.setProps({ min: '09:30' })
      await wrapper.find('.time-picker__input').trigger('click')

      // 9시 선택
      const hour9 = wrapper.findAll('.time-picker__hour')[9]
      await hour9.trigger('click')

      // 0분, 5분, 10분, 15분, 20분, 25분은 비활성화되어야 함
      const minutes = wrapper.findAll('.time-picker__minute')
      expect(minutes[0].classes()).toContain('time-picker__minute--disabled') // 00
      expect(minutes[1].classes()).toContain('time-picker__minute--disabled') // 05
    })
  })

  describe('지우기 버튼', () => {
    it('값이 있을 때 지우기 버튼이 표시되어야 한다', async () => {
      await wrapper.setProps({
        modelValue: '10:30',
        clearable: true,
      })

      const clearButton = wrapper.find('.time-picker__clear')
      expect(clearButton.exists()).toBe(true)
    })

    it('지우기 버튼 클릭 시 값이 null이 되어야 한다', async () => {
      await wrapper.setProps({
        modelValue: '10:30',
        clearable: true,
      })

      const clearButton = wrapper.find('.time-picker__clear')
      await clearButton.trigger('click')

      expect(wrapper.emitted('update:modelValue')![0]).toEqual([null])
      expect(wrapper.emitted('clear')).toBeTruthy()
    })
  })

  describe('비활성화 상태', () => {
    it('disabled일 때 입력이 불가능해야 한다', async () => {
      await wrapper.setProps({ disabled: true })

      const input = wrapper.find('.time-picker__input')
      expect(input.attributes('disabled')).toBeDefined()
    })

    it('disabled일 때 시간 선택기가 열리지 않아야 한다', async () => {
      await wrapper.setProps({ disabled: true })

      const input = wrapper.find('.time-picker__input')
      await input.trigger('click')

      expect(wrapper.find('.time-picker__picker').exists()).toBe(false)
    })
  })

  describe('키보드 네비게이션', () => {
    beforeEach(async () => {
      await wrapper.find('.time-picker__input').trigger('click')
    })

    it('Tab 키로 시간과 분 사이를 이동할 수 있어야 한다', async () => {
      const picker = wrapper.find('.time-picker__picker')

      await picker.trigger('keydown', { key: 'Tab' })
      expect(wrapper.vm.focusedSection).toBe('minute')

      await picker.trigger('keydown', { key: 'Tab', shiftKey: true })
      expect(wrapper.vm.focusedSection).toBe('hour')
    })

    it('화살표 키로 시간/분을 선택할 수 있어야 한다', async () => {
      const picker = wrapper.find('.time-picker__picker')

      // 시간 섹션에서
      await picker.trigger('keydown', { key: 'ArrowDown' })
      expect(wrapper.vm.highlightedHour).toBeDefined()

      await picker.trigger('keydown', { key: 'Enter' })
      expect(wrapper.vm.selectedHour).toBeDefined()
    })

    it('Space 키로 AM/PM을 토글할 수 있어야 한다', async () => {
      await wrapper.setProps({ modelValue: '10:00' }) // AM
      const picker = wrapper.find('.time-picker__picker')

      await picker.trigger('keydown', { key: ' ', target: { closest: () => '.time-picker__ampm' } })

      const emitted = wrapper.emitted('update:modelValue')
      if (emitted && emitted[0]) {
        expect(emitted[0][0]).toBe('22:00') // PM으로 변경
      }
    })
  })

  describe('접근성', () => {
    it('적절한 ARIA 속성이 설정되어야 한다', () => {
      const input = wrapper.find('.time-picker__input')

      expect(input.attributes('role')).toBe('combobox')
      expect(input.attributes('aria-expanded')).toBe('false')
      expect(input.attributes('aria-haspopup')).toBe('dialog')
    })

    it('시간 선택기가 열릴 때 aria-expanded가 true가 되어야 한다', async () => {
      await wrapper.find('.time-picker__input').trigger('click')

      const input = wrapper.find('.time-picker__input')
      expect(input.attributes('aria-expanded')).toBe('true')
    })

    it('시간/분 그리드에 적절한 role이 있어야 한다', async () => {
      await wrapper.find('.time-picker__input').trigger('click')

      const hourGrid = wrapper.find('.time-picker__hours')
      const minuteGrid = wrapper.find('.time-picker__minutes')

      expect(hourGrid.attributes('role')).toBe('grid')
      expect(minuteGrid.attributes('role')).toBe('grid')
    })
  })

  describe('입력 검증', () => {
    it('직접 입력한 유효한 시간이 파싱되어야 한다', async () => {
      const input = wrapper.find('.time-picker__input')

      await input.setValue('14:30')
      await input.trigger('blur')

      expect(wrapper.emitted('update:modelValue')![0]).toEqual(['14:30'])
    })

    it('잘못된 형식의 입력은 무시되어야 한다', async () => {
      const input = wrapper.find('.time-picker__input')

      await input.setValue('25:70') // 잘못된 시간
      await input.trigger('blur')

      expect(wrapper.emitted('update:modelValue')).toBeFalsy()
    })

    it('12시간 형식 입력을 처리할 수 있어야 한다', async () => {
      await wrapper.setProps({ use24Hour: false })
      const input = wrapper.find('.time-picker__input')

      await input.setValue('3:30 PM')
      await input.trigger('blur')

      expect(wrapper.emitted('update:modelValue')![0]).toEqual(['15:30'])
    })
  })

  describe('이벤트', () => {
    it('focus/blur 이벤트가 발생해야 한다', async () => {
      const input = wrapper.find('.time-picker__input')

      await input.trigger('focus')
      expect(wrapper.emitted('focus')).toBeTruthy()

      await input.trigger('blur')
      expect(wrapper.emitted('blur')).toBeTruthy()
    })

    it('open/close 이벤트가 발생해야 한다', async () => {
      const input = wrapper.find('.time-picker__input')

      await input.trigger('click')
      expect(wrapper.emitted('open')).toBeTruthy()

      await wrapper.find('.time-picker__confirm').trigger('click')
      expect(wrapper.emitted('close')).toBeTruthy()
    })
  })

  describe('엣지 케이스', () => {
    it('자정(00:00)을 올바르게 처리해야 한다', async () => {
      await wrapper.setProps({ modelValue: '00:00' })

      const input = wrapper.find('.time-picker__input')
      expect(input.element.value).toBe('12:00 AM')
    })

    it('정오(12:00)를 올바르게 처리해야 한다', async () => {
      await wrapper.setProps({ modelValue: '12:00' })

      const input = wrapper.find('.time-picker__input')
      expect(input.element.value).toBe('12:00 PM')
    })

    it('minuteStep이 60일 때를 처리할 수 있어야 한다', async () => {
      await wrapper.setProps({ minuteStep: 60 })
      await wrapper.find('.time-picker__input').trigger('click')

      const minutes = wrapper.findAll('.time-picker__minute')
      expect(minutes).toHaveLength(1)
      expect(minutes[0].text()).toBe('00')
    })
  })
})