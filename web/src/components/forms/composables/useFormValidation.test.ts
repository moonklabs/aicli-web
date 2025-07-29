import { beforeEach, describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { useFormValidation } from './useFormValidation'

describe('useFormValidation composable', () => {
  describe('기본 검증', () => {
    it('필수 필드 검증이 작동해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          name: { required: true, message: '이름은 필수입니다' },
        },
      })

      const isValid = await validate({ name: '' })

      expect(isValid).toBe(false)
      expect(errors.value.name).toBe('이름은 필수입니다')
    })

    it('값이 있을 때 필수 필드 검증을 통과해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          name: { required: true },
        },
      })

      const isValid = await validate({ name: 'John' })

      expect(isValid).toBe(true)
      expect(errors.value.name).toBeUndefined()
    })
  })

  describe('타입 검증', () => {
    it('이메일 형식을 검증해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          email: {
            type: 'email',
            message: '올바른 이메일 형식이 아닙니다',
          },
        },
      })

      let isValid = await validate({ email: 'invalid-email' })
      expect(isValid).toBe(false)
      expect(errors.value.email).toBe('올바른 이메일 형식이 아닙니다')

      isValid = await validate({ email: 'valid@email.com' })
      expect(isValid).toBe(true)
      expect(errors.value.email).toBeUndefined()
    })

    it('URL 형식을 검증해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          website: { type: 'url' },
        },
      })

      let isValid = await validate({ website: 'not-a-url' })
      expect(isValid).toBe(false)

      isValid = await validate({ website: 'https://example.com' })
      expect(isValid).toBe(true)
    })

    it('숫자 타입을 검증해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          age: { type: 'number' },
        },
      })

      let isValid = await validate({ age: 'not a number' })
      expect(isValid).toBe(false)

      isValid = await validate({ age: 25 })
      expect(isValid).toBe(true)
    })
  })

  describe('길이 검증', () => {
    it('최소 길이를 검증해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          password: {
            minLength: 8,
            message: '비밀번호는 8자 이상이어야 합니다',
          },
        },
      })

      let isValid = await validate({ password: 'short' })
      expect(isValid).toBe(false)
      expect(errors.value.password).toBe('비밀번호는 8자 이상이어야 합니다')

      isValid = await validate({ password: 'longpassword' })
      expect(isValid).toBe(true)
    })

    it('최대 길이를 검증해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          username: {
            maxLength: 10,
            message: '사용자명은 10자 이하여야 합니다',
          },
        },
      })

      let isValid = await validate({ username: 'verylongusername' })
      expect(isValid).toBe(false)

      isValid = await validate({ username: 'shortname' })
      expect(isValid).toBe(true)
    })
  })

  describe('범위 검증', () => {
    it('최소값을 검증해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          age: { min: 18, message: '18세 이상만 가능합니다' },
        },
      })

      let isValid = await validate({ age: 15 })
      expect(isValid).toBe(false)
      expect(errors.value.age).toBe('18세 이상만 가능합니다')

      isValid = await validate({ age: 20 })
      expect(isValid).toBe(true)
    })

    it('최대값을 검증해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          quantity: { max: 100 },
        },
      })

      let isValid = await validate({ quantity: 150 })
      expect(isValid).toBe(false)

      isValid = await validate({ quantity: 50 })
      expect(isValid).toBe(true)
    })
  })

  describe('패턴 검증', () => {
    it('정규식 패턴을 검증해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          phone: {
            pattern: /^\d{3}-\d{4}-\d{4}$/,
            message: '전화번호 형식이 올바르지 않습니다 (000-0000-0000)',
          },
        },
      })

      let isValid = await validate({ phone: '1234567890' })
      expect(isValid).toBe(false)
      expect(errors.value.phone).toContain('전화번호 형식')

      isValid = await validate({ phone: '010-1234-5678' })
      expect(isValid).toBe(true)
    })
  })

  describe('커스텀 검증', () => {
    it('동기 커스텀 검증이 작동해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          password: {
            validator: (value: string) => {
              if (value.includes('password')) {
                return '비밀번호에 "password"를 포함할 수 없습니다'
              }
              return true
            },
          },
        },
      })

      let isValid = await validate({ password: 'mypassword123' })
      expect(isValid).toBe(false)
      expect(errors.value.password).toContain('password')

      isValid = await validate({ password: 'secure123' })
      expect(isValid).toBe(true)
    })

    it('비동기 커스텀 검증이 작동해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          username: {
            asyncValidator: async (value: string) => {
              // API 호출 시뮬레이션
              await new Promise(resolve => setTimeout(resolve, 10))

              if (value === 'admin') {
                return '이미 사용 중인 사용자명입니다'
              }
              return true
            },
          },
        },
      })

      let isValid = await validate({ username: 'admin' })
      expect(isValid).toBe(false)
      expect(errors.value.username).toContain('사용 중')

      isValid = await validate({ username: 'newuser' })
      expect(isValid).toBe(true)
    })
  })

  describe('복합 규칙', () => {
    it('여러 규칙을 동시에 적용할 수 있어야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          password: {
            required: true,
            minLength: 8,
            maxLength: 20,
            pattern: /^(?=.*[A-Za-z])(?=.*\d)/,
            message: {
              required: '비밀번호는 필수입니다',
              minLength: '최소 8자 이상이어야 합니다',
              maxLength: '최대 20자까지 가능합니다',
              pattern: '영문과 숫자를 포함해야 합니다',
            },
          },
        },
      })

      // 빈 값
      let isValid = await validate({ password: '' })
      expect(isValid).toBe(false)
      expect(errors.value.password).toBe('비밀번호는 필수입니다')

      // 너무 짧음
      isValid = await validate({ password: 'abc123' })
      expect(isValid).toBe(false)
      expect(errors.value.password).toBe('최소 8자 이상이어야 합니다')

      // 패턴 불일치
      isValid = await validate({ password: 'onlyletters' })
      expect(isValid).toBe(false)
      expect(errors.value.password).toBe('영문과 숫자를 포함해야 합니다')

      // 유효한 값
      isValid = await validate({ password: 'secure123' })
      expect(isValid).toBe(true)
      expect(errors.value.password).toBeUndefined()
    })
  })

  describe('필드 간 검증', () => {
    it('다른 필드와 비교 검증이 가능해야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          password: { required: true },
          confirmPassword: {
            validator: (value: string, formData: any) => {
              if (value !== formData.password) {
                return '비밀번호가 일치하지 않습니다'
              }
              return true
            },
          },
        },
      })

      const formData = {
        password: 'secure123',
        confirmPassword: 'different',
      }

      let isValid = await validate(formData)
      expect(isValid).toBe(false)
      expect(errors.value.confirmPassword).toBe('비밀번호가 일치하지 않습니다')

      formData.confirmPassword = 'secure123'
      isValid = await validate(formData)
      expect(isValid).toBe(true)
    })
  })

  describe('동적 규칙', () => {
    it('조건부 검증이 가능해야 한다', async () => {
      const isBusinessAccount = ref(false)

      const { validate, errors } = useFormValidation({
        rules: {
          businessNumber: {
            required: () => isBusinessAccount.value,
            pattern: /^\d{3}-\d{2}-\d{5}$/,
            message: {
              required: '사업자등록번호는 필수입니다',
              pattern: '올바른 사업자등록번호 형식이 아닙니다',
            },
          },
        },
      })

      // 개인 계정일 때는 검증 통과
      let isValid = await validate({ businessNumber: '' })
      expect(isValid).toBe(true)

      // 사업자 계정으로 변경
      isBusinessAccount.value = true
      isValid = await validate({ businessNumber: '' })
      expect(isValid).toBe(false)
      expect(errors.value.businessNumber).toBe('사업자등록번호는 필수입니다')
    })
  })

  describe('메서드', () => {
    it('validateField로 특정 필드만 검증할 수 있어야 한다', async () => {
      const { validateField, errors } = useFormValidation({
        rules: {
          email: { required: true, type: 'email' },
          password: { required: true, minLength: 8 },
        },
      })

      const formData = {
        email: 'invalid-email',
        password: '',
      }

      // 이메일 필드만 검증
      const isEmailValid = await validateField('email', formData)
      expect(isEmailValid).toBe(false)
      expect(errors.value.email).toBeDefined()
      expect(errors.value.password).toBeUndefined()
    })

    it('clearErrors로 에러를 초기화할 수 있어야 한다', async () => {
      const { validate, clearErrors, errors } = useFormValidation({
        rules: {
          email: { required: true },
        },
      })

      await validate({ email: '' })
      expect(errors.value.email).toBeDefined()

      clearErrors()
      expect(errors.value.email).toBeUndefined()
    })

    it('clearFieldError로 특정 필드의 에러만 초기화할 수 있어야 한다', async () => {
      const { validate, clearFieldError, errors } = useFormValidation({
        rules: {
          email: { required: true },
          password: { required: true },
        },
      })

      await validate({ email: '', password: '' })
      expect(errors.value.email).toBeDefined()
      expect(errors.value.password).toBeDefined()

      clearFieldError('email')
      expect(errors.value.email).toBeUndefined()
      expect(errors.value.password).toBeDefined()
    })

    it('setError로 수동으로 에러를 설정할 수 있어야 한다', () => {
      const { setError, errors } = useFormValidation({
        rules: {},
      })

      setError('email', '서버 에러가 발생했습니다')
      expect(errors.value.email).toBe('서버 에러가 발생했습니다')
    })
  })

  describe('상태 관리', () => {
    it('isValidating 상태가 올바르게 업데이트되어야 한다', async () => {
      const { validate, isValidating } = useFormValidation({
        rules: {
          username: {
            asyncValidator: async () => {
              await new Promise(resolve => setTimeout(resolve, 10))
              return true
            },
          },
        },
      })

      expect(isValidating.value).toBe(false)

      const validatePromise = validate({ username: 'test' })
      expect(isValidating.value).toBe(true)

      await validatePromise
      expect(isValidating.value).toBe(false)
    })

    it('isDirty 상태가 검증 후 true가 되어야 한다', async () => {
      const { validate, isDirty } = useFormValidation({
        rules: {
          email: { required: true },
        },
      })

      expect(isDirty.value).toBe(false)

      await validate({ email: 'test@example.com' })
      expect(isDirty.value).toBe(true)
    })

    it('hasErrors computed가 올바르게 작동해야 한다', async () => {
      const { validate, hasErrors } = useFormValidation({
        rules: {
          email: { required: true },
        },
      })

      expect(hasErrors.value).toBe(false)

      await validate({ email: '' })
      expect(hasErrors.value).toBe(true)

      await validate({ email: 'test@example.com' })
      expect(hasErrors.value).toBe(false)
    })
  })

  describe('옵션', () => {
    it('validateOnChange 옵션이 작동해야 한다', async () => {
      const { errors, handleFieldChange } = useFormValidation({
        rules: {
          email: { required: true, type: 'email' },
        },
        validateOnChange: true,
      })

      await handleFieldChange('email', 'invalid-email')
      expect(errors.value.email).toBeDefined()

      await handleFieldChange('email', 'valid@email.com')
      expect(errors.value.email).toBeUndefined()
    })

    it('validateOnBlur 옵션이 작동해야 한다', async () => {
      const { errors, handleFieldBlur } = useFormValidation({
        rules: {
          email: { required: true },
        },
        validateOnBlur: true,
      })

      await handleFieldBlur('email', { email: '' })
      expect(errors.value.email).toBeDefined()
    })

    it('기본 에러 메시지가 적용되어야 한다', async () => {
      const { validate, errors } = useFormValidation({
        rules: {
          email: { required: true, type: 'email' },
        },
      })

      await validate({ email: '' })
      expect(errors.value.email).toBe('This field is required')

      await validate({ email: 'invalid' })
      expect(errors.value.email).toContain('valid email')
    })
  })
})