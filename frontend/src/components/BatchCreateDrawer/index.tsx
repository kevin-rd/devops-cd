import {useCallback, useEffect, useState} from 'react'
import {Alert, Button, Drawer, Form, Input, message, Modal, Select, Space, Steps, Table, Tag, Typography,} from 'antd'
import {AppstoreOutlined, CheckCircleOutlined, ExclamationCircleOutlined, FormOutlined} from '@ant-design/icons'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {useTranslation} from 'react-i18next'
import {batchService} from '@/services/batch'
import {projectService, ProjectSimple} from '@/services/project'
import {useAuthStore} from '@/stores/authStore'
import type {CreateBatchRequest} from '@/types'
import AppSelectionTable from '@/components/AppSelectionTable'
import './index.css'
import {ColumnsType} from "antd/es/table";

const {TextArea} = Input
const {Paragraph} = Typography

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

export default function BatchCreateDrawer({open, onClose, onSuccess}: BatchCreateDrawerProps) {
  const {t} = useTranslation()
  const [form] = Form.useForm()
  const {user} = useAuthStore()
  const queryClient = useQueryClient()

  const [currentStep, setCurrentStep] = useState(0)

  // 选中的项目ID
  const [selectedProjectId, setSelectedProjectId] = useState<number>()

  // 选中的应用ID和发布说明
  const [selectedAppIds, setSelectedAppIds] = useState<number[]>([])
  const [releaseNotesMap, setReleaseNotesMap] = useState<Record<number, string>>({})

  // 加载项目列表
  const {data: projectsData} = useQuery({
    queryKey: ['projects'],
    queryFn: async (): Promise<ProjectSimple[]> => {
      // 不传分页参数，后端会返回所有项目的简化列表
      const res = await projectService.getAll()
      return res.data
    },
    enabled: open,
  })

  const projectOptions = projectsData?.map(project => ({
    label: project.name,
    value: project.id,
  }))

  // 创建批次 Mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateBatchRequest) => batchService.create(data),
    onSuccess: () => {
      message.success(t('batch.createSuccess'))
      queryClient.invalidateQueries({queryKey: ['batchList']})
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
    setSelectedProjectId(undefined)
    setSelectedAppIds([])
    setReleaseNotesMap({})
    onClose()
  }

  // 步骤切换处理 - 验证必填字段
  const handleStepChange = (step: number) => {
    // 如果要切换到步骤2（应用管理），先验证基本信息
    if (step === 1) {
      form.validateFields(['batch_number', 'project_id'])
        .then(() => {
          setCurrentStep(step)
        })
        .catch((error) => {
          console.error('Validation failed:', error)
          message.warning('请先填写批次编号和选择项目')
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
    if (selectedAppIds.length === 0) {
      message.warning(t('batch.selectAtLeastOneApp'))
      return
    }

    const formValues = form.getFieldsValue()

    // 构建请求数据
    const requestData: any = {
      batch_number: formValues.batch_number,
      project_id: formValues.project_id,  // 新增：项目ID
      initiator: formValues.initiator || user?.username || 'unknown',
      apps: selectedAppIds.map((appId) => {
        const app: any = {app_id: appId}
        const releaseNotes = releaseNotesMap[appId]?.trim()
        if (releaseNotes) {
          app.release_notes = releaseNotes
        }
        return app
      }),
    }

    // 添加可选字段
    if (formValues.release_notes && formValues.release_notes.trim()) {
      requestData.release_notes = formValues.release_notes.trim()
    }

    console.log('Creating batch with data:', requestData)
    createMutation.mutate(requestData)
  }

  // 处理应用选择变化
  const handleSelectionChange = useCallback(
    (newSelectedIds: number[], newReleaseNotes?: Record<number, string>) => {
      setSelectedAppIds(newSelectedIds)
      if (newReleaseNotes) {
        setReleaseNotesMap(newReleaseNotes)
      }
    },
    []
  )

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
            <div style={{fontSize: 12, color: '#8c8c8c'}}>ID: {record.batch_id}</div>
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
          <ExclamationCircleOutlined style={{color: '#faad14'}}/>
          <span>应用冲突</span>
        </Space>
      ),
      width: 700,
      content: (
        <div style={{marginTop: 16}}>
          <Alert
            message={errorMessage}
            type="warning"
            showIcon
            style={{marginBottom: 16}}
          />

          <Paragraph style={{marginBottom: 12}}>
            以下应用已存在于其他批次中，请处理后再创建：
          </Paragraph>

          <Table
            columns={conflictColumns}
            dataSource={conflicts}
            rowKey="app_id"
            pagination={false}
            size="small"
            style={{marginBottom: 16}}
          />

          <Alert
            message="处理建议"
            description={
              <ul style={{marginBottom: 0, paddingLeft: 20}}>
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
        setSelectedAppIds(prev => prev.filter(id => !conflictAppIds.includes(id)))
        setReleaseNotesMap(prev => {
          const next = {...prev}
          conflictAppIds.forEach(id => {
            delete next[id]
          })
          return next
        })
        message.info(`已自动取消选择 ${conflicts.length} 个冲突应用`)
      },
    })
  }


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
          icon={<CheckCircleOutlined/>}
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
      footerStyle={{textAlign: 'right'}}
      className="batch-create-drawer"
    >
      {/* 步骤指示器 */}
      <div style={{
        display: 'flex',
        justifyContent: 'center',
        marginBottom: 0,
        paddingTop: 24,
        paddingLeft: 24,
        paddingRight: 24
      }}>
        <Steps
          current={currentStep}
          onChange={handleStepChange}
          items={[
            {title: t('batch.step1'), icon: <FormOutlined/>},
            {title: t('batch.step2'), icon: <AppstoreOutlined/>},
          ]}
          style={{maxWidth: 500}}
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
        <div style={{display: currentStep === 0 ? 'block' : 'none'}}>
          <Form.Item
            name="batch_number"
            label={t('batch.batchNumber')}
            rules={[{required: true, message: '请输入批次编号'}]}
          >
            <Input
              placeholder={t('batch.batchNumberPlaceholder')}
              size="large"
            />
          </Form.Item>

          <Form.Item
            name="project_id"
            label="所属项目"
            rules={[{required: true, message: '请选择项目'}]}
          >
            <Select
              placeholder="选择项目"
              size="large"
              options={projectOptions}
              onChange={(value) => {
                setSelectedProjectId(value)
                // 切换项目时，清空已选择的应用
                setSelectedAppIds([])
                setReleaseNotesMap({})
              }}
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>

          <Form.Item
            name="initiator"
            label={t('batch.initiator')}
          >
            <Input disabled size="large"/>
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
        <div style={{display: currentStep === 1 ? 'block' : 'none'}}>
          <AppSelectionTable
            selection={{selectedIds: selectedAppIds}}
            projectId={selectedProjectId}
            onSelectionChange={handleSelectionChange}
            showReleaseNotes={true}
          />
        </div>
      </Form>
    </Drawer>
  )
}

