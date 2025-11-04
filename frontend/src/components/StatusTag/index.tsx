import { Tag } from 'antd'
import type { Batch } from '@/types'

interface StatusTagProps {
  status: number
  approvalStatus?: string
  showApproval?: boolean
}

// 批次状态配置
const BATCH_STATUS_CONFIG: Record<number, { label: string; color: string }> = {
  0: { label: '草稿', color: '#fadb14' },
  10: { label: '已封板', color: 'purple' },
  20: { label: '预发布已触发', color: 'blue' },
  21: { label: '预发布中', color: 'processing' },
  22: { label: '预发布完成', color: 'success' },
  30: { label: '生产已触发', color: 'blue' },
  31: { label: '生产部署中', color: 'warning' },
  32: { label: '生产部署完成', color: 'success' },
  40: { label: '已完成', color: 'success' },
  90: { label: '已取消', color: 'default' },
}

// 审批状态配置
const APPROVAL_STATUS_CONFIG: Record<string, { label: string; color: string }> = {
  pending: { label: '待审批', color: 'blue' },
  approved: { label: '已审批', color: 'success' },
  rejected: { label: '已拒绝', color: 'error' },
  skipped: { label: '已跳过', color: 'default' },
}

export const StatusTag: React.FC<StatusTagProps> = ({ status, approvalStatus, showApproval = false }) => {
  const statusConfig = BATCH_STATUS_CONFIG[status] || { label: '未知', color: 'default' }
  
  if (showApproval && approvalStatus) {
    const approvalConfig = APPROVAL_STATUS_CONFIG[approvalStatus] || { label: approvalStatus, color: 'default' }
    return <Tag color={approvalConfig.color}>{approvalConfig.label}</Tag>
  }
  
  return <Tag color={statusConfig.color}>{statusConfig.label}</Tag>
}

// 导出辅助函数供其他地方使用
export const getBatchStatusText = (status: number): string => {
  return BATCH_STATUS_CONFIG[status]?.label || '未知'
}

export const getApprovalStatusText = (approvalStatus: string): string => {
  return APPROVAL_STATUS_CONFIG[approvalStatus]?.label || approvalStatus
}

