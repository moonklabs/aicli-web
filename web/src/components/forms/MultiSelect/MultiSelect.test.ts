import { beforeEach, describe, expect, it, vi } from 'vitest'
import { VueWrapper, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import MultiSelect from './MultiSelect.vue'

describe('MultiSelect 컴포넌트', () => {
  let wrapper: VueWrapper<any>
  const mockOptions = [
    { label: 'Option 1', value: '1' },
    { label: 'Option 2', value: '2' },
    { label: 'Option 3', value: '3' },
    { label: 'Option 4', value: '4' },
  ]

  beforeEach(() => {
    wrapper = mount(MultiSelect, {
      props: {
        modelValue: [],
        options: mockOptions,
      },
    })
  })

  describe('기본 렌더링', () => {
    it('컴포넌트가 올바르게 렌더링되어야 한다', () => {
      expect(wrapper.exists()).toBe(true)
      expect(wrapper.find('.multi-select').exists()).toBe(true)
    })

    it('플레이스홀더가 표시되어야 한다', () => {
      const placeholder = wrapper.find('.multi-select__placeholder')
      expect(placeholder.exists()).toBe(true)
      expect(placeholder.text()).toBe('Select options...')
    })

    it('커스텀 플레이스홀더가 표시되어야 한다', async () => {
      await wrapper.setProps({ placeholder: '항목을 선택하세요' })
      const placeholder = wrapper.find('.multi-select__placeholder')
      expect(placeholder.text()).toBe('항목을 선택하세요')
    })
  })

  describe('드롭다운 동작', () => {
    it('클릭 시 드롭다운이 열려야 한다', async () => {
      const trigger = wrapper.find('.multi-select__trigger')
      await trigger.trigger('click')

      expect(wrapper.find('.multi-select__dropdown').exists()).toBe(true)
      expect(wrapper.find('.multi-select__dropdown').isVisible()).toBe(true)
    })

    it('드롭다운이 열릴 때 옵션들이 표시되어야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')

      const options = wrapper.findAll('.multi-select__option')
      expect(options).toHaveLength(mockOptions.length)
      expect(options[0].text()).toContain('Option 1')
    })

    it('ESC 키로 드롭다운이 닫혀야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')
      expect(wrapper.find('.multi-select__dropdown').exists()).toBe(true)

      await wrapper.find('.multi-select__trigger').trigger('keydown', { key: 'Escape' })
      await nextTick()

      expect(wrapper.find('.multi-select__dropdown').exists()).toBe(false)
    })
  })

  describe('선택 기능', () => {
    it('옵션 클릭 시 선택되어야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')
      await wrapper.findAll('.multi-select__option')[0].trigger('click')

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0]).toEqual([['1']])
    })

    it('여러 옵션을 선택할 수 있어야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')
      await wrapper.findAll('.multi-select__option')[0].trigger('click')
      await wrapper.findAll('.multi-select__option')[1].trigger('click')

      const emitted = wrapper.emitted('update:modelValue')
      expect(emitted).toBeTruthy()
      expect(emitted![emitted!.length - 1]).toEqual([['1', '2']])
    })

    it('선택된 옵션의 태그가 표시되어야 한다', async () => {
      await wrapper.setProps({ modelValue: ['1', '2'] })

      const tags = wrapper.findAll('.multi-select__tag')
      expect(tags).toHaveLength(2)
      expect(tags[0].text()).toContain('Option 1')
      expect(tags[1].text()).toContain('Option 2')
    })

    it('태그의 X 버튼으로 선택을 해제할 수 있어야 한다', async () => {
      await wrapper.setProps({ modelValue: ['1', '2'] })

      const removeButton = wrapper.find('.multi-select__tag-remove')
      await removeButton.trigger('click')

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0]).toEqual([['2']])
    })
  })

  describe('전체 선택 기능', () => {
    it('전체 선택 버튼이 표시되어야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')

      const selectAllButton = wrapper.find('.multi-select__select-all')
      expect(selectAllButton.exists()).toBe(true)
      expect(selectAllButton.text()).toContain('Select All')
    })

    it('전체 선택 버튼 클릭 시 모든 옵션이 선택되어야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')
      await wrapper.find('.multi-select__select-all').trigger('click')

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0]).toEqual([['1', '2', '3', '4']])
    })

    it('showSelectAll이 false일 때 전체 선택 버튼이 숨겨져야 한다', async () => {
      await wrapper.setProps({ showSelectAll: false })
      await wrapper.find('.multi-select__trigger').trigger('click')

      expect(wrapper.find('.multi-select__select-all').exists()).toBe(false)
    })
  })

  describe('검색 기능', () => {
    it('검색 입력 필드가 표시되어야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')

      const searchInput = wrapper.find('.multi-select__search')
      expect(searchInput.exists()).toBe(true)
    })

    it('검색어 입력 시 옵션이 필터링되어야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')

      const searchInput = wrapper.find('.multi-select__search')
      await searchInput.setValue('Option 1')

      const options = wrapper.findAll('.multi-select__option')
      expect(options).toHaveLength(1)
      expect(options[0].text()).toContain('Option 1')
    })

    it('searchable이 false일 때 검색 입력이 숨겨져야 한다', async () => {
      await wrapper.setProps({ searchable: false })
      await wrapper.find('.multi-select__trigger').trigger('click')

      expect(wrapper.find('.multi-select__search').exists()).toBe(false)
    })

    it('검색 이벤트가 발생해야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')

      const searchInput = wrapper.find('.multi-select__search')
      await searchInput.setValue('test')

      expect(wrapper.emitted('search')).toBeTruthy()
      expect(wrapper.emitted('search')![0]).toEqual(['test'])
    })
  })

  describe('키보드 네비게이션', () => {
    it('화살표 키로 옵션을 탐색할 수 있어야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')

      const trigger = wrapper.find('.multi-select__trigger')
      await trigger.trigger('keydown', { key: 'ArrowDown' })

      const highlightedOption = wrapper.find('.multi-select__option--highlighted')
      expect(highlightedOption.exists()).toBe(true)
    })

    it('Enter 키로 하이라이트된 옵션을 선택할 수 있어야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')

      const trigger = wrapper.find('.multi-select__trigger')
      await trigger.trigger('keydown', { key: 'ArrowDown' })
      await trigger.trigger('keydown', { key: 'Enter' })

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0]).toEqual([['1']])
    })

    it('Space 키로 드롭다운을 열 수 있어야 한다', async () => {
      const trigger = wrapper.find('.multi-select__trigger')
      await trigger.trigger('keydown', { key: ' ' })

      expect(wrapper.find('.multi-select__dropdown').exists()).toBe(true)
    })
  })

  describe('비활성화 상태', () => {
    it('disabled일 때 드롭다운이 열리지 않아야 한다', async () => {
      await wrapper.setProps({ disabled: true })

      const trigger = wrapper.find('.multi-select__trigger')
      await trigger.trigger('click')

      expect(wrapper.find('.multi-select__dropdown').exists()).toBe(false)
    })

    it('disabled일 때 태그 제거 버튼이 표시되지 않아야 한다', async () => {
      await wrapper.setProps({
        disabled: true,
        modelValue: ['1'],
      })

      expect(wrapper.find('.multi-select__tag-remove').exists()).toBe(false)
    })

    it('disabled 클래스가 추가되어야 한다', async () => {
      await wrapper.setProps({ disabled: true })

      expect(wrapper.find('.multi-select--disabled').exists()).toBe(true)
      expect(wrapper.find('.multi-select__trigger--disabled').exists()).toBe(true)
    })
  })

  describe('maxDisplay 기능', () => {
    it('maxDisplay를 초과하면 +n 태그가 표시되어야 한다', async () => {
      await wrapper.setProps({
        modelValue: ['1', '2', '3', '4'],
        maxDisplay: 2,
      })

      const tags = wrapper.findAll('.multi-select__tag')
      expect(tags).toHaveLength(3) // 2개 태그 + 1개 카운트 태그

      const countTag = wrapper.find('.multi-select__tag--count')
      expect(countTag.exists()).toBe(true)
      expect(countTag.text()).toBe('+2')
    })
  })

  describe('커스텀 추출 함수', () => {
    it('커스텀 labelKey 함수가 작동해야 한다', async () => {
      const customOptions = [
        { name: 'Item 1', id: 'a' },
        { name: 'Item 2', id: 'b' },
      ]

      await wrapper.setProps({
        options: customOptions,
        labelKey: (item: any) => `${item.name} (${item.id})`,
        valueKey: 'id',
        modelValue: ['a'],
      })

      const tag = wrapper.find('.multi-select__tag')
      expect(tag.text()).toContain('Item 1 (a)')
    })

    it('커스텀 disabledKey 함수가 작동해야 한다', async () => {
      const customOptions = [
        { label: 'Option 1', value: '1', inactive: true },
        { label: 'Option 2', value: '2', inactive: false },
      ]

      await wrapper.setProps({
        options: customOptions,
        disabledKey: (item: any) => item.inactive,
      })

      await wrapper.find('.multi-select__trigger').trigger('click')

      const options = wrapper.findAll('.multi-select__option')
      expect(options[0].classes()).toContain('multi-select__option--disabled')
      expect(options[1].classes()).not.toContain('multi-select__option--disabled')
    })
  })

  describe('접근성', () => {
    it('적절한 ARIA 속성이 설정되어야 한다', () => {
      const trigger = wrapper.find('.multi-select__trigger')

      expect(trigger.attributes('role')).toBe('combobox')
      expect(trigger.attributes('aria-expanded')).toBe('false')
      expect(trigger.attributes('aria-haspopup')).toBe('listbox')
      expect(trigger.attributes('tabindex')).toBe('0')
    })

    it('드롭다운이 열릴 때 aria-expanded가 true가 되어야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')

      const trigger = wrapper.find('.multi-select__trigger')
      expect(trigger.attributes('aria-expanded')).toBe('true')
    })

    it('labelId가 설정될 때 aria-labelledby가 적용되어야 한다', async () => {
      await wrapper.setProps({ labelId: 'test-label' })

      const trigger = wrapper.find('.multi-select__trigger')
      expect(trigger.attributes('aria-labelledby')).toBe('test-label')
    })

    it('ariaDescribedby가 설정될 때 적용되어야 한다', async () => {
      await wrapper.setProps({ ariaDescribedby: 'test-description' })

      const trigger = wrapper.find('.multi-select__trigger')
      expect(trigger.attributes('aria-describedby')).toBe('test-description')
    })
  })

  describe('이벤트 발생', () => {
    it('change 이벤트가 발생해야 한다', async () => {
      await wrapper.find('.multi-select__trigger').trigger('click')
      await wrapper.findAll('.multi-select__option')[0].trigger('click')

      expect(wrapper.emitted('change')).toBeTruthy()
      expect(wrapper.emitted('change')![0]).toEqual([['1']])
    })

    it('선택 해제 시 remove 이벤트가 발생해야 한다', async () => {
      await wrapper.setProps({ modelValue: ['1'] })

      const removeButton = wrapper.find('.multi-select__tag-remove')
      await removeButton.trigger('click')

      expect(wrapper.emitted('remove')).toBeTruthy()
      expect(wrapper.emitted('remove')![0]).toEqual([{ label: 'Option 1', value: '1' }])
    })
  })

  describe('엣지 케이스', () => {
    it('빈 옵션 배열을 처리할 수 있어야 한다', async () => {
      await wrapper.setProps({ options: [] })
      await wrapper.find('.multi-select__trigger').trigger('click')

      const emptyText = wrapper.find('.multi-select__empty')
      expect(emptyText.exists()).toBe(true)
      expect(emptyText.text()).toBe('No options available')
    })

    it('null/undefined 값을 처리할 수 있어야 한다', async () => {
      await wrapper.setProps({
        options: [
          { label: 'Valid', value: '1' },
          { label: null, value: '2' },
          { value: '3' },
        ],
      })

      expect(() => {
        wrapper.find('.multi-select__trigger').trigger('click')
      }).not.toThrow()
    })
  })
})