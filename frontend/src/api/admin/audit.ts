/**
 * [bmai-fork] Admin Audit API endpoints
 */

import { apiClient } from '../client'

export type AuditContentType =
  | 'conversation'
  | 'code'
  | 'image'
  | 'tool_use'
  | 'reasoning'
  | 'plan'
  | 'script'

export interface AuditLogListItem {
  id: number
  request_id: string
  user_id: number
  organization_id: number | null
  department_id: number | null
  department_path: string | null
  content_type: AuditContentType | null
  model: string
  platform: string
  endpoint: string
  request_summary: string
  response_summary: string
  request_bytes: number
  response_bytes: number
  input_tokens: number
  output_tokens: number
  duration_ms: number
  stream: boolean
  status_code: number
  risk_level: 'low' | 'medium' | 'high' | null
  intercepted: boolean
  created_at: string
}

export interface AuditLogDetail {
  id: number
  request_id: string
  user_id: number
  api_key_id: number
  account_id: number
  organization_id: number | null
  department_id: number | null
  department_path: string
  content_type: string
  request_preview: string
  request_bytes: number
  request_truncated: boolean
  response_preview: string
  response_bytes: number
  response_truncated: boolean
  upstream_request_preview: string
  upstream_request_bytes: number
  upstream_request_truncated: boolean
  upstream_response_preview: string
  upstream_response_bytes: number
  upstream_response_truncated: boolean
  model: string
  platform: string
  endpoint: string
  stream: boolean
  duration_ms: number
  input_tokens: number
  output_tokens: number
  status_code: number
  intercepted: boolean
  intercept_reason: string
  risk_level: string
  risk_tags: string[] | null
  created_at: string
}

export interface AuditLogListResponse {
  items: AuditLogListItem[]
  total: number
  page: number
  page_size: number
}

export interface AuditLogFilter {
  start_time?: string
  end_time?: string
  user_id?: string
  organization_id?: number
  department_id?: number
  department_path?: string
  content_type?: string
  model?: string
  platform?: string
  keyword?: string
  risk_level?: string
  intercepted?: boolean
  page?: number
  page_size?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface AuditSettings {
  enabled: boolean
  max_request_bytes: number
  max_response_bytes: number
  capture_upstream: boolean
  retention_days: number
  classify_response: boolean
}

export interface AuditPartitionInfo {
  Name: string
  Rows: number
  Bytes: number
}

export interface AuditStorageInfo {
  TotalRows: number
  TotalBytes: number
  EarliestRecord: string | null
  Partitions: AuditPartitionInfo[]
}

export const auditAPI = {
  async list(filter: AuditLogFilter, signal?: AbortSignal): Promise<AuditLogListResponse> {
    const { data } = await apiClient.get<{ data: AuditLogListResponse }>('/admin/audit/logs', {
      params: filter,
      signal
    })
    return data.data
  },
  async get(id: number): Promise<AuditLogDetail> {
    const { data } = await apiClient.get<{ data: AuditLogDetail }>(`/admin/audit/logs/${id}`)
    return data.data
  },
  async getSettings(): Promise<AuditSettings> {
    const { data } = await apiClient.get<{ data: AuditSettings }>('/admin/audit/settings')
    return data.data
  },
  async updateSettings(settings: Partial<AuditSettings>): Promise<AuditSettings> {
    const { data } = await apiClient.put<{ data: AuditSettings }>('/admin/audit/settings', settings)
    return data.data
  },
  async getStorage(): Promise<AuditStorageInfo> {
    const { data } = await apiClient.get<{ data: AuditStorageInfo }>('/admin/audit/storage')
    return data.data
  },
  async cleanup(beforeDays: number): Promise<{ deleted: number }> {
    const { data } = await apiClient.post<{ data: { deleted: number } }>('/admin/audit/cleanup', {
      before_days: beforeDays
    })
    return data.data
  }
}
