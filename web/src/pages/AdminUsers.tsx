import { useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Select, Space, Typography, message, Popconfirm, Tag, Collapse, Checkbox, Empty, Input as AntInput } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, StopOutlined, CheckCircleOutlined, MinusCircleOutlined, LockOutlined } from '@ant-design/icons'
import { api } from '../api/client'

const { Title, Text } = Typography
const { Panel } = Collapse

type ToggleState = 'unset' | 'allow' | 'deny'

export default function AdminUsers() {
  const [users, setUsers] = useState<any[]>([])
  const [roles, setRoles] = useState<any[]>([])
  const [connections, setConnections] = useState<any[]>([])
  const [permGroups, setPermGroups] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<any>(null)
  const [form] = Form.useForm()
  const [selectedRole, setSelectedRole] = useState('')
  const [extraPerms, setExtraPerms] = useState<string[]>([])
  const [extraDBAccess, setExtraDBAccess] = useState<string[]>([])
  const [searchPerms, setSearchPerms] = useState('')
  const [saving, setSaving] = useState(false)

  const load = async () => {
    setLoading(true)
    const [u, r, c, p] = await Promise.all([
      api.listUsers(),
      api.listRoles(),
      api.listConnections(),
      api.getPermissionGroups(),
    ])
    if (u.success) setUsers(u.data || [])
    if (r.success) setRoles(r.data || [])
    if (c.success) setConnections(c.data || [])
    if (p.success) setPermGroups(p.data || [])
    setLoading(false)
  }

  useEffect(() => { load() }, [])

  const rolePerms = (): string[] => {
    const role = roles.find(r => r.id === selectedRole)
    return role?.permissions || []
  }

  const getToggleState = (key: string): ToggleState => {
    if (extraPerms.includes(key)) return 'allow'
    if (extraPerms.includes(`-${key}`)) return 'deny'
    return 'unset'
  }

  const cycleToggle = (key: string) => {
    setExtraPerms(prev => {
      const current = getToggleState(key)
      const filtered = prev.filter(p => p !== key && p !== `-${key}`)
      if (current === 'unset') return [...filtered, key]
      if (current === 'allow') return [...filtered, `-${key}`]
      return filtered
    })
  }

  const getToggleIcon = (key: string) => {
    const state = getToggleState(key)
    if (state === 'allow') return <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 16 }} />
    if (state === 'deny') return <StopOutlined style={{ color: '#ff4d4f', fontSize: 16 }} />
    return <MinusCircleOutlined style={{ color: '#d9d9d9', fontSize: 16 }} />
  }

  const isInherited = (key: string): boolean => {
    return rolePerms().includes(key)
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      const vals = await form.validateFields()
      const data = {
        username: vals.username,
        password: vals.password || undefined,
        role: vals.role,
        extra_perm: extraPerms.filter(p => !p.startsWith('-')),
        extra_db_access: extraDBAccess,
      }
      const r = editing
        ? await api.updateUser(editing.id, data)
        : await api.createUser(data)
      if (r.success) {
        message.success(editing ? 'Updated' : 'Created')
        setModalOpen(false)
        form.resetFields()
        setEditing(null)
        setExtraPerms([])
        setExtraDBAccess([])
        load()
      } else {
        message.error(r.error?.message || 'Failed')
      }
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string) => {
    const r = await api.deleteUser(id)
    if (r.success) { message.success('Deleted'); load() }
    else message.error(r.error?.message || 'Failed')
  }

  const openEdit = (user: any) => {
    setEditing(user)
    setSelectedRole(user.role || '')
    setExtraPerms(user.extra_perm || [])
    setExtraDBAccess(user.extra_db_access || [])
    form.setFieldsValue({
      username: user.username,
      role: user.role,
      password: '',
    })
    setSearchPerms('')
    setModalOpen(true)
  }

  const openCreate = () => {
    setEditing(null)
    setSelectedRole('')
    setExtraPerms([])
    setExtraDBAccess([])
    form.resetFields()
    setSearchPerms('')
    setModalOpen(true)
  }

  const filteredGroups = permGroups.map((g: any) => ({
    ...g,
    children: g.children.filter((c: any) =>
      !searchPerms || c.display_name.toLowerCase().includes(searchPerms.toLowerCase()) || c.key.toLowerCase().includes(searchPerms.toLowerCase())
    ),
  })).filter((g: any) => g.children.length > 0)

  const columns = [
    { title: 'Username', dataIndex: 'username', key: 'username' },
    {
      title: 'Role', dataIndex: 'role', key: 'role',
      render: (r: string) => <Tag color="blue">{roles.find(role => role.id === r)?.name || r}</Tag>,
    },
    {
      title: 'Overrides', dataIndex: 'extra_perm', key: 'extra_perm',
      render: (p: string[]) => p?.length ? <Tag color="green">{p.length} overrides</Tag> : '-',
    },
    {
      title: 'Actions', key: 'actions',
      render: (_: any, r: any) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(r)} />
          <Popconfirm title="Delete this user?" onConfirm={() => handleDelete(r.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Users</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Add User</Button>
      </div>
      <Table dataSource={users} columns={columns} rowKey="id" loading={loading} size="middle" />
      <Modal
        title={editing ? `Edit User: ${editing.username}` : 'New User'}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => { setModalOpen(false); setEditing(null) }}
        okText={editing ? 'Update' : 'Create'}
        width={720}
        confirmLoading={saving}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="username" label="Username" rules={[{ required: true }]}><Input disabled={!!editing} /></Form.Item>
          <Form.Item name="password" label="Password" rules={editing ? [] : [{ required: true }]}><Input.Password placeholder={editing ? 'Leave blank to keep current' : ''} /></Form.Item>
          <Form.Item name="role" label="Role" rules={[{ required: true }]}>
            <Select
              placeholder="Select role"
              onChange={(val) => { setSelectedRole(val); setExtraPerms(prev => prev) }}
              options={roles.map(r => ({ label: r.name, value: r.id }))}
            />
          </Form.Item>
          <div style={{ border: '1px solid #f0f0f0', borderRadius: 8, padding: 12, marginBottom: 12 }}>
            <Text strong>Database Access (Per-User)</Text>
            <div style={{ marginTop: 8 }}>
              <Select
                mode="multiple"
                style={{ width: '100%' }}
                placeholder="Select extra database access"
                value={extraDBAccess}
                onChange={setExtraDBAccess}
                options={connections.map(c => ({ label: `${c.name} (${c.type})`, value: c.id }))}
                allowClear
                showSearch
                filterOption={(input, option) => (option?.label as string || '').toLowerCase().includes(input.toLowerCase())}
              />
            </div>
          </div>
          <div style={{ border: '1px solid #f0f0f0', borderRadius: 8, padding: 12 }}>
            <Space style={{ marginBottom: 8 }}>
              <Text strong>Permission Overrides</Text>
              <Tag>Role: {roles.find(r => r.id === selectedRole)?.name || 'none'}</Tag>
            </Space>
            <AntInput.Search
              placeholder="Search permissions..."
              value={searchPerms}
              onChange={e => setSearchPerms(e.target.value)}
              style={{ marginBottom: 8 }}
              allowClear
            />
            {filteredGroups.length > 0 ? (
              <Collapse ghost defaultActiveKey={filteredGroups.map((g: any) => g.name)} style={{ maxHeight: 300, overflow: 'auto' }}>
                {filteredGroups.map((group: any) => (
                  <Panel
                    key={group.name}
                    header={
                      <Space>
                        <span>{group.icon}</span>
                        <Text strong>{group.display_name}</Text>
                      </Space>
                    }
                  >
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                      {group.children.map((entry: any) => {
                        const inherited = isInherited(entry.key)
                        const state = getToggleState(entry.key)
                        return (
                          <div
                            key={entry.key}
                            onClick={() => !inherited && cycleToggle(entry.key)}
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 8,
                              padding: '4px 8px',
                              cursor: inherited ? 'not-allowed' : 'pointer',
                              borderRadius: 4,
                              opacity: inherited && state === 'unset' ? 0.6 : 1,
                            }}
                          >
                            {inherited ? (
                              <LockOutlined style={{ color: '#faad14', fontSize: 14 }} />
                            ) : (
                              <span onClick={(e) => { e.stopPropagation(); cycleToggle(entry.key) }} style={{ cursor: 'pointer' }}>
                                {getToggleIcon(entry.key)}
                              </span>
                            )}
                            <div>
                              <Text style={{ fontSize: 13 }}>{entry.display_name}</Text>
                              <div><Text type="secondary" style={{ fontSize: 11 }}>{entry.description}</Text></div>
                            </div>
                            {inherited && <Tag color="gold" style={{ marginLeft: 'auto', fontSize: 10 }}>inherited</Tag>}
                            {state === 'allow' && !inherited && <Tag color="green" style={{ marginLeft: 'auto', fontSize: 10 }}>override</Tag>}
                            {state === 'deny' && !inherited && <Tag color="red" style={{ marginLeft: 'auto', fontSize: 10 }}>denied</Tag>}
                          </div>
                        )
                      })}
                    </div>
                  </Panel>
                ))}
              </Collapse>
            ) : (
              <Empty description="No permissions match" style={{ margin: '20px 0' }} />
            )}
          </div>
        </Form>
      </Modal>
    </div>
  )
}
