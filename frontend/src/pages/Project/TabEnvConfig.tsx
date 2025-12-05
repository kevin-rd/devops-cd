import {useEffect, useMemo, useRef} from 'react'
import {Alert, Button, Card, Checkbox, Form, Input, message, Select, Space, Spin} from 'antd'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import type {ProjectEnvConfig, UpdateProjectEnvConfigsRequest} from '@/services/projectEnvConfig'
import {projectEnvConfigService} from '@/services/projectEnvConfig'
import type {Cluster} from '@/services/cluster'
import {getClusters} from '@/services/cluster'
import {useDirtyFields} from '@/hooks/useDirtyFields'

interface TabEnvConfigProps {
  projectId: number
}

// 支持的环境列表（可扩展）
const ENVIRONMENTS = [
  {key: 'pre', label: 'Pre 环境'},
  {key: 'prod', label: 'Prod 环境'},
]

interface EnvFormValues {
  allow_clusters: string[]
  default_clusters: string[]
  namespace: string
  deployment_name_template: string
  chart_repo_url: string
  values_repo_url: string
  values_path_template: string
}

const TabEnvConfig = ({projectId}: TabEnvConfigProps) => {
  const queryClient = useQueryClient()
  const [preForm] = Form.useForm<EnvFormValues>()
  const [prodForm] = Form.useForm<EnvFormValues>()

  // 存储后端返回的原始值，便于重置
  const originalValuesRef = useRef<Record<string, EnvFormValues>>({})

  // useDirtyFields for both forms
  const preDirty = useDirtyFields<EnvFormValues>(preForm, {deepCompare: true})
  const prodDirty = useDirtyFields<EnvFormValues>(prodForm, {deepCompare: true})

  // 获取所有集群
  const {data: clustersResponse, isLoading: clustersLoading} = useQuery({
    queryKey: ['clusters'],
    queryFn: async () => {
      return await getClusters({status: 1, page: 1, page_size: 1000})
    },
    staleTime: 60_000,
  })
  const allClusters: Cluster[] = clustersResponse?.data?.items || []

  // 获取项目环境配置
  const {data: configsResponse, isLoading: configsLoading} = useQuery({
    queryKey: ['project-env-configs', projectId],
    queryFn: async () => {
      return await projectEnvConfigService.getList(projectId)
    },
    staleTime: 10_000,
    enabled: !!projectId,
  })

  // 初始化或刷新表单数据
  useEffect(() => {
    if (!configsResponse) return

    const configs: ProjectEnvConfig[] = configsResponse.data || []
    const envConfigMap = new Map<string, EnvFormValues>()
    configs.forEach((config) => {
      envConfigMap.set(config.env, {
        allow_clusters: config.allow_clusters || [],
        default_clusters: config.default_clusters || [],
        namespace: config.namespace || '',
        deployment_name_template: config.deployment_name_template || '',
        chart_repo_url: config.chart_repo_url || '',
        values_repo_url: config.values_repo_url || '',
        values_path_template: config.values_path_template || '',
      })
    })

    ENVIRONMENTS.forEach(({key}) => {
      const values =
        envConfigMap.get(key) || {
          allow_clusters: [],
          default_clusters: [],
          namespace: '',
          deployment_name_template: '',
          chart_repo_url: '',
          values_repo_url: '',
          values_path_template: '',
        }

      if (key === 'pre') {
        preForm.setFieldsValue(values)
        preDirty.setInitialValues(values)
      } else {
        prodForm.setFieldsValue(values)
        prodDirty.setInitialValues(values)
      }

      originalValuesRef.current[key] = values
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [configsResponse, preForm, prodForm, preDirty.setInitialValues, prodDirty.setInitialValues])

  // 批量更新 mutation
  const updateMutation = useMutation({
    mutationFn: ({projectId, data}: { projectId: number; data: UpdateProjectEnvConfigsRequest }) =>
      projectEnvConfigService.updateConfigs(projectId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({queryKey: ['project-env-configs', projectId]})
      queryClient.invalidateQueries({queryKey: ['project-detail', projectId]})
    },
  })

  // 统一保存
  const handleSave = async () => {
    try {
      await Promise.all([preForm.validateFields(), prodForm.validateFields()])

      const configsPayload: Record<string, EnvFormValues> = {}

      // 只有当 pre 环境有变更时才添加到 payload
      if (preDirty.hasDirtyFields()) {
        configsPayload.pre = preDirty.getDirtyValues() as EnvFormValues
      }

      // 只有当 prod 环境有变更时才添加到 payload
      if (prodDirty.hasDirtyFields()) {
        configsPayload.prod = prodDirty.getDirtyValues() as EnvFormValues
      }

      // 如果没有任何变更（理论上按钮应该被禁用了，但为了安全起见），则直接返回
      if (Object.keys(configsPayload).length === 0) {
        message.info('没有需要保存的修改')
        return
      }

      await updateMutation.mutateAsync({
        projectId,
        data: {configs: configsPayload},
      })

      message.success('保存成功')

      // 更新 dirty 状态
      // 保存成功后，当前的表单值即为新的初始值
      if (configsPayload.pre) {
        const currentPreValues = preForm.getFieldsValue()
        preDirty.setInitialValues(currentPreValues)
        originalValuesRef.current.pre = currentPreValues
      }

      if (configsPayload.prod) {
        const currentProdValues = prodForm.getFieldsValue()
        prodDirty.setInitialValues(currentProdValues)
        originalValuesRef.current.prod = currentProdValues
      }
    } catch (error) {
      console.error('保存失败:', error)
      message.error('保存失败，请检查表单')
    }
  }

  // 重置修改
  const handleReset = () => {
    const preValues = originalValuesRef.current.pre
    const prodValues = originalValuesRef.current.prod

    if (preValues) {
      preForm.setFieldsValue(preValues)
      preDirty.setInitialValues(preValues)
    }
    if (prodValues) {
      prodForm.setFieldsValue(prodValues)
      prodDirty.setInitialValues(prodValues)
    }

    message.info('已重置为原始配置')
  }

  const preFormSnapshot = Form.useWatch([], preForm)
  const prodFormSnapshot = Form.useWatch([], prodForm)

  const isLoading = clustersLoading || configsLoading
  const hasChanges = useMemo(() => {
    void preFormSnapshot
    void prodFormSnapshot
    return preDirty.hasDirtyFields() || prodDirty.hasDirtyFields()
  }, [preDirty, prodDirty, preFormSnapshot, prodFormSnapshot])

  if (isLoading) {
    return (
      <div className="env-config-loading">
        <Spin tip="加载中...">
          <div style={{padding: '40px'}}/>
        </Spin>
      </div>
    )
  }

  if (allClusters.length === 0) {
    return (
      <Alert
        message="暂无可用集群"
        description="请先在集群管理中创建集群后再配置环境"
        type="warning"
        showIcon
      />
    )
  }

  const actionButtons = hasChanges ? (
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
    <Card title="环境配置" variant="borderless"
          extra={actionButtons}
          style={{border: 'none', boxShadow: 'none'}}
          styles={{
            header: {margin: 0, padding: "0 12px", fontSize: 16, fontWeight: 600},
            body: {display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 24, padding: 1}
          }}>
      {ENVIRONMENTS.map((env) => {
        const form = env.key === 'pre' ? preForm : prodForm

        return (
          <Card key={env.key} title={env.label}
                style={{border: "none", marginTop: 24}}
                styles={{
                  header: {padding: "0 12px", fontSize: 15, fontWeight: 500, color: "#1890ff", minHeight: 42},
                  body: {padding: "24px 12px"}
                }}>
            <Form form={form} layout="vertical">
              {/* 允许的集群 */}
              <Form.Item
                label="允许的集群"
                name="allow_clusters"
                tooltip="选择此环境允许部署的集群"
                rules={[{required: true, message: '请至少选择一个集群'}]}
              >
                <Select
                  mode="multiple"
                  placeholder="请选择允许的集群"
                  options={allClusters.map((cluster) => ({
                    label: cluster.name,
                    value: cluster.name,
                  }))}
                  showSearch
                  filterOption={(input, option) =>
                    (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                  }
                />
              </Form.Item>

              {/* 默认集群 */}
              <Form.Item noStyle shouldUpdate={(prev, curr) => prev.allow_clusters !== curr.allow_clusters}>
                {({getFieldValue}) => {
                  const allowedClusters = getFieldValue('allow_clusters') || []
                  return (
                    <Form.Item
                      label="默认集群"
                      name="default_clusters"
                      tooltip="从允许的集群中选择默认部署的集群"
                    >
                      <Checkbox.Group
                        options={allowedClusters.map((cluster: string) => ({
                          label: cluster,
                          value: cluster,
                        }))}
                        disabled={allowedClusters.length === 0}
                      />
                    </Form.Item>
                  )
                }}
              </Form.Item>

              {/* Namespace */}
              <Form.Item
                label="Namespace"
                name="namespace"
                tooltip="Kubernetes 命名空间"
                rules={[
                  {max: 63, message: 'Namespace 最长 63 字符'},
                  {
                    pattern: /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/,
                    message: '只能包含小写字母、数字和连字符，且必须以字母数字开头和结尾',
                  },
                ]}
              >
                <Input placeholder="例如: my-app-pre"/>
              </Form.Item>

              {/* Deployment Name Template */}
              <Form.Item
                label="Deployment Name Template"
                name="deployment_name_template"
                tooltip="部署名称模板，支持变量: {{.app_name}}, {{.project}}, {{.env}}"
                rules={[{max: 255, message: '模板最长 255 字符'}]}
              >
                <Input.TextArea
                  rows={2}
                  placeholder="例如: {{.app_name}}-{{.env}}"
                />
              </Form.Item>

              {/* Chart Repo URL */}
              <Form.Item
                label="Chart Repo URL"
                name="chart_repo_url"
                tooltip="Helm Chart 仓库地址"
                rules={[
                  {max: 255, message: 'URL 最长 255 字符'},
                  {type: 'url', message: '请输入有效的 URL'},
                ]}
              >
                <Input placeholder="https://charts.example.com"/>
              </Form.Item>

              {/* Values Repo URL */}
              <Form.Item
                label="Values Repo URL"
                name="values_repo_url"
                tooltip="Values 仓库地址 (可选)"
                rules={[
                  {max: 255, message: 'URL 最长 255 字符'},
                ]}
              >
                <Input placeholder="git@github.com:org/repo.git"/>
              </Form.Item>

              {/* Values Path Template */}
              <Form.Item
                label="Values Path Template"
                name="values_path_template"
                tooltip="Values 文件路径模板，支持变量: {{.app_name}}, {{.project}}, {{.env}}"
                rules={[{max: 255, message: '模板最长 255 字符'}]}
              >
                <Input placeholder="values/{{.env}}/{{.app_name}}.yaml"/>
              </Form.Item>
            </Form>
          </Card>
        )
      })}
    </Card>
  )
}

export default TabEnvConfig

