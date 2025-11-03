import { useEffect, useRef } from 'react'
import { tokenManager } from '@/utils/token'
import axios from 'axios'

// 刷新失败次数
let refreshFailureCount = 0
// 最大重试次数
const MAX_REFRESH_RETRIES = 3
// 基础延迟时间（毫秒）
const BASE_DELAY = 1000

// 延迟函数
const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

// 刷新 token 的函数（带重试和backoff机制）
const refreshAccessToken = async (retryCount: number = 0): Promise<void> => {
  const refreshToken = tokenManager.getRefreshToken()
  if (!refreshToken) {
    throw new Error('No refresh token available')
  }

  try {
    const response = await axios.post(
      `${import.meta.env.VITE_API_BASE_URL || '/api'}/v1/auth/refresh`,
      { refresh_token: refreshToken }
    )

    const { access_token, refresh_token, expires_in } = response.data.data
    tokenManager.setAccessToken(access_token, expires_in)
    // 如果返回了新的refresh_token，更新它
    if (refresh_token) {
      tokenManager.setRefreshToken(refresh_token)
    }
    
    // 刷新成功，重置失败计数
    refreshFailureCount = 0
  } catch (error) {
    // 如果还有重试次数，使用指数退避策略重试
    if (retryCount < MAX_REFRESH_RETRIES) {
      const delayMs = BASE_DELAY * Math.pow(2, retryCount) // 指数退避：1s, 2s, 4s
      console.log(`Token refresh failed, retrying in ${delayMs}ms (attempt ${retryCount + 1}/${MAX_REFRESH_RETRIES})`)
      await delay(delayMs)
      return refreshAccessToken(retryCount + 1)
    }
    
    // 达到最大重试次数，清空token并抛出错误
    refreshFailureCount++
    tokenManager.clearTokens()
    throw error
  }
}

/**
 * Token 自动刷新 Hook
 * 定期检查 token 是否即将过期，如果是则自动刷新
 */
export const useTokenRefresh = () => {
  const intervalRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    // 检查函数
    const checkAndRefreshToken = async () => {
      if (!tokenManager.isAuthenticated()) {
        return
      }

      // 如果 token 即将过期（提前5分钟）或已过期，则刷新
      if (tokenManager.isTokenExpired() || tokenManager.isTokenExpiringSoon(5)) {
        try {
          await refreshAccessToken()
          console.log('Token refreshed successfully')
        } catch (error) {
          // refreshAccessToken已经处理了重试，如果到这里说明所有重试都失败了
          console.error('Failed to refresh token after retries:', error)
          // 刷新失败，清空token并跳转登录
          tokenManager.clearTokens()
          refreshFailureCount = 0
          const currentPath = window.location.pathname + window.location.search
          if (currentPath !== '/login') {
            localStorage.setItem('redirect_path', currentPath)
            window.location.href = '/login'
          }
        }
      }
    }

    // 立即检查一次
    checkAndRefreshToken()

    // 每60秒检查一次（可以根据需要调整）
    intervalRef.current = setInterval(checkAndRefreshToken, 60 * 1000)

    // 清理函数
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
      }
    }
  }, [])
}

