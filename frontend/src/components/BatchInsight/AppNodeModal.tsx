import {Button, Descriptions, Empty, List, message, Modal, Popconfirm, Popover, Select, Space, Spin, Table, Tabs, Tag} from 'antd'
import {useEffect, useRef, useState} from 'react'
import {CheckCircleOutlined, FastForwardOutlined, RetweetOutlined, RocketOutlined} from '@ant-design/icons'
import type {BuildSummary} from '@/types'
import {AppStatus, AppStatusLabel, type DeploymentRecord, ReleaseApp} from '@/types/release_app.ts';
import {batchService} from '@/services/batch.ts'
import {useAuthStore} from '@/stores/authStore.ts'
import {Batch, BatchStatus} from "@/types/batch.ts";
import dayjs from 'dayjs'

interface AppNodeModalProps {
  visible: boolean
  releaseApp: ReleaseApp | null
  batch: Batch
  onClose: () => void
  onRefresh?: () => void
}

const AppNodeModal: React.FC<AppNodeModalProps> = ({visible, releaseApp, batch, onClose, onRefresh,}) => {
  const [triggering, setTriggering] = useState(false)
  const [loading, setLoading] = useState(false)
  const [detailData, setDetailData] = useState<ReleaseApp | null>(null)
  const [selectedRecentBuildId, setSelectedRecentBuildId] = useState<number | null>(null)
  const [retryingDeploymentId, setRetryingDeploymentId] = useState<number | null>(null)
  const [activeDeployTab, setActiveDeployTab] = useState<'pre' | 'prod'>('prod')
  const prevVisibleRef = useRef<boolean>(false)
  const prevReleaseIdRef = useRef<number | null>(null)
  const {user} = useAuthStore()

  // 当弹窗打开时，加载详细信息
  useEffect(() => {
    if (visible && releaseApp?.id) {
      loadReleaseAppDetail(false)
    }
  }, [visible, releaseApp?.id])

  const loadReleaseAppDetail = async (silent: boolean) => {
    if (!releaseApp?.id) return

    try {
      if (!silent) setLoading(true)
      const response = await batchService.getReleaseApp(releaseApp.id)
      const data = response?.data
      if (!data) {
        // 防御：避免后端 data 为空导致渲染期异常
        setDetailData(releaseApp)
        return
      }
      setDetailData(data)
      // 默认选择 recent_build：仅在当前未选择/选择项已不存在时才重置，避免轮询抖动
      const builds = data.recent_builds || []
      if (builds.length > 0) {
        const hasSelected = selectedRecentBuildId != null && builds.some((b: BuildSummary) => b.id === selectedRecentBuildId)
        if (!hasSelected) {
          setSelectedRecentBuildId(builds[0].id)
        }
      }

      // 如果该 release_app 的状态发生变化，刷新依赖图节点（由父级重新拉 batchDetail）
      if (releaseApp?.status != null && data?.status != null && Number(data.status) !== Number(releaseApp.status)) {
        onRefresh?.()
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || '加载应用详情失败')
      // 如果加载失败，使用传入的数据
      setDetailData(releaseApp)
    } finally {
      if (!silent) setLoading(false)
    }
  }

  // 部署中：弹窗打开时轮询该 release_app 详情，若状态变化则触发 onRefresh 更新节点
  useEffect(() => {
    if (!visible || !releaseApp?.id) return
    const isDeploying = !!batch && [20, 21, 30, 31].includes(Number(batch.status))
    if (!isDeploying) return

    const timer = window.setInterval(() => {
      loadReleaseAppDetail(true)
    }, 3000)

    return () => window.clearInterval(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, releaseApp?.id, batch?.status])

  const getPreferredDeployTab = (data: ReleaseApp | null): 'pre' | 'prod' => {
    if (!data) return 'prod'
    if (data.skip_pre_env) return 'prod'
    const status = Number(data.status)
    if (status >= 30) return 'prod' // Prod 阶段优先 Prod
    if (status >= 20) return 'pre'  // Pre 阶段优先 Pre
    return 'pre'
  }

  // 弹窗打开时：根据当前 release_app.status 选择默认 tab（不覆盖用户在同一次打开中手动切换）
  useEffect(() => {
    const prevVisible = prevVisibleRef.current
    const prevReleaseId = prevReleaseIdRef.current
    const curReleaseId = releaseApp?.id ?? null

    if (visible && (!prevVisible || prevReleaseId !== curReleaseId)) {
      setActiveDeployTab(getPreferredDeployTab(detailData || releaseApp))
    }

    prevVisibleRef.current = visible
    prevReleaseIdRef.current = curReleaseId
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, releaseApp?.id])

  if (!releaseApp) return null

  // 使用详细数据或传入的数据
  const displayData = detailData || releaseApp

  const renderDeploymentsTable = (deployments: DeploymentRecord[]) => {
    if (!deployments || deployments.length === 0) {
      return <Empty description="暂无部署记录" image={Empty.PRESENTED_IMAGE_SIMPLE}/>
    }

    return (
      <Table<DeploymentRecord>
        size="small"
        pagination={false}
        rowKey={(r) => String(r.id)}
        dataSource={deployments}
        scroll={{y: 360, x: 'max-content'}}
        columns={[
          {title: 'Cluster', dataIndex: 'cluster_name', key: 'cluster_name', width: 120, ellipsis: true},
          {title: 'Namespace', dataIndex: 'namespace', key: 'namespace', width: 120, ellipsis: true},
          {title: 'Deployment', dataIndex: 'deployment_name', key: 'deployment_name', width: 140, ellipsis: true},
          {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            width: 90,
            render: (status: string) => {
              const color =
                status === 'success' ? 'green'
                  : status === 'failed' ? 'red'
                    : status === 'running' ? 'gold'
                      : 'default'
              return <Tag color={color}>{status}</Tag>
            },
          },
          {
            title: '重试',
            key: 'retry',
            width: 80,
            render: (_: any, r: DeploymentRecord) => (
              <span style={{fontVariantNumeric: 'tabular-nums'}}>
                {r.retry_count}/{r.max_retry_count}
              </span>
            ),
          },
          {
            title: '开始/结束',
            key: 'time',
            width: 220,
            render: (_: any, r: DeploymentRecord) => {
              const start = r.started_at ? dayjs(r.started_at) : null
              const end = r.finished_at ? dayjs(r.finished_at) : null
              const startText = start ? start.format('YYYY-MM-DD HH:mm:ss') : '-'
              const endText = end ? end.format('YYYY-MM-DD HH:mm:ss') : '-'
              return (
                <div style={{lineHeight: 1.3}}>
                  <div>Start: <span style={{fontVariantNumeric: 'tabular-nums'}}>{startText}</span></div>
                  <div>End: <span style={{fontVariantNumeric: 'tabular-nums'}}>{endText}</span></div>
                </div>
              )
            },
          },
          {
            title: '耗时',
            key: 'duration',
            width: 90,
            render: (_: any, r: DeploymentRecord) => {
              const start = r.started_at ? dayjs(r.started_at) : null
              const end = r.finished_at ? dayjs(r.finished_at) : null
              if (!start || !end) return <span style={{color: '#999'}}>-</span>
              const sec = Math.max(0, end.diff(start, 'second'))
              const mm = Math.floor(sec / 60)
              const ss = sec % 60
              return (
                <span style={{fontVariantNumeric: 'tabular-nums'}}>
                  {mm}m {ss}s
                </span>
              )
            },
          },
          {
            title: '错误',
            dataIndex: 'error_message',
            key: 'error_message',
            ellipsis: true,
            render: (msg: string | null | undefined) => (
              msg
                ? <Popover content={<div style={{maxWidth: 520, whiteSpace: 'pre-wrap'}}>{msg}</div>}>
                  <span style={{color: '#ff4d4f'}}>查看</span>
                </Popover>
                : <span style={{color: '#999'}}>-</span>
            ),
          },
          {
            title: '操作',
            key: 'action',
            width: 90,
            render: (_: any, r: DeploymentRecord) => {
              const canRetry = r.status === 'failed'
              return (
                <Popconfirm
                  title="确认重试该部署？"
                  description={`Cluster: ${r.cluster_name}`}
                  okText="重试"
                  cancelText="取消"
                  disabled={!canRetry || retryingDeploymentId === r.id}
                  onConfirm={async () => {
                    try {
                      setRetryingDeploymentId(r.id)
                      await batchService.retryDeployment(r.id, {
                        operator: user?.username || 'unknown',
                        reason: '用户手动重试',
                      })
                      message.success('已触发重试')
                      await loadReleaseAppDetail(false)
                      onRefresh?.()
                    } catch (error: any) {
                      message.error(error.response?.data?.message || '重试失败')
                    } finally {
                      setRetryingDeploymentId(null)
                    }
                  }}
                >
                  <Button
                    size="small"
                    type="link"
                    disabled={!canRetry}
                    loading={retryingDeploymentId === r.id}
                    style={{padding: 0}}
                  >
                    重试
                  </Button>
                </Popconfirm>
              )
            },
          },
        ]}
      />
    )
  }

  const handleTriggerDeploy = async (action: string) => {
    if (!batch) {
      message.error('缺少批次ID或环境信息')
      return
    }

    try {
      setTriggering(true)
      const response = await batchService.manualDeploy({
        batch_id: batch.id,
        release_app_id: displayData.id,
        action: action,
        operator: user?.username || 'unknown',
        reason: '手动触发部署',
      })

      message.success(response.data.message || '触发部署成功')
      onClose()
      onRefresh?.()
    } catch (error: any) {
      message.error(error.response?.data?.message || '触发部署失败')
    } finally {
      setTriggering(false)
    }
  }


  // 渲染手动部署按钮
  const renderManualDeployButton = (releaseApp: ReleaseApp, batch: Batch) => {
    if (!batch || !releaseApp) return null

    // batch已封板状态, release可以提前Pre
    if (batch.status >= BatchStatus.Sealed && batch.status < BatchStatus.PreTriggered) {
      if (!releaseApp.skip_pre_env && releaseApp.status === AppStatus.Tagged) {
        return (
          <Popconfirm title="确认提前发布" description={releaseApp.app_name + " => " + releaseApp.target_tag}
                      disabled={triggering} onConfirm={() => handleTriggerDeploy('manual_trigger_pre')}>
            <Button danger icon={<RocketOutlined/>} disabled={loading}>提前[Pre]发布</Button>
          </Popconfirm>
        )
      }
    }
    // batch已经开始预发布, release可以提前Prod
    if (batch.status >= BatchStatus.Sealed && batch.status < BatchStatus.ProdTriggered) {
      if (releaseApp.status == AppStatus.PreDeployed || (releaseApp.skip_pre_env && releaseApp.status === AppStatus.Tagged)) {
        return (
          <Popconfirm title="确认提前发布" description={releaseApp.app_name + " => " + releaseApp.target_tag}
                      onConfirm={() => handleTriggerDeploy("manual_trigger_prod")}>
            <Button danger icon={<RocketOutlined/>} disabled={loading}>提前[Prod]发布</Button>
          </Popconfirm>
        )
      }
    }
    // 重新发布过, 可以继续发Prod
    if (batch.status >= BatchStatus.Sealed && batch.status < BatchStatus.Completed) {
      if (releaseApp.status == AppStatus.PreDeployed) {
        return (
          <Popconfirm title="确认发布" description={releaseApp.app_name + " => " + releaseApp.target_tag}
                      onConfirm={() => handleTriggerDeploy("manual_trigger_prod")}>
            <Button danger icon={<RocketOutlined/>} disabled={loading}>触发[Prod]发布</Button>
          </Popconfirm>
        )
      }
    }
  }


  // 重新发布（直接使用选中的 recent_build）
  const handleSwitchVersion = async (buildId: number) => {
    if (!batch) {
      message.error('缺少批次ID或环境信息')
      return
    }

    try {
      setTriggering(true)
      const response = await batchService.switchVersion({
        batch_id: batch?.id,
        release_app_id: displayData.id,
        operator: user?.username || 'unknown',
        build_id: buildId,
        reason: '用户触发手动发布',
      })

      message.success(response.message || '触发部署成功')
      onClose()
      onRefresh?.()
    } catch (error: any) {
      message.error(error.response?.message || '触发部署失败')
      console.error(error)
    } finally {
      setTriggering(false)
    }
  }

  const hasNewTag = displayData.latest_build_id && displayData.latest_build_id !== displayData.build_id


  return (
    <>
      <Modal
        title={
          <Space>
            <span>发布详情</span>
            <span style={{color: '#999', fontSize: 13, userSelect: 'none'}}>#{displayData.id}</span>
            <span style={{fontWeight: 500}}>{displayData.app_name}</span>
            {hasNewTag && <Tag color="orange">有新版本</Tag>}
          </Space>
        }
        open={visible}
        onCancel={onClose}
        width={650}
        styles={{
          body: {
            maxHeight: '70vh',
            overflowY: 'auto',
          },
        }}
        footer={
          <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
            <span style={{fontSize: 12, color: '#999'}}>
              提示：底部"手动触发"按钮用于批次未开始时手动触发部署
            </span>
            <Space>
              <Button onClick={onClose}>关闭</Button>
              {renderManualDeployButton(displayData, batch)}
            </Space>
          </div>
        }
      >
        <Spin spinning={loading}>
          {/* 基本信息 */}
          <Descriptions column={12} size="small" bordered>
            <Descriptions.Item label="应用名称" span={8}>
              <Space>
                <span style={{fontSize: 12, color: '#999', userSelect: 'none'}}>#{displayData.app_id}</span>
                <span>{displayData.app_name}</span>
                {displayData.app_type && <Tag color="blue">{displayData.app_type}</Tag>}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="发布策略" span={4}>
              <Tag color={displayData.skip_pre_env ? 'orange' : 'blue'}
                   icon={displayData.skip_pre_env ? <FastForwardOutlined/> : <CheckCircleOutlined/>}>
                {displayData.skip_pre_env ? 'only Prod' : 'Pre & Prod'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="之前版本" span={12}>
              <span style={{fontWeight: 500}}>{displayData.previous_deployed_tag || '-'}</span>
            </Descriptions.Item>

            <Descriptions.Item label="目标版本" span={8}>
              <Space>
                {displayData.build_id && (
                  <span style={{fontSize: 12, color: '#999', userSelect: 'none'}}>#{displayData.build_id}</span>
                )}
                <Popover title={displayData.target_tag} content={
                  <div style={{display: 'flex', flexDirection: 'column', minWidth: '300px'}}>
                    <code> Commit SHA: {displayData.commit_sha?.substring(0, 8) || '-'}</code>
                    <code>Commit Message: {displayData.commit_message}</code>
                    <code> Commit Branch: {displayData.commit_branch}</code>
                  </div>
                }>
                  <span style={{fontWeight: 600}}>{displayData.target_tag || '-'}</span>
                </Popover>

              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="当前状态" span={4}>
              <Popover title="最近10行错误"
                       content={
                         <div style={{maxHeight: '400px', maxWidth: '680px', overflow: 'auto', whiteSpace: 'nowrap',}}>
                           <List
                             size="small"
                             dataSource={displayData.reasons}
                             renderItem={(item) => <List.Item>{item}</List.Item>}
                           />
                         </div>
                       }>
                <Tag color={AppStatusLabel[displayData.status]?.color}>
                  {AppStatusLabel[displayData.status]?.label || `状态${displayData.status}`}
                </Tag>
              </Popover>
            </Descriptions.Item>

            {/* 最新 Build 信息 - 根据 recent_builds 数量显示不同的UI */}
            <Descriptions.Item label="最新Build" span={12}>
              {displayData.recent_builds && displayData.recent_builds.length > 0 && (
                <div style={{display: 'flex', alignItems: 'center', justifyContent: 'space-between'}}>
                  {displayData.recent_builds.length === 1 ? (
                    // 只有一个 recent_build，直接显示
                    <Space>
                      <span style={{
                        fontSize: 12,
                        color: '#999',
                        userSelect: 'none'
                      }}>#{displayData.recent_builds[0].id}</span>
                      <span style={{fontWeight: 500}}>{displayData.recent_builds[0].image_tag}</span>
                      <span style={{color: '#ff4d4f'}}>有新版本可用</span>
                    </Space>
                  ) : (
                    // 多个 recent_builds，显示下拉列表
                    <Select
                      value={selectedRecentBuildId}
                      onChange={setSelectedRecentBuildId}
                      style={{minWidth: 300}}
                      placeholder="选择构建版本"
                    >
                      {displayData.recent_builds.map((build: BuildSummary) => (
                        <Select.Option key={build.id} value={build.id}>
                          <Space>
                            <span style={{fontSize: 12, color: '#999', userSelect: 'none'}}>#{build.id}</span>
                            <span style={{fontWeight: 600}}>{build.image_tag}</span>
                            <span style={{fontSize: 12, color: '#666', userSelect: 'none'}}>
                              {build.commit_message?.substring(0, 30)}
                              {(build.commit_message?.length || 0) > 30 ? '...' : ''}
                            </span>
                          </Space>
                        </Select.Option>
                      ))}
                    </Select>
                  )}
                  <Popconfirm title="确认部署新版本?"
                              onConfirm={() => {
                                selectedRecentBuildId ? handleSwitchVersion(selectedRecentBuildId).then() : message.warning('请先选择要部署的构建版本').then()
                              }}>
                    <Button type="primary" size="small" icon={<RetweetOutlined/>}>切换版本</Button>
                  </Popconfirm>
                </div>
              )}
            </Descriptions.Item>
          </Descriptions>

          {/* 部署详情（deployment 维度） */}
          <div style={{marginTop: 16}}>
            <div style={{fontWeight: 500, marginBottom: 8}}>部署详情</div>
            {(() => {
              const items = [
                ...(displayData.skip_pre_env
                  ? []
                  : [{
                    key: 'pre',
                    label: 'Pre',
                    children: renderDeploymentsTable((displayData.deployments || []).filter(d => d.env === 'pre')),
                  }]),
                {
                  key: 'prod',
                  label: 'Prod',
                  children: renderDeploymentsTable((displayData.deployments || []).filter(d => d.env === 'prod')),
                },
              ]
              const keys = new Set(items.map(i => i.key))
              const safeKey = keys.has(activeDeployTab) ? activeDeployTab : (items[0]?.key as 'pre' | 'prod' | undefined) || 'prod'
              return (
                <Tabs
                  size="small"
                  activeKey={safeKey}
                  onChange={(k) => setActiveDeployTab(k as 'pre' | 'prod')}
                  items={items}
                />
              )
            })()}
          </div>
        </Spin>
      </Modal>

    </>
  )
}

export default AppNodeModal
