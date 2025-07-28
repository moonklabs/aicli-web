import { computed, reactive, ref, watch } from 'vue'

export interface FormStateOptions {
  // 자동 저장 설정
  autoSave?: boolean
  autoSaveDelay?: number

  // 변경 추적 설정
  trackChanges?: boolean

  // 로컬 스토리지 설정
  persistKey?: string

  // 콜백
  onSave?: (data: any) => void | Promise<void>
  onChange?: (data: any, changedField?: string) => void
  onReset?: () => void
}

export function useFormState<T extends Record<string, any>>(
  initialData: T,
  options: FormStateOptions = {},
) {
  const {
    autoSave = false,
    autoSaveDelay = 1000,
    trackChanges = true,
    persistKey,
    onSave,
    onChange,
    onReset,
  } = options

  // 폼 데이터
  const data = reactive<T>({ ...initialData })
  const originalData = ref<T>({ ...initialData })

  // 상태
  const isSaving = ref(false)
  const isSaved = ref(false)
  const saveError = ref<Error | null>(null)

  // 변경 추적
  const changedFields = ref<Set<string>>(new Set())
  const changeHistory = ref<Array<{ field: string; oldValue: any; newValue: any; timestamp: Date }>>([])

  // 자동 저장 타이머
  let autoSaveTimer: ReturnType<typeof setTimeout> | null = null

  // 로컬 스토리지에서 데이터 로드
  const loadFromStorage = () => {
    if (!persistKey) return

    try {
      const stored = localStorage.getItem(persistKey)
      if (stored) {
        const parsed = JSON.parse(stored)
        Object.assign(data, parsed)
        originalData.value = { ...parsed }
      }
    } catch (error) {
      console.error('Failed to load form data from storage:', error)
    }
  }

  // 로컬 스토리지에 데이터 저장
  const saveToStorage = () => {
    if (!persistKey) return

    try {
      localStorage.setItem(persistKey, JSON.stringify(data))
    } catch (error) {
      console.error('Failed to save form data to storage:', error)
    }
  }

  // 변경사항 저장
  const save = async () => {
    isSaving.value = true
    saveError.value = null

    try {
      if (onSave) {
        await onSave(data)
      }

      saveToStorage()
      originalData.value = { ...data }
      changedFields.value.clear()
      isSaved.value = true

      // 3초 후 저장 표시 제거
      setTimeout(() => {
        isSaved.value = false
      }, 3000)
    } catch (error) {
      saveError.value = error as Error
      throw error
    } finally {
      isSaving.value = false
    }
  }

  // 자동 저장 처리
  const handleAutoSave = () => {
    if (!autoSave) return

    if (autoSaveTimer) {
      clearTimeout(autoSaveTimer)
    }

    autoSaveTimer = setTimeout(() => {
      save().catch(console.error)
    }, autoSaveDelay)
  }

  // 필드 값 설정
  const setFieldValue = (field: string, value: any) => {
    const oldValue = data[field as keyof T]

    if (oldValue !== value) {
      data[field as keyof T] = value

      if (trackChanges) {
        changedFields.value.add(field)
        changeHistory.value.push({
          field,
          oldValue,
          newValue: value,
          timestamp: new Date(),
        })
      }

      if (onChange) {
        onChange(data, field)
      }

      handleAutoSave()
    }
  }

  // 여러 필드 값 설정
  const setFieldValues = (values: Partial<T>) => {
    Object.entries(values).forEach(([field, value]) => {
      setFieldValue(field, value)
    })
  }

  // 폼 리셋
  const reset = (newInitialData?: T) => {
    if (newInitialData) {
      originalData.value = { ...newInitialData }
    }

    Object.assign(data, originalData.value)
    changedFields.value.clear()
    changeHistory.value = []

    if (persistKey) {
      localStorage.removeItem(persistKey)
    }

    if (onReset) {
      onReset()
    }
  }

  // 변경사항 되돌리기
  const revert = () => {
    Object.assign(data, originalData.value)
    changedFields.value.clear()
    changeHistory.value = []
  }

  // 특정 필드 되돌리기
  const revertField = (field: string) => {
    data[field as keyof T] = originalData.value[field as keyof T]
    changedFields.value.delete(field)
    changeHistory.value = changeHistory.value.filter(h => h.field !== field)
  }

  // Computed 속성들
  const isDirty = computed(() => changedFields.value.size > 0)

  const hasChanges = computed(() => changedFields.value.size > 0)

  const changedFieldsList = computed(() => Array.from(changedFields.value))

  const canSave = computed(() => !isSaving.value && isDirty.value)

  const canRevert = computed(() => isDirty.value)

  // 데이터 변경 감지
  if (trackChanges) {
    watch(data, (newData) => {
      // 각 필드별로 변경 확인
      Object.keys(newData).forEach(field => {
        const originalValue = originalData.value[field as keyof T]
        const currentValue = newData[field as keyof T]

        if (JSON.stringify(originalValue) !== JSON.stringify(currentValue)) {
          changedFields.value.add(field)
        } else {
          changedFields.value.delete(field)
        }
      })
    }, { deep: true })
  }

  // 초기화
  if (persistKey) {
    loadFromStorage()
  }

  // 클린업
  const cleanup = () => {
    if (autoSaveTimer) {
      clearTimeout(autoSaveTimer)
      autoSaveTimer = null
    }
  }

  return {
    // 데이터
    data,
    originalData,

    // 상태
    isSaving,
    isSaved,
    saveError,
    isDirty,
    hasChanges,
    changedFields: changedFieldsList,
    changeHistory,
    canSave,
    canRevert,

    // 메서드
    setFieldValue,
    setFieldValues,
    save,
    reset,
    revert,
    revertField,
    cleanup,
  }
}

// 폼 스텝 관리를 위한 훅
export interface FormStep {
  id: string
  label: string
  isValid?: () => boolean
  onEnter?: () => void | Promise<void>
  onLeave?: () => void | Promise<void>
}

export function useFormSteps(steps: FormStep[], initialStep = 0) {
  const currentStepIndex = ref(initialStep)
  const visitedSteps = ref<Set<number>>(new Set([initialStep]))
  const isTransitioning = ref(false)

  const currentStep = computed(() => steps[currentStepIndex.value])

  const isFirstStep = computed(() => currentStepIndex.value === 0)

  const isLastStep = computed(() => currentStepIndex.value === steps.length - 1)

  const canGoNext = computed(() => {
    if (isLastStep.value) return false
    const step = steps[currentStepIndex.value]
    return !step.isValid || step.isValid()
  })

  const canGoPrevious = computed(() => !isFirstStep.value)

  const progress = computed(() => {
    return ((currentStepIndex.value + 1) / steps.length) * 100
  })

  const goToStep = async (index: number) => {
    if (index < 0 || index >= steps.length || index === currentStepIndex.value) {
      return
    }

    isTransitioning.value = true

    try {
      // 현재 스텝 나가기
      const currentStep = steps[currentStepIndex.value]
      if (currentStep.onLeave) {
        await currentStep.onLeave()
      }

      // 새 스텝 들어가기
      const newStep = steps[index]
      if (newStep.onEnter) {
        await newStep.onEnter()
      }

      currentStepIndex.value = index
      visitedSteps.value.add(index)
    } finally {
      isTransitioning.value = false
    }
  }

  const next = () => {
    if (canGoNext.value && !isTransitioning.value) {
      goToStep(currentStepIndex.value + 1)
    }
  }

  const previous = () => {
    if (canGoPrevious.value && !isTransitioning.value) {
      goToStep(currentStepIndex.value - 1)
    }
  }

  const reset = () => {
    currentStepIndex.value = 0
    visitedSteps.value.clear()
    visitedSteps.value.add(0)
  }

  const isStepVisited = (index: number) => visitedSteps.value.has(index)

  const isStepActive = (index: number) => currentStepIndex.value === index

  const isStepCompleted = (index: number) => {
    if (index >= currentStepIndex.value) return false
    const step = steps[index]
    return !step.isValid || step.isValid()
  }

  return {
    // 상태
    currentStep,
    currentStepIndex,
    isFirstStep,
    isLastStep,
    canGoNext,
    canGoPrevious,
    progress,
    isTransitioning,
    visitedSteps: Array.from(visitedSteps.value),

    // 메서드
    next,
    previous,
    goToStep,
    reset,
    isStepVisited,
    isStepActive,
    isStepCompleted,
  }
}