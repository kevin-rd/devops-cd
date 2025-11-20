import React from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import MainLayout from '@/components/Layout'
import PrivateRoute from '@/components/PrivateRoute'
import Login from '@/pages/Login'
import RepositoryPage from '@/pages/Repository'
import RepoSourcesPage from '@/pages/RepoSources'
import ProjectPage from '@/pages/Project'
import BatchList from '@/pages/Batch'
import BatchDetail from '@/pages/Batch/Detail'
import BatchInsights from '@/pages/BatchInsights'
import BatchInsightsRedirect from '@/pages/BatchInsights/Redirect'

const AppRoutes: React.FC = () => {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        
        <Route element={<PrivateRoute />}>
          <Route element={<MainLayout />}>
            <Route path="/" element={<Navigate to="/repository" replace />} />
            <Route path="/repository" element={<RepositoryPage />} />
            <Route path="/project" element={<ProjectPage />} />
            <Route path="/batch" element={<BatchList />} />
            <Route path="/batch/:id/detail" element={<BatchDetail />} />
            <Route path="/batch/insights" element={<BatchInsightsRedirect />} />
            <Route path="/batch/:id/insights" element={<BatchInsights />} />
            <Route path="/repo-sources" element={<RepoSourcesPage />} />
          </Route>
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default AppRoutes

