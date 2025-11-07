import { useState, useEffect, useMemo, useCallback } from 'react'
import {
  Drawer,
  Form,
  Input,
  Button,
  Space,
  Table,
  Checkbox,
  Tag,
  message,
  Alert,
  Steps,
  Select,
} from 'antd'
import { SaveOutlined, FormOutlined, AppstoreOutlined, LeftOutlined, RightOutlined } from '@ant-design/icons'
import axios from 'axios'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import type { ColumnsType } from 'antd/es/table'
import { batchService } from '@/services/batch'
import { applicationService } from '@/services/application'
import { useAuthStore } from '@/stores/authStore'
import type { Batch, ReleaseApp, ApplicationWithBuild, UpdateBatchRequest } from '@/types'
import './index.css'

const { TextArea } = Input

interface AppWithSelection {
  id: number
  name: string
  display_name: string
  app_type: string
  repo_name: string
  repo_full_name: string
  last_tag: string
  deployed_tag?: string
  image_tag?: string
  commit_message?: string
  commit_sha?: string
  selected: boolean
  inBatch: boolean // 是否原本在批次中
  releaseNotes?: string // 应用级发布说明
}

interface SelectedAppState {
  id: number
  name: string
  selected: boolean
  inBatch: boolean
  releaseNotes?: string // 应用级发布说明
}

interface BatchEditDrawerProps {
  open: boolean
  batch: Batch | null
  onClose: () => void
  onSuccess: () => void
}

export default function BatchEditDrawer({ open, batch, onClose, onSuccess }: BatchEditDrawerProps) {
  const { t } = useTranslation()
  const [form] = Form.useForm()
  const { user } = useAuthStore()
  const queryClient = useQueryClient()

  const [currentStep, setCurrentStep] = useState(1) // 默认显示应用管理页面
  const [appSelections, setAppSelections] = useState<AppWithSelection[]>([])
  const [searchKeyword, setSearchKeyword] = useState('')
  const [debouncedSearchKeyword, setDebouncedSearchKeyword] = useState('')
  const [expandedRowKeys, setExpandedRowKeys] = useState<number[]>([])
  const [pageSize, setPageSize] = useState(20)
  const [currentPage, setCurrentPage] = useState(1)
  const [selectionStateMap, setSelectionStateMap] = useState<Record<number, SelectedAppState>>({})

// 搜索防抖：延迟500ms后更新实际的搜索关键词
  useEffect(() => {
    const timer = window.setTimeout(() => {
      setDebouncedSearchKeyword(searchKeyword)
    }, 500)

    return () => {
      window.clearTimeout(timer)
    }
  }, [searchKeyword])

  // 搜索关键词变化时重置页码到第一页
  useEffect(() => {
    if (debouncedSearchKeyword !== '' || searchKeyword === debouncedSearchKeyword) {
      setCurrentPage((prev) => (prev === 1 ? prev : 1))
    }
  }, [debouncedSearchKeyword])

  // 查询批次详情（使用 placeholderData 先展示传入的数据）
  const { data: batchDetailResponse, isLoading: loadingBatchDetail, isFetching: fetchingBatchDetail } = useQuery({
    queryKey: ['batchDetail', batch?.id],
    queryFn: async () => {
      if (!batch?.id) return null
      const res = await batchService.get(batch.id)
      return res.data
    },
    enabled: open && !!batch,
    placeholderData: batch || undefined,
    staleTime: 0, // 始终获取最新数据
  })

  // 当获取到最新数据时，同步更新列表页缓存
  useEffect(() => {
    if (batchDetailResponse && batch?.id && !fetchingBatchDetail) {
      // 更新批次列表缓存
      queryClient.setQueriesData(
        { queryKey: ['batchList'], exact: false },
        (oldData: any) => {
          if (!oldData?.items) return oldData
          return {
            ...oldData,
            items: oldData.items.map((item: Batch) =>
              item.id === batch.id ? { ...item, ...batchDetailResponse } : item
            ),
          }
        }
      )

      // 更新展开详情缓存
      queryClient.setQueriesData(
        { queryKey: ['batchDetails'], exact: false },
        (oldData: any) => {
          if (!oldData) return oldData
          return {
            ...oldData,
            [batch.id]: batchDetailResponse,
          }
        }
      )
    }
  }, [batchDetailResponse, batch?.id, fetchingBatchDetail, queryClient])

  const batchDetail = batchDetailResponse || batch

  useEffect(() => {
    if (!batchDetail?.apps) {
      return
    }
    setSelectionStateMap((prev) => {
      const next = { ...prev }
      let changed = false
      batchDetail.apps.forEach((app: ReleaseApp) => {
        const appId = app.app_id
        if (!appId) return
        const name = app.app_name || app.app_display_name || app.repo_name || app.repo_full_name || `应用${appId}`
        const existing = next[appId]
        if (!existing) {
          next[appId] = { id: appId, name, selected: true, inBatch: true }
          changed = true
        } else {
          const updated: SelectedAppState = {
            id: existing.id,
            name,
            inBatch: true,
            selected: existing.selected ?? true,
          }
          if (
            updated.name !== existing.name ||
            updated.inBatch !== existing.inBatch ||
            updated.selected !== existing.selected
          ) {
            next[appId] = updated
            changed = true
          }
        }
      })
      return changed ? next : prev
    })
  }, [batchDetail])

  // 查询所有应用列表（包含构建信息，服务端分页）
  const { data: allAppsResponse, isLoading: loadingAllApps, error: loadingAppsError } = useQuery({
    queryKey: ['applicationsWithBuilds', debouncedSearchKeyword, currentPage, pageSize],
    queryFn: async ({ signal }) => {
      const res = await applicationService.searchWithBuilds({
        page: currentPage,
        page_size: pageSize,
        keyword: debouncedSearchKeyword || undefined,
      }, signal)
      return res.data
    },
    enabled: open && !!batch && searchKeyword === debouncedSearchKeyword,
    staleTime: 30 * 1000,
    gcTime: 60 * 1000,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    meta: {
      errorMessage: false,
    },
  })

  // 显示错误提示（仅在有错误时）
  useEffect(() => {
    const errorMessageKey = 'batch-edit-apps-fetch-error'
    const isCanceled = loadingAppsError && axios.isCancel && axios.isCancel(loadingAppsError)
    const isAbortError = loadingAppsError instanceof DOMException && loadingAppsError.name === 'AbortError'

    if (loadingAppsError && open && !isCanceled && !isAbortError) {
      console.error('加载应用列表失败:', loadingAppsError)
      message.error({ content: '加载应用列表失败，请稍后重试', key: errorMessageKey })
    } else {
      message.destroy(errorMessageKey)
    }
  }, [loadingAppsError, open])

  // 更新批次 Mutation
  const updateMutation = useMutation({
    mutationFn: (data: UpdateBatchRequest) => batchService.update(data),
    onSuccess: (response) => {
      message.success(t('batch.updateSuccess'))

      // 如果返回了更新后的批次数据，更新缓存
      if (response?.data) {
        const updatedBatch = response.data

        // 更新批次详情缓存
        queryClient.setQueryData(['batchDetail', batch?.id], updatedBatch)

        // 更新批次列表缓存中的对应项
        queryClient.setQueriesData(
          { queryKey: ['batchList'], exact: false },
          (oldData: any) => {
            if (!oldData?.items) return oldData
            return {
              ...oldData,
              items: oldData.items.map((item: Batch) =>
                item.id === updatedBatch.id ? { ...item, ...updatedBatch } : item
              ),
            }
          }
        )

        // 更新展开详情缓存
        queryClient.setQueriesData(
          { queryKey: ['batchDetails'], exact: false },
          (oldData: any) => {
            if (!oldData) return oldData
            return {
              ...oldData,
              [updatedBatch.id]: updatedBatch,
            }
          }
        )
      } else {
        // 如果没有返回数据，则失效缓存让其重新获取
        queryClient.invalidateQueries({ queryKey: ['batchList'] })
        queryClient.invalidateQueries({ queryKey: ['batchDetail', batch?.id] })
        queryClient.invalidateQueries({ queryKey: ['batchDetails'] })
      }

      handleClose()
      onSuccess()
    },
    onError: (error: any) => {
      // 显示后端返回的详细错误信息
      const errorMsg = error.response?.data?.message || error.message || t('common.error')
      const errorDetail = error.response?.data?.detail

      if (errorDetail) {
        message.error(`${errorMsg}: ${errorDetail}`, 5) // 显示5秒
      } else {
        message.error(errorMsg, 5)
      }
    },
  })

  // 初始化表单和应用选择
  useEffect(() => {
    if (open && batchDetail && allAppsResponse?.items) {
      // 设置表单初始值
      form.setFieldsValue({
        batch_number: batchDetail.batch_number,
        release_notes: batchDetail.release_notes || '',
      })

      const batchAppIds = batchDetail.apps?.map((app: ReleaseApp) => app.app_id) || []
      const batchAppsMap = new Map<number, ReleaseApp>()
      batchDetail.apps?.forEach((app: ReleaseApp) => {
        batchAppsMap.set(app.app_id, app)
      })

      // 只为当前页的应用同步状态，不修改其他应用的状态
      setSelectionStateMap((prevState) => {
        const nextState = { ...prevState }

        allAppsResponse.items.forEach((app: ApplicationWithBuild) => {
          const inBatch = batchAppIds.includes(app.id)
          const existing = nextState[app.id]

          if (!existing) {
            // 新应用：如果在批次中则选中
            nextState[app.id] = {
              id: app.id,
              name: app.name,
              selected: inBatch,
              inBatch,
            }
          } else {
            // 已存在的应用：保持原有选中状态，只更新 inBatch 标记
            nextState[app.id] = {
              ...existing,
              name: app.name,
              inBatch: existing.inBatch || inBatch,
            }
          }
        })

        return nextState
      })

      // 根据全局状态设置当前页应用的展示
      setAppSelections(allAppsResponse.items.map((app: ApplicationWithBuild) => {
        const inBatch = batchAppIds.includes(app.id)
        const batchApp = batchAppsMap.get(app.id)
        const state = selectionStateMap[app.id]

        return {
          id: app.id,
          name: app.name,
          display_name: app.display_name,
          app_type: app.app_type,
          repo_name: app.repo_name,
          repo_full_name: app.repo_full_name,
          last_tag: app.last_tag,
          deployed_tag: batchApp?.deployed_tag || app.deployed_tag,
          image_tag: batchApp?.image_tag || app.image_tag,
          commit_message: app.commit_message,
          commit_sha: app.commit_sha,
          selected: state?.selected ?? inBatch,
          inBatch: state?.inBatch ?? inBatch,
        }
      }))
    }
  }, [open, batchDetail, allAppsResponse, form])

  // 关闭并重置
  const handleClose = () => {
    form.resetFields()
    setCurrentStep(1) // 重置为默认步骤
    setAppSelections([])
    setSearchKeyword('')
    setDebouncedSearchKeyword('')
    setExpandedRowKeys([])
    setPageSize(20)
    setCurrentPage(1)
    setSelectionStateMap({})
    onClose()
  }

  // 步骤切换处理
  const handleStepChange = (step: number) => {
    // 如果要切换到步骤2（应用管理），先验证基本信息
    if (step === 1) {
      form.validateFields(['batch_number'])
        .then(() => {
          setCurrentStep(step)
        })
        .catch((error) => {
          console.error('Validation failed:', error)
          message.warning('请先填写批次编号')
        })
    } else {
      setCurrentStep(step)
    }
  }

  const handleSelectionChange = useCallback((record: AppWithSelection, selected: boolean) => {
    setAppSelections(prev =>
      prev.map(app =>
        app.id === record.id ? { ...app, selected } : app
      )
    )
    setSelectionStateMap(prev => {
      const next = { ...prev }
      const existing = next[record.id]
      next[record.id] = {
        id: record.id,
        name: record.name,
        selected,
        inBatch: existing?.inBatch ?? record.inBatch,
        releaseNotes: existing?.releaseNotes ?? record.releaseNotes ?? '',
      }
      return next
    })
  }, [])

  const toggleAppSelection = useCallback((record: AppWithSelection) => {
    handleSelectionChange(record, !record.selected)
  }, [handleSelectionChange])

  const handleDeselectSnapshot = useCallback((snapshot: SelectedAppState) => {
    setAppSelections(prev =>
      prev.map(app =>
        app.id === snapshot.id ? { ...app, selected: false } : app
      )
    )
    setSelectionStateMap(prev => {
      const next = { ...prev }
      const existing = next[snapshot.id]
      if (existing) {
        next[snapshot.id] = { ...existing, selected: false }
      }
      return next
    })
  }, [])

  const handleReselectSnapshot = useCallback((snapshot: SelectedAppState) => {
    setAppSelections(prev =>
      prev.map(app =>
        app.id === snapshot.id ? { ...app, selected: true } : app
      )
    )
    setSelectionStateMap(prev => {
      const next = { ...prev }
      const existing = next[snapshot.id]
      next[snapshot.id] = {
        id: snapshot.id,
        name: snapshot.name,
        inBatch: existing?.inBatch ?? snapshot.inBatch,
        selected: true,
        releaseNotes: existing?.releaseNotes ?? snapshot.releaseNotes ?? '',
      }
      return next
    })
  }, [])

  const handleSelectAllChange = useCallback((checked: boolean) => {
    const currentApps = appSelections
    setAppSelections(prev => prev.map(app => ({ ...app, selected: checked })))
    setSelectionStateMap(prev => {
      const next = { ...prev }
      currentApps.forEach(app => {
        next[app.id] = {
          id: app.id,
          name: app.name,
          selected: checked,
          inBatch: next[app.id]?.inBatch ?? app.inBatch,
          releaseNotes: next[app.id]?.releaseNotes ?? '',
        }
      })
      return next
    })
  }, [appSelections])

  const updateAppReleaseNotes = useCallback((appId: number, notes: string) => {
    setAppSelections((prev) =>
      prev.map((app) =>
        app.id === appId ? { ...app, releaseNotes: notes } : app
      )
    )
    setSelectionStateMap((prev) => {
      if (!prev[appId]) {
        return prev
      }
      return {
        ...prev,
        [appId]: {
          ...prev[appId],
          releaseNotes: notes,
        },
      }
    })
  }, [])

  // 保存批次修改
  const handleUpdate = async () => {
    if (!batch || !batchDetail) return

    try {
      const formValues = await form.validateFields()

      // 计算应用变更
      const addedAppIds = selectedAppEntries.filter(item => !item.inBatch).map(item => item.id)
      const removedAppIds = removedAppEntries.map(item => item.id)

      // 构建请求数据
      const requestData: UpdateBatchRequest = {
        batch_id: batch.id,
        operator: user?.username || 'unknown',
      }

      // 添加基本信息变更
      if (formValues.batch_number !== batch.batch_number) {
        requestData.batch_number = formValues.batch_number
      }
      if (formValues.release_notes !== (batch.release_notes || '')) {
        requestData.release_notes = formValues.release_notes?.trim() || ''
      }

      // 添加应用变更
      if (addedAppIds.length > 0) {
        requestData.add_apps = addedAppIds.map(appId => ({ app_id: appId }))
      }
      if (removedAppIds.length > 0) {
        requestData.remove_app_ids = removedAppIds
      }

      // 检查是否有变化
      if (
        !requestData.batch_number &&
        requestData.release_notes === undefined &&
        !requestData.add_apps &&
        !requestData.remove_app_ids
      ) {
        message.info('没有需要更新的内容')
        return
      }

      console.log('Updating batch with data:', requestData)
      updateMutation.mutate(requestData)
    } catch (error) {
      console.error('Validation failed:', error)
    }
  }

  // 应用列表列定义
  const appColumns: ColumnsType<AppWithSelection> = [
    {
      title: (
        <Checkbox
          checked={appSelections.every(app => app.selected)}
          indeterminate={
            appSelections.some(app => app.selected) &&
            !appSelections.every(app => app.selected)
          }
          onChange={(e) => handleSelectAllChange(e.target.checked)}
        />
      ),
      width: 30,
      render: (_: any, record: AppWithSelection) => (
        <Checkbox
          checked={record.selected}
          onChange={() => toggleAppSelection(record)}
        />
      ),
    },
    {
      title: t('batch.appName'),
      dataIndex: 'name',
      key: 'name',
      width: 200,
      ellipsis: true,
      filteredValue: searchKeyword ? [searchKeyword] : null,
      onFilter: (value, record) => {
        const keyword = String(value).toLowerCase()
        return (
          record.name.toLowerCase().includes(keyword) ||
          record.display_name?.toLowerCase().includes(keyword) ||
          record.repo_name?.toLowerCase().includes(keyword) ||
          record.commit_message?.toLowerCase().includes(keyword) ||
          record.commit_sha?.toLowerCase().includes(keyword) ||
          record.image_tag?.toLowerCase().includes(keyword)
        )
      },
      render: (text: string, record: AppWithSelection) => (
        <Space
          style={{ cursor: 'pointer' }}
          onClick={() => toggleAppSelection(record)}
        >
          <span>{text}</span>
          {record.inBatch && <Tag color="green">原有</Tag>}
        </Space>
      ),
    },
    {
      title: t('batch.appType'),
      dataIndex: 'app_type',
      key: 'app_type',
      width: 80,
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    {
      title: t('application.repository'),
      dataIndex: 'repo_full_name',
      key: 'repo_full_name',
      width: 200,
      ellipsis: true,
    },
    {
      title: t('batch.currentVersion'),
      dataIndex: 'deployed_tag',
      key: 'deployed_tag',
      width: 150,
      ellipsis: true,
      render: (tag: string) => tag || <Tag color="default">-</Tag>,
    },
    {
      title: t('build.latestBuild'),
      key: 'latest_version',
      width: 180,
      ellipsis: true,
      render: (_: any, record: AppWithSelection) => {
        if (record.image_tag) {
          return (
            <div>
              <code style={{ fontSize: 11 }}>{record.image_tag}</code>
              {record.commit_message && (
                <div style={{ fontSize: 11, color: '#8c8c8c', marginTop: 2 }}>
                  {record.commit_message.substring(0, 30)}
                  {record.commit_message.length > 30 && '...'}
                </div>
              )}
            </div>
          )
        }
        return <Tag color="default">-</Tag>
      },
    },
  ]

  const selectedAppEntries = useMemo(
    () => Object.values(selectionStateMap).filter(item => item.selected),
    [selectionStateMap]
  )
  const removedAppEntries = useMemo(
    () => Object.values(selectionStateMap).filter(item => item.inBatch && !item.selected),
    [selectionStateMap]
  )
  const selectedCount = selectedAppEntries.length
  const addedCount = useMemo(
    () => selectedAppEntries.filter(item => !item.inBatch).length,
    [selectedAppEntries]
  )
  const removedCount = removedAppEntries.length

  // 底部按钮
  const renderFooter = () => {
    return (
      <Space>
        <Button onClick={handleClose}>
          {t('common.cancel')}
        </Button>
        <Button
          type="primary"
          icon={<SaveOutlined />}
          loading={updateMutation.isPending}
          onClick={handleUpdate}
        >
          {t('common.save')}
        </Button>
      </Space>
    )
  }

  if (!batch) return null

  return (
    <Drawer
      title={t('batch.edit')}
      placement="right"
      width="70%"
      open={open}
      onClose={handleClose}
      destroyOnClose={false}
      maskClosable={false}
      footer={renderFooter()}
      footerStyle={{ textAlign: 'right' }}
      className="batch-edit-drawer"
    >
      {/* 步骤指示器 */}
      <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 0, paddingTop: 24, paddingLeft: 24, paddingRight: 24 }}>
        <Steps
          current={currentStep}
          onChange={handleStepChange}
          items={[
            {
              title: t('batch.step1'),
              icon: <FormOutlined />,
            },
            {
              title: t('batch.step2'),
              icon: <AppstoreOutlined />,
            },
          ]}
          style={{ maxWidth: 500 }}
        />
      </div>

      <Form
        form={form}
        layout="vertical"
      >
        {/* 步骤1: 基本信息 */}
        <div style={{ display: currentStep === 0 ? 'block' : 'none' }}>
          <Form.Item
            name="batch_number"
            label={t('batch.batchNumber')}
            rules={[{ required: true, message: '请输入批次编号' }]}
          >
            <Input
              placeholder={t('batch.batchNumberPlaceholder')}
              size="large"
            />
          </Form.Item>

          <Form.Item
            name="release_notes"
            label={t('batch.releaseNotes')}
          >
            <TextArea
              rows={8}
              placeholder={t('batch.releaseNotesPlaceholder')}
            />
          </Form.Item>
        </div>

        {/* 步骤2: 应用管理 */}
        <div style={{ display: currentStep === 1 ? 'block' : 'none' }}>
          {/* 固定顶部区域：统计信息、搜索框和分页器 */}
          <div
            className="sticky-top-bar"
            style={{
              position: 'sticky',
              top: 0,
              zIndex: 10,
              background: '#fff',
              paddingBottom: 12,
              marginBottom: 12,
              marginLeft: -24,
              marginRight: -24,
              paddingLeft: 24,
              paddingRight: 24,
              paddingTop: 12,
              marginTop: 0
            }}
          >
            {/* 变更统计 */}
            <Alert
              message={
                <div>
                  <div style={{ fontWeight: 500, marginBottom: 8 }}>
                    已选择 {selectedCount} 个应用
                    {addedCount > 0 && <Tag color="green" style={{ marginLeft: 8 }}>新增 {addedCount}</Tag>}
                    {removedCount > 0 && <Tag color="red" style={{ marginLeft: 8 }}>移除 {removedCount}</Tag>}
                  </div>

                  {/* 已选应用列表 */}
                  {selectedCount > 0 && (
                    <div style={{ marginBottom: removedCount > 0 ? 12 : 0 }}>
                      <div style={{ fontSize: 12, color: '#8c8c8c', marginBottom: 4 }}>已选应用：</div>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                        {selectedAppEntries.map(app => (
                          <Tag
                            key={app.id}
                            color={app.inBatch ? 'blue' : 'green'}
                            closable
                            onClose={(e) => {
                              e.preventDefault()
                              handleDeselectSnapshot(app)
                            }}
                          >
                            {app.name} {!app.inBatch && '(新)'}
                          </Tag>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* 已移除应用列表 */}
                  {removedCount > 0 && (
                    <div>
                      <div style={{ fontSize: 12, color: '#8c8c8c', marginBottom: 4 }}>已移除应用：</div>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                        {removedAppEntries.map(app => (
                          <Tag
                            key={app.id}
                            color="red"
                            closable
                            onClose={(e) => {
                              e.preventDefault()
                              handleReselectSnapshot(app)
                            }}
                          >
                            {app.name}
                          </Tag>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              }
              type={selectedCount > 0 ? 'success' : 'info'}
              showIcon={false}
              style={{ marginBottom: 12 }}
            />

            {/* 搜索框和分页器 */}
            <div className="search-pagination-wrapper">
              <Input.Search
                placeholder="搜索应用名称、代码库、Commit、Tag..."
                allowClear
                style={{ width: 400, minWidth: 280 }}
                value={searchKeyword}
                onChange={(e) => setSearchKeyword(e.target.value)}
              />
              <div style={{ flex: 1 }} />
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexShrink: 0 }}>
                <span style={{ fontSize: 13, color: '#8c8c8c', whiteSpace: 'nowrap' }}>
                  共 {allAppsResponse?.total || 0} 个
                </span>
                <Space size={4}>
                  <Button
                    size="small"
                    icon={<LeftOutlined />}
                    disabled={currentPage === 1}
                    onClick={() => setCurrentPage(currentPage - 1)}
                  />
                  <span style={{ fontSize: 13, whiteSpace: 'nowrap', padding: '0 4px' }}>
                    {currentPage} / {Math.ceil((allAppsResponse?.total || 0) / pageSize)}
                  </span>
                  <Button
                    size="small"
                    icon={<RightOutlined />}
                    disabled={currentPage >= Math.ceil((allAppsResponse?.total || 0) / pageSize)}
                    onClick={() => setCurrentPage(currentPage + 1)}
                  />
                  <Select
                    size="small"
                    value={pageSize}
                    onChange={(value) => {
                      setPageSize(value)
                      setCurrentPage(1)
                    }}
                    style={{ width: 90 }}
                    options={[
                      { label: '10/页', value: 10 },
                      { label: '20/页', value: 20 },
                      { label: '50/页', value: 50 },
                      { label: '100/页', value: 100 },
                    ]}
                  />
                </Space>
              </div>
            </div>
          </div>

          {/* 应用表格 */}
          <Table
            columns={appColumns}
            dataSource={appSelections}
            rowKey="id"
            loading={loadingBatchDetail || loadingAllApps}
            pagination={false}
            size="small"
            scroll={{ x: 900 }}
            expandable={{
              expandedRowRender: (record) => (
                <div style={{ padding: '12px 24px' }}>
                  <div style={{ marginBottom: 8 }}>
                    <strong>{t('batch.appReleaseNotes')}:</strong>
                  </div>
                  <TextArea
                    rows={3}
                    placeholder={t('batch.appReleaseNotesPlaceholder')}
                    value={record.releaseNotes}
                    onChange={(e) =>
                      updateAppReleaseNotes(record.id, e.target.value)
                    }
                  />
                </div>
              ),
              rowExpandable: (record) => record.selected,
              expandedRowKeys: expandedRowKeys,
              onExpand: (expanded, record) => {
                if (expanded) {
                  setExpandedRowKeys([...expandedRowKeys, record.id])
                } else {
                  setExpandedRowKeys(expandedRowKeys.filter((key) => key !== record.id))
                }
              },
            }}
          />
        </div>
      </Form>
    </Drawer>
  )
}
