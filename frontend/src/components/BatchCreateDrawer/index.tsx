import { useState, useEffect, useMemo, useCallback } from 'react'
import {
  Drawer,
  Form,
  Input,
  Button,
  Steps,
  Space,
  Table,
  Checkbox,
  Tag,
  message,
  Alert,
  Modal,
  Typography,
  Select,
} from 'antd'
import { CheckCircleOutlined, FormOutlined, AppstoreOutlined, ExclamationCircleOutlined, LeftOutlined, RightOutlined } from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import type { ColumnsType } from 'antd/es/table'
import { batchService } from '@/services/batch'
import { applicationService } from '@/services/application'
import { useAuthStore } from '@/stores/authStore'
import type { ApplicationWithBuild, CreateBatchRequest } from '@/types'
import './index.css'
import axios from 'axios'

const { TextArea } = Input
const { Paragraph } = Typography

interface AppSelection extends ApplicationWithBuild {
  selected: boolean
  releaseNotes?: string
}

interface SelectedAppSnapshot {
  id: number
  name: string
  releaseNotes?: string
}

interface AppConflict {
  app_id: number
  app_name: string
  app_project: string
  batch_id: number
  batch_number: string
  batch_status: number
  batch_status_name: string
}

interface BatchCreateDrawerProps {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

export default function BatchCreateDrawer({ open, onClose, onSuccess }: BatchCreateDrawerProps) {
  const { t } = useTranslation()
  const [form] = Form.useForm()
  const { user } = useAuthStore()
  const queryClient = useQueryClient()
  
  const [currentStep, setCurrentStep] = useState(0)
  const [appSelections, setAppSelections] = useState<AppSelection[]>([])
  const [searchKeyword, setSearchKeyword] = useState('')
  const [debouncedSearchKeyword, setDebouncedSearchKeyword] = useState('')
  const [expandedRowKeys, setExpandedRowKeys] = useState<number[]>([])
  const [pageSize, setPageSize] = useState(20)
  const [currentPage, setCurrentPage] = useState(1)
  const [selectedAppsMap, setSelectedAppsMap] = useState<Record<number, SelectedAppSnapshot>>({})

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
    setCurrentPage((prev) => (prev === 1 ? prev : 1))
  }, [debouncedSearchKeyword])

  // 查询应用列表（包含构建信息，服务端分页）
  const { data: appResponse, isLoading: loadingApps, error: loadingAppsError } = useQuery({
    queryKey: ['applicationsWithBuilds', debouncedSearchKeyword, currentPage, pageSize],
    queryFn: async ({ signal }) => {
      const res = await applicationService.searchWithBuilds({
        page: currentPage,
        page_size: pageSize,
        keyword: debouncedSearchKeyword || undefined,
      }, signal)
      return res.data
    },
    staleTime: 30 * 1000,
    gcTime: 60 * 1000,
    enabled: open && searchKeyword === debouncedSearchKeyword,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    meta: {
      errorMessage: false,
    },
  })

  const applications = appResponse?.items || []
  const totalCount = appResponse?.total || 0

  // 显示错误提示（仅在有错误时）
  useEffect(() => {
    const errorMessageKey = 'batch-create-apps-fetch-error'
    const isCanceled = loadingAppsError && axios.isCancel && axios.isCancel(loadingAppsError)
    const isAbortError = loadingAppsError instanceof DOMException && loadingAppsError.name === 'AbortError'

    if (loadingAppsError && open && !isCanceled && !isAbortError) {
      console.error('加载应用列表失败:', loadingAppsError)
      message.error({ content: '加载应用列表失败，请稍后重试', key: errorMessageKey })
    } else {
      message.destroy(errorMessageKey)
    }
  }, [loadingAppsError, open])

  // 创建批次 Mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateBatchRequest) => batchService.create(data),
    onSuccess: () => {
      message.success(t('batch.createSuccess'))
      queryClient.invalidateQueries({ queryKey: ['batchList'] })
      handleClose()
      onSuccess()
    },
    onError: (error: any) => {
      // 检查是否是应用冲突错误（409）
      if (error.response?.status === 409 && error.response?.data?.data?.conflicts) {
        showConflictModal(error.response.data.data.conflicts, error.response.data.message)
        return
      }
      
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

  // 关闭并重置
  const handleClose = () => {
    form.resetFields()
    setCurrentStep(0)
    setAppSelections([])
    setSearchKeyword('')
    setDebouncedSearchKeyword('')
    setExpandedRowKeys([])
    setPageSize(20)
    setCurrentPage(1)
    setSelectedAppsMap({})
    onClose()
  }

  // 步骤切换处理 - 验证必填字段
  const handleStepChange = (step: number) => {
    // 如果要切换到步骤2（应用管理），先验证基本信息
    if (step === 1) {
      form.validateFields(['batch_number'])
        .then(() => {
          // 重置分页状态
          setCurrentPage(1)
          setPageSize(20)

          // 初始化应用选择列表
          if (appSelections.length === 0) {
            setAppSelections(
              applications.map((app) => ({
                ...app,
                selected: false,
                releaseNotes: '',
              }))
            )
          }

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

  // 步骤1：提交基本信息
  const handleStep1Submit = async () => {
    handleStepChange(1)
  }

  // 步骤2：创建批次
  const handleCreate = async () => {
    const selectedApps = selectedAppEntries

    if (selectedApps.length === 0) {
      message.warning(t('batch.selectAtLeastOneApp'))
      return
    }

    const formValues = form.getFieldsValue()
    
    // 构建请求数据
    const requestData: any = {
      batch_number: formValues.batch_number,
      initiator: formValues.initiator || user?.username || 'unknown',
      apps: selectedApps.map((app) => ({
        app_id: app.id,
      })),
    }
    
    // 添加可选字段
    if (formValues.release_notes && formValues.release_notes.trim()) {
      requestData.release_notes = formValues.release_notes.trim()
    }
    
    // 添加应用级发布说明
    requestData.apps = requestData.apps.map((app: any, index: number) => {
      const originalApp = selectedApps[index]
      const releaseNotes = originalApp.releaseNotes?.trim()
      if (releaseNotes) {
        return {
          ...app,
          release_notes: releaseNotes,
        }
      }
      return app
    })

    console.log('Creating batch with data:', requestData)
    createMutation.mutate(requestData)
  }

  const handleSelectionChange = useCallback((record: AppSelection, selected: boolean) => {
    setAppSelections((prev) =>
      prev.map((app) =>
        app.id === record.id ? { ...app, selected } : app
      )
    )
    setSelectedAppsMap((prev) => {
      const next = { ...prev }
      if (selected) {
        next[record.id] = {
          id: record.id,
          name: record.name,
          releaseNotes: record.releaseNotes ?? prev[record.id]?.releaseNotes ?? '',
        }
      } else {
        delete next[record.id]
      }
      return next
    })
  }, [])

  const toggleAppSelection = useCallback((record: AppSelection) => {
    handleSelectionChange(record, !record.selected)
  }, [handleSelectionChange])

  const handleDeselectSnapshot = useCallback((snapshot: SelectedAppSnapshot) => {
    setAppSelections((prev) =>
      prev.map((app) =>
        app.id === snapshot.id ? { ...app, selected: false } : app
      )
    )
    setSelectedAppsMap((prev) => {
      const next = { ...prev }
      delete next[snapshot.id]
      return next
    })
  }, [])

  const handleSelectAllChange = useCallback((checked: boolean) => {
    const currentApps = appSelections
    setAppSelections((prev) => prev.map((app) => ({ ...app, selected: checked })))
    setSelectedAppsMap((prev) => {
      const next = { ...prev }
      if (checked) {
        currentApps.forEach((app) => {
          next[app.id] = {
            id: app.id,
            name: app.name,
            releaseNotes: next[app.id]?.releaseNotes ?? app.releaseNotes ?? '',
          }
        })
      } else {
        currentApps.forEach((app) => {
          delete next[app.id]
        })
      }
      return next
    })
  }, [appSelections])

  const updateAppReleaseNotes = useCallback((appId: number, notes: string) => {
    setAppSelections((prev) =>
      prev.map((app) =>
        app.id === appId ? { ...app, releaseNotes: notes } : app
      )
    )
    setSelectedAppsMap((prev) => {
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

  // 当应用列表数据变化时，更新选择状态
  useEffect(() => {
    if (currentStep === 1 && applications.length > 0) {
      setAppSelections((prev) => {
        const prevSelectionMap = new Map(prev.map(app => [app.id, app]))

        return applications.map(app => {
          const prevApp = prevSelectionMap.get(app.id)
          const snapshot = selectedAppsMap[app.id]
          return {
            ...app,
            selected: !!snapshot,
            releaseNotes: snapshot?.releaseNotes ?? prevApp?.releaseNotes ?? '',
          }
        })
      })
    }
  }, [applications, currentStep, selectedAppsMap])

  // 显示应用冲突 Modal
  const showConflictModal = (conflicts: AppConflict[], errorMessage: string) => {
    const conflictColumns: ColumnsType<AppConflict> = [
      {
        title: '应用名称',
        dataIndex: 'app_name',
        key: 'app_name',
        width: 200,
        ellipsis: true,
        render: (text: string) => <strong>{text}</strong>,
      },
      {
        title: '所属项目',
        dataIndex: 'app_project',
        key: 'app_project',
        width: 120,
      },
      {
        title: '冲突批次',
        dataIndex: 'batch_number',
        key: 'batch_number',
        width: 150,
        render: (text: string, record: AppConflict) => (
          <div>
            <div>{text}</div>
            <div style={{ fontSize: 12, color: '#8c8c8c' }}>ID: {record.batch_id}</div>
          </div>
        ),
      },
      {
        title: '批次状态',
        dataIndex: 'batch_status_name',
        key: 'batch_status_name',
        width: 100,
        render: (text: string, record: AppConflict) => (
          <Tag color={record.batch_status === 0 ? 'orange' : 'blue'}>{text}</Tag>
        ),
      },
    ]

    Modal.warning({
      title: (
        <Space>
          <ExclamationCircleOutlined style={{ color: '#faad14' }} />
          <span>应用冲突</span>
        </Space>
      ),
      width: 700,
      content: (
        <div style={{ marginTop: 16 }}>
          <Alert
            message={errorMessage}
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
          
          <Paragraph style={{ marginBottom: 12 }}>
            以下应用已存在于其他批次中，请处理后再创建：
          </Paragraph>
          
          <Table
            columns={conflictColumns}
            dataSource={conflicts}
            rowKey="app_id"
            pagination={false}
            size="small"
            style={{ marginBottom: 16 }}
          />
          
          <Alert
            message="处理建议"
            description={
              <ul style={{ marginBottom: 0, paddingLeft: 20 }}>
                <li>从当前批次中取消选择这些应用</li>
                <li>或者先取消/完成冲突的批次，再创建新批次</li>
              </ul>
            }
            type="info"
            showIcon
          />
        </div>
      ),
      okText: '我知道了',
      onOk: () => {
        // 自动取消选择冲突的应用
        const conflictAppIds = conflicts.map(c => c.app_id)
        setAppSelections(prev =>
          prev.map(app =>
            conflictAppIds.includes(app.id)
              ? { ...app, selected: false }
              : app
          )
        )
        setSelectedAppsMap(prev => {
          const next = { ...prev }
          conflictAppIds.forEach(id => {
            delete next[id]
          })
          return next
        })
        message.info(`已自动取消选择 ${conflicts.length} 个冲突应用`)
      },
    })
  }

  // 应用列表列定义
  const appColumns: ColumnsType<AppSelection> = [
    {
      title: (
        <Checkbox
          checked={appSelections.every((app) => app.selected)}
          indeterminate={
            appSelections.some((app) => app.selected) &&
            !appSelections.every((app) => app.selected)
          }
          onChange={(e) => handleSelectAllChange(e.target.checked)}
        />
      ),
      width: 30,
      render: (_: any, record: AppSelection) => (
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
        return Boolean(
          record.name.toLowerCase().includes(keyword) ||
          record.display_name?.toLowerCase().includes(keyword) ||
          record.repo_name?.toLowerCase().includes(keyword) ||
          record.commit_message?.toLowerCase().includes(keyword) ||
          record.commit_sha?.toLowerCase().includes(keyword) ||
          record.image_tag?.toLowerCase().includes(keyword)
        )
      },
      render: (text: string, record: AppSelection) => (
        <span
          style={{ cursor: 'pointer' }}
          onClick={() => toggleAppSelection(record)}
        >
          {text}
        </span>
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
      render: (_: any, record: AppSelection) => {
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
        return <Tag color="default">无构建</Tag>
      },
    },
  ]

  const selectedAppEntries = useMemo(() => Object.values(selectedAppsMap), [selectedAppsMap])
  const selectedCount = selectedAppEntries.length

  // 当打开时重置表单
  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        initiator: user?.username,
      })
    }
  }, [open, user, form])

  // 底部按钮
  const renderFooter = () => {
    if (currentStep === 0) {
      return (
        <Space>
          <Button onClick={handleClose}>
            {t('common.cancel')}
          </Button>
          <Button type="primary" onClick={handleStep1Submit}>
            {t('batch.next')}
          </Button>
        </Space>
      )
    }

    return (
      <Space>
        <Button onClick={() => setCurrentStep(0)}>
          {t('batch.previous')}
        </Button>
        <Button onClick={handleClose}>
          {t('common.cancel')}
        </Button>
        <Button
          type="primary"
          icon={<CheckCircleOutlined />}
          loading={createMutation.isPending}
          onClick={handleCreate}
        >
          {t('batch.createBatch')}
        </Button>
      </Space>
    )
  }

  return (
    <Drawer
      title={t('batch.create')}
      placement="right"
      width="65%"
      open={open}
      onClose={handleClose}
      destroyOnClose={false}
      maskClosable={false}
      footer={renderFooter()}
      footerStyle={{ textAlign: 'right' }}
      className="batch-create-drawer"
    >
      {/* 步骤指示器 */}
      <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 0, paddingTop: 24, paddingLeft: 24, paddingRight: 24 }}>
        <Steps
          current={currentStep}
          onChange={handleStepChange}
          items={[
            { title: t('batch.step1'), icon: <FormOutlined /> },
            { title: t('batch.step2'), icon: <AppstoreOutlined /> },
          ]}
          style={{ maxWidth: 500 }}
        />
      </div>

      <Form
        form={form}
        layout="vertical"
        initialValues={{
          initiator: user?.username,
        }}
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
            name="initiator"
            label={t('batch.initiator')}
          >
            <Input disabled size="large" />
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

        {/* 步骤2: 选择应用 */}
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
            {/* 已选应用列表 */}
            <Alert
              message={
                <div>
                  <div style={{ fontWeight: 500, marginBottom: 8 }}>
                    已选择 {selectedCount} 个应用
                  </div>
                  {selectedCount > 0 && (
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                      {selectedAppEntries.map((app) => (
                        <Tag
                          key={app.id}
                          color="blue"
                          closable
                          onClose={(e) => {
                            e.preventDefault()
                            handleDeselectSnapshot(app)
                          }}
                        >
                          {app.name}
                        </Tag>
                      ))}
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
                  共 {totalCount || 0} 个
                </span>
                <Space size={4}>
                  <Button
                    size="small"
                    icon={<LeftOutlined />}
                    disabled={currentPage === 1}
                    onClick={() => setCurrentPage(currentPage - 1)}
                  />
                  <span style={{ fontSize: 13, whiteSpace: 'nowrap', padding: '0 4px' }}>
                    {currentPage} / {Math.ceil((totalCount || 0) / pageSize)}
                  </span>
                  <Button
                    size="small"
                    icon={<RightOutlined />}
                    disabled={currentPage >= Math.ceil((totalCount || 0) / pageSize)}
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

          <Table
            columns={appColumns}
            dataSource={appSelections}
            rowKey="id"
            loading={loadingApps}
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

