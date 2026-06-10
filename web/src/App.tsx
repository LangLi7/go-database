import { useState, useEffect } from 'react'
import { Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, Button, theme, Dropdown, Avatar, Space, Typography, Tooltip, Spin } from 'antd'
import {
  DashboardOutlined, DatabaseOutlined, ApiOutlined, UserOutlined, TeamOutlined,
  KeyOutlined, SettingOutlined, CodeOutlined, LogoutOutlined, MenuFoldOutlined,
  MenuUnfoldOutlined, ThunderboltOutlined, SunOutlined, MoonOutlined, GlobalOutlined,
} from '@ant-design/icons'
import { api } from './api/client'
import { useAppContext } from './context/AppContext'
import { t, Lang } from './i18n/translations'
import Dashboard from './pages/Dashboard'
import Connections from './pages/Connections'
import Explorer from './pages/Explorer'
import QueryEditor from './pages/QueryEditor'
import AdminUsers from './pages/AdminUsers'
import AdminRoles from './pages/AdminRoles'
import APIKeys from './pages/APIKeys'
import Settings from './pages/Settings'

const { Header, Sider, Content } = Layout
const { Text } = Typography

export default function App() {
  const navigate = useNavigate()
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(false)
  const [checking, setChecking] = useState(true)
  const [authed, setAuthed] = useState(false)
  const { token: themeToken } = theme.useToken()
  const { darkMode, toggleDarkMode, lang, setLanguage } = useAppContext()

  useEffect(() => {
    const onUnauth = () => {
      setAuthed(false)
      navigate('/login')
    }
    api.onUnauthorized(onUnauth)

    if (api.isAuthenticated()) {
      api.verifySession().then(valid => {
        setAuthed(valid)
        setChecking(false)
      }).catch(() => {
        setAuthed(false)
        setChecking(false)
      })
    } else {
      setAuthed(false)
      setChecking(false)
    }

    return () => { api.onUnauthorized(() => {}) }
  }, [navigate])

  const handleLogout = () => {
    api.setToken(null)
    setAuthed(false)
    navigate('/login')
  }

  const selectedKey = '/' + location.pathname.split('/').slice(1, 3).join('/')

  const menuItems = [
    { key: '/', icon: <DashboardOutlined />, label: t('nav.dashboard') },
    { key: '/connections', icon: <ApiOutlined />, label: t('nav.connections') },
    { key: '/explorer', icon: <DatabaseOutlined />, label: t('nav.explorer') },
    { key: '/query', icon: <CodeOutlined />, label: t('nav.query') },
    { type: 'divider' as const },
    { key: '/admin/users', icon: <TeamOutlined />, label: t('nav.users') },
    { key: '/admin/roles', icon: <KeyOutlined />, label: t('nav.roles') },
    { key: '/admin/apikeys', icon: <ThunderboltOutlined />, label: t('nav.apikeys') },
    { type: 'divider' as const },
    { key: '/admin/settings', icon: <SettingOutlined />, label: t('nav.settings') },
  ]

  if (checking) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: darkMode ? '#141414' : '#f0f2f5' }}>
        <Spin size="large" />
      </div>
    )
  }

  if (!authed) {
    return (
      <Routes>
        <Route path="/login" element={
          <div style={{
            minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: darkMode ? '#141414' : '#f0f2f5'
          }}>
            <div style={{
              width: 400, padding: 40,
              background: darkMode ? '#1f1f1f' : '#fff',
              borderRadius: 12, boxShadow: '0 4px 24px rgba(0,0,0,0.08)'
            }}>
              <LoginPage onLogin={() => { setAuthed(true); navigate('/') }} />
            </div>
          </div>
        } />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    )
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed}
        theme={darkMode ? 'dark' : 'light'}
        style={{ borderRight: `1px solid ${darkMode ? '#303030' : '#f0f0f0'}`, boxShadow: '2px 0 8px rgba(0,0,0,0.02)' }}>
        <div style={{
          height: 64, display: 'flex', alignItems: 'center', justifyContent: 'center',
          borderBottom: `1px solid ${darkMode ? '#303030' : '#f0f0f0'}`
        }}>
          <DatabaseOutlined style={{ fontSize: 24, color: themeToken.colorPrimary }} />
          {!collapsed && <Text strong style={{ marginLeft: 10, fontSize: 16, color: themeToken.colorPrimary }}>{t('app.name')}</Text>}
        </div>
        <Menu mode="inline" selectedKeys={[selectedKey]} items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ borderInlineEnd: 'none' }}
          theme={darkMode ? 'dark' : 'light'} aria-label={t('nav.dashboard')} />
      </Sider>
      <Layout>
        <Header style={{
          background: darkMode ? '#141414' : '#fff',
          padding: '0 24px', display: 'flex', alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: `1px solid ${darkMode ? '#303030' : '#f0f0f0'}`,
          height: 64
        }}>
          <Button type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)} />
          <Space>
            <Tooltip title={lang === 'en' ? 'Deutsch' : 'English'}>
              <Button type="text" icon={<GlobalOutlined />}
                onClick={() => setLanguage(lang === 'en' ? 'de' : 'en' as Lang)}>
                {lang.toUpperCase()}
              </Button>
            </Tooltip>
            <Tooltip title={darkMode ? 'Light Mode' : 'Dark Mode'}>
              <Button type="text"
                icon={darkMode ? <SunOutlined /> : <MoonOutlined />}
                onClick={toggleDarkMode} />
            </Tooltip>
            <Dropdown menu={{
              items: [{ key: 'logout', icon: <LogoutOutlined />, label: t('common.logout'), onClick: handleLogout }]
            }}>
              <Space style={{ cursor: 'pointer' }}>
                <Avatar size="small" icon={<UserOutlined />} style={{ backgroundColor: themeToken.colorPrimary }} />
                <Text>{t('nav.dashboard')}</Text>
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content style={{ margin: 24, overflow: 'auto' }}>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/connections" element={<Connections />} />
            <Route path="/explorer" element={<Explorer />} />
            <Route path="/query" element={<QueryEditor />} />
            <Route path="/admin/users" element={<AdminUsers />} />
            <Route path="/admin/roles" element={<AdminRoles />} />
            <Route path="/admin/apikeys" element={<APIKeys />} />
            <Route path="/admin/settings" element={<Settings />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  )
}

function LoginPage({ onLogin }: { onLogin: () => void }) {
  const navigate = useNavigate()
  const { darkMode } = useAppContext()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleLogin = async () => {
    setLoading(true)
    setError('')
    const res = await api.login({ username, password })
    if (res.success && res.data?.token) {
      api.setToken(res.data.token)
      onLogin()
    } else {
      setError(res.error?.message || t('login.error'))
    }
    setLoading(false)
  }

  return (
    <div>
      <div style={{ textAlign: 'center', marginBottom: 32 }}>
        <DatabaseOutlined style={{ fontSize: 48, color: '#6366f1' }} />
        <h1 style={{ margin: '16px 0 4px', fontSize: 24, fontWeight: 700 }}>{t('app.name')}</h1>
        <Text type="secondary">{t('app.tagline')}</Text>
      </div>
      {error && <div style={{ color: '#ff4d4f', marginBottom: 16, textAlign: 'center' }}>{error}</div>}
      <form onSubmit={e => { e.preventDefault(); handleLogin() }}>
        <input
          aria-label={t('login.username')}
          placeholder={t('login.username')}
          value={username}
          onChange={e => setUsername(e.target.value)}
          style={{
            width: '100%', height: 40, marginBottom: 12, padding: '8px 12px',
            border: '1px solid #d9d9d9', borderRadius: 8, fontSize: 14,
            background: darkMode ? '#1f1f1f' : '#fff', color: darkMode ? '#fff' : '#000'
          }}
        />
        <input
          type="password"
          aria-label={t('login.password')}
          placeholder={t('login.password')}
          value={password}
          onChange={e => setPassword(e.target.value)}
          style={{
            width: '100%', height: 40, marginBottom: 24, padding: '8px 12px',
            border: '1px solid #d9d9d9', borderRadius: 8, fontSize: 14,
            background: darkMode ? '#1f1f1f' : '#fff', color: darkMode ? '#fff' : '#000'
          }}
        />
        <button
          type="submit"
          disabled={loading}
          style={{
            width: '100%', height: 44, background: '#6366f1', color: '#fff',
            border: 'none', borderRadius: 8, fontSize: 16, fontWeight: 600,
            cursor: 'pointer', opacity: loading ? 0.7 : 1
          }}
        >
        {loading ? t('login.signing') : t('login.title')}
      </button>
    </form>
  </div>
  )
}
