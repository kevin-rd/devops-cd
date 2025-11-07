import { memo } from 'react'
import { Handle, Position } from 'reactflow'
import type { ReleaseApp } from '@/types'
import '@/styles/status-theme.css'
import styles from './AppNode.module.css'

interface AppNodeData {
  releaseApp: ReleaseApp
  isIsolated: boolean // 是否为游离节点
}

interface AppNodeProps {
  data: AppNodeData
}

// 应用类型颜色映射
const APP_TYPE_COLORS: Record<string, string> = {
  java: '#f56a00',
  go: '#00a5a8',
  python: '#3776ab',
  node: '#68a063',
  static: '#7265e6',
  default: '#1890ff',
}

const AppNode = memo(({ data }: AppNodeProps) => {
  const { releaseApp, isIsolated } = data
  const { app_name, app_type, status } = releaseApp

  // 获取应用类型颜色
  const ribbonColor = app_type ? (APP_TYPE_COLORS[app_type.toLowerCase()] || APP_TYPE_COLORS.default) : APP_TYPE_COLORS.default

  // 根据状态获取样式类
  const getStatusClass = () => {
    if (!status) return ''
    
    // 转换为数字以便比较
    const statusNum = typeof status === 'string' ? parseInt(status, 10) : status
    if (isNaN(statusNum)) return ''
    
    // 预发布等待 (10) - 虚线不滚动
    if (statusNum === 10) {
      return styles.preWaiting
    }
    // 预发布中 (11-12) - 虚线滚动
    if (statusNum >= 11 && statusNum <= 12) {
      return styles.preDeploying
    }
    // 预发布完成 (13)
    if (statusNum === 13) {
      return styles.preDeployed
    }
    // 预发布失败 (14)
    if (statusNum === 14) {
      return styles.preFailed
    }
    // 生产等待 (20) - 虚线不滚动
    if (statusNum === 20) {
      return styles.prodWaiting
    }
    // 生产部署中 (21-22) - 虚线滚动
    if (statusNum >= 21 && statusNum <= 22) {
      return styles.prodDeploying
    }
    // 生产部署完成 (23)
    if (statusNum === 23) {
      return styles.prodDeployed
    }
    // 生产部署失败 (24)
    if (statusNum === 24) {
      return styles.prodFailed
    }
    
    return ''
  }

  const statusClass = getStatusClass()

  return (
    <div className={`${styles.appNode} ${isIsolated ? styles.isolated : ''} ${statusClass}`}>
      {/* 隐藏的连接点 - 保留以支持边的连接，但不显示 */}
      <Handle
        type="target"
        position={Position.Top}
        className={styles.handle}
        isConnectable={false}
        style={{ opacity: 0 }}
      />
      
      {/* 缎带徽标 */}
      {app_type && (
        <div className={styles.ribbon} style={{ backgroundColor: ribbonColor }}>
          {app_type}
        </div>
      )}
      
      <div className={styles.content}>
        <div className={styles.appName} title={app_name}>{app_name}</div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className={styles.handle}
        isConnectable={false}
        style={{ opacity: 0 }}
      />
    </div>
  )
})

AppNode.displayName = 'AppNode'

export default AppNode

