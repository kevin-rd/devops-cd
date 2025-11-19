import React, { useState, useEffect } from 'react'
import { Layout, Menu, Avatar, Dropdown, Space, Button } from 'antd'
import {
  FolderOutlined,
  LogoutOutlined,
  UserOutlined,
  GlobalOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  RocketOutlined,
  ProjectOutlined,
} from '@ant-design/icons'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/authStore'
import type { MenuProps } from 'antd'
import './index.css'

const { Header, Sider, Content } = Layout

const MainLayout: React.FC = () => {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout } = useAuthStore()
  const [collapsed, setCollapsed] = useState(() => {
    if (typeof window === 'undefined') {
      return false
    }
    const stored = localStorage.getItem('layout-sider-collapsed')
    return stored ? stored === 'true' : false
  })

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }
    localStorage.setItem('layout-sider-collapsed', collapsed ? 'true' : 'false')
  }, [collapsed])

  // 菜单项
  const menuItems: MenuProps['items'] = [
    {
      key: '/repository',
      icon: <FolderOutlined />,
      label: t('menu.repository'),
    },
    {
      key: '/project',
      icon: <ProjectOutlined />,
      label: t('menu.project'),
    },
    {
      key: '/batch',
      icon: <RocketOutlined />,
      label: t('menu.batch'),
    },
    {
      key: '/batch/insights',
      icon: <RocketOutlined />,
      label: t('menu.batchInsights'),
    },
  ]

  // 用户下拉菜单
  const userMenuItems: MenuProps['items'] = [
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: t('auth.logout'),
      danger: true,
    },
  ]

  // 语言下拉菜单
  const languageMenuItems: MenuProps['items'] = [
    {
      key: 'zh',
      label: '简体中文',
    },
    {
      key: 'en',
      label: 'English',
    },
  ]

  const handleMenuClick = ({ key }: { key: string }) => {
    navigate(key)
  }

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') {
      logout()
      navigate('/login')
    }
  }

  const handleLanguageChange = ({ key }: { key: string }) => {
    i18n.changeLanguage(key)
    localStorage.setItem('language', key)
  }

  return (
    <Layout className="main-layout">
      <Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        className="layout-sider"
        width={240}
      >
        <div className="logo">
          <FolderOutlined style={{ fontSize: 24 }} />
          {!collapsed && <span>DevOps CD</span>}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[
            location.pathname.includes('/insights') 
              ? '/batch/insights' 
              : location.pathname.startsWith('/batch')
              ? '/batch'
              : location.pathname
          ]}
          items={menuItems}
          onClick={handleMenuClick}
        />
      </Sider>

      <Layout className={collapsed ? 'sider-collapsed' : ''}>
        <Header className="layout-header">
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
            className="trigger"
          />

          <Space size="large">
            <Dropdown menu={{ items: languageMenuItems, onClick: handleLanguageChange }}>
              <Button type="text" icon={<GlobalOutlined />}>
                {i18n.language === 'zh' ? '中文' : 'EN'}
              </Button>
            </Dropdown>

            <Dropdown menu={{ items: userMenuItems, onClick: handleUserMenuClick }}>
              <Space className="user-info">
                <Avatar icon={<UserOutlined />} />
                <span>{user?.display_name || user?.username}</span>
              </Space>
            </Dropdown>
          </Space>
        </Header>

        <Content className="layout-content">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

export default MainLayout

