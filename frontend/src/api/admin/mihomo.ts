import { apiClient } from '../client'
import type { PoolApproval } from './sharedPool'

export interface MihomoNode {
  name: string
  alive: boolean
  delay?: number
}

export interface MihomoStatus {
  enabled: boolean
  version?: string
  configured: boolean
  provider_name?: string
  updated_at?: string
  node_count: number
  alive_count: number
  nodes: MihomoNode[]
  modes: Record<'automatic' | 'directional' | 'dynamic', { mode: string; selection: string }>
  proxy_ids: Partial<Record<'automatic' | 'directional' | 'dynamic', number>>
}

export interface MihomoApprovalResponse {
  approval_required: true
  approval: PoolApproval
}

export async function getStatus(): Promise<MihomoStatus> {
  const { data } = await apiClient.get<MihomoStatus>('/admin/mihomo/status')
  return data
}

export async function updateSubscription(subscriptionUrl: string, reason: string): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.post<MihomoApprovalResponse>('/admin/mihomo/subscription', {
    subscription_url: subscriptionUrl,
    reason
  })
  return data
}

export async function refresh(reason: string): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.post<MihomoApprovalResponse>('/admin/mihomo/refresh', { reason })
  return data
}

export async function updateMode(mode: string, selection: string, reason: string): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.post<MihomoApprovalResponse>('/admin/mihomo/modes', { mode, selection, reason })
  return data
}

export default { getStatus, updateSubscription, refresh, updateMode }
