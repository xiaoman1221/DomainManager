import { useMemo, useState } from 'react'
import { Layout, Menu, Dropdown } from 'antd'
import {
  DashboardOutlined,
  GlobalOutlined,
  DollarOutlined,
  BankOutlined,
  SafetyCertificateOutlined,
  BellOutlined,
  UserOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import AppAvatar from '../components/AppAvatar'
import { notify } from '../utils/toast'

const { Sider, Header, Content } = Layout

const NAV = [
  { key: '/', label: '仪表盘', icon: <DashboardOutlined /> },
  { key: '/domains', label: '域名管理', icon: <GlobalOutlined /> },
  { key: '/price', label: '域名比价', icon: <DollarOutlined /> },
  { key: '/registrars', label: '注册商管理', icon: <BankOutlined /> },
  { key: '/certificates', label: '证书管理', icon: <SafetyCertificateOutlined /> },
  { key: '/notifications', label: '通知管理', icon: <BellOutlined /> },
  { key: '/profile', label: '个人设置', icon: <UserOutlined /> },
]

export default function MainLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout } = useAuth()

  const selectedKey = useMemo(() => {
    const match = NAV.filter((n) => n.key !== '/' && location.pathname.startsWith(n.key)).sort(
      (a, b) => b.key.length - a.key.length
    )[0]
    return match ? match.key : '/'
  }, [location.pathname])

  const pageTitle = useMemo(() => {
    const found = NAV.find((n) => n.key === selectedKey)
    return found ? found.label : ''
  }, [selectedKey])

  const menuItems = useMemo(
    () =>
      NAV.map((n) => ({
        key: n.key,
        icon: n.icon,
        label: n.label,
      })),
    []
  )

  const userItems = [
    { key: 'profile', icon: <UserOutlined />, label: '个人设置' },
    { type: 'divider' },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', danger: true },
  ]

  const onUserCommand = ({ key }) => {
    if (key === 'logout') {
      logout()
      notify('success', '已退出登录')
      navigate('/login')
    } else if (key === 'profile') {
      navigate('/profile')
    }
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        width={224}
        collapsedWidth={64}
        collapsible
        collapsed={collapsed}
        trigger={null}
        className="app-sider"
        theme="light"
      >
        <div className="app-logo">
          <div className="app-logo-mark">DM</div>
          {!collapsed && <div className="app-logo-text">Domain Manager</div>}
        </div>
        <Menu
          className="app-menu"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ borderInlineEnd: 'none' }}
        />
      </Sider>
      <Layout>
        <Header className="app-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <span
              style={{ cursor: 'pointer', color: '#57534e' }}
              onClick={() => setCollapsed((c) => !c)}
            >
              {collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            </span>
            <span className="app-header-title">{pageTitle}</span>
          </div>
          <Dropdown menu={{ items: userItems, onClick: onUserCommand }} placement="bottomRight">
            <div className="app-user">
              <AppAvatar email={user?.email} name={user?.nickname || user?.username} size={30} />
              <span style={{ fontSize: 13, color: '#1c1917', fontWeight: 500 }}>
                {user?.nickname || user?.username || '用户'}
              </span>
              {user?.role === 'admin' && (
                <span
                  style={{
                    fontSize: 11,
                    border: '1px solid var(--border-strong)',
                    borderRadius: 4,
                    padding: '1px 6px',
                    color: '#57534e',
                  }}
                >
                  管理员
                </span>
              )}
            </div>
          </Dropdown>
        </Header>
        <Content>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
