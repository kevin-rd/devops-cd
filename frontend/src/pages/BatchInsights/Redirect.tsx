import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Spin } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { batchService } from '@/services/batch'
import type { Batch } from '@/types'

/**
 * 重定向组件：查询最新批次并跳转到其详情页
 */
export default function BatchInsightsRedirect() {
  const navigate = useNavigate()

  const { data, isLoading, isError } = useQuery({
    queryKey: ['batch-latest'],
    queryFn: async () => {
      const res = await batchService.list({ page: 1, page_size: 1 })
      const raw = res.data as any
      let items: Batch[] = []
      
      if (Array.isArray(raw)) {
        items = raw
      } else if (raw && Array.isArray(raw.items)) {
        items = raw.items
      }
      
      return items.length > 0 ? items[0] : null
    },
    staleTime: 10 * 1000, // 10秒缓存
  })

  useEffect(() => {
    if (data) {
      // 找到最新批次，跳转到详情页
      navigate(`/batch/${data.id}/insights`, { replace: true })
    } else if (isError || (!isLoading && !data)) {
      // 查询失败或没有批次，跳转到批次列表
      navigate('/batch', { replace: true })
    }
  }, [data, isError, isLoading, navigate])

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      height: '100vh',
    }}>
      <Spin size="large" tip="加载最新批次..." />
    </div>
  )
}

