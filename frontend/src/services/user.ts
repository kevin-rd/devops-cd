import request from '@/utils/request'
import type {ApiResponse} from '@/types'

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

// User Service
export const userService = {
  // 获取所有用户列表（用于下拉选择）
  // 注意：如果后端没有这个API，需要先添加
  getAll: () => {
    return request.get<ApiResponse<UserSimple[]>>('/v1/users')
  },
}

