import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { UserInfo } from '@/types'
import { tokenManager } from '@/utils/token'

interface AuthState {
  user: UserInfo | null
  isAuthenticated: boolean
  setUser: (user: UserInfo | null) => void
  login: (user: UserInfo, accessToken: string, refreshToken: string, expiresIn?: number) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: tokenManager.getUserInfo(),
      isAuthenticated: tokenManager.isAuthenticated(),

      setUser: (user) => set({ user, isAuthenticated: !!user }),

      login: (user, accessToken, refreshToken, expiresIn) => {
        tokenManager.setAccessToken(accessToken, expiresIn)
        tokenManager.setRefreshToken(refreshToken)
        tokenManager.setUserInfo(user)
        set({ user, isAuthenticated: true })
      },

      logout: () => {
        tokenManager.clearTokens()
        set({ user: null, isAuthenticated: false })
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)

