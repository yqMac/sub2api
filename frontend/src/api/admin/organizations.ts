/**
 * [bmai-fork] Admin Organizations API endpoints
 */

import { apiClient } from '../client'

export interface Organization {
  ID: number
  TenantKey: string
  Name: string
  Type: string
  CreatedAt: string
  UpdatedAt: string
}

export interface Department {
  ID: number
  OrganizationID: number
  ParentID: number | null
  ExternalID: string
  Name: string
  FullPath: string
  Level: number
  CreatedAt: string
  UpdatedAt: string
}

export interface UserDepartment {
  UserID: number
  DepartmentID: number
  IsPrimary: boolean
  Role: string
  EmployeeID: string
  CreatedAt: string
}

export const organizationAPI = {
  async list(): Promise<Organization[]> {
    const { data } = await apiClient.get<Organization[]>('/admin/organizations')
    return data || []
  },
  async create(input: { tenant_key: string; name: string; type?: string }): Promise<Organization> {
    const { data } = await apiClient.post<Organization>('/admin/organizations', input)
    return data
  },
  async delete(id: number): Promise<void> {
    await apiClient.delete(`/admin/organizations/${id}`)
  },
  async departmentTree(orgId: number): Promise<Department[]> {
    const { data } = await apiClient.get<Department[]>(
      `/admin/organizations/${orgId}/departments`
    )
    return data || []
  },
  async createDepartment(
    orgId: number,
    input: { external_id: string; name: string; parent_id?: number }
  ): Promise<Department> {
    const { data } = await apiClient.post<Department>(
      `/admin/organizations/${orgId}/departments`,
      input
    )
    return data
  },
  async updateDepartment(
    deptId: number,
    input: { name?: string; parent_id?: number | null }
  ): Promise<Department> {
    const { data } = await apiClient.put<Department>(
      `/admin/organizations/departments/${deptId}`,
      input
    )
    return data
  },
  async deleteDepartment(deptId: number): Promise<void> {
    await apiClient.delete(`/admin/organizations/departments/${deptId}`)
  },
  async departmentUsers(
    deptId: number,
    page = 1,
    pageSize = 50
  ): Promise<{ items: UserDepartment[]; total: number }> {
    const { data } = await apiClient.get<{ items: UserDepartment[]; total: number }>(
      `/admin/organizations/departments/${deptId}/users`,
      { params: { page, page_size: pageSize } }
    )
    return data
  },
  async assignUser(
    deptId: number,
    input: { user_id: number; is_primary?: boolean; role?: string; employee_id?: string }
  ): Promise<void> {
    await apiClient.post(`/admin/organizations/departments/${deptId}/users`, input)
  },
  async removeUser(deptId: number, userId: number): Promise<void> {
    await apiClient.delete(`/admin/organizations/departments/${deptId}/users/${userId}`)
  },
  // [bmai-fork] Feishu sync endpoints
  async feishuConfig(orgId: number, config: { app_id: string; app_secret: string }): Promise<void> {
    await apiClient.post(`/admin/organizations/${orgId}/feishu/config`, config)
  },
  async feishuTest(orgId: number): Promise<{ success: boolean; department_count?: number; error?: string }> {
    const { data } = await apiClient.post<{ success: boolean; department_count?: number; error?: string }>(
      `/admin/organizations/${orgId}/feishu/test`
    )
    return data
  },
  async feishuSync(orgId: number): Promise<{ departments_synced: number; users_synced: number; users_matched: number; users_unmatched: number; errors?: string[] }> {
    const { data } = await apiClient.post<any>(`/admin/organizations/${orgId}/feishu/sync`)
    return data
  },
  async feishuStatus(orgId: number): Promise<any> {
    const { data } = await apiClient.get<any>(`/admin/organizations/${orgId}/feishu/status`)
    return data
  }
}
