import React, {useEffect, useMemo, useRef} from 'react'
import {Button, Card, Form, Input, message, Space} from 'antd'
import {useMutation, useQueryClient} from '@tanstack/react-query'
import type {Project} from '@/services/project'
import {projectService} from '@/services/project'
import {useDirtyFields} from '@/hooks/useDirtyFields'

interface BasicInfoTabProps {
  project: Project
  // onEdit 仍然保留以防万一，但本组件内将主要自行处理编辑
  onEdit?: (project: Project) => void
}

interface ProjectBasicFormValues {
  name: string
  owner_name: string
  description: string
}

const TabBasicInfo: React.FC<BasicInfoTabProps> = ({project}) => {
  const queryClient = useQueryClient()
  const [form] = Form.useForm<ProjectBasicFormValues>()
  const originalValuesRef = useRef<ProjectBasicFormValues>({
    name: '',
    owner_name: '',
    description: '',
  })

  // 追踪字段变更
  const {setInitialValues, hasDirtyFields,} = useDirtyFields<ProjectBasicFormValues>(form, {deepCompare: true})

  // 初始化表单数据
  useEffect(() => {
    if (project) {
      const initialValues = {
        name: project.name || '',
        owner_name: project.owner_name || '',
        description: project.description || '',
      }
      form.setFieldsValue(initialValues)
      setInitialValues(initialValues)
      originalValuesRef.current = initialValues
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [project, form, setInitialValues])

  // 更新项目 Mutation
  const updateMutation = useMutation({
    mutationFn: (data: Partial<Project>) =>
      projectService.update(project.id, data),
    onSuccess: () => {
      message.success('保存成功')
      // 刷新项目详情
      queryClient.invalidateQueries({queryKey: ['project-detail', project.id]})
      // 刷新项目列表（因为名字可能变了）
      queryClient.invalidateQueries({queryKey: ['projects']})

      // 更新初始值状态
      const currentValues = form.getFieldsValue()
      setInitialValues(currentValues)
      originalValuesRef.current = currentValues
    },
    onError: () => {
      message.error('保存失败')
    },
  })

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      await updateMutation.mutateAsync(values)
    } catch (error) {
      console.error('Validation failed:', error)
    }
  }

  const handleReset = () => {
    form.setFieldsValue(originalValuesRef.current)
    setInitialValues(originalValuesRef.current) // 重置 dirty 状态
    message.info('已重置')
  }

  // 监听表单变化用于渲染按钮
  const formValues = Form.useWatch([], form)
  const isDirty = useMemo(() => {
    void formValues // 订阅变化
    return hasDirtyFields()
  }, [formValues, hasDirtyFields])

  const actionButtons = isDirty ? (
    <Space>
      <Button onClick={handleReset} disabled={updateMutation.isPending}>
        重置
      </Button>
      <Button type="primary" onClick={handleSave} loading={updateMutation.isPending}>
        保存
      </Button>
    </Space>
  ) : null

  return (
    <Card
      title="基本信息"
      variant="borderless"
      style={{border: 'none', boxShadow: 'none'}}
      extra={actionButtons}
      styles={{
        header: {margin: 0, padding: "0 12px", fontSize: 16, fontWeight: 600},
        body: {padding: "24px 12px"}
      }}
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          name: project.name,
          owner_name: project.owner_name,
          description: project.description,
        }}
      >
        <Space direction="vertical" size="large" style={{width: '100%'}}>
          <Form.Item
            label="项目名称"
            name="name"
            rules={[
              {required: true, message: '请输入项目名称'},
              {max: 100, message: '项目名称最长 100 字符'},
            ]}
          >
            <Input
              variant="underlined"
              placeholder="请输入项目名称"
              maxLength={100}
            />
          </Form.Item>

          <Form.Item
            label="负责人"
            name="owner_name"
            rules={[{max: 100, message: '负责人名称最长 100 字符'}]}
          >
            <Input
              variant="filled"
              placeholder="请输入负责人"
              maxLength={100}
            />
          </Form.Item>

          <Form.Item
            label="描述"
            name="description"
          >
            <Input.TextArea
              variant="filled"
              placeholder="请输入项目描述"
              autoSize={{minRows: 3, maxRows: 10}}
            />
          </Form.Item>
        </Space>
      </Form>
    </Card>
  )
}

export default TabBasicInfo
