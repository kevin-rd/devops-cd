import React from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import MainLayout from '@/components/Layout'
import PrivateRoute from '@/components/PrivateRoute'
import Login from '@/pages/Login'
import RepositoryPage from '@/pages/Repository'
import BatchList from '@/pages/Batch'

const AppRoutes: React.FC = () => {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        
        <Route element={<PrivateRoute />}>
          <Route element={<MainLayout />}>
            <Route path="/" element={<Navigate to="/repository" replace />} />
            <Route path="/repository" element={<RepositoryPage />} />
            <Route path="/batch" element={<BatchList />} />
          </Route>
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default AppRoutes

