import request from '@/utils/request';

export interface AppEnvConfig {
  id: number;
  app_id: number;
  env: string;
  cluster: string;
  replicas: number;
  deployment_name_override?: string;
  config_data?: string;
  status: number;
  created_at: string;
  updated_at: string;
}

export interface CreateAppEnvConfigRequest {
  app_id: number;
  env: string;
  cluster: string;
  replicas: number;
  deployment_name_override?: string;
  config_data?: string;
}

export interface UpdateAppEnvConfigRequest {
  id: number;
  cluster?: string;
  replicas?: number;
  deployment_name_override?: string;
  config_data?: string;
  status?: number;
}

export interface BatchCreateAppEnvConfigsRequest {
  app_id: number;
  configs: Array<{
    env: string;
    cluster: string;
    replicas: number;
    deployment_name_override?: string;
    config_data?: string;
  }>;
}

// 创建应用环境配置
export const createAppEnvConfig = (data: CreateAppEnvConfigRequest) => {
  return request.post<AppEnvConfig>('/app-env-configs', data);
};

// 查询应用环境配置列表
export const listAppEnvConfigs = (params: { app_id: number; env?: string }) => {
  return request.get<AppEnvConfig[]>('/app-env-configs', { params });
};

// 获取应用环境配置详情
export const getAppEnvConfig = (id: number) => {
  return request.get<AppEnvConfig>(`/app-env-configs/${id}`);
};

// 更新应用环境配置
export const updateAppEnvConfig = (id: number, data: UpdateAppEnvConfigRequest) => {
  return request.put<AppEnvConfig>(`/app-env-configs/${id}`, data);
};

// 删除应用环境配置
export const deleteAppEnvConfig = (id: number) => {
  return request.delete(`/app-env-configs/${id}`);
};

// 批量创建应用环境配置
export const batchCreateAppEnvConfigs = (data: BatchCreateAppEnvConfigsRequest) => {
  return request.post<AppEnvConfig[]>('/app-env-configs/batch', data);
};

