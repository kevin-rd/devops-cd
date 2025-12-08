import {Tag} from 'antd'

interface StatusTagProps {
  status: number
  approvalStatus?: string
  showApproval?: boolean
  onApprovalClick?: () => void
  approvalTime?: string
  rejectReason?: string
  approvedBy?: string
}

// 批次状态配置
export const BATCH_STATUS_CONFIG: Record<number, { label: string; color: string }> = {
  0: {label: '草稿', color: 'yellow'},
  10: {label: '已封板', color: 'purple'},
  20: {label: 'Pre已触发', color: 'blue'},
  21: {label: 'Pre进行中', color: 'processing'},
  22: {label: 'Pre部署完成', color: 'success'},
  30: {label: 'Prod已触发', color: 'blue'},
  31: {label: 'Prod进行中', color: 'warning'},
  32: {label: 'Prod部署完成', color: 'success'},
  40: {label: '已完成', color: 'success'},
  90: {label: '已取消', color: 'default'},
}

// 审批状态配置
const APPROVAL_STATUS_CONFIG: Record<string, { label: string; color: string }> = {
  pending: {label: '待审批', color: 'blue'},
  approved: {label: '已审批', color: 'success'},
  rejected: {label: '已拒绝', color: 'error'},
  skipped: {label: '已跳过', color: 'default'},
}

export const StatusTag: React.FC<StatusTagProps> = ({
                                                      status,
                                                      approvalStatus,
                                                      showApproval = false,
                                                      onApprovalClick,
                                                      approvalTime,
                                                      rejectReason,
                                                      approvedBy
                                                    }) => {
  const statusConfig = BATCH_STATUS_CONFIG[status] || {label: '未知', color: 'default'}

  if (showApproval && approvalStatus) {
    const approvalConfig = APPROVAL_STATUS_CONFIG[approvalStatus] || {label: approvalStatus, color: 'default'}
    // 预发布开始后（status >= 20）不能再操作审批
    const isClickable = status < 20 && onApprovalClick

    // 构建下面一行的内容
    const bottomLineContent: string[] = []
    if (approvedBy && rejectReason) {
      bottomLineContent.push(`${approvedBy}: ${rejectReason}`)
    } else if (approvedBy) {
      bottomLineContent.push(`审批人: ${approvedBy}`)
    } else if (rejectReason) {
      bottomLineContent.push(`原因: ${rejectReason}`)
    }
    const bottomLineText = bottomLineContent.join(' ')

    return (
      <div style={{display: 'flex', flexDirection: 'column', gap: '4px'}}>
        <div style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
          <Tag
            color={approvalConfig.color}
            onClick={isClickable ? (e) => {
              e.stopPropagation()
              onApprovalClick()
            } : undefined}
            style={isClickable ? {cursor: 'pointer'} : undefined}
          >
            {approvalConfig.label}
          </Tag>
          {approvalTime && (
            <div style={{fontSize: '12px', color: '#8c8c8c'}}>
              {approvalTime}
            </div>
          )}
        </div>
        {bottomLineText && (
          <div style={{fontSize: '12px', color: '#8c8c8c', wordBreak: 'break-word'}}>
            {bottomLineText}
          </div>
        )}
      </div>
    )
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

