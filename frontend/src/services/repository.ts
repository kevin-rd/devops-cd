import  request from '@/utils/request'
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
    request.get<ApiResponse<PaginatedResponse<Repository>>>('/v1/repositories', {
      params,
    }),

  // 获取代码库详情
  getDetail: (id: number) =>
    request.get<ApiResponse<Repository>>(`/v1/repositories/${id}`),

  // 创建代码库
  create: (data: CreateRepositoryRequest) =>
    request.post<ApiResponse<Repository>>('/v1/repositories', data),

  // 更新代码库
  update: (id: number, data: UpdateRepositoryRequest) =>
    request.put<ApiResponse<Repository>>('/v1/repository', { ...data, id }),

  // 删除代码库
  delete: (id: number) => request.delete<ApiResponse<null>>(`/v1/repositories/${id}`),
}

