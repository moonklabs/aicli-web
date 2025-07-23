<template>
  <span
    v-for="segment in segments"
    :key="segment.index"
    :style="segment.style"
    :class="['ansi-segment', segment.className]"
    v-html="segment.html"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { type AnsiSegment, ansiParser } from '@/utils/ansi-parser'

interface Props {
  text: string
  preserveWhitespace?: boolean
}

interface RenderedSegment {
  index: number
  html: string
  style: Record<string, string>
  className: string[]
}

const props = withDefaults(defineProps<Props>(), {
  preserveWhitespace: true,
})

/**
 * ANSI 텍스트를 파싱하여 렌더링 가능한 세그먼트로 변환
 */
const segments = computed<RenderedSegment[]>(() => {
  if (!props.text) return []

  // ANSI 파서로 텍스트 파싱
  const ansiSegments: AnsiSegment[] = ansiParser.parse(props.text)

  return ansiSegments.map((segment, index) => {
    const style: Record<string, string> = {}
    const className: string[] = ['ansi-segment']

    // 색상 스타일 적용
    if (segment.style.color) {
      style.color = segment.style.color
    }

    if (segment.style.backgroundColor) {
      style.backgroundColor = segment.style.backgroundColor
    }

    // 텍스트 스타일 적용
    if (segment.style.bold) {
      style.fontWeight = 'bold'
      className.push('ansi-bold')
    }

    if (segment.style.italic) {
      style.fontStyle = 'italic'
      className.push('ansi-italic')
    }

    if (segment.style.underline) {
      style.textDecoration = `${style.textDecoration || ''} underline`
      className.push('ansi-underline')
    }

    if (segment.style.strikethrough) {
      style.textDecoration = `${style.textDecoration || ''} line-through`
      className.push('ansi-strikethrough')
    }

    if (segment.style.dim) {
      style.opacity = '0.5'
      className.push('ansi-dim')
    }

    // HTML 이스케이프 처리
    let html = escapeHtml(segment.text)

    // 공백 처리
    if (props.preserveWhitespace) {
      html = html.replace(/ /g, '&nbsp;')
    }

    // 줄바꿈 처리
    html = html.replace(/\n/g, '<br>')

    return {
      index,
      html,
      style,
      className,
    }
  })
})

/**
 * HTML 특수 문자 이스케이프
 */
function escapeHtml(text: string): string {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}
</script>

<style scoped>
.ansi-segment {
  font-family: 'Fira Code', 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 특별한 스타일이 필요한 경우를 위한 클래스들 */
.ansi-bold {
  font-weight: bold;
}

.ansi-italic {
  font-style: italic;
}

.ansi-underline {
  text-decoration: underline;
}

.ansi-strikethrough {
  text-decoration: line-through;
}

.ansi-dim {
  opacity: 0.5;
}

/* 하이퍼링크 감지 및 스타일링 */
.ansi-segment:deep(a) {
  color: #4A9EFF;
  text-decoration: underline;
}

.ansi-segment:deep(a:hover) {
  color: #66B3FF;
  text-decoration: none;
}

/* 선택 가능한 텍스트 스타일 */
.ansi-segment {
  user-select: text;
  -webkit-user-select: text;
  -moz-user-select: text;
  -ms-user-select: text;
}

/* 다크 테마 호환성 */
@media (prefers-color-scheme: dark) {
  .ansi-segment {
    color: #E5E5E5;
  }
}
</style>