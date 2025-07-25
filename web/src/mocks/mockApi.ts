import type { AxiosRequestConfig, AxiosResponse } from 'axios'

// Mock 응답 타입 정의
export interface MockResponse {
  status: number
  data: any
  headers?: Record<string, string>
  delay?: number
}

export interface MockRule {
  method: string
  url: string | RegExp
  response: MockResponse | ((config: AxiosRequestConfig) => MockResponse)
  condition?: (config: AxiosRequestConfig) => boolean
}

// Mock 데이터
const mockUsers = [
  {
    id: '1',
    email: 'admin@example.com',
    name: 'Admin User',
    roles: ['admin', 'user'],
    avatar: null,
    createdAt: '2024-01-01T00:00:00Z',
    lastLoginAt: '2025-01-01T12:00:00Z'
  },
  {
    id: '2',
    email: 'user@example.com',
    name: 'Regular User',
    roles: ['user'],
    avatar: null,
    createdAt: '2024-06-01T00:00:00Z',
    lastLoginAt: '2025-01-01T10:30:00Z'
  }
]

const mockSessions = [
  {
    id: 'session-1',
    userId: '1',
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
    location: 'Seoul, South Korea',
    createdAt: '2025-01-01T12:00:00Z',
    lastAccessAt: '2025-01-01T13:30:00Z',
    expiresAt: '2025-01-02T12:00:00Z',
    isActive: true
  },
  {
    id: 'session-2',
    userId: '2',
    ipAddress: '192.168.1.101',
    userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36',
    location: 'Busan, South Korea',
    createdAt: '2025-01-01T10:30:00Z',
    lastAccessAt: '2025-01-01T13:45:00Z',
    expiresAt: '2025-01-02T10:30:00Z',
    isActive: true
  }
]

const mockLoginHistory = [
  {
    id: 'login-1',
    userId: '1',
    timestamp: '2025-01-01T12:00:00Z',
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
    location: 'Seoul, South Korea',
    success: true,
    riskScore: 0.1,
    method: 'email'
  },
  {
    id: 'login-2',
    userId: '2',
    timestamp: '2025-01-01T10:30:00Z',
    ipAddress: '192.168.1.101',
    userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36',
    location: 'Busan, South Korea',
    success: true,
    riskScore: 0.2,
    method: 'google'
  },
  {
    id: 'login-3',
    userId: '1',
    timestamp: '2024-12-31T23:45:00Z',
    ipAddress: '203.0.113.1',
    userAgent: 'curl/7.68.0',
    location: 'Unknown',
    success: false,
    riskScore: 0.9,
    method: 'email',
    failureReason: 'Invalid password'
  }
]

const mockSecurityEvents = [
  {
    id: 'event-1',
    type: 'suspicious_login',
    severity: 'high',
    title: '의심스러운 로그인 시도',
    description: '알려지지 않은 위치에서 여러 번의 로그인 실패',
    userId: '1',
    ipAddress: '203.0.113.1',
    timestamp: '2024-12-31T23:45:00Z',
    resolved: false,
    metadata: {
      attempts: 5,
      location: 'Unknown',
      riskScore: 0.9
    }
  },
  {
    id: 'event-2',
    type: 'password_change',
    severity: 'medium',
    title: '비밀번호 변경',
    description: '사용자가 비밀번호를 변경했습니다',
    userId: '2',
    ipAddress: '192.168.1.101',
    timestamp: '2024-12-30T14:20:00Z',
    resolved: true,
    metadata: {
      method: 'user_initiated'
    }
  }
]

// Mock 규칙 정의
export const mockRules: MockRule[] = [
  // Health Check
  {
    method: 'HEAD',
    url: '/api/health',
    response: {
      status: 200,
      data: null,
      delay: 100
    }
  },

  // 인증 관련
  {
    method: 'POST',
    url: '/api/auth/login',
    response: (config) => {
      const { email, password } = JSON.parse(config.data || '{}')
      
      if (email === 'admin@example.com' && password === 'admin123') {
        return {
          status: 200,
          data: {
            success: true,
            data: {
              user: mockUsers[0],
              token: 'mock-jwt-token-admin',
              refreshToken: 'mock-refresh-token-admin',
              expiresIn: 3600
            },
            message: '로그인 성공'
          },
          delay: 800
        }
      } else if (email === 'user@example.com' && password === 'user123') {
        return {
          status: 200,
          data: {
            success: true,
            data: {
              user: mockUsers[1],
              token: 'mock-jwt-token-user',
              refreshToken: 'mock-refresh-token-user',
              expiresIn: 3600
            },
            message: '로그인 성공'
          },
          delay: 800
        }
      } else {
        return {
          status: 401,
          data: {
            success: false,
            message: '이메일 또는 비밀번호가 올바르지 않습니다'
          },
          delay: 1000
        }
      }
    }
  },

  {
    method: 'POST',
    url: '/api/auth/refresh',
    response: {
      status: 200,
      data: {
        success: true,
        data: {
          token: 'new-mock-jwt-token',
          expiresIn: 3600
        }
      },
      delay: 300
    }
  },

  {
    method: 'POST',
    url: '/api/auth/logout',
    response: {
      status: 200,
      data: {
        success: true,
        message: '로그아웃 완료'
      },
      delay: 200
    }
  },

  // 사용자 정보
  {
    method: 'GET',
    url: '/api/user/profile',
    response: {
      status: 200,
      data: {
        success: true,
        data: mockUsers[0]
      },
      delay: 300
    }
  },

  {
    method: 'PUT',
    url: '/api/user/profile',
    response: (config) => {
      const updates = JSON.parse(config.data || '{}')
      return {
        status: 200,
        data: {
          success: true,
          data: { ...mockUsers[0], ...updates },
          message: '프로필이 업데이트되었습니다'
        },
        delay: 500
      }
    }
  },

  // 세션 관리
  {
    method: 'GET',
    url: '/api/user/sessions',
    response: {
      status: 200,
      data: {
        success: true,
        data: mockSessions
      },
      delay: 400
    }
  },

  {
    method: 'DELETE',
    url: /\/api\/user\/sessions\/(.+)/,
    response: {
      status: 200,
      data: {
        success: true,
        message: '세션이 종료되었습니다'
      },
      delay: 300
    }
  },

  // 로그인 기록
  {
    method: 'GET',
    url: '/api/user/login-history',
    response: (config) => {
      const params = new URLSearchParams(config.params)
      const page = parseInt(params.get('page') || '1')
      const limit = parseInt(params.get('limit') || '10')
      const startIndex = (page - 1) * limit
      const endIndex = startIndex + limit
      
      return {
        status: 200,
        data: {
          success: true,
          data: {
            items: mockLoginHistory.slice(startIndex, endIndex),
            pagination: {
              page,
              limit,
              total: mockLoginHistory.length,
              totalPages: Math.ceil(mockLoginHistory.length / limit)
            }
          }
        },
        delay: 600
      }
    }
  },

  // 보안 이벤트
  {
    method: 'GET',
    url: '/api/security/events',
    response: {
      status: 200,
      data: {
        success: true,
        data: mockSecurityEvents
      },
      delay: 500
    }
  },

  {
    method: 'PUT',
    url: /\/api\/security\/events\/(.+)\/resolve/,
    response: {
      status: 200,
      data: {
        success: true,
        message: '보안 이벤트가 해결로 표시되었습니다'
      },
      delay: 300
    }
  },

  // 보안 통계
  {
    method: 'GET',
    url: '/api/security/stats',
    response: {
      status: 200,
      data: {
        success: true,
        data: {
          totalLogins: 156,
          failedLogins: 12,
          suspiciousActivities: 3,
          activeSecurityEvents: 1,
          riskScore: 0.3
        }
      },
      delay: 400
    }
  },

  // OAuth 관련
  {
    method: 'GET',
    url: '/api/auth/google/url',
    response: {
      status: 200,
      data: {
        success: true,
        data: {
          url: 'https://accounts.google.com/oauth/authorize?mock=true'
        }
      },
      delay: 200
    }
  },

  {
    method: 'POST',
    url: '/api/auth/google/callback',
    response: {
      status: 200,
      data: {
        success: true,
        data: {
          user: mockUsers[1],
          token: 'mock-google-jwt-token',
          refreshToken: 'mock-google-refresh-token',
          expiresIn: 3600
        }
      },
      delay: 1000
    }
  },

  // 에러 시뮬레이션 (개발/테스트용)
  {
    method: 'GET',
    url: '/api/test/500',
    response: {
      status: 500,
      data: {
        success: false,
        message: 'Internal Server Error (Mock)'
      },
      delay: 1000
    }
  },

  {
    method: 'GET',
    url: '/api/test/timeout',
    response: {
      status: 408,
      data: {
        success: false,
        message: 'Request Timeout (Mock)'
      },
      delay: 5000
    }
  }
]

// Mock API 매처
export class MockApiMatcher {
  private isEnabled = false

  constructor() {
    // 개발 환경에서만 기본 활성화
    this.isEnabled = import.meta.env.DEV && import.meta.env.VITE_USE_MOCK_API !== 'false'
  }

  enable(): void {
    this.isEnabled = true
    console.log('🎭 Mock API enabled')
  }

  disable(): void {
    this.isEnabled = false
    console.log('🎭 Mock API disabled')
  }

  toggle(): void {
    this.isEnabled = !this.isEnabled
    console.log(`🎭 Mock API ${this.isEnabled ? 'enabled' : 'disabled'}`)
  }

  isActive(): boolean {
    return this.isEnabled
  }

  async matchRequest(config: AxiosRequestConfig): Promise<AxiosResponse | null> {
    if (!this.isEnabled) return null

    const method = config.method?.toUpperCase() || 'GET'
    const url = config.url || ''

    for (const rule of mockRules) {
      if (this.matchRule(rule, method, url, config)) {
        const response = typeof rule.response === 'function' 
          ? rule.response(config) 
          : rule.response

        // 지연 시뮬레이션
        if (response.delay) {
          await new Promise(resolve => setTimeout(resolve, response.delay))
        }

        console.log(`🎭 Mock API matched: ${method} ${url}`, {
          status: response.status,
          delay: response.delay
        })

        return {
          data: response.data,
          status: response.status,
          statusText: this.getStatusText(response.status),
          headers: response.headers || {},
          config,
          request: {}
        } as AxiosResponse
      }
    }

    return null
  }

  private matchRule(rule: MockRule, method: string, url: string, config: AxiosRequestConfig): boolean {
    // 메소드 확인
    if (rule.method.toUpperCase() !== method) return false

    // URL 매칭
    if (typeof rule.url === 'string') {
      if (!url.includes(rule.url)) return false
    } else if (rule.url instanceof RegExp) {
      if (!rule.url.test(url)) return false
    }

    // 추가 조건 확인
    if (rule.condition && !rule.condition(config)) return false

    return true
  }

  private getStatusText(status: number): string {
    const statusTexts: Record<number, string> = {
      200: 'OK',
      201: 'Created',
      400: 'Bad Request',
      401: 'Unauthorized',
      403: 'Forbidden',
      404: 'Not Found',
      408: 'Request Timeout',
      429: 'Too Many Requests',
      500: 'Internal Server Error',
      502: 'Bad Gateway',
      503: 'Service Unavailable'
    }
    return statusTexts[status] || 'Unknown'
  }
}

// 전역 인스턴스
export const mockApiMatcher = new MockApiMatcher()

// 개발자 도구에서 사용할 수 있도록 전역 노출
if (typeof window !== 'undefined' && import.meta.env.DEV) {
  (window as any).__mockApi = mockApiMatcher
}