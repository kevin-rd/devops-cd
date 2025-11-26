import {useEffect, useMemo, useState} from 'react'
import {useNavigate, useParams, useSearchParams} from 'react-router-dom'
import {Alert, Button, Card, Empty, Input, message, Modal, Radio, Segmented, Skeleton, Space, Spin, Tabs,} from 'antd'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  EditOutlined,
  LeftOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {useTranslation} from 'react-i18next'
import {batchService} from '@/services/batch'
import type {BatchActionRequest} from '@/types'
import {BatchTimeline} from '@/components/BatchTimeline'
import {useAuthStore} from '@/stores/authStore'
import BatchEditDrawer from '@/components/BatchEditDrawer'
import DependencyGraph from './components/DependencyGraph'
import '@/styles/status-theme.css'
import styles from './index.module.css'
import {Batch} from "@/types/batch.ts";

const {TextArea} = Input

type Environment = 'pre' | 'prod'

const BATCH_PAGE_SIZE = 12

interface BatchOption {
  key: string
  label: string
  batch: Batch
}

const environmentOptions: Array<{ label: string; value: Environment }> = [
  {label: 'Pre', value: 'pre'},
  {label: 'Prod', value: 'prod'},
]

export default function BatchInsights() {
  const {t} = useTranslation()
  const navigate = useNavigate()
  const params = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const {user} = useAuthStore()

  // Timeline 卡片标题组件
  const TimelineCardTitle = ({batch}: { batch: Batch }) => (
    <div className={styles.timelineTitle}>
      <div className={styles.titleMain}>
        <span className={styles.batchId}>#{batch.id}</span>
        <span className={styles.batchName}>{batch.batch_number}</span>
      </div>
      {batch.release_notes && (
        <div className={styles.releaseNotes}>{batch.release_notes}</div>
      )}
    </div>
  )

  const [environment, setEnvironment] = useState<Environment>(
    (searchParams.get('env') as Environment) || 'pre'
  )

  const initialBatchId = Number(params.id)
  const [selectedBatchId, setSelectedBatchId] = useState<number | undefined>(
    Number.isFinite(initialBatchId) ? initialBatchId : undefined
  )

  // 取消批次相关状态
  const [cancelModalVisible, setCancelModalVisible] = useState(false)
  const [cancelReason, setCancelReason] = useState('')

  // 审批相关状态
  const [approvalModalVisible, setApprovalModalVisible] = useState(false)
  const [approvalAction, setApprovalAction] = useState<'approve' | 'reject'>('approve')
  const [approvalReason, setApprovalReason] = useState('')
  const [approvalLoading, setApprovalLoading] = useState(false)

  // 编辑批次相关状态
  const [editDrawerOpen, setEditDrawerOpen] = useState(false)
  const [editingBatch, setEditingBatch] = useState<Batch | null>(null)

  const {data: batchTabData, isLoading: isTabsLoading, isError: tabError, refetch: refetchTabs} = useQuery({
    queryKey: ['batch-insights-tabs'],
    queryFn: async () => {
      const res = await batchService.list({page: 1, page_size: BATCH_PAGE_SIZE})
      const raw = res.data as any
      if (Array.isArray(raw)) {
        return raw as Batch[]
      }
      if (raw && Array.isArray(raw.items)) {
        return raw.items as Batch[]
      }
      return [] as Batch[]
    },
    staleTime: 60 * 1000,
  })

  const batchTabs: BatchOption[] = useMemo(() => {
    if (!batchTabData) return []
    return batchTabData.map((batch) => ({
      key: String(batch.id),
      label: `${batch.batch_number}`,
      batch,
    }))
  }, [batchTabData])

  useEffect(() => {
    if (!selectedBatchId && batchTabs.length > 0) {
      const firstId = Number(batchTabs[0].key)
      setSelectedBatchId(firstId)
      const query = searchParams.toString()
      navigate(`/batch/${firstId}/insights${query ? `?${query}` : ''}`, {replace: true})
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [batchTabs])

  const currentBatchId = useMemo(() => {
    if (selectedBatchId) return selectedBatchId
    if (batchTabs.length > 0) return Number(batchTabs[0].key)
    return undefined
  }, [selectedBatchId, batchTabs])

  const {
    data: batchDetail,
    isLoading: isDetailLoading,
    isError: detailError,
    refetch: refetchDetail,
  } = useQuery({
    queryKey: ['batch-insights-detail', currentBatchId],
    queryFn: async () => {
      if (!currentBatchId) return undefined
      const res = await batchService.get(currentBatchId, 1, 200)
      return res.data as Batch
    },
    enabled: Boolean(currentBatchId),
    staleTime: 30 * 1000,
  })

  // 检查当前批次是否处于部署中状态（用于决定是否启用状态轮询）
  const isDeploying = useMemo(() => {
    if (!batchDetail) return false
    return (
      batchDetail.status === 20 || // 预发布待触发
      batchDetail.status === 21 || // 预发布中
      batchDetail.status === 30 || // 生产部署待触发
      batchDetail.status === 31    // 生产部署中
    )
  }, [batchDetail])

  const isNotFinished = useMemo(() => {
    if (!batchDetail) return false
    return batchDetail.status < 40
  }, [batchDetail])


  // 轻量级状态轮询（仅用于部署中的批次，不设置loading状态）
  // 注意：初次加载时不调用，因为 batch 详情接口已包含状态信息
  // 只在轮询周期（每5秒）时才调用，用于更新状态
  const {data: batchStatus} = useQuery({
    queryKey: ['batch-insights-status', currentBatchId],
    queryFn: async () => {
      if (!currentBatchId) return undefined
      const res = await batchService.getStatus(currentBatchId, 1, 200)
      return res.data as Batch
    },
    // 只有未完成的批次才启用查询
    enabled: isNotFinished && Boolean(currentBatchId),
    // 避免初次挂载时立即执行，只依赖轮询间隔
    refetchOnMount: false,
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
    refetchInterval: () => {
      const interval = isDeploying ? 5_000 : batchDetail?.status === 40 ? 10_000 : 30_000;
      return document.hidden ? interval * 10 : interval;
    },
    refetchIntervalInBackground: false,
    staleTime: 5000,
  })

  // 合并详情数据和状态数据（优先使用状态数据）
  const mergedBatchDetail = useMemo(() => {
    if (!batchDetail) return undefined
    if (!batchStatus) return batchDetail

    // 合并应用数据：保留详情接口的完整数据，只更新状态字段
    let mergedApps = batchDetail.apps
    if (batchStatus.apps && batchStatus.apps.length > 0 && batchDetail.apps && batchDetail.apps.length > 0) {
      // 创建状态映射表
      const statusMap = new Map(
        batchStatus.apps.map((app: any) => [app.app_id, {status: app.status, is_locked: app.is_locked}])
      )

      // 更新详情数据中的状态字段
      mergedApps = batchDetail.apps.map((app) => {
        const statusUpdate = statusMap.get(app.app_id)
        if (statusUpdate) {
          return {
            ...app,
            status: statusUpdate.status,
            is_locked: statusUpdate.is_locked,
          }
        }
        return app
      })
    }

    // 状态数据优先级更高（更新更及时）
    return {
      ...batchDetail,
      status: batchStatus.status ?? batchDetail.status,
      approval_status: batchStatus.approval_status ?? batchDetail.approval_status,
      tagged_at: (batchStatus as any).sealed_at ?? batchDetail.tagged_at,
      pre_deploy_started_at: batchStatus.pre_deploy_started_at ?? batchDetail.pre_deploy_started_at,
      pre_deploy_finished_at: batchStatus.pre_deploy_finished_at ?? batchDetail.pre_deploy_finished_at,
      prod_deploy_started_at: batchStatus.prod_deploy_started_at ?? batchDetail.prod_deploy_started_at,
      prod_deploy_finished_at: batchStatus.prod_deploy_finished_at ?? batchDetail.prod_deploy_finished_at,
      final_accepted_at: batchStatus.final_accepted_at ?? batchDetail.final_accepted_at,
      cancelled_at: batchStatus.cancelled_at ?? batchDetail.cancelled_at,
      updated_at: batchStatus.updated_at ?? batchDetail.updated_at,
      apps: mergedApps, // 使用合并后的应用数据
    }
  }, [batchDetail, batchStatus])

  const releaseApps = mergedBatchDetail?.apps || []

  // 批次操作 Mutation
  const actionMutation = useMutation({
    mutationFn: (data: BatchActionRequest) => batchService.action(data),
    onSuccess: () => {
      message.success(t('batch.actionSuccess'))
      queryClient.invalidateQueries({queryKey: ['batch-insights-tabs']})
      queryClient.invalidateQueries({queryKey: ['batch-insights-detail']})
      refetchTabs()
      refetchDetail()
    },
    onError: (error: any) => {
      message.error(error.response?.data?.message || t('common.error'))
    },
  })

  // 处理批次操作
  const handleAction = (batchId: number, action: string) => {
    if (action === 'cancel') {
      setCancelModalVisible(true)
    } else {
      Modal.confirm({
        title: t('common.confirm'),
        content: t(`batch.confirm${action.charAt(0).toUpperCase() + action.slice(1)}`),
        onOk: () => {
          actionMutation.mutate({
            batch_id: batchId,
            action: action as any,
            operator: user?.username || 'unknown',
          })
        },
      })
    }
  }

  // 确认取消
  const confirmCancel = () => {
    if (!cancelReason.trim()) {
      message.warning('请输入取消原因')
      return
    }
    if (currentBatchId) {
      actionMutation.mutate({
        batch_id: currentBatchId,
        action: 'cancel',
        operator: user?.username || 'unknown',
        reason: cancelReason,
      })
      setCancelModalVisible(false)
      setCancelReason('')
    }
  }

  // 打开编辑抽屉
  const handleEdit = () => {
    if (mergedBatchDetail) {
      setEditingBatch(mergedBatchDetail)
      setEditDrawerOpen(true)
    }
  }

  // 从 timeline 触发的操作处理
  const handleTimelineAction = (action: string) => {
    if (!currentBatchId) return
    handleAction(currentBatchId, action)
  }

  // 确认审批
  const handleConfirmApproval = async () => {
    if (approvalAction === 'reject' && !approvalReason.trim()) {
      message.warning('请输入拒绝原因')
      return
    }

    if (!currentBatchId) return

    setApprovalLoading(true)
    try {
      if (approvalAction === 'approve') {
        await batchService.approve({
          batch_id: currentBatchId,
          operator: user?.username || 'unknown',
        })
        message.success(t('batch.approveSuccess'))
      } else {
        await batchService.reject({
          batch_id: currentBatchId,
          operator: user?.username || 'unknown',
          reason: approvalReason,
        })
        message.success(t('batch.rejectSuccess'))
      }

      queryClient.invalidateQueries({queryKey: ['batch-insights-tabs']})
      queryClient.invalidateQueries({queryKey: ['batch-insights-detail']})
      refetchTabs()
      refetchDetail()

      setApprovalModalVisible(false)
      setApprovalReason('')
    } catch (error: any) {
      message.error(error.response?.data?.message || t('common.error'))
    } finally {
      setApprovalLoading(false)
    }
  }

  // 渲染操作按钮
  const renderActionButtons = (batch: Batch) => {
    // 如果批次已取消，不显示任何操作按钮
    if (batch.status === 90) {
      return null
    }

    const actions = []

    // 编辑按钮（草稿或待审批状态）
    if (batch.status === 0 || batch.approval_status === 'pending') {
      actions.push(
        <Button
          key="edit"
          size="small"
          icon={<EditOutlined/>}
          onClick={handleEdit}
        >
          {t('common.edit')}
        </Button>
      )
    }

    // 封板按钮（草稿状态且审批已通过）
    if (batch.status === 0 && batch.approval_status === 'approved') {
      actions.push(
        <Button
          key="seal"
          size="small"
          icon={<CheckCircleOutlined/>}
          onClick={() => handleAction(batch.id, 'seal')}
        >
          {t('batch.seal')}
        </Button>
      )
    }

    // 开始预发布按钮（已封板状态）
    if (batch.status === 10) {
      actions.push(
        <Button
          key="start_pre_deploy"
          size="small"
          icon={<PlayCircleOutlined/>}
          onClick={() => handleAction(batch.id, 'start_pre_deploy')}
        >
          {t('batch.startPreDeploy')}
        </Button>
      )
    }

    // 开始生产部署按钮（预发布完成状态）
    if (batch.status === 22) {
      actions.push(
        <Button
          key="start_prod_deploy"
          size="small"
          icon={<PlayCircleOutlined/>}
          onClick={() => handleAction(batch.id, 'start_prod_deploy')}
        >
          {t('batch.startProdDeploy')}
        </Button>
      )
    }

    // 生产验收按钮（生产部署完成状态）
    if (batch.status === 32) {
      actions.push(
        <Button
          key="prod_acceptance"
          type="primary"
          size="small"
          icon={<CheckCircleOutlined/>}
          onClick={() => handleAction(batch.id, 'prod_acceptance')}
        >
          {t('batch.prodAcceptance')}
        </Button>
      )
    }

    // 取消批次按钮（未完成且未取消的批次）
    if (batch.status < 40 && batch.status !== 90) {
      actions.push(
        <Button
          key="cancel"
          type="link"
          size="small"
          danger
          onClick={() => handleAction(batch.id, 'cancel')}
        >
          {t('batch.cancelBatch')}
        </Button>
      )
    }

    return <Space size="small">{actions}</Space>
  }

  const handleTabChange = (key: string) => {
    const id = Number(key)
    setSelectedBatchId(id)
    const query = searchParams.toString()
    navigate(`/batch/${id}/insights${query ? `?${query}` : ''}`, {replace: true})
  }

  const handleEnvironmentChange = (value: Environment) => {
    setEnvironment(value)
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('env', value)
      return next
    })
    if (currentBatchId) {
      const query = searchParams.toString()
      const nextQuery = new URLSearchParams(query)
      nextQuery.set('env', value)
      navigate(`/batch/${currentBatchId}/insights?${nextQuery.toString()}`, {replace: true})
    }
  }

  const onBack = () => {
    navigate('/batch')
  }

  // 根据批次状态获取 timeline 卡片的样式类
  const getTimelineCardClass = (batch: Batch) => {
    const status = batch.status

    // 草稿 (0)
    if (status === 0) {
      return styles.timelineDraft
    }
    // 已封板 (10)
    if (status === 10) {
      return styles.timelineSealed
    }
    // 预发布已触发(20) 预发布中(21)
    if (status === 20 || status === 21) {
      return styles.timelinePreDeploying
    }
    // 预发布完成(22)
    if (status === 22) {
      return styles.timelinePreDeployed
    }
    // 预发布失败(23) - 可根据实际状态码调整
    if (status === 23) {
      return styles.timelinePreFailed
    }
    // 生产已触发(30) 生产部署中(31)
    if (status === 30 || status === 31) {
      return styles.timelineProdDeploying
    }
    // 生产部署完成(32)
    if (status === 32) {
      return styles.timelineProdDeployed
    }
    // 生产部署失败(33) - 可根据实际状态码调整
    if (status === 33) {
      return styles.timelineProdFailed
    }
    // 已完成 (40)
    if (status === 40) {
      return styles.timelineCompleted
    }
    // 已取消 (90)
    if (status === 90) {
      return styles.timelineCancelled
    }

    return ''
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <Button icon={<LeftOutlined/>} onClick={onBack} type="link">
          {t('batchInsights.back')}
        </Button>
        <div className={styles.headerControls}>
          <Segmented
            options={environmentOptions.map((option) => ({
              value: option.value,
              label: t(`batchInsights.env.${option.value}`),
            }))}
            value={environment}
            onChange={(value) => handleEnvironmentChange(value as Environment)}
          />
          <Button icon={<ReloadOutlined/>} onClick={() => {
            refetchTabs()
            refetchDetail()
          }}
          >
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      <Card className={styles.card}>
        {tabError ? (
          <Alert
            type="error"
            message={t('batchInsights.loadTabsFailed')}
            showIcon
            action={<Button size="small" onClick={() => refetchTabs()}>{t('common.retry')}</Button>}
          />
        ) : (
          <Tabs
            items={batchTabs}
            activeKey={currentBatchId ? String(currentBatchId) : undefined}
            onChange={handleTabChange}
            tabBarExtraContent={
              isTabsLoading ? <Spin size="small"/> : undefined
            }
          />
        )}

        <div className={styles.content}>
          {isDetailLoading ? (
            <Skeleton active paragraph={{rows: 6}}/>
          ) : detailError ? (
            <Alert
              type="error"
              message={t('batchInsights.loadDetailFailed')}
              showIcon
              action={<Button size="small" onClick={() => refetchDetail()}>{t('common.retry')}</Button>}
            />
          ) : !mergedBatchDetail ? (
            <Empty description={t('batchInsights.noData')}/>
          ) : (
            <>
              <Card
                className={`${styles.section} ${getTimelineCardClass(mergedBatchDetail)}`}
                title={<TimelineCardTitle batch={mergedBatchDetail}/>}
                extra={renderActionButtons(mergedBatchDetail)}
              >
                <BatchTimeline 
                  batch={mergedBatchDetail} 
                  hasPreApps={releaseApps.some(app => !app.skip_pre_env)}
                  onAction={handleTimelineAction}
                />
              </Card>

              <Card className={styles.graphSection}
                    title={`${t('batchInsights.appDetails')} (${mergedBatchDetail.total_apps || mergedBatchDetail.apps?.length || 0})`}>
                <DependencyGraph
                  releaseApps={releaseApps}
                  appTypeConfigs={mergedBatchDetail?.app_type_configs}
                  environment={environment}
                  batch={mergedBatchDetail}
                  onRefresh={() => {
                    refetchTabs()
                    refetchDetail()
                  }}
                />
              </Card>
            </>
          )}
        </div>
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

      {/* 审批批次 Modal */}
      <Modal
        title="批次审批"
        open={approvalModalVisible}
        onOk={handleConfirmApproval}
        onCancel={() => {
          setApprovalModalVisible(false)
          setApprovalReason('')
        }}
        confirmLoading={approvalLoading}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        width={500}
      >
        <Space direction="vertical" style={{width: '100%'}} size="large">
          <div>
            <div style={{marginBottom: 12, fontWeight: 500}}>请选择审批结果：</div>
            <Radio.Group
              value={approvalAction}
              onChange={(e) => setApprovalAction(e.target.value)}
              style={{width: '100%'}}
            >
              <Space direction="vertical" style={{width: '100%'}}>
                <Radio value="approve">
                  <Space>
                    <CheckCircleOutlined style={{color: '#52c41a'}}/>
                    <span>审批通过</span>
                  </Space>
                </Radio>
                <Radio value="reject">
                  <Space>
                    <CloseCircleOutlined style={{color: '#ff4d4f'}}/>
                    <span>审批拒绝</span>
                  </Space>
                </Radio>
              </Space>
            </Radio.Group>
          </div>

          {approvalAction === 'reject' && (
            <div>
              <div style={{marginBottom: 8, fontWeight: 500}}>
                拒绝原因 <span style={{color: '#ff4d4f'}}>*</span>：
              </div>
              <TextArea
                rows={4}
                placeholder="请输入拒绝原因..."
                value={approvalReason}
                onChange={(e) => setApprovalReason(e.target.value)}
                maxLength={500}
                showCount
              />
            </div>
          )}
        </Space>
      </Modal>

      {/* 修改批次 Drawer */}
      <BatchEditDrawer
        open={editDrawerOpen}
        batch={editingBatch}
        onClose={() => {
          setEditDrawerOpen(false)
          setEditingBatch(null)
        }}
        onSuccess={() => {
          setEditDrawerOpen(false)
          setEditingBatch(null)
          refetchTabs()
          refetchDetail()
        }}
      />
    </div>
  )
}


