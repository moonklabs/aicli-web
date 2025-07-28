import { ref, computed, reactive, Ref, ComputedRef } from 'vue'

// 유효성 검사 규칙 타입
export interface ValidationRule {
  required?: boolean | { value: boolean; message?: string }
  min?: number | { value: number; message?: string }
  max?: number | { value: number; message?: string }
  minLength?: number | { value: number; message?: string }
  maxLength?: number | { value: number; message?: string }
  pattern?: RegExp | { value: RegExp; message?: string }
  email?: boolean | { value: boolean; message?: string }
  url?: boolean | { value: boolean; message?: string }
  custom?: ((value: any) => boolean | string)[]
}

// 필드 에러 타입
export interface FieldError {
  field: string
  message: string
  rule: string
}

// 폼 상태 타입
export interface FormState {
  isDirty: boolean
  isValid: boolean
  isValidating: boolean
  errors: Record<string, string[]>
}

// 기본 에러 메시지
const defaultMessages = {
  required: '필수 입력 항목입니다',
  min: (value: number) => `최소값은 ${value}입니다`,
  max: (value: number) => `최대값은 ${value}입니다`,
  minLength: (value: number) => `최소 ${value}자 이상 입력해주세요`,
  maxLength: (value: number) => `최대 ${value}자까지 입력 가능합니다`,
  pattern: '올바른 형식으로 입력해주세요',
  email: '올바른 이메일 주소를 입력해주세요',
  url: '올바른 URL을 입력해주세요'
}

// 이메일 정규식
const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

// URL 정규식
const urlRegex = /^(https?:\/\/)?([\da-z\.-]+)\.([a-z\.]{2,6})([\/\w \.-]*)*\/?$/

export function useFormValidation<T extends Record<string, any>>(
  initialValues: T,
  rules: Partial<Record<keyof T, ValidationRule>>
) {
  // 폼 데이터
  const formData = reactive<T>({ ...initialValues })
  
  // 에러 상태
  const errors = ref<Record<string, string[]>>({})
  
  // 터치된 필드
  const touched = ref<Set<string>>(new Set())
  
  // 폼 상태
  const isDirty = ref(false)
  const isValidating = ref(false)

  // 필드 유효성 검사
  const validateField = async (field: string, value: any): Promise<string[]> => {
    const fieldRules = rules[field as keyof T]
    if (!fieldRules) return []

    const fieldErrors: string[] = []

    // Required 검사
    if (fieldRules.required) {
      const isEmpty = value === null || value === undefined || value === '' || 
                     (Array.isArray(value) && value.length === 0)
      
      if (isEmpty) {
        const message = typeof fieldRules.required === 'object' && fieldRules.required.message
          ? fieldRules.required.message
          : defaultMessages.required
        fieldErrors.push(message)
      }
    }

    // 값이 없으면 다른 검사 스킵
    if (!value && value !== 0) return fieldErrors

    // Min 검사
    if (fieldRules.min !== undefined) {
      const minValue = typeof fieldRules.min === 'object' ? fieldRules.min.value : fieldRules.min
      const message = typeof fieldRules.min === 'object' && fieldRules.min.message
        ? fieldRules.min.message
        : defaultMessages.min(minValue)
      
      if (Number(value) < minValue) {
        fieldErrors.push(message)
      }
    }

    // Max 검사
    if (fieldRules.max !== undefined) {
      const maxValue = typeof fieldRules.max === 'object' ? fieldRules.max.value : fieldRules.max
      const message = typeof fieldRules.max === 'object' && fieldRules.max.message
        ? fieldRules.max.message
        : defaultMessages.max(maxValue)
      
      if (Number(value) > maxValue) {
        fieldErrors.push(message)
      }
    }

    // MinLength 검사
    if (fieldRules.minLength !== undefined) {
      const minLength = typeof fieldRules.minLength === 'object' ? fieldRules.minLength.value : fieldRules.minLength
      const message = typeof fieldRules.minLength === 'object' && fieldRules.minLength.message
        ? fieldRules.minLength.message
        : defaultMessages.minLength(minLength)
      
      if (String(value).length < minLength) {
        fieldErrors.push(message)
      }
    }

    // MaxLength 검사
    if (fieldRules.maxLength !== undefined) {
      const maxLength = typeof fieldRules.maxLength === 'object' ? fieldRules.maxLength.value : fieldRules.maxLength
      const message = typeof fieldRules.maxLength === 'object' && fieldRules.maxLength.message
        ? fieldRules.maxLength.message
        : defaultMessages.maxLength(maxLength)
      
      if (String(value).length > maxLength) {
        fieldErrors.push(message)
      }
    }

    // Pattern 검사
    if (fieldRules.pattern) {
      const pattern = typeof fieldRules.pattern === 'object' ? fieldRules.pattern.value : fieldRules.pattern
      const message = typeof fieldRules.pattern === 'object' && fieldRules.pattern.message
        ? fieldRules.pattern.message
        : defaultMessages.pattern
      
      if (!pattern.test(String(value))) {
        fieldErrors.push(message)
      }
    }

    // Email 검사
    if (fieldRules.email) {
      const message = typeof fieldRules.email === 'object' && fieldRules.email.message
        ? fieldRules.email.message
        : defaultMessages.email
      
      if (!emailRegex.test(String(value))) {
        fieldErrors.push(message)
      }
    }

    // URL 검사
    if (fieldRules.url) {
      const message = typeof fieldRules.url === 'object' && fieldRules.url.message
        ? fieldRules.url.message
        : defaultMessages.url
      
      if (!urlRegex.test(String(value))) {
        fieldErrors.push(message)
      }
    }

    // Custom 검사
    if (fieldRules.custom) {
      for (const customValidator of fieldRules.custom) {
        const result = await customValidator(value)
        if (result !== true && result !== '') {
          fieldErrors.push(typeof result === 'string' ? result : '유효하지 않은 값입니다')
        }
      }
    }

    return fieldErrors
  }

  // 모든 필드 유효성 검사
  const validate = async (): Promise<boolean> => {
    isValidating.value = true
    const newErrors: Record<string, string[]> = {}

    for (const field in rules) {
      const fieldErrors = await validateField(field, formData[field])
      if (fieldErrors.length > 0) {
        newErrors[field] = fieldErrors
      }
    }

    errors.value = newErrors
    isValidating.value = false
    
    return Object.keys(newErrors).length === 0
  }

  // 필드 값 변경 처리
  const handleFieldChange = async (field: string, value: any) => {
    formData[field as keyof T] = value
    isDirty.value = true
    touched.value.add(field)

    // 필드 유효성 검사
    const fieldErrors = await validateField(field, value)
    
    if (fieldErrors.length > 0) {
      errors.value[field] = fieldErrors
    } else {
      delete errors.value[field]
    }
  }

  // 필드 블러 처리
  const handleFieldBlur = (field: string) => {
    touched.value.add(field)
  }

  // 폼 리셋
  const reset = () => {
    Object.assign(formData, initialValues)
    errors.value = {}
    touched.value.clear()
    isDirty.value = false
  }

  // 특정 필드 에러 삭제
  const clearFieldError = (field: string) => {
    delete errors.value[field]
  }

  // 모든 에러 삭제
  const clearErrors = () => {
    errors.value = {}
  }

  // Computed 속성들
  const isValid = computed(() => Object.keys(errors.value).length === 0)
  
  const hasErrors = computed(() => Object.keys(errors.value).length > 0)
  
  const firstError = computed(() => {
    const firstField = Object.keys(errors.value)[0]
    return firstField ? errors.value[firstField][0] : null
  })

  // 필드별 헬퍼 함수 생성
  const getFieldProps = (field: string) => ({
    value: formData[field as keyof T],
    onUpdate: (value: any) => handleFieldChange(field, value),
    onBlur: () => handleFieldBlur(field),
    error: errors.value[field]?.[0],
    errors: errors.value[field] || [],
    hasError: !!errors.value[field]?.length,
    isTouched: touched.value.has(field),
    isValid: !errors.value[field]?.length && touched.value.has(field)
  })

  return {
    // 폼 데이터
    formData,
    
    // 상태
    errors,
    touched,
    isDirty,
    isValidating,
    isValid,
    hasErrors,
    firstError,
    
    // 메서드
    validate,
    handleFieldChange,
    handleFieldBlur,
    reset,
    clearFieldError,
    clearErrors,
    getFieldProps
  }
}

// 개별 필드 유효성 검사 훅
export function useFieldValidation(
  initialValue: any,
  rules: ValidationRule
) {
  const value = ref(initialValue)
  const errors = ref<string[]>([])
  const touched = ref(false)
  const isValidating = ref(false)

  const validate = async () => {
    isValidating.value = true
    const form = useFormValidation({ field: value.value }, { field: rules })
    const fieldErrors = await form.validateField('field', value.value)
    errors.value = fieldErrors
    isValidating.value = false
    return fieldErrors.length === 0
  }

  const handleChange = async (newValue: any) => {
    value.value = newValue
    if (touched.value) {
      await validate()
    }
  }

  const handleBlur = async () => {
    touched.value = true
    await validate()
  }

  const reset = () => {
    value.value = initialValue
    errors.value = []
    touched.value = false
  }

  const hasError = computed(() => errors.value.length > 0)
  const firstError = computed(() => errors.value[0])
  const isValid = computed(() => errors.value.length === 0 && touched.value)

  return {
    value,
    errors,
    touched,
    isValidating,
    hasError,
    firstError,
    isValid,
    validate,
    handleChange,
    handleBlur,
    reset
  }
}