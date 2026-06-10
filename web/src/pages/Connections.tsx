import { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Select, Space, Tag, Typography, message, Popconfirm, Tooltip, Row, Col, Card } from 'antd'
import { PlusOutlined, DeleteOutlined, ReloadOutlined, ApiOutlined, DatabaseOutlined, FolderOpenOutlined, CloudServerOutlined, LaptopOutlined } from '@ant-design/icons'
import { api } from '../api/client'
import { useAppContext } from '../context/AppContext'
import { t } from '../i18n/translations'

const { Title, Text } = Typography

const DB_TYPES = ['postgres', 'mysql', 'mariadb', 'sqlite', 'mongodb', 'redis']

const TYPE_DEFAULTS: Record<string, { port: number; host: string; db: string; hint: string; local: boolean }> = {
  postgres: { port: 5432, host: 'localhost', db: 'sampledb', hint: 'PostgreSQL 16+', local: true },
  mysql: { port: 3306, host: 'localhost', db: 'sampledb', hint: 'MySQL 8+', local: true },
  mariadb: { port: 3307, host: 'localhost', db: 'sampledb', hint: 'MariaDB 11+', local: true },
  sqlite: { port: 0, host: '', db: 'mydb', hint: 'SQLite file-based', local: false },
  mongodb: { port: 27017, host: 'localhost', db: 'sampledb', hint: 'MongoDB 7+', local: true },
  redis: { port: 6379, host: 'localhost', db: '0', hint: 'Redis 7+', local: true },
}

const SAMPLE_CONNECTIONS = [
  { name: 'Postgres Sample', type: 'postgres', file: 'postgres_sample.db', desc: 'E-Commerce: users, products, orders' },
  { name: 'MySQL Sample', type: 'mysql', file: 'mysql_sample.db', desc: 'Blog: users, posts, comments' },
  { name: 'MariaDB Sample', type: 'mariadb', file: 'mariadb_sample.db', desc: 'Inventory: warehouses, stock, suppliers' },
  { name: 'SQLite Sample', type: 'sqlite', file: 'sqlite_sample.db', desc: 'Tasks: projects, tasks, tags' },
  { name: 'MongoDB Sample', type: 'mongodb', file: 'mongodb_sample.db', desc: 'Media: movies, actors, reviews' },
  { name: 'Redis Sample', type: 'redis', file: 'redis_sample.db', desc: 'Session/Cache/Queue' },
]

export default function Connections() {
  const { lang } = useAppContext()
  const [connections, setConnections] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [addTab, setAddTab] = useState('connect')
  const [form] = Form.useForm()
  const [newDbForm] = Form.useForm()
  const [selectedNewType, setSelectedNewType] = useState<string>('sqlite')

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

	const handleCreateConnection = async () => {
		let vals: Record<string, any>
		try { vals = await form.validateFields() } catch { return }
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

	const handleCreateNewDatabase = async () => {
		let vals: Record<string, any>
		try { vals = await newDbForm.validateFields() } catch { return }
		const r = await api.createStandaloneDatabase({ type: vals.type, name: vals.name })
    if (r.success) {
      message.success(`Database "${vals.name}" (${vals.type}) created`)
      setModalOpen(false)
      newDbForm.resetFields()
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

  const handleAddSample = async (sample: typeof SAMPLE_CONNECTIONS[0]) => {
    const r = await api.createConnection({
      name: sample.name,
      type: sample.type,
      source: 'local',
      config: { filepath: `database/storage/${sample.type}/${sample.file}`, database: sample.name },
    })
    if (r.success) { message.success(`Sample "${sample.name}" added`); load() }
    else message.error(r.error?.message || 'Failed')
  }

  const columns = [
    {
      title: t('conn.name'), dataIndex: 'name', key: 'name',
      render: (n: string, r: any) => <><ApiOutlined style={{ marginRight: 8, color: '#6366f1' }} /><Button type="link" onClick={() => handlePing(r.id)}>{n}</Button></>
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
            <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); newDbForm.resetFields(); setAddTab('new'); setModalOpen(true) }}>
              {lang === 'de' ? 'Neue Datenbank' : 'New Database'}
            </Button>
            <Button icon={<CloudServerOutlined />} onClick={() => { form.resetFields(); newDbForm.resetFields(); setAddTab('connect'); setModalOpen(true) }}>
              {lang === 'de' ? 'Verbinden' : 'Connect'}
            </Button>
          </Space>
        </Col>
      </Row>

      <Card
        size="small"
        style={{ marginBottom: 16 }}
        title={<><DatabaseOutlined /> {lang === 'de' ? 'Beispiel-Datenbanken (lokal)' : 'Sample Databases (local)'}</>}
      >
        <Row gutter={[8, 8]}>
          {SAMPLE_CONNECTIONS.map(s => (
            <Col key={s.type}>
              <Tag
                style={{ cursor: 'pointer', padding: '4px 12px', fontSize: 13 }}
                color="blue"
                onClick={() => handleAddSample(s)}
              >
                <LaptopOutlined /> {s.name}
              </Tag>
            </Col>
          ))}
        </Row>
        <Text type="secondary" style={{ fontSize: 12, display: 'block', marginTop: 4 }}>
          {lang === 'de' ? 'Klicke auf eine Beispiel-DB, um sie als Verbindung hinzuzufügen' : 'Click a sample to add it as a connection'}
        </Text>
      </Card>

      <Table dataSource={connections} columns={columns} rowKey="id" loading={loading} size="middle" pagination={false} />

      <Modal
        title={addTab === 'new'
          ? (lang === 'de' ? 'Neue Datenbank erstellen' : 'Create New Database')
          : (lang === 'de' ? 'Verbindung hinzufügen' : 'Add Connection')}
        open={modalOpen}
        onCancel={() => { setModalOpen(false); form.resetFields(); newDbForm.resetFields() }}
        footer={null}
        width={520}
      >
        {addTab === 'connect' ? (
          <>
            <div style={{ marginBottom: 16, padding: '8px 12px', background: '#e6f7ff', borderRadius: 6 }}>
              <Text><CloudServerOutlined style={{ marginRight: 6 }} />
                {lang === 'de'
                  ? 'Verbinde zu einer bestehenden Datenbank (Postgres, MySQL, etc.)'
                  : 'Connect to an existing database (Postgres, MySQL, etc.)'}
              </Text>
            </div>
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
              <Button type="primary" onClick={handleCreateConnection}>{t('common.create')}</Button>
            </Form>
          </>
        ) : (
          <>
            <div style={{ marginBottom: 16, padding: '8px 12px', background: '#f6ffed', borderRadius: 6 }}>
              <Text><FolderOpenOutlined style={{ marginRight: 6 }} />
                {lang === 'de'
                  ? 'Erstelle eine neue lokale Datenbank. SQLite wird direkt als Datei angelegt. Für Postgres/MySQL/etc. wird localhost:Port + CREATE DATABASE probiert (Server muss laufen).'
                  : 'Create a new local database. SQLite creates a file directly. For Postgres/MySQL/etc. attempts localhost:port + CREATE DATABASE (server must be running).'}
              </Text>
            </div>
            <Form form={newDbForm} layout="vertical">
              <Form.Item name="type" label={t('conn.type')} rules={[{ required: true }]}>
                <Select
                  options={DB_TYPES.map(t => ({
                    label: `${t.charAt(0).toUpperCase() + t.slice(1)} (${TYPE_DEFAULTS[t]?.hint || ''})`,
                    value: t
                  }))}
                  placeholder={lang === 'de' ? 'Typ wählen' : 'Select type'}
                  onChange={(val) => setSelectedNewType(val)}
                />
              </Form.Item>
              <Form.Item name="name" label={lang === 'de' ? 'Datenbankname' : 'Database Name'} rules={[{ required: true }]}>
                <Input placeholder={lang === 'de' ? 'z.B. meine_db' : 'e.g. my_database'} />
              </Form.Item>
              {selectedNewType === 'sqlite' ? (
                <div style={{ padding: '8px 12px', background: '#f5f5f5', borderRadius: 6, marginBottom: 12 }}>
                  <Text style={{ color: '#595959' }}><FolderOpenOutlined /> database/storage/sqlite/{'<name>'}.db</Text>
                </div>
              ) : (
                <div style={{ padding: '8px 12px', background: '#fff7e6', borderRadius: 6, marginBottom: 12 }}>
                  <Text style={{ color: '#d46b08' }}>
                    <CloudServerOutlined style={{ marginRight: 6 }} />
                    {lang === 'de'
                      ? `${selectedNewType}://localhost:${TYPE_DEFAULTS[selectedNewType]?.port || '??'} – Ein ${selectedNewType}-Server muss lokal laufen`
                      : `${selectedNewType}://localhost:${TYPE_DEFAULTS[selectedNewType]?.port || '??'} – A ${selectedNewType} server must be running locally`}
                  </Text>
                </div>
              )}
              <Button type="primary" onClick={handleCreateNewDatabase}>{t('common.create')}</Button>
            </Form>
          </>
        )}
      </Modal>
    </div>
  )
}
