import request from '@/utils/request'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types'

export type RepoPlatform = 'gitea' | 'gitlab' | 'github'

export interface RepoSource {
  id: number
  platform: RepoPlatform
  base_url: string
  namespace: string
  enabled: boolean
  default_project_id?: number
  default_project_name?: string
  default_team_id?: number
  default_team_name?: string
  last_synced_at?: string
  last_status?: string
  last_message?: string
  created_at: string
  updated_at: string
  has_token: boolean
}

export interface RepoSourceQueryParams extends PaginationParams {
  keyword?: string
  platform?: RepoPlatform
  base_url?: string
  namespace?: string
  enabled?: boolean
}

export interface CreateRepoSourceRequest {
  platform: RepoPlatform
  base_url: string
  namespace: string
  token: string
  enabled?: boolean
  default_project_id?: number
  default_team_id?: number
}

export interface UpdateRepoSourceRequest extends Partial<CreateRepoSourceRequest> {
  id: number
}

export const repoSourceService = {
  getList: (params?: RepoSourceQueryParams) => {
    return request.get<ApiResponse<PaginatedResponse<RepoSource>>>('/v1/repo-sources', {
      params,
    })
  },

  create: (data: CreateRepoSourceRequest) => {
    return request.post<ApiResponse<RepoSource>>('/v1/repo-sources', data)
  },

  update: (data: UpdateRepoSourceRequest) => {
    return request.put<ApiResponse<RepoSource>>('/v1/repo-sources', data)
  },

  delete: (id: number) => {
    return request.delete<ApiResponse<void>>(`/v1/repo-sources/${id}`)
  },

  testConnection: (id: number) => {
    return request.post<ApiResponse<void>>(`/v1/repo-sources/${id}/test`)
  },

  syncNow: (id: number) => {
    return request.post<ApiResponse<{ success: number; failed: number }>>(
      `/v1/repo-sources/${id}/sync`
    )
  },
}

