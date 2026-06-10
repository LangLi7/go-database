import { useEffect, useState } from 'react'
import { Row, Col, Card, Statistic, Table, Tag, Spin, Typography } from 'antd'
import { ApiOutlined, ThunderboltOutlined, TeamOutlined, DatabaseOutlined } from '@ant-design/icons'
import { api } from '../api/client'
import { t } from '../i18n/translations'

const { Title } = Typography

export default function Dashboard() {
  const [connections, setConnections] = useState<any[]>([])
  const [users, setUsers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    Promise.all([
      api.listConnections(),
      api.listUsers(),
    ]).then(([connRes, userRes]) => {
      if (cancelled) return
      if (connRes.success) setConnections(connRes.data || [])
      if (userRes.success) setUsers(userRes.data || [])
    }).catch(() => {
      if (!cancelled) setConnections([])
    }).finally(() => {
      if (!cancelled) setLoading(false)
    })
    return () => { cancelled = true }
  }, [])

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />

  const connected = connections.filter(c => c.state === 'connected')

  const columns = [
    { title: t('conn.name'), dataIndex: 'name', key: 'name' },
    { title: t('conn.type'), dataIndex: 'type', key: 'type', render: (t: string) => <Tag>{t}</Tag> },
    { title: t('conn.state'), dataIndex: 'state', key: 'state', render: (s: string) => <Tag color={s === 'connected' ? 'green' : s === 'error' ? 'red' : 'orange'}>{s}</Tag> },
    { title: t('conn.latency'), dataIndex: 'latency_ms', key: 'latency', render: (v: number) => v ? `${v}ms` : '-' },
    { title: t('conn.source'), dataIndex: 'source', key: 'source' },
  ]

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>{t('dashboard.title')}</Title>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title={t('dashboard.connections')} value={connections.length} prefix={<ApiOutlined />} valueStyle={{ color: '#6366f1' }} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title={t('dashboard.active')} value={connected.length} prefix={<ThunderboltOutlined />} valueStyle={{ color: '#22c55e' }} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title={t('dashboard.users')} value={users.length} prefix={<TeamOutlined />} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title={t('dashboard.databases')} value={connections.length} prefix={<DatabaseOutlined />} valueStyle={{ color: '#3b82f6' }} /></Card>
        </Col>
      </Row>
      <Card title={t('nav.connections')} style={{ marginTop: 24 }}>
        <Table dataSource={connections} columns={columns} rowKey="id" pagination={false} size="small" />
      </Card>
    </div>
  )
}
