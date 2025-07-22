import { apiDelete, apiGet, apiPost } from '@/api'
import type {
  DockerContainerInfo,
  DockerImageInfo,
  DockerStats,
} from '@/types/api'

export const dockerApi = {
  /**
   * 컨테이너 목록 조회
   */
  getContainers: async (params?: {
    all?: boolean
    status?: string
    workspaceId?: string
  }): Promise<DockerContainerInfo[]> => {
    const response = await apiGet<DockerContainerInfo[]>('/docker/containers', { params })
    return response.data.data
  },

  /**
   * 컨테이너 상세 정보 조회
   */
  getContainer: async (id: string): Promise<DockerContainerInfo> => {
    const response = await apiGet<DockerContainerInfo>(`/docker/containers/${id}`)
    return response.data.data
  },

  /**
   * 컨테이너 시작
   */
  startContainer: async (id: string): Promise<DockerContainerInfo> => {
    const response = await apiPost<DockerContainerInfo>(`/docker/containers/${id}/start`)
    return response.data.data
  },

  /**
   * 컨테이너 중지
   */
  stopContainer: async (id: string, force?: boolean): Promise<DockerContainerInfo> => {
    const response = await apiPost<DockerContainerInfo>(`/docker/containers/${id}/stop`, {
      force: force || false,
    })
    return response.data.data
  },

  /**
   * 컨테이너 재시작
   */
  restartContainer: async (id: string): Promise<DockerContainerInfo> => {
    const response = await apiPost<DockerContainerInfo>(`/docker/containers/${id}/restart`)
    return response.data.data
  },

  /**
   * 컨테이너 삭제
   */
  removeContainer: async (id: string, force?: boolean): Promise<void> => {
    await apiDelete(`/docker/containers/${id}`, {
      params: { force: force || false },
    })
  },

  /**
   * 컨테이너 로그 조회
   */
  getContainerLogs: async (id: string, params?: {
    tail?: number
    since?: string
    follow?: boolean
    timestamps?: boolean
  }): Promise<string[]> => {
    const response = await apiGet<string[]>(`/docker/containers/${id}/logs`, { params })
    return response.data.data
  },

  /**
   * 컨테이너 통계 조회
   */
  getContainerStats: async (id: string): Promise<DockerStats> => {
    const response = await apiGet<DockerStats>(`/docker/containers/${id}/stats`)
    return response.data.data
  },

  /**
   * 컨테이너에서 명령 실행
   */
  execCommand: async (id: string, command: string[], options?: {
    workingDir?: string
    environment?: Record<string, string>
    user?: string
    privileged?: boolean
    tty?: boolean
  }): Promise<{
    output: string
    exitCode: number
    error?: string
  }> => {
    const response = await apiPost(`/docker/containers/${id}/exec`, {
      command,
      ...options,
    })
    return response.data.data
  },

  /**
   * 이미지 목록 조회
   */
  getImages: async (params?: {
    all?: boolean
    dangling?: boolean
  }): Promise<DockerImageInfo[]> => {
    const response = await apiGet<DockerImageInfo[]>('/docker/images', { params })
    return response.data.data
  },

  /**
   * 이미지 상세 정보 조회
   */
  getImage: async (id: string): Promise<DockerImageInfo> => {
    const response = await apiGet<DockerImageInfo>(`/docker/images/${id}`)
    return response.data.data
  },

  /**
   * 이미지 풀
   */
  pullImage: async (imageName: string, tag?: string): Promise<{
    status: string
    progress: number
    message?: string
  }> => {
    const response = await apiPost('/docker/images/pull', {
      name: imageName,
      tag: tag || 'latest',
    })
    return response.data.data
  },

  /**
   * 이미지 삭제
   */
  removeImage: async (id: string, force?: boolean): Promise<void> => {
    await apiDelete(`/docker/images/${id}`, {
      params: { force: force || false },
    })
  },

  /**
   * 이미지 빌드
   */
  buildImage: async (data: {
    dockerfile: string
    context: string
    tags: string[]
    buildArgs?: Record<string, string>
  }): Promise<{
    buildId: string
    status: string
    logs: string[]
  }> => {
    const response = await apiPost('/docker/images/build', data)
    return response.data.data
  },

  /**
   * Docker 시스템 정보 조회
   */
  getSystemInfo: async (): Promise<{
    version: string
    apiVersion: string
    gitCommit: string
    goVersion: string
    os: string
    arch: string
    kernelVersion: string
    buildTime: string
    containers: {
      total: number
      running: number
      paused: number
      stopped: number
    }
    images: {
      total: number
      size: number
    }
    storage: {
      driver: string
      size: number
      available: number
    }
  }> => {
    const response = await apiGet('/docker/system/info')
    return response.data.data
  },

  /**
   * Docker 디스크 사용량 조회
   */
  getDiskUsage: async (): Promise<{
    containers: {
      active: number
      size: number
      reclaimable: number
    }
    images: {
      active: number
      size: number
      reclaimable: number
    }
    volumes: {
      active: number
      size: number
      reclaimable: number
    }
    buildCache: {
      size: number
      reclaimable: number
    }
  }> => {
    const response = await apiGet('/docker/system/df')
    return response.data.data
  },

  /**
   * Docker 시스템 정리
   */
  systemPrune: async (options?: {
    volumes?: boolean
    all?: boolean
    filters?: Record<string, string>
  }): Promise<{
    containersDeleted: string[]
    imagesDeleted: string[]
    volumesDeleted: string[]
    spaceReclaimed: number
  }> => {
    const response = await apiPost('/docker/system/prune', options)
    return response.data.data
  },
}