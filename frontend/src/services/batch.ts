import { get, post, put } from '@/utils/request'
import type {
  ApiResponse,
  PaginatedResponse,
  Batch,
  CreateBatchRequest,
  UpdateBatchRequest,
  BatchQueryParams,
  BatchActionRequest,
  BatchApproveRequest,
  BatchRejectRequest,
	UpdateReleaseDependenciesRequest,
	ReleaseDependenciesResponse,
} from '@/types'

export const batchService = {
  // 创建批次
  create: (data: CreateBatchRequest) =>
    post<ApiResponse<{ batch_id: number; batch_number: string; message: string }>>(
      '/v1/batch',
      data
    ),

  // 获取批次列表（改为 GET 请求）
  list: (params?: BatchQueryParams) =>
    get<ApiResponse<PaginatedResponse<Batch>>>('/v1/batches', { params }),

  // 获取批次详情（改为 GET 请求，支持应用列表分页）
  get: (id: number, appPage?: number, appPageSize?: number) =>
    get<ApiResponse<Batch>>('/v1/batch', { 
      params: { 
        id,
        app_page: appPage,
        app_page_size: appPageSize
      } 
    }),

  // 获取批次状态（轻量级接口，用于状态轮询）
  getStatus: (id: number, appPage?: number, appPageSize?: number) =>
    get<ApiResponse<Batch>>('/v1/batch/status', { 
      params: { 
        id,
        app_page: appPage,
        app_page_size: appPageSize
      } 
    }),

  // 更新批次
  update: (data: UpdateBatchRequest) =>
    put<ApiResponse<{ message: string }>>('/v1/batch', data),

  // 审批通过
  approve: (data: BatchApproveRequest) =>
    post<ApiResponse<{ message: string }>>('/v1/batch/approve', data),

  // 审批拒绝
  reject: (data: BatchRejectRequest) =>
    post<ApiResponse<{ message: string }>>('/v1/batch/reject', data),

  // 批次操作（封板、部署、验收等）
  action: (data: BatchActionRequest) =>
    post<ApiResponse<{ message: string }>>('/v1/batch/action', data),

  // 更新批次发布应用（构建版本等）
  updateBuilds: (data: { batch_id: number; operator: string; build_changes: Record<number, number> }) =>
    put<ApiResponse<{ message: string; batch_id: number; update_count: number }>>('/v1/batch/release_app', data),

	// 更新发布应用临时依赖
	updateDependencies: (releaseAppId: number, data: UpdateReleaseDependenciesRequest) =>
		put<ApiResponse<ReleaseDependenciesResponse>>(`/v1/release_app/${releaseAppId}/dependencies`, data),
}

