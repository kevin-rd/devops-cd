import { get, post, put, del } from '@/utils/request'
import type {
  ApiResponse,
  PaginatedResponse,
  Application,
  ApplicationWithBuild,
  CreateApplicationRequest,
  UpdateApplicationRequest,
  ApplicationQueryParams,
  ApplicationType,
} from '@/types'

export const applicationService = {
  // 获取应用列表
  getList: (params?: ApplicationQueryParams) =>
    get<ApiResponse<PaginatedResponse<Application>>>('/v1/applications', {
      params,
    }),

  // 获取应用列表（包含最新构建信息）
  searchWithBuilds: (params?: ApplicationQueryParams, signal?: AbortSignal) =>
    get<ApiResponse<PaginatedResponse<ApplicationWithBuild>>>('/v1/application_builds', {
      params,
      skipErrorMessage: true,
      signal,
    }),

  // 获取应用详情
  getDetail: (id: number) =>
    get<ApiResponse<Application>>(`/v1/applications/${id}`),

  // 创建应用
  create: (data: CreateApplicationRequest) =>
    post<ApiResponse<Application>>('/v1/application', data),

  // 更新应用
  update: (id: number, data: UpdateApplicationRequest) =>
    put<ApiResponse<Application>>('/v1/application', { ...data, id }),

  // 删除应用
  delete: (id: number) => del<ApiResponse<null>>('/v1/application/delete', { params: { id } }),

  // 获取应用类型列表
  getTypes: () =>
    get<ApiResponse<{ types: ApplicationType[]; total: number }>>('/v1/application/types'),
}

