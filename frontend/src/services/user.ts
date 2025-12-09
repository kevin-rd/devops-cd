import request from '@/utils/request'
import type {ApiResponse, PaginatedResponse, PaginationParams} from '@/types'

// User Types
export interface User {
  id: number
  username: string
  display_name?: string
  email?: string
  system_roles?: string[]
  created_at: string
  updated_at: string
}

export interface UserSimple {
  id: number
  username: string
  display_name?: string
  email?: string
}

export interface UserSearchParams extends PaginationParams {
  keyword?: string
}

// User Service
export const userService = {
  // 获取所有用户列表（用于下拉选择）
  // 注意：如果后端没有这个API，需要先添加
  getAll: () => {
    return request.get<ApiResponse<UserSimple[]>>('/v1/users')
  },

  // 按关键词搜索用户（keyword 可选，默认按 updated_at 排序由后端处理）
  search: (params: UserSearchParams) => {
    return request.get<ApiResponse<PaginatedResponse<UserSimple>>>('/v1/users/search', {
      params: {
        page_size: 20,
        ...params,
      },
    })
  },
}

