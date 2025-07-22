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

export const useDockerStore = defineStore('docker', () => {
  // 상태
  const containers = ref<DockerContainer[]>([])
  const images = ref<DockerImage[]>([])
  const networks = ref<DockerNetwork[]>([])
  const stats = ref<Map<string, DockerStats>>(new Map())
  const isLoading = ref(false)
  const error = ref<string | null>(null)

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

  // 컨테이너 목록 새로고침
  const refreshContainers = async (): Promise<boolean> => {
    try {
      setLoading(true)
      setError(null)

      // TODO: API 호출로 컨테이너 목록 가져오기
      console.log('Refreshing container list...')
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
    isLoading,
    error,

    // 계산된 속성
    runningContainers,
    stoppedContainers,
    containersByWorkspace,
    containerById,
    totalContainers,
    totalImages,
    totalNetworks,

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
    getContainerLogs,
    refreshImages,
    pullImage,
    startStatsMonitoring,
    stopStatsMonitoring,
    refreshNetworks,
    removeImage,
    cleanupContainers,
    cleanupImages,
  }
})