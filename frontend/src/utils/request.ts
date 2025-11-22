import axios, {AxiosError, AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig} from 'axios'
import {message} from 'antd'
import qs from 'qs'
import {tokenManager} from './token'

// 创建 axios 实例
const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 30000,
  // 自定义参数序列化器，将数组参数序列化为 status=1&status=2 格式（不带方括号）
  paramsSerializer: (params) => {
    return qs.stringify(params, {arrayFormat: 'repeat'})
  },
})

// 标记是否正在刷新 token
let isRefreshing = false
// 刷新失败次数
let refreshFailureCount = 0
// 最大重试次数
const MAX_REFRESH_RETRIES = 3
// 基础延迟时间（毫秒）
const BASE_DELAY = 1000

// 延迟函数
const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

// 存储等待重试的请求
let failedQueue: Array<{
  resolve: (value?: any) => void
  reject: (reason?: any) => void
}> = []

// 处理队列中的请求
const processQueue = (error: any = null) => {
  failedQueue.forEach((promise) => {
    if (error) {
      promise.reject(error)
    } else {
      promise.resolve()
    }
  })
  failedQueue = []
}

// 刷新 token 的函数（带重试和backoff机制）
const refreshAccessToken = async (retryCount: number = 0): Promise<string> => {
  const refreshToken = tokenManager.getRefreshToken()
  if (!refreshToken) {
    throw new Error('No refresh token available')
  }

  try {
    const response = await axios.post(
      `${import.meta.env.VITE_API_BASE_URL || '/api'}/v1/auth/refresh`,
      {refresh_token: refreshToken}
    )

    const {access_token, refresh_token, expires_in} = response.data.data
    tokenManager.setAccessToken(access_token, expires_in)
    // 如果返回了新的refresh_token，更新它
    if (refresh_token) {
      tokenManager.setRefreshToken(refresh_token)
    }

    // 刷新成功，重置失败计数
    refreshFailureCount = 0
    return access_token
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

// 请求拦截器
request.interceptors.request.use(
  async (config) => {
    // 检查 token 是否即将过期或已过期，如果是则自动刷新
    if (tokenManager.isAuthenticated()) {
      if (tokenManager.isTokenExpired() || tokenManager.isTokenExpiringSoon(5)) {
        // 如果正在刷新 token，等待刷新完成
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
            failedQueue.push({
              resolve: () => {
                const token = tokenManager.getAccessToken()
                if (token) {
                  config.headers.Authorization = `Bearer ${token}`
                }
                resolve(config)
              },
              reject: (err) => reject(err),
            })
          })
        }

        // 开始刷新 token
        isRefreshing = true
        try {
          await refreshAccessToken()
          // 刷新成功后，处理队列并更新当前请求的token
          processQueue()
          const token = tokenManager.getAccessToken()
          if (token) {
            config.headers.Authorization = `Bearer ${token}`
          }
        } catch (error) {
          // 刷新失败，清空队列
          processQueue(error)
          // 刷新失败时不清空token，让请求继续，响应拦截器会处理401错误
          console.error('Token refresh failed:', error)
        } finally {
          isRefreshing = false
        }
      }
    }

    // 添加 token
    const token = tokenManager.getAccessToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // 调试：打印请求信息
    if (config.url?.includes('batch')) {
    }

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response: AxiosResponse) => {
    const {data} = response
    const configWithMeta = response.config as AxiosRequestConfig & { skipErrorMessage?: boolean }
    const skipErrorMessage = configWithMeta.skipErrorMessage

    // 如果是直接返回的数据（非标准格式）
    if (response.config.responseType === 'blob') {
      return response
    }

    // 标准格式：{ code, message, data }
    if (data.code === 200 || data.code === 0) {
      return data
    }

    // 如果响应数据中的code是401（Token相关错误），需要刷新token
    if (data.code === 401) {
      const originalRequest = response.config as InternalAxiosRequestConfig & {
        _retry?: boolean,
        skipErrorMessage?: boolean
      }

      // 如果是刷新 token 的请求，直接返回错误
      if (originalRequest.url?.includes('/auth/refresh')) {
        message.error('登录已过期，请重新登录')
        tokenManager.clearTokens()
        refreshFailureCount = 0
        const currentPath = window.location.pathname + window.location.search
        if (currentPath !== '/login') {
          localStorage.setItem('redirect_path', currentPath)
        }
        window.location.href = '/login'
        return Promise.reject(new Error(data.message || 'Token解析失败'))
      }

      // 标记为已重试，防止无限循环
      if (!originalRequest._retry) {
        originalRequest._retry = true

        // 如果正在刷新 token，将请求加入队列
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
            failedQueue.push({resolve, reject})
          })
            .then(() => {
              // token 刷新成功，重试原请求
              originalRequest.headers.Authorization = `Bearer ${tokenManager.getAccessToken()}`
              return request(originalRequest)
            })
            .catch((err) => {
              return Promise.reject(err)
            })
        }

        // 开始刷新 token（带重试机制）
        isRefreshing = true

        return refreshAccessToken()
          .then((newAccessToken) => {
            // 更新请求头
            originalRequest.headers.Authorization = `Bearer ${newAccessToken}`

            // 处理队列中的请求
            processQueue()

            // 重试原请求（使用新的token）
            return request(originalRequest)
          })
          .catch((refreshError) => {
            // refreshAccessToken已经处理了重试，如果到这里说明所有重试都失败了
            processQueue(refreshError)
            message.error('登录已过期，请重新登录')

            // 保存当前路径
            const currentPath = window.location.pathname + window.location.search
            if (currentPath !== '/login') {
              localStorage.setItem('redirect_path', currentPath)
            }

            window.location.href = '/login'
            refreshFailureCount = 0
            return Promise.reject(refreshError)
          })
          .finally(() => {
            isRefreshing = false
          })
      }

      // 如果已经重试过但仍然失败，返回错误
      return Promise.reject(new Error(data.message || 'Token解析失败'))
    }

    // 业务错误
    if (!skipErrorMessage) {
      message.error(data.message || 'Request failed')
    }
    return Promise.reject(new Error(data.message || 'Request failed'))
  },
  async (error: AxiosError<any>) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean,
      skipErrorMessage?: boolean
    }

    // 调试：打印错误信息
    if (error.config?.url?.includes('batch')) {
      console.error('Batch API Error:', {
        url: error.config?.url,
        status: error.response?.status,
        data: error.response?.data,
        message: error.message,
      })
    }

    // 处理不同的错误状态
    if (error.response) {
      const {status, data: responseData} = error.response
      const skipErrorMessage = originalRequest?.skipErrorMessage

      // 检查是否是401错误（HTTP状态码401或响应数据中的code是401）
      const isTokenError = status === 401 || responseData?.code === 401

      // 处理 401 错误 - Token 过期或解析失败
      if (isTokenError && !originalRequest._retry) {
        // 如果是刷新 token 的请求失败（已经重试过），直接跳转登录
        if (originalRequest.url?.includes('/auth/refresh')) {
          // refreshAccessToken函数已经处理了重试，如果到这里说明所有重试都失败了
          message.error('登录已过期，请重新登录')
          tokenManager.clearTokens()
          refreshFailureCount = 0
          // 保存当前路径
          const currentPath = window.location.pathname + window.location.search
          if (currentPath !== '/login') {
            localStorage.setItem('redirect_path', currentPath)
          }
          window.location.href = '/login'
          return Promise.reject(error)
        }

        // 标记为已重试，防止无限循环
        originalRequest._retry = true

        // 如果正在刷新 token，将请求加入队列
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
            failedQueue.push({resolve, reject})
          })
            .then(() => {
              // token 刷新成功，重试原请求
              originalRequest.headers.Authorization = `Bearer ${tokenManager.getAccessToken()}`
              return request(originalRequest)
            })
            .catch((err) => {
              return Promise.reject(err)
            })
        }

        // 开始刷新 token（带重试机制）
        isRefreshing = true

        try {
          const newAccessToken = await refreshAccessToken()

          // 更新请求头
          originalRequest.headers.Authorization = `Bearer ${newAccessToken}`

          // 处理队列中的请求
          processQueue()

          // 重试原请求（使用新的token）
          return request(originalRequest)
        } catch (refreshError) {
          // refreshAccessToken已经处理了重试，如果到这里说明所有重试都失败了
          processQueue(refreshError)
          message.error('登录已过期，请重新登录')

          // 保存当前路径
          const currentPath = window.location.pathname + window.location.search
          if (currentPath !== '/login') {
            localStorage.setItem('redirect_path', currentPath)
          }

          window.location.href = '/login'
          refreshFailureCount = 0
          return Promise.reject(refreshError)
        } finally {
          isRefreshing = false
        }
      }

      // 处理其他错误状态
      switch (status) {
        case 403:
          if (!skipErrorMessage) {
            message.error('访问被拒绝')
          }
          break
        default:
          if (!skipErrorMessage && responseData?.message) {
            message.error(responseData.message)
          }
      }
    } else if (error.request) {
      // 网络错误在拦截器中显示
      if (!(originalRequest && originalRequest.skipErrorMessage)) {
        message.error('网络错误，请检查您的连接')
      }
    }

    return Promise.reject(error)
  }
)

export default request



