// 날짜 유틸리티 함수

/**
 * 날짜를 상대적인 시간으로 표시 (예: "2분 전", "1시간 전")
 */
export function timeAgo(date: string | Date): string {
  const now = new Date()
  const targetDate = typeof date === 'string' ? new Date(date) : date
  const diffInSeconds = Math.floor((now.getTime() - targetDate.getTime()) / 1000)

  if (diffInSeconds < 60) {
    return '방금 전'
  }

  const diffInMinutes = Math.floor(diffInSeconds / 60)
  if (diffInMinutes < 60) {
    return `${diffInMinutes}분 전`
  }

  const diffInHours = Math.floor(diffInMinutes / 60)
  if (diffInHours < 24) {
    return `${diffInHours}시간 전`
  }

  const diffInDays = Math.floor(diffInHours / 24)
  if (diffInDays < 7) {
    return `${diffInDays}일 전`
  }

  const diffInWeeks = Math.floor(diffInDays / 7)
  if (diffInWeeks < 4) {
    return `${diffInWeeks}주 전`
  }

  const diffInMonths = Math.floor(diffInDays / 30)
  if (diffInMonths < 12) {
    return `${diffInMonths}개월 전`
  }

  const diffInYears = Math.floor(diffInDays / 365)
  return `${diffInYears}년 전`
}

/**
 * 날짜를 로케일에 맞게 포맷팅
 */
export function formatDate(date: string | Date, format: 'full' | 'date' | 'time' | 'datetime' = 'datetime'): string {
  const targetDate = typeof date === 'string' ? new Date(date) : date

  const options: Intl.DateTimeFormatOptions = {
    timeZone: 'Asia/Seoul',
  }

  switch (format) {
    case 'full':
      options.year = 'numeric'
      options.month = 'long'
      options.day = 'numeric'
      options.hour = '2-digit'
      options.minute = '2-digit'
      options.second = '2-digit'
      break
    case 'date':
      options.year = 'numeric'
      options.month = '2-digit'
      options.day = '2-digit'
      break
    case 'time':
      options.hour = '2-digit'
      options.minute = '2-digit'
      options.second = '2-digit'
      break
    case 'datetime':
    default:
      options.year = 'numeric'
      options.month = '2-digit'
      options.day = '2-digit'
      options.hour = '2-digit'
      options.minute = '2-digit'
      break
  }

  return targetDate.toLocaleString('ko-KR', options)
}

/**
 * ISO 날짜 문자열을 로컬 날짜 문자열로 변환
 */
export function toLocalDateString(isoString: string): string {
  return new Date(isoString).toLocaleDateString('ko-KR')
}

/**
 * ISO 날짜 문자열을 로컬 시간 문자열로 변환
 */
export function toLocalTimeString(isoString: string): string {
  return new Date(isoString).toLocaleTimeString('ko-KR')
}

/**
 * 현재 날짜를 ISO 문자열로 반환
 */
export function nowISO(): string {
  return new Date().toISOString()
}

/**
 * 날짜가 오늘인지 확인
 */
export function isToday(date: string | Date): boolean {
  const today = new Date()
  const targetDate = typeof date === 'string' ? new Date(date) : date

  return today.toDateString() === targetDate.toDateString()
}

/**
 * 두 날짜가 같은 날인지 확인
 */
export function isSameDay(date1: string | Date, date2: string | Date): boolean {
  const d1 = typeof date1 === 'string' ? new Date(date1) : date1
  const d2 = typeof date2 === 'string' ? new Date(date2) : date2

  return d1.toDateString() === d2.toDateString()
}

/**
 * 날짜 범위 내에 있는지 확인
 */
export function isWithinRange(date: string | Date, startDate: string | Date, endDate: string | Date): boolean {
  const targetDate = typeof date === 'string' ? new Date(date) : date
  const start = typeof startDate === 'string' ? new Date(startDate) : startDate
  const end = typeof endDate === 'string' ? new Date(endDate) : endDate

  return targetDate >= start && targetDate <= end
}