import { get, post } from '@/utils/request'
import type {
  ApiResponse,
  LoginRequest,
  LoginResponse,
  UserInfo,
} from '@/types'

export const authService = {
  // 登录
  login: (data: LoginRequest) =>
    post<ApiResponse<LoginResponse>>('/v1/auth/login', data),

  // 登出
  logout: () => post<ApiResponse<null>>('/v1/auth/logout'),

  // 刷新 Token
  refreshToken: (refreshToken: string) =>
    post<ApiResponse<{ access_token: string; expires_in: number }>>(
      '/v1/auth/refresh',
      { refresh_token: refreshToken }
    ),

  // 获取当前用户信息
  getCurrentUser: () => get<ApiResponse<UserInfo>>('/v1/auth/me'),
}

