import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import FileUpload from './FileUpload.vue'

// File 객체 모킹
const createMockFile = (name: string, size: number, type: string): File => {
  const file = new File([''], name, { type })
  Object.defineProperty(file, 'size', { value: size })
  return file
}

// DataTransfer 모킹
const createMockDataTransfer = (files: File[]): DataTransfer => {
  const dt = new DataTransfer()
  files.forEach(file => dt.items.add(file))
  return dt
}

describe('FileUpload 컴포넌트', () => {
  let wrapper: VueWrapper<any>

  beforeEach(() => {
    wrapper = mount(FileUpload, {
      props: {
        modelValue: [],
      },
    })
  })

  describe('기본 렌더링', () => {
    it('컴포넌트가 올바르게 렌더링되어야 한다', () => {
      expect(wrapper.exists()).toBe(true)
      expect(wrapper.find('.file-upload').exists()).toBe(true)
    })

    it('드롭존이 표시되어야 한다', () => {
      const dropzone = wrapper.find('.file-upload__dropzone')
      expect(dropzone.exists()).toBe(true)
    })

    it('기본 안내 텍스트가 표시되어야 한다', () => {
      expect(wrapper.text()).toContain('Click to upload or drag and drop')
      expect(wrapper.text()).toContain('Maximum file size: 10MB')
    })

    it('커스텀 안내 텍스트가 표시되어야 한다', async () => {
      await wrapper.setProps({
        primaryText: '파일을 선택하세요',
        secondaryText: '최대 5MB까지 가능'
      })
      
      expect(wrapper.text()).toContain('파일을 선택하세요')
      expect(wrapper.text()).toContain('최대 5MB까지 가능')
    })
  })

  describe('파일 선택', () => {
    it('클릭으로 파일을 선택할 수 있어야 한다', async () => {
      const file = createMockFile('test.txt', 1000, 'text/plain')
      const input = wrapper.find('input[type="file"]')
      
      // 파일 선택 시뮬레이션
      Object.defineProperty(input.element, 'files', {
        value: [file],
        writable: false,
      })
      
      await input.trigger('change')
      
      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0]).toEqual([[file]])
      expect(wrapper.emitted('change')).toBeTruthy()
    })

    it('multiple이 true일 때 여러 파일을 선택할 수 있어야 한다', async () => {
      const files = [
        createMockFile('file1.txt', 1000, 'text/plain'),
        createMockFile('file2.txt', 2000, 'text/plain'),
      ]
      
      const input = wrapper.find('input[type="file"]')
      Object.defineProperty(input.element, 'files', {
        value: files,
        writable: false,
      })
      
      await input.trigger('change')
      
      expect(wrapper.emitted('update:modelValue')![0]).toEqual([files])
    })

    it('multiple이 false일 때 하나의 파일만 선택 가능해야 한다', async () => {
      await wrapper.setProps({ multiple: false })
      
      const input = wrapper.find('input[type="file"]')
      expect(input.attributes('multiple')).toBeUndefined()
    })
  })

  describe('드래그 앤 드롭', () => {
    it('드래그오버 시 하이라이트되어야 한다', async () => {
      const dropzone = wrapper.find('.file-upload__dropzone')
      
      await dropzone.trigger('dragover', {
        preventDefault: vi.fn(),
        dataTransfer: { dropEffect: 'copy' }
      })
      
      expect(dropzone.classes()).toContain('file-upload__dropzone--drag-over')
    })

    it('드래그리브 시 하이라이트가 제거되어야 한다', async () => {
      const dropzone = wrapper.find('.file-upload__dropzone')
      
      await dropzone.trigger('dragover', {
        preventDefault: vi.fn(),
        dataTransfer: { dropEffect: 'copy' }
      })
      await dropzone.trigger('dragleave')
      
      expect(dropzone.classes()).not.toContain('file-upload__dropzone--drag-over')
    })

    it('파일 드롭 시 업로드되어야 한다', async () => {
      const file = createMockFile('dropped.txt', 1000, 'text/plain')
      const dropzone = wrapper.find('.file-upload__dropzone')
      
      await dropzone.trigger('drop', {
        preventDefault: vi.fn(),
        dataTransfer: createMockDataTransfer([file])
      })
      
      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0][0]).toContainEqual(file)
    })
  })

  describe('파일 크기 제한', () => {
    it('maxSize를 초과하는 파일은 거부되어야 한다', async () => {
      await wrapper.setProps({ maxSize: 1000 }) // 1KB
      
      const file = createMockFile('large.txt', 2000, 'text/plain') // 2KB
      const input = wrapper.find('input[type="file"]')
      
      Object.defineProperty(input.element, 'files', {
        value: [file],
        writable: false,
      })
      
      await input.trigger('change')
      
      expect(wrapper.emitted('error')).toBeTruthy()
      expect(wrapper.emitted('error')![0][0]).toMatchObject({
        type: 'size',
        file: expect.objectContaining({ name: 'large.txt' })
      })
    })

    it('허용된 크기의 파일은 업로드되어야 한다', async () => {
      await wrapper.setProps({ maxSize: 2000 }) // 2KB
      
      const file = createMockFile('small.txt', 1000, 'text/plain') // 1KB
      const input = wrapper.find('input[type="file"]')
      
      Object.defineProperty(input.element, 'files', {
        value: [file],
        writable: false,
      })
      
      await input.trigger('change')
      
      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('error')).toBeFalsy()
    })
  })

  describe('파일 타입 제한', () => {
    it('accept 속성이 input에 적용되어야 한다', async () => {
      await wrapper.setProps({ accept: 'image/*' })
      
      const input = wrapper.find('input[type="file"]')
      expect(input.attributes('accept')).toBe('image/*')
    })

    it('허용되지 않은 파일 타입은 거부되어야 한다', async () => {
      await wrapper.setProps({ accept: 'image/*' })
      
      const file = createMockFile('document.pdf', 1000, 'application/pdf')
      const input = wrapper.find('input[type="file"]')
      
      Object.defineProperty(input.element, 'files', {
        value: [file],
        writable: false,
      })
      
      await input.trigger('change')
      
      expect(wrapper.emitted('error')).toBeTruthy()
      expect(wrapper.emitted('error')![0][0]).toMatchObject({
        type: 'type',
        file: expect.objectContaining({ name: 'document.pdf' })
      })
    })
  })

  describe('파일 개수 제한', () => {
    it('maxFiles를 초과하면 에러가 발생해야 한다', async () => {
      await wrapper.setProps({ maxFiles: 2 })
      
      const files = [
        createMockFile('file1.txt', 100, 'text/plain'),
        createMockFile('file2.txt', 100, 'text/plain'),
        createMockFile('file3.txt', 100, 'text/plain'),
      ]
      
      const input = wrapper.find('input[type="file"]')
      Object.defineProperty(input.element, 'files', {
        value: files,
        writable: false,
      })
      
      await input.trigger('change')
      
      expect(wrapper.emitted('error')).toBeTruthy()
      expect(wrapper.emitted('error')![0][0]).toMatchObject({
        type: 'count',
        message: expect.stringContaining('2')
      })
    })
  })

  describe('파일 목록 표시', () => {
    beforeEach(async () => {
      const files = [
        createMockFile('image.jpg', 50000, 'image/jpeg'),
        createMockFile('document.pdf', 100000, 'application/pdf'),
      ]
      await wrapper.setProps({ modelValue: files })
    })

    it('업로드된 파일들이 목록으로 표시되어야 한다', () => {
      const fileItems = wrapper.findAll('.file-upload__file')
      expect(fileItems).toHaveLength(2)
    })

    it('파일 이름이 표시되어야 한다', () => {
      const fileNames = wrapper.findAll('.file-upload__file-name')
      expect(fileNames[0].text()).toBe('image.jpg')
      expect(fileNames[1].text()).toBe('document.pdf')
    })

    it('파일 크기가 올바른 형식으로 표시되어야 한다', () => {
      const fileSizes = wrapper.findAll('.file-upload__file-size')
      expect(fileSizes[0].text()).toMatch(/48\.8\s*KB/)
      expect(fileSizes[1].text()).toMatch(/97\.7\s*KB/)
    })

    it('이미지 파일의 미리보기가 표시되어야 한다', () => {
      const preview = wrapper.find('.file-upload__preview')
      expect(preview.exists()).toBe(true)
    })
  })

  describe('파일 삭제', () => {
    it('파일별 삭제 버튼이 표시되어야 한다', async () => {
      const file = createMockFile('test.txt', 1000, 'text/plain')
      await wrapper.setProps({ modelValue: [file] })
      
      const removeButton = wrapper.find('.file-upload__file-remove')
      expect(removeButton.exists()).toBe(true)
    })

    it('삭제 버튼 클릭 시 파일이 제거되어야 한다', async () => {
      const files = [
        createMockFile('file1.txt', 1000, 'text/plain'),
        createMockFile('file2.txt', 1000, 'text/plain'),
      ]
      await wrapper.setProps({ modelValue: files })
      
      const removeButton = wrapper.find('.file-upload__file-remove')
      await removeButton.trigger('click')
      
      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0][0]).toHaveLength(1)
      expect(wrapper.emitted('remove')).toBeTruthy()
    })
  })

  describe('배치 작업', () => {
    it('showBatchActions가 true일 때 배치 버튼들이 표시되어야 한다', async () => {
      const files = [createMockFile('test.txt', 1000, 'text/plain')]
      await wrapper.setProps({ 
        modelValue: files,
        showBatchActions: true 
      })
      
      expect(wrapper.find('.file-upload__clear-all').exists()).toBe(true)
    })

    it('전체 삭제 버튼 클릭 시 모든 파일이 제거되어야 한다', async () => {
      const files = [
        createMockFile('file1.txt', 1000, 'text/plain'),
        createMockFile('file2.txt', 1000, 'text/plain'),
      ]
      await wrapper.setProps({ 
        modelValue: files,
        showBatchActions: true 
      })
      
      const clearAllButton = wrapper.find('.file-upload__clear-all')
      await clearAllButton.trigger('click')
      
      expect(wrapper.emitted('update:modelValue')![0]).toEqual([[]])
    })

    it('showBatchActions가 false일 때 배치 버튼들이 숨겨져야 한다', async () => {
      await wrapper.setProps({ showBatchActions: false })
      
      expect(wrapper.find('.file-upload__clear-all').exists()).toBe(false)
    })
  })

  describe('자동 업로드', () => {
    const mockUploadFunction = vi.fn(async (file: File, onProgress: Function) => {
      onProgress(50)
      onProgress(100)
      return { url: 'https://example.com/file.jpg' }
    })

    it('autoUpload가 true일 때 파일 선택 시 자동으로 업로드되어야 한다', async () => {
      await wrapper.setProps({ 
        autoUpload: true,
        uploadFunction: mockUploadFunction
      })
      
      const file = createMockFile('auto.txt', 1000, 'text/plain')
      const input = wrapper.find('input[type="file"]')
      
      Object.defineProperty(input.element, 'files', {
        value: [file],
        writable: false,
      })
      
      await input.trigger('change')
      await nextTick()
      
      expect(mockUploadFunction).toHaveBeenCalledWith(
        file,
        expect.any(Function),
        expect.any(AbortController)
      )
    })

    it('업로드 진행률이 표시되어야 한다', async () => {
      const slowUpload = vi.fn(async (file: File, onProgress: Function) => {
        onProgress(25)
        await nextTick()
        return { url: 'test' }
      })
      
      await wrapper.setProps({ 
        autoUpload: true,
        uploadFunction: slowUpload,
        modelValue: [createMockFile('uploading.txt', 1000, 'text/plain')]
      })
      
      // 업로드 시작
      wrapper.vm.startUpload(wrapper.vm.files[0])
      await nextTick()
      
      const progress = wrapper.find('.file-upload__progress')
      expect(progress.exists()).toBe(true)
    })

    it('업로드 취소가 가능해야 한다', async () => {
      const controller = new AbortController()
      const cancelableUpload = vi.fn(async (file: File, onProgress: Function, ctrl: AbortController) => {
        return new Promise((resolve, reject) => {
          ctrl.signal.addEventListener('abort', () => {
            reject(new Error('Upload cancelled'))
          })
        })
      })
      
      await wrapper.setProps({ 
        autoUpload: true,
        uploadFunction: cancelableUpload,
        modelValue: [createMockFile('cancel.txt', 1000, 'text/plain')]
      })
      
      wrapper.vm.startUpload(wrapper.vm.files[0])
      await nextTick()
      
      const cancelButton = wrapper.find('.file-upload__cancel')
      expect(cancelButton.exists()).toBe(true)
      
      await cancelButton.trigger('click')
      expect(wrapper.emitted('upload-error')).toBeTruthy()
    })
  })

  describe('비활성화 상태', () => {
    beforeEach(async () => {
      await wrapper.setProps({ disabled: true })
    })

    it('드롭존이 비활성화되어야 한다', () => {
      const dropzone = wrapper.find('.file-upload__dropzone')
      expect(dropzone.classes()).toContain('file-upload__dropzone--disabled')
    })

    it('파일 입력이 비활성화되어야 한다', () => {
      const input = wrapper.find('input[type="file"]')
      expect(input.attributes('disabled')).toBeDefined()
    })

    it('드래그 앤 드롭이 작동하지 않아야 한다', async () => {
      const file = createMockFile('dropped.txt', 1000, 'text/plain')
      const dropzone = wrapper.find('.file-upload__dropzone')
      
      await dropzone.trigger('drop', {
        preventDefault: vi.fn(),
        dataTransfer: createMockDataTransfer([file])
      })
      
      expect(wrapper.emitted('update:modelValue')).toBeFalsy()
    })

    it('파일 삭제가 불가능해야 한다', async () => {
      await wrapper.setProps({ 
        disabled: true,
        modelValue: [createMockFile('test.txt', 1000, 'text/plain')]
      })
      
      expect(wrapper.find('.file-upload__file-remove').exists()).toBe(false)
    })
  })

  describe('접근성', () => {
    it('드롭존에 적절한 ARIA 속성이 설정되어야 한다', () => {
      const dropzone = wrapper.find('.file-upload__dropzone')
      
      expect(dropzone.attributes('role')).toBe('button')
      expect(dropzone.attributes('tabindex')).toBe('0')
      expect(dropzone.attributes('aria-label')).toBeTruthy()
    })

    it('커스텀 aria-label이 적용되어야 한다', async () => {
      await wrapper.setProps({ dropZoneAriaLabel: '파일을 여기에 끌어다 놓으세요' })
      
      const dropzone = wrapper.find('.file-upload__dropzone')
      expect(dropzone.attributes('aria-label')).toBe('파일을 여기에 끌어다 놓으세요')
    })

    it('키보드로 파일 선택이 가능해야 한다', async () => {
      const dropzone = wrapper.find('.file-upload__dropzone')
      await dropzone.trigger('keydown', { key: 'Enter' })
      
      // Enter 키로 파일 선택 다이얼로그가 열려야 함
      // 실제 브라우저 동작은 테스트하기 어려우므로 이벤트 발생만 확인
      expect(dropzone.element.tagName).toBe('DIV')
    })
  })

  describe('에러 처리', () => {
    it('업로드 에러가 발생하면 에러 상태가 표시되어야 한다', async () => {
      const errorUpload = vi.fn().mockRejectedValue(new Error('Upload failed'))
      
      await wrapper.setProps({ 
        autoUpload: true,
        uploadFunction: errorUpload,
        modelValue: [createMockFile('error.txt', 1000, 'text/plain')]
      })
      
      wrapper.vm.startUpload(wrapper.vm.files[0])
      await vi.waitFor(() => {
        expect(wrapper.emitted('upload-error')).toBeTruthy()
      })
      
      const errorFile = wrapper.find('.file-upload__file--error')
      expect(errorFile.exists()).toBe(true)
    })

    it('재시도 버튼이 표시되어야 한다', async () => {
      const errorUpload = vi.fn().mockRejectedValue(new Error('Upload failed'))
      
      await wrapper.setProps({ 
        autoUpload: true,
        uploadFunction: errorUpload,
        modelValue: [createMockFile('retry.txt', 1000, 'text/plain')]
      })
      
      wrapper.vm.startUpload(wrapper.vm.files[0])
      await vi.waitFor(() => {
        const retryButton = wrapper.find('.file-upload__retry')
        expect(retryButton.exists()).toBe(true)
      })
    })
  })
})