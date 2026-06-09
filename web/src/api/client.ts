import type { APIResponse, LoginRequest, TokenResponse } from './types'

const BASE = '/api/v1'

class APIClient {
  private token: string | null = null

  constructor() {
    const saved = localStorage.getItem('token')
    if (saved) this.token = saved
  }

  setToken(token: string | null) {
    this.token = token
    if (token) localStorage.setItem('token', token)
    else localStorage.removeItem('token')
  }

  getToken(): string | null { return this.token }

  isAuthenticated(): boolean { return this.token !== null }

  private async request<T>(method: string, path: string, body?: any): Promise<APIResponse<T>> {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    if (this.token) headers['Authorization'] = `Bearer ${this.token}`
    const res = await fetch(`${BASE}${path}`, { method, headers, body: body ? JSON.stringify(body) : undefined })
    if (res.status === 204) return { success: true }
    return res.json()
  }

  login(data: LoginRequest) { return this.request<TokenResponse>('POST', '/auth/login', data) }
  refreshToken(token: string) { return this.request<TokenResponse>('POST', '/auth/refresh', { token }) }
  changePassword(oldPw: string, newPw: string) { return this.request('POST', '/auth/change-password', { old_password: oldPw, new_password: newPw }) }

  listConnections() { return this.request<any[]>('GET', '/connections') }
  getConnection(id: string) { return this.request<any>('GET', `/connections/${id}`) }
  createConnection(data: any) { return this.request<any>('POST', '/connections', data) }
  deleteConnection(id: string) { return this.request('DELETE', `/connections/${id}`) }
  pingConnection(id: string) { return this.request<any>('GET', `/connections/${id}/ping`) }

  createStandaloneDatabase(data: any) { return this.request<any>('POST', '/databases/standalone', data) }
  listDatabases(id: string) { return this.request<string[]>('GET', `/connections/${id}/databases`) }
  createDatabase(id: string, name: string) { return this.request<any>('POST', `/connections/${id}/databases`, { name }) }
  dropDatabase(id: string, name: string) { return this.request('DELETE', `/connections/${id}/databases/${encodeURIComponent(name)}`) }

  listTables(id: string) { return this.request<string[]>('GET', `/connections/${id}/tables`) }
  getSchema(id: string) { return this.request<any>('GET', `/connections/${id}/schema`) }
  createTable(id: string, name: string, columns: string) { return this.request<any>('POST', `/connections/${id}/tables`, { name, columns }) }
  dropTable(id: string, name: string) { return this.request('DELETE', `/connections/${id}/tables/${encodeURIComponent(name)}`) }

  query(id: string, sql: string) { return this.request<any>('POST', `/connections/${id}/query`, { query: sql }) }
  execute(id: string, sql: string) { return this.request<any>('POST', `/connections/${id}/execute`, { query: sql }) }

  browseTable(id: string, table: string, params?: { page?: number; per_page?: number; sort?: string; dir?: string; filter?: string }) {
    const qs = new URLSearchParams()
    if (params) { Object.entries(params).forEach(([k, v]) => { if (v) qs.set(k, String(v)) }) }
    const q = qs.toString() ? `?${qs.toString()}` : ''
    return this.request<any>('GET', `/connections/${id}/browse/${table}${q}`)
  }

  insertRow(id: string, table: string, data: Record<string, any>) { return this.request<any>('POST', `/connections/${id}/row/${table}`, data) }
  updateRow(id: string, table: string, pk: string, val: string, data: Record<string, any>) { return this.request<any>('PUT', `/connections/${id}/row/${table}/${pk}/${val}`, data) }
  deleteRow(id: string, table: string, pk: string, val: string) { return this.request('DELETE', `/connections/${id}/row/${table}/${pk}/${val}`) }

  getStats() { return this.request<any>('GET', '/admin/stats') }
  getActivity() { return this.request<any[]>('GET', '/admin/activity') }

  listUsers() { return this.request<any[]>('GET', '/admin/users') }
  createUser(data: any) { return this.request<any>('POST', '/admin/users', data) }
  updateUser(id: string, data: any) { return this.request<any>('PUT', `/admin/users/${id}`, data) }
  deleteUser(id: string) { return this.request('DELETE', `/admin/users/${id}`) }
  getUserPermissions(id: string) { return this.request<any>('GET', `/admin/users/${id}/permissions`) }
  setUserPermissions(id: string, data: any) { return this.request<any>('PUT', `/admin/users/${id}/permissions`, data) }

  listRoles() { return this.request<any[]>('GET', '/admin/roles') }
  createRole(data: any) { return this.request<any>('POST', '/admin/roles', data) }
  updateRole(id: string, data: any) { return this.request<any>('PUT', `/admin/roles/${id}`, data) }
  deleteRole(id: string) { return this.request('DELETE', `/admin/roles/${id}`) }
  setRolePermissions(id: string, data: any) { return this.request<any>('PUT', `/admin/roles/${id}/permissions`, data) }
  getPermissionGroups() { return this.request<any[]>('GET', '/admin/permission-groups') }

  listAPIKeys() { return this.request<any[]>('GET', '/apikeys') }
  createAPIKey(data: any) { return this.request<any>('POST', '/apikeys', data) }
  deleteAPIKey(prefix: string) { return this.request('DELETE', `/apikeys/${prefix}`) }

  getSuggestions(connId: string, input: string, table?: string) { return this.request<any[]>('POST', '/suggest', { connection_id: connId, input, current_table: table || '' }) }
  executeSafe(connId: string, sql: string, confirmHigh?: boolean) { return this.request<any>('POST', '/execute/safe', { connection_id: connId, sql, confirm_high: confirmHigh || false }) }
}

export const api = new APIClient()
