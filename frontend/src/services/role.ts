import request from '@/utils/request'
import type {ApiResponse} from '@/types'

export const roleService = {
  // 获取系统角色列表（无需按项目/团队区分）
  getAll: () => {
    return request.get<ApiResponse<string[]>>('/v1/roles')
  },
}

