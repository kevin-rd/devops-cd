import React, {useMemo, useState} from 'react'
import {Button, Card, Collapse, Form, Input, message, Modal, Popconfirm, Select, Space, Table, Tag} from 'antd'
import type {ColumnsType} from 'antd/es/table'
import {DeleteOutlined, EditOutlined, PlusOutlined, TeamOutlined, UserOutlined} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {useTranslation} from 'react-i18next'
import type {Project} from '@/services/project'
import type {CreateTeamRequest, Team} from '@/services/team'
import {teamService} from '@/services/team'
import type {CreateTeamMemberRequest, TeamMember, UpdateTeamMemberRoleRequest} from '@/services/teamMember'
import {teamMemberService} from '@/services/teamMember'

interface TabTeamProps {
  project: Project
}

const TabTeam: React.FC<TabTeamProps> = ({project}) => {
  const {t} = useTranslation()
  const queryClient = useQueryClient()
  const [teamForm] = Form.useForm()
  const [memberForm] = Form.useForm()

  // State
  const [expandedTeamKeys, setExpandedTeamKeys] = useState<string[]>([])
  const [teamModalVisible, setTeamModalVisible] = useState(false)
  const [memberModalVisible, setMemberModalVisible] = useState(false)
  const [roleModalVisible, setRoleModalVisible] = useState(false)

  const [editingTeam, setEditingTeam] = useState<Team | null>(null)
  const [editingMemberForRole, setEditingMemberForRole] = useState<TeamMember | null>(null)

  const teams = useMemo(() => project.teams || [], [project.teams])

  // --- Mutations ---

  // 1. Team CRUD
  const teamMutation = useMutation({
    mutationFn: async (values: CreateTeamRequest & { id?: number }) => {
      if (editingTeam) {
        return await teamService.update(editingTeam.id, values)
      }
      return await teamService.create(values)
    },
    onSuccess: () => {
      message.success(editingTeam ? t('team.updateSuccess') : t('team.createSuccess'))
      setTeamModalVisible(false)
      setEditingTeam(null)
      teamForm.resetFields()
      queryClient.invalidateQueries({queryKey: ['project-detail', project.id]})
    },
    onError: () => {
      message.error(editingTeam ? t('team.updateFailed') : t('team.createFailed'))
    },
  })

  const deleteTeamMutation = useMutation({
    mutationFn: (id: number) => teamService.delete(id),
    onSuccess: () => {
      message.success('删除团队成功')
      queryClient.invalidateQueries({queryKey: ['project-detail', project.id]})
    },
    onError: () => {
      message.error('删除团队失败')
    },
  })

  // 2. Member CRUD
  const addMemberMutation = useMutation({
    mutationFn: (data: CreateTeamMemberRequest) => teamMemberService.add(data),
    onSuccess: (_, variables) => {
      message.success('添加成员成功')
      setMemberModalVisible(false)
      memberForm.resetFields()
      if (variables?.team_id) {
        queryClient.invalidateQueries({queryKey: ['team-members', variables.team_id]})
      }
      queryClient.invalidateQueries({queryKey: ['project-detail', project.id]})
    },
    onError: () => {
      message.error('添加成员失败')
    },
  })

  const updateRoleMutation = useMutation({
    mutationFn: ({id, data}: { id: number; data: UpdateTeamMemberRoleRequest }) =>
      teamMemberService.updateRole(id, data),
    onSuccess: () => {
      message.success('更新角色成功')
      const teamId = editingMemberForRole?.team_id
      setRoleModalVisible(false)
      setEditingMemberForRole(null)
      if (teamId) {
        queryClient.invalidateQueries({queryKey: ['team-members', teamId]})
      }
      queryClient.invalidateQueries({queryKey: ['project-detail', project.id]})
    },
    onError: () => {
      message.error('更新角色失败')
    },
  })

  const deleteMemberMutation = useMutation({
    mutationFn: ({id}: { id: number; teamId: number }) => teamMemberService.delete(id),
    onSuccess: (_, variables) => {
      message.success('删除成员成功')
      if (variables?.teamId) {
        queryClient.invalidateQueries({queryKey: ['team-members', variables.teamId]})
      }
      queryClient.invalidateQueries({queryKey: ['project-detail', project.id]})
    },
    onError: () => {
      message.error('删除成员失败')
    },
  })

  // --- Handlers ---

  const handleCreateTeam = () => {
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

  const handleAddMember = (teamId: number) => {
    memberForm.resetFields()
    memberForm.setFieldsValue({
      team_id: teamId,
      roles: [],
    })
    setMemberModalVisible(true)
  }

  const handleMemberSubmit = () => {
    memberForm.validateFields().then((values) => {
      addMemberMutation.mutate(values as CreateTeamMemberRequest)
    })
  }

  const handleEditRole = (member: TeamMember) => {
    setEditingMemberForRole(member)
    memberForm.setFieldsValue({
      roles: member.roles,
    })
    setRoleModalVisible(true)
  }

  const handleRoleSubmit = () => {
    if (!editingMemberForRole) return
    memberForm.validateFields().then((values) => {
      updateRoleMutation.mutate({
        id: editingMemberForRole.id,
        data: {roles: values.roles},
      })
    })
  }

  // --- Render Helpers ---

  const fetchTeamMembers = async (teamId: number) => {
    try {
      const res = await teamMemberService.getList({
        team_id: teamId,
        page: 1,
        page_size: 100,
      })
      return res.data?.items || []
    } catch (error) {
      console.error(error)
      return []
    }
  }

  const memberColumns: ColumnsType<TeamMember> = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      width: 150,
      render: (text) => (
        <Space>
          <UserOutlined/>
          <span>{text}</span>
        </Space>
      ),
    },
    {
      title: '显示名称',
      dataIndex: 'display_name',
      key: 'display_name',
      width: 150,
      render: (text) => text || <span style={{color: '#999'}}>-</span>,
    },
    {
      title: 'Email',
      dataIndex: 'email',
      key: 'email',
      width: 200,
      render: (text) => text || <span style={{color: '#999'}}>-</span>,
    },
    {
      title: '角色',
      dataIndex: 'roles',
      key: 'roles',
      render: (roles: string[]) => (
        <Space wrap>
          {roles && roles.length > 0 ? (
            roles.map((role) => (
              <Tag key={role} color="blue">
                {role}
              </Tag>
            ))
          ) : (
            <span style={{color: '#999'}}>-</span>
          )}
        </Space>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_, member) => (
        <Space size="small">
          <Button type="text" size="small" icon={<EditOutlined/>} onClick={() => handleEditRole(member)}/>
          <Popconfirm title="确定要删除该成员吗？"
                      onConfirm={() => deleteMemberMutation.mutate({id: member.id, teamId: member.team_id})}>
            <Button type="text" size="small" danger icon={<DeleteOutlined/>}/>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const TeamMembersPanel: React.FC<{ teamId: number; active: boolean }> = ({teamId, active}) => {
    const {data = [], isLoading} = useQuery<TeamMember[]>({
      queryKey: ['team-members', teamId],
      queryFn: () => fetchTeamMembers(teamId),
      enabled: active,
      staleTime: 1000 * 60,
    })

    return (
      <Table
        columns={memberColumns}
        dataSource={data}
        rowKey="id"
        loading={active && isLoading && data.length === 0}
        pagination={false}
        size="small"
        locale={{
          emptyText: active ? '暂无成员' : '请展开查看',
        }}
      />
    )
  }

  return (
    <>
      <Card
        title="团队管理"
        variant="borderless"
        style={{border: 'none', boxShadow: 'none'}}
        extra={
          teams.length > 0 && (
            <Button type="primary" size="small" icon={<PlusOutlined/>} onClick={handleCreateTeam}>
              创建团队
            </Button>
          )
        }
        styles={{
          header: {margin: 0, padding: "0 12px", fontSize: 16, fontWeight: 600},
          body: {padding: "24px 12px"}
        }}
      >
        {teams.length === 0 ? (
          <div style={{
            textAlign: 'center',
            padding: '40px',
            color: '#999',
            border: '1px dashed #d9d9d9',
            borderRadius: 8
          }}>
            <TeamOutlined style={{fontSize: 48, marginBottom: 16}}/>
            <p>暂无团队</p>
            <Button type="primary" onClick={handleCreateTeam}>
              创建第一个团队
            </Button>
          </div>
        ) : (
          <Collapse
            accordion
            activeKey={expandedTeamKeys}
            onChange={(keys) => setExpandedTeamKeys(keys as string[])}
            items={teams.map((team) => {
              const key = team.id.toString()
              const isExpanded = expandedTeamKeys.includes(key)
              return {
                key,
                label: (
                  <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
                    <Space>
                      <TeamOutlined style={{color: '#1890ff'}}/>
                      <span style={{fontWeight: 500}}>{team.name}</span>
                      {team.leader_name && <Tag color="purple">{team.leader_name}</Tag>}
                    </Space>
                    <Space>
                      <Button
                        type="text"
                        size="small"
                        icon={<EditOutlined/>}
                        onClick={(e) => {
                          e.stopPropagation()
                          handleEditTeam(team)
                        }}
                      >
                        编辑
                      </Button>
                      <Button
                        type="text"
                        size="small"
                        icon={<PlusOutlined/>}
                        onClick={(e) => {
                          handleAddMember(team.id)
                          e.stopPropagation()
                        }}
                      >
                        成员
                      </Button>
                      <Popconfirm
                        title="确定要删除该团队吗？"
                        onConfirm={(e) => {
                          e?.stopPropagation()
                          deleteTeamMutation.mutate(team.id)
                        }}
                        onCancel={(e) => e?.stopPropagation()}
                      >
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined/>}
                          onClick={(e) => e.stopPropagation()}
                        >
                          删除
                        </Button>
                      </Popconfirm>
                    </Space>
                  </div>
                ),
                children: <TeamMembersPanel teamId={team.id} active={isExpanded}/>,
              }
            })}
          />
        )}
      </Card>

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
          <Form.Item name="project_id" hidden>
            <Input/>
          </Form.Item>

          <Form.Item
            name="name"
            label={t('team.name')}
            rules={[
              {required: true, message: t('team.nameRequired')},
              {max: 100, message: t('team.nameTooLong')},
            ]}
          >
            <Input placeholder="例如: backend-team"/>
          </Form.Item>

          <Form.Item
            name="leader_name"
            label={t('team.leader')}
            rules={[{max: 100, message: t('team.leaderTooLong')}]}
          >
            <Input placeholder="负责人用户名"/>
          </Form.Item>

          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea rows={3} placeholder={t('team.descriptionPlaceholder')}/>
          </Form.Item>
        </Form>
      </Modal>

      {/* 添加成员 Modal */}
      <Modal
        title="添加成员"
        open={memberModalVisible}
        onOk={handleMemberSubmit}
        onCancel={() => {
          setMemberModalVisible(false)
          memberForm.resetFields()
        }}
        confirmLoading={addMemberMutation.isPending}
        width={520}
      >
        <Form form={memberForm} layout="vertical">
          <Form.Item name="team_id" hidden>
            <Input/>
          </Form.Item>
          <Form.Item
            name="user_id"
            label="用户ID"
            rules={[{required: true, message: '请输入用户ID'}]}
          >
            <Input placeholder="请输入用户ID" type="number"/>
          </Form.Item>
          <Form.Item
            name="roles"
            label="角色"
            tooltip="可以输入多个角色，用逗号分隔"
          >
            <Select
              mode="tags"
              placeholder="输入角色后按回车"
              tokenSeparators={[',']}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑角色 Modal */}
      <Modal
        title="编辑角色"
        open={roleModalVisible}
        onOk={handleRoleSubmit}
        onCancel={() => {
          setRoleModalVisible(false)
          setEditingMemberForRole(null)
          memberForm.resetFields()
        }}
        confirmLoading={updateRoleMutation.isPending}
        width={520}
      >
        <Form form={memberForm} layout="vertical">
          <Form.Item
            name="roles"
            label="角色"
            rules={[{required: true, message: '请至少选择一个角色'}]}
            tooltip="可以输入多个角色，用逗号分隔"
          >
            <Select
              mode="tags"
              placeholder="输入角色后按回车"
              tokenSeparators={[',']}
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

export default TabTeam
