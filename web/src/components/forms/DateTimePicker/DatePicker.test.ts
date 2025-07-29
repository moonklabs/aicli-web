import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import DatePicker from './DatePicker.vue'

describe('DatePicker 컴포넌트', () => {
  let wrapper: VueWrapper<any>
  const today = new Date()
  const tomorrow = new Date(today)
  tomorrow.setDate(tomorrow.getDate() + 1)
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  beforeEach(() => {
    wrapper = mount(DatePicker, {
      props: {
        modelValue: null,
      },
    })
  })

  describe('기본 렌더링', () => {
    it('컴포넌트가 올바르게 렌더링되어야 한다', () => {
      expect(wrapper.exists()).toBe(true)
      expect(wrapper.find('.date-picker').exists()).toBe(true)
    })

    it('입력 필드가 표시되어야 한다', () => {
      const input = wrapper.find('.date-picker__input')
      expect(input.exists()).toBe(true)
    })

    it('플레이스홀더가 표시되어야 한다', () => {
      const input = wrapper.find('.date-picker__input')
      expect(input.attributes('placeholder')).toBe('날짜를 선택하세요')
    })

    it('커스텀 플레이스홀더가 적용되어야 한다', async () => {
      await wrapper.setProps({ placeholder: 'Select date' })
      const input = wrapper.find('.date-picker__input')
      expect(input.attributes('placeholder')).toBe('Select date')
    })
  })

  describe('캘린더 동작', () => {
    it('입력 필드 클릭 시 캘린더가 열려야 한다', async () => {
      const input = wrapper.find('.date-picker__input')
      await input.trigger('click')
      
      expect(wrapper.find('.date-picker__calendar').exists()).toBe(true)
    })

    it('캘린더 아이콘 클릭 시 캘린더가 열려야 한다', async () => {
      const icon = wrapper.find('.date-picker__icon')
      await icon.trigger('click')
      
      expect(wrapper.find('.date-picker__calendar').exists()).toBe(true)
    })

    it('ESC 키로 캘린더가 닫혀야 한다', async () => {
      await wrapper.find('.date-picker__input').trigger('click')
      expect(wrapper.find('.date-picker__calendar').exists()).toBe(true)
      
      await wrapper.find('.date-picker__input').trigger('keydown', { key: 'Escape' })
      await nextTick()
      
      expect(wrapper.find('.date-picker__calendar').exists()).toBe(false)
    })

    it('외부 클릭 시 캘린더가 닫혀야 한다', async () => {
      await wrapper.find('.date-picker__input').trigger('click')
      expect(wrapper.find('.date-picker__calendar').exists()).toBe(true)
      
      // 외부 클릭 시뮬레이션
      document.body.click()
      await nextTick()
      
      expect(wrapper.find('.date-picker__calendar').exists()).toBe(false)
    })
  })

  describe('날짜 선택', () => {
    beforeEach(async () => {
      await wrapper.find('.date-picker__input').trigger('click')
    })

    it('날짜 셀이 올바르게 렌더링되어야 한다', () => {
      const days = wrapper.findAll('.date-picker__day')
      expect(days.length).toBeGreaterThan(0)
    })

    it('날짜 클릭 시 선택되어야 한다', async () => {
      const days = wrapper.findAll('.date-picker__day:not(.date-picker__day--other-month)')
      const targetDay = days[15] // 중간쯤 날짜 선택
      
      await targetDay.trigger('click')
      
      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0][0]).toBeInstanceOf(Date)
      expect(wrapper.emitted('change')).toBeTruthy()
    })

    it('선택된 날짜가 하이라이트되어야 한다', async () => {
      const selectedDate = new Date()
      await wrapper.setProps({ modelValue: selectedDate })
      await wrapper.find('.date-picker__input').trigger('click')
      
      const selectedDay = wrapper.find('.date-picker__day--selected')
      expect(selectedDay.exists()).toBe(true)
    })

    it('오늘 날짜가 표시되어야 한다', () => {
      const todayCell = wrapper.find('.date-picker__day--today')
      expect(todayCell.exists()).toBe(true)
    })
  })

  describe('월/년 네비게이션', () => {
    beforeEach(async () => {
      await wrapper.find('.date-picker__input').trigger('click')
    })

    it('이전/다음 월 버튼이 작동해야 한다', async () => {
      const currentMonth = wrapper.find('.date-picker__current-month').text()
      
      const prevButton = wrapper.find('.date-picker__prev-month')
      await prevButton.trigger('click')
      
      const newMonth = wrapper.find('.date-picker__current-month').text()
      expect(newMonth).not.toBe(currentMonth)
    })

    it('월 선택 드롭다운이 작동해야 한다', async () => {
      const monthSelect = wrapper.find('.date-picker__month-select')
      await monthSelect.trigger('click')
      
      const monthOptions = wrapper.findAll('.date-picker__month-option')
      expect(monthOptions).toHaveLength(12)
      
      await monthOptions[5].trigger('click') // 6월 선택
      expect(wrapper.emitted('navigate')).toBeTruthy()
    })

    it('년도 선택 드롭다운이 작동해야 한다', async () => {
      const yearSelect = wrapper.find('.date-picker__year-select')
      await yearSelect.trigger('click')
      
      const yearOptions = wrapper.findAll('.date-picker__year-option')
      expect(yearOptions.length).toBeGreaterThan(0)
    })
  })

  describe('날짜 제한', () => {
    it('min 날짜 이전은 비활성화되어야 한다', async () => {
      await wrapper.setProps({ 
        min: today,
        modelValue: today 
      })
      await wrapper.find('.date-picker__input').trigger('click')
      
      // 어제 날짜를 찾아서 비활성화 확인
      const days = wrapper.findAll('.date-picker__day')
      const yesterdayCell = days.find(day => {
        const date = new Date(day.attributes('data-date'))
        return date.toDateString() === yesterday.toDateString()
      })
      
      if (yesterdayCell) {
        expect(yesterdayCell.classes()).toContain('date-picker__day--disabled')
      }
    })

    it('max 날짜 이후는 비활성화되어야 한다', async () => {
      await wrapper.setProps({ 
        max: today,
        modelValue: today 
      })
      await wrapper.find('.date-picker__input').trigger('click')
      
      const days = wrapper.findAll('.date-picker__day')
      const tomorrowCell = days.find(day => {
        const date = new Date(day.attributes('data-date'))
        return date.toDateString() === tomorrow.toDateString()
      })
      
      if (tomorrowCell) {
        expect(tomorrowCell.classes()).toContain('date-picker__day--disabled')
      }
    })

    it('비활성화된 날짜는 선택할 수 없어야 한다', async () => {
      await wrapper.setProps({ min: today })
      await wrapper.find('.date-picker__input').trigger('click')
      
      const disabledDay = wrapper.find('.date-picker__day--disabled')
      if (disabledDay.exists()) {
        await disabledDay.trigger('click')
        expect(wrapper.emitted('update:modelValue')).toBeFalsy()
      }
    })
  })

  describe('오늘 버튼', () => {
    it('오늘 버튼이 표시되어야 한다', async () => {
      await wrapper.find('.date-picker__input').trigger('click')
      
      const todayButton = wrapper.find('.date-picker__today-button')
      expect(todayButton.exists()).toBe(true)
      expect(todayButton.text()).toBe('오늘')
    })

    it('오늘 버튼 클릭 시 오늘 날짜가 선택되어야 한다', async () => {
      await wrapper.find('.date-picker__input').trigger('click')
      
      const todayButton = wrapper.find('.date-picker__today-button')
      await todayButton.trigger('click')
      
      const emitted = wrapper.emitted('update:modelValue')
      expect(emitted).toBeTruthy()
      const selectedDate = emitted![0][0] as Date
      expect(selectedDate.toDateString()).toBe(today.toDateString())
    })

    it('커스텀 오늘 버튼 텍스트가 적용되어야 한다', async () => {
      await wrapper.setProps({ todayButtonText: 'Today' })
      await wrapper.find('.date-picker__input').trigger('click')
      
      const todayButton = wrapper.find('.date-picker__today-button')
      expect(todayButton.text()).toBe('Today')
    })
  })

  describe('날짜 형식', () => {
    it('기본 형식으로 날짜가 표시되어야 한다', async () => {
      const date = new Date(2024, 0, 15) // 2024-01-15
      await wrapper.setProps({ modelValue: date })
      
      const input = wrapper.find('.date-picker__input')
      expect(input.element.value).toBe('2024-01-15')
    })

    it('커스텀 형식이 적용되어야 한다', async () => {
      const date = new Date(2024, 0, 15)
      await wrapper.setProps({ 
        modelValue: date,
        format: 'DD/MM/YYYY'
      })
      
      const input = wrapper.find('.date-picker__input')
      expect(input.element.value).toBe('15/01/2024')
    })

    it('한국어 형식이 적용되어야 한다', async () => {
      const date = new Date(2024, 0, 15)
      await wrapper.setProps({ 
        modelValue: date,
        format: 'YYYY년 MM월 DD일'
      })
      
      const input = wrapper.find('.date-picker__input')
      expect(input.element.value).toBe('2024년 01월 15일')
    })
  })

  describe('로케일', () => {
    it('한국어 로케일이 기본으로 적용되어야 한다', async () => {
      await wrapper.find('.date-picker__input').trigger('click')
      
      const weekdays = wrapper.findAll('.date-picker__weekday')
      expect(weekdays[0].text()).toBe('일') // 일요일
      expect(weekdays[1].text()).toBe('월') // 월요일
    })

    it('영어 로케일이 적용되어야 한다', async () => {
      await wrapper.setProps({ locale: 'en' })
      await wrapper.find('.date-picker__input').trigger('click')
      
      const weekdays = wrapper.findAll('.date-picker__weekday')
      expect(weekdays[0].text()).toBe('Su') // Sunday
      expect(weekdays[1].text()).toBe('Mo') // Monday
    })
  })

  describe('주 시작일', () => {
    it('기본값은 일요일부터 시작해야 한다', async () => {
      await wrapper.find('.date-picker__input').trigger('click')
      
      const weekdays = wrapper.findAll('.date-picker__weekday')
      expect(weekdays[0].text()).toBe('일')
    })

    it('월요일부터 시작하도록 설정할 수 있어야 한다', async () => {
      await wrapper.setProps({ firstDayOfWeek: 1 })
      await wrapper.find('.date-picker__input').trigger('click')
      
      const weekdays = wrapper.findAll('.date-picker__weekday')
      expect(weekdays[0].text()).toBe('월')
      expect(weekdays[6].text()).toBe('일')
    })
  })

  describe('지우기 버튼', () => {
    it('값이 있을 때 지우기 버튼이 표시되어야 한다', async () => {
      await wrapper.setProps({ 
        modelValue: new Date(),
        clearable: true 
      })
      
      const clearButton = wrapper.find('.date-picker__clear')
      expect(clearButton.exists()).toBe(true)
    })

    it('지우기 버튼 클릭 시 값이 null이 되어야 한다', async () => {
      await wrapper.setProps({ 
        modelValue: new Date(),
        clearable: true 
      })
      
      const clearButton = wrapper.find('.date-picker__clear')
      await clearButton.trigger('click')
      
      expect(wrapper.emitted('update:modelValue')![0]).toEqual([null])
      expect(wrapper.emitted('clear')).toBeTruthy()
    })

    it('clearable이 false일 때 지우기 버튼이 표시되지 않아야 한다', async () => {
      await wrapper.setProps({ 
        modelValue: new Date(),
        clearable: false 
      })
      
      expect(wrapper.find('.date-picker__clear').exists()).toBe(false)
    })
  })

  describe('비활성화 상태', () => {
    it('disabled일 때 입력이 불가능해야 한다', async () => {
      await wrapper.setProps({ disabled: true })
      
      const input = wrapper.find('.date-picker__input')
      expect(input.attributes('disabled')).toBeDefined()
    })

    it('disabled일 때 캘린더가 열리지 않아야 한다', async () => {
      await wrapper.setProps({ disabled: true })
      
      const input = wrapper.find('.date-picker__input')
      await input.trigger('click')
      
      expect(wrapper.find('.date-picker__calendar').exists()).toBe(false)
    })

    it('disabled 클래스가 추가되어야 한다', async () => {
      await wrapper.setProps({ disabled: true })
      
      expect(wrapper.find('.date-picker--disabled').exists()).toBe(true)
    })
  })

  describe('키보드 네비게이션', () => {
    beforeEach(async () => {
      await wrapper.find('.date-picker__input').trigger('click')
    })

    it('화살표 키로 날짜를 탐색할 수 있어야 한다', async () => {
      const calendar = wrapper.find('.date-picker__calendar')
      
      await calendar.trigger('keydown', { key: 'ArrowRight' })
      let focused = wrapper.find('.date-picker__day--focused')
      expect(focused.exists()).toBe(true)
      
      await calendar.trigger('keydown', { key: 'ArrowDown' })
      focused = wrapper.find('.date-picker__day--focused')
      expect(focused.exists()).toBe(true)
    })

    it('Enter 키로 포커스된 날짜를 선택할 수 있어야 한다', async () => {
      const calendar = wrapper.find('.date-picker__calendar')
      
      await calendar.trigger('keydown', { key: 'ArrowRight' })
      await calendar.trigger('keydown', { key: 'Enter' })
      
      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    })

    it('Home/End 키로 주의 시작/끝으로 이동해야 한다', async () => {
      const calendar = wrapper.find('.date-picker__calendar')
      
      await calendar.trigger('keydown', { key: 'Home' })
      let focused = wrapper.find('.date-picker__day--focused')
      expect(focused.exists()).toBe(true)
      
      await calendar.trigger('keydown', { key: 'End' })
      focused = wrapper.find('.date-picker__day--focused')
      expect(focused.exists()).toBe(true)
    })

    it('PageUp/PageDown으로 월을 이동해야 한다', async () => {
      const calendar = wrapper.find('.date-picker__calendar')
      const initialMonth = wrapper.find('.date-picker__current-month').text()
      
      await calendar.trigger('keydown', { key: 'PageDown' })
      const nextMonth = wrapper.find('.date-picker__current-month').text()
      expect(nextMonth).not.toBe(initialMonth)
    })
  })

  describe('접근성', () => {
    it('적절한 ARIA 속성이 설정되어야 한다', () => {
      const input = wrapper.find('.date-picker__input')
      
      expect(input.attributes('role')).toBe('combobox')
      expect(input.attributes('aria-expanded')).toBe('false')
      expect(input.attributes('aria-haspopup')).toBe('dialog')
    })

    it('캘린더가 열릴 때 aria-expanded가 true가 되어야 한다', async () => {
      await wrapper.find('.date-picker__input').trigger('click')
      
      const input = wrapper.find('.date-picker__input')
      expect(input.attributes('aria-expanded')).toBe('true')
    })

    it('캘린더에 적절한 ARIA 레이블이 있어야 한다', async () => {
      await wrapper.find('.date-picker__input').trigger('click')
      
      const calendar = wrapper.find('.date-picker__calendar')
      expect(calendar.attributes('role')).toBe('dialog')
      expect(calendar.attributes('aria-label')).toBeTruthy()
    })

    it('커스텀 ARIA 레이블이 적용되어야 한다', async () => {
      await wrapper.setProps({ 
        ariaLabel: '생년월일 선택',
        calendarAriaLabel: '생년월일 선택 캘린더'
      })
      
      const input = wrapper.find('.date-picker__input')
      expect(input.attributes('aria-label')).toBe('생년월일 선택')
      
      await input.trigger('click')
      const calendar = wrapper.find('.date-picker__calendar')
      expect(calendar.attributes('aria-label')).toBe('생년월일 선택 캘린더')
    })
  })

  describe('이벤트', () => {
    it('focus/blur 이벤트가 발생해야 한다', async () => {
      const input = wrapper.find('.date-picker__input')
      
      await input.trigger('focus')
      expect(wrapper.emitted('focus')).toBeTruthy()
      
      await input.trigger('blur')
      expect(wrapper.emitted('blur')).toBeTruthy()
    })

    it('navigate 이벤트가 발생해야 한다', async () => {
      await wrapper.find('.date-picker__input').trigger('click')
      
      const prevButton = wrapper.find('.date-picker__prev-month')
      await prevButton.trigger('click')
      
      expect(wrapper.emitted('navigate')).toBeTruthy()
    })
  })

  describe('엣지 케이스', () => {
    it('유효하지 않은 날짜를 처리할 수 있어야 한다', async () => {
      await wrapper.setProps({ modelValue: new Date('invalid') })
      
      const input = wrapper.find('.date-picker__input')
      expect(input.element.value).toBe('')
    })

    it('윤년을 올바르게 처리해야 한다', async () => {
      // 2024년은 윤년
      const leapDate = new Date(2024, 1, 29) // 2024-02-29
      await wrapper.setProps({ modelValue: leapDate })
      await wrapper.find('.date-picker__input').trigger('click')
      
      const selectedDay = wrapper.find('.date-picker__day--selected')
      expect(selectedDay.exists()).toBe(true)
      expect(selectedDay.text()).toBe('29')
    })

    it('매우 오래된/미래의 날짜를 처리할 수 있어야 한다', async () => {
      const oldDate = new Date(1900, 0, 1)
      const futureDate = new Date(2100, 11, 31)
      
      await wrapper.setProps({ modelValue: oldDate })
      expect(() => wrapper.find('.date-picker__input').trigger('click')).not.toThrow()
      
      await wrapper.setProps({ modelValue: futureDate })
      expect(() => wrapper.find('.date-picker__input').trigger('click')).not.toThrow()
    })
  })
})