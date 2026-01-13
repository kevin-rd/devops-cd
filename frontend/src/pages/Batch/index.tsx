import {useEffect, useMemo, useState} from 'react'
import {Badge, Button, Card, DatePicker, Input, message, Modal, Pagination, Radio, Select, Space, Spin, Table, Tag, Tooltip,} from 'antd'
import {CheckCircleOutlined, CloseCircleOutlined, PlusOutlined, ReloadOutlined, SaveOutlined, UndoOutlined,} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {useTranslation} from 'react-i18next'
import dayjs from 'dayjs'
import {formatCreatedTime} from '@/utils/time'

import type {ColumnsType} from 'antd/es/table'
import {batchService} from '@/services/batch'
import {projectService, ProjectSimple} from '@/services/project'
import {useNavigate} from 'react-router-dom'
import {useAuthStore} from '@/stores/authStore'
import {StatusTag} from '@/components/StatusTag'
import {BatchTimeline} from '@/components/BatchTimeline'
import type {BatchActionRequest, BatchQueryParams, BuildSummary} from '@/types'
import './index.css'
import './status-card.css'
import {Batch, BatchAction, BatchStatus} from "@/types/batch.ts";
import {ReleaseApp} from "@/types/release_app.ts";
import {BatchStatusConfig} from "@/pages/Batch/utils/status.tsx";

const {RangePicker} = DatePicker
const {TextArea} = Input

export default function BatchList() {
  const {t} = useTranslation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const {user} = useAuthStore()
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
  const [refreshingList, setRefreshingList] = useState(false)
  const [refreshingDetails, setRefreshingDetails] = useState(false)
  const [createModalOpen, setCreateModalOpen] = useState(false)

  // 创建Modal相关状态
  const [createFormData, setCreateFormData] = useState({
    batch_number: '',
    project_id: undefined as number | undefined,
    initiator: '',
    release_notes: '',
  })

  // 审批相关状态
  const [approvalModalVisible, setApprovalModalVisible] = useState(false)
  const [approvalBatchId, setApprovalBatchId] = useState<number | null>(null)
  const [approvalAction, setApprovalAction] = useState<'approve' | 'reject'>('approve')
  const [approvalReason, setApprovalReason] = useState('')
  const [approvalLoading, setApprovalLoading] = useState(false)

  // 【新增】每个批次的应用列表分页状态 {batchId: {page, pageSize}}
  const [batchAppPagination, setBatchAppPagination] = useState<Record<number, { page: number, pageSize: number }>>({})

  // 【新增】每个批次的构建修改状态 {batchId: {appId: buildId}}
  const [buildChanges, setBuildChanges] = useState<Record<number, Record<number, number>>>({})

  // 防抖输入框的本地状态
  const [initiatorInput] = useState<string>(params.initiator || '')
  const [keywordInput, setKeywordInput] = useState<string>(params.keyword || '')

  // 发起人防抖效果
  useEffect(() => {
    const timer = setTimeout(() => {
      setParams((prev) => ({...prev, initiator: initiatorInput || undefined, page: 1}))
    }, 500)
    return () => clearTimeout(timer)
  }, [initiatorInput])

  // 关键词防抖效果
  useEffect(() => {
    const timer = setTimeout(() => {
      setParams((prev) => ({...prev, keyword: keywordInput || undefined, page: 1}))
    }, 500)
    return () => clearTimeout(timer)
  }, [keywordInput])

  // 查询项目列表（用于创建Modal）
  const {data: projectsData} = useQuery({
    queryKey: ['projects'],
    queryFn: async (): Promise<ProjectSimple[]> => {
      const res = await projectService.getAll()
      return res.data
    },
    enabled: createModalOpen, // 只有打开Modal时才查询
  })

  const projectOptions = projectsData?.map(project => ({
    label: project.name,
    value: project.id,
  }))

  // 查询批次列表 - 如果有待部署中的批次，每5秒自动刷新
  const {data: batchResponse, isLoading, refetch} = useQuery({
    queryKey: ['batchList', params],
    queryFn: async () => {
      const res = await batchService.list(params)
      // 后端返回格式: { code: 200, message: "success", data: { items: [...], total: 2, page: 1, page_size: 20 } }
      // res.data 是一个包含 items, total, page, page_size 的对象
      const pageData = res.data
      return {
        items: Array.isArray(pageData?.items) ? pageData.items : [],
        total: pageData?.total || 0,
        page: pageData?.page || 1,
        page_size: pageData?.page_size || 20,
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
          batch.status === BatchStatus.PreTriggered || batch.status === BatchStatus.PreDeploying || // 预发布中
          batch.status === BatchStatus.ProdTriggered || batch.status === BatchStatus.ProdDeploying    // 生产部署中
      )

      return hasDeployingBatches ? 5_000 : 30_000 // 如果有部署中的批次，每5秒刷新一次
    },
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
  })

  const batches = batchResponse?.items || []
  const total = batchResponse?.total || 0

  // 查询批次详情（用于展开行 - 完整详情，不轮询）
  const {data: batchDetailsMap = {}, refetch: refetchDetails, isFetching: isFetchingDetails} = useQuery({
    queryKey: ['batchDetails', expandedRowKeys, batchAppPagination],
    queryFn: async () => {
      const detailsMap: Record<number, Batch> = {}
      await Promise.all(
        expandedRowKeys.map(async (id) => {
          const pagination = batchAppPagination[id] || {page: 1, pageSize: 20}
          // 获取批次列表信息以判断状态
          const batch = batches.find((b) => b.id === id)
          // 封板后（status >= 10）不需要构建记录
          const withoutRecentBuilds = !!batch && batch.status >= BatchStatus.Sealed
          const res = await batchService.get(id, pagination.page, pagination.pageSize, !withoutRecentBuilds)
          detailsMap[id] = res.data as Batch
        })
      )
      return detailsMap
    },
    enabled: expandedRowKeys.length > 0,
    staleTime: 2_000,
    // 不再在这里轮询，改用下面的轻量级状态轮询
  })

  // 检查是否有展开的批次处于部署中状态（用于决定是否启用状态轮询）
  const hasDeployingExpandedBatch = useMemo(() => {
    if (expandedRowKeys.length === 0) return false
    if (Object.keys(batchDetailsMap).length === 0) return false

    return expandedRowKeys.some((id) => {
      const batch = batchDetailsMap[id]
      return batch && (
        batch.status === BatchStatus.PreTriggered || batch.status === BatchStatus.PreDeploying || // 预发布中
        batch.status === BatchStatus.ProdTriggered || batch.status === BatchStatus.ProdDeploying    // 生产部署中
      )
    })
  }, [expandedRowKeys, batchDetailsMap])

  // 轻量级状态轮询（仅用于部署中的批次，不设置loading状态）
  // 注意：初次展开时不调用，因为 batch 详情接口已包含状态信息
  // 只在轮询周期（每5秒）时才调用，用于更新状态
  const {data: batchStatusMap = {}} = useQuery({
    queryKey: ['batchStatus', expandedRowKeys, batchAppPagination],
    queryFn: async () => {
      const statusMap: Record<number, Batch> = {}
      await Promise.all(
        expandedRowKeys.map(async (id) => {
          const pagination = batchAppPagination[id] || {page: 1, pageSize: 20}
          const res = await batchService.getStatus(id, pagination.page, pagination.pageSize)
          statusMap[id] = res.data as Batch
        })
      )
      return statusMap
    },
    // 只有在有部署中的批次时才启用查询
    enabled: hasDeployingExpandedBatch,
    // 避免初次挂载时立即执行，只依赖轮询间隔
    refetchOnMount: false,
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
    refetchInterval: () => {
      if (document.hidden) {
        return 30_000;
      }
      return 5_000;
    },
    refetchIntervalInBackground: false,
    staleTime: 5000,
  })

  // 当获取到详情数据或状态数据后，同步更新列表缓存中的数据
  useEffect(() => {
    if (batchResponse && (Object.keys(batchDetailsMap).length > 0 || Object.keys(batchStatusMap).length > 0)) {
      // 更新列表缓存中的数据
      queryClient.setQueryData(['batchList', params], (oldData: any) => {
        if (!oldData) return oldData

        const updatedItems = oldData.items.map((item: Batch) => {
          const detail = batchDetailsMap[item.id]
          const status = batchStatusMap[item.id]

          // 优先使用详情数据，其次使用状态数据
          if (detail) {
            // 合并详情数据到列表项，优先使用详情数据
            // 注意：保留 item 的 app_count 等列表字段
            return {
              ...item,
              ...detail,
              // 确保关键字段使用最新的值
              status: detail.status ?? item.status,
              approval_status: detail.approval_status ?? item.approval_status,
              app_count: item.app_count, // 保留列表的 app_count
            }
          } else if (status) {
            // 只更新状态相关字段，保留列表项的其他数据
            // 注意：状态接口返回 sealed_at，而详情接口返回 tagged_at
            // 合并应用数据：保留列表项的完整数据，只更新状态字段
            let mergedApps = item.apps
            if (status.apps && status.apps.length > 0 && item.apps && item.apps.length > 0) {
              // 创建状态映射表
              const statusMap = new Map(
                status.apps.map((app: any) => [app.app_id, {status: app.status, is_locked: app.is_locked}])
              )

              // 更新列表数据中的状态字段
              mergedApps = item.apps.map((app) => {
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

            return {
              ...item,
              status: status.status ?? item.status,
              approval_status: status.approval_status ?? item.approval_status,
              tagged_at: (status as any).sealed_at ?? item.tagged_at, // 状态接口字段名不同
              pre_deploy_started_at: status.pre_deploy_started_at ?? item.pre_deploy_started_at,
              pre_deploy_finished_at: status.pre_deploy_finished_at ?? item.pre_deploy_finished_at,
              prod_deploy_started_at: status.prod_deploy_started_at ?? item.prod_deploy_started_at,
              prod_deploy_finished_at: status.prod_deploy_finished_at ?? item.prod_deploy_finished_at,
              final_accepted_at: status.final_accepted_at ?? item.final_accepted_at,
              cancelled_at: status.cancelled_at ?? item.cancelled_at,
              updated_at: status.updated_at ?? item.updated_at,
              apps: mergedApps, // 使用合并后的应用数据
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
  }, [batchDetailsMap, batchStatusMap])

  // 批次操作 Mutation
  const actionMutation = useMutation({
    mutationFn: (data: BatchActionRequest) => batchService.action(data),
    onSuccess: (_, req) => {
      message.success(t('batch.actionSuccess'))

      queryClient.setQueryData(['batchList', params], (old: any) => {
        return {
          ...old,
          items: old.items.map((b: Batch) => b.id === req.batch_id ? {
            ...b,
            status:
              req.action === BatchAction.Seal ? BatchStatus.Sealed :
                req.action === BatchAction.StartPreDeploy ? BatchStatus.PreTriggered :
                  req.action === BatchAction.StartProdDeploy ? BatchStatus.ProdTriggered :
                    req.action === BatchAction.Cancel ? BatchStatus.Cancelled :
                      req.action === BatchAction.Complete ? BatchStatus.Completed :
                        b.status
          } : b),
        }
      })

      // 使用部分匹配来刷新所有相关查询
      // queryClient.invalidateQueries({queryKey: ['batchList']})
      // queryClient.invalidateQueries({queryKey: ['batchDetails']})

      if (req.action === BatchAction.Seal) {
        queryClient.invalidateQueries({queryKey: ['batchDetail', req.batch_id]})
        navigate(`/batch/${req.batch_id}/detail`)
      } else if (req.action === BatchAction.StartPreDeploy || req.action === BatchAction.StartProdDeploy) {
        queryClient.invalidateQueries({queryKey: ['batchDetail', req.batch_id]})
        navigate(`/batch/${req.batch_id}/detail?tab=graph`)
      } else {
        queryClient.invalidateQueries({queryKey: ['batchList']})
      }
    },
    onError: () => {
    },
  })

  // 创建批次 Mutation
  const createBatchMutation = useMutation({
    mutationFn: (data: any) => batchService.create(data),
    onSuccess: () => {
      message.success(t('batch.createSuccess'))
      setCreateModalOpen(false)
      setCreateFormData({
        batch_number: '',
        project_id: undefined,
        initiator: '',
        release_notes: '',
      })
      refetch()
    },
    onError: (error: any) => {
      const errorMsg = error.response?.data?.message || error.message || t('common.error')
      message.error(errorMsg)
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
    if (action === BatchAction.Cancel) {
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
        action: BatchAction.Cancel,
        operator: user?.username || 'unknown',
        reason: cancelReason,
      })
      setCancelModalVisible(false)
      setCancelReason('')
      setCurrentBatchId(null)
    }
  }

  // 打开审批 Modal
  const handleOpenApproval = (batchId: number) => {
    setApprovalBatchId(batchId)
    setApprovalAction('approve')
    setApprovalReason('')
    setApprovalModalVisible(true)
  }

  // 确认审批
  const handleConfirmApproval = async () => {
    if (approvalAction === 'reject' && !approvalReason.trim()) {
      message.warning('请输入拒绝原因')
      return
    }

    if (!approvalBatchId) return

    setApprovalLoading(true)
    try {
      if (approvalAction === 'approve') {
        await batchService.approve({
          batch_id: approvalBatchId,
          operator: user?.username || 'unknown',
        })
        message.success(t('batch.approveSuccess'))
      } else {
        await batchService.reject({
          batch_id: approvalBatchId,
          operator: user?.username || 'unknown',
          reason: approvalReason,
        })
        message.success(t('batch.rejectSuccess'))
      }

      await queryClient.invalidateQueries({queryKey: ['batchList']})
      await queryClient.invalidateQueries({queryKey: ['batchDetails']})
      refetch()

      setApprovalModalVisible(false)
      setApprovalBatchId(null)
      setApprovalReason('')
    } catch (error: any) {
      message.error(error.response?.data?.message || t('common.error'))
    } finally {
      setApprovalLoading(false)
    }
  }

  // 处理创建Modal
  const handleCreateModalOpen = () => {
    setCreateFormData({
      batch_number: '',
      project_id: undefined,
      initiator: user?.username || '',
      release_notes: '',
    })
    setCreateModalOpen(true)
  }

  const handleCreateModalClose = () => {
    setCreateModalOpen(false)
    setCreateFormData({
      batch_number: '',
      project_id: undefined,
      initiator: '',
      release_notes: '',
    })
  }

  const handleCreateBatch = async () => {
    if (!createFormData.batch_number.trim()) {
      message.warning('请输入批次编号')
      return
    }
    if (!createFormData.project_id) {
      message.warning('请选择所属项目')
      return
    }

    const requestData: any = {
      batch_number: createFormData.batch_number.trim(),
      project_id: createFormData.project_id,
      initiator: createFormData.initiator || user?.username || 'unknown',
      apps: [], // 空的应用列表
    }

    if (createFormData.release_notes.trim()) {
      requestData.release_notes = createFormData.release_notes.trim()
    }

    createBatchMutation.mutate(requestData)
  }

  // 处理刷新按钮
  const handleRefresh = async () => {
    if (expandedRowKeys.length > 0) {
      setRefreshingDetails(true)
      try {
        await refetchDetails()
      } catch (error: any) {
        message.error('刷新批次详情失败，请稍后重试')
      }

      try {
        await refetch()
      } catch (error: any) {
        message.error('刷新批次列表失败，请稍后重试')
      } finally {
        setRefreshingDetails(false)
      }
      return
    }

    try {
      setRefreshingList(true)
      await refetch()
    } catch (error: any) {
      message.error('刷新批次列表失败，请稍后重试')
    } finally {
      setRefreshingList(false)
    }
  }

  const renderActionButtons = (record: Batch) => {
    // 如果批次已取消，不显示任何操作按钮
    if (record.status === 90) {
      // todo: 显示取消时间、原因等
      return null
    }

    return (
      <Space size="small" wrap>
        {/*编辑按钮*/}
        {/*{(record.status === 0) && (*/}
        {/*  <Button*/}
        {/*    size="small"*/}
        {/*    icon={<EditOutlined/>}*/}
        {/*    onClick={(e) => {*/}
        {/*      e.stopPropagation()*/}
        {/*      // 优先使用已展开的详情数据，否则使用列表数据*/}
        {/*      const batchToEdit = batchDetailsMap[record.id] || record*/}
        {/*      setEditingBatch(batchToEdit)*/}
        {/*      setEditDrawerOpen(true)*/}
        {/*    }}*/}
        {/*  >*/}
        {/*    {t('common.edit')}*/}
        {/*  </Button>*/}
        {/*)}*/}

        {/*封板按钮*/}
        {record.status === 0 && record.app_count > 0 && (
          <Button
            size="small"
            icon={<CheckCircleOutlined/>}
            type="primary"
            onClick={(e) => {
              e.stopPropagation()
              handleAction(record.id, 'seal')
            }}
          >
            {t('batch.seal')}
          </Button>
        )}

        {/*开始Pre发布按钮*/}
        {/*{(record.status === 10 && record.approval_status === 'approved') && (*/}
        {/*  <Button*/}
        {/*    size="small"*/}
        {/*    icon={<PlayCircleOutlined/>}*/}
        {/*    type="primary"*/}
        {/*    onClick={(e) => {*/}
        {/*      e.stopPropagation()*/}
        {/*      handleAction(record.id, 'start_pre_deploy')*/}
        {/*    }}*/}
        {/*  >*/}
        {/*    {t('batch.startPreDeploy')}*/}
        {/*  </Button>*/}
        {/*)}*/}
        {/*/!*开始Prod发布按钮*!/*/}
        {/*{record.status === 22 && (*/}
        {/*  <Button*/}
        {/*    size="small"*/}
        {/*    icon={<PlayCircleOutlined/>}*/}
        {/*    danger*/}
        {/*    onClick={(e) => {*/}
        {/*      e.stopPropagation()*/}
        {/*      handleAction(record.id, BatchAction.StartProdDeploy)*/}
        {/*    }}*/}
        {/*  >*/}
        {/*    {t('batch.startProdDeploy')}*/}
        {/*  </Button>*/}
        {/*)}*/}
        {record.status === 32 && (
          <Button
            type="primary"
            size="small"
            icon={<CheckCircleOutlined/>}
            onClick={(e) => {
              e.stopPropagation()
              handleAction(record.id, BatchAction.ConfirmProd)
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
            handleAction(record.id, BatchAction.Cancel)
          }}
          disabled={record.status >= 40 || record.status === 90}
        >
          {t('batch.cancelBatch')}
        </Button>
      </Space>
    )
  }

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
            <Spin/>
          </div>
        </div>
      )
    }

    const batchId = detail.id
    const batchStatusValue = Number(detail.status)
    const isBatchCompleted = batchStatusValue === 40
    const canEditBuilds = batchStatusValue < 10 // 只有草稿状态可以修改

    // 获取当前批次的分页状态
    const pagination = batchAppPagination[batchId] || {page: 1, pageSize: 20}

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
          const newChanges = {...prev}
          delete newChanges[batchId]
          return newChanges
        })

        // 刷新批次详情
        await queryClient.invalidateQueries({queryKey: ['batchDetails']})
        await refetchDetails()
      } catch (error: any) {
        message.error(error.response?.data?.message || '更新失败，请重试')
      }
    }

    // 还原/取消所有修改
    const handleCancelBuildChanges = () => {
      setBuildChanges(prev => {
        const newChanges = {...prev}
        delete newChanges[batchId]
        return newChanges
      })
      message.info('已取消所有修改')
    }

    // 处理分页变更
    const handlePaginationChange = (page: number, pageSize: number) => {
      setBatchAppPagination(prev => ({
        ...prev,
        [batchId]: {page, pageSize}
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
            <span style={{color: '#999', fontSize: 12}}>#{record.app_id} </span>
            <span style={{fontWeight: 500, fontSize: 13}}>{name}</span>
            {record.release_notes && (
              <div style={{fontSize: 12, color: '#8c8c8c', marginTop: 2}}>
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
          <Tag color="blue" style={{fontSize: 12}}>{type}</Tag>
        ),
      },
      {
        title: t('application.repository'),
        dataIndex: 'repo_full_name',
        key: 'repo_full_name',
        width: 180,
        ellipsis: true,
        render: (text: string) => (
          <span style={{fontSize: 13}}>{text || '-'}</span>
        ),
      },
      {
        title: isBatchCompleted ? t('batch.oldVersion') : t('batch.currentVersion'),
        key: isBatchCompleted ? 'old_version' : 'current_version',
        width: 150,
        ellipsis: true,
        render: (_: any, record: ReleaseApp) => (
          <span style={{fontSize: 13}}>
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
              <div style={{display: 'flex', alignItems: 'center', gap: 4}}>
                <Tooltip title={displayLabel} placement="topRight">
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
                              <code style={{
                                fontSize: 12,
                                fontWeight: isInitial ? 600 : 400,
                              }}>
                                {build.image_tag}
                              </code>
                            </div>
                            <div
                              style={{
                                color: '#8c8c8c',
                                fontSize: 11,
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
                                fontSize: 10,
                                display: 'flex',
                                justifyContent: 'space-between',
                                alignItems: 'center',
                              }}
                            >
                              <span>{dayjs(build.build_created).format('YYYY-MM-DD HH:mm')}</span>
                              <span style={{marginLeft: 8, flexShrink: 0}}>#{build.id}</span>
                            </div>
                          </div>
                        </Select.Option>
                      )
                    })}
                  </Select>
                </Tooltip>
                {isModified ? (
                  <Tooltip title="还原此应用的修改">
                    <Button
                      type="text"
                      size="small"
                      icon={<UndoOutlined/>}
                      style={{
                        padding: '0 2px',
                        minWidth: '18px',
                        height: '18px',
                        color: '#faad14',
                      }}
                      onClick={(e) => {
                        e.stopPropagation()
                        // 还原单个应用的修改
                        setBuildChanges(prev => {
                          const newChanges = {...prev}
                          if (newChanges[batchId]) {
                            const batchChanges = {...newChanges[batchId]}
                            delete batchChanges[record.app_id]
                            if (Object.keys(batchChanges).length === 0) {
                              delete newChanges[batchId]
                            } else {
                              newChanges[batchId] = batchChanges
                            }
                          }
                          return newChanges
                        })
                      }}
                    />
                  </Tooltip>
                ) : (
                  <span style={{width: 18, height: 18, display: 'inline-block'}}/>
                )}
              </div>
            )
          }
          // 否则显示普通文本
          return (
            <div style={{fontSize: 13}}>
              {record.target_tag ? (
                <code style={{fontSize: 13}}>{record.target_tag}</code>
              ) : (
                '-'
              )}
            </div>
          )
        },
      },
      {
        title: t('batch.commitMessage'),
        key: 'commit_message',
        ellipsis: true,
        render: (_: any, record: ReleaseApp) => {
          // 如果用户选择了不同的 build，显示该 build 的 commit message
          const selectedBuildId = currentBuildChanges[record.app_id] || record.build_id
          let displayCommitMessage = record.commit_message

          // 如果有 recent_builds，从中查找对应的 commit message
          if (record.recent_builds && record.recent_builds.length > 0) {
            const selectedBuild = record.recent_builds.find((b: BuildSummary) => b.id === selectedBuildId)
            if (selectedBuild && selectedBuild.commit_message) {
              displayCommitMessage = selectedBuild.commit_message
            }
          }

          return (
            <span style={{fontSize: 13}}>{displayCommitMessage || '-'}</span>
          )
        },
      },
    ]

    const totalApps = detail.total_apps || detail.apps?.length || 0
    const showPagination = totalApps > pagination.pageSize

    return (
      <div className="batch-card-detail">
        <div className="batch-card-detail-sections">
          <div>
            <div className="batch-card-detail-section-title"
                 style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
              <span>{t('batch.appList')} ({totalApps})</span>
              <Space size="small" style={{height: 28, alignItems: 'center'}}>
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
                      icon={<UndoOutlined/>}
                      onClick={handleCancelBuildChanges}
                      size="small"
                    >还原全部
                    </Button>
                    <Button
                      type="primary"
                      icon={<SaveOutlined/>}
                      onClick={handleSaveBuildChanges}
                      size="small"
                    >应用全部 ({Object.keys(currentBuildChanges).length})
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
                    <Spin/>
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
    const status = batchStatusMap[record.id]
    const isExpanded = expandedRowKeys.includes(record.id)
    const isDetailLoading = isExpanded && (isFetchingDetails || refreshingDetails)

    // 合并数据：优先使用 detail（完整详情），其次使用 status（轻量状态），最后使用 record（列表数据）
    const mergedBatch = detail ? detail : (status ? {...record, ...status} : record)

    // 根据状态添加 CSS 类（使用最新的状态）
    const currentStatus = mergedBatch.status
    const statusClass = BatchStatusConfig[currentStatus]?.class_name || 'default'

    // 时间轴使用合并后的数据
    const timelineBatch: Batch = mergedBatch

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
      <Badge.Ribbon key={record.id} className="batch-status-ribbon" placement="start"
                    text={BatchStatusConfig[currentStatus]?.label || 'unknown: ' + currentStatus}
                    color={BatchStatusConfig[currentStatus]?.color}>
        <div className={`batch-card ${isExpanded ? 'expanded' : ''} ${statusClass}`}>
          <div className="batch-card-main" onClick={() => {
            if (record.status >= BatchStatus.Sealed) {
              navigate(`/batch/${record.id}/detail?tab=graph`)
            } else {
              navigate(`/batch/${record.id}/detail`)
            }
          }}>
            {record.release_notes ? (
              <Tooltip title={record.release_notes}
                       overlayInnerStyle={{fontSize: '12px', padding: '6px 10px'}}>
                {batchNumberContent}</Tooltip>
            ) : (
              batchNumberContent
            )}
            <div className="batch-cell batch-cell-created">
              {(() => {
                const {time, dayInfo} = formatCreatedTime(record.created_at)
                return (
                  <div>
                    <div style={{fontSize: '12px', fontWeight: 500}}>{dayInfo}</div>
                    <div style={{fontSize: '11px', color: '#8c8c8c'}}>{time}</div>
                  </div>
                )
              })()}
            </div>
            <div className="batch-cell batch-cell-apps">
            <span className="batch-app-count" style={{cursor: 'pointer'}}
                  onClick={(e) => {
                    e.stopPropagation()
                    handleCardToggle(record)
                  }}>{record.app_count || 0} {t('batch.apps')}
            </span>
            </div>
            <div className="batch-cell batch-cell-approval">
              <StatusTag
                status={currentStatus}
                approvalStatus={detail?.approval_status ?? record.approval_status}
                showApproval
                onApprovalClick={() => handleOpenApproval(record.id)}
                approvalTime={
                  detail?.approved_at
                    ? dayjs(detail.approved_at).format('MM-DD HH:mm')
                    : record.approved_at
                      ? dayjs(record.approved_at).format('MM-DD HH:mm')
                      : undefined
                }
                rejectReason={detail?.reject_reason ?? record.reject_reason}
                approvedBy={detail?.approved_by ?? record.approved_by}
              />
            </div>
            <div className="batch-cell batch-cell-actions">
              {renderActionButtons(record)}
            </div>
          </div>

          <div className="batch-card-timeline"
               onClick={(e) => {
                 // 阻止触发卡片的展开/收起
                 e.stopPropagation()
                 // 跳转到洞察页面
                 // navigate(`/batch/${record.id}/insights`)
                 navigate(`/batch/${record.id}/detail?tab=graph`)
               }}
               style={{cursor: 'pointer'}}
          >
            <BatchTimeline
              batch={timelineBatch}
              onAction={(action) => handleAction(record.id, action)}
            />
          </div>

          {isExpanded && renderBatchDetail(detail, isDetailLoading)}
        </div>
      </Badge.Ribbon>
    );
  }


  return (
    <div className="batch-list-container">
      <Card
        title={t('batch.title')}
        extra={
          <Space>
            <Button
              icon={<ReloadOutlined/>}
              onClick={handleRefresh}
              loading={refreshingDetails || (refreshingList && expandedRowKeys.length === 0)}
            >
              {t('common.refresh')}
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined/>}
              onClick={handleCreateModalOpen}
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
              style={{width: 200}}
              value={params.status}
              onChange={(value) => setParams({...params, status: value, page: 1})}
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
              style={{width: 120}}
              value={params.approval_status}
              onChange={(value) => setParams({...params, approval_status: value, page: 1})}
            >
              <Select.Option value="pending">{t('batch.approvalPending')}</Select.Option>
              <Select.Option value="approved">{t('batch.approvalApproved')}</Select.Option>
              <Select.Option value="rejected">{t('batch.approvalRejected')}</Select.Option>
              <Select.Option value="skipped">{t('batch.approvalSkipped')}</Select.Option>
            </Select>

            {/*<Input*/}
            {/*  placeholder={t('batch.filterByInitiator')}*/}
            {/*  allowClear*/}
            {/*  style={{width: 150}}*/}
            {/*  value={initiatorInput}*/}
            {/*  onChange={(e) => setInitiatorInput(e.target.value)}*/}
            {/*  onClear={() => setInitiatorInput('')}*/}
            {/*/>*/}

            <Input
              placeholder={t('batch.filterByKeyword')}
              allowClear
              style={{width: 180}}
              value={keywordInput}
              onChange={(e) => setKeywordInput(e.target.value)}
              onClear={() => setKeywordInput('')}
            />

            <RangePicker
              style={{width: 240}}
              format="YYYY-MM-DD"
              defaultValue={[dayjs().subtract(30, 'day'), dayjs()]}
              presets={[
                {label: '最近3天', value: [dayjs().subtract(3, 'day'), dayjs()]},
                {label: '最近7天', value: [dayjs().subtract(7, 'day'), dayjs()]},
                {label: '最近14天', value: [dayjs().subtract(14, 'day'), dayjs()]},
                {label: '最近30天', value: [dayjs().subtract(30, 'day'), dayjs()]},
                {label: '最近90天', value: [dayjs().subtract(90, 'day'), dayjs()]},
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
          <Pagination
            current={params.page}
            pageSize={params.page_size}
            total={total}
            showSizeChanger
            showQuickJumper={false}
            showTotal={(total) => `${t('common.total')} ${total} ${t('common.unit')}`}
            onChange={(page, pageSize) => {
              setParams({...params, page, page_size: pageSize})
            }}
          />
        </div>

        {/* 列表区域 */}
        {/* 只在用户主动刷新列表时显示 loading 状态，自动轮询不显示 */}
        <div className="batch-card-list">
          <div className="batch-card-header">
            <div className="batch-header-cell batch-cell-number">{t('batch.batchNumber')}</div>
            <div className="batch-header-cell batch-cell-created">{t('batch.createdAt')}</div>
            <div className="batch-header-cell batch-cell-apps">{t('batch.appCount')}</div>
            <div className="batch-header-cell batch-cell-approval">{t('batch.approvalStatus')}</div>
            <div className="batch-header-cell batch-cell-actions">{t('common.action')}</div>
          </div>

          <div>
            {batches.length ? (
              batches.map((record) => renderBatchCard(record))
            ) : (
              <div className="batch-card-empty">暂无批次数据</div>
            )}
            {(isLoading || (refreshingList && expandedRowKeys.length === 0)) && (
              <div className="batch-list-refresh-mask">
                <Spin/>
              </div>
            )}
          </div>

        </div>

        <div className="batch-pagination">
          <Pagination
            current={params.page}
            pageSize={params.page_size}
            total={total}
            showSizeChanger
            showQuickJumper
            showTotal={(total) => `${t('common.total')} ${total} ${t('common.unit')}`}
            onChange={(page, pageSize) => {
              setParams({...params, page, page_size: pageSize})
            }}
          />
        </div>

      </Card>

      {/* 创建批次 Modal */}
      <Modal
        title="创建批次"
        open={createModalOpen}
        onOk={handleCreateBatch}
        onCancel={handleCreateModalClose}
        confirmLoading={createBatchMutation.isPending}
        okText="创建"
        cancelText="取消"
        width={600}
      >
        <div style={{padding: '16px 0'}}>
          <Space direction="vertical" style={{width: '100%'}} size="large">
            <div>
              <div style={{marginBottom: 8, fontWeight: 500}}>
                批次编号 <span style={{color: '#ff4d4f'}}>*</span>
              </div>
              <Input
                placeholder="请输入批次编号"
                value={createFormData.batch_number}
                onChange={(e) => setCreateFormData(prev => ({...prev, batch_number: e.target.value}))}
              />
            </div>

            <div>
              <div style={{marginBottom: 8, fontWeight: 500}}>
                所属项目 <span style={{color: '#ff4d4f'}}>*</span>
              </div>
              <Select
                placeholder="请选择项目"
                style={{width: '100%'}}
                value={createFormData.project_id}
                onChange={(value) => setCreateFormData(prev => ({...prev, project_id: value}))}
                options={projectOptions}
                showSearch
                optionFilterProp="label"
              />
            </div>

            <div>
              <div style={{marginBottom: 8, fontWeight: 500}}>
                发起人
              </div>
              <Input
                value={createFormData.initiator}
                onChange={(e) => setCreateFormData(prev => ({...prev, initiator: e.target.value}))}
                disabled
              />
            </div>

            <div>
              <div style={{marginBottom: 8, fontWeight: 500}}>
                发布说明
              </div>
              <TextArea
                rows={4}
                placeholder="请输入发布说明（可选）"
                value={createFormData.release_notes}
                onChange={(e) => setCreateFormData(prev => ({...prev, release_notes: e.target.value}))}
              />
            </div>
          </Space>
        </div>
      </Modal>

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
          setApprovalBatchId(null)
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

    </div>
  )
}

