import { apiClient } from '../client'

export type PoolPeriodType = 'day' | 'week' | 'month' | 'custom'
export type PoolAccountStatus = 'active' | 'warning' | 'banned' | 'inactive'
export type PoolSettlementStatus = 'draft' | 'locked' | 'paid'
export type PoolLifecycleEventType = 'banned_confirmed' | 'recovered' | 'refund' | 'replaced' | 'retired'
export type PoolCostEntryType = 'purchase' | 'renewal' | 'topup' | 'price_version' | 'adjustment'
export type PoolApprovalAction = 'UPDATE_ACCOUNT' | 'VIEW_CREDENTIAL'
export type PoolApprovalStatus = 'pending' | 'approved' | 'rejected' | 'expired' | 'consumed'

export interface PoolApproval {
  id: number
  action_type: PoolApprovalAction
  account_id: number
  account_name: string
  status: PoolApprovalStatus
  reason: string
  base_revision?: string | null
  requested_by_user_id: number
  requested_by_email: string
  decided_by_user_id?: number | null
  decided_by_email?: string | null
  decision_reason?: string | null
  requested_at: string
  expires_at?: string | null
  decided_at?: string | null
  reveal_expires_at?: string | null
  revealed_at?: string | null
  is_primary_bypass?: boolean
  changes?: Record<string, unknown> | null
}

export interface PoolApprovalList {
  items: PoolApproval[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface CreatePoolApprovalRequest {
  action_type: PoolApprovalAction
  account_id: number
  reason: string
  payload?: {
    account_update?: Record<string, unknown>
    pool_update?: Record<string, unknown>
  }
}

export interface PoolCredentialReveal {
  account_id: number
  credentials: Record<string, unknown>
  revealed_at: string
}

export interface PoolPeriodParams {
  start: string
  end: string
  period_type?: PoolPeriodType
}

export interface SharedPoolSummary {
  total_accounts: number
  active_accounts: number
  recovered_accounts: number
  banned_accounts: number
  total_purchase_cost: number
  total_usage_value: number
  pending_recovery: number
  banned_loss: number
  roi_rate: number
}

export interface SharedPoolAccountCost {
  id: number
  account_id: number
  account_name: string
  provider_identity: string
  contributor_user_id?: number | null
  contributor_name: string
  uploader_user_id?: number | null
  uploader_name: string
  purchase_source_id?: number | null
  purchase_source_name: string
  purchase_url?: string | null
  order_no?: string | null
  purchase_cost: number
  entry_type?: PoolCostEntryType
  currency: string
  settled_cost?: number
  service_start: string
  service_end: string
  warranty_end?: string | null
  notes?: string | null
  status: PoolAccountStatus
  usage_value: number
  roi_rate: number
  remaining_cost: number
  banned_loss: number
  recovered_at?: string | null
  latest_recovery_at?: string | null
  currently_recovered?: boolean
  net_profit?: number
  current_net_loss?: number
  estimated_recovery_days?: number | null
  observation_days?: number
}

export interface SharedPoolOverview {
  summary: SharedPoolSummary
  accounts: SharedPoolAccountCost[]
  period_start: string
  period_end: string
  currency: string
}

export interface SharedPoolCostList {
  items: SharedPoolAccountCost[]
  total: number
}

export interface CreateSharedPoolCostRequest {
  account_id: number
  provider_identity: string
  contributor_user_id: number
  uploader_user_id: number
  purchase_source_name: string
  purchase_url?: string
  order_no?: string
  purchase_cost: number
  entry_type: PoolCostEntryType
  currency: string
  service_start: string
  service_end: string
  warranty_end?: string
  notes?: string
}

export interface CreateSharedPoolIntakeRequest {
  provider_identity: string
  contributor_user_id: number
  uploader_user_id: number
  purchase_source_name: string
  entry_type: PoolCostEntryType
  original_amount: string
  currency: string
  fx_rate: string
  cny_amount_minor: number
  service_start: string
  service_end: string
  warranty_end?: string
  paid_at?: string
  order_no?: string
  purchase_url?: string
  notes?: string
}

export interface CreateSharedPoolCostEntryRequest {
  account_id: number
  payer_user_id: number
  purchase_source_id?: number
  entry_type: PoolCostEntryType
  original_amount: string
  currency: string
  fx_rate: string
  service_start: string
  service_end: string
  warranty_end?: string
  order_no?: string
  purchase_url?: string
  notes?: string
}

export interface RecordSharedPoolLifecycleRequest {
  account_id: number
  event_type: PoolLifecycleEventType
  occurred_at: string
  reason?: string
  payer_user_id?: number
  refund_amount?: number
  replacement_account_id?: number
  transferred_cost?: number
}

export interface SharedPoolSettlementLine {
  user_id: number
  user_name: string
  usage_weight: number
  usage_share: number
  allocated_cost: number
  contribution_credit: number
  adjustment: number
  net_amount: number
  payment_status?: 'pending' | 'paid'
}

export interface SharedPoolSettlementPreview {
  id?: number
  status: PoolSettlementStatus
  period_type: PoolPeriodType
  period_start: string
  period_end: string
  currency: string
  total_cost: number
  total_usage_weight: number
  carry_forward: number
  unpriced_usage_count: number
  pricing_coverage: number
  lines: SharedPoolSettlementLine[]
}

export interface SharedPoolSourceStat {
  id?: number
  name: string
  account_count: number
  sample_size: number
  purchase_cost: number
  usage_value: number
  roi_rate: number
  ban_rate_7d: number
  ban_rate_30d: number
  ban_rate_90d: number
  refund_rate: number
  average_survival_days: number
  average_recovery_days?: number | null
}

export interface SharedPoolSourceList { items: SharedPoolSourceStat[] }

interface RawPoolAccount {
  id: number
  name: string
  platform: string
  provider_identity?: string | null
  contributor_user_id?: number | null
  contributor_email?: string | null
  created_by_user_id?: number | null
  created_by_email?: string | null
  cost_sharing_enabled: boolean
  latest_lifecycle_status: string
  net_cost_minor: number
}

interface RawPoolCost {
  id: number
  account_id: number
  account_name: string
  payer_user_id: number
  payer_email: string
  purchase_source_id?: number | null
  purchase_source?: string | null
  entry_type: string
  currency: string
  original_amount: string
  cny_amount_minor: number
  fx_rate: string
  service_start: string
  service_end: string
  warranty_end?: string | null
  order_no?: string | null
  purchase_url?: string | null
  note?: string | null
}

interface RawRecoveryAccount {
  account_id: number
  account_name: string
  provider_identity?: string | null
  purchase_source?: string | null
  lifecycle_status: string
  net_cost_minor: number
  value_minor: number
  unrecovered_minor: number
  banned_loss_minor: number
  current_net_loss_minor: number
  net_profit_minor: number
  recovery_rate: string
  estimated_recovery_days?: number | null
  first_recovery_at?: string | null
  latest_recovery_at?: string | null
  currently_recovered: boolean
  observation_days: number
}

interface RawSourceStat {
  name: string
  account_count: number
  sample_size: number
  purchase_cost_minor: number
  value_minor: number
  recovery_rate: string
  ban_rate_7d: string
  ban_rate_30d: string
  ban_rate_90d: string
  refund_rate: string
  average_survival_days: string
}

interface RawOverview {
  start_at: string
  end_at: string
  total_cost_minor: number
  total_value_minor: number
  unrecovered_minor: number
  banned_loss_minor: number
  recovery_rate: string
  recovered_accounts: number
  total_accounts: number
  accounts: RawRecoveryAccount[]
  source_stats?: RawSourceStat[]
}

interface RawSettlementLine {
  user_id: number
  user_email: string
  username: string
  usage_weight: string
  usage_share: string
  allocated_cost_minor: number
  contribution_credit_minor: number
  adjustment_minor: number
  net_amount_minor: number
  payment_status: 'unpaid' | 'paid'
}

interface RawSettlement {
  id: number
  period_type: PoolPeriodType
  period_start: string
  period_end: string
  status: 'draft' | 'locked'
  total_cost_minor: number
  carry_out_minor: number
  total_usage_weight: string
  pricing_coverage: string
  unpriced_usage_count: number
  fx_rate: string
  lines: RawSettlementLine[]
}

interface RawFXRate {
  rate: string
}

const minorToAmount = (value: number): number => value / 100
const ratioToPercent = (value: string | number): number => Number(value || 0) * 100

const mapLifecycleStatus = (status: string): PoolAccountStatus => {
  if (status === 'banned_confirmed') return 'banned'
  if (status === 'retired' || status === 'replaced') return 'inactive'
  if (status === 'refund') return 'warning'
  return 'active'
}

const addLocalDays = (value: string, days: number): string => {
  const [year, month, day] = value.split('-').map(Number)
  const date = new Date(year, month - 1, day)
  date.setDate(date.getDate() + days)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const overviewParams = (params: PoolPeriodParams) => ({
  start_date: params.start,
  end_date: params.period_type === 'custom' ? addLocalDays(params.end, 1) : params.end
})

async function getRawOverview(params: PoolPeriodParams): Promise<RawOverview> {
  const { data } = await apiClient.get<RawOverview>('/admin/pool/overview', { params: overviewParams(params) })
  return data
}

export async function getOverview(params: PoolPeriodParams): Promise<SharedPoolOverview> {
  const raw = await getRawOverview(params)
  const accounts = raw.accounts || []
  return {
    summary: {
      total_accounts: raw.total_accounts,
      active_accounts: accounts.filter((item) => mapLifecycleStatus(item.lifecycle_status) === 'active').length,
      recovered_accounts: raw.recovered_accounts,
      banned_accounts: accounts.filter((item) => item.lifecycle_status === 'banned_confirmed').length,
      total_purchase_cost: minorToAmount(raw.total_cost_minor),
      total_usage_value: minorToAmount(raw.total_value_minor),
      pending_recovery: minorToAmount(raw.unrecovered_minor),
      banned_loss: minorToAmount(raw.banned_loss_minor),
      roi_rate: ratioToPercent(raw.recovery_rate)
    },
    accounts: accounts.map((item) => ({
      id: item.account_id,
      account_id: item.account_id,
      account_name: item.account_name,
      provider_identity: item.provider_identity || '',
      contributor_name: '',
      uploader_name: '',
      purchase_source_name: item.purchase_source || '',
      purchase_cost: minorToAmount(item.net_cost_minor),
      currency: 'CNY',
      service_start: '',
      service_end: '',
      status: mapLifecycleStatus(item.lifecycle_status),
      usage_value: minorToAmount(item.value_minor),
      roi_rate: ratioToPercent(item.recovery_rate),
      remaining_cost: minorToAmount(item.unrecovered_minor),
      banned_loss: minorToAmount(item.banned_loss_minor),
      recovered_at: item.first_recovery_at,
      latest_recovery_at: item.latest_recovery_at,
      currently_recovered: item.currently_recovered,
      net_profit: minorToAmount(item.net_profit_minor),
      current_net_loss: minorToAmount(item.current_net_loss_minor),
      estimated_recovery_days: item.estimated_recovery_days,
      observation_days: item.observation_days
    })),
    period_start: raw.start_at,
    period_end: raw.end_at,
    currency: 'CNY'
  }
}

export async function listAccountCosts(params?: PoolPeriodParams): Promise<SharedPoolCostList> {
  const [{ data: costs }, { data: accounts }, overview] = await Promise.all([
    apiClient.get<RawPoolCost[]>('/admin/pool/costs'),
    apiClient.get<RawPoolAccount[]>('/admin/pool/accounts'),
    params ? getRawOverview(params) : Promise.resolve<RawOverview | null>(null)
  ])
  const accountByID = new Map(accounts.map((item) => [item.id, item]))
  const recoveryByID = new Map((overview?.accounts || []).map((item) => [item.account_id, item]))
  const items = costs.map((cost) => {
    const account = accountByID.get(cost.account_id)
    const recovery = recoveryByID.get(cost.account_id)
    return {
      id: cost.id,
      account_id: cost.account_id,
      account_name: cost.account_name,
      provider_identity: account?.provider_identity || '',
      contributor_user_id: account?.contributor_user_id,
      contributor_name: account?.contributor_email || '',
      uploader_user_id: account?.created_by_user_id,
      uploader_name: account?.created_by_email || '',
      purchase_source_id: cost.purchase_source_id,
      purchase_source_name: cost.purchase_source || '',
      purchase_url: cost.purchase_url,
      order_no: cost.order_no,
      notes: cost.note,
      purchase_cost: minorToAmount(cost.cny_amount_minor),
      entry_type: cost.entry_type as PoolCostEntryType,
      currency: 'CNY',
      service_start: cost.service_start.slice(0, 10),
      service_end: cost.service_end.slice(0, 10),
      warranty_end: cost.warranty_end?.slice(0, 10),
      status: mapLifecycleStatus(account?.latest_lifecycle_status || 'active'),
      usage_value: minorToAmount(recovery?.value_minor || 0),
      roi_rate: ratioToPercent(recovery?.recovery_rate || 0),
      remaining_cost: minorToAmount(recovery?.unrecovered_minor || 0),
      banned_loss: minorToAmount(recovery?.banned_loss_minor || 0)
    }
  })
  return { items, total: items.length }
}

type PoolWriteOperation = { fingerprint: string; key: string }
const poolWriteOperations = new Map<string, PoolWriteOperation>()

async function postPoolWrite(operationID: string, keyPrefix: string, url: string, payload: unknown): Promise<void> {
  const storageKey = `sub2api:admin:${operationID}`
  const fingerprint = JSON.stringify(payload)
  let operation = poolWriteOperations.get(operationID)
  try {
    if (!operation) {
      const stored = globalThis.sessionStorage?.getItem(storageKey)
      if (stored) operation = JSON.parse(stored) as PoolWriteOperation
    }
  } catch {
    // In-memory retry protection still works when browser storage is unavailable.
  }
  if (!operation || operation.fingerprint !== fingerprint) {
    const requestID = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
    operation = { fingerprint, key: `${keyPrefix}-${requestID}` }
  }
  poolWriteOperations.set(operationID, operation)
  try { globalThis.sessionStorage?.setItem(storageKey, JSON.stringify(operation)) } catch { /* memory fallback */ }
  await apiClient.post(url, payload, { headers: { 'Idempotency-Key': operation.key } })
  poolWriteOperations.delete(operationID)
  try { globalThis.sessionStorage?.removeItem(storageKey) } catch { /* memory fallback */ }
}

export async function createAccountIntake(
  accountId: number,
  payload: CreateSharedPoolIntakeRequest
): Promise<void> {
  await postPoolWrite(`pool-intake:${accountId}`, `pool-intake-${accountId}`, `/admin/pool/accounts/${accountId}/intake`, payload)
}

export async function createCost(payload: CreateSharedPoolCostEntryRequest): Promise<void> {
  await postPoolWrite(`pool-cost:${payload.account_id}`, `pool-cost-${payload.account_id}`, '/admin/pool/costs', payload)
}

export async function recordLifecycleEvent(payload: RecordSharedPoolLifecycleRequest): Promise<void> {
  await apiClient.post('/admin/pool/lifecycle', {
    account_id: payload.account_id,
    event_type: payload.event_type,
    event_at: payload.occurred_at,
    reason: payload.reason || undefined,
    payer_user_id: payload.payer_user_id || undefined,
    refund_amount_minor: Math.round((payload.refund_amount || 0) * 100),
    replacement_account_id: payload.replacement_account_id || undefined,
    transferred_cost_minor: Math.round((payload.transferred_cost || 0) * 100)
  })
}

export async function getLatestFXRate(): Promise<number> {
  const { data } = await apiClient.get<RawFXRate[]>('/admin/pool/fx-rates')
  return Number(data[0]?.rate || 1)
}

export async function saveFXRate(rate: number, effectiveDate: string): Promise<number> {
  const { data } = await apiClient.post<RawFXRate>('/admin/pool/fx-rates', {
    base_currency: 'USD',
    quote_currency: 'CNY',
    rate: rate.toString(),
    effective_from: new Date(`${effectiveDate}T00:00:00+08:00`).toISOString(),
    source: 'manual'
  })
  return Number(data.rate)
}

const mapSettlement = (raw: RawSettlement): SharedPoolSettlementPreview => {
  const fxRate = Number(raw.fx_rate || 1)
  return {
    id: raw.id,
    status: raw.status,
    period_type: raw.period_type,
    period_start: raw.period_start,
    period_end: raw.period_end,
    currency: 'CNY',
    total_cost: minorToAmount(raw.total_cost_minor),
    total_usage_weight: Number(raw.total_usage_weight || 0) * fxRate,
    carry_forward: minorToAmount(raw.carry_out_minor),
    unpriced_usage_count: raw.unpriced_usage_count,
    pricing_coverage: ratioToPercent(raw.pricing_coverage),
    lines: (raw.lines || []).map((line) => ({
      user_id: line.user_id,
      user_name: line.username || line.user_email,
      usage_weight: Number(line.usage_weight || 0) * fxRate,
      usage_share: ratioToPercent(line.usage_share),
      allocated_cost: minorToAmount(line.allocated_cost_minor),
      contribution_credit: minorToAmount(line.contribution_credit_minor),
      adjustment: minorToAmount(line.adjustment_minor),
      net_amount: minorToAmount(line.net_amount_minor),
      payment_status: line.payment_status === 'paid' ? 'paid' : 'pending'
    }))
  }
}

export async function previewSettlement(params: PoolPeriodParams): Promise<SharedPoolSettlementPreview> {
  const { data } = await apiClient.post<RawSettlement>('/admin/pool/settlements/draft', {
    period_type: params.period_type || 'custom',
    start_date: params.start,
    end_date: params.end
  })
  return mapSettlement(data)
}

export async function lockSettlement(payload: PoolPeriodParams & { settlement_id?: number }): Promise<SharedPoolSettlementPreview> {
  if (!payload.settlement_id) throw new Error('settlement_id is required')
  const { data } = await apiClient.post<RawSettlement>(`/admin/pool/settlements/${payload.settlement_id}/lock`)
  return mapSettlement(data)
}

export async function listSources(params?: PoolPeriodParams): Promise<SharedPoolSourceList> {
  if (!params) return { items: [] }
  const raw = await getRawOverview(params)
  return {
    items: (raw.source_stats || []).map((item) => ({
      name: item.name,
      account_count: item.account_count,
      sample_size: item.sample_size,
      purchase_cost: minorToAmount(item.purchase_cost_minor),
      usage_value: minorToAmount(item.value_minor),
      roi_rate: ratioToPercent(item.recovery_rate),
      ban_rate_7d: ratioToPercent(item.ban_rate_7d),
      ban_rate_30d: ratioToPercent(item.ban_rate_30d),
      ban_rate_90d: ratioToPercent(item.ban_rate_90d),
      refund_rate: ratioToPercent(item.refund_rate),
      average_survival_days: Number(item.average_survival_days || 0)
    }))
  }
}

export async function createApproval(payload: CreatePoolApprovalRequest): Promise<PoolApproval> {
  const { data } = await apiClient.post<PoolApproval>('/admin/pool/approvals', payload)
  return data
}

export async function listApprovals(params: {
  status?: PoolApprovalStatus
  action_type?: PoolApprovalAction
  account_id?: number
  requested_by_user_id?: number
  page?: number
  page_size?: number
} = {}): Promise<PoolApprovalList> {
  const { data } = await apiClient.get<PoolApprovalList>('/admin/pool/approvals', { params })
  return data
}

export async function approveApproval(id: number, reason?: string): Promise<PoolApproval> {
  const { data } = await apiClient.post<PoolApproval>(`/admin/pool/approvals/${id}/approve`, {
    reason: reason?.trim() || undefined
  })
  return data
}

export async function rejectApproval(id: number, reason: string): Promise<PoolApproval> {
  const { data } = await apiClient.post<PoolApproval>(`/admin/pool/approvals/${id}/reject`, {
    reason: reason.trim()
  })
  return data
}

export async function revealApproval(id: number): Promise<PoolCredentialReveal> {
  const { data } = await apiClient.post<PoolCredentialReveal>(`/admin/pool/approvals/${id}/reveal`)
  return data
}

export default {
  getOverview,
  listAccountCosts,
  createAccountIntake,
  createCost,
  recordLifecycleEvent,
  getLatestFXRate,
  saveFXRate,
  previewSettlement,
  lockSettlement,
  listSources,
  createApproval,
  listApprovals,
  approveApproval,
  rejectApproval,
  revealApproval
}
