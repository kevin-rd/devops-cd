// Token 管理工具

const ACCESS_TOKEN_KEY = 'access_token'
const REFRESH_TOKEN_KEY = 'refresh_token'
const TOKEN_EXPIRES_KEY = 'token_expires_at'
const USER_INFO_KEY = 'user_info'

export const tokenManager = {
  // 获取 Access Token
  getAccessToken(): string | null {
    return localStorage.getItem(ACCESS_TOKEN_KEY)
  },

  // 设置 Access Token（包含过期时间）
  setAccessToken(token: string, expiresIn?: number): void {
    localStorage.setItem(ACCESS_TOKEN_KEY, token)
    if (expiresIn) {
      // expiresIn 是秒数，转换为过期时间戳（毫秒）
      const expiresAt = Date.now() + expiresIn * 1000
      localStorage.setItem(TOKEN_EXPIRES_KEY, expiresAt.toString())
    }
  },

  // 获取 Token 过期时间
  getTokenExpiresAt(): number | null {
    const expiresAt = localStorage.getItem(TOKEN_EXPIRES_KEY)
    return expiresAt ? parseInt(expiresAt, 10) : null
  },

  // 检查 Token 是否即将过期（提前5分钟刷新）
  isTokenExpiringSoon(refreshBeforeMinutes: number = 5): boolean {
    const expiresAt = this.getTokenExpiresAt()
    if (!expiresAt) {
      return false
    }
    const now = Date.now()
    const refreshBefore = refreshBeforeMinutes * 60 * 1000 // 转换为毫秒
    return expiresAt - now <= refreshBefore
  },

  // 检查 Token 是否已过期
  isTokenExpired(): boolean {
    const expiresAt = this.getTokenExpiresAt()
    if (!expiresAt) {
      return false
    }
    return Date.now() >= expiresAt
  },

  // 获取 Refresh Token
  getRefreshToken(): string | null {
    return localStorage.getItem(REFRESH_TOKEN_KEY)
  },

  // 设置 Refresh Token
  setRefreshToken(token: string): void {
    localStorage.setItem(REFRESH_TOKEN_KEY, token)
  },

  // 清除所有 Token
  clearTokens(): void {
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
    localStorage.removeItem(TOKEN_EXPIRES_KEY)
    localStorage.removeItem(USER_INFO_KEY)
  },

  // 检查是否已登录
  isAuthenticated(): boolean {
    return !!this.getAccessToken()
  },

  // 保存用户信息
  setUserInfo(userInfo: any): void {
    localStorage.setItem(USER_INFO_KEY, JSON.stringify(userInfo))
  },

  // 获取用户信息
  getUserInfo(): any {
    const info = localStorage.getItem(USER_INFO_KEY)
    return info ? JSON.parse(info) : null
  },
}

