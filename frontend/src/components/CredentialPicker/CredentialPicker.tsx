import {Button, Col, Form, Input, Modal, Row, Select, Space, Switch} from 'antd'
import {useMemo, useState} from 'react'
import type {CreateCredentialRequest, Credential} from '@/services/credential.ts'
import {credentialService} from '@/services/credential.ts'

type CredentialType = Credential['type']
type CredentialScope = Credential['scope']

interface CredentialPickerProps {
  value?: string
  onChange?: (value?: string) => void

  projectId: number
  credentials: Credential[]
  allowedTypes: CredentialType[]
  allowedScopes?: CredentialScope[] // 默认: ['project','global']

  // 仅展示 name 以该前缀开头的凭据；创建时也会强制/自动补齐前缀
  namePrefix?: string

  placeholder?: string
  disabled?: boolean
  allowClear?: boolean

  // 创建成功后回调（用于外部触发 refetch）
  onCreated?: () => void
}

export const CredentialPicker = ({
                                   value,
                                   onChange,
                                   projectId,
                                   credentials,
                                   allowedTypes,
                                   allowedScopes = ['project', 'global'],
                                   namePrefix = '',
                                   placeholder,
                                   disabled,
                                   allowClear = true,
                                   onCreated,
                                 }: CredentialPickerProps) => {
  const [open, setOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()

  const filteredCredentials = useMemo(() => {
    return credentials.filter((c) => {
      if (!allowedTypes.includes(c.type)) return false
      if (allowedScopes.length > 0 && !allowedScopes.includes(c.scope)) return false
      if (namePrefix && !c.name.startsWith(namePrefix)) return false
      return true
    })
  }, [credentials, allowedTypes, allowedScopes, namePrefix])

  const options = useMemo(
    () =>
      filteredCredentials.map((c) => ({
        label: `${c.name} (${c.type}, ${c.scope}, id:${c.id})`,
        value: `id:${c.id}`,
      })),
    [filteredCredentials]
  )

  const openModal = () => {
    form.resetFields()

    const allowProject = allowedScopes.includes('project')
    const allowGlobal = allowedScopes.includes('global')
    const defaultIsGlobal = !allowProject && allowGlobal

    form.setFieldsValue({
      is_global: defaultIsGlobal,
      type: allowedTypes[0],
      name: namePrefix,
      description: '',
    })
    setOpen(true)
  }

  const handleOk = async () => {
    const v = await form.validateFields()
    const scope: CredentialScope = v.is_global ? 'global' : 'project'
    const type: CredentialType = v.type

    let name: string = (v.name || '').trim()
    if (namePrefix && !name.startsWith(namePrefix)) {
      name = `${namePrefix}${name}`
    }
    if (!name) {
      return
    }

    const data: Record<string, any> = {}
    if (type === 'basic_auth') {
      data.username = v.username
      data.password = v.password
    } else if (type === 'token') {
      data.token = v.token
    } else if (type === 'ssh_key') {
      data.private_key = v.private_key
      if (v.passphrase) data.passphrase = v.passphrase
    } else if (type === 'tls_client_cert') {
      data.cert = v.cert
      data.key = v.key
      if (v.ca) data.ca = v.ca
    }

    const payload: CreateCredentialRequest = {
      scope,
      name,
      type,
      data,
      meta: v.description ? {description: v.description} : undefined,
    }
    if (scope === 'project') {
      payload.project_id = projectId
    }

    setSubmitting(true)
    try {
      await credentialService.create(payload)
      setOpen(false)
      onCreated?.()
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <Space.Compact style={{width: '100%'}}>
        <Select
          value={value}
          onChange={onChange}
          allowClear={allowClear}
          disabled={disabled}
          options={options}
          placeholder={placeholder || '选择凭据（可选）'}
          showSearch
          filterOption={(input, option) =>
            (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
          }
        />
        <Button onClick={openModal} disabled={disabled}>
          新增
        </Button>
      </Space.Compact>

      <Modal title="新增凭据" open={open} onCancel={() => setOpen(false)} onOk={handleOk}
             okButtonProps={{loading: submitting}} destroyOnClose>
        <Form layout="vertical" form={form}>
          <Form.Item label="类型">
            <Row gutter={16} align='middle'>
              <Col span={12} style={{justifyContent: 'flex-end'}}>
                <Form.Item name="type" rules={[{required: true}]} noStyle>
                  <Select
                    style={{flex: 1}}
                    options={allowedTypes.map((t) => ({label: t, value: t}))}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <div style={{display: 'flex', justifyContent: 'flex-end'}}>
                  <Form.Item name="is_global" valuePropName="checked" noStyle>
                    <Switch disabled={allowedScopes.length === 1} checkedChildren="全局" unCheckedChildren="项目"/>
                  </Form.Item>
                </div>
              </Col>
            </Row>
          </Form.Item>

          <Form.Item
            label="名称"
            name="name"
            tooltip={namePrefix ? `建议使用前缀：${namePrefix}` : undefined}
            rules={[
              {required: true, message: '请输入名称'},
              {
                validator: async (_, val) => {
                  const s = String(val || '').trim()
                  if (namePrefix && s && !s.startsWith(namePrefix)) {
                    // 允许不带前缀，提交时会自动补齐；这里不报错
                    return Promise.resolve()
                  }
                  return Promise.resolve()
                },
              },
            ]}
          >
            <Input placeholder={namePrefix ? `${namePrefix}xxx` : '例如: my-cred'}/>
          </Form.Item>

          <Form.Item label="备注" name="description">
            <Input placeholder="可选，用于说明用途"/>
          </Form.Item>

          <Form.Item noStyle shouldUpdate>
            {({getFieldValue}) => {
              const t = getFieldValue('type') as CredentialType
              if (t === 'basic_auth') {
                return (
                  <>
                    <Form.Item label="Username" name="username" rules={[{required: true}]}>
                      <Input/>
                    </Form.Item>
                    <Form.Item label="Password" name="password" rules={[{required: true}]}>
                      <Input.Password/>
                    </Form.Item>
                  </>
                )
              }
              if (t === 'token') {
                return (
                  <Form.Item label="Token" name="token" rules={[{required: true}]}>
                    <Input.Password/>
                  </Form.Item>
                )
              }
              if (t === 'ssh_key') {
                return (
                  <>
                    <Form.Item label="Private Key" name="private_key" rules={[{required: true}]}>
                      <Input.TextArea rows={6} placeholder="-----BEGIN..."/>
                    </Form.Item>
                    <Form.Item label="Passphrase" name="passphrase">
                      <Input.Password/>
                    </Form.Item>
                  </>
                )
              }
              // tls_client_cert
              return (
                <>
                  <Form.Item label="Client Cert" name="cert" rules={[{required: true}]}>
                    <Input.TextArea rows={4}/>
                  </Form.Item>
                  <Form.Item label="Client Key" name="key" rules={[{required: true}]}>
                    <Input.TextArea rows={4}/>
                  </Form.Item>
                  <Form.Item label="CA (optional)" name="ca">
                    <Input.TextArea rows={4}/>
                  </Form.Item>
                </>
              )
            }}
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}


