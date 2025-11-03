import React from 'react'
import { ConfigProvider, App as AntApp } from 'antd'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import zhCN from 'antd/locale/zh_CN'
import enUS from 'antd/locale/en_US'
import { useTranslation } from 'react-i18next'
import { useTokenRefresh } from '@/hooks/useTokenRefresh'
import AppRoutes from '@/routes'
import './App.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

const App: React.FC = () => {
  const { i18n } = useTranslation()
  
  // 启用 token 自动刷新
  useTokenRefresh()

  return (
    <ConfigProvider
      locale={i18n.language === 'zh' ? zhCN : enUS}
      theme={{
        token: {
          colorPrimary: '#667eea',
          borderRadius: 8,
        },
        components: {
          Button: {
            borderRadius: 8,
          },
          Input: {
            borderRadius: 8,
          },
          Card: {
            borderRadius: 12,
          },
        },
      }}
    >
      <AntApp>
        <QueryClientProvider client={queryClient}>
          <AppRoutes />
        </QueryClientProvider>
      </AntApp>
    </ConfigProvider>
  )
}

export default App

