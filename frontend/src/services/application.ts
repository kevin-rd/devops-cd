import request from '@/utils/request'
import type {
  ApiResponse,
  Application,
  ApplicationQueryParams,
  ApplicationType,
  ApplicationWithBuild,
  CreateApplicationRequest,
  PaginatedResponse,
  UpdateApplicationRequest,
} from '@/types'

export const applicationService = {
  // 获取应用列表
  getList: (params?: ApplicationQueryParams) =>
    request.get<ApiResponse<PaginatedResponse<Application>>>('/v1/applications', {
      params,
    }),

  // 获取应用列表（包含最新构建信息）
  searchWithBuilds: (params?: ApplicationQueryParams, signal?: AbortSignal): Promise<ApiResponse<PaginatedResponse<ApplicationWithBuild>>> =>
    request.get<ApiResponse<PaginatedResponse<ApplicationWithBuild>>>('/v1/application_builds', {
      params,
      // skipErrorMessage: true,
      signal,
    }),

  // 获取应用详情
  getDetail: (id: number) =>
    request.get<ApiResponse<Application>>(`/v1/applications/${id}`),

  // 创建应用
  create: (data: CreateApplicationRequest) =>
    request.post<ApiResponse<Application>>('/v1/application', data),

  // 更新应用
  update: (id: number, data: UpdateApplicationRequest) =>
    request.put<ApiResponse<Application>>('/v1/application', {...data, id}),

  // 删除应用
  delete: (id: number) => request.delete<ApiResponse<null>>('/v1/application/delete', {params: {id}}),

  // 获取应用类型列表
  getTypes: (): Promise<ApiResponse<{ types: ApplicationType[]; total: number }>> =>
    request.get<ApiResponse<{ types: ApplicationType[]; total: number }>>('/v1/application/types'),
}

