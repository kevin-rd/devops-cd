import request from '@/utils/request'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types'

// Project Types
export interface Project {
  id: number
  name: string
  display_name?: string
  description?: string
  owner_name?: string
  created_at: string
  updated_at: string
}

export interface ProjectSimple {
  id: number
  name: string
  display_name?: string
}

export interface CreateProjectRequest {
  name: string
  display_name?: string
  description?: string
  owner_name?: string
}

export interface UpdateProjectRequest extends Partial<CreateProjectRequest> {
  id: number
}

export interface ProjectQueryParams extends PaginationParams {
  keyword?: string
}

// Project Service
export const projectService = {
  // 创建项目
  create: (data: CreateProjectRequest) => {
    return request.post<ApiResponse<Project>>('/api/v1/project', data)
  },

  // 获取项目详情
  getById: (id: number) => {
    return request.get<ApiResponse<Project>>('/api/v1/project/detail', {
      params: { id },
    })
  },

  // 获取项目列表
  getList: (params?: ProjectQueryParams) => {
    return request.get<ApiResponse<PaginatedResponse<Project>>>('/api/v1/projects', {
      params,
    })
  },

  // 获取所有项目（用于下拉选择）
  getAll: () => {
    return request.get<ApiResponse<ProjectSimple[]>>('/api/v1/projects/all')
  },

  // 更新项目
  update: (id: number, data: Partial<CreateProjectRequest>) => {
    return request.put<ApiResponse<Project>>('/api/v1/project', {
      id,
      ...data,
    })
  },

  // 删除项目
  delete: (id: number) => {
    return request.delete<ApiResponse<void>>(`/api/v1/project/${id}`)
  },
}

