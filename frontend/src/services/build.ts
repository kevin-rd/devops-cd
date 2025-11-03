import request from '@/utils/request'
import type {
  ApiResponse,
  PaginatedResponse,
  Build,
  BuildQueryParams,
} from '@/types'

/**
 * 构建记录服务
 */
export const buildService = {
  /**
   * 查询构建记录列表
   */
  getList: (params?: BuildQueryParams) => {
    return request.get<ApiResponse<PaginatedResponse<Build>>>('/v1/builds', {
      params,
    })
  },

  /**
   * 获取构建详情
   */
  getById: (id: number) => {
    return request.get<ApiResponse<Build>>('/v1/build', {
      params: { id },
    })
  },

  /**
   * 根据应用和构建号查询构建记录
   */
  getByAppAndNumber: (appId: number, buildNumber: number) => {
    return request.get<ApiResponse<Build>>('/v1/build/app', {
      params: {
        app_id: appId,
        build_number: buildNumber,
      },
    })
  },
}

export default buildService

