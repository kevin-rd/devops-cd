import { get, post, put, del } from '@/utils/request'
import type {
  ApiResponse,
  PaginatedResponse,
  Repository,
  CreateRepositoryRequest,
  UpdateRepositoryRequest,
  RepositoryQueryParams,
} from '@/types'

export const repositoryService = {
  // 获取代码库列表
  getList: (params?: RepositoryQueryParams) =>
    get<ApiResponse<PaginatedResponse<Repository>>>('/v1/repositories', {
      params,
    }),

  // 获取代码库详情
  getDetail: (id: number) =>
    get<ApiResponse<Repository>>(`/v1/repositories/${id}`),

  // 创建代码库
  create: (data: CreateRepositoryRequest) =>
    post<ApiResponse<Repository>>('/v1/repositories', data),

  // 更新代码库
  update: (id: number, data: UpdateRepositoryRequest) =>
    put<ApiResponse<Repository>>('/v1/repository', { ...data, id }),

  // 删除代码库
  delete: (id: number) => del<ApiResponse<null>>(`/v1/repositories/${id}`),
}

