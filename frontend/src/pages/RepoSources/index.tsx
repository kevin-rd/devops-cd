import React, {useEffect, useMemo, useState} from 'react'
import {Button, Card, Form, Input, message, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Tooltip,} from 'antd'
import type {ColumnsType} from 'antd/es/table'
import {ApiOutlined, DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, SyncOutlined,} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import type {ApiResponse, BackendPaginatedResponse} from '@/types'
import {
  type CreateRepoSourceRequest,
  type RepoPlatform,
  type RepoSource,
  repoSourceService,
} from '@/services/repoSource'
import { projectService, type ProjectSimple } from '@/services/project'
import { teamService, type TeamSimple } from '@/services/team'
import './index.css'
import dayjs from "dayjs";

const platformOptions: { label: string; value: RepoPlatform }[] = [
  {label: 'Gitea', value: 'gitea'},
  {label: 'GitLab', value: 'gitlab'},
  {label: 'GitHub', value: 'github'},
]

const RepoSourcesPage: React.FC = () => {
  const [form] = Form.useForm()
  const queryClient = useQueryClient()

  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keyword, setKeyword] = useState('')
  const [platformFilter, setPlatformFilter] = useState<RepoPlatform | undefined>()
  const [modalVisible, setModalVisible] = useState(false)
  const [editingSource, setEditingSource] = useState<RepoSource | null>(null)
  
  // 模态框中选择的项目ID（用于联动团队列表）
  const [modalProjectId, setModalProjectId] = useState<number | undefined>()

  const {data: response, isLoading} = useQuery<BackendPaginatedResponse<RepoSource>>({
    queryKey: ['repo-sources', page, pageSize, keyword, platformFilter],
    queryFn: async () => {
      const res = await repoSourceService.getList({
        page,
        page_size: pageSize,
        keyword: keyword || undefined,
        platform: platformFilter,
      })
      return res as unknown as BackendPaginatedResponse<RepoSource>
    },
  })

  const sources = response?.data || []
  const total = response?.total || 0

  // 查询所有项目（用于下拉选择）
  const { data: projectsResponse } = useQuery<ApiResponse<ProjectSimple[]>>({
    queryKey: ['projects_all'],
    queryFn: async () => {
      const res = await projectService.getAll()
      return res as unknown as ApiResponse<ProjectSimple[]>
    },
    staleTime: 60000,  // 1分钟缓存
  })

  const projects: ProjectSimple[] = projectsResponse?.data || []

  // 查询所有团队（用于下拉选择）
  const { data: teamsResponse } = useQuery<ApiResponse<TeamSimple[]>>({
    queryKey: ['teams_all'],
    queryFn: async () => {
      const res = await teamService.getList()
      return res as unknown as ApiResponse<TeamSimple[]>
    },
    staleTime: 60000,  // 1分钟缓存
  })

  const teams: TeamSimple[] = teamsResponse?.data || []

  // 根据模态框中选择的项目过滤团队列表
  const modalFilteredTeams = modalProjectId 
    ? teams.filter(team => team.project_id === modalProjectId)
    : teams

  const closeModal = () => {
    setModalVisible(false)
    setEditingSource(null)
    setModalProjectId(undefined)
    form.resetFields()
  }

  const openCreateModal = () => {
    setEditingSource(null)
    setModalProjectId(undefined)
    form.resetFields()
    form.setFieldsValue({
      enabled: true,
    })
    setModalVisible(true)
  }

  const openEditModal = (record: RepoSource) => {
    setEditingSource(record)
    setModalProjectId(record.default_project_id)
    setModalVisible(true)
  }

  useEffect(() => {
    if (modalVisible && editingSource) {
      setTimeout(() => {
        form.setFieldsValue({
          platform: editingSource.platform,
          base_url: editingSource.base_url,
          namespace: editingSource.namespace,
          enabled: editingSource.enabled,
          default_project_id: editingSource.default_project_id,
          default_team_id: editingSource.default_team_id,
          // token 不回显
        })
      }, 0)
    }

    // 可选：关闭时重置表单
    if (!modalVisible) {
      form.resetFields()
    }
  }, [modalVisible, editingSource, form])

  const mutation = useMutation({
    mutationFn: async (values: CreateRepoSourceRequest & { id?: number }) => {
      if (editingSource) {
        return repoSourceService.update({...values, id: editingSource.id})
      }
      return repoSourceService.create(values)
    },
    onSuccess: () => {
      message.success(editingSource ? '更新成功' : '创建成功')
      closeModal()
      queryClient.invalidateQueries({queryKey: ['repo-sources']})
    },
    onError: (err: any) => {
      message.error(err?.message || '操作失败')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => repoSourceService.delete(id),
    onSuccess: () => {
      message.success('删除成功')
      queryClient.invalidateQueries({queryKey: ['repo-sources']})
    },
    onError: () => {
      message.error('删除失败')
    },
  })

  const testMutation = useMutation({
    mutationFn: (id: number) => repoSourceService.testConnection(id),
    onSuccess: () => {
      message.success('连接测试成功')
    },
    onError: () => {
      message.error('连接测试失败')
    },
  })

  const syncMutation = useMutation({
    mutationFn: (id: number) => repoSourceService.syncNow(id),
    onSuccess: () => {
      message.success('已触发同步')
      queryClient.invalidateQueries({queryKey: ['repo-sources']})
    },
    onError: () => {
      message.error('同步失败')
    },
  })

  const columns: ColumnsType<RepoSource> = useMemo(
    () => [
      {
        title: '平台',
        dataIndex: 'platform',
        width: 80,
        render: (value: RepoPlatform) => {
          const colorMap: Record<RepoPlatform, string> = {
            gitea: 'green',
            gitlab: 'orange',
            github: 'blue',
          }
          return <Tag color={colorMap[value]}>{value}</Tag>
        },
      },
      {
        title: 'Base URL',
        dataIndex: 'base_url',
        ellipsis: true,
      },
      {
        title: 'Namespace',
        dataIndex: 'namespace',
        width: 140,
        render: (_: any, record) => (
          // <span>{`${record.base_url.replace(/^https?:\/\//, '')}/${record.namespace}`}</span>
          <span>{record.namespace}</span>
        ),
      },
      {
        title: '默认项目',
        dataIndex: 'default_project_name',
        width: 120,
        render: (text: string) => text ? <Tag color="purple">{text}</Tag> : <span style={{ color: '#999' }}>-</span>,
      },
      {
        title: '默认团队',
        dataIndex: 'default_team_name',
        width: 120,
        render: (text: string) => text ? <Tag color="green">{text}</Tag> : <span style={{ color: '#999' }}>-</span>,
      },
      {
        title: '启用',
        dataIndex: 'enabled',
        width: 100,
        render: (val: boolean) => (
          <Tag color={val ? 'success' : 'default'}>{val ? '启用' : '禁用'}</Tag>
        ),
      },
      {
        title: '最近同步',
        dataIndex: 'last_synced_at',
        width: 220,
        render: (value: string | undefined, record) =>
          value ? (
            <Space size={4}>
              <span>{dayjs(value).format('YYYY-MM-DD HH:mm:ss')}</span>
              {record.last_status && (
                <Tag color={record.last_status === 'success' ? 'green' : 'red'}>
                  {record.last_status === 'success' ? '成功' : '失败'}
                </Tag>
              )}
            </Space>
          ) : (
            <span>-</span>
          ),
      },
      {
        title: '备注',
        dataIndex: 'last_message',
        ellipsis: {showTitle: false},
        render: (text: string | undefined) =>
          text ? (
            <Tooltip title={text}>
              <span>{text}</span>
            </Tooltip>
          ) : (
            <span>-</span>
          ),
      },
      {
        title: '操作',
        key: 'actions',
        width: 200,
        render: (_, record) => (
          <Space size="small">
            <Button
              size="small"
              icon={<SyncOutlined/>}
              onClick={() => syncMutation.mutate(record.id)}
              loading={syncMutation.isPending && syncMutation.variables === record.id}
            >
              同步
            </Button>
            <Button
              size="small"
              icon={<EditOutlined/>}
              onClick={() => openEditModal(record)}
            >
              编辑
            </Button>
            <Popconfirm
              title="确认删除该仓库源？"
              onConfirm={() => deleteMutation.mutate(record.id)}
            >
              <Button size="small" danger icon={<DeleteOutlined/>}/>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [testMutation.isPending, testMutation.variables, syncMutation.isPending, syncMutation.variables]
  )

  const handleSearch = () => {
    setPage(1)
    queryClient.invalidateQueries({queryKey: ['repo-sources']})
  }

  const handleResetFilters = () => {
    setKeyword('')
    setPlatformFilter(undefined)
    setPage(1)
    queryClient.invalidateQueries({queryKey: ['repo-sources']})
  }

  const handleSubmit = () => {
    form.validateFields().then((values) => {
      mutation.mutate(values)
    })
  }

  const handleTestConnection = () => {
    if (!editingSource) {
      message.info('请保存后再测试连接')
      return
    }
    testMutation.mutate(editingSource.id)
  }

  return (
    <div className="repo-sources-page">
      <Card
        title="仓库源管理"
        extra={
          <Button type="primary" icon={<PlusOutlined/>} onClick={openCreateModal}>
            新增仓库源
          </Button>
        }
      >
        <Space className="repo-sources-filters" wrap>
          <Input
            placeholder="按 BaseURL 或 Namespace 搜索"
            value={keyword}
            style={{width: 280}}
            onChange={(e) => setKeyword(e.target.value)}
            allowClear
          />
          <Select
            placeholder="平台"
            style={{width: 160}}
            allowClear
            value={platformFilter}
            options={platformOptions}
            onChange={(val) => setPlatformFilter(val)}
          />
          <Space>
            <Button type="primary" icon={<ReloadOutlined/>} onClick={handleSearch}>
              查询
            </Button>
            <Button onClick={handleResetFilters}>重置</Button>
          </Space>
        </Space>

        <Table<RepoSource>
          rowKey="id"
          loading={isLoading}
          columns={columns}
          dataSource={sources}
          pagination={{
            current: page,
            pageSize,
            total,
            onChange: (p, size) => {
              setPage(p)
              setPageSize(size || 10)
            },
            showSizeChanger: true,
          }}
        />
      </Card>

      <Modal
        title={editingSource ? '编辑仓库源' : '新增仓库源'}
        open={modalVisible}
        onCancel={closeModal}
        onOk={handleSubmit}
        confirmLoading={mutation.isPending}
        destroyOnHidden
        okText="保存"
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item
            label="平台"
            name="platform"
            rules={[{required: true, message: '请选择平台'}]}
          >
            <Select options={platformOptions}/>
          </Form.Item>
          <Form.Item
            label="Base URL"
            name="base_url"
            rules={[{required: true, message: '请输入 Base URL'}]}
          >
            <Input placeholder="https://git.example.com"/>
          </Form.Item>
          <Form.Item
            label="Namespace"
            name="namespace"
            rules={[{required: true, message: '请输入 Namespace'}]}
          >
            <Input placeholder="team-name"/>
          </Form.Item>
          <Form.Item
            label="访问令牌"
            name="token"
            rules={
              editingSource
                ? [{required: false}]
                : [{required: true, message: '请输入访问令牌'}]
            }
            extra={editingSource ? '不填写则保持原有 Token' : undefined}
          >
            <Input.Password placeholder="请粘贴访问令牌" autoComplete="new-password"/>
          </Form.Item>
          <Form.Item
            label="默认项目"
            name="default_project_id"
            extra="扫描时自动为新代码库设置此项目"
          >
            <Select
              placeholder="选择默认项目（可选）"
              allowClear
              onChange={(value) => {
                setModalProjectId(value)
                // 当项目改变时，清空团队选择
                form.setFieldValue('default_team_id', undefined)
              }}
            >
              {projects.map((project: ProjectSimple) => (
                <Select.Option key={project.id} value={project.id}>
                  {project.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            label="默认团队"
            name="default_team_id"
            extra="扫描时自动为新代码库设置此团队（需先选择项目）"
          >
            <Select
              placeholder="选择默认团队（可选）"
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
          <Form.Item label="启用" name="enabled" valuePropName="checked">
            <Switch/>
          </Form.Item>
          <Form.Item label="连接测试">
            <Button
              icon={<ApiOutlined/>}
              onClick={handleTestConnection}
              disabled={!editingSource}
              loading={testMutation.isPending}
            >
              测试连接
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default RepoSourcesPage

