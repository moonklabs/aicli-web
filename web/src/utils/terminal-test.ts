// 터미널 기능 테스트 유틸리티
// 개발 환경에서 터미널 기능을 테스트하기 위한 헬퍼 함수들

import { AnsiParser } from './ansi-parser'
import type { TerminalLog } from '@/stores/terminal'

/**
 * ANSI 파서 테스트 케이스
 */
export function testAnsiParser(): boolean {
  console.group('🧪 ANSI Parser 테스트')

  const testCases = [
    {
      name: '기본 색상 테스트',
      input: '\x1b[31mRed text\x1b[0m \x1b[32mGreen text\x1b[0m',
      expectAnsi: true,
      expectSegments: 3,
    },
    {
      name: '볼드 및 이탤릭 테스트',
      input: '\x1b[1mBold\x1b[0m \x1b[3mItalic\x1b[0m',
      expectAnsi: true,
      expectSegments: 3,
    },
    {
      name: '256색 테스트',
      input: '\x1b[38;5;196mBright Red\x1b[0m',
      expectAnsi: true,
      expectSegments: 2,
    },
    {
      name: 'RGB 색상 테스트',
      input: '\x1b[38;2;255;100;0mOrange\x1b[0m',
      expectAnsi: true,
      expectSegments: 2,
    },
    {
      name: '일반 텍스트 테스트',
      input: 'Normal text without ANSI codes',
      expectAnsi: false,
      expectSegments: 1,
    },
  ]

  let passedTests = 0

  testCases.forEach(testCase => {
    try {
      const result = new AnsiParser().parse(testCase.input)
      const hasAnsi = testCase.input.includes('\x1b[')

      const ansiMatch = hasAnsi === testCase.expectAnsi
      const segmentMatch = result.length === testCase.expectSegments

      if (ansiMatch && segmentMatch) {
        console.log(`✅ ${testCase.name}: PASS`)
        passedTests++
      } else {
        console.error(`❌ ${testCase.name}: FAIL`)
        console.error(`  Expected ANSI: ${testCase.expectAnsi}, Got: ${hasAnsi}`)
        console.error(`  Expected segments: ${testCase.expectSegments}, Got: ${result.length}`)
      }
    } catch (error) {
      console.error(`❌ ${testCase.name}: ERROR - ${error}`)
    }
  })

  const success = passedTests === testCases.length
  console.log(`\n📊 결과: ${passedTests}/${testCases.length} 테스트 통과`)
  console.groupEnd()

  return success
}

/**
 * 가상 Claude 출력 생성기
 */
export function generateMockClaudeOutput(type: 'normal' | 'colored' | 'progress' | 'error' = 'normal'): string {
  const outputs = {
    normal: [
      'Starting Claude CLI...',
      'Processing file: example.ts',
      'Found 15 issues to address',
      'Applying fixes...',
      'Fix applied: Remove unused imports',
      'Fix applied: Add type annotations',
      'All fixes completed successfully!',
    ],
    colored: [
      '\x1b[32m✓\x1b[0m Starting Claude CLI...',
      '\x1b[33m⚠\x1b[0m Processing file: \x1b[36mexample.ts\x1b[0m',
      '\x1b[31m✗\x1b[0m Found \x1b[1m15\x1b[0m issues to address',
      '\x1b[32m✓\x1b[0m Applying fixes...',
      '\x1b[32m✓\x1b[0m Fix applied: \x1b[33mRemove unused imports\x1b[0m',
      '\x1b[32m✓\x1b[0m Fix applied: \x1b[33mAdd type annotations\x1b[0m',
      '\x1b[1m\x1b[32m✓ All fixes completed successfully!\x1b[0m',
    ],
    progress: [
      'Progress: [████████░░] 80% (4/5 files)',
      '\x1b[2K\rProgress: [█████████░] 90% (4.5/5 files)',
      '\x1b[2K\rProgress: [██████████] 100% (5/5 files)',
      '\x1b[32m✓ Process completed!\x1b[0m',
    ],
    error: [
      '\x1b[31mError:\x1b[0m Failed to parse file',
      '\x1b[31mStack trace:\x1b[0m',
      '  at parseFile (parser.js:42:15)',
      '  at processFile (main.js:23:8)',
      '\x1b[33mWarning:\x1b[0m Continuing with next file...',
    ],
  }

  return outputs[type][Math.floor(Math.random() * outputs[type].length)]
}

/**
 * 모의 터미널 로그 생성
 */
export function generateMockTerminalLogs(count = 50): TerminalLog[] {
  const logs: TerminalLog[] = []
  const startTime = Date.now() - (count * 1000) // 과거 시간부터 시작

  for (let i = 0; i < count; i++) {
    const timestamp = new Date(startTime + (i * 1000)).toISOString()
    const logTypes: TerminalLog['type'][] = ['output', 'output', 'output', 'error', 'system']
    const logLevels: TerminalLog['level'][] = ['info', 'info', 'warn', 'error', 'debug']

    const type = logTypes[Math.floor(Math.random() * logTypes.length)]
    const level = logLevels[Math.floor(Math.random() * logLevels.length)]

    let content: string
    if (type === 'error') {
      content = generateMockClaudeOutput('error')
    } else if (Math.random() > 0.7) {
      content = generateMockClaudeOutput('colored')
    } else {
      content = generateMockClaudeOutput('normal')
    }

    const log: TerminalLog = {
      id: `log_${i}`,
      timestamp,
      type,
      content,
      level,
    }

    // ANSI 파싱 적용
    const parsed = new AnsiParser().parse(content)
    const hasAnsi = content.includes('\x1b[')
    if (hasAnsi) {
      log.parsed = {
        raw: content,
        html: new AnsiParser().toHtml(content),
        plainText: content.replace(/\x1b\[[0-9;]*m/g, ''),
        hasAnsi: true,
      }
    }

    logs.push(log)
  }

  return logs
}

/**
 * WebSocket 연결 시뮬레이션
 */
export class MockClaudeStream {
  private callbacks: Map<string, ((data: any) => void)[]> = new Map()
  private connected = false
  private interval: NodeJS.Timeout | null = null

  connect(): Promise<void> {
    return new Promise((resolve) => {
      setTimeout(() => {
        this.connected = true
        this.emit('statusChange', 'connected')
        this.emit('sessionEvent', {
          type: 'user_joined',
          user_id: 'test_user',
          user_name: 'Test User',
          timestamp: new Date().toISOString(),
        })
        resolve()
      }, 1000)
    })
  }

  disconnect(): void {
    this.connected = false
    if (this.interval) {
      clearInterval(this.interval)
      this.interval = null
    }
    this.emit('statusChange', 'disconnected')
  }

  executeClaudeCommand(command: string): boolean {
    if (!this.connected) return false

    this.emit('claude_message', {
      type: 'system',
      content: `Executing command: ${command}`,
      timestamp: new Date().toISOString(),
      message_id: `msg_${Date.now()}`,
    })

    // 시뮬레이션된 출력 생성
    this.startOutputSimulation()

    return true
  }

  private startOutputSimulation(): void {
    let outputCount = 0
    const maxOutputs = 10 + Math.floor(Math.random() * 20)

    this.interval = setInterval(() => {
      if (outputCount >= maxOutputs) {
        this.stopOutputSimulation()
        return
      }

      const outputType = Math.random() > 0.9 ? 'error' : 'output'
      const content = generateMockClaudeOutput(
        Math.random() > 0.7 ? 'colored' : 'normal',
      )

      this.emit('claude_message', {
        type: outputType,
        content,
        timestamp: new Date().toISOString(),
        message_id: `msg_${Date.now()}_${outputCount}`,
      })

      outputCount++
    }, 100 + Math.random() * 500) // 100-600ms 간격
  }

  private stopOutputSimulation(): void {
    if (this.interval) {
      clearInterval(this.interval)
      this.interval = null
    }

    this.emit('claude_message', {
      type: 'completed',
      content: JSON.stringify({ status: 'success', exit_code: 0 }),
      timestamp: new Date().toISOString(),
      message_id: `msg_${Date.now()}_complete`,
    })
  }

  on(event: string, callback: (data: any) => void): () => void {
    if (!this.callbacks.has(event)) {
      this.callbacks.set(event, [])
    }

    this.callbacks.get(event)!.push(callback)

    return () => {
      const callbacks = this.callbacks.get(event)
      if (callbacks) {
        const index = callbacks.indexOf(callback)
        if (index > -1) {
          callbacks.splice(index, 1)
        }
      }
    }
  }

  private emit(event: string, data: any): void {
    const callbacks = this.callbacks.get(event)
    if (callbacks) {
      callbacks.forEach(callback => {
        try {
          callback(data)
        } catch (error) {
          console.error(`Error in ${event} callback:`, error)
        }
      })
    }
  }

  // 가짜 속성들 (실제 useClaudeStream과 호환성을 위해)
  get isConnected() {
    return { value: this.connected }
  }

  get status() {
    return { value: this.connected ? 'connected' : 'disconnected' }
  }

  send() { return true }
  reconnect() { return this.connect() }
  onStatusChange(callback: any) { return this.on('statusChange', callback) }
  onError(callback: any) { return this.on('error', callback) }
  onClaudeOutput(callback: any) { return this.on('claude_message', callback) }
  onClaudeError(callback: any) { return this.on('claude_message', callback) }
  onClaudeComplete(callback: any) { return this.on('claude_message', callback) }
  onClaudeFailed(callback: any) { return this.on('claude_message', callback) }
  onSessionEvent(callback: any) { return this.on('sessionEvent', callback) }
  onUserJoined(callback: any) { return this.on('sessionEvent', callback) }
  onUserLeft(callback: any) { return this.on('sessionEvent', callback) }
  sendUserInput() { return true }
  stopClaudeExecution() {
    this.stopOutputSimulation()
    return true
  }
  clearOutput() {}
  exportOutput() { return '' }
  processClaudeOutput(content: string) {
    return {
      raw: content,
      processed: content,
      hasAnsi: false,
      html: content,
      plainText: content,
    }
  }
}

/**
 * 성능 테스트
 */
export function testTerminalPerformance(logCount = 1000): void {
  console.group(`🚀 성능 테스트 (${logCount}개 로그)`)

  const startTime = performance.now()

  // 로그 생성
  const logs = generateMockTerminalLogs(logCount)
  const generationTime = performance.now() - startTime

  // ANSI 파싱 성능 테스트
  const parseStartTime = performance.now()
  logs.forEach(log => {
    new AnsiParser().parse(log.content)
  })
  const parseTime = performance.now() - parseStartTime

  // HTML 렌더링 성능 테스트
  const renderStartTime = performance.now()
  logs.forEach(log => {
    if (log.parsed?.hasAnsi) {
      new AnsiParser().toHtml(log.content)
    }
  })
  const renderTime = performance.now() - renderStartTime

  const totalTime = performance.now() - startTime

  console.log('📊 성능 결과:')
  console.log(`  로그 생성: ${generationTime.toFixed(2)}ms`)
  console.log(`  ANSI 파싱: ${parseTime.toFixed(2)}ms (${(parseTime / logCount).toFixed(3)}ms/log)`)
  console.log(`  HTML 렌더링: ${renderTime.toFixed(2)}ms (${(renderTime / logCount).toFixed(3)}ms/log)`)
  console.log(`  총 시간: ${totalTime.toFixed(2)}ms`)

  // 성능 평가
  const parsePerformance = parseTime / logCount
  const renderPerformance = renderTime / logCount

  if (parsePerformance < 0.1 && renderPerformance < 0.1) {
    console.log(`✅ 성능: 우수 (파싱: ${parsePerformance.toFixed(3)}ms, 렌더링: ${renderPerformance.toFixed(3)}ms)`)
  } else if (parsePerformance < 0.5 && renderPerformance < 0.5) {
    console.log(`⚠️ 성능: 양호 (파싱: ${parsePerformance.toFixed(3)}ms, 렌더링: ${renderPerformance.toFixed(3)}ms)`)
  } else {
    console.log(`❌ 성능: 저하 (파싱: ${parsePerformance.toFixed(3)}ms, 렌더링: ${renderPerformance.toFixed(3)}ms)`)
  }

  console.groupEnd()
}

/**
 * 전체 터미널 시스템 통합 테스트
 */
export function runTerminalIntegrationTest(): Promise<boolean> {
  return new Promise((resolve) => {
    console.group('🔧 터미널 통합 테스트')

    let testsCompleted = 0
    const totalTests = 4
    let allTestsPassed = true

    const checkCompletion = () => {
      testsCompleted++
      if (testsCompleted >= totalTests) {
        console.log(`\n📊 통합 테스트 결과: ${allTestsPassed ? '✅ 모든 테스트 통과' : '❌ 일부 테스트 실패'}`)
        console.groupEnd()
        resolve(allTestsPassed)
      }
    }

    // 1. ANSI 파서 테스트
    console.log('1️⃣ ANSI 파서 테스트...')
    const ansiResult = testAnsiParser()
    if (!ansiResult) allTestsPassed = false
    checkCompletion()

    // 2. 성능 테스트
    console.log('2️⃣ 성능 테스트...')
    testTerminalPerformance(500)
    checkCompletion()

    // 3. 모의 WebSocket 연결 테스트
    console.log('3️⃣ WebSocket 연결 테스트...')
    const mockStream = new MockClaudeStream()

    mockStream.on('statusChange', (status) => {
      console.log(`  연결 상태 변경: ${status}`)
      if (status === 'connected') {
        console.log('  ✅ WebSocket 연결 성공')

        // 명령 실행 테스트
        const executeResult = mockStream.executeClaudeCommand('test command')
        console.log(`  명령 실행: ${executeResult ? '✅ 성공' : '❌ 실패'}`)

        setTimeout(() => {
          mockStream.disconnect()
          checkCompletion()
        }, 2000)
      }
    })

    mockStream.connect().catch((error) => {
      console.error('  ❌ WebSocket 연결 실패:', error)
      allTestsPassed = false
      checkCompletion()
    })

    // 4. 메모리 사용량 테스트
    console.log('4️⃣ 메모리 사용량 테스트...')
    const initialMemory = (performance as any).memory?.usedJSHeapSize || 0
    generateMockTerminalLogs(5000) // 메모리 테스트용 로그 생성
    const memoryAfterGeneration = (performance as any).memory?.usedJSHeapSize || 0

    const memoryIncrease = memoryAfterGeneration - initialMemory
    console.log(`  메모리 증가량: ${(memoryIncrease / 1024 / 1024).toFixed(2)}MB`)

    if (memoryIncrease < 50 * 1024 * 1024) { // 50MB 미만
      console.log('  ✅ 메모리 사용량 양호')
    } else {
      console.log('  ⚠️ 메모리 사용량 주의')
    }

    checkCompletion()
  })
}

/**
 * 개발 환경에서 터미널 테스트 실행
 */
export function runDevelopmentTests(): void {
  if (import.meta.env.DEV) {
    console.log('🧪 개발 환경 터미널 테스트 시작...')

    // 콘솔에서 직접 호출할 수 있도록 전역 객체에 추가
    ;(window as any).terminalTest = {
      testAnsiParser,
      generateMockClaudeOutput,
      generateMockTerminalLogs,
      testTerminalPerformance,
      runTerminalIntegrationTest,
      MockClaudeStream,
    }

    console.log('💡 사용 가능한 테스트 함수들:')
    console.log('  - window.terminalTest.testAnsiParser()')
    console.log('  - window.terminalTest.testTerminalPerformance(1000)')
    console.log('  - window.terminalTest.runTerminalIntegrationTest()')
    console.log('  - window.terminalTest.generateMockTerminalLogs(50)')
  }
}