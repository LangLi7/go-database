import { useEffect, useState, useCallback, useMemo } from 'react'
import { Row, Col, Card, Tree, Table, Button, Modal, Input, Space, Tag, Typography, message, Spin, Select, Tooltip, Popconfirm, Empty } from 'antd'
import { DatabaseOutlined, TableOutlined, PlusOutlined, DeleteOutlined, ReloadOutlined, FolderOpenOutlined } from '@ant-design/icons'
import { api } from '../api/client'

const { Text, Title } = Typography
const { TextArea } = Input

interface TreeNode {
  title: string
  key: string
  icon: React.ReactNode
  children?: TreeNode[]
  isLeaf?: boolean
  dataType?: string
}

export default function Explorer() {
  const [connections, setConnections] = useState<any[]>([])
  const [treeData, setTreeData] = useState<TreeNode[]>([])
  const [selectedTable, setSelectedTable] = useState<string | null>(null)
  const [selectedConnId, setSelectedConnId] = useState<string | null>(null)
  const [browseData, setBrowseData] = useState<any>(null)
  const [browseLoading, setBrowseLoading] = useState(false)
  const [loading, setLoading] = useState(true)
  const [ddModal, setDdModal] = useState<{ type: 'database' | 'table'; action: 'create' | 'drop' } | null>(null)
  const [ddName, setDdName] = useState('')
  const [ddColumns, setDdColumns] = useState('')
  const [refreshing, setRefreshing] = useState<string | null>(null)

  const loadTree = useCallback(async () => {
    setLoading(true)
    const r = await api.listConnections()
    if (!r.success || !r.data) { setLoading(false); return }
    setConnections(r.data)
    const nodes: TreeNode[] = []
    for (const conn of r.data) {
      const connNode: TreeNode = {
        title: `${conn.name} (${conn.type})`,
        key: `conn:${conn.id}`,
        icon: <DatabaseOutlined />,
        children: [],
      }
      nodes.push(connNode)
    }
    setTreeData(nodes)
    setLoading(false)
  }, [])

  useEffect(() => { loadTree() }, [loadTree])

  const loadDatabases = async (connId: string, connNode: any) => {
    const idx = treeData.findIndex(n => n.key === connNode.key)
    if (idx === -1) return
    const r = await api.listDatabases(connId)
    if (!r.success || !r.data) { message.error(r.error?.message || 'Failed to load databases'); return }
    const dbs = r.data.map((db: string) => ({
      title: db, key: `db:${connId}:${db}`, icon: <FolderOpenOutlined />, isLeaf: false, children: [],
    }))
    const newTree = [...treeData]
    newTree[idx].children = dbs
    setTreeData(newTree)
  }

  const loadTables = async (connId: string, dbName: string, dbNode: any) => {
    const r = await api.listTables(connId)
    if (!r.success || !r.data) { message.error(r.error?.message || 'Failed to load tables'); return }
    const tables = r.data.map((t: string) => ({
      title: t, key: `tbl:${connId}:${t}`, icon: <TableOutlined />, isLeaf: true, dataType: 'table',
    }))
    const newTree = [...treeData]
    for (const conn of newTree) {
      if (conn.children) {
        for (const db of conn.children) {
          if (db.key === dbNode.key) {
            db.children = tables
          }
        }
      }
    }
    setTreeData(newTree)
  }

  const onSelect = async (keys: React.Key[], info: any) => {
    const key = keys[0] as string
    if (!key) return

    if (key.startsWith('conn:')) {
      const connId = key.split(':')[1]
      setSelectedConnId(connId)
      setSelectedTable(null)
      setBrowseData(null)
      loadDatabases(connId, info.node)
    } else if (key.startsWith('db:')) {
      const [, connId, dbName] = key.split(':')
      setSelectedConnId(connId)
      setSelectedTable(null)
      setBrowseData(null)
      loadTables(connId, dbName, info.node)
    } else if (key.startsWith('tbl:')) {
      const [, connId, table] = key.split(':')
      setSelectedConnId(connId)
      setSelectedTable(table)
      loadBrowseData(connId, table)
    }
  }

  const loadBrowseData = async (connId: string, table: string, page = 1) => {
    setBrowseLoading(true)
    const r = await api.browseTable(connId, table, { page, per_page: 50 })
    if (r.success) setBrowseData(r.data)
    else message.error(r.error?.message || 'Failed to load data')
    setBrowseLoading(false)
  }

  const handleRefresh = async (connId: string) => {
    setRefreshing(connId)
    const r = await api.pingConnection(connId)
    if (r.success) { message.success('Refreshed'); loadTree() }
    else message.error('Refresh failed')
    setRefreshing(null)
  }

  const handleCreateDatabase = async () => {
    if (!selectedConnId || !ddName) return
    const r = await api.createDatabase(selectedConnId, ddName)
    if (r.success) { message.success(`Database "${ddName}" created`); setDdModal(null); setDdName(''); loadTree() }
    else message.error(r.error?.message || 'Failed')
  }

  const handleDropDatabase = async () => {
    if (!selectedConnId || !ddName) return
    const r = await api.dropDatabase(selectedConnId, ddName)
    if (r.success) { message.success(`Database "${ddName}" dropped`); setDdModal(null); setDdName(''); loadTree() }
    else message.error(r.error?.message || 'Failed')
  }

  const handleCreateTable = async () => {
    if (!selectedConnId || !ddName || !ddColumns) return
    const r = await api.createTable(selectedConnId, ddName, ddColumns)
    if (r.success) { message.success(`Table "${ddName}" created`); setDdModal(null); setDdName(''); setDdColumns(''); loadTree() }
    else message.error(r.error?.message || 'Failed')
  }

  const handleDropTable = async () => {
    if (!selectedConnId || !ddName) return
    const r = await api.dropTable(selectedConnId, ddName)
    if (r.success) { message.success(`Table "${ddName}" dropped`); setDdModal(null); setDdName(''); setSelectedTable(null); setBrowseData(null); loadTree() }
    else message.error(r.error?.message || 'Failed')
  }

  const browseColumns = useMemo(() => browseData?.columns?.map((c: string) => ({
    title: c, dataIndex: c, key: c, ellipsis: true,
    render: (v: any) => v === null ? <Text type="secondary">NULL</Text> : String(v),
  })) || [], [browseData])

  const browseDataSource = useMemo(() => browseData?.data?.map((row: any[], i: number) => {
    const obj: Record<string, any> = { _key: i }
    browseData.columns.forEach((col: string, ci: number) => { obj[col] = row[ci] })
    return obj
  }) || [], [browseData])

  return (
    <div style={{ height: 'calc(100vh - 160px)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>Database Explorer</Title>
        <Space>
          <Select
            placeholder="Quick action"
            style={{ width: 200 }}
            onChange={(val: string) => {
              if (val === 'createdb') setDdModal({ type: 'database', action: 'create' })
              else if (val === 'dropdb') setDdModal({ type: 'database', action: 'drop' })
              else if (val === 'createtable') setDdModal({ type: 'table', action: 'create' })
              else if (val === 'droptable') setDdModal({ type: 'table', action: 'drop' })
            }}
            options={[
              { label: 'Create Database', value: 'createdb' },
              { label: 'Drop Database', value: 'dropdb' },
              { label: 'Create Table', value: 'createtable' },
              { label: 'Drop Table', value: 'droptable' },
            ]}
          />
          {selectedConnId && (
            <Button icon={<ReloadOutlined />} loading={!!refreshing} onClick={() => handleRefresh(selectedConnId)}>Refresh</Button>
          )}
        </Space>
      </div>
      <Row gutter={16} style={{ height: '100%' }}>
        <Col span={6}>
          <Card size="small" title="Explorer" style={{ height: '100%', overflow: 'auto' }}>
            {loading ? <Spin style={{ display: 'block', margin: '40px auto' }} /> : (
              treeData.length === 0 ? <Empty description="No connections" /> :
              <Tree treeData={treeData} onSelect={onSelect} showIcon defaultExpandAll aria-label="Database tree" />
            )}
          </Card>
        </Col>
        <Col span={18}>
          <Card size="small" title={selectedTable ? `Table: ${selectedTable}` : 'Select a table'} loading={browseLoading} style={{ height: '100%', overflow: 'auto' }}>
            {browseData ? (
              <Table
                dataSource={browseDataSource}
                columns={browseColumns}
                rowKey="_key"
                size="small"
                scroll={{ x: 'max-content', y: 'calc(100vh - 320px)' }}
                pagination={{
                  current: browseData.page || 1,
                  total: browseData.total || 0,
                  pageSize: browseData.per_page || 50,
                  onChange: (p) => selectedConnId && selectedTable && loadBrowseData(selectedConnId, selectedTable, p),
                  showTotal: (t) => `${t} rows`,
                }}
              />
            ) : (
              <Empty description="Select a table from the tree to browse data" />
            )}
          </Card>
        </Col>
      </Row>
      <Modal
        title={`${ddModal?.action === 'create' ? 'Create' : 'Drop'} ${ddModal?.type === 'database' ? 'Database' : 'Table'}`}
        open={!!ddModal}
        onOk={ddModal?.type === 'database' ? (ddModal?.action === 'create' ? handleCreateDatabase : handleDropDatabase) : (ddModal?.action === 'create' ? handleCreateTable : handleDropTable)}
        onCancel={() => { setDdModal(null); setDdName(''); setDdColumns('') }}
        okText={ddModal?.action === 'create' ? 'Create' : 'Drop'}
        okButtonProps={{ danger: ddModal?.action === 'drop' }}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Input placeholder="Name" value={ddName} onChange={e => setDdName(e.target.value)} />
          {ddModal?.type === 'table' && ddModal?.action === 'create' && (
            <TextArea rows={6} placeholder="Columns: id INT PRIMARY KEY, name VARCHAR(255), ..." value={ddColumns} onChange={e => setDdColumns(e.target.value)} />
          )}
          {ddModal?.action === 'drop' && (
            <Text type="danger">This action cannot be undone. Are you sure?</Text>
          )}
        </Space>
      </Modal>
    </div>
  )
}
