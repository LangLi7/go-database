import type { APIResponse, LoginRequest, TokenResponse, Connection, QueryResult, BrowseResult, User, Role, APIKey, APIKeyCreateResponse, Stats } from './types'

const BASE = '/api/v1'

class APIClient {
  private token: string | null = null
  private redirectCallback: (() => void) | null = null

  constructor() {
    const saved = localStorage.getItem('token')
    if (saved) this.token = saved
  }

  onUnauthorized(cb: () => void) {
    this.redirectCallback = cb
  }

  setToken(token: string | null) {
    this.token = token
    if (token) localStorage.setItem('token', token)
    else localStorage.removeItem('token')
  }

  getToken(): string | null { return this.token }

  isAuthenticated(): boolean { return this.token !== null }

  async verifySession(): Promise<boolean> {
    if (!this.token) return false
    const r = await this.request('GET', '/auth/verify')
    if (!r.success) {
      this.setToken(null)
      return false
    }
    return true
  }

  private async request<T>(method: string, path: string, body?: any): Promise<APIResponse<T>> {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    if (this.token) headers['Authorization'] = `Bearer ${this.token}`
    try {
      const res = await fetch(`${BASE}${path}`, { method, headers, body: body ? JSON.stringify(body) : undefined })
      if (res.status === 401) {
        this.setToken(null)
        if (this.redirectCallback) this.redirectCallback()
        return { success: false, error: { code: 'UNAUTHORIZED', message: 'Session expired' } }
      }
      if (res.status === 204) return { success: true }
      try {
        return await res.json()
      } catch {
        return { success: false, error: { code: 'PARSE_ERROR', message: `Invalid response: ${res.status}` } }
      }
    } catch (err) {
      return { success: false, error: { code: 'NETWORK_ERROR', message: err instanceof Error ? err.message : 'Network request failed' } }
    }
  }

  login(data: LoginRequest) { return this.request<TokenResponse>('POST', '/auth/login', data) }
  refreshToken(token: string) { return this.request<TokenResponse>('POST', '/auth/refresh', { token }) }
  verifyToken() { return this.request<any>('GET', '/auth/verify') }
  changePassword(oldPw: string, newPw: string) { return this.request('POST', '/auth/change-password', { old_password: oldPw, new_password: newPw }) }

  listConnections() { return this.request<Connection[]>('GET', '/connections') }
  getConnection(id: string) { return this.request<Connection>('GET', `/connections/${id}`) }
  createConnection(data: Partial<Connection>) { return this.request<Connection>('POST', '/connections', data) }
  deleteConnection(id: string) { return this.request('DELETE', `/connections/${id}`) }
  pingConnection(id: string) { return this.request<{ latency_ms: number }>('GET', `/connections/${id}/ping`) }

  createStandaloneDatabase(data: { type: string; name: string }) { return this.request<{ id: string; name: string; type: string }>('POST', '/databases/standalone', data) }
  listDatabases(id: string) { return this.request<string[]>('GET', `/connections/${id}/databases`) }
  createDatabase(id: string, name: string) { return this.request<{ name: string }>('POST', `/connections/${id}/databases`, { name }) }
  dropDatabase(id: string, name: string) { return this.request('DELETE', `/connections/${id}/databases/${encodeURIComponent(name)}`) }

  listTables(id: string) { return this.request<string[]>('GET', `/connections/${id}/tables`) }
  getSchema(id: string) { return this.request<{ tables: any[] }>('GET', `/connections/${id}/schema`) }
  createTable(id: string, name: string, columns: string) { return this.request<{ name: string }>('POST', `/connections/${id}/tables`, { name, columns }) }
  dropTable(id: string, name: string) { return this.request('DELETE', `/connections/${id}/tables/${encodeURIComponent(name)}`) }

  query(id: string, sql: string) { return this.request<QueryResult>('POST', `/connections/${id}/query`, { query: sql }) }
  execute(id: string, sql: string) { return this.request<QueryResult>('POST', `/connections/${id}/execute`, { query: sql }) }

  browseTable(id: string, table: string, params?: { page?: number; per_page?: number; sort?: string; dir?: string; filter?: string }) {
    const qs = new URLSearchParams()
    if (params) { Object.entries(params).forEach(([k, v]) => { if (v) qs.set(k, String(v)) }) }
    const q = qs.toString() ? `?${qs.toString()}` : ''
    return this.request<BrowseResult>('GET', `/connections/${id}/browse/${table}${q}`)
  }

  insertRow(id: string, table: string, data: Record<string, any>) { return this.request<QueryResult>('POST', `/connections/${id}/row/${table}`, data) }
  updateRow(id: string, table: string, pk: string, val: string, data: Record<string, any>) { return this.request<QueryResult>('PUT', `/connections/${id}/row/${table}/${pk}/${val}`, data) }
  deleteRow(id: string, table: string, pk: string, val: string) { return this.request<QueryResult>('DELETE', `/connections/${id}/row/${table}/${pk}/${val}`) }

  getStats() { return this.request<Stats>('GET', '/admin/stats') }
  getActivity() { return this.request<{ timestamp: string; event: string; user: string }[]>('GET', '/admin/activity') }

  listUsers() { return this.request<User[]>('GET', '/admin/users') }
  createUser(data: { username: string; password: string; role: string }) { return this.request<User>('POST', '/admin/users', data) }
  updateUser(id: string, data: Partial<User>) { return this.request<User>('PUT', `/admin/users/${id}`, data) }
  deleteUser(id: string) { return this.request('DELETE', `/admin/users/${id}`) }
  getUserPermissions(id: string) { return this.request<{ permissions: string[] }>('GET', `/admin/users/${id}/permissions`) }
  setUserPermissions(id: string, data: { permissions: string[] }) { return this.request<{ permissions: string[] }>('PUT', `/admin/users/${id}/permissions`, data) }

  listRoles() { return this.request<Role[]>('GET', '/admin/roles') }
  createRole(data: { name: string; permissions: string[]; db_access: string[] }) { return this.request<Role>('POST', '/admin/roles', data) }
  updateRole(id: string, data: Partial<Role>) { return this.request<Role>('PUT', `/admin/roles/${id}`, data) }
  deleteRole(id: string) { return this.request('DELETE', `/admin/roles/${id}`) }
  setRolePermissions(id: string, data: { permissions: string[]; db_access: string[] }) { return this.request<Role>('PUT', `/admin/roles/${id}/permissions`, data) }
  getPermissionGroups() { return this.request<{ key: string; display_name: string; children: { key: string; display_name: string }[] }[]>('GET', '/admin/permission-groups') }

  listAPIKeys() { return this.request<APIKey[]>('GET', '/apikeys') }
  createAPIKey(data: { name: string; permissions: string[] }) { return this.request<APIKeyCreateResponse>('POST', '/apikeys', data) }
  deleteAPIKey(prefix: string) { return this.request('DELETE', `/apikeys/${prefix}`) }

  getSuggestions(connId: string, input: string, table?: string) { return this.request<{ text: string; type: string; confidence: number }[]>('POST', '/suggest', { connection_id: connId, input, current_table: table || '' }) }
  executeSafe(connId: string, sql: string, confirmHigh?: boolean) { return this.request<QueryResult>('POST', '/execute/safe', { connection_id: connId, sql, confirm_high: confirmHigh || false }) }
}

export const api = new APIClient()
