import {useMemo, useState} from 'react'
import type {ReactNode} from 'react'
import {Button, Card, Checkbox, Descriptions, Empty, Input, message, Modal, Pagination, Select, Space, Spin, Table, Tag,} from 'antd'
import {
  ArrowLeftOutlined,
  CheckCircleOutlined,
  EditOutlined,
  PlayCircleOutlined,
  SaveOutlined,
  StopOutlined,
} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {useNavigate, useParams} from 'react-router-dom'
import {useTranslation} from 'react-i18next'
import dayjs from 'dayjs'
import type {ColumnsType} from 'antd/es/table'
import {batchService} from '@/services/batch'
import {applicationService} from '@/services/application'
import {StatusTag} from '@/components/StatusTag'
import {BatchTimeline} from '@/components/BatchTimeline'
import {useAuthStore} from '@/stores/authStore'
import type {
  ApplicationWithBuild,
  Batch,
  BatchActionRequest,
  BuildSummary,
  ReleaseApp,
  UpdateReleaseDependenciesRequest,
} from '@/types'
import './Detail.css'

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
  const {user} = useAuthStore()
  const queryClient = useQueryClient()
  const [cancelModalVisible, setCancelModalVisible] = useState(false)
  const [cancelReason, setCancelReason] = useState('')
  const [manageAppsModalVisible, setManageAppsModalVisible] = useState(false)
  const [selectedAppIds, setSelectedAppIds] = useState<number[]>([])
  
  // 【新增】应用列表分页状态
  const [appPage, setAppPage] = useState(1)
  const [appPageSize, setAppPageSize] = useState(20)
  
  // 【新增】构建修改状态（app_id -> selected_build_id）
  const [buildChanges, setBuildChanges] = useState<Record<number, number>>({})

  // 【新增】依赖配置状态
  const [dependencyModalVisible, setDependencyModalVisible] = useState(false)
  const [editingRelease, setEditingRelease] = useState<ReleaseApp | null>(null)
  const [tempDependencySelection, setTempDependencySelection] = useState<number[]>([])

  // 查询批次详情（支持分页）
  const {data: batchData, isLoading} = useQuery({
    queryKey: ['batchDetail', id, appPage, appPageSize],
    queryFn: async () => {
      const res = await batchService.get(Number(id), appPage, appPageSize)
      console.log('Batch detail response:', res)
      return res.data as Batch
    },
    enabled: !!id,
  })

  const batch = batchData

  const appIdMap = useMemo(() => {
    const map = new Map<number, ReleaseApp>()
    batch?.apps?.forEach(app => {
      map.set(app.app_id, app)
    })
    return map
  }, [batch?.apps])

  // 查询所有应用（用于管理应用，包含构建信息）
  const {data: allAppsResponse} = useQuery({
    queryKey: ['applicationsWithBuilds'],
    queryFn: async () => {
      const res = await applicationService.searchWithBuilds({page_size: 1000})
      return res.data
    },
    enabled: manageAppsModalVisible, // 只在打开 Modal 时查询
  })

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
    onError: (error: any) => {
      message.error(error.response?.data?.message || t('common.error'))
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
      queryClient.invalidateQueries({queryKey: ['batchDetail', id, appPage, appPageSize]})
      setDependencyModalVisible(false)
      setEditingRelease(null)
      setTempDependencySelection([])
    },
    onError: (error: any) => {
      message.error(error.response?.data?.message || t('common.error'))
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
    onError: (error: any) => {
      message.error(error.response?.data?.message || t('common.error'))
    },
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

  // 打开管理应用 Modal
  const handleManageApps = () => {
    // 初始化已选应用 ID
    const currentAppIds = batch?.apps?.map(app => app.app_id) || []
    setSelectedAppIds(currentAppIds)
    setManageAppsModalVisible(true)
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
      type?: 'primary' | 'default'
      danger?: boolean
      action: BatchActionRequest['action']
    }> = []

    // 已封板：可以开始预发布
    if (batch.status === 10) {
      actions.push({
        key: 'start_pre_deploy',
        label: t('batch.startPreDeploy'),
        icon: <PlayCircleOutlined/>,
        type: 'primary',
        action: 'start_pre_deploy',
      })
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
        action: 'start_prod_deploy',
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

  // 【新增】处理构建选择变更
  const handleBuildChange = (appId: number, buildId: number) => {
    setBuildChanges(prev => ({
      ...prev,
      [appId]: buildId
    }))
  }

  // 【新增】保存构建变更
  const saveBuildChanges = () => {
    // TODO: 调用后端 API 保存构建变更
    message.info(`TODO: 保存构建变更 ${JSON.stringify(buildChanges)}`)
    // 成功后清空变更记录
    // setBuildChanges({})
    // queryClient.invalidateQueries({queryKey: ['batchDetail', id]})
  }

  const appColumns: ColumnsType<ReleaseApp> = [
    {
      title: t('batch.appName'),
      dataIndex: 'app_name',
      key: 'app_name',
      width: 180,
      fixed: 'left',
      ellipsis: true,
    },
    {
      title: t('batch.appType'),
      dataIndex: 'app_type',
      key: 'app_type',
      width: 100,
      render: (type: string) => (
        <Tag color="blue">{type}</Tag>
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
      title: isBatchCompleted ? t('batch.oldVersion') : t('batch.currentVersion'),
      key: isBatchCompleted ? 'old_version' : 'current_version',
      width: 140,
      ellipsis: true,
      render: (_: any, record: ReleaseApp) => (
        isBatchCompleted ? (record.previous_deployed_tag || '-') : (record.deployed_tag || '-')
      ),
    },
    {
      title: isBatchCompleted ? t('batch.deployed') : t('batch.pendingDeploy'),
      key: isBatchCompleted ? 'deployed' : 'pending_deploy',
      width: 200,
      render: (_: any, record: ReleaseApp) => {
        // 如果批次未封板且有 recent_builds，显示下拉选择
        if (!isBatchCompleted && batch && batch.status < 10 && record.recent_builds && record.recent_builds.length > 0) {
          const currentValue = buildChanges[record.app_id] || record.build_id
          return (
            <Select
              style={{ width: '100%' }}
              value={currentValue}
              onChange={(value) => handleBuildChange(record.app_id, value)}
              size="small"
              optionLabelProp="label"
            >
              {record.recent_builds.map((build: BuildSummary) => (
                <Select.Option 
                  key={build.id} 
                  value={build.id}
                  label={build.image_tag}
                >
                  <div style={{fontSize: 12}}>
                    <div><code>{build.image_tag}</code></div>
                    <div style={{color: '#8c8c8c', fontSize: 11}}>
                      {build.commit_message?.substring(0, 40)}
                      {(build.commit_message?.length || 0) > 40 && '...'}
                    </div>
                    <div style={{color: '#8c8c8c', fontSize: 10}}>
                      {dayjs(build.build_created).format('MM-DD HH:mm')}
                    </div>
                  </div>
                </Select.Option>
              ))}
            </Select>
          )
        }
        // 否则显示普通文本
        return <code style={{fontSize: 12}}>{record.target_tag || '-'}</code>
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
    {
      title: t('batch.commitMessage'),
      dataIndex: 'commit_message',
      key: 'commit_message',
      width: 250,
      ellipsis: true,
      render: (text: string) => (
        <span style={{fontSize: 12}}>{text || '-'}</span>
      ),
    },
  ]

  return (
    <div className="batch-detail-container">
      {/* 头部导航 */}
      <div className="batch-detail-header">
        <Button
          icon={<ArrowLeftOutlined/>}
          onClick={() => navigate('/batch')}
        >
          {t('common.back')}
        </Button>
        <h2>{batch.batch_number}</h2>
      </div>

      {/* 批次基本信息 */}
      <Card className="batch-info-card" title={t('batch.batchInfo')}>
        <Descriptions column={2} bordered>
          <Descriptions.Item label={t('batch.batchNumber')} span={2}>
            {batch.batch_number}
          </Descriptions.Item>
          <Descriptions.Item label={t('batch.initiator')}>
            {batch.initiator}
          </Descriptions.Item>
          <Descriptions.Item label={t('batch.createdAt')}>
            {dayjs(batch.created_at).format('YYYY-MM-DD HH:mm:ss')}
          </Descriptions.Item>
          <Descriptions.Item label={t('batch.status')}>
            <StatusTag status={batch.status}/>
          </Descriptions.Item>
          <Descriptions.Item label={t('batch.approvalStatus')}>
            <StatusTag status={0} approvalStatus={batch.approval_status} showApproval/>
            {batch.approved_by && (
              <span style={{marginLeft: 8, color: '#8c8c8c', fontSize: 12}}>
                by {batch.approved_by}
              </span>
            )}
          </Descriptions.Item>
          {batch.release_notes && (
            <Descriptions.Item label={t('batch.releaseNotes')} span={2}>
              <div style={{whiteSpace: 'pre-wrap'}}>{batch.release_notes}</div>
            </Descriptions.Item>
          )}
        </Descriptions>

        {/* 操作按钮 */}
        <div className="batch-actions">
          <Space size="middle">
            {getAvailableActions().map((action) => (
              <Button
                key={action.key}
                type={action.type}
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
              >
                {action.label}
              </Button>
            ))}
          </Space>
        </div>
      </Card>

      {/* 上线流程时间线 */}
      <BatchTimeline batch={batch}/>

      {/* 应用列表 */}
      <Card
        title={`${t('batch.apps')} (${batch.total_apps || batch.apps?.length || 0})`}
        extra={
          <Space>
            {/* 分页器（如果应用数量大于 page_size 才显示） */}
            {batch.total_apps && batch.total_apps > (batch.app_page_size || 20) && (
              <Pagination
                simple
                current={appPage}
                pageSize={appPageSize}
                total={batch.total_apps}
                onChange={(page, pageSize) => {
                  setAppPage(page)
                  setAppPageSize(pageSize || 20)
                }}
                showSizeChanger
                pageSizeOptions={['10', '20', '50']}
                size="small"
              />
            )}
            
            {/* 应用按钮（有构建变更时显示） */}
            {Object.keys(buildChanges).length > 0 && (
              <Button
                type="primary"
                icon={<SaveOutlined/>}
                onClick={saveBuildChanges}
                size="small"
              >
                应用 ({Object.keys(buildChanges).length})
              </Button>
            )}
            
            {/* 管理应用按钮 */}
            {batch.status < 10 && (
              <Button
                icon={<EditOutlined/>}
                onClick={handleManageApps}
                size="small"
              >
                {t('batch.manageApps')}
              </Button>
            )}
          </Space>
        }
      >
        <Table
          key={`batch-table-${batchStatusValue}-${isBatchCompleted ? 'completed' : 'in-progress'}`}
          columns={appColumns}
          dataSource={batch.apps || []}
          rowKey="id"
          pagination={false}
          scroll={{ x: 1200 }}
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
      </Card>

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
        width={700}
      >
        <div style={{marginBottom: 16}}>
          <div style={{marginBottom: 8, fontWeight: 500}}>
            {t('batch.selectApps')} ({selectedAppIds.length} {t('batch.selectedApps')})
          </div>
          <div style={{maxHeight: 400, overflowY: 'auto'}}>
            {allAppsResponse?.items?.map((app: ApplicationWithBuild) => (
              <div
                key={app.id}
                style={{
                  padding: '8px 12px',
                  marginBottom: 4,
                  border: '1px solid #f0f0f0',
                  borderRadius: 4,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                }}
              >
                <Checkbox
                  checked={selectedAppIds.includes(app.id)}
                  onChange={(e) => {
                    if (e.target.checked) {
                      setSelectedAppIds([...selectedAppIds, app.id])
                    } else {
                      setSelectedAppIds(selectedAppIds.filter(id => id !== app.id))
                    }
                  }}
                >
                  <Space>
                    <span style={{fontWeight: 500}}>{app.name}</span>
                    <Tag color="blue">{app.app_type}</Tag>
                    {app.image_tag && (
                      <Tag color="green">{app.image_tag}</Tag>
                    )}
                  </Space>
                </Checkbox>
                <div style={{fontSize: 11, color: '#8c8c8c', textAlign: 'right'}}>
                  <div>{app.repo_name}</div>
                  {app.commit_message && (
                    <div style={{marginTop: 2}}>
                      {app.commit_message.substring(0, 30)}
                      {app.commit_message.length > 30 && '...'}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      </Modal>
    </div>
  )
}

