import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import {
  Alert,
  Button,
  Card,
  Empty,
  Input,
  message,
  Modal,
  Segmented,
  Skeleton,
  Space,
  Spin,
  Tabs,
} from 'antd'
import { 
  LeftOutlined, 
  ReloadOutlined,
  EditOutlined,
  CheckCircleOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useTranslation } from 'react-i18next'
import { batchService } from '@/services/batch'
import type { Batch, BatchActionRequest } from '@/types'
import { StatusTag } from '@/components/StatusTag'
import { BatchTimeline } from '@/components/BatchTimeline'
import { useAuthStore } from '@/stores/authStore'
import BatchEditDrawer from '@/components/BatchEditDrawer'
import DependencyGraph from './components/DependencyGraph'
import '@/styles/status-theme.css'
import styles from './index.module.css'

const { TextArea } = Input

type Environment = 'pre' | 'prod'

const BATCH_PAGE_SIZE = 12

interface BatchOption {
  key: string
  label: string
  batch: Batch
}

const formatTime = (value?: string | null) =>
  value ? dayjs(value).format('YYYY-MM-DD HH:mm') : undefined

const environmentOptions: Array<{ label: string; value: Environment }> = [
  { label: 'Pre', value: 'pre' },
  { label: 'Prod', value: 'prod' },
]

export default function BatchInsights() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const params = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const { user } = useAuthStore()

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

  // 编辑批次相关状态
  const [editDrawerOpen, setEditDrawerOpen] = useState(false)
  const [editingBatch, setEditingBatch] = useState<Batch | null>(null)

  const { data: batchTabData, isLoading: isTabsLoading, isError: tabError, refetch: refetchTabs } = useQuery({
    queryKey: ['batch-insights-tabs'],
    queryFn: async () => {
      const res = await batchService.list({ page: 1, page_size: BATCH_PAGE_SIZE })
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
      navigate(`/batch/${firstId}/insights${query ? `?${query}` : ''}`, { replace: true })
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

  const releaseApps = batchDetail?.apps || []

  // 批次操作 Mutation
  const actionMutation = useMutation({
    mutationFn: (data: BatchActionRequest) => batchService.action(data),
    onSuccess: () => {
      message.success(t('batch.actionSuccess'))
      queryClient.invalidateQueries({ queryKey: ['batch-insights-tabs'] })
      queryClient.invalidateQueries({ queryKey: ['batch-insights-detail'] })
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
    if (batchDetail) {
      setEditingBatch(batchDetail)
      setEditDrawerOpen(true)
    }
  }

  // 从 timeline 触发的操作处理
  const handleTimelineAction = (action: string) => {
    if (!currentBatchId) return
    handleAction(currentBatchId, action)
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
          icon={<EditOutlined />}
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
          icon={<CheckCircleOutlined />}
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
          icon={<PlayCircleOutlined />}
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
          icon={<PlayCircleOutlined />}
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
          icon={<CheckCircleOutlined />}
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
    navigate(`/batch/${id}/insights${query ? `?${query}` : ''}`, { replace: true })
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
      navigate(`/batch/${currentBatchId}/insights?${nextQuery.toString()}`, { replace: true })
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
    // 预发布等待 (20)
    if (status === 20) {
      return styles.timelinePreWaiting
    }
    // 预发布中 (21)
    if (status === 21) {
      return styles.timelinePreDeploying
    }
    // 预发布完成 (22)
    if (status === 22) {
      return styles.timelinePreDeployed
    }
    // 预发布失败 (23) - 可根据实际状态码调整
    if (status === 23) {
      return styles.timelinePreFailed
    }
    // 生产等待 (30)
    if (status === 30) {
      return styles.timelineProdWaiting
    }
    // 生产部署中 (31)
    if (status === 31) {
      return styles.timelineProdDeploying
    }
    // 生产部署完成 (32)
    if (status === 32) {
      return styles.timelineProdDeployed
    }
    // 生产部署失败 (33) - 可根据实际状态码调整
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
        <Button icon={<LeftOutlined />} onClick={onBack} type="link">
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
          <Button icon={<ReloadOutlined />} onClick={() => {
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
              isTabsLoading ? <Spin size="small" /> : undefined
            }
          />
        )}

        <div className={styles.content}>
          {isDetailLoading ? (
            <Skeleton active paragraph={{ rows: 6 }} />
          ) : detailError ? (
            <Alert
              type="error"
              message={t('batchInsights.loadDetailFailed')}
              showIcon
              action={<Button size="small" onClick={() => refetchDetail()}>{t('common.retry')}</Button>}
            />
          ) : !batchDetail ? (
            <Empty description={t('batchInsights.noData')} />
          ) : (
            <>
              <Card className={styles.section} title={t('batchInsights.batchInfo')}>
                <div className={styles.batchInfoGrid}>
                  <div>
                    <div className={styles.label}>{t('batch.batchNumber')}</div>
                    <div className={styles.value}>{batchDetail.batch_number}</div>
                  </div>
                  <div>
                    <div className={styles.label}>{t('batch.initiator')}</div>
                    <div className={styles.value}>{batchDetail.initiator || '-'}</div>
                  </div>
                  <div>
                    <div className={styles.label}>{t('batch.status')}</div>
                    <StatusTag status={batchDetail.status} />
                  </div>
                  <div>
                    <div className={styles.label}>{t('batch.approvalStatus')}</div>
                    <StatusTag status={batchDetail.status} approvalStatus={batchDetail.approval_status} showApproval />
                  </div>
                  <div>
                    <div className={styles.label}>{t('batch.createdAt')}</div>
                    <div className={styles.value}>{formatTime(batchDetail.created_at) || '-'}</div>
                  </div>
                  <div>
                    <div className={styles.label}>{t('batchInsights.appCount')}</div>
                    <div className={styles.value}>{batchDetail.total_apps || batchDetail.apps?.length || 0}</div>
                  </div>
                </div>
              </Card>

              <Card 
                className={`${styles.section} ${getTimelineCardClass(batchDetail)}`}
                title={t('batchInsights.timeline')}
                extra={renderActionButtons(batchDetail)}
              >
                <BatchTimeline batch={batchDetail} onAction={handleTimelineAction} />
              </Card>

              <Card className={styles.graphSection} title={t('batchInsights.dependencyGraph')}>
                <DependencyGraph releaseApps={releaseApps} appTypeConfigs={batchDetail?.app_type_configs} />
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
        <div style={{ marginBottom: 16 }}>{t('batch.cancelConfirm')}</div>
        <TextArea
          rows={4}
          placeholder="请输入取消原因..."
          value={cancelReason}
          onChange={(e) => setCancelReason(e.target.value)}
        />
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


