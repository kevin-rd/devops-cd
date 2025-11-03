import { get, post, put, del } from '@/utils/request'
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

  // 获取批次详情（改为 GET 请求）
  get: (id: number) =>
    get<ApiResponse<Batch>>('/v1/batch', { params: { id } }),

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
}

