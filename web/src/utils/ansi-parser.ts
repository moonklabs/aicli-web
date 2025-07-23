/**
 * ANSI 이스케이프 시퀀스 파서
 * Claude CLI 출력의 색상 및 스타일을 HTML로 변환
 */

export interface AnsiStyle {
  color?: string
  backgroundColor?: string
  bold?: boolean
  italic?: boolean
  underline?: boolean
  dim?: boolean
  strikethrough?: boolean
}

export interface AnsiSegment {
  text: string
  style: AnsiStyle
}

// ANSI 색상 코드 매핑 (Standard 16 colors)
const ANSI_COLORS = {
  // Standard colors (30-37, 90-97)
  30: '#000000', // Black
  31: '#CD3131', // Red
  32: '#0DBC79', // Green
  33: '#E5E510', // Yellow
  34: '#2472C8', // Blue
  35: '#BC3FBC', // Magenta
  36: '#11A8CD', // Cyan
  37: '#E5E5E5', // White

  // Bright colors (90-97)
  90: '#666666', // Bright Black (Gray)
  91: '#F14C4C', // Bright Red
  92: '#23D18B', // Bright Green
  93: '#F5F543', // Bright Yellow
  94: '#3B8EEA', // Bright Blue
  95: '#D670D6', // Bright Magenta
  96: '#29B8DB', // Bright Cyan
  97: '#FFFFFF', // Bright White
}

const ANSI_BG_COLORS = {
  // Background colors (40-47, 100-107)
  40: '#000000', 41: '#CD3131', 42: '#0DBC79', 43: '#E5E510',
  44: '#2472C8', 45: '#BC3FBC', 46: '#11A8CD', 47: '#E5E5E5',
  100: '#666666', 101: '#F14C4C', 102: '#23D18B', 103: '#F5F543',
  104: '#3B8EEA', 105: '#D670D6', 106: '#29B8DB', 107: '#FFFFFF',
}

export class AnsiParser {
  private currentStyle: AnsiStyle = {}

  /**
   * ANSI 이스케이프 시퀀스가 포함된 텍스트를 파싱하여 스타일 정보와 함께 반환
   */
  parse(text: string): AnsiSegment[] {
    const segments: AnsiSegment[] = []
    const ansiRegex = /\x1b\[[0-9;]*m/g

    let lastIndex = 0
    let match: RegExpExecArray | null

    // ANSI 시퀀스를 찾으면서 텍스트를 분할
    while ((match = ansiRegex.exec(text)) !== null) {
      // ANSI 시퀀스 이전의 텍스트 추가
      if (match.index > lastIndex) {
        const textContent = text.slice(lastIndex, match.index)
        if (textContent) {
          segments.push({
            text: textContent,
            style: { ...this.currentStyle },
          })
        }
      }

      // ANSI 시퀀스 파싱하여 스타일 업데이트
      this.parseAnsiSequence(match[0])
      lastIndex = ansiRegex.lastIndex
    }

    // 마지막 남은 텍스트 추가
    if (lastIndex < text.length) {
      const textContent = text.slice(lastIndex)
      if (textContent) {
        segments.push({
          text: textContent,
          style: { ...this.currentStyle },
        })
      }
    }

    return segments
  }

  /**
   * 단일 ANSI 이스케이프 시퀀스를 파싱하여 스타일 상태 업데이트
   */
  private parseAnsiSequence(sequence: string): void {
    // \x1b[...m 형태에서 숫자 부분 추출
    const codes = sequence.slice(2, -1).split(';').map(code => parseInt(code) || 0)

    for (let i = 0; i < codes.length; i++) {
      const code = codes[i]

      switch (code) {
        case 0: // Reset all
          this.currentStyle = {}
          break

        case 1: // Bold
          this.currentStyle.bold = true
          break

        case 2: // Dim
          this.currentStyle.dim = true
          break

        case 3: // Italic
          this.currentStyle.italic = true
          break

        case 4: // Underline
          this.currentStyle.underline = true
          break

        case 9: // Strikethrough
          this.currentStyle.strikethrough = true
          break

        case 22: // Normal intensity (not bold/dim)
          this.currentStyle.bold = false
          this.currentStyle.dim = false
          break

        case 23: // Not italic
          this.currentStyle.italic = false
          break

        case 24: // Not underlined
          this.currentStyle.underline = false
          break

        case 29: // Not strikethrough
          this.currentStyle.strikethrough = false
          break

        case 38: // 256-color or RGB foreground
          i = this.parse256ColorOrRGB(codes, i, 'color')
          break

        case 48: // 256-color or RGB background
          i = this.parse256ColorOrRGB(codes, i, 'backgroundColor')
          break

        case 39: // Default foreground color
          this.currentStyle.color = undefined
          break

        case 49: // Default background color
          this.currentStyle.backgroundColor = undefined
          break

        default:
          // Standard color codes
          if ((code >= 30 && code <= 37) || (code >= 90 && code <= 97)) {
            this.currentStyle.color = ANSI_COLORS[code as keyof typeof ANSI_COLORS]
          } else if ((code >= 40 && code <= 47) || (code >= 100 && code <= 107)) {
            this.currentStyle.backgroundColor = ANSI_BG_COLORS[code as keyof typeof ANSI_BG_COLORS]
          }
          break
      }
    }
  }

  /**
   * 256색 또는 RGB 색상 파싱
   */
  private parse256ColorOrRGB(codes: number[], startIndex: number, property: 'color' | 'backgroundColor'): number {
    if (startIndex + 1 >= codes.length) return startIndex

    const type = codes[startIndex + 1]

    if (type === 5) { // 256-color
      if (startIndex + 2 >= codes.length) return startIndex + 1
      const colorIndex = codes[startIndex + 2]
      this.currentStyle[property] = this.get256Color(colorIndex)
      return startIndex + 2

    } else if (type === 2) { // RGB
      if (startIndex + 4 >= codes.length) return startIndex + 1
      const r = codes[startIndex + 2]
      const g = codes[startIndex + 3]
      const b = codes[startIndex + 4]
      this.currentStyle[property] = `rgb(${r}, ${g}, ${b})`
      return startIndex + 4
    }

    return startIndex + 1
  }

  /**
   * 256색 팔레트에서 색상 반환
   */
  private get256Color(index: number): string {
    if (index < 16) {
      // Standard 16 colors
      const standardCodes = [30, 31, 32, 33, 34, 35, 36, 37, 90, 91, 92, 93, 94, 95, 96, 97]
      return ANSI_COLORS[standardCodes[index] as keyof typeof ANSI_COLORS] || '#FFFFFF'
    }

    if (index < 232) {
      // 216 color cube (6x6x6)
      const colorIndex = index - 16
      const r = Math.floor(colorIndex / 36)
      const g = Math.floor((colorIndex % 36) / 6)
      const b = colorIndex % 6

      const toHex = (n: number) => {
        const values = [0, 95, 135, 175, 215, 255]
        return values[n].toString(16).padStart(2, '0')
      }

      return `#${toHex(r)}${toHex(g)}${toHex(b)}`
    }

    // Grayscale colors (232-255)
    const gray = 8 + (index - 232) * 10
    const hex = gray.toString(16).padStart(2, '0')
    return `#${hex}${hex}${hex}`
  }

  /**
   * 스타일 객체를 CSS 스타일 문자열로 변환
   */
  static styleToCss(style: AnsiStyle): string {
    const cssProperties: string[] = []

    if (style.color) {
      cssProperties.push(`color: ${style.color}`)
    }

    if (style.backgroundColor) {
      cssProperties.push(`background-color: ${style.backgroundColor}`)
    }

    if (style.bold) {
      cssProperties.push('font-weight: bold')
    }

    if (style.dim) {
      cssProperties.push('opacity: 0.5')
    }

    if (style.italic) {
      cssProperties.push('font-style: italic')
    }

    if (style.underline) {
      cssProperties.push('text-decoration: underline')
    }

    if (style.strikethrough) {
      cssProperties.push('text-decoration: line-through')
    }

    return cssProperties.join('; ')
  }

  /**
   * HTML 특수 문자 이스케이프
   */
  static escapeHtml(text: string): string {
    const htmlEntities: Record<string, string> = {
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#39;',
      ' ': '&nbsp;',
    }

    return text.replace(/[&<>"' ]/g, (match) => htmlEntities[match] || match)
  }

  /**
   * 전체 텍스트를 HTML로 변환
   */
  toHtml(text: string): string {
    const segments = this.parse(text)

    return segments.map(segment => {
      const escapedText = AnsiParser.escapeHtml(segment.text)
      const cssStyle = AnsiParser.styleToCss(segment.style)

      if (cssStyle) {
        return `<span style="${cssStyle}">${escapedText}</span>`
      }

      return escapedText
    }).join('')
  }

  /**
   * 파서 상태 리셋
   */
  reset(): void {
    this.currentStyle = {}
  }
}

// 싱글톤 인스턴스 제공
export const ansiParser = new AnsiParser()