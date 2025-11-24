import React, {useState} from 'react'
import {
  Button,
  Card,
  Col,
  Form,
  Input,
  message,
  Modal,
  Pagination,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Tooltip,
} from 'antd'
import type {ColumnsType} from 'antd/es/table'
import {
  AppstoreOutlined,
  DeleteOutlined,
  EditOutlined,
  FolderOutlined,
  HistoryOutlined,
  LinkOutlined,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {useTranslation} from 'react-i18next'
import {repositoryService} from '@/services/repository'
import {applicationService} from '@/services/application'
import type {ProjectSimple} from '@/services/project'
import {projectService} from '@/services/project'
import type {TeamSimple} from '@/services/team'
import {teamService} from '@/services/team'
import BuildHistoryDrawer from '@/components/BuildHistoryDrawer'
import EnvClusterConfig from '@/components/EnvClusterConfig'
import type {ApiResponse, Application, CreateApplicationRequest, CreateRepositoryRequest, Repository,} from '@/types'
import './index.css'

interface AppTypeOption {
  value: string
  label: string
  color: string
  description?: string
}

type RepositoryFormValues = Partial<CreateRepositoryRequest>
type ApplicationFormValues = Partial<CreateApplicationRequest>

const RepositoryPage: React.FC = () => {
  const {t} = useTranslation()
  const queryClient = useQueryClient()
  const [repoForm] = Form.useForm()
  const [appForm] = Form.useForm()

  const [repoModalVisible, setRepoModalVisible] = useState(false)
  const [appModalVisible, setAppModalVisible] = useState(false)
  const [editingRepo, setEditingRepo] = useState<Repository | null>(null)
  const [editingApp, setEditingApp] = useState<Application | null>(null)
  const [expandedRowKeys, setExpandedRowKeys] = useState<React.Key[]>([])

  // 模态框中选择的项目ID（用于联动团队列表）
  const [modalProjectId, setModalProjectId] = useState<number | undefined>()
  // 应用模态框中的项目ID（用于过滤团队列表）
  const [appModalProjectId, setAppModalProjectId] = useState<number | undefined>()

  // 分页状态
  const [repoPage, setRepoPage] = useState(1)
  const [repoPageSize, setRepoPageSize] = useState(10)

  // 筛选状态
  const [keyword, setKeyword] = useState('')
  const [projectId, setProjectId] = useState<number | undefined>()
  const [teamId, setTeamId] = useState<number | undefined>()

  // 特殊值：-1 表示"无归属"
  const NO_RELATION = -1

  // 构建历史 Drawer 状态
  const [buildDrawerVisible, setBuildDrawerVisible] = useState(false)
  const [selectedAppId, setSelectedAppId] = useState<number | null>(null)
  const [selectedAppName, setSelectedAppName] = useState('')

  // 查询代码库列表（包含应用）
  const {data: repoResponse, isLoading: repoLoading} = useQuery({
    queryKey: ['repositories', repoPage, repoPageSize, keyword, projectId, teamId],
    queryFn: async () => {
      // 处理特殊值：-1 表示查询无归属的，转换为 0 或不传
      const actualProjectId = projectId === NO_RELATION ? 0 : projectId
      const actualTeamId = teamId === NO_RELATION ? 0 : teamId

      const res = await repositoryService.getList({
        page: repoPage,
        page_size: repoPageSize,
        keyword: keyword || undefined,
        project_id: actualProjectId,
        team_id: actualTeamId,
        with_applications: true,  // 请求包含应用列表
      })
      return res.data
    },
  })

  const repoData = repoResponse?.items || []
  const repoTotal = repoResponse?.total || 0

  // 查询应用类型列表（永久缓存，页面加载时获取一次）
  const {data: appTypesResponse} = useQuery({
    queryKey: ['applicationTypes'],
    queryFn: async () => {
      const res = await applicationService.getTypes()
      return res.data
    },
    staleTime: Infinity,  // 数据永不过期
    gcTime: Infinity,  // 永久缓存（garbage collection time）
  })

  const appTypes: AppTypeOption[] = appTypesResponse?.types ?? []

  // 查询所有项目（用于下拉选择）
  const {data: projectsResponse} = useQuery<ApiResponse<ProjectSimple[]>>({
    queryKey: ['projects_all'],
    queryFn: async () => {
      const res = await projectService.getAll()
      return res as unknown as ApiResponse<ProjectSimple[]>
    },
    staleTime: 60000,  // 1分钟缓存
  })

  const projects: ProjectSimple[] = projectsResponse?.data || []

  // 查询所有团队（用于下拉选择）
  const {data: teamsResponse} = useQuery<ApiResponse<TeamSimple[]>>({
    queryKey: ['teams_all'],
    queryFn: async () => {
      const res = await teamService.getList()
      return res as unknown as ApiResponse<TeamSimple[]>
    },
    staleTime: 60000,  // 1分钟缓存
  })

  const teams: TeamSimple[] = teamsResponse?.data || []

  // 根据选择的项目过滤团队列表（用于页面筛选）
  const filteredTeams = projectId && projectId !== NO_RELATION
    ? teams.filter(team => team.project_id === projectId)
    : teams

  // 根据模态框中选择的项目过滤团队列表（用于 Repository 模态框）
  const modalFilteredTeams = modalProjectId
    ? teams.filter(team => team.project_id === modalProjectId)
    : teams

  // 根据应用模态框中选择的项目过滤团队列表（用于 Application 模态框）
  const appModalFilteredTeams = appModalProjectId
    ? teams.filter(team => team.project_id === appModalProjectId)
    : teams

  // 根据 app_type 值获取类型配置
  const getAppTypeConfig = (appType: string) => {
    return appTypes.find(type => type.value === appType)
  }

  // 创建/更新代码库
  const repoMutation = useMutation({
    mutationFn: async (values: RepositoryFormValues) => {
      if (editingRepo) {
        return await repositoryService.update(editingRepo.id, values)
      }
      return await repositoryService.create(values as CreateRepositoryRequest)
    },
    onSuccess: () => {
      message.success(
        editingRepo ? t('repository.updateSuccess') : t('repository.createSuccess')
      )
      setRepoModalVisible(false)
      repoForm.resetFields()
      setEditingRepo(null)
      queryClient.invalidateQueries({queryKey: ['repositories']})
    },
  })

  // 删除代码库
  const deleteRepoMutation = useMutation({
    mutationFn: (id: number) => repositoryService.delete(id),
    onSuccess: () => {
      message.success(t('repository.deleteSuccess'))
      queryClient.invalidateQueries({queryKey: ['repositories']})
    },
  })

  // 创建/更新应用
  const appMutation = useMutation({
    mutationFn: async (values: ApplicationFormValues) => {
      if (editingApp) {
        return await applicationService.update(editingApp.id, values)
      }
      return await applicationService.create(values as CreateApplicationRequest)
    },
    onSuccess: () => {
      message.success(
        editingApp ? t('application.updateSuccess') : t('application.createSuccess')
      )
      setAppModalVisible(false)
      appForm.resetFields()
      setEditingApp(null)
      queryClient.invalidateQueries({queryKey: ['repositories']})
    },
  })

  // 删除应用
  const deleteAppMutation = useMutation({
    mutationFn: (id: number) => applicationService.delete(id),
    onSuccess: () => {
      message.success(t('application.deleteSuccess'))
      queryClient.invalidateQueries({queryKey: ['applications']})
    },
  })

  // 处理函数
  const handleCreateRepo = () => {
    setEditingRepo(null)
    repoForm.resetFields()
    setModalProjectId(undefined)  // 重置模态框项目选择
    setRepoModalVisible(true)
  }

  const handleEditRepo = (repo: Repository) => {
    setEditingRepo(repo)
    repoForm.setFieldsValue(repo)
    const projectId = repo.project_id || undefined
    setModalProjectId(projectId)  // 设置模态框项目选择

    // 如果项目下只有一个团队，自动选择它
    if (projectId) {
      const projectTeams = teams.filter(team => team.project_id === projectId)
      if (projectTeams.length === 1) {
        repoForm.setFieldValue('team_id', projectTeams[0].id)
      }
    }

    setRepoModalVisible(true)
  }

  const handleCreateApp = (repoId: number) => {
    setEditingApp(null)
    // 找到当前 repo
    const currentRepo = repoData.find(repo => repo.id === repoId)

    // 检查该 repo 是否已有应用
    const hasApps = (currentRepo?.applications?.length || 0) > 0

    // 设置应用模态框的项目ID（用于过滤团队列表）
    setAppModalProjectId(currentRepo?.project_id)

    appForm.resetFields()
    appForm.setFieldsValue({
      repo_id: repoId,
      name: hasApps ? '' : currentRepo?.name,  // 如果没有应用，默认使用 repo 名称
      project_id: currentRepo?.project_id,  // 继承 repo 的项目
      team_id: currentRepo?.team_id,  // 继承 repo 的团队
    })
    setAppModalVisible(true)
  }

  const handleEditApp = (app: Application) => {
    setEditingApp(app)
    // 设置应用模态框的项目ID（用于过滤团队列表）
    setAppModalProjectId(app.project_id)
    
    appForm.setFieldsValue({
      ...app,
      env_clusters: app.env_clusters || {},
    })
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

  // 处理筛选重置
  const handleResetFilters = () => {
    setKeyword('')
    setProjectId(undefined)
    setTeamId(undefined)
    setRepoPage(1)
  }

  // 筛选条件变化时重置到第一页
  const handleFilterChange = () => {
    setRepoPage(1)
  }

  // Repository 表格列定义
  const repoColumns: ColumnsType<Repository> = [
    {
      title: t('repository.name'),
      dataIndex: 'name',
      key: 'name',
      width: 450,
      render: (_, record) => {
        const appCount = record.applications?.length || 0
        const fullName = `${record.namespace}/${record.name}`
        return (
          <div style={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%'}}>
            <Space>
              <FolderOutlined style={{color: '#1890ff'}}/>
              <span style={{color: '#999', fontSize: 12, userSelect: 'none'}}>#{record.id} </span>
              <span className="repo-name" style={{userSelect: 'text'}}>{fullName}</span>
              {record.git_url && (
                <Tooltip title={record.git_url}>
                  <a
                    href={record.git_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <LinkOutlined style={{fontSize: 13, color: '#1890ff'}}/>
                  </a>
                </Tooltip>
              )}
            </Space>
            <span className="app-count" style={{marginLeft: 16, whiteSpace: 'nowrap'}}>
              <AppstoreOutlined style={{fontSize: 12, marginRight: 4}}/>
              {appCount} 个应用
            </span>
          </div>
        )
      },
    },
    // {
    //   title: t('repository.gitType'),
    //   dataIndex: 'git_type',
    //   key: 'git_type',
    //   width: 120,
    //   render: (text) => <Tag color="cyan">{text}</Tag>,
    // },
    {
      title: t('repository.projectAndTeam'),
      key: 'project_name-team_name',
      width: 100,
      render: (_, record) =>
        record.project_name || record.team_name ? (
          <Tag>
            <span>{record.project_name ? record.project_name : '-'}</span>
            <span> / </span>
            <span>{record.team_name ? record.team_name : '-'}</span>
          </Tag>
        ) : (
          <Tag style={{color: '#999'}}>-</Tag>
        )
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
              icon={<PlusOutlined/>}
              onClick={() => handleCreateApp(record.id)}
            />
          </Tooltip>
          <Tooltip title={t('common.edit')}>
            <Button
              type="text"
              size="small"
              icon={<EditOutlined/>}
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
              icon={<DeleteOutlined/>}
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
      render: (text, record) => (
        <Space style={{paddingLeft: 24}}>
          <AppstoreOutlined style={{color: '#52c41a'}}/>
          <span style={{color: '#999', fontSize: 12, userSelect: 'none'}}>#{record.id} </span>
          <span style={{userSelect: 'text'}}>{text}</span>
        </Space>
      ),
    },
    {
      title: t('application.project'),
      key: 'project_name-team_name',
      width: 120,
      ellipsis: true,
      render: (_, record) =>
        <Tag>
          <span>{record.project_name ? record.project_name : '-'}</span>
          <span> / </span>
          <span>{record.team_name ? record.team_name : '-'}</span>
        </Tag>
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
              icon={<HistoryOutlined/>}
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
              icon={<EditOutlined/>}
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
              icon={<DeleteOutlined/>}
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
            <FolderOutlined/>
            <span>{t('repository.title')}</span>
          </Space>
        }
        extra={
          <Space>
            <Button
              icon={<ReloadOutlined/>}
              onClick={() => {
                queryClient.invalidateQueries({queryKey: ['repositories']})
                queryClient.invalidateQueries({queryKey: ['applications']})  // 保留以刷新其他可能的应用查询
              }}
            >
              {t('common.refresh')}
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined/>}
              onClick={handleCreateRepo}
            >
              {t('repository.create')}
            </Button>
          </Space>
        }
      >
        {/* 筛选器 */}
        <div style={{marginBottom: 16}}>
          <Space size="middle" wrap>
            <Input.Search
              placeholder={t('repository.keywordPlaceholder')}
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                handleFilterChange()
              }}
              onSearch={handleFilterChange}
              style={{width: 280}}
              allowClear
            />
            <Select
              placeholder={t('repository.selectProject')}
              value={projectId}
              onChange={(value) => {
                setProjectId(value)
                // 当项目改变时，清空团队选择（因为团队列表会联动变化）
                // 如果选择了"无归属"，也清空团队
                if (value === NO_RELATION) {
                  setTeamId(undefined)
                }
                handleFilterChange()
              }}
              style={{width: 200}}
              allowClear
            >
              <Select.Option value={undefined}>{t('repository.allProjects')}</Select.Option>
              <Select.Option value={NO_RELATION}>{t('repository.noProject')}</Select.Option>
              {projects.map((project: ProjectSimple) => (
                <Select.Option key={project.id} value={project.id}>
                  {project.name}
                </Select.Option>
              ))}
            </Select>
            <Select
              placeholder={t('repository.selectTeam')}
              value={teamId}
              onChange={(value) => {
                setTeamId(value)
                handleFilterChange()
              }}
              style={{width: 200}}
              allowClear
              disabled={projectId === NO_RELATION || (!projectId && projectId !== 0)}
            >
              <Select.Option value={undefined}>{t('repository.allTeams')}</Select.Option>
              <Select.Option value={NO_RELATION}>{t('repository.noTeam')}</Select.Option>
              {filteredTeams.map((team: TeamSimple) => (
                <Select.Option key={team.id} value={team.id}>
                  {team.name}
                </Select.Option>
              ))}
            </Select>
            <Button onClick={handleResetFilters}>{t('common.reset')}</Button>
          </Space>
        </div>
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
            style: {cursor: 'pointer'},
          })}
        />

        {repoTotal > repoPageSize && (
          <div style={{marginTop: 16, textAlign: 'right'}}>
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
          setModalProjectId(undefined)  // 重置模态框项目选择
          repoForm.resetFields()
        }}
        confirmLoading={repoMutation.isPending}
        width={600}
      >
        <Form form={repoForm} layout="vertical">
          {/* 编辑模式下显示代码库标识 */}
          {editingRepo && (
            <div style={{
              marginBottom: 24,
              padding: '12px 16px',
              background: '#f5f5f5',
              borderRadius: 4,
              border: '1px solid #d9d9d9'
            }}>
              <Space direction="vertical" size={4} style={{width: '100%'}}>
                <div style={{fontSize: 12, color: '#999'}}>代码库</div>
                <div style={{fontSize: 14, fontWeight: 500}}>
                  <FolderOutlined style={{marginRight: 8, color: '#1890ff'}}/>
                  {editingRepo.namespace}/{editingRepo.name}
                </div>
                {editingRepo.git_url && (
                  <div style={{fontSize: 12, color: '#666'}}>
                    {editingRepo.git_url}
                  </div>
                )}
              </Space>
            </div>
          )}

          {/* 创建模式下显示所有字段 */}
          {!editingRepo && (
            <>
              <Form.Item
                name="name"
                label={t('repository.name')}
                rules={[{required: true}]}
              >
                <Input placeholder="my-repo"/>
              </Form.Item>

              <Form.Item name="description" label={t('common.description')}>
                <Input.TextArea rows={3}/>
              </Form.Item>

              <Form.Item
                name="git_url"
                label={t('repository.gitUrl')}
                rules={[{required: true}]}
              >
                <Input placeholder="https://gitea.company.com/namespace/repo.git"/>
              </Form.Item>

              <Form.Item
                name="git_type"
                label={t('repository.gitType')}
                rules={[{required: true}]}
                initialValue="gitea"
              >
                <Select>
                  <Select.Option value="gitea">Gitea</Select.Option>
                  <Select.Option value="gitlab">GitLab</Select.Option>
                  <Select.Option value="github">GitHub</Select.Option>
                </Select>
              </Form.Item>

              <Form.Item name="git_token" label={t('repository.gitToken')}>
                <Input.Password placeholder="Optional"/>
              </Form.Item>
            </>
          )}

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="project_id" label={t('repository.project')} rules={[{required: true}]}>
                <Select
                  placeholder={t('repository.selectProject')}
                  allowClear
                  onChange={(value) => {
                    setModalProjectId(value)

                    // 当项目改变时，检查该项目下的团队数量
                    if (value) {
                      const projectTeams = teams.filter(team => team.project_id === value)
                      if (projectTeams.length === 1) {
                        // 如果只有一个团队，自动选择它
                        repoForm.setFieldValue('team_id', projectTeams[0].id)
                      } else {
                        // 如果有多个团队或没有团队，清空选择
                        repoForm.setFieldValue('team_id', undefined)
                      }
                    } else {
                      // 如果清空项目选择，也清空团队选择
                      repoForm.setFieldValue('team_id', undefined)
                    }
                  }}
                >
                  {projects.map((project: ProjectSimple) => (
                    <Select.Option key={project.id} value={project.id}>
                      {project.name}
                    </Select.Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="team_id" label={t('repository.team')}>
                <Select
                  placeholder={t('repository.selectTeam')}
                  allowClear
                  disabled={!modalProjectId}
                >
                  {modalFilteredTeams.map((team: TeamSimple) => (
                    <Select.Option key={team.id} value={team.id}>
                      {team.name}
                    </Select.Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>
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
          setAppModalProjectId(undefined)
          appForm.resetFields()
        }}
        confirmLoading={appMutation.isPending}
        width={700}
      >
        <Tabs
          defaultActiveKey="basic"
          items={[
            {
              key: 'basic',
              label: '基本信息',
              children: (
                <Form form={appForm} layout="vertical">
                  <Row gutter={16}>
                    <Col span={8}>
                      <Form.Item
                        name="repo_id"
                        label={t('application.repository')}
                        rules={[{required: true}]}
                      >
                        <Select disabled>
                          {repoData?.map((repo) => (
                            <Select.Option key={repo.id} value={repo.id}>
                              {repo.name}
                            </Select.Option>
                          ))}
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={16}>
                      <Form.Item
                        name="name"
                        label={t('application.name')}
                        rules={[{required: true}]}
                      >
                        <Input placeholder="my-service"/>
                      </Form.Item>
                    </Col>
                  </Row>

                  <Form.Item name="description" label={t('common.description')}>
                    <Input.TextArea rows={3}/>
                  </Form.Item>

                  <Form.Item
                    name="app_type"
                    label={t('application.appType')}
                    rules={[{required: true}]}
                  >
                    <Select placeholder={t('application.appType')}>
                      {appTypes.map((type: AppTypeOption) => (
                        <Select.Option key={type.value} value={type.value}>
                          <Space>
                            <span style={{color: type.color}}>●</span>
                            <span>{type.label}</span>
                            {type.description && (
                              <span style={{color: '#999', fontSize: '12px'}}>
                                ({type.description})
                              </span>
                            )}
                          </Space>
                        </Select.Option>
                      ))}
                    </Select>
                  </Form.Item>

                  <Row gutter={16}>
                    <Col span={12}>
                      <Form.Item
                        name="project_id"
                        label={t('application.project')}
                        rules={[{required: true, message: t('repository.selectProject')}]}
                      >
                        <Select
                          placeholder={t('repository.selectProject')}
                          allowClear
                          disabled={editingApp !== null}
                          onChange={(value) => {
                            // 当项目改变时，更新应用模态框的项目ID并清空团队选择
                            setAppModalProjectId(value)
                            appForm.setFieldValue('team_id', undefined)
                          }}
                        >
                          {projects?.map((project: ProjectSimple) => (
                            <Select.Option key={project.id} value={project.id}>
                              {project.name}
                            </Select.Option>
                          ))}
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item
                        name="team_id"
                        label={t('application.team')}
                      >
                        <Select
                          placeholder={t('repository.selectTeam')}
                          allowClear
                        >
                          {appModalFilteredTeams?.map((team: TeamSimple) => (
                            <Select.Option key={team.id} value={team.id}>
                              {team.name}
                            </Select.Option>
                          ))}
                        </Select>
                      </Form.Item>
                    </Col>
                  </Row>
                </Form>
              ),
            },
            {
              key: 'env-cluster',
              label: '环境集群配置',
              children: (
                <Form form={appForm} layout="vertical">
                  <Form.Item
                    name="env_clusters"
                    label="应用的环境集群配置"
                    tooltip="只能选择项目允许的环境和集群。如果项目未配置，需要先在项目管理中配置。"
                    rules={[{required: true, message: '请配置至少一个环境集群'}]}
                  >
                    <EnvClusterConfig projectId={appModalProjectId}/>
                  </Form.Item>
                </Form>
              ),
            },
          ]}
        />
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

