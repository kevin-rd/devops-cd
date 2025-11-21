import request from '@/utils/request'
import type {ApiResponse, PaginatedResponse, PaginationParams} from '@/types'
import type {Team} from '@/services/team'

// Project Types
export interface Project {
  id: number
  name: string
  description?: string
  owner_name?: string
  created_at: string
  updated_at: string
  teams?: Team[]
}

export interface ProjectSimple {
  id: number
  name: string
}

export interface CreateProjectRequest {
  name: string
  description?: string
  owner_name?: string
  create_default_team?: boolean
}

export interface UpdateProjectRequest extends Partial<CreateProjectRequest> {
  id: number
}

export interface ProjectQueryParams extends PaginationParams {
  keyword?: string
  with_teams?: boolean
}

// Project Service
export const projectService = {
  // 创建项目
  create: (data: CreateProjectRequest) => {
    return request.post<ApiResponse<Project>>('/v1/project', data)
  },

  // 获取项目详情
  getById: (id: number) => {
    return request.get<ApiResponse<Project>>('/v1/project/detail', {
      params: {id},
    })
  },

  // 获取项目列表
  getList: (params?: ProjectQueryParams) => {
    return request.get<ApiResponse<PaginatedResponse<Project>>>('/v1/projects', {
      params,
    })
  },

  // 获取所有项目（用于下拉选择）
  getAll: () => {
    return request.get<any, ApiResponse<ProjectSimple[]>>('/v1/projects')
  },

  // 更新项目
  update: (id: number, data: Partial<CreateProjectRequest>) => {
    return request.put<ApiResponse<Project>>('/v1/project', {
      id,
      ...data,
    })
  },

  // 删除项目
  delete: (id: number) => {
    return request.delete<ApiResponse<void>>(`/v1/project/${id}`)
  },
}

