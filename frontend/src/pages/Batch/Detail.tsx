import type {ReactNode} from 'react'
import {useEffect, useMemo, useState} from 'react'
import {Button, Card, Checkbox, Empty, Input, message, Modal, Segmented, Select, Space, Spin, Table, Tag,} from 'antd'
import {
  CheckCircleOutlined,
  EditOutlined,
  FastForwardOutlined,
  LeftOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SaveOutlined,
  StopOutlined,
  UndoOutlined,
} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {useNavigate, useParams, useSearchParams} from 'react-router-dom'
import {useTranslation} from 'react-i18next'
import dayjs from 'dayjs'
import type {ColumnsType} from 'antd/es/table'
import {batchService} from '@/services/batch'
import {BatchTimeline} from '@/components/BatchTimeline'
import {useAuthStore} from '@/stores/authStore'
import type {BatchActionRequest, BuildSummary, UpdateReleaseDependenciesRequest,} from '@/types'
import DependencyGraph from '@/components/BatchInsight/DependencyGraph'
import AppSelectionTable from '@/components/AppSelectionTable'
import '@/styles/status-theme.css'
import './Detail.css'
import {ReleaseApp} from "@/types/release_app.ts";
import {Batch, BatchAction, BatchStatus} from "@/types/batch.ts";

type DependencyOption = {
  label: ReactNode
  value: number
  disabled?: boolean
}

const {TextArea} = Input

export default function BatchDetail() {
  const {t} = useTranslation()
  const navigate = useNavigate()
  const {id} = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const {user} = useAuthStore()
  const queryClient = useQueryClient()
  const [cancelModalVisible, setCancelModalVisible] = useState(false)
  const [cancelReason, setCancelReason] = useState('')
  const [manageAppsModalVisible, setManageAppsModalVisible] = useState(false)
  const [selectedAppIds, setSelectedAppIds] = useState<number[]>([])

  // 依赖配置状态
  const [dependencyModalVisible, setDependencyModalVisible] = useState(false)
  const [editingRelease, setEditingRelease] = useState<ReleaseApp | null>(null)
  const [tempDependencySelection, setTempDependencySelection] = useState<number[]>([])

  // 视图模式状态（从 URL 参数读取默认值，如果没有则为 'list'）
  const tabParam = searchParams.get('tab') as 'list' | 'graph' | null
  const [viewMode, setViewMode] = useState<'list' | 'graph'>(
    tabParam === 'graph' || tabParam === 'list' ? tabParam : 'list'
  )

  // 构建修改状态（app_id -> selected_build_id）
  const [buildChanges, setBuildChanges] = useState<Record<number, number>>({})

  // 查询批次详情
  const {data: batchData, isLoading} = useQuery({
    queryKey: ['batchDetail', id],
    queryFn: async () => {
      const withoutRecentBuilds = !!batch && batch.status >= BatchStatus.Sealed
      const res = await batchService.get(Number(id), 1, 200, !withoutRecentBuilds)
      return res.data as Batch
    },
    enabled: !!id,
  })

  const batch = batchData

  // 根据批次状态设置默认视图（仅在首次加载且没有 URL 参数指定时）
  useEffect(() => {
    // 如果 URL 中有明确指定 tab 参数，则不自动切换
    if (tabParam) return

    if (batch && viewMode === 'list') {
      // status >= 22 表示预发布完成或更晚，默认显示图形视图
      if (batch.status >= 22) {
        setViewMode('graph')
        // 同步更新 URL 参数
        const newSearchParams = new URLSearchParams(searchParams)
        newSearchParams.set('tab', 'graph')
        navigate(`/batch/${id}/detail?${newSearchParams.toString()}`, {replace: true})
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [batch?.status])

  const appIdMap = useMemo(() => {
    const map = new Map<number, ReleaseApp>()
    batch?.apps?.forEach(app => {
      map.set(app.app_id, app)
    })
    return map
  }, [batch?.apps])

  // 计算应用统计信息
  const appStatistics = useMemo(() => {
    const apps = batch?.apps || []
    const preApps = apps.filter(app => !app.skip_pre_env)
    const skipPreApps = apps.filter(app => app.skip_pre_env)
    return {
      total: apps.length,
      preAppsCount: preApps.length,
      skipPreCount: skipPreApps.length,
      hasPreApps: preApps.length > 0,
      allSkipPre: apps.length > 0 && skipPreApps.length === apps.length,
    }
  }, [batch?.apps])


  // 更新批次应用 Mutation
  const updateAppsMutation = useMutation({
    mutationFn: (data: { add_app_ids: number[]; remove_app_ids: number[] }) =>
      batchService.update({
        batch_id: Number(id),
        operator: user?.username || 'unknown',
        add_apps: data.add_app_ids.map(app_id => ({app_id})),
        remove_app_ids: data.remove_app_ids,
      }),
    onSuccess: () => {
      message.success(t('batch.updateSuccess'))
      queryClient.invalidateQueries({queryKey: ['batchDetail', id]})
      setManageAppsModalVisible(false)
    },
    onError: () => {
    },
  })
  const updateDependenciesMutation = useMutation({
    mutationFn: ({
                   releaseAppId,
                   payload,
                 }: {
      releaseAppId: number
      payload: UpdateReleaseDependenciesRequest
    }) => batchService.updateDependencies(releaseAppId, payload),
    onSuccess: () => {
      message.success('依赖配置已更新')
      queryClient.invalidateQueries({queryKey: ['batchDetail', id]})
      setDependencyModalVisible(false)
      setEditingRelease(null)
      setTempDependencySelection([])
    },
    onError: () => {
    },
  })

  // 批次操作 Mutation
  const actionMutation = useMutation({
    mutationFn: (data: BatchActionRequest) => batchService.action(data),
    onSuccess: (_response, variables) => {
      const actionMessages: Record<string, string> = {
        seal: t('batch.sealSuccess'),
        start_pre_deploy: t('batch.startPreDeploySuccess'),
        finish_pre_deploy: t('batch.finishPreDeploySuccess'),
        start_prod_deploy: t('batch.startProdDeploySuccess'),
        finish_prod_deploy: t('batch.finishProdDeploySuccess'),
        complete: t('batch.completeSuccess'),
        cancel: t('batch.cancelSuccess'),
      }
      message.success(actionMessages[variables.action] || t('common.success'))
      queryClient.invalidateQueries({queryKey: ['batchDetail', id]})
      queryClient.invalidateQueries({queryKey: ['batchList']})
      setCancelModalVisible(false)
      setCancelReason('')
    },
    // 移除 onError 中的 message.error，避免与拦截器重复显示
    // 拦截器已经统一处理了错误提示
  })

  // 处理操作
  const handleAction = (action: BatchActionRequest['action'], needConfirm = true) => {
    if (!batch) return

    const confirmMessages: Record<string, string> = {
      seal: t('batch.sealConfirm'),
      start_pre_deploy: t('batch.startPreDeployConfirm'),
      start_prod_deploy: t('batch.startProdDeployConfirm'),
      complete: t('batch.completeConfirm'),
    }

    const doAction = () => {
      actionMutation.mutate({
        batch_id: batch.id,
        action,
        operator: user?.username || 'unknown',
        reason: action === 'cancel' ? cancelReason : undefined,
      })
    }

    if (needConfirm && confirmMessages[action]) {
      Modal.confirm({
        title: t('common.confirm'),
        content: confirmMessages[action],
        onOk: doAction,
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
      })
    } else {
      doAction()
    }
  }

  // 处理取消批次
  const handleCancel = () => {
    setCancelModalVisible(true)
  }

  // 确认取消
  const confirmCancel = () => {
    if (!cancelReason.trim()) {
      message.warning('请输入取消原因')
      return
    }
    handleAction('cancel', false)
  }

  // 提交应用修改
  const handleSubmitApps = () => {
    if (!batch) return

    const currentAppIds = batch.apps?.map(app => app.app_id) || []
    const addAppIds = selectedAppIds.filter(id => !currentAppIds.includes(id))
    const removeAppIds = currentAppIds.filter(id => !selectedAppIds.includes(id))

    if (addAppIds.length === 0 && removeAppIds.length === 0) {
      message.info('没有变更')
      setManageAppsModalVisible(false)
      return
    }

    updateAppsMutation.mutate({
      add_app_ids: addAppIds,
      remove_app_ids: removeAppIds,
    })
  }

  const handleOpenDependencies = (release: ReleaseApp) => {
    setEditingRelease(release)
    setTempDependencySelection(release.temp_depends_on || [])
    setDependencyModalVisible(true)
  }

  const handleDependencySelectionChange = (values: Array<number | string>) => {
    if (!editingRelease) return
    const defaultSet = new Set(editingRelease.default_depends_on || [])
    const numericValues = values.map(value => Number(value))
    const filtered = numericValues.filter(id => !defaultSet.has(id))
    setTempDependencySelection(filtered)
  }

  const handleSaveDependencies = () => {
    if (!batch || !editingRelease) return
    updateDependenciesMutation.mutate({
      releaseAppId: editingRelease.id,
      payload: {
        batch_id: batch.id,
        operator: user?.username || 'unknown',
        temp_depends_on: tempDependencySelection,
      },
    })
  }

  const handleCloseDependencyModal = () => {
    setDependencyModalVisible(false)
    setEditingRelease(null)
    setTempDependencySelection([])
  }

  // 处理构建选择变更
  const handleBuildChange = (appId: number, buildId: number) => {
    setBuildChanges(prev => ({
      ...prev,
      [appId]: buildId
    }))
  }

  // 保存构建变更
  const handleSaveBuildChanges = async () => {
    if (!batch) return
    try {
      await batchService.updateBuilds({
        batch_id: batch.id,
        operator: user?.username || 'unknown',
        build_changes: buildChanges,
      })
      message.success('构建版本更新成功')

      // 清空修改记录
      setBuildChanges({})

      // 刷新批次详情
      await queryClient.invalidateQueries({queryKey: ['batchDetail', id]})
    } catch (error: any) {
      message.error(error.response?.data?.message || '更新失败，请重试')
    }
  }

  // 还原/取消所有修改
  const handleCancelBuildChanges = () => {
    setBuildChanges({})
    message.info('已取消所有修改')
  }

  // 打开管理应用 Modal
  const handleManageApps = () => {
    // 初始化已选应用 ID
    const currentAppIds = batch?.apps?.map(app => app.app_id) || []
    setSelectedAppIds(currentAppIds)
    setManageAppsModalVisible(true)
  }

  const dependencyOptions = useMemo<DependencyOption[]>(() => {
    if (!batch || !editingRelease) return []
    const defaultSet = new Set(editingRelease.default_depends_on || [])
    return (batch.apps || [])
      .filter(app => app.app_id !== editingRelease.app_id)
      .map(app => ({
        label: (
          <Space size={4}>
            <span>{app.app_name}</span>
            <Tag color="blue">{app.app_type}</Tag>
            {defaultSet.has(app.app_id) && <Tag color="purple">默认</Tag>}
          </Space>
        ),
        value: app.app_id,
        disabled: defaultSet.has(app.app_id),
      }))
  }, [batch, editingRelease])

  const dependencyValues = useMemo<number[]>(() => {
    if (!editingRelease) return []
    const defaultDeps = editingRelease.default_depends_on || []
    return Array.from(new Set([...defaultDeps, ...tempDependencySelection]))
  }, [editingRelease, tempDependencySelection])

  const dependencyOptionValueSet = useMemo<Set<number>>(() => {
    return new Set(dependencyOptions.map(option => Number(option.value)))
  }, [dependencyOptions])

  const dependencyGroupValues = useMemo<number[]>(() => {
    if (dependencyOptionValueSet.size === 0) {
      return []
    }
    return dependencyValues.filter(id => dependencyOptionValueSet.has(id))
  }, [dependencyValues, dependencyOptionValueSet])

  const missingDefaultDependencies = useMemo<number[]>(() => {
    if (!editingRelease) return []
    const defaultDeps = editingRelease.default_depends_on || []
    return defaultDeps.filter(id => !appIdMap.has(id))
  }, [editingRelease, appIdMap])

  // 根据状态判断可用操作
  const getAvailableActions = () => {
    if (!batch) return []

    const actions: Array<{
      key: string
      label: string
      icon: React.ReactNode
      type?: 'primary' | 'default' | 'link'
      danger?: boolean
      action: BatchActionRequest['action']
    }> = []

    // 已封板：根据是否有pre应用决定显示哪个按钮
    if (batch.status === 10) {
      if (appStatistics.hasPreApps) {
        // 有需要pre的应用，显示"开始预发布"按钮
        actions.push({
          key: 'start_pre_deploy',
          label: `${t('batch.startPreDeploy')} (${appStatistics.preAppsCount} 个应用)`,
          icon: <PlayCircleOutlined/>,
          type: 'primary',
          action: BatchAction.StartPreDeploy,
        })
      }

      if (appStatistics.allSkipPre) {
        // 所有应用都跳过pre，显示"直接开始生产部署"按钮
        actions.push({
          key: 'start_prod_deploy',
          label: `${t('batch.startProdDeploy')} (跳过预发布)`,
          icon: <FastForwardOutlined/>,
          type: 'primary',
          action: BatchAction.StartProdDeploy,
        })
      }
    }

    // 预发布中：可以完成预发布
    if (batch.status === 21) {
      actions.push({
        key: 'finish_pre_deploy',
        label: t('batch.finishPreDeploy'),
        icon: <CheckCircleOutlined/>,
        type: 'primary',
        action: 'finish_pre_deploy',
      })
    }

    // 预发布完成：可以开始生产部署
    if (batch.status === 22) {
      actions.push({
        key: 'start_prod_deploy',
        label: t('batch.startProdDeploy'),
        icon: <PlayCircleOutlined/>,
        type: 'primary',
        action: BatchAction.StartProdDeploy,
      })
    }

    // 生产部署中：可以完成生产部署
    if (batch.status === 31) {
      actions.push({
        key: 'finish_prod_deploy',
        label: t('batch.finishProdDeploy'),
        icon: <CheckCircleOutlined/>,
        type: 'primary',
        action: 'finish_prod_deploy',
      })
    }

    // 生产部署完成：可以最终验收
    if (batch.status === 32) {
      actions.push({
        key: 'complete',
        label: t('batch.complete'),
        icon: <CheckCircleOutlined/>,
        type: 'primary',
        action: 'complete',
      })
    }

    // 审批通过：可以封板
    if (batch.status === 0 && batch.approval_status === 'approved') {
      actions.push({
        key: 'seal',
        label: t('batch.seal'),
        icon: <CheckCircleOutlined/>,
        type: 'primary',
        action: 'seal',
      })
    }

    // 未完成且未取消的批次可以取消
    if (batch.status < 40 && batch.status !== 90) {
      actions.push({
        key: 'cancel',
        label: t('batch.cancelBatch'),
        icon: <StopOutlined/>,
        danger: true,
        type: 'link',
        action: 'cancel',
      })
    }

    return actions
  }

  if (isLoading) {
    return (
      <div style={{padding: 24, textAlign: 'center'}}>
        <Spin size="large"/>
      </div>
    )
  }

  if (!batch) {
    return (
      <div style={{padding: 24, textAlign: 'center'}}>
        <Empty description="批次不存在"/>
      </div>
    )
  }

  const batchStatusValue = Number(batch.status)
  const isBatchCompleted = batchStatusValue === 40

  const appColumns: ColumnsType<ReleaseApp> = [
    {
      title: t('batch.appName'),
      dataIndex: 'app_name',
      key: 'app_name',
      width: 180,
      fixed: 'left',
      ellipsis: true,
      render: (name: string, record: ReleaseApp) => (
        <div style={{display: 'flex', alignItems: 'center', overflow: 'hidden'}}>
          <span style={{
            color: '#999',
            fontSize: 12,
            userSelect: 'none',
            flexShrink: 0,
            marginRight: 4
          }}>#{record.app_id} </span>
          <span style={{
            fontWeight: 500,
            fontSize: 13,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}>{name}
          </span>
        </div>
      )
    },
    {
      title: t('batch.appType'),
      dataIndex: 'app_type',
      key: 'app_type',
      width: 100,
      render: (type: string) => (
        <Tag color={batch?.app_type_configs?.[type]?.color || 'blue'}>{type}</Tag>
      ),
    },
    {
      title: '发布策略',
      dataIndex: 'skip_pre_env',
      key: 'skip_pre_env',
      width: 120,
      render: (skipPre: boolean) => (
        <Tag
          color={skipPre ? 'orange' : 'blue'}
          icon={skipPre ? <FastForwardOutlined/> : <CheckCircleOutlined/>}
        >
          {skipPre ? '直接Prod' : 'Pre+Prod'}
        </Tag>
      ),
    },
    {
      title: '代码库',
      dataIndex: 'repo_full_name',
      key: 'repo_full_name',
      width: 200,
      ellipsis: true,
      render: (text: string) => text || '-'
    },
    {
      // 当前生产版本
      title: isBatchCompleted ? t('batch.oldVersion') : t('batch.currentVersion'),
      key: isBatchCompleted ? 'old_version' : 'current_version',
      width: 140,
      ellipsis: true,
      render: (_: any, record: ReleaseApp) => (
        isBatchCompleted ? (record.previous_deployed_tag || '-') : (record.deployed_tag || '-')
      ),
    },
    {
      // 待部署
      title: isBatchCompleted ? t('batch.deployed') : t('batch.pendingDeploy'),
      key: isBatchCompleted ? 'deployed' : 'pending_deploy',
      width: 250,
      render: (_: any, record: ReleaseApp) => {
        // 如果批次未封板且有 recent_builds，显示下拉选择
        if (!isBatchCompleted && batch && batch.status < 10 && record.recent_builds && record.recent_builds.length > 0) {
          const currentValue = buildChanges[record.app_id] || record.build_id
          const isModified = record.app_id in buildChanges
          const initialBuildId = record.build_id

          // 处理长文本：优先显示后段
          const formatLabel = (text: string, maxLen: number = 20) => {
            if (!text) return ''
            if (text.length <= maxLen) return text
            return '...' + text.slice(-(maxLen - 3))
          }

          // 获取当前选中的构建的 commit message
          const selectedBuild = record.recent_builds.find((b: BuildSummary) => b.id === currentValue)
          const displayCommitMessage = selectedBuild?.commit_message || record.commit_message

          return (
            <div>
              <div style={{display: 'flex', alignItems: 'center', gap: 4, marginBottom: 4}}>
                <Select
                  style={{width: 'calc(100% - 22px)', fontSize: 13}}
                  value={currentValue}
                  onChange={(value) => handleBuildChange(record.app_id, value)}
                  size="small"
                  optionLabelProp="label"
                  status={isModified ? 'warning' : undefined}
                  dropdownMatchSelectWidth={false}
                  dropdownStyle={{width: 280}}
                >
                  {record.recent_builds.map((build: BuildSummary) => {
                    const isInitial = build.id === initialBuildId
                    return (
                      <Select.Option
                        key={build.id}
                        value={build.id}
                        label={formatLabel(build.image_tag)}
                      >
                        <div style={{fontSize: 11}}>
                          <div>
                            <code style={{fontSize: 12, fontWeight: isInitial ? 600 : 400}}>
                              {build.image_tag}
                            </code>
                          </div>
                          <div style={{
                            color: '#8c8c8c',
                            fontSize: 11,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                            direction: 'rtl',
                            textAlign: 'left',
                          }}>
                            {build.commit_message || ''}
                          </div>
                          <div style={{
                            color: '#8c8c8c',
                            fontSize: 10,
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                          }}>
                            <span>{dayjs(build.build_created).format('YYYY-MM-DD HH:mm')}</span>
                            <span style={{marginLeft: 8, flexShrink: 0}}>#{build.id}</span>
                          </div>
                        </div>
                      </Select.Option>
                    )
                  })}
                </Select>
                {isModified ? (
                  <Button
                    type="text"
                    size="small"
                    icon={<UndoOutlined/>}
                    style={{
                      padding: '0 4px',
                      minWidth: '22px',
                      height: '22px',
                      color: '#faad14',
                    }}
                    onClick={(e) => {
                      e.stopPropagation()
                      // 还原单个应用的修改
                      setBuildChanges(prev => {
                        const newChanges = {...prev}
                        delete newChanges[record.app_id]
                        return newChanges
                      })
                    }}
                  />
                ) : (
                  <span style={{width: 22, height: 22, display: 'inline-block'}}/>
                )}
              </div>
              {displayCommitMessage && (
                <div style={{
                  fontSize: 11,
                  color: '#8c8c8c',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}>
                  {displayCommitMessage}
                </div>
              )}
            </div>
          )
        }
        // 否则显示普通文本
        return (
          <div>
            <div style={{marginBottom: 4}}>
              <code style={{fontSize: 12}}>{record.target_tag || '-'}</code>
            </div>
            {record.commit_message && (
              <div style={{
                fontSize: 11,
                color: '#8c8c8c',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}>
                {record.commit_message}
              </div>
            )}
          </div>
        )
      },
    },
    {
      title: '依赖',
      key: 'dependencies',
      width: 220,
      render: (_: any, record: ReleaseApp) => {
        const defaultDeps = record.default_depends_on || []
        const tempDeps = record.temp_depends_on || []
        const defaultSet = new Set(defaultDeps)
        const tempOnly = tempDeps.filter(id => !defaultSet.has(id))
        const defaultTags = defaultDeps.map(depId => {
          const target = appIdMap.get(depId)
          const label = target?.app_name || `#${depId}`
          return (
            <Tag key={`default-${record.id}-${depId}`} color="purple">
              {label}
              <span style={{marginLeft: 4, fontSize: 10}}>默认</span>
            </Tag>
          )
        })
        const tempTags = tempOnly.map(depId => {
          const target = appIdMap.get(depId)
          const label = target?.app_name || `#${depId}`
          return (
            <Tag key={`temp-${record.id}-${depId}`} color="geekblue">
              {label}
              <span style={{marginLeft: 4, fontSize: 10}}>临时</span>
            </Tag>
          )
        })
        const hasDeps = defaultTags.length + tempTags.length > 0

        const canEditDependencies = !!batch && batch.status < 10 && !record.is_locked

        return (
          <Space direction="vertical" size={4}>
            {hasDeps ? (
              <Space size={[4, 4]} wrap>
                {defaultTags}
                {tempTags}
              </Space>
            ) : (
              <span style={{fontSize: 12, color: '#8c8c8c'}}>无</span>
            )}
            {!!batch && batch.status < 10 && (
              <Button
                size="small"
                onClick={() => handleOpenDependencies(record)}
                disabled={!canEditDependencies}
              >
                设置依赖
              </Button>
            )}
          </Space>
        )
      },
    },
  ]

  // 根据批次状态获取对应的CSS类名（用于卡片样式）
  const getStatusClassName = () => {
    const status = Number(batch.status)
    if (status === BatchStatus.Draft) return 'status-draft'
    if (status === BatchStatus.Sealed) return 'status-sealed'
    if (status === BatchStatus.PreTriggered) return 'status-pre-deploying' // 不是用waiting css
    if (status === BatchStatus.PreDeploying) return 'status-pre-deploying'
    if (status === BatchStatus.PreDeployed) return 'status-pre-deployed'
    if (status === BatchStatus.ProdTriggered) return 'status-prod-deploying' // 不使用waiting css
    if (status === BatchStatus.ProdDeploying) return 'status-prod-deploying'
    if (status === BatchStatus.ProdDeployed) return 'status-prod-deployed'
    if (status === BatchStatus.Completed) return 'status-completed'
    if (status === BatchStatus.Cancelled) return 'status-cancelled'
    return ''
  }

  return (
    <div className="batch-detail-container">

      {/* 头部区域 */}
      <div className="batch-detail-header">
        <Button icon={<LeftOutlined/>} onClick={() => navigate('/batch')} type="link">
          {t('batchInsights.back')}
        </Button>
        <div className="batch-detail-header-controls">
          <Space size="middle">
            {/* 视图切换 */}
            <Segmented
              value={viewMode}
              onChange={(value) => {
                const newMode = value as 'list' | 'graph'
                setViewMode(newMode)
                // 同步更新 URL 参数
                const newSearchParams = new URLSearchParams(searchParams)
                newSearchParams.set('tab', newMode)
                navigate(`/batch/${id}/detail?${newSearchParams.toString()}`, {replace: true})
              }}
              options={[
                {label: t('batch.modeAppList'), value: 'list'},
                {label: t('batch.modeInsights'), value: 'graph'},
              ]}
            />
            {/* 刷新按钮 */}
            <Button
              icon={<ReloadOutlined/>}
              onClick={() => {
                queryClient.invalidateQueries({queryKey: ['batchDetail', id]}).then(
                  message.success('已刷新')
                )
              }}
            >刷新
            </Button>
          </Space>
        </div>
      </div>

      {/* 主区域 */}
      <div className="batch-detail-content">
        {/* 上方: 批次信息和时间线 Section */}
        <Card
          className={`batch-detail-section batch-info-section batch-info-fixed-height ${getStatusClassName()}`}
          title={
            <div className="batch-info-title">
              <div className="title-main">
                <span className="batch-id">#{batch.id}</span>
                <span className="batch-name">{batch.batch_number}</span>
              </div>
              {batch.release_notes && (
                <div className="release-notes">{batch.release_notes}</div>
              )}
            </div>
          }
          extra={
            <Space size="small">
              {/* 操作按钮 */}
              {getAvailableActions().map((action) => (
                <Button
                  key={action.key}
                  type={action.type}
                  size="small"
                  danger={action.danger}
                  icon={action.icon}
                  loading={actionMutation.isPending}
                  onClick={() => {
                    if (action.action === 'cancel') {
                      handleCancel()
                    } else {
                      handleAction(action.action)
                    }
                  }}
                >{action.label}
                </Button>
              ))}
            </Space>
          }
        >
          <div className="batch-info-content">
            {viewMode === 'list' ? (
              /* 应用列表模式：显示批次基本信息 */
              <div className="batch-descriptions-wrapper">
                <div className="batch-info-item">
                  <span className="batch-info-label">{t('batch.initiator')}:</span>
                  <span className="batch-info-value">{batch.initiator}</span>
                </div>
                <div className="batch-info-item">
                  <span className="batch-info-label">{t('batch.createdAt')}:</span>
                  <span className="batch-info-value">{dayjs(batch.created_at).format('YYYY-MM-DD HH:mm:ss')}</span>
                </div>
                <div className="batch-info-item">
                  <span className="batch-info-label">{t('batch.totalApps')}:</span>
                  <span className="batch-info-value">
                    {appStatistics.total} 个应用
                    {appStatistics.total > 0 && (
                      <span style={{marginLeft: 8, fontSize: 12, color: '#8c8c8c'}}>
                        (
                        {appStatistics.hasPreApps && (
                          <span>
                            <Tag color="blue" style={{margin: '0 4px'}}>
                              {appStatistics.preAppsCount} 需预发布
                            </Tag>
                          </span>
                        )}
                        {appStatistics.skipPreCount > 0 && (
                          <span>
                            <Tag color="orange" style={{margin: '0 4px'}}>
                              {appStatistics.skipPreCount} 跳过预发布
                            </Tag>
                          </span>
                        )}
                        )
                      </span>
                    )}
                  </span>
                </div>
              </div>
            ) : (
              /* 发布详情模式：显示上线流程时间线 */
              <div className="batch-timeline-wrapper">
                <BatchTimeline
                  batch={batch}
                  hasPreApps={appStatistics.hasPreApps}
                  onAction={(action) => handleAction(action as BatchActionRequest['action'])}
                />
              </div>
            )}
          </div>
        </Card>

        {/* 下方: 应用列表或依赖图 Section */}
        <Card
          className="batch-detail-section batch-content-card"
          title={(viewMode === 'list' ? t('batch.appList') : t('batchInsights.appDetails')) +
            " (" + (batch.total_apps || batch.apps?.length || 0) + " " + t('batch.apps') + ")"}
          extra={
            viewMode === 'list' && (
              <Space size="small">
                {Object.keys(buildChanges).length > 0 ? (
                  /* 有构建修改时，只显示还原和应用按钮 */
                  <>
                    <Button icon={<UndoOutlined/>} onClick={handleCancelBuildChanges} size="small">
                      还原全部
                    </Button>
                    <Button type="primary" icon={<SaveOutlined/>} onClick={handleSaveBuildChanges} size="small">
                      应用全部 ({Object.keys(buildChanges).length})
                    </Button>
                  </>
                ) : (
                  /* 没有构建修改时，显示编辑应用和封板按钮（草稿状态） */
                  batch.status < 10 && (
                    <>
                      <Button icon={<EditOutlined/>} onClick={handleManageApps} size="small">
                        添加应用
                      </Button>
                      <Button
                        type="primary"
                        icon={<CheckCircleOutlined/>}
                        onClick={() => handleAction('seal')}
                        size="small"
                      >
                        {t('batch.seal')}
                      </Button>
                    </>
                  )
                )}
              </Space>
            )
          }
        >
          {viewMode === 'list' ? (
            /* 应用列表模式：显示应用列表 */
            <Table
              key={`batch-table-${batchStatusValue}-${isBatchCompleted ? 'completed' : 'in-progress'}`}
              columns={appColumns}
              dataSource={batch.apps || []}
              rowKey="id"
              pagination={false}
              scroll={{x: 1200}}
              expandable={{
                expandedRowRender: (record) => (
                  <div style={{padding: '12px 24px', background: '#fafafa'}}>
                    {record.release_notes && (
                      <div style={{marginBottom: 8}}>
                        <strong>{t('batch.appReleaseNotes')}:</strong>
                        <div style={{marginTop: 4, whiteSpace: 'pre-wrap'}}>
                          {record.release_notes}
                        </div>
                      </div>
                    )}
                    <div style={{fontSize: 12, color: '#8c8c8c'}}>
                      <div>Commit: {record.commit_sha?.substring(0, 8)}</div>
                      <div>Image: {record.image_url}</div>
                    </div>
                  </div>
                ),
                rowExpandable: (record) => !!record.release_notes || !!record.commit_sha,
              }}
            />
          ) : (
            /* 发布详情模式：显示应用依赖图 */
            <DependencyGraph releaseApps={batch.apps || []} batch={batch} appTypeConfigs={batch.app_type_configs}
                             onRefresh={() => queryClient.invalidateQueries({queryKey: ['batchDetail', id]})}
            />
          )}
        </Card>
      </div>

      {/* 取消批次 Modal */}
      <Modal
        title={t('batch.cancelBatch')}
        open={cancelModalVisible}
        onOk={confirmCancel}
        onCancel={() => {
          setCancelModalVisible(false)
          setCancelReason('')
        }}
        confirmLoading={actionMutation.isPending}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <div style={{marginBottom: 16}}>{t('batch.cancelConfirm')}</div>
        <TextArea
          rows={4}
          placeholder="请输入取消原因..."
          value={cancelReason}
          onChange={(e) => setCancelReason(e.target.value)}
        />
      </Modal>

      {/*设置依赖 Modal*/}
      <Modal
        title="设置依赖"
        open={dependencyModalVisible}
        onOk={handleSaveDependencies}
        onCancel={handleCloseDependencyModal}
        confirmLoading={updateDependenciesMutation.isPending}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={520}
      >
        {editingRelease ? (
          <div>
            <div style={{marginBottom: 12}}>
              <div style={{fontWeight: 500, display: 'flex', alignItems: 'center', gap: 8}}>
                <span>{editingRelease.app_name}</span>
                {editingRelease.app_type && <Tag color="blue">{editingRelease.app_type}</Tag>}
              </div>
              <div style={{fontSize: 12, color: '#8c8c8c'}}>
                默认依赖已锁定不可取消，可额外勾选临时依赖。
              </div>
            </div>

            {dependencyOptions.length > 0 ? (
              <Checkbox.Group
                style={{display: 'flex', flexDirection: 'column', gap: 8}}
                value={dependencyGroupValues}
                options={dependencyOptions}
                onChange={handleDependencySelectionChange}
              />
            ) : (
              <Empty description="无可配置的依赖应用" image={Empty.PRESENTED_IMAGE_SIMPLE}/>
            )}

            {missingDefaultDependencies.length > 0 && (
              <div style={{marginTop: 12, fontSize: 12, color: '#faad14'}}>
                以下默认依赖未包含在当前批次中：
                {missingDefaultDependencies.map(id => appIdMap.get(id)?.app_name || `#${id}`).join('、')}
              </div>
            )}
          </div>
        ) : (
          <Spin/>
        )}
      </Modal>

      {/* 管理应用 Modal */}
      <Modal
        title={t('batch.manageApps')}
        open={manageAppsModalVisible}
        onOk={handleSubmitApps}
        onCancel={() => setManageAppsModalVisible(false)}
        confirmLoading={updateAppsMutation.isPending}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={1200}
        style={{top: 20}}
      >
        <AppSelectionTable
          projectId={batch?.project_id}
          selection={{
            selectedIds: selectedAppIds,
            existingIds: batch?.apps?.map(app => app.app_id) || [],
            mode: 'edit',
          }}
          onSelectionChange={(selectedIds) => {
            setSelectedAppIds(selectedIds)
          }}
          showReleaseNotes={false}
        />
      </Modal>
    </div>
  )
}

