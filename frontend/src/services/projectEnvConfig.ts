import request from '@/utils/request'
import type {ApiResponse} from '@/types'

// ProjectEnvConfig Types
export interface ProjectEnvConfig {
  id: number
  project_id: number
  env: string
  allow_clusters: string[]
  default_clusters: string[]
  schema_version?: number
  artifacts_json?: unknown
  created_at: string
  updated_at: string
}

export interface ProjectEnvConfigRequest {
  allow_clusters: string[]
  default_clusters?: string[]
  schema_version?: number
  artifacts_json?: unknown
}

export interface UpdateProjectEnvConfigsRequest {
  configs: Record<string, ProjectEnvConfigRequest> // key: env (pre/prod)
}

// API Methods
export const projectEnvConfigService = {
  // 查询项目环境配置列表
  getList: async (projectId: number): Promise<ApiResponse<ProjectEnvConfig[]>> => {
    return request.get(`/v1/project/${projectId}/env`)
  },

  // 批量更新项目环境配置
  updateConfigs: async (
    projectId: number,
    data: UpdateProjectEnvConfigsRequest
  ): Promise<ApiResponse<void>> => {
    return request.put(`/v1/project/${projectId}/env`, data)
  },
}

