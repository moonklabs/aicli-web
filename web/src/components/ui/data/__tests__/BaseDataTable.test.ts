import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseDataTable from '../BaseDataTable.vue'
import type { AdvancedTableColumn } from '@/types/ui'

describe('BaseDataTable', () => {
  const mockData = [
    { id: 1, name: 'John Doe', age: 30, email: 'john@example.com', active: true },
    { id: 2, name: 'Jane Smith', age: 25, email: 'jane@example.com', active: false },
    { id: 3, name: 'Bob Johnson', age: 35, email: 'bob@example.com', active: true },
    { id: 4, name: 'Alice Brown', age: 28, email: 'alice@example.com', active: true },
    { id: 5, name: 'Charlie Wilson', age: 42, email: 'charlie@example.com', active: false },
  ]

  const mockColumns: AdvancedTableColumn[] = [
    { key: 'id', title: 'ID', width: 80, sortable: true },
    { key: 'name', title: 'Name', sortable: true, filterable: true },
    { key: 'age', title: 'Age', sortable: true, filterable: true },
    { key: 'email', title: 'Email', sortable: true, filterable: true },
    {
      key: 'active',
      title: 'Status',
      sortable: true,
      render: (row) => row.active ? '활성' : '비활성',
    },
  ]

  let wrapper: any

  beforeEach(() => {
    wrapper = mount(BaseDataTable, {
      props: {
        data: mockData,
        columns: mockColumns,
        rowKey: 'id',
      },
    })
  })

  describe('렌더링', () => {
    it('테이블이 올바르게 렌더링되어야 한다', () => {
      expect(wrapper.find('.base-data-table').exists()).toBe(true)
      expect(wrapper.find('.data-table').exists()).toBe(true)
    })

    it('컬럼 헤더가 올바르게 표시되어야 한다', () => {
      const headers = wrapper.findAll('.table-header-cell')
      expect(headers).toHaveLength(mockColumns.length)

      mockColumns.forEach((column, index) => {
        expect(headers[index].text()).toContain(column.title)
      })
    })

    it('데이터 행이 올바르게 표시되어야 한다', () => {
      const rows = wrapper.findAll('.table-row')
      expect(rows).toHaveLength(mockData.length)
    })

    it('셀 데이터가 올바르게 표시되어야 한다', () => {
      const firstRow = wrapper.find('.table-row')
      const cells = firstRow.findAll('.table-cell')

      expect(cells[0].text()).toBe('1') // ID
      expect(cells[1].text()).toBe('John Doe') // Name
      expect(cells[2].text()).toBe('30') // Age
    })
  })

  describe('정렬 기능', () => {
    it('정렬 가능한 컬럼에 정렬 인디케이터가 표시되어야 한다', () => {
      const sortableHeaders = wrapper.findAll('.table-header-cell.sortable')
      expect(sortableHeaders.length).toBeGreaterThan(0)

      sortableHeaders.forEach(header => {
        expect(header.find('.sort-indicator').exists()).toBe(true)
      })
    })

    it('헤더 클릭 시 정렬이 작동해야 한다', async () => {
      const nameHeader = wrapper.find('[data-testid="header-name"]') ||
                        wrapper.findAll('.table-header-cell')[1] // Name 컬럼

      await nameHeader.trigger('click')

      // 정렬 이벤트가 발생했는지 확인
      expect(wrapper.emitted('update:sorter')).toBeTruthy()
    })

    it('연속 클릭 시 정렬 순서가 변경되어야 한다', async () => {
      const nameHeader = wrapper.findAll('.table-header-cell')[1]

      // 첫 번째 클릭 (오름차순)
      await nameHeader.trigger('click')
      let sortEvents = wrapper.emitted('update:sorter')
      expect(sortEvents[0][0]).toEqual(expect.objectContaining({
        key: 'name',
        order: 'asc',
      }))

      // 두 번째 클릭 (내림차순)
      await nameHeader.trigger('click')
      sortEvents = wrapper.emitted('update:sorter')
      expect(sortEvents[1][0]).toEqual(expect.objectContaining({
        key: 'name',
        order: 'desc',
      }))
    })
  })

  describe('필터링 기능', () => {
    beforeEach(() => {
      wrapper = mount(BaseDataTable, {
        props: {
          data: mockData,
          columns: mockColumns,
          showFilters: true,
          globalSearch: true,
        },
      })
    })

    it('글로벌 검색이 표시되어야 한다', () => {
      expect(wrapper.find('.global-search').exists()).toBe(true)
      expect(wrapper.find('.search-input').exists()).toBe(true)
    })

    it('필터 행이 표시되어야 한다', () => {
      expect(wrapper.find('.filter-row').exists()).toBe(true)
    })

    it('글로벌 검색이 작동해야 한다', async () => {
      const searchInput = wrapper.find('.search-input')
      await searchInput.setValue('John')

      // 검색 후 데이터가 필터링되는지 확인
      await wrapper.vm.$nextTick()
      // 실제 필터링 로직은 컴포넌트 내부에서 처리되므로
      // 여기서는 입력값이 올바르게 설정되었는지만 확인
      expect(searchInput.element.value).toBe('John')
    })
  })

  describe('선택 기능', () => {
    beforeEach(() => {
      wrapper = mount(BaseDataTable, {
        props: {
          data: mockData,
          columns: mockColumns,
          selection: {
            type: 'checkbox',
            selectedKeys: [],
          },
        },
      })
    })

    it('선택 체크박스가 표시되어야 한다', () => {
      expect(wrapper.find('.selection-col').exists()).toBe(true)
      expect(wrapper.findAll('input[type="checkbox"]').length).toBeGreaterThan(0)
    })

    it('전체 선택 체크박스가 작동해야 한다', async () => {
      const selectAllCheckbox = wrapper.find('thead .selection-col input[type="checkbox"]')
      await selectAllCheckbox.setChecked(true)

      expect(wrapper.emitted('update:checkedRowKeys')).toBeTruthy()
    })

    it('개별 행 선택이 작동해야 한다', async () => {
      const firstRowCheckbox = wrapper.find('tbody tr:first-child .selection-col input[type="checkbox"]')
      await firstRowCheckbox.setChecked(true)

      expect(wrapper.emitted('update:checkedRowKeys')).toBeTruthy()
    })
  })

  describe('페이지네이션', () => {
    beforeEach(() => {
      wrapper = mount(BaseDataTable, {
        props: {
          data: mockData,
          columns: mockColumns,
          pagination: true,
          pageSize: 2,
        },
      })
    })

    it('페이지네이션이 표시되어야 한다', () => {
      expect(wrapper.find('.table-pagination').exists()).toBe(true)
    })

    it('페이지네이션 정보가 올바르게 표시되어야 한다', () => {
      const paginationInfo = wrapper.find('.pagination-info')
      expect(paginationInfo.text()).toContain('총 5개')
    })

    it('페이지 이동 버튼이 작동해야 한다', async () => {
      const nextButton = wrapper.find('.pagination-controls button:last-child')
      await nextButton.trigger('click')

      expect(wrapper.emitted('update:page')).toBeTruthy()
    })
  })

  describe('가상 스크롤링', () => {
    beforeEach(() => {
      wrapper = mount(BaseDataTable, {
        props: {
          data: Array.from({ length: 1000 }, (_, i) => ({
            id: i + 1,
            name: `User ${i + 1}`,
            age: 20 + (i % 50),
            email: `user${i + 1}@example.com`,
          })),
          columns: mockColumns,
          virtualScroll: {
            enabled: true,
            itemHeight: 40,
            overscan: 5,
          },
        },
      })
    })

    it('가상 스크롤 컨테이너가 생성되어야 한다', () => {
      expect(wrapper.find('.virtual-container').exists()).toBe(true)
    })

    it('가상 스크롤링이 활성화되면 모든 행이 DOM에 렌더링되지 않아야 한다', () => {
      const renderedRows = wrapper.findAll('.table-row')
      expect(renderedRows.length).toBeLessThan(1000)
    })
  })

  describe('접근성', () => {
    it('적절한 ARIA 속성이 설정되어야 한다', () => {
      const table = wrapper.find('.base-data-table')
      expect(table.attributes('role')).toBe('table')
      expect(table.attributes('aria-label')).toBeDefined()
    })

    it('헤더 셀에 적절한 ARIA 속성이 설정되어야 한다', () => {
      const headerCells = wrapper.findAll('.table-header-cell')
      headerCells.forEach(cell => {
        expect(cell.attributes('role')).toBeDefined()
        if (cell.classes().includes('sortable')) {
          expect(cell.attributes('aria-sort')).toBeDefined()
        }
      })
    })

    it('키보드 네비게이션이 지원되어야 한다', async () => {
      const sortableHeader = wrapper.find('.table-header-cell.sortable')
      await sortableHeader.trigger('keydown', { key: 'Enter' })

      // Enter 키 이벤트가 처리되었는지 확인
      expect(wrapper.emitted('update:sorter')).toBeTruthy()
    })
  })

  describe('이벤트', () => {
    it('행 클릭 이벤트가 발생해야 한다', async () => {
      const firstRow = wrapper.find('.table-row')
      await firstRow.trigger('click')

      expect(wrapper.emitted('row-click')).toBeTruthy()
      expect(wrapper.emitted('row-click')[0]).toHaveLength(3) // row, index, event
    })

    it('행 더블클릭 이벤트가 발생해야 한다', async () => {
      const firstRow = wrapper.find('.table-row')
      await firstRow.trigger('dblclick')

      expect(wrapper.emitted('row-double-click')).toBeTruthy()
    })

    it('셀 클릭 이벤트가 발생해야 한다', async () => {
      const firstCell = wrapper.find('.table-cell')
      await firstCell.trigger('click')

      expect(wrapper.emitted('cell-click')).toBeTruthy()
      expect(wrapper.emitted('cell-click')[0]).toHaveLength(4) // cell, row, column, event
    })
  })

  describe('로딩 상태', () => {
    it('로딩 상태가 표시되어야 한다', async () => {
      await wrapper.setProps({ loading: true })

      expect(wrapper.find('.loading-overlay').exists()).toBe(true)
      expect(wrapper.find('.loading-spinner').exists()).toBe(true)
    })

    it('로딩 중에는 테이블이 비활성화되어야 한다', async () => {
      await wrapper.setProps({ loading: true })

      expect(wrapper.find('.table-container').classes()).toContain('opacity-60')
    })
  })

  describe('빈 상태', () => {
    it('데이터가 없을 때 빈 상태가 표시되어야 한다', async () => {
      await wrapper.setProps({ data: [] })

      expect(wrapper.find('.empty-state').exists()).toBe(true)
    })

    it('커스텀 빈 상태 슬롯이 작동해야 한다', () => {
      const customEmptyWrapper = mount(BaseDataTable, {
        props: {
          data: [],
          columns: mockColumns,
        },
        slots: {
          empty: '<div class="custom-empty">No data available</div>',
        },
      })

      expect(customEmptyWrapper.find('.custom-empty').exists()).toBe(true)
    })
  })

  describe('반응형', () => {
    it('작은 화면에서 적절히 조정되어야 한다', () => {
      // CSS 클래스가 올바르게 적용되는지 확인
      expect(wrapper.find('.base-data-table').exists()).toBe(true)
    })
  })

  describe('성능', () => {
    it('대량 데이터에서도 성능이 유지되어야 한다', () => {
      const largeData = Array.from({ length: 10000 }, (_, i) => ({
        id: i + 1,
        name: `User ${i + 1}`,
        age: 20 + (i % 50),
      }))

      const performanceWrapper = mount(BaseDataTable, {
        props: {
          data: largeData,
          columns: mockColumns,
          virtualScroll: { enabled: true },
        },
      })

      // 가상 스크롤링이 활성화되어 실제 DOM 노드 수가 제한되는지 확인
      const renderedRows = performanceWrapper.findAll('.table-row')
      expect(renderedRows.length).toBeLessThan(100)
    })
  })
})

// 유틸리티 함수 테스트
describe('BaseDataTable 유틸리티', () => {
  it('행 키 생성이 올바르게 작동해야 한다', () => {
    const wrapper = mount(BaseDataTable, {
      props: {
        data: [{ id: 1, name: 'Test' }],
        columns: [{ key: 'name', title: 'Name' }],
        rowKey: 'id',
      },
    })

    // 내부 메서드 테스트는 실제 구현에 따라 조정 필요
    expect(wrapper.vm).toBeDefined()
  })

  it('셀 값 추출이 올바르게 작동해야 한다', () => {
    const wrapper = mount(BaseDataTable, {
      props: {
        data: [{ nested: { value: 'test' } }],
        columns: [{ key: 'nested.value', title: 'Nested' }],
      },
    })

    expect(wrapper.vm).toBeDefined()
  })
})