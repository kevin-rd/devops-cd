import {useEffect, useMemo, useRef, useState} from 'react'
import {Alert, Button, Card, Checkbox, Empty, Form, Input, message, Select, Space, Spin, Switch} from 'antd'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import type {ProjectEnvConfig, UpdateProjectEnvConfigsRequest} from '@/services/projectEnvConfig'
import {projectEnvConfigService} from '@/services/projectEnvConfig'
import type {Cluster} from '@/services/cluster'
import {getClusters} from '@/services/cluster'
import {useDirtyFields} from '@/hooks/useDirtyFields'
import type {Credential} from '@/services/credential'
import {credentialService} from '@/services/credential'
import {DeleteOutlined} from "@ant-design/icons";

interface TabEnvConfigProps {
  projectId: number
  refreshTrigger: number
}

// 支持的环境列表（可扩展）
const ENVIRONMENTS = [
  {key: 'pre', label: 'Pre 环境'},
  {key: 'prod', label: 'Prod 环境'},
]

interface EnvFormValues {
  allow_clusters: string[]
  default_clusters: string[]
  deployment_name_template: string
  artifacts_json?: ArtifactsV1
}

type ChartSourceType = 'helm_repo' | 'webhook'
type ValuesLayerType = 'git' | 'http_file' | 'inline' | 'pipeline_artifact'

interface ArtifactsV1 {
  schema_version: 1
  namespace_template: string
  config_chart?: ChartSpecV1
  app_chart?: ChartSpecV1
}

interface ChartSpecV1 {
  enabled: boolean
  depends_on_config_chart?: boolean
  release_name_template?: string
  chart: ChartSourceSpecV1
  values?: ValuesLayerV1[]
}

interface ChartSourceSpecV1 {
  type: ChartSourceType
  repo_url?: string
  credential_ref?: string
  chart_name_template?: string
  chart_version_template?: string
  artifact_url_template?: string
}

interface ValuesLayerV1 {
  type: ValuesLayerType
  credential_ref?: string
  repo_url?: string
  ref_template?: string
  path_template?: string
  url_template?: string
  content?: string
}

const defaultArtifactsFromLegacy = (legacy: Partial<ProjectEnvConfig>): ArtifactsV1 => {
  return {
    schema_version: 1,
    namespace_template: legacy.namespace || '',
    config_chart: {
      enabled: false,
      chart: {type: 'helm_repo'},
      values: [],
    },
    app_chart: {
      enabled: true,
      depends_on_config_chart: true,
      chart: {
        type: 'helm_repo',
        repo_url: legacy.chart_repo_url || '',
        chart_name_template: '{{.app_type}}',
        chart_version_template: '{{.build.image_tag}}',
      },
      values:
        legacy.values_repo_url && legacy.values_path_template
          ? [
            {
              type: 'git',
              repo_url: legacy.values_repo_url,
              ref_template: 'main',
              path_template: legacy.values_path_template,
            },
          ]
          : [],
    },
  }
}

const TabEnvConfig = ({projectId, refreshTrigger}: TabEnvConfigProps) => {
  const queryClient = useQueryClient()
  const [preForm] = Form.useForm<EnvFormValues>()
  const [prodForm] = Form.useForm<EnvFormValues>()
  const [chartCollapsed, setChartCollapsed] = useState<Record<string, { config: boolean; app: boolean }>>({
    pre: {config: false, app: false},
    prod: {config: false, app: false},
  })

  // 存储后端返回的原始值，便于重置
  const originalValuesRef = useRef<Record<string, EnvFormValues>>({})

  // useDirtyFields for both forms
  const preDirty = useDirtyFields<EnvFormValues>(preForm, {deepCompare: true})
  const prodDirty = useDirtyFields<EnvFormValues>(prodForm, {deepCompare: true})

  // 获取所有集群
  const {data: clustersResponse, isLoading: clustersLoading} = useQuery({
    queryKey: ['clusters', refreshTrigger],
    queryFn: async () => {
      return await getClusters({status: 1, page: 1, page_size: 1000})
    },
    staleTime: 60_000,
  })
  const allClusters: Cluster[] = clustersResponse?.data?.items || []

  // 获取凭据（项目级 + 全局）
  const {data: credentialsResponse} = useQuery({
    queryKey: ['credentials', projectId, refreshTrigger],
    queryFn: async () => {
      const [projectRes, globalRes] = await Promise.all([
        credentialService.list({scope: 'project', project_id: projectId}),
        credentialService.list({scope: 'global'}),
      ])
      return [...(projectRes.data || []), ...(globalRes.data || [])]
    },
    staleTime: 30_000,
    enabled: !!projectId,
  })
  const credentials: Credential[] = credentialsResponse || []
  const credentialOptions = useMemo(
    () =>
      credentials.map((c) => ({
        label: `${c.name} (${c.type}, ${c.scope}, id:${c.id})`,
        value: `id:${c.id}`,
      })),
    [credentials]
  )

  // 获取项目环境配置
  const {data: configsResponse, isLoading: configsLoading} = useQuery({
    queryKey: ['project-env-configs', projectId, refreshTrigger],
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
      const legacyForDefault: Partial<ProjectEnvConfig> = {
        namespace: config.namespace || '',
        chart_repo_url: config.chart_repo_url || '',
        values_repo_url: config.values_repo_url || '',
        values_path_template: config.values_path_template || '',
      }
      const artifacts = (config.artifacts_json as ArtifactsV1 | undefined) || defaultArtifactsFromLegacy(legacyForDefault)
      envConfigMap.set(config.env, {
        allow_clusters: config.allow_clusters || [],
        default_clusters: config.default_clusters || [],
        deployment_name_template: config.deployment_name_template || '',
        artifacts_json: artifacts,
      })
    })

    ENVIRONMENTS.forEach(({key}) => {
      const values =
        envConfigMap.get(key) || {
          allow_clusters: [],
          default_clusters: [],
          deployment_name_template: '',
          artifacts_json: defaultArtifactsFromLegacy({
            namespace: '',
            chart_repo_url: '',
            values_repo_url: '',
            values_path_template: ''
          }),
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

      const configsPayload: UpdateProjectEnvConfigsRequest['configs'] = {}

      // 只有当 pre 环境有变更时才添加到 payload
      if (preDirty.hasDirtyFields()) {
        // 后端 allow_clusters 为必填：这里直接提交完整表单值，避免只提交 dirty fields 导致校验失败
        configsPayload.pre = preForm.getFieldsValue() as any
      }

      // 只有当 prod 环境有变更时才添加到 payload
      if (prodDirty.hasDirtyFields()) {
        configsPayload.prod = prodForm.getFieldsValue() as any
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
        const collapsed = chartCollapsed[env.key] || {config: false, app: false}

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
              <Form.Item label="Namespace (Go Template)" tooltip="">
                <Space.Compact style={{width: '100%'}}>
                  <Form.Item
                    name={['artifacts_json', 'namespace_template']}
                    noStyle
                    rules={[{required: true, message: '请填写 namespace_template'}]}
                  >
                    <Input placeholder="{{.project}}-{{.env}}"/>
                  </Form.Item>
                  <Form.Item name={['artifacts_json', 'schema_version']} noStyle initialValue={1}>
                    <Input style={{display: 'none'}}/>
                  </Form.Item>
                </Space.Compact>
              </Form.Item>

              {/* Deployment Name Template */}
              {/*<Form.Item*/}
              {/*  label="Deployment Name Template"*/}
              {/*  name="deployment_name_template"*/}
              {/*  tooltip="部署名称模板，支持变量: {{.app_name}}, {{.project}}, {{.env}}"*/}
              {/*  rules={[{max: 255, message: '模板最长 255 字符'}]}*/}
              {/*>*/}
              {/*  <Input.TextArea*/}
              {/*    rows={2}*/}
              {/*    placeholder="例如: {{.app_name}}-{{.env}}"*/}
              {/*  />*/}
              {/*</Form.Item>*/}

              {/* config_chart */}
              <Card size="small" title="Before Deploy: config" style={{marginBottom: 12}}
                    extra={
                      <Space>
                        <Form.Item name={['artifacts_json', 'config_chart', 'enabled']} valuePropName="checked" noStyle>
                          <Switch size="small"/>
                        </Form.Item>
                        <Button
                          type="link"
                          size="small"
                          onClick={() =>
                            setChartCollapsed((prev) => ({
                              ...prev,
                              [env.key]: {...(prev[env.key] || {config: false, app: false}), config: !collapsed.config},
                            }))
                          }
                        >
                          {collapsed.config ? '展开' : '收起'}
                        </Button>
                      </Space>
                    }
              >
                {!collapsed.config && (
                  <>
                    <Form.Item
                      label="Release Name Template"
                      name={['artifacts_json', 'config_chart', 'release_name_template']}
                    >
                      <Input placeholder="{{.app_name}}-config"/>
                    </Form.Item>

                    <Form.Item label="Chart Source Type" name={['artifacts_json', 'config_chart', 'chart', 'type']}>
                      <Select
                        options={[
                          {label: 'helm_repo', value: 'helm_repo'},
                          {label: 'webhook', value: 'webhook'},
                        ]}
                      />
                    </Form.Item>

                    <Form.Item noStyle shouldUpdate>
                      {({getFieldValue}) => {
                        const t = getFieldValue(['artifacts_json', 'config_chart', 'chart', 'type']) as ChartSourceType
                        return (
                          <>
                            {t === 'helm_repo' && (
                              <>
                                <Form.Item
                                  label="Repo URL"
                                  name={['artifacts_json', 'config_chart', 'chart', 'repo_url']}
                                >
                                  <Input placeholder="https://charts.example.com"/>
                                </Form.Item>
                                <Form.Item
                                  label="Credential"
                                  name={['artifacts_json', 'config_chart', 'chart', 'credential_ref']}
                                >
                                  <Select allowClear options={credentialOptions} placeholder="选择凭据（可选）"/>
                                </Form.Item>
                                <Form.Item
                                  label="Chart Name Template"
                                  name={['artifacts_json', 'config_chart', 'chart', 'chart_name_template']}
                                >
                                  <Input placeholder="{{.app_name}}-config"/>
                                </Form.Item>
                                <Form.Item
                                  label="Chart Version Template"
                                  name={['artifacts_json', 'config_chart', 'chart', 'chart_version_template']}
                                >
                                  <Input placeholder="{{.build.image_tag}}"/>
                                </Form.Item>

                                <Form.List name={['artifacts_json', 'config_chart', 'values']}>
                                  {(fields, {add, remove}) => (
                                    <>
                                      {fields.map((field) => (
                                        <Card
                                          key={field.key}
                                          size="small"
                                          title={`values[${field.name}]`}
                                          style={{marginBottom: 8}}
                                          extra={<Button size='small' icon=<DeleteOutlined/>
                                                         onClick={() => remove(field.name)}>删除</Button>}
                                        >
                                          <Form.Item label="Type" name={[field.name, 'type']}>
                                            <Select
                                              options={[
                                                {label: 'git', value: 'git'},
                                                {label: 'http_file', value: 'http_file'},
                                                {label: 'inline', value: 'inline'},
                                                {label: 'pipeline_artifact', value: 'pipeline_artifact'},
                                              ]}
                                            />
                                          </Form.Item>
                                          <Form.Item label="Credential" name={[field.name, 'credential_ref']}>
                                            <Select allowClear options={credentialOptions}
                                                    placeholder="选择凭据（可选）"/>
                                          </Form.Item>
                                          <Form.Item noStyle shouldUpdate>
                                            {({getFieldValue}) => {
                                              const tp = getFieldValue([
                                                'artifacts_json',
                                                'config_chart',
                                                'values',
                                                field.name,
                                                'type',
                                              ]) as ValuesLayerType
                                              if (tp === 'git') {
                                                return (
                                                  <>
                                                    <Form.Item label="Repo URL" name={[field.name, 'repo_url']}>
                                                      <Input placeholder="git@github.com:org/repo.git"/>
                                                    </Form.Item>
                                                    <Form.Item label="Ref Template" name={[field.name, 'ref_template']}>
                                                      <Input placeholder="main"/>
                                                    </Form.Item>
                                                    <Form.Item label="Path Template"
                                                               name={[field.name, 'path_template']}>
                                                      <Input placeholder="values/{{.env}}/{{.app_name}}.yaml"/>
                                                    </Form.Item>
                                                  </>
                                                )
                                              }
                                              if (tp === 'inline') {
                                                return (
                                                  <Form.Item label="Content" name={[field.name, 'content']}>
                                                    <Input.TextArea rows={6} placeholder="key: value"/>
                                                  </Form.Item>
                                                )
                                              }
                                              return (
                                                <Form.Item label="URL Template" name={[field.name, 'url_template']}>
                                                  <Input placeholder="https://.../values.yaml"/>
                                                </Form.Item>
                                              )
                                            }}
                                          </Form.Item>
                                        </Card>
                                      ))}
                                      <Button onClick={() => add({type: 'git'})} block>
                                        添加 values layer
                                      </Button>
                                    </>
                                  )}
                                </Form.List>
                              </>
                            )}
                              {t === 'webhook' && (
                                <>
                                  <Form.Item
                                    label="Webhook URL (Go Template)"
                                    name={['artifacts_json', 'config_chart', 'chart', 'artifact_url_template']}
                                    tooltip="从流水线产物拉取 chart.tgz 的 URL 模板"
                                  >
                                    <Input placeholder="https://artifact.example.com/{{.build.image_tag}}/config.tgz"/>
                                  </Form.Item>
                                  <Form.Item
                                    label="Credential"
                                    name={['artifacts_json', 'config_chart', 'chart', 'credential_ref']}
                                  >
                                    <Select allowClear options={credentialOptions} placeholder="选择凭据（可选）"/>
                                  </Form.Item>
                                </>
                              )}
                          </>
                        )
                      }}
                    </Form.Item>


                  </>
                )}
              </Card>

              {/* app_chart */}
              <Card
                size="small"
                title="app_chart"
                extra={
                  <Space>
                    <Form.Item name={['artifacts_json', 'app_chart', 'enabled']} valuePropName="checked" noStyle>
                      <Switch size="small"/>
                    </Form.Item>
                    <Button type="link" size="small"
                            onClick={() =>
                              setChartCollapsed((prev) => ({
                                ...prev,
                                [env.key]: {...(prev[env.key] || {config: false, app: false}), app: !collapsed.app},
                              }))
                            }>
                      {collapsed.app ? '展开' : '收起'}
                    </Button>
                  </Space>
                }
              >
                {!collapsed.app && (
                  <>
                    <Form.Item label="Release Name (Go Template)"
                               name={['artifacts_json', 'app_chart', 'release_name_template']}>
                      <Input placeholder="{{.app_name}}"/>
                    </Form.Item>

                    <Form.Item label="Chart Source Type" name={['artifacts_json', 'app_chart', 'chart', 'type']}>
                      <Select
                        options={[
                          {label: 'helm_repo', value: 'helm_repo'},
                          {label: 'webhook', value: 'webhook'}
                        ]}
                      />
                    </Form.Item>

                    <Form.Item noStyle shouldUpdate>
                      {({getFieldValue}) => {
                        const t = getFieldValue(['artifacts_json', 'app_chart', 'chart', 'type']) as ChartSourceType
                        return (
                          <>
                            {t === 'helm_repo' && (
                              <>
                                <Form.Item label="Repo URL" name={['artifacts_json', 'app_chart', 'chart', 'repo_url']}>
                                  <Input placeholder="https://charts.example.com"/>
                                </Form.Item>
                                <Form.Item
                                  label="Credential"
                                  name={['artifacts_json', 'app_chart', 'chart', 'credential_ref']}
                                >
                                  <Select allowClear options={credentialOptions} placeholder="选择凭据（可选）"/>
                                </Form.Item>

                                <Form.Item
                                  label="Chart Name Template"
                                  name={['artifacts_json', 'app_chart', 'chart', 'chart_name_template']}
                                >
                                  <Input placeholder="{{.app_type}}"/>
                                </Form.Item>
                                <Form.Item
                                  label="Chart Version Template"
                                  name={['artifacts_json', 'app_chart', 'chart', 'chart_version_template']}
                                >
                                  <Input placeholder="{{.build.image_tag}}"/>
                                </Form.Item>

                                <Form.List name={['artifacts_json', 'app_chart', 'values']}>
                                  {(fields, {add, remove}) => (
                                    <>
                                      {fields.map((field) => (
                                        <Card
                                          key={field.key}
                                          size="small"
                                          title={`values[${field.name}]`}
                                          style={{marginBottom: 8}}
                                          extra={<Button size='small' icon=<DeleteOutlined/>
                                                         onClick={() => remove(field.name)}>删除</Button>}
                                        >
                                          <Form.Item label="Type" name={[field.name, 'type']}>
                                            <Select
                                              options={[
                                                {label: 'git', value: 'git'},
                                                {label: 'http_file', value: 'http_file'},
                                                {label: 'inline', value: 'inline'},
                                                {label: 'pipeline_artifact', value: 'pipeline_artifact'},
                                              ]}
                                            />
                                          </Form.Item>
                                          <Form.Item label="Credential" name={[field.name, 'credential_ref']}>
                                            <Select allowClear options={credentialOptions}
                                                    placeholder="选择凭据（可选）"/>
                                          </Form.Item>
                                          <Form.Item noStyle shouldUpdate>
                                            {({getFieldValue}) => {
                                              const tp = getFieldValue([
                                                'artifacts_json',
                                                'app_chart',
                                                'values',
                                                field.name,
                                                'type',
                                              ]) as ValuesLayerType
                                              if (tp === 'git') {
                                                return (
                                                  <>
                                                    <Form.Item label="Repo URL" name={[field.name, 'repo_url']}>
                                                      <Input placeholder="git@github.com:org/repo.git"/>
                                                    </Form.Item>
                                                    <Form.Item label="Ref Template" name={[field.name, 'ref_template']}>
                                                      <Input placeholder="main"/>
                                                    </Form.Item>
                                                    <Form.Item label="Path Template"
                                                               name={[field.name, 'path_template']}>
                                                      <Input placeholder="values/{{.env}}/{{.app_name}}.yaml"/>
                                                    </Form.Item>
                                                  </>
                                                )
                                              }
                                              if (tp === 'inline') {
                                                return (
                                                  <Form.Item label="Content" name={[field.name, 'content']}>
                                                    <Input.TextArea rows={6} placeholder="key: value"/>
                                                  </Form.Item>
                                                )
                                              }
                                              return (
                                                <Form.Item label="URL Template" name={[field.name, 'url_template']}>
                                                  <Input placeholder="https://.../values.yaml"/>
                                                </Form.Item>
                                              )
                                            }}
                                          </Form.Item>
                                        </Card>
                                      ))}
                                      <Button onClick={() => add({type: 'git'})} block>
                                        添加 values layer
                                      </Button>
                                    </>
                                  )}
                                </Form.List>
                              </>
                            )}
                            {t === 'webhook' && (
                              <Empty description="暂无实现"/>
                            )}
                          </>
                        )
                      }}
                    </Form.Item>
                  </>
                )}
              </Card>
            </Form>
          </Card>
        )
      })}
    </Card>
  )
}

export default TabEnvConfig

