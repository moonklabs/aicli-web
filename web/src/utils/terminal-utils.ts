/**
 * 터미널 관련 유틸리티 함수들
 */

export interface TerminalLine {
  id: string
  content: string
  timestamp: Date
  type: 'output' | 'input' | 'error' | 'system'
  raw?: string // 원본 ANSI 포함 텍스트
}

export interface TerminalSize {
  cols: number
  rows: number
}

/**
 * 새로운 터미널 라인 생성
 */
export function createTerminalLine(
  content: string,
  type: TerminalLine['type'] = 'output',
  raw?: string,
): TerminalLine {
  return {
    id: generateLineId(),
    content,
    timestamp: new Date(),
    type,
    raw: raw || content,
  }
}

/**
 * 유니크한 라인 ID 생성
 */
export function generateLineId(): string {
  return `line-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
}

/**
 * 터미널 크기 계산
 */
export function calculateTerminalSize(
  container: HTMLElement,
  fontSize = 14,
  lineHeight = 1.4,
): TerminalSize {
  const containerWidth = container.clientWidth
  const containerHeight = container.clientHeight

  // 대략적인 문자 너비 (monospace 폰트 기준)
  const charWidth = fontSize * 0.6
  const lineHeightPx = fontSize * lineHeight

  return {
    cols: Math.floor(containerWidth / charWidth),
    rows: Math.floor(containerHeight / lineHeightPx),
  }
}

/**
 * 텍스트에서 ANSI 이스케이프 시퀀스 제거
 */
export function stripAnsi(text: string): string {
  // ANSI 이스케이프 시퀀스 제거 정규식
  const ansiRegex = /\x1b\[[0-9;]*m/g
  return text.replace(ansiRegex, '')
}

/**
 * 터미널 라인 배열을 텍스트로 변환
 */
export function linesToText(lines: TerminalLine[]): string {
  return lines.map(line => line.content).join('\n')
}

/**
 * 터미널 라인 배열을 원본 ANSI 텍스트로 변환
 */
export function linesToRawText(lines: TerminalLine[]): string {
  return lines.map(line => line.raw || line.content).join('\n')
}

/**
 * 터미널 히스토리에서 검색
 */
export function searchInLines(
  lines: TerminalLine[],
  query: string,
  caseSensitive = false,
): TerminalLine[] {
  if (!query.trim()) return lines

  const searchText = caseSensitive ? query : query.toLowerCase()

  return lines.filter(line => {
    const content = caseSensitive ? line.content : line.content.toLowerCase()
    return content.includes(searchText)
  })
}

/**
 * 터미널 라인 필터링 (타입별)
 */
export function filterLinesByType(
  lines: TerminalLine[],
  types: TerminalLine['type'][],
): TerminalLine[] {
  return lines.filter(line => types.includes(line.type))
}

/**
 * 터미널 스크롤 상태 관리
 */
export class TerminalScrollManager {
  private isAutoScrollEnabled = true
  private userScrolledUp = false
  private lastScrollTop = 0

  constructor(private container: HTMLElement) {
    this.setupScrollListener()
  }

  private setupScrollListener(): void {
    this.container.addEventListener('scroll', () => {
      const { scrollTop, scrollHeight, clientHeight } = this.container
      const isAtBottom = scrollTop + clientHeight >= scrollHeight - 5 // 5px 여유

      // 사용자가 수동으로 스크롤했는지 감지
      if (scrollTop < this.lastScrollTop && !isAtBottom) {
        this.userScrolledUp = true
        this.isAutoScrollEnabled = false
      } else if (isAtBottom) {
        this.userScrolledUp = false
        this.isAutoScrollEnabled = true
      }

      this.lastScrollTop = scrollTop
    })
  }

  /**
   * 자동 스크롤 활성화 상태
   */
  get autoScrollEnabled(): boolean {
    return this.isAutoScrollEnabled
  }

  /**
   * 맨 아래로 스크롤
   */
  scrollToBottom(): void {
    this.container.scrollTop = this.container.scrollHeight
    this.isAutoScrollEnabled = true
    this.userScrolledUp = false
  }

  /**
   * 자동 스크롤 토글
   */
  toggleAutoScroll(): void {
    this.isAutoScrollEnabled = !this.isAutoScrollEnabled
    if (this.isAutoScrollEnabled) {
      this.scrollToBottom()
    }
  }

  /**
   * 새 콘텐츠 추가 시 호출
   */
  onNewContent(): void {
    if (this.isAutoScrollEnabled) {
      // 다음 프레임에서 스크롤 (DOM 업데이트 후)
      requestAnimationFrame(() => {
        this.scrollToBottom()
      })
    }
  }
}

/**
 * 터미널 입력 히스토리 관리
 */
export class TerminalHistory {
  private history: string[] = []
  private currentIndex = -1
  private maxSize = 1000

  constructor(maxSize?: number) {
    if (maxSize) {
      this.maxSize = maxSize
    }
  }

  /**
   * 새 명령어 추가
   */
  add(command: string): void {
    if (!command.trim()) return

    // 중복 제거 (연속된 같은 명령어)
    if (this.history[this.history.length - 1] !== command) {
      this.history.push(command)

      // 크기 제한
      if (this.history.length > this.maxSize) {
        this.history.shift()
      }
    }

    this.currentIndex = -1 // 리셋
  }

  /**
   * 이전 명령어 가져오기 (위 화살표)
   */
  getPrevious(): string | null {
    if (this.history.length === 0) return null

    if (this.currentIndex === -1) {
      this.currentIndex = this.history.length - 1
    } else if (this.currentIndex > 0) {
      this.currentIndex--
    }

    return this.history[this.currentIndex]
  }

  /**
   * 다음 명령어 가져오기 (아래 화살표)
   */
  getNext(): string | null {
    if (this.history.length === 0 || this.currentIndex === -1) return null

    if (this.currentIndex < this.history.length - 1) {
      this.currentIndex++
      return this.history[this.currentIndex]
    } else {
      this.currentIndex = -1
      return ''
    }
  }

  /**
   * 검색 (부분 일치)
   */
  search(query: string): string[] {
    if (!query.trim()) return []

    return this.history.filter(cmd =>
      cmd.toLowerCase().includes(query.toLowerCase()),
    ).reverse() // 최신순
  }

  /**
   * 전체 히스토리 가져오기
   */
  getAll(): string[] {
    return [...this.history].reverse() // 최신순
  }

  /**
   * 히스토리 클리어
   */
  clear(): void {
    this.history = []
    this.currentIndex = -1
  }
}

/**
 * 터미널 텍스트 선택 관리
 */
export class TerminalSelection {
  private startLine = -1
  private startChar = -1
  private endLine = -1
  private endChar = -1

  /**
   * 선택 시작
   */
  start(lineIndex: number, charIndex: number): void {
    this.startLine = lineIndex
    this.startChar = charIndex
    this.endLine = lineIndex
    this.endChar = charIndex
  }

  /**
   * 선택 업데이트
   */
  update(lineIndex: number, charIndex: number): void {
    this.endLine = lineIndex
    this.endChar = charIndex
  }

  /**
   * 선택 완료
   */
  end(): { start: { line: number; char: number }; end: { line: number; char: number } } | null {
    if (this.startLine === -1) return null

    // 시작과 끝 정규화 (위치 순서 맞추기)
    const isReversed = this.startLine > this.endLine ||
      (this.startLine === this.endLine && this.startChar > this.endChar)

    const result = {
      start: {
        line: isReversed ? this.endLine : this.startLine,
        char: isReversed ? this.endChar : this.startChar,
      },
      end: {
        line: isReversed ? this.startLine : this.endLine,
        char: isReversed ? this.startChar : this.endChar,
      },
    }

    this.clear()
    return result
  }

  /**
   * 선택 클리어
   */
  clear(): void {
    this.startLine = -1
    this.startChar = -1
    this.endLine = -1
    this.endChar = -1
  }

  /**
   * 현재 선택 상태
   */
  get current(): { start: { line: number; char: number }; end: { line: number; char: number } } | null {
    if (this.startLine === -1) return null

    return {
      start: { line: this.startLine, char: this.startChar },
      end: { line: this.endLine, char: this.endChar },
    }
  }
}

/**
 * 디바운스 함수
 */
export function debounce<T extends (...args: any[]) => void>(
  func: T,
  wait: number,
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null

  return (...args: Parameters<T>) => {
    if (timeout) {
      clearTimeout(timeout)
    }

    timeout = setTimeout(() => {
      func(...args)
    }, wait)
  }
}

/**
 * 스로틀 함수
 */
export function throttle<T extends (...args: any[]) => void>(
  func: T,
  limit: number,
): (...args: Parameters<T>) => void {
  let inThrottle = false

  return (...args: Parameters<T>) => {
    if (!inThrottle) {
      func(...args)
      inThrottle = true
      setTimeout(() => {
        inThrottle = false
      }, limit)
    }
  }
}