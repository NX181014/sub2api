import { apiClient } from '../client'
import type { PoolApproval } from './sharedPool'

export type MihomoRouteKind = 'dedicated' | 'automatic' | 'latency' | 'fallback' | 'dynamic' | 'directional'
export type MihomoHealth = 'healthy' | 'degraded' | 'failed' | 'unknown'

export interface MihomoNode {
  id?: number
  key?: string
  name: string
  display_name?: string
  original_name?: string
  node_key?: string
  subscription_id?: number
  subscription_name?: string
  alive: boolean
  delay?: number
  delay_ms?: number
  region?: string
  tags?: string[]
  enabled?: boolean
  excluded?: boolean
  exit_ip?: string
  last_seen_at?: string
  upstream_removed_at?: string
}

export interface MihomoSubscription {
  id: number
  name: string
  enabled: boolean
  status: MihomoHealth | 'active' | 'disabled' | 'refreshing'
  masked_url?: string
  source_host?: string
  masked_host?: string
  refresh_interval_minutes: number
  refresh_interval_seconds?: number
  node_count: number
  alive_count: number
  used_bytes?: number
  total_bytes?: number
  quota_used_bytes?: number
  quota_total_bytes?: number
  expires_at?: string
  last_refreshed_at?: string
  last_error?: string
}

export interface MihomoRoute {
  id: number
  name: string
  kind: MihomoRouteKind
  subscription_ids: number[]
  subscription_names?: string[]
  node_ids: Array<number | string>
  listener_port: number
  proxy_id: number
  enabled: boolean
  current_node?: string
  current_node_id?: number
  exit_ip?: string
  exit_healthy?: boolean
  health: MihomoHealth
  latency_ms?: number
  exit_delay_ms?: number
  account_count: number
  last_checked_at?: string
}

export interface MihomoRuntimeStatus {
  enabled: boolean
  version?: string
  configured: boolean
  controller_connected?: boolean
  config_valid?: boolean
  generated_at?: string
  last_reload_at?: string
  last_reload_error?: string
}

export interface MihomoWorkbench {
  status: MihomoRuntimeStatus
  subscriptions: MihomoSubscription[]
  nodes: MihomoNode[]
  routes: MihomoRoute[]
}

export interface MihomoLegacyImportPreview {
  available: boolean
  already_imported: boolean
  provider_name?: string
  subscription_host?: string
  node_count: number
  route_count: number
  affected_account_count: number
  routes: Array<{
    name: string
    kind: MihomoRouteKind
    listener_port: number
    proxy_id: number
    node_count: number
    account_count: number
  }>
}

export interface MihomoStatus {
  enabled: boolean
  version?: string
  configured: boolean
  provider_name?: string
  subscription_configured: boolean
  subscription_host?: string
  updated_at?: string
  node_count: number
  alive_count: number
  nodes: MihomoNode[]
  modes: Record<'automatic' | 'directional' | 'dynamic', { mode: string; selection: string }>
  proxy_ids: Partial<Record<'automatic' | 'directional' | 'dynamic', number>>
}

export interface MihomoApprovalResponse {
  approval_required: boolean
  approval?: PoolApproval
  message?: string
}

export interface MihomoSubscriptionInput {
  name: string
  subscription_url?: string
  enabled: boolean
  refresh_interval_minutes: number
  reason: string
}

export interface MihomoRouteInput {
  name: string
  kind: MihomoRouteKind
  subscription_ids: number[]
  node_ids: Array<number | string>
  enabled: boolean
  reason: string
}

export interface MihomoNodeActionInput {
  action: 'test' | 'exclude' | 'restore' | 'enable' | 'disable' | 'create_dedicated_routes'
  node_ids: Array<number | string>
  reason?: string
}

export async function getStatus(): Promise<MihomoStatus> {
  const { data } = await apiClient.get<MihomoStatus>('/admin/mihomo/status')
  return data
}

export async function getWorkbench(): Promise<MihomoWorkbench> {
  const { data } = await apiClient.get<MihomoWorkbench>('/admin/mihomo/workbench')
  const subscriptions = (data.subscriptions || []).map(item => ({
    ...item,
    enabled: item.enabled ?? item.status !== 'disabled',
    source_host: item.source_host || item.masked_host,
    refresh_interval_minutes: item.refresh_interval_minutes || Math.max(5, Math.round((item.refresh_interval_seconds || 3600) / 60)),
    used_bytes: item.used_bytes ?? item.quota_used_bytes,
    total_bytes: item.total_bytes ?? item.quota_total_bytes,
    node_count: item.node_count || 0,
    alive_count: item.alive_count || 0
  }))
  const subscriptionNames = new Map(subscriptions.map(item => [item.id, item.name]))
  const nodes = (data.nodes || []).map(item => ({
    ...item,
    key: item.key || item.node_key,
    name: item.name || item.display_name || item.original_name || item.node_key || '未命名节点',
    delay: item.delay ?? item.delay_ms,
    subscription_name: item.subscription_name || subscriptionNames.get(item.subscription_id || 0)
  }))
  const nodeNames = new Map(nodes.map(item => [item.id, item.display_name || item.name]))
  const routes = (data.routes || []).map(item => ({
    ...item,
    subscription_ids: item.subscription_ids || [],
    subscription_names: item.subscription_names || (item.subscription_ids || []).map(id => subscriptionNames.get(id) || `订阅 #${id}`),
    node_ids: item.node_ids || [],
    proxy_id: item.proxy_id || 0,
    enabled: item.enabled ?? !['disabled', 'inactive'].includes((item as MihomoRoute & { status?: string }).status || ''),
    current_node: item.current_node || nodeNames.get(item.current_node_id),
    health: item.health || (item.exit_healthy === true ? 'healthy' : item.exit_healthy === false ? 'failed' : 'unknown'),
    latency_ms: item.latency_ms ?? item.exit_delay_ms,
    account_count: item.account_count || 0
  }))
  return { ...data, subscriptions, nodes, routes }
}

export async function getImportPreview(): Promise<MihomoLegacyImportPreview> {
  const { data } = await apiClient.get<MihomoLegacyImportPreview>('/admin/mihomo/import-preview')
  return data
}

export async function importLegacy(reason: string): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.post<MihomoApprovalResponse>('/admin/mihomo/import', { reason })
  return data
}

export async function createSubscription(input: MihomoSubscriptionInput): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.post<MihomoApprovalResponse>('/admin/mihomo/subscriptions', input)
  return data
}

export async function updateWorkbenchSubscription(id: number, input: MihomoSubscriptionInput): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.put<MihomoApprovalResponse>(`/admin/mihomo/subscriptions/${id}`, input)
  return data
}

export async function deleteSubscription(id: number, reason: string): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.delete<MihomoApprovalResponse>(`/admin/mihomo/subscriptions/${id}`, { data: { reason } })
  return data
}

export async function refreshSubscription(id: number, reason: string): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.post<MihomoApprovalResponse>(`/admin/mihomo/subscriptions/${id}/refresh`, { reason })
  return data
}

export async function createRoute(input: MihomoRouteInput): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.post<MihomoApprovalResponse>('/admin/mihomo/routes', input)
  return data
}

export async function updateRoute(id: number, input: MihomoRouteInput): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.put<MihomoApprovalResponse>(`/admin/mihomo/routes/${id}`, input)
  return data
}

export async function deleteRoute(id: number, reason: string): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.delete<MihomoApprovalResponse>(`/admin/mihomo/routes/${id}`, { data: { reason } })
  return data
}

export async function testRoute(id: number): Promise<MihomoRoute> {
  const { data } = await apiClient.post<MihomoRoute>(`/admin/mihomo/routes/${id}/test`)
  return data
}

export async function runNodeAction(input: MihomoNodeActionInput): Promise<MihomoApprovalResponse> {
  const { data } = await apiClient.post<MihomoApprovalResponse>('/admin/mihomo/nodes/actions', input)
  return data
}

// Compatibility endpoints retained while existing installations import their first subscription.
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

export default {
  getStatus,
  getWorkbench,
  getImportPreview,
  importLegacy,
  createSubscription,
  updateWorkbenchSubscription,
  deleteSubscription,
  refreshSubscription,
  createRoute,
  updateRoute,
  deleteRoute,
  testRoute,
  runNodeAction,
  updateSubscription,
  refresh,
  updateMode
}
