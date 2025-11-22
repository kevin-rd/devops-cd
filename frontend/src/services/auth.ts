import request from '@/utils/request'
import type {ApiResponse, LoginRequest, LoginResponse, UserInfo,} from '@/types'

export const authService = {
  // 登录
  login: (data: LoginRequest) =>
    request.post<ApiResponse<LoginResponse>>('/v1/auth/login', data),

  // 登出
  logout: () => request.post<ApiResponse<null>>('/v1/auth/logout'),

  // 刷新 Token
  refreshToken: (refreshToken: string) =>
    request.post<ApiResponse<{ access_token: string; expires_in: number }>>(
      '/v1/auth/refresh',
      {refresh_token: refreshToken}
    ),

  // 获取当前用户信息
  getCurrentUser: () => request.get<ApiResponse<UserInfo>>('/v1/auth/me'),
}

