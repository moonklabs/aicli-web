// 문자열 유틸리티 함수

/**
 * 문자열을 kebab-case로 변환 (예: "HelloWorld" -> "hello-world")
 */
export function toKebabCase(str: string): string {
  return str
    .replace(/([a-z])([A-Z])/g, '$1-$2')
    .replace(/[\s_]+/g, '-')
    .toLowerCase()
}

/**
 * 문자열을 camelCase로 변환 (예: "hello-world" -> "helloWorld")
 */
export function toCamelCase(str: string): string {
  return str
    .replace(/[-_\s]+(.)?/g, (_, char) => char ? char.toUpperCase() : '')
    .replace(/^[A-Z]/, char => char.toLowerCase())
}

/**
 * 문자열을 PascalCase로 변환 (예: "hello-world" -> "HelloWorld")
 */
export function toPascalCase(str: string): string {
  return str
    .replace(/[-_\s]+(.)?/g, (_, char) => char ? char.toUpperCase() : '')
    .replace(/^[a-z]/, char => char.toUpperCase())
}

/**
 * 문자열을 snake_case로 변환 (예: "HelloWorld" -> "hello_world")
 */
export function toSnakeCase(str: string): string {
  return str
    .replace(/([a-z])([A-Z])/g, '$1_$2')
    .replace(/[-\s]+/g, '_')
    .toLowerCase()
}

/**
 * 문자열의 첫 글자를 대문자로 변환
 */
export function capitalize(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase()
}

/**
 * 문자열을 지정된 길이로 잘라내고 생략 기호 추가
 */
export function truncate(str: string, maxLength: number, ellipsis = '...'): string {
  if (str.length <= maxLength) return str
  return str.slice(0, maxLength - ellipsis.length) + ellipsis
}

/**
 * 문자열이 비어있는지 확인 (null, undefined, 공백 문자열 포함)
 */
export function isEmpty(str: string | null | undefined): boolean {
  return !str || str.trim().length === 0
}

/**
 * 문자열이 비어있지 않은지 확인
 */
export function isNotEmpty(str: string | null | undefined): str is string {
  return !isEmpty(str)
}

/**
 * 문자열에서 HTML 태그 제거
 */
export function stripHtml(str: string): string {
  return str.replace(/<[^>]*>/g, '')
}

/**
 * 문자열에서 특수 문자 제거
 */
export function removeSpecialChars(str: string): string {
  return str.replace(/[^a-zA-Z0-9가-힣\s]/g, '')
}

/**
 * 문자열에서 공백 문자 정규화 (여러 공백을 하나로, 앞뒤 공백 제거)
 */
export function normalizeWhitespace(str: string): string {
  return str.replace(/\s+/g, ' ').trim()
}

/**
 * 이메일 주소 검증
 */
export function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(email)
}

/**
 * URL 검증
 */
export function isValidUrl(url: string): boolean {
  try {
    new URL(url)
    return true
  } catch {
    return false
  }
}

/**
 * 문자열을 바이트 단위로 자르기 (한글 고려)
 */
export function truncateBytes(str: string, maxBytes: number): string {
  let bytes = 0
  let result = ''

  for (const char of str) {
    const charBytes = new Blob([char]).size
    if (bytes + charBytes > maxBytes) break
    bytes += charBytes
    result += char
  }

  return result
}

/**
 * 문자열의 바이트 길이 계산 (한글 고려)
 */
export function getByteLength(str: string): number {
  return new Blob([str]).size
}

/**
 * 문자열을 마스킹 처리 (개인정보 보호용)
 */
export function maskString(str: string, start = 0, end = 0, maskChar = '*'): string {
  if (str.length <= start + end) return str

  const startPart = str.slice(0, start)
  const endPart = str.slice(-end)
  const maskLength = str.length - start - end
  const maskPart = maskChar.repeat(maskLength)

  return startPart + maskPart + endPart
}

/**
 * 전화번호 마스킹
 */
export function maskPhoneNumber(phoneNumber: string): string {
  // 010-1234-5678 -> 010-****-5678
  return phoneNumber.replace(/(\d{3})-?\d{4}-?(\d{4})/, '$1-****-$2')
}

/**
 * 이메일 마스킹
 */
export function maskEmail(email: string): string {
  const [localPart, domain] = email.split('@')
  if (!domain) return email

  const maskedLocal = localPart.length > 2
    ? localPart.slice(0, 2) + '*'.repeat(localPart.length - 2)
    : localPart

  return `${maskedLocal}@${domain}`
}

/**
 * 문자열 배열을 자연스럽게 연결 (예: ["a", "b", "c"] -> "a, b 그리고 c")
 */
export function joinNaturally(items: string[], separator = ', ', lastSeparator = ' 그리고 '): string {
  if (items.length === 0) return ''
  if (items.length === 1) return items[0]
  if (items.length === 2) return items.join(lastSeparator)

  const allButLast = items.slice(0, -1)
  const last = items[items.length - 1]

  return allButLast.join(separator) + lastSeparator + last
}

/**
 * 랜덤 문자열 생성
 */
export function generateRandomString(length = 8, chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'): string {
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}

/**
 * UUID v4 생성 (간단한 버전)
 */
export function generateUUID(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0
    const v = c === 'x' ? r : (r & 0x3 | 0x8)
    return v.toString(16)
  })
}