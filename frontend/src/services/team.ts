import request from '@/utils/request'
import type { ApiResponse } from '@/types'

// Team Types
export interface Team {
  id: number
  name: string
  project_id: number
  description?: string
  leader_name?: string
  created_at: string
  updated_at: string
}

export interface TeamSimple {
  id: number
  name: string
  project_id: number
}

export interface CreateTeamRequest {
  name: string
  project_id: number
  description?: string
  leader_name?: string
}

export interface UpdateTeamRequest extends Partial<CreateTeamRequest> {
  id: number
}

// Team Service
export const teamService = {
  // 创建团队
  create: (data: CreateTeamRequest) => {
    return request.post<ApiResponse<Team>>('/v1/team', data)
  },

  // 获取团队详情
  getById: (id: number) => {
    return request.get<ApiResponse<Team>>('/v1/team', {
      params: { id },
    })
  },

  // 获取所有团队（用于下拉选择）
  getList: (projectId?: number) => {
    return request.get<ApiResponse<TeamSimple[]>>('/v1/teams', {
      params: {project_id: projectId}
    })
  },

  // 更新团队
  update: (id: number, data: Partial<CreateTeamRequest>) => {
    return request.put<ApiResponse<Team>>('/v1/team', {
      id,
      ...data,
    })
  },

  // 删除团队
  delete: (id: number) => {
    return request.delete<ApiResponse<void>>(`/v1/team/${id}`)
  },
}

