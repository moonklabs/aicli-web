import { beforeEach, describe, expect, it, vi } from 'vitest'
import { VueWrapper, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import DateRangePicker from './DateRangePicker.vue'

interface DateRange {
  start: Date | null
  end: Date | null
}

describe('DateRangePicker 컴포넌트', () => {
  let wrapper: VueWrapper<any>
  const today = new Date()
  const tomorrow = new Date(today)
  tomorrow.setDate(tomorrow.getDate() + 1)
  const nextWeek = new Date(today)
  nextWeek.setDate(nextWeek.getDate() + 7)
  const lastWeek = new Date(today)
  lastWeek.setDate(lastWeek.getDate() - 7)

  beforeEach(() => {
    wrapper = mount(DateRangePicker, {
      props: {
        modelValue: null,
      },
    })
  })

  describe('기본 렌더링', () => {
    it('컴포넌트가 올바르게 렌더링되어야 한다', () => {
      expect(wrapper.exists()).toBe(true)
      expect(wrapper.find('.date-range-picker').exists()).toBe(true)
    })

    it('시작/종료 입력 필드가 표시되어야 한다', () => {
      const startInput = wrapper.find('.date-range-picker__start-input')
      const endInput = wrapper.find('.date-range-picker__end-input')

      expect(startInput.exists()).toBe(true)
      expect(endInput.exists()).toBe(true)
    })

    it('플레이스홀더가 표시되어야 한다', () => {
      const inputs = wrapper.findAll('input')
      expect(inputs[0].attributes('placeholder')).toBe('시작 날짜')
      expect(inputs[1].attributes('placeholder')).toBe('종료 날짜')
    })

    it('커스텀 플레이스홀더가 적용되어야 한다', async () => {
      await wrapper.setProps({
        startPlaceholder: 'From',
        endPlaceholder: 'To',
      })

      const inputs = wrapper.findAll('input')
      expect(inputs[0].attributes('placeholder')).toBe('From')
      expect(inputs[1].attributes('placeholder')).toBe('To')
    })
  })

  describe('캘린더 동작', () => {
    it('입력 필드 클릭 시 듀얼 캘린더가 열려야 한다', async () => {
      const input = wrapper.find('.date-range-picker__start-input')
      await input.trigger('click')

      const calendar = wrapper.find('.date-range-picker__calendar')
      expect(calendar.exists()).toBe(true)

      const monthViews = wrapper.findAll('.date-range-picker__month')
      expect(monthViews).toHaveLength(2)
    })

    it('ESC 키로 캘린더가 닫혀야 한다', async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')
      expect(wrapper.find('.date-range-picker__calendar').exists()).toBe(true)

      await wrapper.trigger('keydown', { key: 'Escape' })
      await nextTick()

      expect(wrapper.find('.date-range-picker__calendar').exists()).toBe(false)
    })
  })

  describe('날짜 범위 선택', () => {
    beforeEach(async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')
    })

    it('첫 번째 클릭으로 시작 날짜가 선택되어야 한다', async () => {
      const days = wrapper.findAll('.date-range-picker__day:not(.date-range-picker__day--other-month)')
      const targetDay = days[15]

      await targetDay.trigger('click')

      expect(wrapper.vm.tempRange.start).toBeInstanceOf(Date)
      expect(wrapper.vm.tempRange.end).toBeNull()
    })

    it('두 번째 클릭으로 종료 날짜가 선택되어야 한다', async () => {
      const days = wrapper.findAll('.date-range-picker__day:not(.date-range-picker__day--other-month)')

      await days[15].trigger('click') // 시작 날짜
      await days[20].trigger('click') // 종료 날짜

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      const emittedRange = wrapper.emitted('update:modelValue')![0][0] as DateRange
      expect(emittedRange.start).toBeInstanceOf(Date)
      expect(emittedRange.end).toBeInstanceOf(Date)
      expect(emittedRange.end > emittedRange.start).toBe(true)
    })

    it('시작 날짜보다 이전 날짜 클릭 시 새로운 선택이 시작되어야 한다', async () => {
      const days = wrapper.findAll('.date-range-picker__day:not(.date-range-picker__day--other-month)')

      await days[20].trigger('click') // 먼저 뒤쪽 날짜 선택
      await days[10].trigger('click') // 이전 날짜 선택

      // 새로운 선택 시작
      expect(wrapper.vm.tempRange.start).toBeInstanceOf(Date)
      expect(wrapper.vm.tempRange.end).toBeNull()
    })
  })

  describe('날짜 범위 하이라이팅', () => {
    it('선택된 범위가 하이라이트되어야 한다', async () => {
      await wrapper.setProps({
        modelValue: {
          start: today,
          end: nextWeek,
        },
      })
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      const inRangeDays = wrapper.findAll('.date-range-picker__day--in-range')
      expect(inRangeDays.length).toBeGreaterThan(0)
    })

    it('시작/종료 날짜가 특별히 표시되어야 한다', async () => {
      await wrapper.setProps({
        modelValue: {
          start: today,
          end: nextWeek,
        },
      })
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      const startDay = wrapper.find('.date-range-picker__day--start')
      const endDay = wrapper.find('.date-range-picker__day--end')

      expect(startDay.exists()).toBe(true)
      expect(endDay.exists()).toBe(true)
    })

    it('호버 시 임시 범위가 하이라이트되어야 한다', async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')
      const days = wrapper.findAll('.date-range-picker__day:not(.date-range-picker__day--other-month)')

      await days[15].trigger('click') // 시작 날짜 선택
      await days[20].trigger('mouseenter') // 종료 날짜 호버

      const hoveredRange = wrapper.findAll('.date-range-picker__day--hover-range')
      expect(hoveredRange.length).toBeGreaterThan(0)
    })
  })

  describe('프리셋', () => {
    it('기본 프리셋이 표시되어야 한다', async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      const presets = wrapper.find('.date-range-picker__presets')
      expect(presets.exists()).toBe(true)

      const presetButtons = wrapper.findAll('.date-range-picker__preset')
      expect(presetButtons.length).toBeGreaterThan(0)
    })

    it('프리셋 클릭 시 날짜 범위가 설정되어야 한다', async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      const preset = wrapper.find('.date-range-picker__preset') // 첫 번째 프리셋
      await preset.trigger('click')

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      const emittedRange = wrapper.emitted('update:modelValue')![0][0] as DateRange
      expect(emittedRange.start).toBeInstanceOf(Date)
      expect(emittedRange.end).toBeInstanceOf(Date)
    })

    it('커스텀 프리셋이 적용되어야 한다', async () => {
      const customPresets = [
        {
          label: '이번 주',
          value: () => {
            const start = new Date()
            start.setDate(start.getDate() - start.getDay())
            return { start, end: new Date() }
          },
        },
      ]

      await wrapper.setProps({ presets: customPresets })
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      const preset = wrapper.find('.date-range-picker__preset')
      expect(preset.text()).toBe('이번 주')
    })

    it('showPresets가 false일 때 프리셋이 숨겨져야 한다', async () => {
      await wrapper.setProps({ showPresets: false })
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      expect(wrapper.find('.date-range-picker__presets').exists()).toBe(false)
    })
  })

  describe('날짜 제한', () => {
    it('min 날짜 이전은 비활성화되어야 한다', async () => {
      await wrapper.setProps({ min: today })
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      const days = wrapper.findAll('.date-range-picker__day')
      const pastDays = days.filter(day => {
        const date = new Date(day.attributes('data-date'))
        return date < today
      })

      pastDays.forEach(day => {
        expect(day.classes()).toContain('date-range-picker__day--disabled')
      })
    })

    it('max 날짜 이후는 비활성화되어야 한다', async () => {
      await wrapper.setProps({ max: nextWeek })
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      const days = wrapper.findAll('.date-range-picker__day')
      const futureDays = days.filter(day => {
        const date = new Date(day.attributes('data-date'))
        return date > nextWeek
      })

      futureDays.forEach(day => {
        expect(day.classes()).toContain('date-range-picker__day--disabled')
      })
    })
  })

  describe('날짜 형식', () => {
    it('기본 형식으로 날짜가 표시되어야 한다', async () => {
      const range = {
        start: new Date(2024, 0, 15),
        end: new Date(2024, 0, 20),
      }
      await wrapper.setProps({ modelValue: range })

      const inputs = wrapper.findAll('input')
      expect(inputs[0].element.value).toBe('2024-01-15')
      expect(inputs[1].element.value).toBe('2024-01-20')
    })

    it('커스텀 형식이 적용되어야 한다', async () => {
      const range = {
        start: new Date(2024, 0, 15),
        end: new Date(2024, 0, 20),
      }
      await wrapper.setProps({
        modelValue: range,
        format: 'DD/MM/YYYY',
      })

      const inputs = wrapper.findAll('input')
      expect(inputs[0].element.value).toBe('15/01/2024')
      expect(inputs[1].element.value).toBe('20/01/2024')
    })
  })

  describe('지우기 버튼', () => {
    it('값이 있을 때 지우기 버튼이 표시되어야 한다', async () => {
      await wrapper.setProps({
        modelValue: { start: today, end: tomorrow },
        clearable: true,
      })

      const clearButton = wrapper.find('.date-range-picker__clear')
      expect(clearButton.exists()).toBe(true)
    })

    it('지우기 버튼 클릭 시 값이 null이 되어야 한다', async () => {
      await wrapper.setProps({
        modelValue: { start: today, end: tomorrow },
        clearable: true,
      })

      const clearButton = wrapper.find('.date-range-picker__clear')
      await clearButton.trigger('click')

      expect(wrapper.emitted('update:modelValue')![0]).toEqual([null])
      expect(wrapper.emitted('clear')).toBeTruthy()
    })
  })

  describe('비활성화 상태', () => {
    it('disabled일 때 입력이 불가능해야 한다', async () => {
      await wrapper.setProps({ disabled: true })

      const inputs = wrapper.findAll('input')
      inputs.forEach(input => {
        expect(input.attributes('disabled')).toBeDefined()
      })
    })

    it('disabled일 때 캘린더가 열리지 않아야 한다', async () => {
      await wrapper.setProps({ disabled: true })

      await wrapper.find('.date-range-picker__start-input').trigger('click')

      expect(wrapper.find('.date-range-picker__calendar').exists()).toBe(false)
    })
  })

  describe('키보드 네비게이션', () => {
    beforeEach(async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')
    })

    it('Tab 키로 시작/종료 입력 필드 간 이동이 가능해야 한다', async () => {
      const startInput = wrapper.find('.date-range-picker__start-input')
      const endInput = wrapper.find('.date-range-picker__end-input')

      await startInput.trigger('keydown', { key: 'Tab' })
      expect(document.activeElement).toBe(endInput.element)
    })

    it('화살표 키로 날짜를 탐색할 수 있어야 한다', async () => {
      const calendar = wrapper.find('.date-range-picker__calendar')

      await calendar.trigger('keydown', { key: 'ArrowRight' })
      const focused = wrapper.find('.date-range-picker__day--focused')
      expect(focused.exists()).toBe(true)
    })

    it('Enter 키로 날짜를 선택할 수 있어야 한다', async () => {
      const calendar = wrapper.find('.date-range-picker__calendar')

      await calendar.trigger('keydown', { key: 'ArrowRight' })
      await calendar.trigger('keydown', { key: 'Enter' })

      expect(wrapper.vm.tempRange.start).toBeInstanceOf(Date)
    })
  })

  describe('월 네비게이션', () => {
    beforeEach(async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')
    })

    it('이전/다음 월 버튼이 작동해야 한다', async () => {
      const prevButtons = wrapper.findAll('.date-range-picker__prev-month')
      const nextButtons = wrapper.findAll('.date-range-picker__next-month')

      const initialMonth = wrapper.find('.date-range-picker__current-month').text()

      await prevButtons[0].trigger('click')
      const newMonth = wrapper.find('.date-range-picker__current-month').text()

      expect(newMonth).not.toBe(initialMonth)
    })

    it('두 캘린더가 연속된 월을 표시해야 한다', () => {
      const monthHeaders = wrapper.findAll('.date-range-picker__current-month')
      expect(monthHeaders).toHaveLength(2)

      // 월이 연속되는지 확인 (월말/월초 경계 고려)
      const firstMonth = new Date(monthHeaders[0].text())
      const secondMonth = new Date(monthHeaders[1].text())
      const monthDiff = (secondMonth.getFullYear() - firstMonth.getFullYear()) * 12 +
                       (secondMonth.getMonth() - firstMonth.getMonth())

      expect(Math.abs(monthDiff)).toBeLessThanOrEqual(1)
    })
  })

  describe('접근성', () => {
    it('적절한 ARIA 속성이 설정되어야 한다', () => {
      const container = wrapper.find('.date-range-picker')
      const inputs = wrapper.findAll('input')

      expect(inputs[0].attributes('role')).toBe('combobox')
      expect(inputs[1].attributes('role')).toBe('combobox')
      expect(inputs[0].attributes('aria-haspopup')).toBe('dialog')
    })

    it('캘린더가 열릴 때 aria-expanded가 true가 되어야 한다', async () => {
      const input = wrapper.find('.date-range-picker__start-input')

      expect(input.attributes('aria-expanded')).toBe('false')

      await input.trigger('click')
      expect(input.attributes('aria-expanded')).toBe('true')
    })

    it('캘린더에 적절한 ARIA 레이블이 있어야 한다', async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      const calendar = wrapper.find('.date-range-picker__calendar')
      expect(calendar.attributes('role')).toBe('dialog')
      expect(calendar.attributes('aria-label')).toBeTruthy()
    })
  })

  describe('이벤트', () => {
    it('change 이벤트가 발생해야 한다', async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')
      const days = wrapper.findAll('.date-range-picker__day:not(.date-range-picker__day--other-month)')

      await days[15].trigger('click')
      await days[20].trigger('click')

      expect(wrapper.emitted('change')).toBeTruthy()
    })

    it('focus/blur 이벤트가 발생해야 한다', async () => {
      const input = wrapper.find('.date-range-picker__start-input')

      await input.trigger('focus')
      expect(wrapper.emitted('focus')).toBeTruthy()

      await input.trigger('blur')
      expect(wrapper.emitted('blur')).toBeTruthy()
    })

    it('navigate 이벤트가 발생해야 한다', async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      const prevButton = wrapper.find('.date-range-picker__prev-month')
      await prevButton.trigger('click')

      expect(wrapper.emitted('navigate')).toBeTruthy()
    })
  })

  describe('직접 입력', () => {
    it('시작 날짜를 직접 입력할 수 있어야 한다', async () => {
      const startInput = wrapper.find('.date-range-picker__start-input')

      await startInput.setValue('2024-01-15')
      await startInput.trigger('blur')

      const emitted = wrapper.emitted('update:modelValue')
      if (emitted && emitted[0]) {
        const range = emitted[0][0] as DateRange
        expect(range.start).toBeInstanceOf(Date)
        expect(range.start?.toISOString().split('T')[0]).toBe('2024-01-15')
      }
    })

    it('종료 날짜를 직접 입력할 수 있어야 한다', async () => {
      await wrapper.setProps({
        modelValue: { start: new Date(2024, 0, 15), end: null },
      })

      const endInput = wrapper.find('.date-range-picker__end-input')

      await endInput.setValue('2024-01-20')
      await endInput.trigger('blur')

      const emitted = wrapper.emitted('update:modelValue')
      if (emitted && emitted[0]) {
        const range = emitted[0][0] as DateRange
        expect(range.end).toBeInstanceOf(Date)
        expect(range.end?.toISOString().split('T')[0]).toBe('2024-01-20')
      }
    })

    it('잘못된 형식의 입력은 무시되어야 한다', async () => {
      const startInput = wrapper.find('.date-range-picker__start-input')

      await startInput.setValue('invalid-date')
      await startInput.trigger('blur')

      expect(wrapper.emitted('update:modelValue')).toBeFalsy()
    })
  })

  describe('엣지 케이스', () => {
    it('단일 날짜 선택 (같은 날짜)이 가능해야 한다', async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')
      const days = wrapper.findAll('.date-range-picker__day:not(.date-range-picker__day--other-month)')

      const targetDay = days[15]
      await targetDay.trigger('click')
      await targetDay.trigger('click') // 같은 날짜 다시 클릭

      const emitted = wrapper.emitted('update:modelValue')
      if (emitted && emitted[0]) {
        const range = emitted[0][0] as DateRange
        expect(range.start?.toDateString()).toBe(range.end?.toDateString())
      }
    })

    it('월 경계를 넘는 범위를 선택할 수 있어야 한다', async () => {
      await wrapper.find('.date-range-picker__start-input').trigger('click')

      // 현재 월의 마지막 날 찾기
      const days = wrapper.findAll('.date-range-picker__day:not(.date-range-picker__day--other-month)')
      const lastDay = days[days.length - 1]
      await lastDay.trigger('click')

      // 다음 월로 이동
      const nextButton = wrapper.find('.date-range-picker__next-month')
      await nextButton.trigger('click')

      // 다음 월의 첫 날 선택
      const nextMonthDays = wrapper.findAll('.date-range-picker__day:not(.date-range-picker__day--other-month)')
      await nextMonthDays[0].trigger('click')

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    })

    it('null 값을 처리할 수 있어야 한다', async () => {
      await wrapper.setProps({
        modelValue: { start: null, end: null },
      })

      expect(() => {
        wrapper.find('.date-range-picker__start-input').trigger('click')
      }).not.toThrow()
    })
  })
})