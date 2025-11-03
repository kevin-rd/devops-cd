import React, { useState } from 'react'
import { Form, Input, Button, Card, Radio, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { authService } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'
import type { LoginRequest } from '@/types'
import './Login.css'

const Login: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { login } = useAuthStore()
  const [loading, setLoading] = useState(false)
  const [form] = Form.useForm()

  const handleSubmit = async (values: LoginRequest) => {
    setLoading(true)
    try {
      const response = await authService.login(values)
      const { access_token, refresh_token, expires_in, user } = response.data

      login(user, access_token, refresh_token, expires_in)
      message.success(t('auth.loginSuccess'))
      
      // 获取保存的重定向路径
      const redirectPath = localStorage.getItem('redirect_path')
      if (redirectPath && redirectPath !== '/login') {
        localStorage.removeItem('redirect_path')
        navigate(redirectPath)
      } else {
        navigate('/')
      }
    } catch (error) {
      message.error(t('auth.loginFailed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-container">
      <Card className="login-card">
        <div className="login-header">
          <h1>DevOps CD Platform</h1>
          <p>{t('auth.pleaseLogin')}</p>
        </div>

        <Form
          form={form}
          name="login"
          initialValues={{ auth_type: 'local' }}
          onFinish={handleSubmit}
          size="large"
        >
          <Form.Item
            name="username"
            rules={[
              { required: true, message: t('validation.required', { field: t('auth.username') }) },
            ]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder={t('auth.username')}
            />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[
              { required: true, message: t('validation.required', { field: t('auth.password') }) },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder={t('auth.password')}
            />
          </Form.Item>

          <Form.Item name="auth_type">
            <Radio.Group>
              <Radio value="local">{t('auth.local')}</Radio>
              <Radio value="ldap">{t('auth.ldap')}</Radio>
            </Radio.Group>
          </Form.Item>

          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
            >
              {t('auth.login')}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}

export default Login

