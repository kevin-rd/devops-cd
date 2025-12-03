import React, {useMemo, useState} from 'react'
import type {MenuProps} from 'antd'
import {
  Button,
  Card,
  Checkbox,
  Form,
  Input,
  Layout,
  Menu,
  message,
  Modal,
  Popconfirm,
  Space,
  Tabs,
  Tooltip,
} from 'antd'
import {
  AppstoreOutlined,
  DeleteOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  PlusOutlined,
  ProjectOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {useTranslation} from 'react-i18next'
import type {CreateProjectRequest, Project} from '@/services/project'
import {projectService} from '@/services/project'
import type {BackendPaginatedResponse} from '@/types'
import EnvClusterConfig from '@/components/EnvClusterConfig'
import TabBasicInfo from './TabBasicInfo.tsx'
import TabTeam from './TabTeam.tsx'
import TabEnvConfig from './TabEnvConfig.tsx'
import './index.css'

const {Sider, Content} = Layout

const ProjectPage: React.FC = () => {
  const {t} = useTranslation()
  const queryClient = useQueryClient()
  const [form] = Form.useForm()

  // 布局状态
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null)

  // Modal 状态
  const [modalVisible, setModalVisible] = useState(false)

  // 编辑状态
  const [editingProject, setEditingProject] = useState<Project | null>(null)

  // 分页和搜索状态
  const [page, setPage] = useState(1)
  const pageSize = 10
  const [keyword, setKeyword] = useState('')

  // Tabs 状态
  const [activeTabKey, setActiveTabKey] = useState('basic')

  // 查询项目列表（简化版，用于左侧列表）
  const {data: response} = useQuery<BackendPaginatedResponse<Project>>({
    queryKey: ['projects', page, pageSize, keyword],
    queryFn: async () => {
      const res = await projectService.getList({
        page,
        page_size: pageSize,
        keyword,
        with_teams: false, // 左侧列表不需要团队信息
      })
      return res as unknown as BackendPaginatedResponse<Project>
    },
  })

  const projects = useMemo(() => response?.data?.items ?? [], [response])
  const total = response?.data?.total || 0

  // 默认选择第一个项目
  React.useEffect(() => {
    if (!selectedProjectId && projects.length > 0) {
      setSelectedProjectId(projects[0].id)
    }
  }, [projects, selectedProjectId])

  // 查询选中项目的详情（包含团队）
  const {data: projectDetailResponse, isLoading: isLoadingDetail} = useQuery({
    queryKey: ['project-detail', selectedProjectId],
    queryFn: async () => {
      if (!selectedProjectId) return null
      const res = await projectService.getById(selectedProjectId, true)
      return res.data
    },
    staleTime: 10_000,
    enabled: !!selectedProjectId,
  })

  const selectedProject = projectDetailResponse

  // 创建/更新项目
  const mutation = useMutation({
    mutationFn: async (values: CreateProjectRequest & { id?: number }) => {
      if (editingProject) {
        return await projectService.update(editingProject.id, values)
      } else {
        return await projectService.create(values)
      }
    },
    onSuccess: () => {
      message.success(
        editingProject ? t('project.updateSuccess') : t('project.createSuccess')
      )
      setModalVisible(false)
      form.resetFields()
      setEditingProject(null)
      queryClient.invalidateQueries({queryKey: ['projects']})
      if (selectedProjectId) {
        queryClient.invalidateQueries({queryKey: ['project-detail', selectedProjectId]})
      }
    },
    onError: () => {
      message.error(
        editingProject ? t('project.updateFailed') : t('project.createFailed')
      )
    },
  })

  // 删除项目
  const deleteMutation = useMutation({
    mutationFn: (id: number) => projectService.delete(id),
    onSuccess: (_, deletedId) => {
      message.success(t('project.deleteSuccess'))
      queryClient.invalidateQueries({queryKey: ['projects']})
      if (selectedProjectId === deletedId) {
        setSelectedProjectId(null)
      }
    },
    onError: () => {
      message.error(t('project.deleteFailed'))
    },
  })

  // 处理函数
  const handleCreate = () => {
    setEditingProject(null)
    form.resetFields()
    form.setFieldsValue({
      create_default_team: true,
    })
    setModalVisible(true)
  }

  const handleEdit = (project: Project) => {
    setEditingProject(project)
    form.setFieldsValue({
      name: project.name,
      description: project.description,
      owner_name: project.owner_name,
      allowed_env_clusters: project.allowed_env_clusters || {},
      default_env_clusters: project.default_env_clusters || {},
      create_default_team: undefined,
    })
    setModalVisible(true)
  }

  const handleSubmit = () => {
    form.validateFields().then((values) => {
      mutation.mutate(values)
    })
  }

  const handleSearch = (value: string) => {
    setKeyword(value)
    setPage(1)
  }


  const menuSelectedKeys: string[] = selectedProjectId ? [selectedProjectId.toString()] : []

  const projectMenuItems = useMemo<MenuProps['items']>(() => {
    if (!projects || projects.length === 0) {
      return []
    }
    return projects.map((project) => ({
      key: project.id.toString(),
      icon: <AppstoreOutlined/>,
      label: (
        <div className="project-menu-item">
          <div className="project-menu-item-title">{project.name}</div>
          {project.owner_name && (
            <div className="project-menu-item-subtitle">{project.owner_name}</div>
          )}
        </div>
      ),
      title: project.name,
    }))
  }, [projects])

  const handleMenuSelect: MenuProps['onSelect'] = ({key}) => {
    const projectId = Number(key)
    if (!Number.isNaN(projectId)) {
      setSelectedProjectId(projectId)
    }
  }

  // tabs
  const mainTab = [
    {
      key: 'basic',
      label: '基本信息',
      children: selectedProject ? (
        <TabBasicInfo project={selectedProject} onEdit={handleEdit}/>
      ) : null,
    }, {
      key: 'env',
      label: '环境配置',
      children: selectedProject ? (
        <TabEnvConfig projectId={selectedProject.id}/>
      ) : null,
    }, {
      key: 'teams',
      label: '团队管理',
      children: selectedProject ? (
        <TabTeam project={selectedProject}/>
      ) : null,
    }]
  const mainTabContent: Record<string, React.ReactNode> = Object.fromEntries(
    mainTab.map(({key, children}) => [key, children])
  )

  return (
    <div className="project-page">
      <Layout style={{minHeight: 'calc(100vh - 48px)'}}>
        {/* 左侧项目列表 */}
        <Sider
          width={sidebarCollapsed ? 72 : 280}
          theme="light"
          collapsible
          collapsed={sidebarCollapsed}
          trigger={null}
          className="project-sider"
        >
          <div className="project-sider-body">
            <div className="project-sider-header">
              {!sidebarCollapsed && (
                <Space size="small">
                  <ProjectOutlined/>
                  <span className="project-sider-title">项目列表</span>
                </Space>
              )}
              <Button
                type="text"
                icon={sidebarCollapsed ? <MenuUnfoldOutlined/> : <MenuFoldOutlined/>}
                onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
              />
            </div>

            {!sidebarCollapsed ? (
              <div className="project-sider-actions">
                <Input.Search
                  placeholder="搜索项目"
                  allowClear
                  onSearch={handleSearch}
                  enterButton={<SearchOutlined/>}
                />
                <Button
                  type="primary"
                  icon={<PlusOutlined/>}
                  block
                  onClick={handleCreate}
                >
                  新建项目
                </Button>
              </div>
            ) : (
              <div className="project-sider-actions-collapsed">
                <Tooltip title="新建项目" placement="right">
                  <Button
                    type="primary"
                    shape="circle"
                    icon={<PlusOutlined/>}
                    onClick={handleCreate}
                  />
                </Tooltip>
              </div>
            )}

            <Menu
              mode="inline"
              items={projectMenuItems}
              selectedKeys={menuSelectedKeys}
              onSelect={handleMenuSelect}
              inlineCollapsed={sidebarCollapsed}
              className="project-menu"
            />

            {!sidebarCollapsed && total > pageSize && (
              <div className="project-menu-footer">
                <Button
                  type="link"
                  block
                  onClick={() => setPage(page + 1)}
                  disabled={page * pageSize >= total}
                >
                  加载更多
                </Button>
              </div>
            )}
          </div>
        </Sider>

        {/* 右侧项目详情 */}
        <Content>
          {selectedProjectId ? (
            <Card loading={isLoadingDetail}
                  style={{minWidth: 768}}
                  title={
                    <Space>
                      <span style={{color: '#999', fontSize: 14, userSelect: 'none'}}>#{selectedProject?.id}</span>
                      <span>{selectedProject?.name}</span>
                    </Space>
                  }
                  extra={
                    <Popconfirm title="确定要删除该项目吗？" onConfirm={() => deleteMutation.mutate(selectedProjectId)}>
                      <Button danger icon={<DeleteOutlined/>}>删除项目</Button>
                    </Popconfirm>
                  }
                  tabList={mainTab.map(({key, label}) => ({key, label}))}
                  activeTabKey={activeTabKey}
                  tabBarExtraContent={
                    <Button key="refresh" icon={<ReloadOutlined/>} onClick={() => {
                      if (selectedProjectId) {
                        queryClient.invalidateQueries({queryKey: ['project-detail', selectedProjectId]})
                      }
                    }}>刷新</Button>}
                  onTabChange={key => setActiveTabKey(key)}
            >
              {mainTabContent[activeTabKey]}
            </Card>
          ) : (
            <Card>
              <div style={{textAlign: 'center', padding: '80px', color: '#999'}}>
                <ProjectOutlined style={{fontSize: 64, marginBottom: 16}}/>
                <p style={{fontSize: 16}}>请从左侧选择一个项目</p>
              </div>
            </Card>
          )}
        </Content>
      </Layout>

      {/* 创建/编辑项目 Modal */}
      <Modal
        title={editingProject ? t('project.edit') : t('project.create')}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => {
          setModalVisible(false)
          setEditingProject(null)
          form.resetFields()
        }}
        confirmLoading={mutation.isPending}
        width={800}
        style={{top: 20}}
      >
        <Form form={form} layout="vertical">
          <Tabs
            defaultActiveKey="basic"
            items={[
              {
                key: 'basic',
                label: '基本信息',
                children: (
                  <>
                    <Form.Item
                      name="name"
                      label={t('project.name')}
                      rules={[
                        {required: true, message: t('project.nameRequired')},
                        {max: 100, message: t('project.nameTooLong')},
                      ]}
                    >
                      <Input placeholder="my-project" disabled={!!editingProject}/>
                    </Form.Item>

                    <Form.Item
                      name="owner_name"
                      label={t('project.owner')}
                      rules={[{max: 100, message: t('project.ownerTooLong')}]}
                    >
                      <Input placeholder="owner"/>
                    </Form.Item>

                    <Form.Item name="description" label={t('common.description')}>
                      <Input.TextArea rows={4} placeholder={t('project.descriptionPlaceholder')}/>
                    </Form.Item>

                    {!editingProject && (
                      <Form.Item
                        name="create_default_team"
                        valuePropName="checked"
                        initialValue
                      >
                        <Checkbox>{t('project.createDefaultTeam')}</Checkbox>
                      </Form.Item>
                    )}
                  </>
                ),
              },
              {
                key: 'env_cluster',
                label: '环境集群配置',
                children: (
                  <>
                    <Form.Item
                      name="allowed_env_clusters"
                      label="允许的环境和集群"
                      tooltip="配置项目下应用可以部署的环境和集群。如果不配置,应用将无法创建环境配置。"
                    >
                      <EnvClusterConfig/>
                    </Form.Item>

                    <Form.Item
                      noStyle
                      shouldUpdate={(prevValues, currentValues) =>
                        prevValues.allowed_env_clusters !== currentValues.allowed_env_clusters
                      }
                    >
                      {({getFieldValue}) => {
                        const allowedEnvClusters = getFieldValue('allowed_env_clusters')
                        return (
                          <Form.Item
                            name="default_env_clusters"
                            label="默认环境集群配置"
                            tooltip="配置项目的默认环境集群。只能选择上方'允许的环境和集群'中已配置的选项。"
                            dependencies={['allowed_env_clusters']}
                            rules={[
                              ({getFieldValue}) => ({
                                validator(_, value) {
                                  if (!value || Object.keys(value).length === 0) {
                                    return Promise.resolve()
                                  }
                                  const allowedEnvClusters = getFieldValue('allowed_env_clusters')
                                  if (!allowedEnvClusters || Object.keys(allowedEnvClusters).length === 0) {
                                    return Promise.reject(new Error('请先配置允许的环境和集群'))
                                  }

                                  // 校验是否为子集
                                  for (const env in value) {
                                    if (!allowedEnvClusters[env]) {
                                      return Promise.reject(new Error(`环境 '${env}' 不在允许的环境列表中`))
                                    }
                                    const allowedClusters = allowedEnvClusters[env] || []
                                    const defaultClusters = value[env] || []
                                    for (const cluster of defaultClusters) {
                                      if (!allowedClusters.includes(cluster)) {
                                        return Promise.reject(
                                          new Error(`集群 '${cluster}' 不在环境 '${env}' 的允许集群列表中`)
                                        )
                                      }
                                    }
                                  }
                                  return Promise.resolve()
                                },
                              }),
                            ]}
                          >
                            <EnvClusterConfig allowedOptions={allowedEnvClusters}/>
                          </Form.Item>
                        )
                      }}
                    </Form.Item>
                  </>
                ),
              },
            ]}
          />
        </Form>
      </Modal>
    </div>
  )
}

export default ProjectPage
