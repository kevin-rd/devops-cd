import React, {useEffect, useState} from 'react'
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
  Segmented,
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
import {useDirtyFields} from '@/hooks/useDirtyFields'
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

  // 视图模式切换
  const [viewMode, setViewMode] = useState<'repo' | 'app'>('app')

  // 模态框中选择的项目ID（用于联动团队列表）
  const [modalProjectId, setModalProjectId] = useState<number | undefined>()
  // 应用模态框中的项目ID（用于过滤团队列表）
  const [appModalProjectId, setAppModalProjectId] = useState<number | undefined>()

  // Repository 视图 - 分页状态
  const [repoPage, setRepoPage] = useState(1)
  const [repoPageSize, setRepoPageSize] = useState(20)

  // Repository 视图 - 筛选状态
  const [keyword, setKeyword] = useState('')
  const [projectId, setProjectId] = useState<number | undefined>()
  const [teamId, setTeamId] = useState<number | undefined>()

  // 🔥 Application 视图 - 分页状态
  const [appPage, setAppPage] = useState(1)
  const [appPageSize, setAppPageSize] = useState(20)

  // 🔥 Application 视图 - 筛选状态
  const [appKeyword, setAppKeyword] = useState('')
  const [appProjectId, setAppProjectId] = useState<number | undefined>()
  const [appTeamId, setAppTeamId] = useState<number | undefined>()
  const [appTypeFilter, setAppTypeFilter] = useState<string | undefined>()

  // 特殊值：-1 表示"无归属"
  const NO_RELATION = -1

  // 构建历史 Drawer 状态
  const [buildDrawerVisible, setBuildDrawerVisible] = useState(false)
  const [selectedAppId, setSelectedAppId] = useState<number | null>(null)
  const [selectedAppName, setSelectedAppName] = useState('')

  // 🔥 Dirty Fields 功能 - Application
  const {
    setInitialValues: setAppInitialValues,
    getDirtyValues: getAppDirtyValues,
    getDirtyFields: getAppDirtyFields,
    resetDirty: resetAppDirty,
  } = useDirtyFields<Application>(appForm, {
    excludeFields: ['id', 'created_at', 'updated_at', 'status', 'repo_name', 'namespace', 'project_name', 'team_name', 'last_tag'],
    deepCompare: true,
    treatEmptyAsSame: true,
  })

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
    enabled: viewMode === 'repo',  // 只在 repo 视图时查询
  })

  const repoData = repoResponse?.items || []
  const repoTotal = repoResponse?.total || 0

  // 查询应用列表（Application 视图）
  const {data: appListResponse, isLoading: appListLoading} = useQuery({
    queryKey: ['applications', appPage, appPageSize, appKeyword, appProjectId, appTeamId, appTypeFilter],
    queryFn: async () => {
      // 处理特殊值：-1 表示查询无归属的，转换为 0
      const actualProjectId = appProjectId === NO_RELATION ? 0 : appProjectId
      const actualTeamId = appTeamId === NO_RELATION ? 0 : appTeamId

      const res = await applicationService.getList({
        page: appPage,
        page_size: appPageSize,
        keyword: appKeyword || undefined,
        project_id: actualProjectId,
        team_id: actualTeamId,
        app_type: appTypeFilter || undefined,
      })
      return res.data
    },
    enabled: viewMode === 'app',  // 🔥 只在 app 视图时查询
  })

  const appListData = appListResponse?.items || []
  const appListTotal = appListResponse?.total || 0

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

  // 🔥 查询项目详情（包含 teams 和 default_env_clusters）
  // 用于创建/编辑应用时获取项目的完整信息
  const {data: projectDetailResponse, isLoading: projectDetailLoading} = useQuery({
    queryKey: ['project-detail', appModalProjectId],
    queryFn: async () => {
      if (!appModalProjectId) return null
      const res = await projectService.getById(appModalProjectId, true)  // with_teams=true
      return res.data
    },
    enabled: !!appModalProjectId && appModalVisible,  // 只在 modal 打开且有 projectId 时查询
  })

  const projectDetail = projectDetailResponse

  // 🔥 自动预填充 default_env_clusters（仅创建模式）
  useEffect(() => {
    if (!editingApp && projectDetail?.default_env_clusters && appModalVisible) {
      // 创建模式下，如果项目有 default_env_clusters，自动设置
      const currentEnvClusters = appForm.getFieldValue('env_clusters')
      // 只有在 env_clusters 为空时才自动设置
      if (!currentEnvClusters || Object.keys(currentEnvClusters).length === 0) {
        appForm.setFieldValue('env_clusters', projectDetail.default_env_clusters)
      }
    }
  }, [projectDetail, editingApp, appModalVisible, appForm])

  // 根据选择的项目过滤团队列表（用于页面筛选 - Repo 视图）
  const filteredTeams = projectId && projectId !== NO_RELATION
    ? teams.filter(team => team.project_id === projectId)
    : teams

  // 🔥 根据选择的项目过滤团队列表（用于页面筛选 - App 视图）
  const appFilteredTeams = appProjectId && appProjectId !== NO_RELATION
    ? teams.filter(team => team.project_id === appProjectId)
    : teams

  // 根据模态框中选择的项目过滤团队列表（用于 Repository 模态框）
  const modalFilteredTeams = modalProjectId
    ? teams.filter(team => team.project_id === modalProjectId)
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
    onSuccess: (response) => {
      message.success(
        editingApp ? t('application.updateSuccess') : t('application.createSuccess')
      )

      // 使用返回的数据直接更新缓存，避免重新请求
      if (response?.data) {
        // 更新 Repository 视图的缓存
        queryClient.setQueryData(
          ['repositories', repoPage, repoPageSize, keyword, projectId, teamId],
          (oldData: { items: Repository[]; total: number; page: number; page_size: number } | undefined) => {
            if (!oldData?.items) {
              return oldData
            }

            return {
              ...oldData,
              items: oldData.items.map((repo: Repository) => {
                // 如果是更新操作，更新对应的应用
                if (editingApp && repo.applications) {
                  return {
                    ...repo,
                    applications: repo.applications.map((app: Application) =>
                      app.id === response.data.id ? {...app, ...response.data} : app
                    ),
                  }
                }

                // 如果是创建操作，添加新应用到对应的 repo
                if (!editingApp && repo.id === response.data.repo_id) {
                  return {
                    ...repo,
                    // 确保 applications 数组存在，如果不存在则创建
                    applications: repo.applications
                      ? [...repo.applications, response.data]
                      : [response.data],
                  }
                }

                return repo
              }),
            }
          }
        )

        // 更新 Application 视图的缓存
        queryClient.setQueryData(
          ['applications', appPage, appPageSize, appKeyword, appProjectId, appTeamId, appTypeFilter],
          (oldData: { items: Application[]; total: number; page: number; page_size: number } | undefined) => {
            if (!oldData?.items) {
              return oldData
            }

            return {
              ...oldData,
              items: editingApp
                ? // 更新操作：替换对应的应用
                oldData.items.map((app: Application) =>
                  app.id === response.data.id ? {...app, ...response.data} : app
                )
                : // 创建操作：在列表开头添加新应用
                [response.data, ...oldData.items],
              total: editingApp ? oldData.total : oldData.total + 1,
            }
          }
        )
      }

      setAppModalVisible(false)
      appForm.resetFields()
      resetAppDirty()
      setEditingApp(null)
      setAppModalProjectId(undefined)
    },
  })

  // 删除应用
  const deleteAppMutation = useMutation({
    mutationFn: (id: number) => applicationService.delete(id),
    onSuccess: () => {
      message.success(t('application.deleteSuccess'))
      // 🔥 同时刷新两个视图的查询
      queryClient.invalidateQueries({queryKey: ['repositories']})
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

    // 🔥 如果 repo 没有归属 project，不允许创建 app
    if (!currentRepo?.project_id) {
      message.error('该代码库未归属任何项目，无法创建应用。请先为代码库分配项目。')
      return
    }

    // 检查该 repo 是否已有应用
    const hasApps = (currentRepo?.applications?.length || 0) > 0

    // 设置应用模态框的项目ID（用于查询项目详情）
    setAppModalProjectId(currentRepo.project_id)

    appForm.resetFields()
    appForm.setFieldsValue({
      repo_id: repoId,
      name: hasApps ? '' : currentRepo?.name,  // 如果没有应用，默认使用 repo 名称
      project_id: currentRepo.project_id,  // 🔥 固定为 repo 的项目（不允许修改）
      team_id: currentRepo?.team_id,  // 继承 repo 的团队
      // env_clusters 将在 project 详情加载后自动设置 default 值
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

    // 🔥 设置初始值，用于追踪字段变化
    setAppInitialValues(app)

    setAppModalVisible(true)
  }

  const handleRepoSubmit = () => {
    repoForm.validateFields().then((values) => {
      repoMutation.mutate(values)
    })
  }

  const handleAppSubmit = () => {
    appForm.validateFields().then((values) => {
      // 🔥 如果是编辑模式，只提交修改过的字段
      let submitValues = values

      if (editingApp) {
        const dirtyValues = getAppDirtyValues()

        // 如果没有任何修改，提示用户
        if (Object.keys(dirtyValues).length === 0) {
          message.info('没有任何修改')
          return
        }

        submitValues = dirtyValues

        // 打印调试信息（可选）
        console.log('📝 Dirty fields:', getAppDirtyFields())
        console.log('📦 Submitting values:', submitValues)
      }

      appMutation.mutate(submitValues)
    })
  }

  // 查看构建历史
  const handleViewBuilds = (app: Application) => {
    setSelectedAppId(app.id)
    setSelectedAppName(app.display_name || app.name)
    setBuildDrawerVisible(true)
  }

  // 处理筛选重置 - Repo 视图
  const handleResetFilters = () => {
    setKeyword('')
    setProjectId(undefined)
    setTeamId(undefined)
    setRepoPage(1)
  }

  // 🔥 处理筛选重置 - App 视图
  const handleResetAppFilters = () => {
    setAppKeyword('')
    setAppProjectId(undefined)
    setAppTeamId(undefined)
    setAppTypeFilter(undefined)
    setAppPage(1)
  }

  // 筛选条件变化时重置到第一页 - Repo 视图
  const handleFilterChange = () => {
    setRepoPage(1)
  }

  // 🔥 筛选条件变化时重置到第一页 - App 视图
  const handleAppFilterChange = () => {
    setAppPage(1)
  }

  // 🔥 Repository 表格列定义
  const repoColumns: ColumnsType<Repository> = [
    {
      title: t('repository.name'),
      dataIndex: 'name',
      key: 'name',
      width: 400,
      render: (_, record) => {
        const appCount = record.applications?.length || 0
        const fullName = `${record.namespace}/${record.name}`
        return (
          <div style={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%'}}>
            <Space>
              {/*<FolderOutlined style={{color: '#1890ff'}}/>*/}
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

  // Repository Application 子表格列定义
  const appColumns: ColumnsType<Application> = [
    {
      title: t('application.name'),
      dataIndex: 'name',
      key: 'name',
      width: 400,
      render: (text, record) => (
        <Space style={{paddingLeft: 12}}>
          <AppstoreOutlined style={{color: '#52c41a'}}/>
          <span style={{color: '#999', fontSize: 12, userSelect: 'none'}}>#{record.id} </span>
          <span style={{fontWeight: 500, userSelect: 'text'}}>{text}</span>
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
      width: 100,
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
      title: '环境集群',
      dataIndex: 'env_clusters',
      key: 'env_clusters',
      width: 200,
      hidden: true,
      render: (envClusters: Record<string, string[]>) => {
        if (!envClusters || Object.keys(envClusters).length === 0) {
          return <Tag style={{color: '#999'}}>-</Tag>
        }
        return (
          <Space size={[0, 4]} wrap>
            {Object.entries(envClusters).map(([env, clusters]) => (
              <Tag key={env} color="blue">
                {env}: {clusters.join(', ')}
              </Tag>
            ))}
          </Space>
        )
      },
    },
    {
      title: t('application.lastTag'),
      dataIndex: 'deployed_tag',
      key: 'deployed_tag',
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

  // 🔥 Application 视图的表格列定义
  const appListColumns: ColumnsType<Application> = [
    {
      title: t('application.name'),
      dataIndex: 'name',
      key: 'name',
      width: 300,
      render: (text, record) => (
        <Space>
          <AppstoreOutlined style={{color: '#52c41a'}}/>
          <span style={{color: '#999', fontSize: 12, userSelect: 'none'}}>#{record.id} </span>
          <span style={{fontWeight: 500, userSelect: 'text'}}>{text}</span>
        </Space>
      ),
    },
    {
      title: '所属代码库',
      dataIndex: 'repo_name',
      key: 'repo_name',
      width: 220,
      render: (repoName, record) => {
        if (!repoName) return <Tag style={{color: '#999'}}>-</Tag>
        const fullName = record.namespace ? `${record.namespace}/${repoName}` : repoName
        return (
          <Space>
            {/*<FolderOutlined style={{color: '#1890ff', fontSize: 12}}/>*/}
            <span style={{fontSize: 13, userSelect: 'text'}}>{fullName}</span>
          </Space>
        )
      },
    },
    {
      title: t('application.project'),
      key: 'project_name-team_name',
      width: 150,
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
      title: t('application.appType'),
      dataIndex: 'app_type',
      key: 'app_type',
      width: 80,
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
        return <Tag color="default">{appType}</Tag>
      },
    },
    {
      title: '环境集群',
      dataIndex: 'env_clusters',
      key: 'env_clusters',
      width: 200,
      render: (envClusters: Record<string, string[]>) => {
        if (!envClusters || Object.keys(envClusters).length === 0) {
          return <Tag style={{color: '#999'}}>-</Tag>
        }
        return (
          <Space size={[0, 4]} wrap>
            {Object.entries(envClusters).map(([env, clusters]) => (
              <Tag key={env} color="blue">
                {env}: {clusters.join(', ')}
              </Tag>
            ))}
          </Space>
        )
      },
    },
    {
      title: t('application.lastTag'),
      dataIndex: 'deployed_tag',
      key: 'deployed_tag',
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
              onClick={() => handleViewBuilds(record)}
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
            <Segmented
              value={viewMode}
              onChange={(value) => setViewMode(value as 'repo' | 'app')}
              options={[
                {label: '仓库视图', value: 'repo'},
                {label: '应用视图', value: 'app'},
              ]}
              style={{marginLeft: 16}}
            />
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
        {/* Repository 视图 - 筛选器和分页器 */}
        {viewMode === 'repo' && (
          <div style={{
            marginBottom: 16,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: '16px'
          }}>
            <Space size="middle" wrap>
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
                style={{width: 140}}
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
                style={{width: 140}}
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
              <Input.Search
                placeholder={t('repository.keywordPlaceholder')}
                value={keyword}
                onChange={(e) => {
                  setKeyword(e.target.value)
                  handleFilterChange()
                }}
                onSearch={handleFilterChange}
                style={{width: 240}}
                allowClear
              />
              <Button onClick={handleResetFilters}>{t('common.reset')}</Button>
            </Space>

            {/* 🔥 分页器移到右侧 */}
            {repoTotal > 0 && (
              <Pagination
                current={repoPage}
                pageSize={repoPageSize}
                total={repoTotal}
                onChange={(page, pageSize) => {
                  setRepoPage(page)
                  setRepoPageSize(pageSize)
                }}
                showSizeChanger
                // showQuickJumper
                showTotal={(total) => `${t('common.total')} ${total} ${t('common.unit')}`}
              />
            )}
          </div>
        )}

        {/* Application 视图 - 筛选器和分页器 */}
        {viewMode === 'app' && (
          <div style={{
            marginBottom: 16,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: '16px'
          }}>
            <Space size="middle" wrap>
              <Select
                placeholder={t('repository.selectProject')}
                value={appProjectId}
                onChange={(value) => {
                  setAppProjectId(value)
                  // 当项目改变时，清空团队选择
                  if (value === NO_RELATION) {
                    setAppTeamId(undefined)
                  }
                  handleAppFilterChange()
                }}
                style={{width: 140}}
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
                value={appTeamId}
                onChange={(value) => {
                  setAppTeamId(value)
                  handleAppFilterChange()
                }}
                style={{width: 140}}
                allowClear
                disabled={appProjectId === NO_RELATION || (!appProjectId && appProjectId !== 0)}
              >
                <Select.Option value={undefined}>{t('repository.allTeams')}</Select.Option>
                <Select.Option value={NO_RELATION}>{t('repository.noTeam')}</Select.Option>
                {appFilteredTeams.map((team: TeamSimple) => (
                  <Select.Option key={team.id} value={team.id}>
                    {team.name}
                  </Select.Option>
                ))}
              </Select>
              <Select
                placeholder="应用类型"
                value={appTypeFilter}
                onChange={(value) => {
                  setAppTypeFilter(value)
                  handleAppFilterChange()
                }}
                style={{width: 140}}
                allowClear
              >
                <Select.Option value={undefined}>全部类型</Select.Option>
                {appTypes.map((type: AppTypeOption) => (
                  <Select.Option key={type.value} value={type.value}>
                    <Space size={4}>
                      <span style={{color: type.color}}>●</span>
                      <span>{type.label}</span>
                    </Space>
                  </Select.Option>
                ))}
              </Select>
              <Input.Search
                placeholder="搜索应用名称"
                value={appKeyword}
                onChange={(e) => {
                  setAppKeyword(e.target.value)
                  handleAppFilterChange()
                }}
                onSearch={handleAppFilterChange}
                style={{width: 240}}
                allowClear
              />
              <Button onClick={handleResetAppFilters}>{t('common.reset')}</Button>
            </Space>

            {/* 分页器移到右侧 */}
            {appListTotal > 0 && (
              <Pagination
                current={appPage}
                pageSize={appPageSize}
                total={appListTotal}
                onChange={(page, pageSize) => {
                  setAppPage(page)
                  setAppPageSize(pageSize)
                }}
                showSizeChanger
                showTotal={(total) => `${t('common.total')} ${total} ${t('common.unit')}`}
              />
            )}
          </div>
        )}

        {/* Repository 视图 - 表格 */}
        {viewMode === 'repo' && (
          <Table
            columns={repoColumns}
            dataSource={repoData}
            rowKey="id"
            loading={repoLoading}
            pagination={false}
            sticky={true}
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
                    // showHeader={false}
                    sticky={{offsetHeader: 55}}
                    size="small"
                    className="app-table"
                    scroll={{x: 'max-content', scrollToFirstRowOnChange: true}}
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
        )}

        {/* 🔥 Application 视图 - 表格 */}
        {viewMode === 'app' && (
          <Table
            columns={appListColumns}
            dataSource={appListData}
            rowKey="id"
            loading={appListLoading}
            pagination={false}
          />
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
          resetAppDirty()
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
                          disabled={true}  // 🔥 始终禁用，创建时继承 repo 的 project，编辑时不允许修改
                          loading={projectDetailLoading}
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
                          loading={projectDetailLoading}
                        >
                          {/* 🔥 使用 projectDetail.teams 而不是全局 teams */}
                          {projectDetail?.teams?.map((team) => (
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
                    {/* 🔥 传入 projectDetail，避免重复查询 */}
                    <EnvClusterConfig
                      projectId={appModalProjectId}
                      project={projectDetail || undefined}
                    />
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

