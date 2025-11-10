import { post } from '@/utils/request'
import type { ApiResponse } from '@/types'

export const releaseService = {
  // 切换版本 - 部署新版本
  switchVersion: (params: {
    release_app_id: number
    latest_build_id: number
    operator: string
  }) =>
    post<ApiResponse<{
      release_app_id: number
      build_id: number
      deployment_id: number
    }>>(
      `/v1/release_app/${params.release_app_id}/switch-version`,
      {
        latest_build_id: params.latest_build_id,
        operator: params.operator,
      }
    ),
}

