import { memo } from 'react'
import { Handle, Position } from 'reactflow'
import type { ReleaseApp } from '@/types'
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
  const { app_name, app_type } = releaseApp

  // 获取应用类型颜色
  const ribbonColor = app_type ? (APP_TYPE_COLORS[app_type.toLowerCase()] || APP_TYPE_COLORS.default) : APP_TYPE_COLORS.default

  return (
    <div className={`${styles.appNode} ${isIsolated ? styles.isolated : ''}`}>
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

