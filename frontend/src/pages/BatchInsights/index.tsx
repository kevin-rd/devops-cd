import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import {
  Alert,
  Button,
  Card,
  Empty,
  Segmented,
  Skeleton,
  Space,
  Spin,
  Tabs,
  Tag,
} from 'antd'
import { LeftOutlined, ReloadOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useTranslation } from 'react-i18next'
import { batchService } from '@/services/batch'
import type { Batch, ReleaseApp } from '@/types'
import { StatusTag } from '@/components/StatusTag'
import { BatchTimeline } from '@/components/BatchTimeline'
import DependencyGraph from './components/DependencyGraph'
import styles from './index.module.css'

type Environment = 'pre' | 'prod'

const BATCH_PAGE_SIZE = 12

interface BatchOption {
  key: string
  label: string
  batch: Batch
}

const formatTime = (value?: string | null) =>
  value ? dayjs(value).format('YYYY-MM-DD HH:mm') : undefined

const buildTimelineItems = (batch?: Batch) => {
  if (!batch) return []

  const items: Array<{
    title: string
    time?: string
    status: 'finish' | 'process' | 'wait'
  }> = [
    {
      title: 'batch.stage.sealed',
      time: formatTime(batch.tagged_at),
      status: batch.status >= 10 ? 'finish' : 'wait',
    },
    {
      title: 'batch.stage.preDeploy',
      time: formatTime(batch.pre_deploy_started_at),
      status: batch.status >= 21 ? 'finish' : batch.status >= 20 ? 'process' : 'wait',
    },
    {
      title: 'batch.stage.preDeployed',
      time: formatTime(batch.pre_deploy_finished_at),
      status: batch.status >= 22 ? 'finish' : 'wait',
    },
    {
      title: 'batch.stage.prodDeploy',
      time: formatTime(batch.prod_deploy_started_at),
      status: batch.status >= 31 ? 'finish' : batch.status >= 30 ? 'process' : 'wait',
    },
    {
      title: 'batch.stage.prodDeployed',
      time: formatTime(batch.prod_deploy_finished_at),
      status: batch.status >= 32 ? 'finish' : 'wait',
    },
  ]

  return items
}

const environmentOptions: Array<{ label: string; value: Environment }> = [
  { label: 'Pre', value: 'pre' },
  { label: 'Prod', value: 'prod' },
]

export default function BatchInsights() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const params = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()

  const [environment, setEnvironment] = useState<Environment>(
    (searchParams.get('env') as Environment) || 'pre'
  )

  const initialBatchId = Number(params.id)
  const [selectedBatchId, setSelectedBatchId] = useState<number | undefined>(
    Number.isFinite(initialBatchId) ? initialBatchId : undefined
  )

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

              <Card className={styles.section} title={t('batchInsights.timeline')}>
                <div className={styles.timelineWrapper}>
                  <BatchTimeline batch={batchDetail} />
                </div>
                <div className={styles.timelineLegend}>
                  {buildTimelineItems(batchDetail).map((item) => (
                    <div key={item.title} className={styles.timelineItem}>
                      <div className={styles.timelineItemTitle}>{t(item.title)}</div>
                      <div className={styles.timelineItemTime}>{item.time || t('common.notStarted')}</div>
                    </div>
                  ))}
                </div>
              </Card>

              <Card className={styles.graphSection} title={t('batchInsights.dependencyGraph')}>
                <DependencyGraph releaseApps={releaseApps} appTypeConfigs={batchDetail?.app_type_configs} />
              </Card>
            </>
          )}
        </div>
      </Card>
    </div>
  )
}


