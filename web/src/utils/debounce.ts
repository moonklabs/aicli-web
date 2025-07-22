// 디바운스 및 스로틀 유틸리티 함수

/**
 * 디바운스 함수 - 마지막 호출 후 지정된 시간이 지나야 실행
 */
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number,
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null

  return function executedFunction(...args: Parameters<T>): void {
    const later = () => {
      timeout = null
      func(...args)
    }

    if (timeout !== null) {
      clearTimeout(timeout)
    }
    timeout = setTimeout(later, wait)
  }
}

/**
 * 스로틀 함수 - 지정된 시간 간격으로만 실행
 */
export function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number,
): (...args: Parameters<T>) => void {
  let inThrottle = false

  return function executedFunction(...args: Parameters<T>): void {
    if (!inThrottle) {
      func(...args)
      inThrottle = true
      setTimeout(() => {
        inThrottle = false
      }, limit)
    }
  }
}

/**
 * 프로미스 디바운스 - 비동기 함수용 디바운스
 */
export function debounceAsync<T extends (...args: any[]) => Promise<any>>(
  func: T,
  wait: number,
): (...args: Parameters<T>) => Promise<ReturnType<T>> {
  let timeout: NodeJS.Timeout | null = null
  let resolveList: Array<{
    resolve: (value: ReturnType<T>) => void
    reject: (error: any) => void
  }> = []

  return function executedFunction(...args: Parameters<T>): Promise<ReturnType<T>> {
    return new Promise((resolve, reject) => {
      resolveList.push({ resolve, reject })

      if (timeout !== null) {
        clearTimeout(timeout)
      }

      timeout = setTimeout(async () => {
        const currentResolveList = resolveList
        resolveList = []
        timeout = null

        try {
          const result = await func(...args)
          currentResolveList.forEach(({ resolve }) => resolve(result))
        } catch (error) {
          currentResolveList.forEach(({ reject }) => reject(error))
        }
      }, wait)
    })
  }
}

/**
 * 실행 빈도 제한 - 최대 실행 횟수 제한
 */
export function rateLimit<T extends (...args: any[]) => any>(
  func: T,
  maxCalls: number,
  timeWindow: number,
): (...args: Parameters<T>) => boolean {
  let calls: number[] = []

  return function executedFunction(...args: Parameters<T>): boolean {
    const now = Date.now()
    calls = calls.filter(time => now - time < timeWindow)

    if (calls.length < maxCalls) {
      calls.push(now)
      func(...args)
      return true
    }

    return false // 실행 횟수 초과
  }
}

/**
 * 한 번만 실행되는 함수
 */
export function once<T extends (...args: any[]) => any>(
  func: T,
): (...args: Parameters<T>) => ReturnType<T> | undefined {
  let hasRun = false
  let result: ReturnType<T>

  return function executedFunction(...args: Parameters<T>): ReturnType<T> | undefined {
    if (!hasRun) {
      hasRun = true
      result = func(...args)
      return result
    }
    return result
  }
}

/**
 * 메모이제이션 - 결과를 캐시하여 같은 인자로 호출 시 캐시된 결과 반환
 */
export function memoize<T extends (...args: any[]) => any>(
  func: T,
  keyGenerator?: (...args: Parameters<T>) => string,
): T {
  const cache = new Map<string, ReturnType<T>>()

  return function memoizedFunction(...args: Parameters<T>): ReturnType<T> {
    const key = keyGenerator ? keyGenerator(...args) : JSON.stringify(args)

    if (cache.has(key)) {
      return cache.get(key)!
    }

    const result = func(...args)
    cache.set(key, result)
    return result
  } as T
}

/**
 * 지연 실행 - 지정된 시간 후에 함수 실행
 */
export function delay<T extends (...args: any[]) => any>(
  func: T,
  wait: number,
): (...args: Parameters<T>) => Promise<ReturnType<T>> {
  return function delayedFunction(...args: Parameters<T>): Promise<ReturnType<T>> {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve(func(...args))
      }, wait)
    })
  }
}

/**
 * 재시도 함수 - 실패 시 지정된 횟수만큼 재시도
 */
export async function retry<T>(
  func: () => Promise<T>,
  attempts = 3,
  delayMs = 1000,
): Promise<T> {
  let lastError: Error

  for (let i = 0; i < attempts; i++) {
    try {
      return await func()
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error))
      if (i < attempts - 1) {
        await new Promise(resolve => setTimeout(resolve, delayMs * (i + 1)))
      }
    }
  }

  throw lastError!
}

/**
 * 타임아웃이 있는 프로미스
 */
export function withTimeout<T>(
  promise: Promise<T>,
  timeoutMs: number,
  timeoutMessage = 'Operation timed out',
): Promise<T> {
  const timeoutPromise = new Promise<never>((_, reject) => {
    setTimeout(() => reject(new Error(timeoutMessage)), timeoutMs)
  })

  return Promise.race([promise, timeoutPromise])
}

/**
 * 조건부 실행 - 조건이 참일 때만 함수 실행
 */
export function when<T extends (...args: any[]) => any>(
  condition: boolean | ((...args: Parameters<T>) => boolean),
  func: T,
): (...args: Parameters<T>) => ReturnType<T> | undefined {
  return function conditionalFunction(...args: Parameters<T>): ReturnType<T> | undefined {
    const shouldExecute = typeof condition === 'function' ? condition(...args) : condition
    return shouldExecute ? func(...args) : undefined
  }
}

/**
 * 함수 실행 시간 측정
 */
export function measureTime<T extends (...args: any[]) => any>(
  func: T,
  onComplete?: (executionTime: number) => void,
): (...args: Parameters<T>) => ReturnType<T> {
  return function timedFunction(...args: Parameters<T>): ReturnType<T> {
    const startTime = performance.now()
    const result = func(...args)
    const endTime = performance.now()
    const executionTime = endTime - startTime

    if (onComplete) {
      onComplete(executionTime)
    }

    return result
  }
}