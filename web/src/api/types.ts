export interface APIResponse<T = any> {
  success: boolean
  data?: T
  error?: APIError
  meta?: { timestamp: string }
}

export interface APIError {
  code: string
  message: string
  details?: any
}

export interface LoginRequest {
  username: string
  password: string
}

export interface TokenResponse {
  token: string
  user_id: string
  username: string
  role: string
}

export interface Connection {
  id: string
  name: string
  type: string
  source: string
  config: any
  state: 'connected' | 'disconnected' | 'error' | 'connecting'
  latency_ms: number
  error?: string
  tags?: string[]
  created_at: string
  updated_at: string
}

export interface ConnectionSummary {
  id: string
  name: string
  type: string
  source: string
  state: string
  latency_ms: number
  tags?: string[]
}

export interface QueryResult {
  columns: string[]
  rows: any[][]
  rows_affected: number
  duration_ms: number
}

export interface BrowseResult {
  data: any[][]
  columns: string[]
  page: number
  per_page: number
  total: number
  total_pages: number
  duration_ms: number
}

export interface User {
  id: string
  username: string
  role: string
  extra_perm?: string[]
}

export interface Role {
  id: string
  name: string
  permissions: string[]
  db_access: string[]
}

export interface APIKey {
  prefix: string
  name: string
  permissions: string[]
  created_at: string
}

export interface APIKeyCreateResponse {
  raw_key: string
  prefix: string
  name: string
  permissions: string[]
  formatted: string
}

export interface Stats {
  connections: number
  queries: number
  users: number
  databases: number
}
