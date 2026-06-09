import { useEffect, useState, useCallback } from 'react'
import { Row, Col, Card, Button, Modal, Form, Input, Space, Typography, message, Popconfirm, Collapse, Checkbox, Select, Tag, Empty, Input as AntInput } from 'antd'
import { PlusOutlined, DeleteOutlined, EditOutlined, SaveOutlined } from '@ant-design/icons'
import { api } from '../api/client'

const { Title, Text } = Typography
const { Panel } = Collapse

export default function AdminRoles() {
  const [roles, setRoles] = useState<any[]>([])
  const [connections, setConnections] = useState<any[]>([])
  const [permGroups, setPermGroups] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedRole, setSelectedRole] = useState<any>(null)
  const [editingName, setEditingName] = useState('')
  const [selectedPerms, setSelectedPerms] = useState<string[]>([])
  const [selectedDBAccess, setSelectedDBAccess] = useState<string[]>([])
  const [searchPerms, setSearchPerms] = useState('')
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [createForm] = Form.useForm()

  const load = useCallback(async () => {
    setLoading(true)
    const [r, c, p] = await Promise.all([
      api.listRoles(),
      api.listConnections(),
      api.getPermissionGroups(),
    ])
    if (r.success) setRoles(r.data || [])
    if (c.success) setConnections(c.data || [])
    if (p.success) setPermGroups(p.data || [])
    setLoading(false)
  }, [])

  useEffect(() => { load() }, [load])

  const selectRole = (role: any) => {
    setSelectedRole(role)
    setEditingName(role.name)
    setSelectedPerms(role.permissions || [])
    setSelectedDBAccess(role.db_access || [])
    setSearchPerms('')
  }

  const togglePerm = (key: string) => {
    setSelectedPerms(prev =>
      prev.includes(key) ? prev.filter(p => p !== key) : [...prev, key]
    )
  }

  const saveRole = async () => {
    if (!selectedRole) return
    const r = await api.updateRole(selectedRole.id, {
      name: editingName,
      permissions: selectedPerms,
      db_access: selectedDBAccess,
    })
    if (r.success) {
      message.success('Role updated')
      load()
      setSelectedRole((prev: any) => ({ ...prev, name: editingName, permissions: selectedPerms, db_access: selectedDBAccess }))
    } else {
      message.error(r.error?.message || 'Failed')
    }
  }

  const handleCreate = async () => {
    const vals = await createForm.validateFields()
    const r = await api.createRole({
      name: vals.name,
      permissions: [],
      db_access: [],
    })
    if (r.success) {
      message.success('Created')
      setCreateModalOpen(false)
      createForm.resetFields()
      load()
    } else {
      message.error(r.error?.message || 'Failed')
    }
  }

  const handleDelete = async (id: string) => {
    const r = await api.deleteRole(id)
    if (r.success) {
      message.success('Deleted')
      if (selectedRole?.id === id) setSelectedRole(null)
      load()
    } else {
      message.error(r.error?.message || 'Failed')
    }
  }

  const isModified = () => {
    if (!selectedRole) return false
    return editingName !== selectedRole.name ||
      JSON.stringify(selectedPerms.sort()) !== JSON.stringify((selectedRole.permissions || []).sort()) ||
      JSON.stringify(selectedDBAccess.sort()) !== JSON.stringify((selectedRole.db_access || []).sort())
  }

  const filteredGroups = permGroups.map((g: any) => ({
    ...g,
    children: g.children.filter((c: any) =>
      !searchPerms || c.display_name.toLowerCase().includes(searchPerms.toLowerCase()) || c.key.toLowerCase().includes(searchPerms.toLowerCase())
    ),
  })).filter((g: any) => g.children.length > 0)

  const allPermKeys = permGroups.flatMap((g: any) => g.children.map((c: any) => c.key))

  return (
    <div>
      <Title level={4} style={{ marginBottom: 16 }}>Roles</Title>
      <Row gutter={16} style={{ height: 'calc(100vh - 180px)' }}>
        <Col span={6}>
          <Card size="small" title="Roles" extra={
            <Button type="primary" size="small" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)} />
          } bodyStyle={{ padding: 0, overflow: 'auto', maxHeight: 'calc(100vh - 240px)' }}>
            {roles.map(role => (
              <div
                key={role.id}
                onClick={() => selectRole(role)}
                style={{
                  padding: '10px 16px',
                  cursor: 'pointer',
                  borderLeft: selectedRole?.id === role.id ? '3px solid #1890ff' : '3px solid transparent',
                  background: selectedRole?.id === role.id ? '#e6f7ff' : 'transparent',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  borderBottom: '1px solid #f0f0f0',
                }}
              >
                <div>
                  <Text strong>{role.name}</Text>
                  <div><Text type="secondary" style={{ fontSize: 12 }}>{role.id}</Text></div>
                </div>
                <Popconfirm title="Delete role?" onConfirm={(e) => { e?.stopPropagation(); handleDelete(role.id) }}>
                  <Button size="small" danger icon={<DeleteOutlined />} type="text" onClick={(e) => e.stopPropagation()} />
                </Popconfirm>
              </div>
            ))}
            {roles.length === 0 && <div style={{ padding: 24, textAlign: 'center' }}><Text type="secondary">No roles</Text></div>}
          </Card>
        </Col>
        <Col span={18}>
          {selectedRole ? (
            <Card
              size="small"
              title={
                <Space>
                  <Input
                    value={editingName}
                    onChange={e => setEditingName(e.target.value)}
                    style={{ width: 200, fontWeight: 600 }}
                    bordered={false}
                    onPressEnter={saveRole}
                  />
                  <Text type="secondary" style={{ fontSize: 12 }}>{selectedRole.id}</Text>
                </Space>
              }
              extra={
                <Space>
                  {isModified() && <Button type="primary" size="small" icon={<SaveOutlined />} onClick={saveRole}>Save Changes</Button>}
                </Space>
              }
              bodyStyle={{ padding: 0, height: 'calc(100vh - 240px)', display: 'flex', flexDirection: 'column' }}
            >
              <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0' }}>
                <Text strong>Database Access</Text>
                <div style={{ marginTop: 8 }}>
                  <Select
                    mode="multiple"
                    style={{ width: '100%' }}
                    placeholder="Select connections this role can access"
                    value={selectedDBAccess}
                    onChange={setSelectedDBAccess}
                    options={connections.map(c => ({ label: `${c.name} (${c.type})`, value: c.id }))}
                    allowClear
                    showSearch
                    filterOption={(input, option) => (option?.label as string || '').toLowerCase().includes(input.toLowerCase())}
                  />
                </div>
              </div>
              <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0' }}>
                <Text strong>Permissions</Text>
                <AntInput.Search
                  placeholder="Search permissions..."
                  value={searchPerms}
                  onChange={e => setSearchPerms(e.target.value)}
                  style={{ marginTop: 8 }}
                  allowClear
                />
              </div>
              <div style={{ flex: 1, overflow: 'auto', padding: '8px 0' }}>
                {filteredGroups.length > 0 ? (
                  <Collapse ghost defaultActiveKey={filteredGroups.map((g: any) => g.name)}>
                    {filteredGroups.map((group: any) => (
                      <Panel
                        key={group.name}
                        header={
                          <Space>
                            <span>{group.icon}</span>
                            <Text strong>{group.display_name}</Text>
                            <Tag style={{ marginLeft: 8 }}>{group.children.filter((c: any) => selectedPerms.includes(c.key)).length}/{group.children.length}</Tag>
                          </Space>
                        }
                      >
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                          {group.children.map((entry: any) => (
                            <div
                              key={entry.key}
                              onClick={() => togglePerm(entry.key)}
                              style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: 8,
                                padding: '4px 8px',
                                cursor: 'pointer',
                                borderRadius: 4,
                                background: selectedPerms.includes(entry.key) ? '#f6ffed' : 'transparent',
                              }}
                            >
                              <Checkbox checked={selectedPerms.includes(entry.key)} />
                              <div>
                                <Text style={{ fontSize: 13 }}>{entry.display_name}</Text>
                                <div><Text type="secondary" style={{ fontSize: 11 }}>{entry.description}</Text></div>
                              </div>
                              <Tag style={{ marginLeft: 'auto', fontSize: 10 }}>{entry.key}</Tag>
                            </div>
                          ))}
                        </div>
                      </Panel>
                    ))}
                  </Collapse>
                ) : (
                  <Empty description="No permissions match your search" style={{ marginTop: 40 }} />
                )}
              </div>
            </Card>
          ) : (
            <Card style={{ height: 'calc(100vh - 240px)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <Empty description="Select a role to edit its permissions" />
            </Card>
          )}
        </Col>
      </Row>
      <Modal title="New Role" open={createModalOpen} onOk={handleCreate} onCancel={() => setCreateModalOpen(false)} okText="Create">
        <Form form={createForm} layout="vertical">
          <Form.Item name="name" label="Role Name" rules={[{ required: true }]}><Input placeholder="e.g. analyst" /></Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
