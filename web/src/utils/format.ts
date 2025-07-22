// 포맷팅 유틸리티 함수

/**
 * 바이트 크기를 사람이 읽기 쉬운 형태로 변환
 */
export function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return '0 Bytes'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']

  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}

/**
 * 숫자를 천 단위 구분자가 있는 형태로 포맷팅
 */
export function formatNumber(num: number): string {
  return num.toLocaleString('ko-KR')
}

/**
 * 백분율 포맷팅
 */
export function formatPercentage(value: number, total: number, decimals = 1): string {
  if (total === 0) return '0%'
  const percentage = (value / total) * 100
  return `${percentage.toFixed(decimals)}%`
}

/**
 * 통화 포맷팅 (한국 원화)
 */
export function formatCurrency(amount: number): string {
  return amount.toLocaleString('ko-KR', {
    style: 'currency',
    currency: 'KRW',
  })
}

/**
 * 소수점 포맷팅
 */
export function formatDecimal(num: number, decimals = 2): string {
  return num.toFixed(decimals)
}

/**
 * 큰 숫자를 축약형으로 포맷팅 (예: 1,234,567 -> 1.2M)
 */
export function formatLargeNumber(num: number, decimals = 1): string {
  const suffixes = ['', 'K', 'M', 'B', 'T']
  const tier = Math.log10(Math.abs(num)) / 3 | 0

  if (tier === 0) return num.toString()

  const suffix = suffixes[tier]
  const scale = Math.pow(10, tier * 3)
  const scaled = num / scale

  return scaled.toFixed(decimals) + suffix
}

/**
 * 지속 시간 포맷팅 (밀리초를 "1시간 30분" 형태로)
 */
export function formatDuration(ms: number): string {
  if (ms < 0) return '0초'

  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) {
    const remainingHours = hours % 24
    return remainingHours > 0 ? `${days}일 ${remainingHours}시간` : `${days}일`
  }

  if (hours > 0) {
    const remainingMinutes = minutes % 60
    return remainingMinutes > 0 ? `${hours}시간 ${remainingMinutes}분` : `${hours}시간`
  }

  if (minutes > 0) {
    const remainingSeconds = seconds % 60
    return remainingSeconds > 0 ? `${minutes}분 ${remainingSeconds}초` : `${minutes}분`
  }

  return `${seconds}초`
}

/**
 * 짧은 지속 시간 포맷팅 (밀리초를 "01:30:45" 형태로)
 */
export function formatDurationShort(ms: number): string {
  const totalSeconds = Math.floor(ms / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (hours > 0) {
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
  }

  return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
}

/**
 * 진행률을 프로그레스 바 텍스트로 포맷팅
 */
export function formatProgress(current: number, total: number): string {
  const percentage = total > 0 ? Math.round((current / total) * 100) : 0
  return `${formatNumber(current)} / ${formatNumber(total)} (${percentage}%)`
}

/**
 * 속도 포맷팅 (바이트/초를 MB/s 등으로)
 */
export function formatSpeed(bytesPerSecond: number): string {
  return `${formatBytes(bytesPerSecond)}/s`
}

/**
 * IP 주소 포맷팅 및 검증
 */
export function formatIpAddress(ip: string): string {
  // IPv4 정규식
  const ipv4Regex = /^(\d{1,3}\.){3}\d{1,3}$/

  if (ipv4Regex.test(ip)) {
    return ip
  }

  // IPv6는 그대로 반환 (복잡한 포맷팅은 생략)
  return ip
}

/**
 * 포트 번호 포맷팅
 */
export function formatPort(port: number | string): string {
  const portNum = typeof port === 'string' ? parseInt(port, 10) : port

  if (isNaN(portNum) || portNum < 0 || portNum > 65535) {
    return 'Invalid Port'
  }

  return portNum.toString()
}

/**
 * 메모리 사용량 포맷팅 (사용량/총량)
 */
export function formatMemoryUsage(used: number, total: number): string {
  const usedStr = formatBytes(used)
  const totalStr = formatBytes(total)
  const percentage = formatPercentage(used, total)

  return `${usedStr} / ${totalStr} (${percentage})`
}

/**
 * CPU 사용률 포맷팅
 */
export function formatCpuUsage(percentage: number): string {
  return `${percentage.toFixed(1)}%`
}

/**
 * 네트워크 대역폭 포맷팅
 */
export function formatBandwidth(bytesPerSecond: number, _showDirection = true): {
  download: string
  upload: string
  combined: string
} {
  // 일반적으로 다운로드/업로드를 구분하지 않고 총 대역폭만 표시
  const speed = formatSpeed(bytesPerSecond)

  return {
    download: speed,
    upload: speed,
    combined: speed,
  }
}

/**
 * 상태 뱃지용 텍스트 포맷팅
 */
export function formatStatusBadge(status: string): string {
  switch (status.toLowerCase()) {
    case 'running':
      return '실행 중'
    case 'stopped':
      return '중지됨'
    case 'paused':
      return '일시정지'
    case 'error':
      return '오류'
    case 'pending':
      return '대기 중'
    case 'completed':
      return '완료'
    case 'failed':
      return '실패'
    case 'cancelled':
      return '취소됨'
    case 'active':
      return '활성'
    case 'inactive':
      return '비활성'
    default:
      return status
  }
}

/**
 * 파일 확장자에서 파일 타입 아이콘 반환
 */
export function getFileTypeIcon(filename: string): string {
  const extension = filename.split('.').pop()?.toLowerCase()

  switch (extension) {
    case 'js':
    case 'jsx':
    case 'ts':
    case 'tsx':
      return '📄'
    case 'vue':
      return '💚'
    case 'html':
    case 'htm':
      return '🌐'
    case 'css':
    case 'scss':
    case 'sass':
      return '🎨'
    case 'json':
      return '📋'
    case 'md':
    case 'markdown':
      return '📝'
    case 'png':
    case 'jpg':
    case 'jpeg':
    case 'gif':
    case 'svg':
      return '🖼️'
    case 'pdf':
      return '📕'
    case 'zip':
    case 'rar':
    case '7z':
      return '📦'
    default:
      return '📄'
  }
}

/**
 * 날짜를 포맷팅
 */
export function formatDate(dateString: string): string {
  if (!dateString) return '-'

  const date = new Date(dateString)
  if (isNaN(date.getTime())) return '-'

  const now = new Date()
  const diffInMs = now.getTime() - date.getTime()
  const diffInDays = Math.floor(diffInMs / (1000 * 60 * 60 * 24))

  if (diffInDays === 0) {
    // 오늘
    return date.toLocaleTimeString('ko-KR', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
    })
  } else if (diffInDays === 1) {
    // 어제
    return '어제'
  } else if (diffInDays < 7) {
    // 일주일 이내
    return `${diffInDays}일 전`
  } else if (diffInDays < 30) {
    // 한달 이내
    const weeks = Math.floor(diffInDays / 7)
    return `${weeks}주 전`
  } else if (diffInDays < 365) {
    // 일년 이내
    const months = Math.floor(diffInDays / 30)
    return `${months}개월 전`
  } else {
    // 일년 이상
    const years = Math.floor(diffInDays / 365)
    return `${years}년 전`
  }
}

/**
 * 절대 날짜 포맷팅
 */
export function formatAbsoluteDate(dateString: string): string {
  if (!dateString) return '-'

  const date = new Date(dateString)
  if (isNaN(date.getTime())) return '-'

  return date.toLocaleDateString('ko-KR', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'short',
  })
}

/**
 * 날짜와 시간을 포맷팅
 */
export function formatDateTime(dateString: string): string {
  if (!dateString) return '-'

  const date = new Date(dateString)
  if (isNaN(date.getTime())) return '-'

  return date.toLocaleString('ko-KR', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}