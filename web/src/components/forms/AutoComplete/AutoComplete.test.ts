import { beforeEach, describe, expect, it, vi } from 'vitest'
import { VueWrapper, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import AutoComplete from './AutoComplete.vue'

describe('AutoComplete 컴포넌트', () => {
  let wrapper: VueWrapper<any>
  const mockSuggestions = ['Apple', 'Banana', 'Cherry', 'Date', 'Elderberry']

  beforeEach(() => {
    vi.useFakeTimers()
    wrapper = mount(AutoComplete, {
      props: {
        modelValue: '',
        suggestions: mockSuggestions,
      },
    })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('기본 렌더링', () => {
    it('컴포넌트가 올바르게 렌더링되어야 한다', () => {
      expect(wrapper.exists()).toBe(true)
      expect(wrapper.find('.autocomplete').exists()).toBe(true)
    })

    it('입력 필드가 표시되어야 한다', () => {
      const input = wrapper.find('.autocomplete__input')
      expect(input.exists()).toBe(true)
      expect(input.attributes('type')).toBe('text')
    })

    it('플레이스홀더가 표시되어야 한다', () => {
      const input = wrapper.find('.autocomplete__input')
      expect(input.attributes('placeholder')).toBe('Type to search...')
    })

    it('커스텀 플레이스홀더가 적용되어야 한다', async () => {
      await wrapper.setProps({ placeholder: '검색어를 입력하세요' })
      const input = wrapper.find('.autocomplete__input')
      expect(input.attributes('placeholder')).toBe('검색어를 입력하세요')
    })
  })

  describe('제안 목록 동작', () => {
    it('최소 길이 이상 입력 시 제안 목록이 표시되어야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('a')
      vi.runAllTimers()
      await nextTick()

      expect(wrapper.find('.autocomplete__suggestions').exists()).toBe(true)
    })

    it('minLength 미만 입력 시 제안 목록이 표시되지 않아야 한다', async () => {
      await wrapper.setProps({ minLength: 3 })
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('ab')
      vi.runAllTimers()
      await nextTick()

      expect(wrapper.find('.autocomplete__suggestions').exists()).toBe(false)
    })

    it('입력값과 일치하는 제안만 표시되어야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')
      vi.runAllTimers()
      await nextTick()

      const suggestions = wrapper.findAll('.autocomplete__suggestion')
      expect(suggestions).toHaveLength(1)
      expect(suggestions[0].text()).toBe('Apple')
    })

    it('대소문자 구분 없이 검색되어야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('APPLE')
      vi.runAllTimers()
      await nextTick()

      const suggestions = wrapper.findAll('.autocomplete__suggestion')
      expect(suggestions).toHaveLength(1)
      expect(suggestions[0].text()).toBe('Apple')
    })

    it('caseSensitive가 true일 때 대소문자를 구분해야 한다', async () => {
      await wrapper.setProps({ caseSensitive: true })
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('APPLE')
      vi.runAllTimers()
      await nextTick()

      const suggestions = wrapper.findAll('.autocomplete__suggestion')
      expect(suggestions).toHaveLength(0)
    })
  })

  describe('하이라이팅', () => {
    it('매칭된 텍스트가 하이라이트되어야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')
      vi.runAllTimers()
      await nextTick()

      const highlight = wrapper.find('.autocomplete__highlight')
      expect(highlight.exists()).toBe(true)
      expect(highlight.text()).toBe('App')
    })

    it('highlightMatches가 false일 때 하이라이트가 없어야 한다', async () => {
      await wrapper.setProps({ highlightMatches: false })
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')
      vi.runAllTimers()
      await nextTick()

      expect(wrapper.find('.autocomplete__highlight').exists()).toBe(false)
    })
  })

  describe('선택 기능', () => {
    it('제안 항목 클릭 시 선택되어야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')
      vi.runAllTimers()
      await nextTick()

      const suggestion = wrapper.find('.autocomplete__suggestion')
      await suggestion.trigger('click')

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0]).toEqual(['Apple'])
      expect(wrapper.emitted('select')).toBeTruthy()
      expect(wrapper.emitted('select')![0]).toEqual(['Apple'])
    })

    it('선택 후 제안 목록이 닫혀야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')
      vi.runAllTimers()
      await nextTick()

      await wrapper.find('.autocomplete__suggestion').trigger('click')
      await nextTick()

      expect(wrapper.find('.autocomplete__suggestions').exists()).toBe(false)
    })
  })

  describe('키보드 네비게이션', () => {
    beforeEach(async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('a')
      vi.runAllTimers()
      await nextTick()
    })

    it('ArrowDown 키로 다음 항목으로 이동해야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.trigger('keydown', { key: 'ArrowDown' })

      const highlighted = wrapper.find('.autocomplete__suggestion--highlighted')
      expect(highlighted.exists()).toBe(true)
      expect(highlighted.text()).toBe('Apple')
    })

    it('ArrowUp 키로 이전 항목으로 이동해야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.trigger('keydown', { key: 'ArrowDown' })
      await input.trigger('keydown', { key: 'ArrowDown' })
      await input.trigger('keydown', { key: 'ArrowUp' })

      const highlighted = wrapper.find('.autocomplete__suggestion--highlighted')
      expect(highlighted.text()).toBe('Apple')
    })

    it('Enter 키로 하이라이트된 항목을 선택해야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.trigger('keydown', { key: 'ArrowDown' })
      await input.trigger('keydown', { key: 'Enter' })

      expect(wrapper.emitted('update:modelValue')![0]).toEqual(['Apple'])
      expect(wrapper.emitted('select')![0]).toEqual(['Apple'])
    })

    it('Escape 키로 제안 목록이 닫혀야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.trigger('keydown', { key: 'Escape' })
      await nextTick()

      expect(wrapper.find('.autocomplete__suggestions').exists()).toBe(false)
    })
  })

  describe('디바운싱', () => {
    it('기본 지연 시간이 적용되어야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')

      // 지연 시간 전에는 제안 목록이 표시되지 않음
      expect(wrapper.find('.autocomplete__suggestions').exists()).toBe(false)

      // 지연 시간 후 표시됨
      vi.runAllTimers()
      await nextTick()
      expect(wrapper.find('.autocomplete__suggestions').exists()).toBe(true)
    })

    it('커스텀 지연 시간이 적용되어야 한다', async () => {
      await wrapper.setProps({ delay: 500 })
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')

      // 300ms 후에도 표시되지 않음
      vi.advanceTimersByTime(300)
      await nextTick()
      expect(wrapper.find('.autocomplete__suggestions').exists()).toBe(false)

      // 500ms 후 표시됨
      vi.advanceTimersByTime(200)
      await nextTick()
      expect(wrapper.find('.autocomplete__suggestions').exists()).toBe(true)
    })
  })

  describe('로딩 상태', () => {
    it('로딩 인디케이터가 표시되어야 한다', async () => {
      await wrapper.setProps({ loading: true })
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')

      expect(wrapper.find('.autocomplete__loading').exists()).toBe(true)
    })

    it('로딩 중에는 제안 목록이 표시되지 않아야 한다', async () => {
      await wrapper.setProps({ loading: true })
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')
      vi.runAllTimers()
      await nextTick()

      expect(wrapper.find('.autocomplete__suggestions').exists()).toBe(false)
    })
  })

  describe('지우기 버튼', () => {
    it('값이 있을 때 지우기 버튼이 표시되어야 한다', async () => {
      await wrapper.setProps({
        modelValue: 'Apple',
        clearable: true,
      })

      expect(wrapper.find('.autocomplete__clear').exists()).toBe(true)
    })

    it('지우기 버튼 클릭 시 값이 초기화되어야 한다', async () => {
      await wrapper.setProps({
        modelValue: 'Apple',
        clearable: true,
      })

      await wrapper.find('.autocomplete__clear').trigger('click')

      expect(wrapper.emitted('update:modelValue')![0]).toEqual([''])
      expect(wrapper.emitted('clear')).toBeTruthy()
    })

    it('clearable이 false일 때 지우기 버튼이 표시되지 않아야 한다', async () => {
      await wrapper.setProps({
        modelValue: 'Apple',
        clearable: false,
      })

      expect(wrapper.find('.autocomplete__clear').exists()).toBe(false)
    })
  })

  describe('새 항목 생성', () => {
    it('showCreateOption이 true일 때 생성 옵션이 표시되어야 한다', async () => {
      await wrapper.setProps({ showCreateOption: true })

      const input = wrapper.find('.autocomplete__input')
      await input.setValue('NewItem')
      vi.runAllTimers()
      await nextTick()

      const createOption = wrapper.find('.autocomplete__create-option')
      expect(createOption.exists()).toBe(true)
      expect(createOption.text()).toContain('Create')
      expect(createOption.text()).toContain('NewItem')
    })

    it('커스텀 생성 텍스트가 표시되어야 한다', async () => {
      await wrapper.setProps({
        showCreateOption: true,
        createOptionText: '새로 만들기',
      })

      const input = wrapper.find('.autocomplete__input')
      await input.setValue('NewItem')
      vi.runAllTimers()
      await nextTick()

      const createOption = wrapper.find('.autocomplete__create-option')
      expect(createOption.text()).toContain('새로 만들기')
    })
  })

  describe('커스텀 필터링', () => {
    it('커스텀 필터 메서드가 적용되어야 한다', async () => {
      const customFilter = (query: string, item: any) => {
        return item.startsWith(query.toUpperCase())
      }

      await wrapper.setProps({
        filterMethod: customFilter,
        suggestions: ['APPLE', 'APRICOT', 'BANANA'],
      })

      const input = wrapper.find('.autocomplete__input')
      await input.setValue('AP')
      vi.runAllTimers()
      await nextTick()

      const suggestions = wrapper.findAll('.autocomplete__suggestion')
      expect(suggestions).toHaveLength(2)
      expect(suggestions[0].text()).toBe('APPLE')
      expect(suggestions[1].text()).toBe('APRICOT')
    })
  })

  describe('객체 배열 처리', () => {
    const objectSuggestions = [
      { id: 1, name: 'Apple', category: 'Fruit' },
      { id: 2, name: 'Banana', category: 'Fruit' },
      { id: 3, name: 'Carrot', category: 'Vegetable' },
    ]

    it('labelKey로 라벨을 추출해야 한다', async () => {
      await wrapper.setProps({
        suggestions: objectSuggestions,
        labelKey: 'name',
        valueKey: 'id',
      })

      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')
      vi.runAllTimers()
      await nextTick()

      const suggestion = wrapper.find('.autocomplete__suggestion')
      expect(suggestion.text()).toBe('Apple')
    })

    it('labelKey 함수로 라벨을 추출해야 한다', async () => {
      await wrapper.setProps({
        suggestions: objectSuggestions,
        labelKey: (item: any) => `${item.name} (${item.category})`,
        valueKey: 'id',
      })

      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')
      vi.runAllTimers()
      await nextTick()

      const suggestion = wrapper.find('.autocomplete__suggestion')
      expect(suggestion.text()).toBe('Apple (Fruit)')
    })
  })

  describe('비활성화 상태', () => {
    it('disabled일 때 입력이 불가능해야 한다', async () => {
      await wrapper.setProps({ disabled: true })

      const input = wrapper.find('.autocomplete__input')
      expect(input.attributes('disabled')).toBeDefined()
    })

    it('disabled일 때 지우기 버튼이 표시되지 않아야 한다', async () => {
      await wrapper.setProps({
        disabled: true,
        modelValue: 'Apple',
        clearable: true,
      })

      expect(wrapper.find('.autocomplete__clear').exists()).toBe(false)
    })
  })

  describe('접근성', () => {
    it('적절한 ARIA 속성이 설정되어야 한다', () => {
      const input = wrapper.find('.autocomplete__input')

      expect(input.attributes('role')).toBe('combobox')
      expect(input.attributes('aria-autocomplete')).toBe('list')
      expect(input.attributes('aria-expanded')).toBe('false')
    })

    it('제안 목록이 열릴 때 aria-expanded가 true가 되어야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')
      vi.runAllTimers()
      await nextTick()

      expect(input.attributes('aria-expanded')).toBe('true')
    })

    it('제안 목록에 적절한 role이 설정되어야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('app')
      vi.runAllTimers()
      await nextTick()

      const suggestions = wrapper.find('.autocomplete__suggestions')
      expect(suggestions.attributes('role')).toBe('listbox')
    })
  })

  describe('이벤트 발생', () => {
    it('input 이벤트가 발생해야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('test')

      expect(wrapper.emitted('input')).toBeTruthy()
      expect(wrapper.emitted('input')![0]).toEqual(['test'])
    })

    it('search 이벤트가 발생해야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')
      await input.setValue('test')
      vi.runAllTimers()

      expect(wrapper.emitted('search')).toBeTruthy()
      expect(wrapper.emitted('search')![0]).toEqual(['test'])
    })

    it('focus/blur 이벤트가 발생해야 한다', async () => {
      const input = wrapper.find('.autocomplete__input')

      await input.trigger('focus')
      expect(wrapper.emitted('focus')).toBeTruthy()

      await input.trigger('blur')
      expect(wrapper.emitted('blur')).toBeTruthy()
    })
  })

  describe('엣지 케이스', () => {
    it('빈 제안 목록을 처리할 수 있어야 한다', async () => {
      await wrapper.setProps({ suggestions: [] })

      const input = wrapper.find('.autocomplete__input')
      await input.setValue('test')
      vi.runAllTimers()
      await nextTick()

      const emptyMessage = wrapper.find('.autocomplete__empty')
      expect(emptyMessage.exists()).toBe(true)
      expect(emptyMessage.text()).toBe('No results found')
    })

    it('maxSuggestions가 적용되어야 한다', async () => {
      await wrapper.setProps({ maxSuggestions: 2 })

      const input = wrapper.find('.autocomplete__input')
      await input.setValue('e') // 여러 결과가 나올 검색어
      vi.runAllTimers()
      await nextTick()

      const suggestions = wrapper.findAll('.autocomplete__suggestion')
      expect(suggestions.length).toBeLessThanOrEqual(2)
    })
  })
})