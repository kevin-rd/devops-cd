import request from '../utils/request';
import { ApiResponse, PaginatedResponse } from '../types';

/**
 * 集群接口
 */

// 集群类型定义
export interface Cluster {
  id: number;
  name: string;
  description?: string;
  region?: string;
  status: number;
  created_at: string;
  updated_at: string;
}

// 创建集群请求
export interface CreateClusterRequest {
  name: string;
  description?: string;
  region?: string;
}

// 更新集群请求
export interface UpdateClusterRequest {
  name?: string;
  description?: string;
  region?: string;
  status?: number;
}

// 集群列表请求
export interface ClusterListRequest {
  name?: string;
  status?: number;
  page?: number;
  page_size?: number;
}

/**
 * 获取集群列表
 */
export const getClusters = (params?: ClusterListRequest) => {
  return request.get<ApiResponse<PaginatedResponse<Cluster>>>('/v1/clusters', { params });
};

/**
 * 获取集群详情
 */
export const getCluster = (id: number) => {
  return request.get<ApiResponse<Cluster>>(`/v1/clusters/${id}`);
};

/**
 * 创建集群
 */
export const createCluster = (data: CreateClusterRequest) => {
  return request.post<ApiResponse<Cluster>>('/v1/clusters', data);
};

/**
 * 更新集群
 */
export const updateCluster = (id: number, data: UpdateClusterRequest) => {
  return request.put<ApiResponse<Cluster>>(`/v1/clusters/${id}`, data);
};

/**
 * 删除集群
 */
export const deleteCluster = (id: number) => {
  return request.delete<ApiResponse<void>>(`/v1/clusters/${id}`);
};

