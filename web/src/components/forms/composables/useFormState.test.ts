import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { useFormState } from './useFormState'

describe('useFormState composable', () => {
  describe('기본 상태 관리', () => {
    it('초기값으로 폼 상태를 생성해야 한다', () => {
      const initialValues = {
        name: 'John',
        email: 'john@example.com',
        age: 30,
      }

      const { formData } = useFormState({ initialValues })

      expect(formData.value).toEqual(initialValues)
    })

    it('빈 객체로 시작할 수 있어야 한다', () => {
      const { formData } = useFormState()

      expect(formData.value).toEqual({})
    })
  })

  describe('필드 업데이트', () => {
    it('updateField로 개별 필드를 업데이트할 수 있어야 한다', () => {
      const { formData, updateField } = useFormState({
        initialValues: { name: 'John', age: 30 },
      })

      updateField('name', 'Jane')

      expect(formData.value.name).toBe('Jane')
      expect(formData.value.age).toBe(30)
    })

    it('중첩된 필드를 업데이트할 수 있어야 한다', () => {
      const { formData, updateField } = useFormState({
        initialValues: {
          user: {
            name: 'John',
            address: {
              city: 'Seoul',
            },
          },
        },
      })

      updateField('user.address.city', 'Busan')

      expect(formData.value.user.address.city).toBe('Busan')
      expect(formData.value.user.name).toBe('John')
    })

    it('배열 인덱스를 사용하여 업데이트할 수 있어야 한다', () => {
      const { formData, updateField } = useFormState({
        initialValues: {
          hobbies: ['reading', 'gaming', 'coding'],
        },
      })

      updateField('hobbies[1]', 'swimming')

      expect(formData.value.hobbies).toEqual(['reading', 'swimming', 'coding'])
    })
  })

  describe('필드 값 가져오기', () => {
    it('getFieldValue로 필드 값을 가져올 수 있어야 한다', () => {
      const { getFieldValue } = useFormState({
        initialValues: { name: 'John', age: 30 },
      })

      expect(getFieldValue('name')).toBe('John')
      expect(getFieldValue('age')).toBe(30)
    })

    it('중첩된 필드 값을 가져올 수 있어야 한다', () => {
      const { getFieldValue } = useFormState({
        initialValues: {
          user: {
            profile: {
              bio: 'Hello world',
            },
          },
        },
      })

      expect(getFieldValue('user.profile.bio')).toBe('Hello world')
    })

    it('존재하지 않는 필드는 undefined를 반환해야 한다', () => {
      const { getFieldValue } = useFormState({
        initialValues: { name: 'John' },
      })

      expect(getFieldValue('email')).toBeUndefined()
    })
  })

  describe('isDirty 상태', () => {
    it('초기 상태에서는 isDirty가 false여야 한다', () => {
      const { isDirty } = useFormState({
        initialValues: { name: 'John' },
      })

      expect(isDirty.value).toBe(false)
    })

    it('필드가 변경되면 isDirty가 true가 되어야 한다', () => {
      const { isDirty, updateField } = useFormState({
        initialValues: { name: 'John' },
      })

      updateField('name', 'Jane')

      expect(isDirty.value).toBe(true)
    })

    it('원래 값으로 되돌리면 isDirty가 false가 되어야 한다', () => {
      const { isDirty, updateField } = useFormState({
        initialValues: { name: 'John' },
      })

      updateField('name', 'Jane')
      expect(isDirty.value).toBe(true)

      updateField('name', 'John')
      expect(isDirty.value).toBe(false)
    })

    it('dirtyFields로 변경된 필드를 추적할 수 있어야 한다', () => {
      const { dirtyFields, updateField } = useFormState({
        initialValues: { name: 'John', email: 'john@example.com' },
      })

      expect(dirtyFields.value).toEqual([])

      updateField('name', 'Jane')
      expect(dirtyFields.value).toContain('name')
      expect(dirtyFields.value).not.toContain('email')

      updateField('email', 'jane@example.com')
      expect(dirtyFields.value).toContain('name')
      expect(dirtyFields.value).toContain('email')
    })
  })

  describe('리셋 기능', () => {
    it('reset으로 초기값으로 되돌릴 수 있어야 한다', () => {
      const initialValues = { name: 'John', age: 30 }
      const { formData, updateField, reset } = useFormState({ initialValues })

      updateField('name', 'Jane')
      updateField('age', 25)

      reset()

      expect(formData.value).toEqual(initialValues)
    })

    it('reset 후 isDirty가 false가 되어야 한다', () => {
      const { isDirty, updateField, reset } = useFormState({
        initialValues: { name: 'John' },
      })

      updateField('name', 'Jane')
      expect(isDirty.value).toBe(true)

      reset()
      expect(isDirty.value).toBe(false)
    })

    it('resetField로 특정 필드만 초기화할 수 있어야 한다', () => {
      const { formData, updateField, resetField } = useFormState({
        initialValues: { name: 'John', email: 'john@example.com' },
      })

      updateField('name', 'Jane')
      updateField('email', 'jane@example.com')

      resetField('name')

      expect(formData.value.name).toBe('John')
      expect(formData.value.email).toBe('jane@example.com')
    })

    it('새로운 값으로 리셋할 수 있어야 한다', () => {
      const { formData, reset } = useFormState({
        initialValues: { name: 'John' },
      })

      const newValues = { name: 'Jane', email: 'jane@example.com' }
      reset(newValues)

      expect(formData.value).toEqual(newValues)
    })
  })

  describe('폼 제출', () => {
    it('setValues로 여러 필드를 한 번에 설정할 수 있어야 한다', () => {
      const { formData, setValues } = useFormState({
        initialValues: { name: 'John', age: 30 },
      })

      setValues({ name: 'Jane', age: 25, email: 'jane@example.com' })

      expect(formData.value).toEqual({
        name: 'Jane',
        age: 25,
        email: 'jane@example.com',
      })
    })

    it('부분 업데이트가 가능해야 한다', () => {
      const { formData, setValues } = useFormState({
        initialValues: { name: 'John', age: 30, city: 'Seoul' },
      })

      setValues({ name: 'Jane' }, true) // merge = true

      expect(formData.value).toEqual({
        name: 'Jane',
        age: 30,
        city: 'Seoul',
      })
    })

    it('getValues로 현재 폼 데이터를 가져올 수 있어야 한다', () => {
      const initialValues = { name: 'John', age: 30 }
      const { getValues } = useFormState({ initialValues })

      expect(getValues()).toEqual(initialValues)
    })
  })

  describe('필드 터치 상태', () => {
    it('초기 상태에서는 어떤 필드도 터치되지 않아야 한다', () => {
      const { touchedFields } = useFormState({
        initialValues: { name: 'John' },
      })

      expect(touchedFields.value).toEqual([])
    })

    it('touchField로 필드를 터치 상태로 만들 수 있어야 한다', () => {
      const { touchedFields, touchField } = useFormState({
        initialValues: { name: 'John' },
      })

      touchField('name')

      expect(touchedFields.value).toContain('name')
    })

    it('isFieldTouched로 특정 필드의 터치 상태를 확인할 수 있어야 한다', () => {
      const { isFieldTouched, touchField } = useFormState({
        initialValues: { name: 'John', email: 'john@example.com' },
      })

      expect(isFieldTouched('name')).toBe(false)

      touchField('name')

      expect(isFieldTouched('name')).toBe(true)
      expect(isFieldTouched('email')).toBe(false)
    })

    it('reset 시 터치 상태도 초기화되어야 한다', () => {
      const { touchedFields, touchField, reset } = useFormState({
        initialValues: { name: 'John' },
      })

      touchField('name')
      expect(touchedFields.value).toContain('name')

      reset()
      expect(touchedFields.value).toEqual([])
    })
  })

  describe('폼 상태 추적', () => {
    it('isSubmitting 상태를 관리할 수 있어야 한다', async () => {
      const { isSubmitting, setSubmitting } = useFormState()

      expect(isSubmitting.value).toBe(false)

      setSubmitting(true)
      expect(isSubmitting.value).toBe(true)

      setSubmitting(false)
      expect(isSubmitting.value).toBe(false)
    })

    it('submitCount를 추적할 수 있어야 한다', () => {
      const { submitCount, incrementSubmitCount } = useFormState()

      expect(submitCount.value).toBe(0)

      incrementSubmitCount()
      expect(submitCount.value).toBe(1)

      incrementSubmitCount()
      expect(submitCount.value).toBe(2)
    })

    it('isValid 상태를 설정할 수 있어야 한다', () => {
      const { isValid, setValid } = useFormState()

      expect(isValid.value).toBe(true) // 기본값

      setValid(false)
      expect(isValid.value).toBe(false)

      setValid(true)
      expect(isValid.value).toBe(true)
    })
  })

  describe('onChange 콜백', () => {
    it('필드 변경 시 onChange 콜백이 호출되어야 한다', () => {
      const onChange = vi.fn()
      const { updateField } = useFormState({
        initialValues: { name: 'John' },
        onChange,
      })

      updateField('name', 'Jane')

      expect(onChange).toHaveBeenCalledWith({
        name: 'Jane',
      })
    })

    it('setValues 시에도 onChange가 호출되어야 한다', () => {
      const onChange = vi.fn()
      const { setValues } = useFormState({
        initialValues: { name: 'John' },
        onChange,
      })

      setValues({ name: 'Jane', age: 25 })

      expect(onChange).toHaveBeenCalledWith({
        name: 'Jane',
        age: 25,
      })
    })
  })

  describe('필드 배열 관리', () => {
    it('배열 필드에 항목을 추가할 수 있어야 한다', () => {
      const { formData, arrayHelpers } = useFormState({
        initialValues: {
          hobbies: ['reading', 'gaming'],
        },
      })

      const { push } = arrayHelpers('hobbies')
      push('coding')

      expect(formData.value.hobbies).toEqual(['reading', 'gaming', 'coding'])
    })

    it('배열 필드에서 항목을 제거할 수 있어야 한다', () => {
      const { formData, arrayHelpers } = useFormState({
        initialValues: {
          hobbies: ['reading', 'gaming', 'coding'],
        },
      })

      const { remove } = arrayHelpers('hobbies')
      remove(1)

      expect(formData.value.hobbies).toEqual(['reading', 'coding'])
    })

    it('배열 필드의 항목을 이동할 수 있어야 한다', () => {
      const { formData, arrayHelpers } = useFormState({
        initialValues: {
          hobbies: ['reading', 'gaming', 'coding'],
        },
      })

      const { move } = arrayHelpers('hobbies')
      move(0, 2)

      expect(formData.value.hobbies).toEqual(['gaming', 'coding', 'reading'])
    })

    it('배열 필드에 여러 항목을 삽입할 수 있어야 한다', () => {
      const { formData, arrayHelpers } = useFormState({
        initialValues: {
          hobbies: ['reading', 'gaming'],
        },
      })

      const { insert } = arrayHelpers('hobbies')
      insert(1, 'swimming')

      expect(formData.value.hobbies).toEqual(['reading', 'swimming', 'gaming'])
    })
  })

  describe('엣지 케이스', () => {
    it('깊은 중첩 객체를 처리할 수 있어야 한다', () => {
      const { formData, updateField } = useFormState({
        initialValues: {
          level1: {
            level2: {
              level3: {
                level4: {
                  value: 'deep',
                },
              },
            },
          },
        },
      })

      updateField('level1.level2.level3.level4.value', 'updated')

      expect(formData.value.level1.level2.level3.level4.value).toBe('updated')
    })

    it('undefined 초기값을 처리할 수 있어야 한다', () => {
      const { formData, updateField } = useFormState({
        initialValues: {
          name: undefined,
          age: null,
        },
      })

      expect(formData.value.name).toBeUndefined()
      expect(formData.value.age).toBeNull()

      updateField('name', 'John')
      expect(formData.value.name).toBe('John')
    })

    it('원시 타입 배열을 처리할 수 있어야 한다', () => {
      const { formData, updateField } = useFormState({
        initialValues: {
          numbers: [1, 2, 3],
          strings: ['a', 'b', 'c'],
        },
      })

      updateField('numbers[1]', 5)
      updateField('strings[2]', 'z')

      expect(formData.value.numbers).toEqual([1, 5, 3])
      expect(formData.value.strings).toEqual(['a', 'b', 'z'])
    })

    it('동적으로 필드를 추가할 수 있어야 한다', () => {
      const { formData, updateField } = useFormState({
        initialValues: {},
      })

      updateField('newField', 'value')
      updateField('nested.field', 'nestedValue')

      expect(formData.value).toEqual({
        newField: 'value',
        nested: {
          field: 'nestedValue',
        },
      })
    })
  })
})