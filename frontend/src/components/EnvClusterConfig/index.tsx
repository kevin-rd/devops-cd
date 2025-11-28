import React, {useEffect, useState} from 'react'
import {Alert, Checkbox, Empty, Spin} from 'antd'
import {useQuery} from '@tanstack/react-query'
import type {Cluster} from '@/services/cluster'
import {getClusters} from '@/services/cluster'
import type {Project} from '@/services/project'
import {projectService} from '@/services/project'
import './index.css'

interface EnvClusterConfigProps {
  value?: Record<string, string[]>
  onChange?: (value: Record<string, string[]>) => void
  projectId?: number  // 项目ID，用于获取允许的环境集群配置
  project?: Project  // 新增：直接传入项目详情，避免重复查询
  allowedOptions?: Record<string, string[]>  // 限制可选的环境和集群（用于 default_env_clusters）
}

// 只支持 pre 和 prod 两个环境
const ENVS = [
  {key: 'pre', label: 'Pre'},
  {key: 'prod', label: 'Prod'},
]

const EnvClusterConfig: React.FC<EnvClusterConfigProps> = ({
  value = {}, 
  onChange, 
  projectId, 
  project,  // 新增：直接传入的项目详情
  allowedOptions
}) => {
  const [config, setConfig] = useState<Record<string, string[]>>(value)

  // 获取集群列表（获取所有启用的集群，传大的 page_size）
  const {data: clustersResponse, isLoading: clustersLoading} = useQuery({
    queryKey: ['clusters'],
    queryFn: async () => {
      return await getClusters({status: 1, page: 1, page_size: 1000})
    },
  })

  // 获取项目允许的环境集群配置（仅在没有传入 project 时查询）
  const {data: projectEnvClustersResponse, isLoading: projectLoading} = useQuery({
    queryKey: ['project-env-clusters', projectId],
    queryFn: async () => {
      if (!projectId || project) return null  // 如果已经有 project 数据，不查询
      const res = await projectService.getAvailableEnvClusters(projectId)
      return res.data
    },
    enabled: !!projectId && !project,  // 只有在没有 project 时才启用
  })

  // 后端返回格式: {code, message, data: {items: [...], total, page, page_size}}
  const allClusters: Cluster[] = clustersResponse?.data?.items || []
  
  // 项目允许的环境集群配置
  // 优先级：allowedOptions > 传入的 project.allowed_env_clusters > API 查询结果
  const allowedEnvClusters = 
    allowedOptions || 
    project?.allowed_env_clusters || 
    projectEnvClustersResponse?.allowed_env_clusters || 
    {}

  useEffect(() => {
    setConfig(value || {})
  }, [value])

  // 检查某个环境的某个集群是否被选中
  const isChecked = (env: string, clusterName: string) => {
    return config[env]?.includes(clusterName) || false
  }

  // 切换选中状态
  const handleToggle = (env: string, clusterName: string, checked: boolean) => {
    const newConfig = {...config}

    if (checked) {
      // 添加集群
      if (!newConfig[env]) {
        newConfig[env] = []
      }
      if (!newConfig[env].includes(clusterName)) {
        newConfig[env] = [...newConfig[env], clusterName]
      }
    } else {
      // 移除集群
      if (newConfig[env]) {
        newConfig[env] = newConfig[env].filter(c => c !== clusterName)
        // 如果该环境没有集群了,删除该环境
        if (newConfig[env].length === 0) {
          delete newConfig[env]
        }
      }
    }

    setConfig(newConfig)
    onChange?.(newConfig)
  }

  // 获取某个环境允许的集群列表
  const getAvailableClusters = (env: string) => {
    if (!projectId && !project && !allowedOptions) return allClusters  // 如果没有限制，返回所有集群
    const allowedClusterNames = allowedEnvClusters[env] || []
    return allClusters.filter(cluster => allowedClusterNames.includes(cluster.name))
  }

  // 检查某个环境是否被允许
  const isEnvAllowed = (env: string) => {
    if (!projectId && !project && !allowedOptions) return true  // 如果没有限制，允许所有环境
    return env in allowedEnvClusters
  }

  const isLoading = clustersLoading || (!project && projectLoading)

  if (isLoading) {
    return <Spin/>
  }

  // 如果有限制条件但没有配置allowed_env_clusters
  if ((projectId || project || allowedOptions) && Object.keys(allowedEnvClusters).length === 0) {
    return (
      <Alert
        message={allowedOptions ? "请先配置允许的环境集群" : "项目未配置环境集群"}
        description={allowedOptions ? "请先在上方'允许的环境和集群'中配置可用的环境和集群。" : "该项目尚未配置允许的环境和集群，请先在项目管理中配置。"}
        type="warning"
        showIcon
      />
    )
  }

  if (allClusters.length === 0) {
    return (
      <Empty
        description="暂无可用集群,请先在集群管理中创建集群"
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      />
    )
  }

  return (
    <div className="env-cluster-table">
      {/* 固定的第一列 - 环境标签 */}
      <div className="fixed-column">
        {ENVS.filter(env => isEnvAllowed(env.key)).map(env => (
          <div key={env.key} className="env-cell">
            {env.label}:
          </div>
        ))}
      </div>

      {/* 可滚动的集群列 */}
      <div className="scrollable-columns">
        {ENVS.filter(env => isEnvAllowed(env.key)).map(env => {
          const availableClusters = getAvailableClusters(env.key)
          return (
            <div key={env.key} className="env-row">
              {availableClusters.map(cluster => (
                <div key={cluster.name} className="cluster-cell">
                  <Checkbox
                    checked={isChecked(env.key, cluster.name)}
                    onChange={(e) => handleToggle(env.key, cluster.name, e.target.checked)}
                  >
                    {cluster.name}
                  </Checkbox>
                </div>
              ))}
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default EnvClusterConfig

