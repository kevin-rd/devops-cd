import React, {useState} from 'react'
import {
  Button,
  Card,
  Checkbox,
  Form,
  Input,
  message,
  Modal,
  Pagination,
  Popconfirm,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
} from 'antd'
import type {ColumnsType} from 'antd/es/table'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ProjectOutlined,
  ReloadOutlined,
  SearchOutlined,
  TeamOutlined,
} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {useTranslation} from 'react-i18next'
import type {CreateProjectRequest, Project} from '@/services/project'
import {projectService} from '@/services/project'
import type {CreateTeamRequest, Team} from '@/services/team'
import {teamService} from '@/services/team'
import type {BackendPaginatedResponse} from '@/types'
import EnvClusterConfig from '@/components/EnvClusterConfig'
import './index.css'

const ProjectPage: React.FC = () => {
  const {t} = useTranslation()
  const queryClient = useQueryClient()
  const [form] = Form.useForm()
  const [teamForm] = Form.useForm()

  const [modalVisible, setModalVisible] = useState(false)
  const [teamModalVisible, setTeamModalVisible] = useState(false)
  const [editingProject, setEditingProject] = useState<Project | null>(null)
  const [editingTeam, setEditingTeam] = useState<Team | null>(null)
  const [expandedRowKeys, setExpandedRowKeys] = useState<React.Key[]>([])

  // 分页和搜索状态
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keyword, setKeyword] = useState('')

  // 查询项目列表
  const {data: response, isLoading} = useQuery<BackendPaginatedResponse<Project>>({
    queryKey: ['projects', page, pageSize, keyword],
    queryFn: async () => {
      const res = await projectService.getList({
        page,
        page_size: pageSize,
        keyword,
        with_teams: true,
      })
      return res as unknown as BackendPaginatedResponse<Project>
    },
  })

  const projects = response?.data || []
  const total = response?.total || 0

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
    onSuccess: () => {
      message.success(t('project.deleteSuccess'))
      queryClient.invalidateQueries({queryKey: ['projects']})
    },
    onError: () => {
      message.error(t('project.deleteFailed'))
    },
  })

  const teamMutation = useMutation({
    mutationFn: async (values: CreateTeamRequest & { id?: number }) => {
      if (editingTeam) {
        return await teamService.update(editingTeam.id, values)
      }
      return await teamService.create(values)
    },
    onSuccess: () => {
      message.success(
        editingTeam ? t('team.updateSuccess') : t('team.createSuccess')
      )
      setTeamModalVisible(false)
      setEditingTeam(null)
      teamForm.resetFields()
      queryClient.invalidateQueries({queryKey: ['projects']})
      queryClient.invalidateQueries({queryKey: ['teams_all']})
    },
    onError: () => {
      message.error(
        editingTeam ? t('team.updateFailed') : t('team.createFailed')
      )
    },
  })

  const deleteTeamMutation = useMutation({
    mutationFn: (id: number) => teamService.delete(id),
    onSuccess: () => {
      message.success(t('team.deleteSuccess'))
      queryClient.invalidateQueries({queryKey: ['projects']})
      queryClient.invalidateQueries({queryKey: ['teams_all']})
    },
    onError: () => {
      message.error(t('team.deleteFailed'))
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

  const handleCreateTeam = (project: Project) => {
    setEditingTeam(null)
    teamForm.resetFields()
    teamForm.setFieldsValue({
      project_id: project.id,
    })
    setTeamModalVisible(true)
  }

  const handleEditTeam = (team: Team) => {
    setEditingTeam(team)
    teamForm.setFieldsValue({
      project_id: team.project_id,
      name: team.name,
      leader_name: team.leader_name,
      description: team.description,
    })
    setTeamModalVisible(true)
  }

  const handleTeamSubmit = () => {
    teamForm.validateFields().then((values) => {
      const payload = editingTeam
        ? {...values, id: editingTeam.id}
        : values
      teamMutation.mutate(payload as CreateTeamRequest & { id?: number })
    })
  }

  // 表格列定义
  const columns: ColumnsType<Project> = [
    {
      title: t('project.name'),
      dataIndex: 'name',
      key: 'name',
      width: 200,
      render: (text) => (
        <Space>
          <ProjectOutlined style={{color: '#1890ff'}}/>
          <span className="project-name">{text}</span>
        </Space>
      ),
    },
    {
      title: t('project.owner'),
      dataIndex: 'owner_name',
      key: 'owner_name',
      width: 150,
      render: (text) => text ? <Tag color="blue">{text}</Tag> : <span style={{color: '#999'}}>-</span>,
    },
    {
      title: t('common.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (text) => text || <span style={{color: '#999'}}>-</span>,
    },
    {
      title: t('project.teamCount'),
      key: 'team_count',
      width: 120,
      render: (_, project) => (
        <Tag color="geekblue">{project.teams?.length || 0}</Tag>
      ),
    },
    {
      title: t('common.action'),
      key: 'action',
      width: 240,
      fixed: 'right',
      render: (_, project) => (
        <Space size="small" wrap>
          <Button type="text" size="small" icon={<PlusOutlined/>}
                  onClick={(e) => {
                    e.stopPropagation()
                    handleCreateTeam(project)
                  }}
          >Team
          </Button>
          <Button type="text" size="small"
                  icon={<EditOutlined/>}
                  onClick={(e) => {
                    e.stopPropagation()
                    handleEdit(project)
                  }}
          >{t('common.edit')}
          </Button>
          <Popconfirm
            title={t('project.deleteConfirm')}
            onConfirm={() => deleteMutation.mutate(project.id)}
            onCancel={(e) => e?.stopPropagation()}
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined/>}
              onClick={(e) => e.stopPropagation()}
            >
              {/*{t('common.delete')}*/}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const teamColumns: ColumnsType<Team> = [
    {
      title: t('team.name'),
      dataIndex: 'name',
      key: 'name',
      width: 200,
      render: (text) => (
        <Space>
          <TeamOutlined style={{color: '#52c41a'}}/>
          <span>{text}</span>
        </Space>
      ),
    },
    {
      title: t('team.leader'),
      dataIndex: 'leader_name',
      key: 'leader_name',
      width: 160,
      render: (text) => (text ? <Tag color="purple">{text}</Tag> : <span style={{color: '#999'}}>-</span>),
    },
    {
      title: t('common.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (text) => text || <span style={{color: '#999'}}>-</span>,
    },
    {
      title: t('common.action'),
      key: 'action',
      width: 160,
      render: (_, team) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<EditOutlined/>}
            onClick={(e) => {
              e.stopPropagation()
              handleEditTeam(team)
            }}
          >
            {t('common.edit')}
          </Button>
          <Popconfirm
            title={t('team.deleteConfirm')}
            onConfirm={() => deleteTeamMutation.mutate(team.id)}
            onCancel={(e) => e?.stopPropagation()}
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined/>}
              onClick={(e) => e.stopPropagation()}
              loading={deleteTeamMutation.isPending}
            >
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const renderTeamTable = (project: Project) => {
    const teams = project.teams || []
    return (
      <Table className="project-team-table"
             columns={teamColumns}
             dataSource={teams}
             rowKey="id"
             pagination={false}
             size="small"
             locale={{
               emptyText: t('team.noData'),
             }}
      />
    )
  }

  return (
    <div className="project-page">
      <Card
        title={
          <Space>
            <ProjectOutlined/>
            <span>{t('project.title')}</span>
          </Space>
        }
        extra={
          <Space>
            <Input.Search
              placeholder={t('project.searchPlaceholder')}
              allowClear
              style={{width: 250}}
              onSearch={handleSearch}
              enterButton={<SearchOutlined/>}
            />
            <Button
              icon={<ReloadOutlined/>}
              onClick={() => queryClient.invalidateQueries({queryKey: ['projects']})}
            >
              {t('common.refresh')}
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined/>}
              onClick={handleCreate}
            >
              {t('project.create')}
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={projects}
          rowKey="id"
          loading={isLoading}
          pagination={false}
          expandable={{
            expandedRowKeys,
            onExpandedRowsChange: (keys) => setExpandedRowKeys(keys as React.Key[]),
            expandedRowRender: (record) => renderTeamTable(record),
            rowExpandable: () => true,
            expandRowByClick: true,
          }}
        />

        {total > pageSize && (
          <div style={{marginTop: 16, textAlign: 'right'}}>
            <Pagination
              current={page}
              pageSize={pageSize}
              total={total}
              onChange={(page, pageSize) => {
                setPage(page)
                setPageSize(pageSize)
              }}
              showSizeChanger
              showQuickJumper
              showTotal={(total) => `${t('common.total')} ${total} ${t('project.items')}`}
            />
          </div>
        )}
      </Card>

      {/* 创建/编辑 Modal */}
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

      {/* Team Modal */}
      <Modal
        title={editingTeam ? t('team.edit') : t('team.create')}
        open={teamModalVisible}
        onOk={handleTeamSubmit}
        onCancel={() => {
          setTeamModalVisible(false)
          setEditingTeam(null)
          teamForm.resetFields()
        }}
        confirmLoading={teamMutation.isPending}
        width={520}
      >
        <Form form={teamForm} layout="vertical">
          <Form.Item
            name="project_id"
            label={t('project.name')}
            rules={[{required: true, message: t('project.nameRequired')}]}
          >
            <Select placeholder={t('project.selectProject')} disabled={!!editingTeam}>
              {projects.map((project) => (
                <Select.Option key={project.id} value={project.id}>
                  {project.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="name"
            label={t('team.name')}
            rules={[
              {required: true, message: t('team.nameRequired')},
              {max: 100, message: t('team.nameTooLong')},
            ]}
          >
            <Input placeholder="backend-team"/>
          </Form.Item>

          <Form.Item
            name="leader_name"
            label={t('team.leader')}
            rules={[{max: 100, message: t('team.leaderTooLong')}]}
          >
            <Input placeholder="leader"/>
          </Form.Item>

          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea rows={3} placeholder={t('team.descriptionPlaceholder')}/>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default ProjectPage

