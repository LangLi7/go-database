import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { Card, Select, Button, Input, Table, Space, Typography, message, Tag, Tooltip, Modal, Spin } from 'antd'

import { PlayCircleOutlined, ClearOutlined, HistoryOutlined, WarningOutlined, ExclamationCircleOutlined } from '@ant-design/icons'
import { api } from '../api/client'
import { useAppContext } from '../context/AppContext'
import { t } from '../i18n/translations'

const { Text, Title } = Typography
const { TextArea } = Input

interface Suggestion {
  text: string
  description: string
  type: string
  confidence: number
  risk_level: string
}

const RISK_COLORS: Record<string, string> = { LOW: 'green', MEDIUM: 'orange', HIGH: 'red' }
const TYPE_ICONS: Record<string, string> = { keyword: '🔤', table: '📋', column: '📊', statement: '⚡', intent: '💡' }

export default function QueryEditor() {
  const { lang } = useAppContext()
  const [connections, setConnections] = useState<any[]>([])
  const [connId, setConnId] = useState<string>('')

  const [sql, setSql] = useState('')
  const [result, setResult] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [history, setHistory] = useState<string[]>([])
  const [suggestions, setSuggestions] = useState<Suggestion[]>([])
  const [suggLoading, setSuggLoading] = useState(false)
  const [selectedSugg, setSelectedSugg] = useState(0)
  const [showSuggestions, setShowSuggestions] = useState(false)
  const [confirmModal, setConfirmModal] = useState<{ sql: string; risk: string; info: string } | null>(null)
  const textAreaRef = useRef<any>(null)
  const suggTimerRef = useRef<number>(0)

  useEffect(() => {
    let cancelled = false
    api.listConnections().then(r => { if (!cancelled && r.success) setConnections(r.data || []) })
    return () => {
      cancelled = true
      if (suggTimerRef.current) clearTimeout(suggTimerRef.current)
    }
  }, [])

  const fetchSuggestions = useCallback(async (input: string) => {
    if (!connId || input.length < 2) { setSuggestions([]); setShowSuggestions(false); return }
    setSuggLoading(true)
    const r = await api.getSuggestions(connId, input)
    if (r.success && r.data && r.data.length > 0) {
      setSuggestions(r.data as Suggestion[])
      setShowSuggestions(true)
      setSelectedSugg(0)
    } else {
      setSuggestions([])
      setShowSuggestions(false)
    }
    setSuggLoading(false)
  }, [connId])

  const handleInputChange = (val: string) => {
    setSql(val)
    setShowSuggestions(false)
    if (suggTimerRef.current) clearTimeout(suggTimerRef.current)
    suggTimerRef.current = window.setTimeout(() => fetchSuggestions(val), 300)
  }

  const applySuggestion = (s: Suggestion) => {
    const lastSpace = sql.lastIndexOf(' ')
    const prefix = lastSpace >= 0 ? sql.substring(0, lastSpace + 1) : ''
    setSql(prefix + s.text + ' ')
    setShowSuggestions(false)
    textAreaRef.current?.focus()
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (showSuggestions && suggestions.length > 0) {
      if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedSugg(i => Math.min(i + 1, suggestions.length - 1)); return }
      if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedSugg(i => Math.max(i - 1, 0)); return }
      if (e.key === 'Tab' || e.key === 'Enter') { e.preventDefault(); applySuggestion(suggestions[selectedSugg]); return }
      if (e.key === 'Escape') { setShowSuggestions(false); return }
    }
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) { e.preventDefault(); handleRun() }
  }

  const handleRun = async () => {
    if (!connId) { message.warning(lang === 'de' ? 'Verbindung wählen' : 'Select a connection'); return }
    if (!sql.trim()) { message.warning(lang === 'de' ? 'SQL eingeben' : 'Enter a query'); return }
    setShowSuggestions(false)
    setLoading(true)
    setError('')
    const r = await api.executeSafe(connId, sql.trim())
    if (r.success) {
      setResult(r.data)
      setHistory(h => [sql.trim(), ...h.slice(0, 19)])
    } else if (r.error?.code === 'CONFIRMATION_REQUIRED') {
      setConfirmModal({ sql: sql.trim(), risk: 'HIGH', info: r.error.message })
    } else if (r.error?.code === 'PERMISSION_DENIED') {
      setError(r.error.message)
      message.error(r.error.message)
    } else {
      setError(r.error?.message || 'Query failed')
    }
    setLoading(false)
  }

  const handleConfirmExecute = async () => {
    if (!confirmModal) return
    setLoading(true)
    const r = await api.executeSafe(connId, confirmModal.sql, true)
    setConfirmModal(null)
    if (r.success) {
      setResult(r.data)
      setHistory(h => [confirmModal.sql, ...h.slice(0, 19)])
      message.success(`Query completed`)
    } else {
      setError(r.error?.message || 'Query failed')
    }
    setLoading(false)
  }

  const columns = useMemo(() => result?.columns?.map((c: string) => ({
    title: c, dataIndex: c, key: c, ellipsis: true,
    render: (v: any) => v === null ? <Text type="secondary">NULL</Text> : String(v),
  })) || [], [result])

  const dataSource = useMemo(() => result?.rows?.map((row: any[], i: number) => {
    const obj: Record<string, any> = { _key: i }
    result.columns.forEach((col: string, ci: number) => { obj[col] = row[ci] })
    return obj
  }) || [], [result])

  return (
    <div>
      <Title level={4} style={{ marginBottom: 16 }}>{t('query.title')}</Title>
      <Space style={{ marginBottom: 12, width: '100%', flexWrap: 'wrap' }}>
        <Select
          placeholder={t('query.select_conn')}
          style={{ width: 300 }}
          value={connId || undefined}
          onChange={(val) => setConnId(val)}
          options={connections.map(c => ({ label: `${c.name} (${c.type})`, value: c.id }))}
        />
        <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleRun} loading={loading}>{t('query.run')}</Button>
        <Button icon={<ClearOutlined />} onClick={() => { setSql(''); setResult(null); setError(''); setSuggestions([]) }}>{t('query.clear')}</Button>
      </Space>

      <div style={{ position: 'relative' }}>
        <Card size="small" style={{ marginBottom: 12 }}>
          <TextArea
            ref={textAreaRef as any}
            rows={6}
            value={sql}
            onChange={e => handleInputChange(e.target.value)}
            onKeyDown={handleKeyDown}
            style={{ fontFamily: '"SF Mono", "Fira Code", "Consolas", monospace', fontSize: 13 }}
            placeholder={t('query.placeholder')}
          />
        </Card>
        {showSuggestions && suggestions.length > 0 && (
          <Card size="small" style={{
            position: 'absolute', top: '100%', left: 0, right: 0, zIndex: 100,
            maxHeight: 280, overflow: 'auto', boxShadow: '0 4px 12px rgba(0,0,0,0.15)'
          }}>
            {suggestions.map((s, i) => (
              <div key={i}
                onClick={() => applySuggestion(s)}
                onMouseEnter={() => setSelectedSugg(i)}
                style={{
                  padding: '6px 12px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8,
                  background: i === selectedSugg ? '#f0f5ff' : 'transparent', borderRadius: 4,
                }}>
                <Text style={{ fontFamily: 'monospace', fontSize: 13, flex: 1 }}>
                  {TYPE_ICONS[s.type] || '•'} {s.text}
                </Text>
                <Tag color={RISK_COLORS[s.risk_level] || 'default'} style={{ fontSize: 10, lineHeight: '16px' }}>{s.risk_level}</Tag>
                <Text type="secondary" style={{ fontSize: 11, maxWidth: 200 }}>{s.description}</Text>
                <Text style={{ fontSize: 11, color: '#6366f1', width: 36, textAlign: 'right' }}>
                  {Math.round(s.confidence * 100)}%
                </Text>
              </div>
            ))}
          </Card>
        )}
      </div>

      {error && (
        <Card size="small" style={{ marginBottom: 12, borderColor: '#ff4d4f' }}>
          <Text type="danger"><WarningOutlined /> {error}</Text>
        </Card>
      )}

      {result && (
        <Card size="small" title={
          <Space>
            <Tag color="blue">{result.rows?.length || 0} {t('query.rows')}</Tag>
            <Tag>{result.duration_ms}ms</Tag>
            {result.rows_affected !== undefined && <Tag>affected: {result.rows_affected}</Tag>}
          </Space>
        }>
          <Table dataSource={dataSource} columns={columns} rowKey="_key" size="small"
            scroll={{ x: 'max-content', y: 400 }} pagination={result.rows?.length > 50 ? { pageSize: 50 } : false} />
        </Card>
      )}

      {history.length > 0 && (
        <Card size="small" title={<><HistoryOutlined /> {t('query.history')}</>} style={{ marginTop: 12 }}>
          {history.map((q, i) => (
            <div key={i} style={{ padding: '4px 0', cursor: 'pointer', fontFamily: 'monospace', fontSize: 12 }}
              onClick={() => { setSql(q); setResult(null); setError('') }}>
              {q.substring(0, 120)}{q.length > 120 ? '...' : ''}
            </div>
          ))}
        </Card>
      )}

      <Modal
        title={<><ExclamationCircleOutlined style={{ color: '#faad14' }} /> High Risk Query</>}
        open={!!confirmModal}
        onOk={handleConfirmExecute}
        onCancel={() => setConfirmModal(null)}
        okText="Execute Anyway"
        okButtonProps={{ danger: true }}
        cancelText={t('common.cancel')}
      >
        <p>{confirmModal?.info}</p>
        <Card size="small" style={{ fontFamily: 'monospace', fontSize: 13, background: '#fffbe6' }}>
          {confirmModal?.sql}
        </Card>
        <p style={{ marginTop: 12, color: '#ff4d4f' }}>{t('common.undo')}</p>
      </Modal>
    </div>
  )
}
