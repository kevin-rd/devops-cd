import React, { useState } from 'react'
import {
  Card,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  message,
  Popconfirm,
  Tooltip,
  Tag,
  Pagination,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  FolderOutlined,
  AppstoreOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ReloadOutlined,
  LinkOutlined,
  HistoryOutlined,
} from '@ant-design/icons'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { repositoryService } from '@/services/repository'
import { applicationService } from '@/services/application'
import BuildHistoryDrawer from '@/components/BuildHistoryDrawer'
import type { Repository, Application } from '@/types'
import './index.css'

const RepositoryPage: React.FC = () => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [repoForm] = Form.useForm()
  const [appForm] = Form.useForm()

  const [repoModalVisible, setRepoModalVisible] = useState(false)
  const [appModalVisible, setAppModalVisible] = useState(false)
  const [editingRepo, setEditingRepo] = useState<Repository | null>(null)
  const [editingApp, setEditingApp] = useState<Application | null>(null)
  const [selectedRepoId, setSelectedRepoId] = useState<number | null>(null)
  const [expandedRowKeys, setExpandedRowKeys] = useState<React.Key[]>([])
  
  // 分页状态
  const [repoPage, setRepoPage] = useState(1)
  const [repoPageSize, setRepoPageSize] = useState(10)

  // 构建历史 Drawer 状态
  const [buildDrawerVisible, setBuildDrawerVisible] = useState(false)
  const [selectedAppId, setSelectedAppId] = useState<number | null>(null)
  const [selectedAppName, setSelectedAppName] = useState('')

  // 查询代码库列表（包含应用）
  const { data: repoResponse, isLoading: repoLoading } = useQuery({
    queryKey: ['repositories', repoPage, repoPageSize],
    queryFn: async () => {
      const res = await repositoryService.getList({
        page: repoPage,
        page_size: repoPageSize,
        with_applications: true,  // 请求包含应用列表
      })
      return res.data
    },
  })

  const repoData = repoResponse?.items || []
  const repoTotal = repoResponse?.total || 0

  // 查询应用类型列表（永久缓存，页面加载时获取一次）
  const { data: appTypesResponse } = useQuery({
    queryKey: ['applicationTypes'],
    queryFn: async () => {
      const res = await applicationService.getTypes()
      return res.data
    },
    staleTime: Infinity,  // 数据永不过期
    gcTime: Infinity,  // 永久缓存（garbage collection time）
  })

  const appTypes = appTypesResponse?.types ?? []

  // 根据 app_type 值获取类型配置
  const getAppTypeConfig = (appType: string) => {
    return appTypes.find(type => type.value === appType)
  }

  // 创建/更新代码库
  const repoMutation = useMutation({
    mutationFn: async (values: any) => {
      if (editingRepo) {
        return await repositoryService.update(editingRepo.id, values)
      } else {
        return await repositoryService.create(values)
      }
    },
    onSuccess: () => {
      message.success(
        editingRepo ? t('repository.updateSuccess') : t('repository.createSuccess')
      )
      setRepoModalVisible(false)
      repoForm.resetFields()
      setEditingRepo(null)
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
    },
  })

  // 删除代码库
  const deleteRepoMutation = useMutation({
    mutationFn: (id: number) => repositoryService.delete(id),
    onSuccess: () => {
      message.success(t('repository.deleteSuccess'))
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
    },
  })

  // 创建/更新应用
  const appMutation = useMutation({
    mutationFn: async (values: any) => {
      if (editingApp) {
        return await applicationService.update(editingApp.id, values)
      } else {
        return await applicationService.create(values)
      }
    },
    onSuccess: () => {
      message.success(
        editingApp ? t('application.updateSuccess') : t('application.createSuccess')
      )
      setAppModalVisible(false)
      appForm.resetFields()
      setEditingApp(null)
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
    },
  })

  // 删除应用
  const deleteAppMutation = useMutation({
    mutationFn: (id: number) => applicationService.delete(id),
    onSuccess: () => {
      message.success(t('application.deleteSuccess'))
      queryClient.invalidateQueries({ queryKey: ['applications'] })
    },
  })

  // 处理函数
  const handleCreateRepo = () => {
    setEditingRepo(null)
    repoForm.resetFields()
    setRepoModalVisible(true)
  }

  const handleEditRepo = (repo: Repository) => {
    setEditingRepo(repo)
    repoForm.setFieldsValue(repo)
    setRepoModalVisible(true)
  }

  const handleCreateApp = (repoId: number) => {
    setEditingApp(null)
    setSelectedRepoId(repoId)
    
    // 找到当前 repo
    const currentRepo = repoData.find(repo => repo.id === repoId)
    
    // 检查该 repo 是否已有应用
    const hasApps = (currentRepo?.applications?.length || 0) > 0
    
    appForm.resetFields()
    appForm.setFieldsValue({ 
      repo_id: repoId,
      name: hasApps ? '' : currentRepo?.name,  // 如果没有应用，默认使用 repo 名称
    })
    setAppModalVisible(true)
  }

  const handleEditApp = (app: Application) => {
    setEditingApp(app)
    setSelectedRepoId(app.repo_id)
    appForm.setFieldsValue(app)
    setAppModalVisible(true)
  }

  const handleRepoSubmit = () => {
    repoForm.validateFields().then((values) => {
      repoMutation.mutate(values)
    })
  }

  const handleAppSubmit = () => {
    appForm.validateFields().then((values) => {
      appMutation.mutate(values)
    })
  }

  // 查看构建历史
  const handleViewBuilds = (app: Application) => {
    setSelectedAppId(app.id)
    setSelectedAppName(app.display_name || app.name)
    setBuildDrawerVisible(true)
  }

  // Repository 表格列定义
  const repoColumns: ColumnsType<Repository> = [
    {
      title: t('repository.name'),
      dataIndex: 'name',
      key: 'name',
      width: 300,
      render: (text, record) => (
        <Space>
          <FolderOutlined style={{ color: '#1890ff' }} />
          <span className="repo-name">{text}</span>
          {record.git_url && (
            <Tooltip title={record.git_url}>
              <a
                href={record.git_url}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
              >
                <LinkOutlined style={{ fontSize: 13, color: '#1890ff' }} />
              </a>
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: t('repository.project'),
      dataIndex: 'project',
      key: 'project',
      width: 150,
      render: (text) => text && <Tag color="blue">{text}</Tag>,
    },
    {
      title: t('repository.gitType'),
      dataIndex: 'git_type',
      key: 'git_type',
      width: 120,
      render: (text) => <Tag color="cyan">{text}</Tag>,
    },
    {
      title: t('application.list'),
      key: 'app_count',
      width: 150,
      render: (_, record) => {
        const appCount = record.applications?.length || 0
        return (
          <span className="app-count">
            <AppstoreOutlined style={{ fontSize: 12, marginRight: 4 }} />
            {appCount} 个应用
          </span>
        )
      },
    },
    {
      title: t('common.action'),
      key: 'action',
      width: 150,
      render: (_, record) => (
        <Space size="small" onClick={(e) => e.stopPropagation()}>
          <Tooltip title={t('application.create')}>
            <Button
              type="text"
              size="small"
              icon={<PlusOutlined />}
              onClick={() => handleCreateApp(record.id)}
            />
          </Tooltip>
          <Tooltip title={t('common.edit')}>
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEditRepo(record)}
            />
          </Tooltip>
          <Popconfirm
            title={t('repository.deleteConfirm')}
            onConfirm={() => deleteRepoMutation.mutate(record.id)}
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
            />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  // Application 子表格列定义
  const appColumns: ColumnsType<Application> = [
    {
      title: t('application.name'),
      dataIndex: 'name',
      key: 'name',
      width: 300,
      render: (text) => (
        <Space style={{ paddingLeft: 24 }}>
          <AppstoreOutlined style={{ color: '#52c41a' }} />
          <span>{text}</span>
        </Space>
      ),
    },
    {
      title: '',
      key: 'empty1',
      width: 150,
    },
    {
      title: t('application.appType'),
      dataIndex: 'app_type',
      key: 'app_type',
      width: 120,
      render: (appType: string) => {
        const typeConfig = getAppTypeConfig(appType)
        if (typeConfig) {
          return (
            <Tag color={typeConfig.color}>
              <Space size={4}>
                <span>●</span>
                <span>{typeConfig.label}</span>
              </Space>
            </Tag>
          )
        }
        // 如果找不到配置，使用默认样式
        return <Tag color="default">{appType}</Tag>
      },
    },
    {
      title: t('application.lastTag'),
      dataIndex: 'last_tag',
      key: 'last_tag',
      width: 150,
      render: (text) => text && <Tag color="purple">{text}</Tag>,
    },
    {
      title: t('common.action'),
      key: 'action',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title={t('application.viewBuilds')}>
            <Button
              type="text"
              size="small"
              icon={<HistoryOutlined />}
              onClick={(e) => {
                e.stopPropagation()
                handleViewBuilds(record)
              }}
            />
          </Tooltip>
          <Tooltip title={t('common.edit')}>
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEditApp(record)}
            />
          </Tooltip>
          <Popconfirm
            title={t('application.deleteConfirm')}
            onConfirm={() => deleteAppMutation.mutate(record.id)}
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
            />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="repository-page">
      <Card
        title={
          <Space>
            <FolderOutlined />
            <span>{t('repository.title')}</span>
          </Space>
        }
        extra={
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
      queryClient.invalidateQueries({ queryKey: ['applications'] })  // 保留以刷新其他可能的应用查询
              }}
            >
              {t('common.reset')}
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={handleCreateRepo}
            >
              {t('repository.create')}
            </Button>
          </Space>
        }
      >
        <Table
          columns={repoColumns}
          dataSource={repoData}
          rowKey="id"
          loading={repoLoading}
          pagination={false}
          expandable={{
            expandedRowKeys,
            onExpandedRowsChange: (keys) => setExpandedRowKeys(keys as React.Key[]),
            expandRowByClick: true,
            showExpandColumn: false,
            expandedRowRender: (record) => {
              const apps = record.applications || []
              return (
                <Table
                  columns={appColumns}
                  dataSource={apps}
                  rowKey="id"
                  pagination={false}
                  showHeader={false}
                  size="small"
                  className="app-table"
                />
              )
            },
            rowExpandable: (record) => {
              return (record.applications?.length || 0) > 0
            },
          }}
          onRow={() => ({
            style: { cursor: 'pointer' },
          })}
        />

        {repoTotal > repoPageSize && (
          <div style={{ marginTop: 16, textAlign: 'right' }}>
            <Pagination
              current={repoPage}
              pageSize={repoPageSize}
              total={repoTotal}
              onChange={(page, pageSize) => {
                setRepoPage(page)
                setRepoPageSize(pageSize)
              }}
              showSizeChanger
              showQuickJumper
              showTotal={(total) => `${t('common.total')} ${total} ${t('repository.list')}`}
            />
          </div>
        )}
      </Card>

      {/* Repository Modal */}
      <Modal
        title={editingRepo ? t('repository.edit') : t('repository.create')}
        open={repoModalVisible}
        onOk={handleRepoSubmit}
        onCancel={() => {
          setRepoModalVisible(false)
          setEditingRepo(null)
          repoForm.resetFields()
        }}
        confirmLoading={repoMutation.isPending}
        width={600}
      >
        <Form form={repoForm} layout="vertical">
          <Form.Item
            name="name"
            label={t('repository.name')}
            rules={[{ required: true }]}
          >
            <Input placeholder="my-repo" />
          </Form.Item>

          <Form.Item
            name="project"
            label={t('repository.project')}
            rules={[{ required: true }]}
          >
            <Input placeholder="my-project" />
          </Form.Item>

          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea rows={3} />
          </Form.Item>

          <Form.Item
            name="git_url"
            label={t('repository.gitUrl')}
            rules={[{ required: true }]}
          >
            <Input placeholder="https://gitea.company.com/team/project.git" />
          </Form.Item>

          <Form.Item
            name="git_type"
            label={t('repository.gitType')}
            rules={[{ required: true }]}
            initialValue="gitea"
          >
            <Select>
              <Select.Option value="gitea">Gitea</Select.Option>
              <Select.Option value="gitlab">GitLab</Select.Option>
              <Select.Option value="github">GitHub</Select.Option>
            </Select>
          </Form.Item>

          <Form.Item name="git_token" label={t('repository.gitToken')}>
            <Input.Password placeholder="Optional" />
          </Form.Item>

          <Form.Item
            name="default_branch"
            label={t('repository.defaultBranch')}
            initialValue="main"
          >
            <Input />
          </Form.Item>

          <Form.Item name="language" label={t('repository.language')}>
            <Input placeholder="Go, Java, Python, etc." />
          </Form.Item>
        </Form>
      </Modal>

      {/* Application Modal */}
      <Modal
        title={editingApp ? t('application.edit') : t('application.create')}
        open={appModalVisible}
        onOk={handleAppSubmit}
        onCancel={() => {
          setAppModalVisible(false)
          setEditingApp(null)
          setSelectedRepoId(null)
          appForm.resetFields()
        }}
        confirmLoading={appMutation.isPending}
        width={600}
      >
        <Form form={appForm} layout="vertical">
          <Form.Item
            name="repo_id"
            label={t('application.repository')}
            rules={[{ required: true }]}
          >
            <Select disabled>
              {repoData?.map((repo) => (
                <Select.Option key={repo.id} value={repo.id}>
                  {repo.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="name"
            label={t('application.name')}
            rules={[{ required: true }]}
          >
            <Input placeholder="my-service" />
          </Form.Item>

          <Form.Item name="display_name" label={t('application.displayName')}>
            <Input placeholder="My Service" />
          </Form.Item>

          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea rows={3} />
          </Form.Item>

          <Form.Item
            name="app_type"
            label={t('application.appType')}
            rules={[{ required: true }]}
          >
            <Select placeholder={t('application.appType')}>
              {appTypes.map((type: any) => (
                <Select.Option key={type.value} value={type.value}>
                  <Space>
                    <span style={{ color: type.color }}>●</span>
                    <span>{type.label}</span>
                    {type.description && (
                      <span style={{ color: '#999', fontSize: '12px' }}>
                        ({type.description})
                      </span>
                    )}
                  </Space>
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* Build History Drawer */}
      <BuildHistoryDrawer
        open={buildDrawerVisible}
        appId={selectedAppId}
        appName={selectedAppName}
        onClose={() => {
          setBuildDrawerVisible(false)
          setSelectedAppId(null)
          setSelectedAppName('')
        }}
      />
    </div>
  )
}

export default RepositoryPage

