import React from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'

const PrivateRoute: React.FC = () => {
  const { isAuthenticated } = useAuthStore()
  const location = useLocation()

  if (!isAuthenticated) {
    // 保存当前尝试访问的路径
    const currentPath = location.pathname + location.search
    localStorage.setItem('redirect_path', currentPath)
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}

export default PrivateRoute

