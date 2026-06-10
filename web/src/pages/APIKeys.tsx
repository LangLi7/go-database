import { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Space, Typography, message, Popconfirm, Tag } from 'antd'
import { PlusOutlined, DeleteOutlined, CopyOutlined, EyeOutlined, EyeInvisibleOutlined } from '@ant-design/icons'
import { api } from '../api/client'

const { Title, Text } = Typography

export default function APIKeys() {
  const [keys, setKeys] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [newKey, setNewKey] = useState<any>(null)
  const [showKey, setShowKey] = useState(false)
  const [form] = Form.useForm()

  const load = async () => {
    setLoading(true)
    const r = await api.listAPIKeys()
    if (r.success) setKeys(r.data || [])
    setLoading(false)
  }

  useEffect(() => { load() }, [])

	const handleCreate = async () => {
		let vals: Record<string, any>
		try { vals = await form.validateFields() } catch { return }
		const r = await api.createAPIKey({ name: vals.name || '', permissions: vals.permissions || [] })
    if (r.success && r.data) {
      setNewKey(r.data)
      message.success('API Key created')
      form.resetFields()
      load()
    } else {
      message.error(r.error?.message || 'Failed')
    }
  }

  const handleDelete = async (prefix: string) => {
    const r = await api.deleteAPIKey(prefix)
    if (r.success) { message.success('Deleted'); load() }
    else message.error(r.error?.message || 'Failed')
  }

  const copyKey = (key: string) => {
    navigator.clipboard.writeText(key).then(() => message.success('Copied!'))
  }

  const columns = [
    { title: 'Name', dataIndex: 'name', key: 'name' },
    { title: 'Prefix', dataIndex: 'prefix', key: 'prefix', render: (p: string) => <Tag>{p}...</Tag> },
    { title: 'Created', dataIndex: 'created_at', key: 'created_at' },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, r: any) => (
        <Popconfirm title="Revoke this API key?" onConfirm={() => handleDelete(r.prefix)}>
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>API Keys</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { setNewKey(null); form.resetFields(); setModalOpen(true) }}>Create Key</Button>
      </div>
      <Table dataSource={keys} columns={columns} rowKey="prefix" loading={loading} size="middle" />
      <Modal title="Create API Key" open={modalOpen} onOk={newKey ? undefined : handleCreate} onCancel={() => { setModalOpen(false); setNewKey(null) }} okText="Create" footer={newKey ? undefined : undefined}>
        {newKey ? (
          <div>
            <Text strong>Your API Key (shown once):</Text>
            <div style={{ margin: '12px 0', padding: 12, background: '#f5f5f5', borderRadius: 8, wordBreak: 'break-all', fontFamily: 'monospace', display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ flex: 1 }}>{showKey ? newKey.raw_key : '••••••••••••'}</span>
              <Button type="text" icon={showKey ? <EyeInvisibleOutlined /> : <EyeOutlined />} onClick={() => setShowKey(!showKey)} />
            </div>
            <Button icon={<CopyOutlined />} onClick={() => copyKey(newKey.raw_key)}>Copy</Button>
            <Button style={{ marginLeft: 8 }} onClick={() => { setModalOpen(false); setNewKey(null) }}>Close</Button>
          </div>
        ) : (
          <Form form={form} layout="vertical">
            <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
          </Form>
        )}
      </Modal>
    </div>
  )
}
