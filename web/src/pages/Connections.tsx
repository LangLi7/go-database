import { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Select, Space, Tag, Typography, message, Popconfirm, Tooltip, Row, Col, Card, Tabs } from 'antd'
import { PlusOutlined, DeleteOutlined, ReloadOutlined, ApiOutlined, DatabaseOutlined, CodeOutlined, FolderOpenOutlined } from '@ant-design/icons'
import { api } from '../api/client'
import { useAppContext } from '../context/AppContext'
import { t } from '../i18n/translations'

const { Title, Text } = Typography
const { TextArea } = Input

const DB_TYPES = ['postgres', 'mysql', 'mariadb', 'sqlite', 'mongodb', 'redis']

const TYPE_DEFAULTS: Record<string, { port: number; host: string; db: string; hint: string }> = {
  postgres: { port: 5432, host: 'localhost', db: 'sampledb', hint: 'PostgreSQL 16+' },
  mysql: { port: 3306, host: 'localhost', db: 'sampledb', hint: 'MySQL 8+' },
  mariadb: { port: 3307, host: 'localhost', db: 'sampledb', hint: 'MariaDB 11+' },
  sqlite: { port: 0, host: '', db: 'mydb', hint: 'SQLite file-based' },
  mongodb: { port: 27017, host: 'localhost', db: 'sampledb', hint: 'MongoDB 7+' },
  redis: { port: 6379, host: 'localhost', db: '0', hint: 'Redis 7+' },
}

export default function Connections() {
  const { lang } = useAppContext()
  const [connections, setConnections] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [createDBOpen, setCreateDBOpen] = useState(false)
  const [dbTab, setDbTab] = useState('standalone')
  const [form] = Form.useForm()
  const [dbForm] = Form.useForm()
  const [standaloneForm] = Form.useForm()

  const load = async () => {
    setLoading(true)
    const r = await api.listConnections()
    if (r.success) setConnections(r.data || [])
    setLoading(false)
  }

  useEffect(() => { load() }, [])

  const handleTypeChange = (type: string) => {
    const def = TYPE_DEFAULTS[type]
    if (!def) return
    form.setFieldsValue({
      config: { host: def.host, port: def.port, database: def.db },
      source: def.host,
    })
  }

  const handleCreate = async () => {
    const vals = await form.validateFields()
    const r = await api.createConnection(vals)
    if (r.success) {
      message.success(t('conn.created'))
      setModalOpen(false)
      form.resetFields()
      load()
    } else {
      message.error(r.error?.message || 'Failed')
    }
  }

  const handleDelete = async (id: string) => {
    const r = await api.deleteConnection(id)
    if (r.success) { message.success(t('conn.deleted')); load() }
    else message.error(r.error?.message || 'Failed')
  }

  const handlePing = async (id: string) => {
    const r = await api.pingConnection(id)
    if (r.success) message.success(`${t('conn.ping_ok')}: ${r.data?.latency_ms || 0}ms`)
    else message.error(r.error?.message || 'Ping failed')
  }

  const handleCreateStandalone = async () => {
    const vals = await standaloneForm.validateFields()
    const r = await api.createStandaloneDatabase({ type: vals.type, name: vals.name })
    if (r.success) {
      message.success(`Database "${vals.name}" (${vals.type}) created`)
      setCreateDBOpen(false)
      standaloneForm.resetFields()
      load()
    } else {
      message.error(r.error?.message || 'Failed')
    }
  }

  const handleCreateDB = async () => {
    const vals = await dbForm.validateFields()
    const r = await api.createDatabase(vals.connection_id, vals.name)
    if (r.success) {
      message.success(`Database "${vals.name}" created`)
      setCreateDBOpen(false)
      dbForm.resetFields()
    } else {
      message.error(r.error?.message || 'Failed')
    }
  }

  const columns = [
    {
      title: t('conn.name'), dataIndex: 'name', key: 'name',
      render: (n: string, r: any) => <><ApiOutlined style={{ marginRight: 8, color: '#6366f1' }} /><a onClick={() => handlePing(r.id)}>{n}</a></>
    },
    { title: t('conn.type'), dataIndex: 'type', key: 'type', render: (t: string) => <Tag color="blue">{t}</Tag> },
    { title: t('conn.source'), dataIndex: 'source', key: 'source' },
    {
      title: t('conn.state'), dataIndex: 'state', key: 'state',
      render: (s: string) => <Tag color={s === 'connected' ? 'green' : s === 'error' ? 'red' : 'orange'}>{s}</Tag>
    },
    {
      title: t('conn.latency'), dataIndex: 'latency_ms', key: 'latency',
      render: (v: number) => v ? `${v}ms` : '-'
    },
    {
      title: t('conn.actions'), key: 'actions',
      render: (_: any, r: any) => (
        <Space>
          <Tooltip title={t('conn.ping')}><Button size="small" icon={<ReloadOutlined />} onClick={() => handlePing(r.id)} /></Tooltip>
          <Popconfirm title={t('common.confirm_delete')} onConfirm={() => handleDelete(r.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col flex="auto">
          <Title level={4} style={{ margin: 0 }}>{t('conn.title')}</Title>
          <Text type="secondary">{t('sample.hint')}</Text>
        </Col>
        <Col>
          <Space>
            <Button icon={<DatabaseOutlined />} onClick={() => setCreateDBOpen(true)}>{t('conn.create')}</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setModalOpen(true) }}>{t('conn.add')}</Button>
          </Space>
        </Col>
      </Row>

      <Card size="small" style={{ marginBottom: 16, background: '#fafafa' }}>
        <Space>
          <CodeOutlined />
          <Text type="secondary">{t('sample.quickstart_text')}</Text>
          <Tag style={{ fontFamily: 'monospace', cursor: 'pointer' }}
            onClick={() => { navigator.clipboard.writeText('docker-compose --profile samples up -d'); message.success('Copied!') }}>
            {t('sample.docker')}
          </Tag>
        </Space>
      </Card>

      <Table dataSource={connections} columns={columns} rowKey="id" loading={loading} size="middle" pagination={false} />

      <Modal title={t('conn.add')} open={modalOpen} onOk={handleCreate} onCancel={() => setModalOpen(false)}
        okText={t('common.create')} cancelText={t('common.cancel')} width={520}>
        <Form form={form} layout="vertical" initialValues={{ config: { host: 'localhost', port: 5432 } }}>
          <Form.Item name="name" label={t('conn.form.name')} rules={[{ required: true }]}>
            <Input placeholder={lang === 'de' ? 'Meine Datenbank' : 'My Database'} />
          </Form.Item>
          <Form.Item name="type" label={t('conn.type')} rules={[{ required: true }]}>
            <Select
              options={DB_TYPES.map(t => ({
                label: `${t.charAt(0).toUpperCase() + t.slice(1)} (${TYPE_DEFAULTS[t]?.hint || ''})`,
                value: t
              }))}
              onChange={handleTypeChange}
            />
          </Form.Item>
          <Form.Item name="source" label={t('conn.form.source')} rules={[{ required: true }]}>
            <Input placeholder={t('conn.example.source')} />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name={['config', 'host']} label={t('conn.form.host')}>
                <Input placeholder={t('conn.example.host')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name={['config', 'port']} label={t('conn.form.port')}>
                <Input type="number" placeholder={t('conn.example.port')} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name={['config', 'database']} label={t('conn.form.database')}>
            <Input placeholder={lang === 'de' ? 'z.B. sampledb' : 'e.g. sampledb'} />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name={['config', 'user']} label={t('conn.form.user')}>
                <Input placeholder={lang === 'de' ? 'z.B. postgres' : 'e.g. postgres'} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name={['config', 'password']} label={t('conn.form.password')}>
                <Input.Password placeholder="***" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name={['config', 'filepath']} label={t('conn.form.filepath')}>
            <Input placeholder={lang === 'de' ? 'pfad/zur/datenbank.db' : 'path/to/database.db'} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('conn.create')} open={createDBOpen} onCancel={() => { setCreateDBOpen(false); dbForm.resetFields(); standaloneForm.resetFields() }}
        footer={null} width={520}>
        <Tabs activeKey={dbTab} onChange={setDbTab} items={[
          {
            key: 'standalone',
            label: <span><FolderOpenOutlined /> {lang === 'de' ? 'Neue Datenbank' : 'New Database'}</span>,
            children: (
              <Form form={standaloneForm} layout="vertical">
                <Form.Item name="type" label={t('conn.type')} rules={[{ required: true }]}>
                  <Select
                    options={DB_TYPES.map(t => ({
                      label: `${t.charAt(0).toUpperCase() + t.slice(1)} (${TYPE_DEFAULTS[t]?.hint || ''})`,
                      value: t
                    }))}
                    placeholder={lang === 'de' ? 'Typ wählen' : 'Select type'}
                  />
                </Form.Item>
                <Form.Item name="name" label={lang === 'de' ? 'Datenbankname' : 'Database Name'} rules={[{ required: true }]}>
                  <Input placeholder={lang === 'de' ? 'z.B. meine_db' : 'e.g. my_database'} />
                </Form.Item>
                <div style={{ padding: '8px 0', background: '#f5f5f5', borderRadius: 6, marginBottom: 12, paddingLeft: 12 }}>
                  <Text type="secondary">
                    <FolderOpenOutlined /> database/storage/{'<type>'}/{'<name>'}
                  </Text>
                </div>
                <Button type="primary" onClick={handleCreateStandalone}>{t('common.create')}</Button>
              </Form>
            ),
          },
          {
            key: 'on_connection',
            label: <span><ApiOutlined /> {lang === 'de' ? 'Auf Verbindung' : 'On Connection'}</span>,
            children: (
              <Form form={dbForm} layout="vertical">
                <Form.Item name="connection_id" label={t('conn.title')} rules={[{ required: true }]}>
                  <Select
                    options={connections.filter(c => c.state === 'connected').map(c => ({ label: `${c.name} (${c.type})`, value: c.id }))}
                    placeholder={lang === 'de' ? 'Verbindung wählen' : 'Select connection'}
                  />
                </Form.Item>
                <Form.Item name="name" label={lang === 'de' ? 'Datenbankname' : 'Database Name'} rules={[{ required: true }]}>
                  <Input placeholder={lang === 'de' ? 'z.B. neue_datenbank' : 'e.g. new_database'} />
                </Form.Item>
                <Button type="primary" onClick={handleCreateDB}>{t('common.create')}</Button>
              </Form>
            ),
          },
        ]} />
      </Modal>
    </div>
  )
}
