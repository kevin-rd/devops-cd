import { useState, useEffect } from 'react'
import {
  Table,
  Card,
  Button,
  Input,
  Select,
  Space,
  Tag,
  message,
  DatePicker,
  Modal,
  Pagination,
  Spin,
  Tooltip,
} from 'antd'
import {
  PlusOutlined,
  ReloadOutlined,
  EditOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  PlayCircleOutlined,
  SaveOutlined,
  UndoOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'
import type { ColumnsType } from 'antd/es/table'
import { batchService } from '@/services/batch'
import { useAuthStore } from '@/stores/authStore'
import { StatusTag } from '@/components/StatusTag'
import { BatchTimeline } from '@/components/BatchTimeline'
import BatchCreateDrawer from '@/components/BatchCreateDrawer'
import BatchEditDrawer from '@/components/BatchEditDrawer'
import type { Batch, BatchQueryParams, ReleaseApp, BatchActionRequest, BuildSummary } from '@/types'
import './index.css'

const { RangePicker } = DatePicker
const { TextArea } = Input

export default function BatchList() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { user } = useAuthStore()
  const [params, setParams] = useState<BatchQueryParams>({
    page: 1,
    page_size: 20,
    start_time: dayjs().subtract(30, 'day').startOf('day').toISOString(),
    end_time: dayjs().endOf('day').toISOString(),
  })
  const [expandedRowKeys, setExpandedRowKeys] = useState<number[]>([])
  const [cancelModalVisible, setCancelModalVisible] = useState(false)
  const [cancelReason, setCancelReason] = useState('')
  const [currentBatchId, setCurrentBatchId] = useState<number | null>(null)
  const [createDrawerOpen, setCreateDrawerOpen] = useState(false)
  const [editDrawerOpen, setEditDrawerOpen] = useState(false)
  const [editingBatch, setEditingBatch] = useState<Batch | null>(null)
  const [refreshingList, setRefreshingList] = useState(false)
  const [refreshingDetails, setRefreshingDetails] = useState(false)
  
  // 【新增】每个批次的应用列表分页状态 {batchId: {page, pageSize}}
  const [batchAppPagination, setBatchAppPagination] = useState<Record<number, {page: number, pageSize: number}>>({})
  
  // 【新增】每个批次的构建修改状态 {batchId: {appId: buildId}}
  const [buildChanges, setBuildChanges] = useState<Record<number, Record<number, number>>>({})

  // 防抖输入框的本地状态
  const [initiatorInput, setInitiatorInput] = useState<string>(params.initiator || '')
  const [keywordInput, setKeywordInput] = useState<string>(params.keyword || '')

  // 发起人防抖效果
  useEffect(() => {
    const timer = setTimeout(() => {
      setParams((prev) => ({ ...prev, initiator: initiatorInput || undefined, page: 1 }))
    }, 500)
    return () => clearTimeout(timer)
  }, [initiatorInput])

  // 关键词防抖效果
  useEffect(() => {
    const timer = setTimeout(() => {
      setParams((prev) => ({ ...prev, keyword: keywordInput || undefined, page: 1 }))
    }, 500)
    return () => clearTimeout(timer)
  }, [keywordInput])

  // 查询批次列表 - 如果有待部署中的批次，每5秒自动刷新
  const { data: batchResponse, isLoading, isFetching, refetch } = useQuery({
    queryKey: ['batchList', params],
    queryFn: async () => {
      const res = await batchService.list(params)
      // 后端返回格式: { code: 200, message: "success", data: [...], total: 2, page: 1, size: 20 }
      // res 本身就是这个对象（经过 axios 拦截器处理）
      console.log('Batch list response:', res)

      // data 是数组，total/page/size 在根级别
      return {
        items: Array.isArray(res.data) ? res.data : [],
        total: (res as any).total || 0,
        page: (res as any).page || 1,
        page_size: (res as any).size || (res as any).page_size || 20,
      }
    },
    placeholderData: (previousData) => previousData,
    refetchInterval: (query) => {
      // 检查查询结果中是否有部署中的批次
      const data = query.state.data
      if (!data) return false

      const batches = data.items || []
      const hasDeployingBatches = batches.some(
        (batch: Batch) =>
          batch.status === 20 || // 预发布待触发
          batch.status === 21 || // 预发布中
          batch.status === 30 || // 生产部署待触发
          batch.status === 31    // 生产部署中
      )

      return hasDeployingBatches ? 5000 : false // 如果有部署中的批次，每5秒刷新一次
    },
    refetchIntervalInBackground: true, // 即使页面不在焦点也继续轮询
  })

  const batches = batchResponse?.items || []
  const total = batchResponse?.total || 0

  // 查询批次详情（用于展开行）
  const { data: batchDetailsMap = {}, refetch: refetchDetails, isFetching: isFetchingDetails } = useQuery({
    queryKey: ['batchDetails', expandedRowKeys, batchAppPagination],
    queryFn: async () => {
      const detailsMap: Record<number, Batch> = {}
      await Promise.all(
        expandedRowKeys.map(async (id) => {
          const pagination = batchAppPagination[id] || { page: 1, pageSize: 20 }
          const res = await batchService.get(id, pagination.page, pagination.pageSize)
          detailsMap[id] = res.data as Batch
        })
      )
      return detailsMap
    },
    enabled: expandedRowKeys.length > 0,
    staleTime: 0, // 立即标记为过期，确保每次展开都重新获取
    // 如果展开的批次是部署中的，也定时刷新详情
    refetchInterval: (query) => {
      const batchIds = query.queryKey[1] as number[]
      if (batchIds.length === 0) return false

      // 从查询结果中获取批次详情，检查是否处于部署中状态
      const detailsMap = query.state.data as Record<number, Batch> | undefined
      if (!detailsMap) return false

      // 检查展开的批次是否处于部署中状态
      const hasDeployingBatch = batchIds.some((id) => {
        const batch = detailsMap[id]
        return batch && (
          batch.status === 20 ||
          batch.status === 21 ||
          batch.status === 30 ||
          batch.status === 31
        )
      })

      return hasDeployingBatch ? 5000 : false // 每5秒刷新一次
    },
    refetchIntervalInBackground: true,
  })

  // 当获取到详情数据后，同步更新列表缓存中的数据
  useEffect(() => {
    if (batchResponse && Object.keys(batchDetailsMap).length > 0) {
      // 更新列表缓存中的数据
      queryClient.setQueryData(['batchList', params], (oldData: any) => {
        if (!oldData) return oldData

        const updatedItems = oldData.items.map((item: Batch) => {
          const detail = batchDetailsMap[item.id]
          if (detail) {
            // 合并详情数据到列表项，优先使用详情数据
            return {
              ...item,
              ...detail,
              // 确保关键字段使用最新的值
              status: detail.status ?? item.status,
              approval_status: detail.approval_status ?? item.approval_status,
            }
          }
          return item
        })

        return {
          ...oldData,
          items: updatedItems,
        }
      })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [batchDetailsMap])

  // 批次操作 Mutation
  const actionMutation = useMutation({
    mutationFn: (data: BatchActionRequest) => batchService.action(data),
    onSuccess: () => {
      message.success(t('batch.actionSuccess'))
      // 使用部分匹配来刷新所有相关查询
      queryClient.invalidateQueries({ queryKey: ['batchList'] })
      queryClient.invalidateQueries({ queryKey: ['batchDetails'] })
      // 手动触发重新获取
      refetch()
    },
    onError: (error: any) => {
      message.error(error.response?.data?.message || t('common.error'))
    },
  })

  // 处理展开/折叠
  const handleExpand = (expanded: boolean, record: Batch) => {
    if (expanded) {
      setExpandedRowKeys([record.id]) // 只展开一个
      // 注意：不需要手动调用 invalidateQueries，因为 expandedRowKeys 变化会自动触发 useQuery
      // staleTime: 0 确保了每次展开都会重新获取最新数据
    } else {
      setExpandedRowKeys([])
    }
  }

  // 处理批次操作
  const handleAction = (batchId: number, action: string) => {
    setCurrentBatchId(batchId)
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
      setCurrentBatchId(null)
    }
  }

  // 处理刷新按钮
  const handleRefresh = async () => {
    if (expandedRowKeys.length > 0) {
      setRefreshingDetails(true)
      try {
        const detailResult = await refetchDetails()
        if (detailResult.error) {
          throw detailResult.error
        }
      } catch (error: any) {
        console.error('刷新批次详情失败:', error)
        message.error('刷新批次详情失败，请稍后重试')
      }

      try {
        const listResult = await refetch()
        if (listResult.error) {
          throw listResult.error
        }
      } catch (error: any) {
        console.error('刷新批次列表失败:', error)
        message.error('刷新批次列表失败，请稍后重试')
      } finally {
        setRefreshingDetails(false)
      }

      return
    }

    try {
      setRefreshingList(true)
      const listResult = await refetch()
      if (listResult.error) {
        throw listResult.error
      }
    } catch (error: any) {
      console.error('刷新批次列表失败:', error)
      message.error('刷新批次列表失败，请稍后重试')
    } finally {
      setRefreshingList(false)
    }
  }

  const renderActionButtons = (record: Batch) => (
    <Space size="small" wrap>
      {(record.status === 0 || record.approval_status === 'pending') && (
        <Button
          size="small"
          icon={<EditOutlined />}
          onClick={(e) => {
            e.stopPropagation()
            // 优先使用已展开的详情数据，否则使用列表数据
            const batchToEdit = batchDetailsMap[record.id] || record
            setEditingBatch(batchToEdit)
            setEditDrawerOpen(true)
          }}
        >
          {t('common.edit')}
        </Button>
      )}

      {record.status === 0 && (
        <Button
          size="small"
          icon={<CheckCircleOutlined />}
          onClick={(e) => {
            e.stopPropagation()
            handleAction(record.id, 'seal')
          }}
        >
          {t('batch.seal')}
        </Button>
      )}

      {record.approval_status === 'pending' && (
        <>
          <Button
            size="small"
            type="primary"
            icon={<CheckCircleOutlined />}
            onClick={(e) => {
              e.stopPropagation()
              Modal.confirm({
                title: t('batch.approve'),
                content: t('batch.approveConfirm'),
                onOk: async () => {
                  try {
                    await batchService.approve({
                      batch_id: record.id,
                      operator: user?.username || 'unknown',
                    })
                    message.success(t('batch.approveSuccess'))
                    await queryClient.invalidateQueries({ queryKey: ['batchList'] })
                    await queryClient.invalidateQueries({ queryKey: ['batchDetails'] })
                    refetch()
                  } catch (error: any) {
                    message.error(error.response?.data?.message || t('common.error'))
                  }
                },
              })
            }}
          >
            {t('batch.approve')}
          </Button>
          <Button
            size="small"
            danger
            icon={<CloseCircleOutlined />}
            onClick={(e) => {
              e.stopPropagation()
              Modal.confirm({
                title: t('batch.reject'),
                content: (
                  <div>
                    <div style={{ marginBottom: 8 }}>{t('batch.rejectConfirm')}</div>
                    <Input.TextArea
                      id="reject-reason"
                      rows={3}
                      placeholder={t('batch.rejectReasonPlaceholder')}
                    />
                  </div>
                ),
                onOk: async () => {
                  const reason = (document.getElementById('reject-reason') as HTMLTextAreaElement)?.value
                  if (!reason?.trim()) {
                    message.warning(t('batch.rejectReasonRequired'))
                    return Promise.reject()
                  }
                  try {
                    await batchService.reject({
                      batch_id: record.id,
                      operator: user?.username || 'unknown',
                      reason,
                    })
                    message.success(t('batch.rejectSuccess'))
                    await queryClient.invalidateQueries({ queryKey: ['batchList'] })
                    await queryClient.invalidateQueries({ queryKey: ['batchDetails'] })
                    refetch()
                  } catch (error: any) {
                    message.error(error.response?.data?.message || t('common.error'))
                  }
                },
              })
            }}
          >
            {t('batch.reject')}
          </Button>
        </>
      )}

      {record.status === 10 && (
        <Button
          size="small"
          icon={<PlayCircleOutlined />}
          onClick={(e) => {
            e.stopPropagation()
            handleAction(record.id, 'start_pre_deploy')
          }}
        >
          {t('batch.startPreDeploy')}
        </Button>
      )}
      {record.status === 22 && (
        <Button
          size="small"
          icon={<PlayCircleOutlined />}
          onClick={(e) => {
            e.stopPropagation()
            handleAction(record.id, 'start_prod_deploy')
          }}
        >
          {t('batch.startProdDeploy')}
        </Button>
      )}
      {record.status === 32 && (
        <Button
          type="primary"
          size="small"
          icon={<CheckCircleOutlined />}
          onClick={(e) => {
            e.stopPropagation()
            handleAction(record.id, 'prod_acceptance')
          }}
        >
          {t('batch.prodAcceptance')}
        </Button>
      )}
      <Button
        type="link"
        size="small"
        danger
        onClick={(e) => {
          e.stopPropagation()
          handleAction(record.id, 'cancel')
        }}
        disabled={record.status >= 40 || record.status === 90}
      >
        {t('batch.cancelBatch')}
      </Button>
    </Space>
  )

  const handleCardToggle = (record: Batch) => {
    const isExpanded = expandedRowKeys.includes(record.id)
    handleExpand(!isExpanded, record)
  }

  const renderBatchDetail = (detail?: Batch, isLoading?: boolean) => {
    if (!detail) {
      return (
        <div className="batch-card-detail">
          <div style={{
            padding: 24,
            textAlign: 'center',
            color: '#8c8c8c',
            minHeight: '200px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center'
          }}>
            <Spin />
          </div>
        </div>
      )
    }

    const batchId = detail.id
    const batchStatusValue = Number(detail.status)
    const isBatchCompleted = batchStatusValue === 40
    const canEditBuilds = batchStatusValue < 10 // 只有草稿状态可以修改

    // 获取当前批次的分页状态
    const pagination = batchAppPagination[batchId] || { page: 1, pageSize: 20 }
    
    // 获取当前批次的构建修改状态
    const currentBuildChanges = buildChanges[batchId] || {}
    
    // 处理构建选择变更
    const handleBuildChange = (appId: number, buildId: number) => {
      setBuildChanges(prev => ({
        ...prev,
        [batchId]: {
          ...(prev[batchId] || {}),
          [appId]: buildId
        }
      }))
    }
    
    // 保存构建变更
    const handleSaveBuildChanges = async () => {
      try {
        await batchService.updateBuilds({
          batch_id: batchId,
          operator: user?.username || 'unknown',
          build_changes: currentBuildChanges,
        })
        message.success('构建版本更新成功')
        
        // 清空该批次的修改记录
        setBuildChanges(prev => {
          const newChanges = { ...prev }
          delete newChanges[batchId]
          return newChanges
        })
        
        // 刷新批次详情
        await queryClient.invalidateQueries({ queryKey: ['batchDetails'] })
        await refetchDetails()
      } catch (error: any) {
        message.error(error.response?.data?.message || '更新失败，请重试')
      }
    }
    
    // 还原/取消所有修改
    const handleCancelBuildChanges = () => {
      setBuildChanges(prev => {
        const newChanges = { ...prev }
        delete newChanges[batchId]
        return newChanges
      })
      message.info('已取消所有修改')
    }
    
    // 处理分页变更
    const handlePaginationChange = (page: number, pageSize: number) => {
      setBatchAppPagination(prev => ({
        ...prev,
        [batchId]: { page, pageSize }
      }))
    }

    const detailAppColumns: ColumnsType<ReleaseApp> = [
      {
        title: t('batch.appName'),
        dataIndex: 'app_name',
        key: 'app_name',
        width: 180,
        ellipsis: true,
        render: (name: string, record: ReleaseApp) => (
          <div>
            <div style={{ fontWeight: 500, fontSize: 13 }}>{name}</div>
            {record.release_notes && (
              <div style={{ fontSize: 11, color: '#8c8c8c', marginTop: 2 }}>
                {record.release_notes}
              </div>
            )}
          </div>
        ),
      },
      {
        title: t('batch.appType'),
        dataIndex: 'app_type',
        key: 'app_type',
        width: 80,
        render: (type: string) => (
          <Tag color="blue" style={{ fontSize: 11 }}>{type}</Tag>
        ),
      },
      {
        title: '代码库',
        dataIndex: 'repo_full_name',
        key: 'repo_full_name',
        width: 180,
        ellipsis: true,
        render: (text: string) => (
          <span style={{ fontSize: 11 }}>{text || '-'}</span>
        ),
      },
      {
        title: isBatchCompleted ? t('batch.oldVersion') : t('batch.currentVersion'),
        key: isBatchCompleted ? 'old_version' : 'current_version',
        width: 150,
        ellipsis: true,
        render: (_: any, record: ReleaseApp) => (
          <span style={{ fontSize: 11 }}>
            {isBatchCompleted ? (record.previous_deployed_tag || '-') : (record.deployed_tag || '-')}
          </span>
        ),
      },
      {
        title: isBatchCompleted ? t('batch.deployed') : t('batch.pendingDeploy'),
        key: isBatchCompleted ? 'deployed' : 'pending_deploy',
        width: 200,
        render: (_: any, record: ReleaseApp) => {
          // 如果可以编辑且有 recent_builds，显示下拉选择
          if (canEditBuilds && record.recent_builds && record.recent_builds.length > 0) {
            const currentValue = currentBuildChanges[record.app_id] || record.build_id
            const isModified = record.app_id in currentBuildChanges
            const initialBuildId = record.build_id

            // 获取当前选中的构建，用于显示
            const selectedBuild = record.recent_builds.find((b: BuildSummary) => b.id === currentValue)
            const displayLabel = selectedBuild ? selectedBuild.image_tag : ''

            // 处理长文本：如果超过20个字符，优先显示后段
            const formatLabel = (text: string, maxLen: number = 20) => {
              if (!text) return ''
              if (text.length <= maxLen) return text
              return '...' + text.slice(-(maxLen - 3))
            }

            return (
              <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <Tooltip title={displayLabel} placement="topLeft">
                  <Select
                    style={{ width: 'calc(100% - 22px)' }}
                    value={currentValue}
                    onChange={(value) => handleBuildChange(record.app_id, value)}
                    size="small"
                    optionLabelProp="label"
                    status={isModified ? 'warning' : undefined}
                    dropdownMatchSelectWidth={false}
                    dropdownStyle={{ width: 320 }}
                  >
                    {record.recent_builds.map((build: BuildSummary) => {
                      const isInitial = build.id === initialBuildId
                      return (
                        <Select.Option
                          key={build.id}
                          value={build.id}
                          label={formatLabel(build.image_tag)}
                        >
                          <div style={{ fontSize: 11 }}>
                            <div>
                              <code style={{
                                fontSize: 11,
                                fontWeight: isInitial ? 600 : 400,
                              }}>
                                {build.image_tag}
                              </code>
                            </div>
                            <div
                              style={{
                                color: '#8c8c8c',
                                fontSize: 10,
                                overflow: 'hidden',
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                                direction: 'rtl',
                                textAlign: 'left',
                              }}
                            >
                              {build.commit_message || ''}
                            </div>
                            <div
                              style={{
                                color: '#8c8c8c',
                                fontSize: 9,
                                display: 'flex',
                                justifyContent: 'space-between',
                                alignItems: 'center',
                              }}
                            >
                              <span>{dayjs(build.build_created).format('YYYY-MM-DD HH:mm')}</span>
                              <span style={{ marginLeft: 8, flexShrink: 0 }}>#{build.id}</span>
                            </div>
                          </div>
                        </Select.Option>
                      )
                    })}
                  </Select>
                </Tooltip>
                {isModified ? (
                  <Tooltip title="已修改">
                    <ExclamationCircleOutlined style={{ color: '#faad14', fontSize: 14 }} />
                  </Tooltip>
                ) : (
                  <span style={{ width: 14, height: 14 }} />
                )}
              </div>
            )
          }
          // 否则显示普通文本
          return (
            <div style={{ fontSize: 11 }}>
              {record.target_tag ? (
                <code style={{ fontSize: 11 }}>{record.target_tag}</code>
              ) : (
                '-'
              )}
            </div>
          )
        },
      },
      {
        title: t('batch.commitMessage'),
        dataIndex: 'commit_message',
        key: 'commit_message',
        ellipsis: true,
        render: (text: string) => (
          <span style={{ fontSize: 11 }}>{text || '-'}</span>
        ),
      },
    ]

    const totalApps = detail.total_apps || detail.apps?.length || 0
    const showPagination = totalApps > pagination.pageSize

    return (
      <div className="batch-card-detail">
        <div className="batch-card-detail-sections">
          <div>
            <div className="batch-card-detail-section-title" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span>{t('batch.appList')} ({totalApps})</span>
              <Space size="small" style={{ height: 28, alignItems: 'center' }}>
                {/* 分页器 */}
                {showPagination && (
                  <Pagination
                    simple
                    size="small"
                    current={pagination.page}
                    pageSize={pagination.pageSize}
                    total={totalApps}
                    onChange={handlePaginationChange}
                    showSizeChanger
                    pageSizeOptions={['10', '20', '50']}
                  />
                )}
                
                {/* 还原和应用按钮 */}
                {Object.keys(currentBuildChanges).length > 0 && (
                  <>
                    <Button
                      icon={<UndoOutlined />}
                      onClick={handleCancelBuildChanges}
                      size="small"
                    >
                      还原
                    </Button>
                    <Button
                      type="primary"
                      icon={<SaveOutlined />}
                      onClick={handleSaveBuildChanges}
                      size="small"
                    >
                      应用 ({Object.keys(currentBuildChanges).length})
                    </Button>
                  </>
                )}
              </Space>
            </div>
            <div className="batch-apps-section">
              <div style={{
                position: 'relative',
                opacity: isLoading ? 0.6 : 1,
                transition: 'opacity 0.2s ease',
                pointerEvents: isLoading ? 'none' : 'auto',
                minHeight: detail.apps && detail.apps.length > 0 ? 'auto' : '200px'
              }}>
                <Table
                  columns={detailAppColumns}
                  dataSource={detail.apps || []}
                  rowKey="id"
                  pagination={false}
                  size="small"
                  rowClassName={(record: ReleaseApp) => 
                    record.app_id in currentBuildChanges ? 'batch-app-row-modified' : ''
                  }
                />
                {isLoading && (
                  <div style={{
                    position: 'absolute',
                    top: '50%',
                    left: '50%',
                    transform: 'translate(-50%, -50%)',
                    zIndex: 10
                  }}>
                    <Spin />
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  const renderBatchCard = (record: Batch) => {
    const detail = batchDetailsMap[record.id]
    const isExpanded = expandedRowKeys.includes(record.id)
    const isDetailLoading = isExpanded && (isFetchingDetails || refreshingDetails)

    // 根据状态添加 CSS 类（使用最新的状态）
    const currentStatus = detail?.status ?? record.status
    const statusClass =
      currentStatus === 0 ? 'status-draft' :            // 草稿 - 淡黄色
      currentStatus === 10 ? 'status-sealed' :          // 已封板 - 紫色
      currentStatus === 20 || currentStatus === 21 ? 'status-pre-deploying' : // 预发布中 - 蓝色流光
      currentStatus === 22 ? 'status-pre-deployed' :     // 预发布完成 - 固定蓝色
      currentStatus === 30 || currentStatus === 31 ? 'status-prod-deploying' : // 生产部署中 - 橙色流光
      currentStatus === 32 ? 'status-prod-deployed' :    // 生产部署完成 - 固定橙色
      currentStatus === 40 ? 'status-completed' :        // 已完成 - 绿色
      currentStatus === 90 ? 'status-cancelled' : ''     // 已取消 - 灰色

    // 合并数据：优先使用 detail（最新数据），如果 detail 不存在或某些字段为空，则使用 record
    const timelineBatch: Batch = detail ? {
      ...record,
      ...detail,
      // 确保关键字段使用最新的值
      status: detail.status ?? record.status,
      approval_status: detail.approval_status ?? record.approval_status,
    } : record

    const batchNumberContent = (
      <div className="batch-cell batch-cell-number">
        <div className="batch-number-text">
          <span className="batch-number-id">#{record.id}</span>
          <span className="batch-number-main">{record.batch_number}</span>
        </div>
        <div className="batch-subtext">{t('batch.initiator')}: {record.initiator || '-'}</div>
      </div>
    )

    return (
      <div key={record.id} className={`batch-card ${isExpanded ? 'expanded' : ''} ${statusClass}`}>
        <div className="batch-card-main" onClick={() => handleCardToggle(record)}>
          {record.release_notes ? (
            <Tooltip
              title={record.release_notes}
              color="#1890ff"
              overlayInnerStyle={{ fontSize: '12px', padding: '6px 10px' }}
            >
              {batchNumberContent}
            </Tooltip>
          ) : (
            batchNumberContent
          )}
          <div className="batch-cell batch-cell-apps">
            <span className="batch-app-count">{record.app_count || 0} {t('batch.apps')}</span>
          </div>
          <div className="batch-cell batch-cell-status">
            <StatusTag status={currentStatus} />
          </div>
          <div className="batch-cell batch-cell-approval">
            <StatusTag status={currentStatus} approvalStatus={detail?.approval_status ?? record.approval_status} showApproval />
          </div>
          <div className="batch-cell batch-cell-created">
            {dayjs(record.created_at).format('YYYY-MM-DD HH:mm')}
          </div>
          <div
            className="batch-cell batch-cell-actions"
            onClick={(e) => {
              e.stopPropagation()
            }}
          >
            {renderActionButtons(record)}
          </div>
        </div>

        <div className="batch-card-timeline">
          <BatchTimeline batch={timelineBatch} />
        </div>

        {isExpanded && renderBatchDetail(detail, isDetailLoading)}
      </div>
    )
  }

  const renderBatchCards = () => {
    if (isLoading && !batchResponse) {
      return (
        <div className="batch-card-empty">
          <Spin />
        </div>
      )
    }

    const isRefreshing = isFetching && !refreshingList && !refreshingDetails

    return (
      <div
        className={`batch-card-list-wrapper ${(refreshingList && expandedRowKeys.length === 0) || isRefreshing ? 'refreshing' : ''}`}
      >
        <div className="batch-card-list">
          <div className="batch-card-header">
            <div className="batch-header-cell batch-cell-number">{t('batch.batchNumber')}</div>
            <div className="batch-header-cell batch-cell-apps">{t('batch.appCount')}</div>
            <div className="batch-header-cell batch-cell-status">{t('batch.status')}</div>
            <div className="batch-header-cell batch-cell-approval">{t('batch.approvalStatus')}</div>
            <div className="batch-header-cell batch-cell-created">{t('batch.createdAt')}</div>
            <div className="batch-header-cell batch-cell-actions">{t('common.action')}</div>
          </div>
          {batches.length ? (
            batches.map((record) => renderBatchCard(record))
          ) : (
            <div className="batch-card-empty">暂无批次数据</div>
          )}
        </div>
        {((refreshingList && expandedRowKeys.length === 0) || isRefreshing) && (
          <div className="batch-list-refresh-mask">
            <Spin />
          </div>
        )}
      </div>
    )
  }


  return (
    <div className="batch-list-container">
      <Card
        title={t('batch.title')}
        extra={
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={handleRefresh}
              loading={refreshingDetails || (refreshingList && expandedRowKeys.length === 0)}
            >
              {t('common.reset')}
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateDrawerOpen(true)}
            >
              {t('batch.create')}
            </Button>
          </Space>
        }
      >
        {/* 筛选区 */}
        <div className="batch-filters">
          <Space wrap size="middle">
            <Select
              mode="multiple"
              placeholder={t('batch.filterByStatus')}
              allowClear
              style={{ width: 200 }}
              value={params.status}
              onChange={(value) => setParams({ ...params, status: value, page: 1 })}
              maxTagCount="responsive"
            >
              <Select.Option value={0}>{t('batch.statusDraft')}</Select.Option>
              <Select.Option value={10}>{t('batch.statusSealed')}</Select.Option>
              <Select.Option value={21}>{t('batch.statusPreDeploying')}</Select.Option>
              <Select.Option value={22}>{t('batch.statusPreDeployed')}</Select.Option>
              <Select.Option value={31}>{t('batch.statusProdDeploying')}</Select.Option>
              <Select.Option value={32}>{t('batch.statusProdDeployed')}</Select.Option>
              <Select.Option value={40}>{t('batch.statusCompleted')}</Select.Option>
              <Select.Option value={90}>{t('batch.statusCancelled')}</Select.Option>
            </Select>

            <Select
              placeholder={t('batch.filterByApproval')}
              allowClear
              style={{ width: 130 }}
              value={params.approval_status}
              onChange={(value) => setParams({ ...params, approval_status: value, page: 1 })}
            >
              <Select.Option value="pending">{t('batch.approvalPending')}</Select.Option>
              <Select.Option value="approved">{t('batch.approvalApproved')}</Select.Option>
              <Select.Option value="rejected">{t('batch.approvalRejected')}</Select.Option>
              <Select.Option value="skipped">{t('batch.approvalSkipped')}</Select.Option>
            </Select>

            <Input
              placeholder={t('batch.filterByInitiator')}
              allowClear
              style={{ width: 150 }}
              value={initiatorInput}
              onChange={(e) => setInitiatorInput(e.target.value)}
              onClear={() => setInitiatorInput('')}
            />

            <Input
              placeholder={t('batch.filterByKeyword')}
              allowClear
              style={{ width: 200 }}
              value={keywordInput}
              onChange={(e) => setKeywordInput(e.target.value)}
              onClear={() => setKeywordInput('')}
            />

            <RangePicker
              style={{ width: 280 }}
              format="YYYY-MM-DD"
              defaultValue={[dayjs().subtract(30, 'day'), dayjs()]}
              presets={[
                { label: '最近3天', value: [dayjs().subtract(3, 'day'), dayjs()] },
                { label: '最近7天', value: [dayjs().subtract(7, 'day'), dayjs()] },
                { label: '最近14天', value: [dayjs().subtract(14, 'day'), dayjs()] },
                { label: '最近30天', value: [dayjs().subtract(30, 'day'), dayjs()] },
                { label: '最近90天', value: [dayjs().subtract(90, 'day'), dayjs()] },
              ]}
              onChange={(dates) => {
                if (dates) {
                  setParams({
                    ...params,
                    start_time: dates[0]?.startOf('day').toISOString(),
                    end_time: dates[1]?.endOf('day').toISOString(),
                    page: 1,
                  })
                } else {
                  setParams({
                    ...params,
                    start_time: undefined,
                    end_time: undefined,
                    page: 1,
                  })
                }
              }}
            />
          </Space>
        </div>

        {/* 列表区域 */}
        {renderBatchCards()}

        <div className="batch-pagination">
          <Pagination
            current={params.page}
            pageSize={params.page_size}
            total={total}
            showSizeChanger
            showQuickJumper
            showTotal={(total) => `${t('common.total')} ${total} ${t('batch.list')}`}
            onChange={(page, pageSize) => {
              setParams({ ...params, page, page_size: pageSize })
            }}
          />
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

      {/* 创建批次 Drawer */}
      <BatchCreateDrawer
        open={createDrawerOpen}
        onClose={() => setCreateDrawerOpen(false)}
        onSuccess={() => {
          setCreateDrawerOpen(false)
          refetch()
        }}
      />

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
          refetch()
        }}
      />
    </div>
  )
}

