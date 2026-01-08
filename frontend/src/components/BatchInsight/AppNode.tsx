import {memo, useCallback, useMemo} from 'react'
import {Badge} from 'antd'
import {Handle, Position} from 'reactflow'
import type {ReleaseApp} from '@/types/release_app.ts'
import {AppStatus} from "@/types/release_app.ts";
import '@/styles/status-theme.css'
import styles from './AppNode.module.css'

interface AppNodeData {
  releaseApp: ReleaseApp
  isIsolated: boolean // 是否为游离节点
  onNodeClick?: (releaseApp: ReleaseApp) => void
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

const AppNode = memo(({data}: AppNodeProps) => {
  const {releaseApp, isIsolated, onNodeClick} = data
  const {id, app_name, app_type, status} = releaseApp

  // 获取应用类型颜色
  const ribbonColor = app_type ? (APP_TYPE_COLORS[app_type.toLowerCase()] || APP_TYPE_COLORS.default) : APP_TYPE_COLORS.default

  // 计算是否有新 tag（只要 latest_build_id 和 build_id 不相等就显示）
  const hasNewTag = useMemo(() => {
    if (!releaseApp.latest_build_id || !releaseApp.build_id) return false
    return releaseApp.latest_build_id !== releaseApp.build_id
  }, [releaseApp])

  // 根据状态获取样式类
  const getStatusClass = () => {
    if (!status) return ''

    // 转换为数字以便比较
    const statusNum = typeof status === 'string' ? parseInt(status, 10) : status
    if (isNaN(statusNum)) return ''

    // todo: 还需要添加预发布前的状态

    // 预发布等待 (20) - 虚线不滚动
    if (statusNum === AppStatus.PreWaiting) {
      return styles.preWaiting
    }
    // 预发布中 (21-22) - 虚线滚动
    if (statusNum >= AppStatus.PreCanTrigger && statusNum < AppStatus.PreDeployed) {
      return styles.preDeploying
    }
    // 预发布完成 (23)
    if (statusNum === AppStatus.PreDeployed) {
      return styles.preDeployed
    }
    // 预发布失败 (24)
    if (statusNum === AppStatus.PreFailed) {
      return styles.preFailed
    }
    // 预发布验收完成 (25)
    if (statusNum === AppStatus.PreAccepted) {
      return styles.preAccepted
    }
    // 生产等待 (30) - 虚线不滚动
    if (statusNum === AppStatus.ProdWaiting) {
      return styles.prodWaiting
    }
    // 生产部署中 (31-32) - 虚线滚动
    if (statusNum >= AppStatus.ProdCanTrigger && statusNum < AppStatus.ProdDeployed) {
      return styles.prodDeploying
    }
    // 生产部署完成 (33)
    if (statusNum === AppStatus.ProdDeployed) {
      return styles.prodDeployed
    }
    // 生产部署失败 (34)
    if (statusNum === AppStatus.ProdFailed) {
      return styles.prodFailed
    }
    // 生产验收完成 (35)
    if (statusNum === AppStatus.ProdAccepted) {
      return styles.prodAccepted
    }

    return ''
  }

  const handleClick = useCallback(() => {
    if (onNodeClick) {
      onNodeClick(releaseApp)
    }
  }, [releaseApp, onNodeClick])

  const statusClass = getStatusClass()

  const nodeContent = (
    <div className={`${styles.appNode} ${isIsolated ? styles.isolated : ''} ${statusClass}`}>
      {/* 隐藏的连接点 - 保留以支持边的连接，但不显示 */}
      <Handle
        type="target"
        position={Position.Top}
        className={styles.handle}
        isConnectable={false}
        style={{opacity: 0}}
      />

      <div className={styles.content} title={app_name}>
        <span style={{fontSize: 12, color: '#999'}}>#{id}</span>
        <span className={styles.appName}>{app_name}</span>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className={styles.handle}
        isConnectable={false}
        style={{opacity: 0}}
      />
    </div>
  )

  return (
    <Badge onClick={handleClick} style={{cursor: 'pointer'}}
           dot={hasNewTag}
           color="#ff4d4f"
           offset={[-8, 8]}
    >
      {app_type ? (
        <Badge.Ribbon text={app_type} color={ribbonColor} placement="start">{nodeContent}</Badge.Ribbon>
      ) : (
        nodeContent
      )}
    </Badge>
  )
})

AppNode.displayName = 'AppNode'

export default AppNode

