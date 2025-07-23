import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export interface DockerContainer {
  id: string
  name: string
  image: string
  status: 'running' | 'stopped' | 'paused' | 'restarting' | 'removing' | 'dead' | 'created'
  state: string
  ports: DockerPort[]
  mounts: DockerMount[]
  createdAt: string
  startedAt?: string
  finishedAt?: string
  workspaceId?: string
  environment?: Record<string, string>
}

export interface DockerPort {
  privatePort: number
  publicPort?: number
  type: 'tcp' | 'udp'
  ip?: string
}

export interface DockerMount {
  source: string
  destination: string
  mode: string
  type: 'bind' | 'volume' | 'tmpfs'
}

export interface DockerStats {
  containerId: string
  cpuPercent: number
  memoryUsage: number
  memoryLimit: number
  memoryPercent: number
  networkRx: number
  networkTx: number
  blockRead: number
  blockWrite: number
  pids: number
  timestamp: string
}

export interface DockerImage {
  id: string
  repository: string
  tag: string
  size: number
  created: string
}

export interface DockerNetwork {
  id: string
  name: string
  driver: string
  scope: string
  created: string
  containers: Record<string, any>
}

export interface ContainerLogs {
  containerId: string
  logs: LogEntry[]
  isStreaming: boolean
}

export interface LogEntry {
  timestamp: Date
  stream: 'stdout' | 'stderr'
  message: string
}

export interface DockerStatusMessage {
  type: 'container_status' | 'container_stats' | 'container_logs' | 'container_list'
  containerId?: string
  workspaceId?: string
  status?: string
  stats?: DockerStats
  logs?: LogEntry[]
  containers?: DockerContainer[]
}

export const useDockerStore = defineStore('docker', () => {
  // 상태
  const containers = ref<DockerContainer[]>([])
  const images = ref<DockerImage[]>([])
  const networks = ref<DockerNetwork[]>([])
  const stats = ref<Map<string, DockerStats>>(new Map())
  const containerLogs = ref<Map<string, ContainerLogs>>(new Map())
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  
  // WebSocket 모니터링 상태
  const isMonitoring = ref(false)
  const wsConnection = ref<WebSocket | null>(null)
  const connectionStatus = ref<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected')

  // 계산된 속성
  const runningContainers = computed(() =>
    containers.value.filter(c => c.status === 'running'),
  )

  const stoppedContainers = computed(() =>
    containers.value.filter(c => c.status === 'stopped'),
  )

  const containersByWorkspace = computed(() => (workspaceId: string) =>
    containers.value.filter(c => c.workspaceId === workspaceId),
  )

  const containerById = computed(() => (id: string) =>
    containers.value.find(c => c.id === id),
  )

  const totalContainers = computed(() => containers.value.length)

  const totalImages = computed(() => images.value.length)

  const totalNetworks = computed(() => networks.value.length)

  // 로그 관련 계산된 속성
  const getContainerLogs = computed(() => (containerId: string) => 
    containerLogs.value.get(containerId)
  )

  // 액션
  const setContainers = (containerList: DockerContainer[]) => {
    containers.value = containerList
  }

  const setImages = (imageList: DockerImage[]) => {
    images.value = imageList
  }

  const setNetworks = (networkList: DockerNetwork[]) => {
    networks.value = networkList
  }

  const addContainer = (container: DockerContainer) => {
    const existingIndex = containers.value.findIndex(c => c.id === container.id)
    if (existingIndex !== -1) {
      containers.value[existingIndex] = container
    } else {
      containers.value.push(container)
    }
  }

  const updateContainer = (id: string, updates: Partial<DockerContainer>) => {
    const index = containers.value.findIndex(c => c.id === id)
    if (index !== -1) {
      containers.value[index] = { ...containers.value[index], ...updates }
    }
  }

  const removeContainer = (id: string) => {
    containers.value = containers.value.filter(c => c.id !== id)
    stats.value.delete(id)
  }

  const setStats = (containerId: string, containerStats: DockerStats) => {
    stats.value.set(containerId, containerStats)
  }

  const setLoading = (loading: boolean) => {
    isLoading.value = loading
  }

  const setError = (errorMessage: string | null) => {
    error.value = errorMessage
  }

  // 실시간 모니터링 시작
  const startMonitoring = async (workspaceId?: string): Promise<void> => {
    if (isMonitoring.value) return

    try {
      connectionStatus.value = 'connecting'
      
      // WebSocket 연결 URL 구성
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const host = window.location.host
      const wsUrl = `${protocol}//${host}/ws/docker-status${workspaceId ? `/${workspaceId}` : ''}`
      
      wsConnection.value = new WebSocket(wsUrl)
      
      wsConnection.value.onopen = () => {
        connectionStatus.value = 'connected'
        isMonitoring.value = true
        console.log('Docker monitoring WebSocket connected')
      }
      
      wsConnection.value.onmessage = (event) => {
        try {
          const data: DockerStatusMessage = JSON.parse(event.data)
          handleDockerStatusUpdate(data)
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }
      
      wsConnection.value.onerror = (error) => {
        connectionStatus.value = 'error'
        console.error('Docker monitoring WebSocket error:', error)
      }
      
      wsConnection.value.onclose = () => {
        connectionStatus.value = 'disconnected'
        isMonitoring.value = false
        wsConnection.value = null
        console.log('Docker monitoring WebSocket disconnected')
      }
      
    } catch (err) {
      connectionStatus.value = 'error'
      setError(err instanceof Error ? err.message : 'WebSocket 연결 실패')
    }
  }

  // 모니터링 중지
  const stopMonitoring = (): void => {
    if (wsConnection.value) {
      wsConnection.value.close()
      wsConnection.value = null
    }
    isMonitoring.value = false
    connectionStatus.value = 'disconnected'
  }

  // WebSocket 메시지 처리
  const handleDockerStatusUpdate = (data: DockerStatusMessage): void => {
    switch (data.type) {
      case 'container_status':
        if (data.containerId && data.status) {
          updateContainer(data.containerId, { 
            status: data.status as DockerContainer['status'],
            startedAt: data.status === 'running' ? new Date().toISOString() : undefined,
            finishedAt: ['stopped', 'dead'].includes(data.status) ? new Date().toISOString() : undefined
          })
        }
        break
      
      case 'container_stats':
        if (data.containerId && data.stats) {
          setStats(data.containerId, data.stats)
        }
        break
      
      case 'container_logs':
        if (data.containerId && data.logs) {
          appendContainerLogs(data.containerId, data.logs)
        }
        break
      
      case 'container_list':
        if (data.containers) {
          setContainers(data.containers)
        }
        break
    }
  }

  // 컨테이너 로그 추가
  const appendContainerLogs = (containerId: string, newLogs: LogEntry[]): void => {
    const existingLogs = containerLogs.value.get(containerId)
    if (existingLogs) {
      existingLogs.logs.push(...newLogs)
      // 최대 1000개 로그만 유지
      if (existingLogs.logs.length > 1000) {
        existingLogs.logs = existingLogs.logs.slice(-1000)
      }
    } else {
      containerLogs.value.set(containerId, {
        containerId,
        logs: newLogs,
        isStreaming: false
      })
    }
  }

  // 로그 스트리밍 시작/중지
  const startLogStreaming = (containerId: string): void => {
    const logs = containerLogs.value.get(containerId)
    if (logs) {
      logs.isStreaming = true
    }
  }

  const stopLogStreaming = (containerId: string): void => {
    const logs = containerLogs.value.get(containerId)
    if (logs) {
      logs.isStreaming = false
    }
  }

  // 컨테이너 목록 새로고침 (더미 데이터 추가)
  const refreshContainers = async (): Promise<boolean> => {
    try {
      setLoading(true)
      setError(null)

      // TODO: API 호출로 컨테이너 목록 가져오기
      // 더미 데이터 추가
      const dummyContainers: DockerContainer[] = [
        {
          id: 'container-aicli-web',
          name: 'aicli-web-dev',
          image: 'node:18-alpine',
          status: 'running',
          state: 'running',
          workspaceId: '1',
          ports: [
            { privatePort: 3000, publicPort: 3000, type: 'tcp' },
            { privatePort: 8080, publicPort: 8080, type: 'tcp' }
          ],
          mounts: [
            { source: '/workspace/aicli-web', destination: '/app', mode: 'rw', type: 'bind' }
          ],
          createdAt: '2025-07-23T10:00:00Z',
          startedAt: '2025-07-23T10:05:00Z',
          environment: {
            NODE_ENV: 'development',
            PORT: '3000'
          }
        },
        {
          id: 'container-sample-project',
          name: 'sample-project-dev',
          image: 'node:20-alpine',
          status: 'stopped',
          state: 'exited',
          workspaceId: '2',
          ports: [
            { privatePort: 3000, publicPort: 3001, type: 'tcp' }
          ],
          mounts: [
            { source: '/workspace/sample-project', destination: '/app', mode: 'rw', type: 'bind' }
          ],
          createdAt: '2025-07-22T14:00:00Z',
          finishedAt: '2025-07-22T18:30:00Z'
        }
      ]
      
      setContainers(dummyContainers)

      // 더미 통계 데이터
      setStats('container-aicli-web', {
        containerId: 'container-aicli-web',
        cpuPercent: 15.5,
        memoryUsage: 134217728, // 128MB
        memoryLimit: 536870912, // 512MB
        memoryPercent: 25,
        networkRx: 1024000,
        networkTx: 2048000,
        blockRead: 512000,
        blockWrite: 256000,
        pids: 10,
        timestamp: new Date().toISOString()
      })

      // 더미 로그 데이터
      containerLogs.value.set('container-aicli-web', {
        containerId: 'container-aicli-web',
        logs: [
          {
            timestamp: new Date('2025-07-23T12:30:00'),
            stream: 'stdout',
            message: '[INFO] Server started on port 3000'
          },
          {
            timestamp: new Date('2025-07-23T12:31:00'),
            stream: 'stdout',
            message: '[INFO] WebSocket server listening on port 8080'
          },
          {
            timestamp: new Date('2025-07-23T12:32:00'),
            stream: 'stderr',
            message: '[WARN] High memory usage detected'
          }
        ],
        isStreaming: false
      })

      console.log('Container list refreshed with dummy data')
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to refresh containers'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 컨테이너 시작
  const startContainer = async (id: string): Promise<boolean> => {
    try {
      setLoading(true)
      updateContainer(id, { status: 'running', startedAt: new Date().toISOString() })

      // TODO: API 호출로 컨테이너 시작
      console.log(`Starting container ${id}`)
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to start container'
      setError(errorMessage)
      updateContainer(id, { status: 'stopped' })
      return false
    } finally {
      setLoading(false)
    }
  }

  // 컨테이너 중지
  const stopContainer = async (id: string): Promise<boolean> => {
    try {
      setLoading(true)
      updateContainer(id, { status: 'stopped', finishedAt: new Date().toISOString() })

      // TODO: API 호출로 컨테이너 중지
      console.log(`Stopping container ${id}`)
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to stop container'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 컨테이너 재시작
  const restartContainer = async (id: string): Promise<boolean> => {
    try {
      setLoading(true)
      updateContainer(id, { status: 'restarting' })

      // TODO: API 호출로 컨테이너 재시작
      console.log(`Restarting container ${id}`)

      // 잠시 후 running 상태로 변경
      setTimeout(() => {
        updateContainer(id, { status: 'running', startedAt: new Date().toISOString() })
      }, 1000)

      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to restart container'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 컨테이너 삭제
  const removeContainerById = async (id: string, force = false): Promise<boolean> => {
    try {
      setLoading(true)

      if (!force) {
        updateContainer(id, { status: 'removing' })
      }

      // TODO: API 호출로 컨테이너 삭제
      console.log(`Removing container ${id} (force: ${force})`)
      removeContainer(id)
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to remove container'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 컨테이너 로그 가져오기
  const getContainerLogs = async (id: string, tail?: number): Promise<string[]> => {
    try {
      // TODO: API 호출로 컨테이너 로그 가져오기
      console.log(`Getting logs for container ${id} (tail: ${tail})`)
      return [`Sample log line 1 for ${id}`, `Sample log line 2 for ${id}`]
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to get container logs'
      setError(errorMessage)
      return []
    }
  }

  // 이미지 목록 새로고침
  const refreshImages = async (): Promise<boolean> => {
    try {
      setLoading(true)
      setError(null)

      // TODO: API 호출로 이미지 목록 가져오기
      console.log('Refreshing image list...')
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to refresh images'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 이미지 풀
  const pullImage = async (imageName: string): Promise<boolean> => {
    try {
      setLoading(true)
      setError(null)

      // TODO: API 호출로 이미지 풀
      console.log(`Pulling image: ${imageName}`)
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : `Failed to pull image: ${imageName}`
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 통계 업데이트 시작/중지
  const startStatsMonitoring = (containerId: string) => {
    // TODO: WebSocket 또는 Server-Sent Events를 통한 실시간 통계 수신
    console.log(`Starting stats monitoring for container ${containerId}`)
  }

  const stopStatsMonitoring = (containerId: string) => {
    // TODO: 통계 모니터링 중지
    console.log(`Stopping stats monitoring for container ${containerId}`)
    stats.value.delete(containerId)
  }

  // 네트워크 목록 새로고침
  const refreshNetworks = async (): Promise<boolean> => {
    try {
      setLoading(true)
      setError(null)

      // TODO: API 호출로 네트워크 목록 가져오기
      console.log('Refreshing network list...')
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to refresh networks'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 이미지 삭제
  const removeImage = async (id: string): Promise<boolean> => {
    try {
      setLoading(true)

      // TODO: API 호출로 이미지 삭제
      console.log(`Removing image ${id}`)
      images.value = images.value.filter(i => i.id !== id)
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to remove image'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 컨테이너 정리
  const cleanupContainers = async (): Promise<boolean> => {
    try {
      setLoading(true)

      // TODO: API 호출로 중지된 컨테이너 정리
      console.log('Cleaning up stopped containers')
      containers.value = containers.value.filter(c => c.status !== 'stopped')
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to cleanup containers'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 이미지 정리
  const cleanupImages = async (): Promise<boolean> => {
    try {
      setLoading(true)

      // TODO: API 호출로 사용하지 않는 이미지 정리
      console.log('Cleaning up unused images')
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to cleanup images'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  return {
    // 상태
    containers,
    images,
    networks,
    stats,
    containerLogs,
    isLoading,
    error,
    isMonitoring,
    connectionStatus,

    // 계산된 속성
    runningContainers,
    stoppedContainers,
    containersByWorkspace,
    containerById,
    totalContainers,
    totalImages,
    totalNetworks,
    getContainerLogs,

    // 액션
    setContainers,
    setImages,
    setNetworks,
    addContainer,
    updateContainer,
    removeContainer,
    setStats,
    setLoading,
    setError,
    refreshContainers,
    startContainer,
    stopContainer,
    restartContainer,
    removeContainerById,
    refreshImages,
    pullImage,
    startStatsMonitoring,
    stopStatsMonitoring,
    refreshNetworks,
    removeImage,
    cleanupContainers,
    cleanupImages,

    // 새로운 액션 (WebSocket 모니터링)
    startMonitoring,
    stopMonitoring,
    appendContainerLogs,
    startLogStreaming,
    stopLogStreaming,
  }
})