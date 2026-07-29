import { apiClient } from '../client'

export type PoolPeriodType = 'day' | 'week' | 'month' | 'custom'
export type PoolAccountStatus = 'active' | 'warning' | 'banned' | 'inactive'
export type PoolSettlementStatus = 'draft' | 'locked' | 'paid'
export type PoolLifecycleEventType = 'banned_confirmed' | 'recovered' | 'refund' | 'replaced' | 'retired'
export type PoolCostEntryType = 'purchase' | 'renewal' | 'topup' | 'price_version' | 'refund' | 'adjustment' | 'replacement_in' | 'replacement_out' | 'write_off'
export type PoolApprovalAction = 'UPDATE_ACCOUNT' | 'VIEW_CREDENTIAL' | 'DELETE_ACCOUNT'
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
  account_id?: number
  uploader_user_id?: number
  payer_user_id?: number
  purchase_source_id?: number
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
  uploaded_at?: string | null
  purchase_source_id?: number | null
  purchase_source_name: string
  purchase_url?: string | null
  order_no?: string | null
  purchase_cost: number
  expected_token_count?: number | null
  cost_sharing_enabled?: boolean
  entry_type?: PoolCostEntryType
  currency: string
  settled_cost?: number
  service_start: string
  service_end: string
  warranty_end?: string | null
  notes?: string | null
  paid_at?: string | null
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

export interface SharedPoolCostSummary {
  account_id: number
  account_name: string
  provider_identity?: string | null
  account_status?: string | null
  uploader_user_id?: number | null
  uploader_email?: string | null
  uploader_username?: string | null
  contributor_user_id?: number | null
  contributor_email?: string | null
  expected_token_count?: number | null
  priced_expected_token_count?: number
  remaining_expected_token_count?: number
  latest_payer_user_id?: number | null
  latest_payer_email?: string | null
  latest_purchase_source_id?: number | null
  latest_purchase_source?: string | null
  latest_order_no?: string | null
  purchased_at?: string | null
  latest_service_start?: string | null
  latest_service_end?: string | null
  latest_lifecycle_status: string
  latest_lifecycle_at?: string | null
  entry_count: number
  purchase_cost_minor?: number
  refund_minor?: number
  written_off_minor?: number
  net_cost_minor: number
  total_usage_tokens: number
  recognized_cost_minor: number
  remaining_cost_minor: number
  cost_progress?: string | null
}

export interface SharedPoolCostSummaryQuery {
  page?: number
  page_size?: number
  search?: string
  uploader_user_id?: number
  payer_user_id?: number
  purchase_source_id?: number
  lifecycle_status?: string
  has_cost?: boolean
}

export interface SharedPoolLedgerEntry {
  id: number
  account_id: number
  account_name: string
  payer_user_id: number
  payer_email: string
  purchase_source_id?: number | null
  purchase_source?: string | null
  entry_type: PoolCostEntryType
  currency: string
  original_amount: string
  cny_amount_minor: number
  fx_rate: string
  service_start: string
  service_end: string
  warranty_end?: string | null
  paid_at: string
  order_no?: string | null
  purchase_url?: string | null
  note?: string | null
  expected_token_count?: number | null
  created_at?: string
}

export interface SharedPoolLedgerEntryQuery {
  page?: number
  page_size?: number
  search?: string
  account_id?: number
  uploader_user_id?: number
  payer_user_id?: number
  purchase_source_id?: number
  entry_type?: PoolCostEntryType
  start_date?: string
  end_date?: string
}

export interface SharedPoolPaginated<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface SharedPoolPurchaseSource {
  id: number
  name: string
  active: boolean
}

export type SharedPoolBatchAmountMode = 'per_account' | 'order_total'

export interface BatchSharedPoolCostRequest {
  amount_mode: SharedPoolBatchAmountMode
  common: {
    payer_user_id: number
    purchase_source_id?: number
    entry_type: PoolCostEntryType
    original_amount: string
    currency: string
    fx_rate?: string
    service_start: string
    service_end: string
    warranty_end?: string
    paid_at?: string
    order_no?: string
    purchase_url?: string
    notes?: string
    expected_token_count?: number
  }
  accounts: Array<{
    account_id: number
    original_amount?: string
    expected_token_count?: number
  }>
}

export interface BatchSharedPoolCostResult {
  amount_mode: SharedPoolBatchAmountMode
  account_count: number
  total_original_amount: string
  total_cny_amount_minor: number
  entries: SharedPoolLedgerEntry[]
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
  expected_token_count: number
  cost_sharing_enabled: boolean
  entry_type: PoolCostEntryType
  currency: string
  service_start: string
  service_end: string
  warranty_end?: string
	paid_at?: string
  notes?: string
}

export interface CreateSharedPoolIntakeRequest {
  provider_identity: string
  contributor_user_id: number
  uploader_user_id: number
  purchase_source_name: string
  entry_type: PoolCostEntryType
  original_amount: string
  expected_token_count: number
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
  expected_token_count: number
  currency: string
  fx_rate: string
  service_start: string
  service_end: string
  warranty_end?: string
  order_no?: string
  purchase_url?: string
  notes?: string
  paid_at?: string
  supersedes_id?: number
  provider_identity?: string
  contributor_user_id?: number
  uploader_user_id?: number
  cost_sharing_enabled?: boolean
  approval_reason?: string
}

export interface SharedPoolCostWriteResult {
  approval_required?: boolean
  approval?: PoolApproval
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
  confirmation_status: 'pending' | 'confirmed'
  confirmed_by_user_id?: number
  confirmed_at?: string
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
  accounts: Array<{
    account_id: number
    account_name: string
    uploaded_at: string
    purchase_cost: number
    usage_value: number
    roi_rate: number
  }>
}

export interface SharedPoolUploaderSourceGroup {
  uploader_user_id?: number | null
  uploader_name: string
  account_count: number
  purchase_cost: number
  usage_value: number
  roi_rate: number
  ban_rate_30d: number
  sources: SharedPoolSourceStat[]
}

export interface SharedPoolSourceList { items: SharedPoolUploaderSourceGroup[] }

interface RawPoolAccount {
  id: number
  name: string
  platform: string
  provider_identity?: string | null
  contributor_user_id?: number | null
  contributor_email?: string | null
  created_by_user_id?: number | null
  created_by_email?: string | null
  created_by_username?: string | null
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
  expected_token_count?: number | null
  paid_at?: string
  created_at?: string
}

interface RawRecoveryAccount {
  account_id: number
  account_name: string
  provider_identity?: string | null
  uploader_user_id?: number | null
  uploader_username?: string | null
  uploaded_at: string
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
  purchased_at?: string | null
  banned_at?: string | null
  refunded: boolean
  survival_days: number
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
  confirmation_status: 'pending' | 'confirmed'
  confirmed_by_user_id?: number
  confirmed_at?: string
}

interface RawSettlement {
  id: number
  period_type: PoolPeriodType
  period_start: string
  period_end: string
  status: PoolSettlementStatus
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
      uploader_user_id: item.uploader_user_id,
      uploader_name: item.uploader_username || '',
      uploaded_at: item.uploaded_at,
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
      uploader_name: account?.created_by_username || account?.created_by_email || '',
      uploaded_at: undefined,
      purchase_source_id: cost.purchase_source_id,
      purchase_source_name: cost.purchase_source || '',
      purchase_url: cost.purchase_url,
      order_no: cost.order_no,
      notes: cost.note,
      paid_at: cost.paid_at,
      purchase_cost: minorToAmount(cost.cny_amount_minor),
      expected_token_count: cost.expected_token_count,
      cost_sharing_enabled: account?.cost_sharing_enabled ?? true,
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

export async function listCostSummaries(
  params: SharedPoolCostSummaryQuery = {}
): Promise<SharedPoolPaginated<SharedPoolCostSummary>> {
  const { data } = await apiClient.get<SharedPoolPaginated<SharedPoolCostSummary>>('/admin/pool/cost-summaries', { params })
  return data
}

export async function listLedgerEntries(
  params: SharedPoolLedgerEntryQuery = {}
): Promise<SharedPoolPaginated<SharedPoolLedgerEntry>> {
  const { data } = await apiClient.get<SharedPoolPaginated<SharedPoolLedgerEntry>>('/admin/pool/cost-entries', { params })
  return data
}

export async function listPurchaseSources(): Promise<SharedPoolPurchaseSource[]> {
  const { data } = await apiClient.get<SharedPoolPurchaseSource[]>('/admin/pool/sources')
  return data
}

export async function createPurchaseSource(name: string): Promise<SharedPoolPurchaseSource> {
  const { data } = await apiClient.post<SharedPoolPurchaseSource>('/admin/pool/sources', { name })
  return data
}

type PoolWriteOperation = { fingerprint: string; key: string }
const poolWriteOperations = new Map<string, PoolWriteOperation>()

async function postPoolWrite<T = void>(operationID: string, keyPrefix: string, url: string, payload: unknown): Promise<T> {
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
  const { data } = await apiClient.post<T>(url, payload, { headers: { 'Idempotency-Key': operation.key } })
  poolWriteOperations.delete(operationID)
  try { globalThis.sessionStorage?.removeItem(storageKey) } catch { /* memory fallback */ }
  return data
}

export async function createAccountIntake(
  accountId: number,
  payload: CreateSharedPoolIntakeRequest
): Promise<void> {
  await postPoolWrite(`pool-intake:${accountId}`, `pool-intake-${accountId}`, `/admin/pool/accounts/${accountId}/intake`, payload)
}

export async function createCost(payload: CreateSharedPoolCostEntryRequest): Promise<SharedPoolCostWriteResult> {
  return postPoolWrite<SharedPoolCostWriteResult>(`pool-cost:${payload.account_id}`, `pool-cost-${payload.account_id}`, '/admin/pool/costs', payload)
}

export async function createBatchCosts(payload: BatchSharedPoolCostRequest): Promise<BatchSharedPoolCostResult> {
  return postPoolWrite<BatchSharedPoolCostResult>('pool-cost:batch', 'pool-cost-batch', '/admin/pool/costs/batch', payload)
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
      payment_status: line.payment_status === 'paid' ? 'paid' : 'pending',
      confirmation_status: line.confirmation_status === 'confirmed' ? 'confirmed' : 'pending',
      confirmed_by_user_id: line.confirmed_by_user_id,
      confirmed_at: line.confirmed_at
    }))
  }
}

export async function previewSettlement(params: PoolPeriodParams): Promise<SharedPoolSettlementPreview> {
  const { data } = await apiClient.post<RawSettlement>('/admin/pool/settlements/draft', {
    period_type: params.period_type || 'custom',
    start_date: params.start,
    end_date: params.end,
    account_id: params.account_id,
    uploader_user_id: params.uploader_user_id,
    payer_user_id: params.payer_user_id,
    purchase_source_id: params.purchase_source_id
  })
  return mapSettlement(data)
}

export async function lockSettlement(payload: PoolPeriodParams & { settlement_id?: number }): Promise<SharedPoolSettlementPreview> {
  if (!payload.settlement_id) throw new Error('settlement_id is required')
  const { data } = await apiClient.post<RawSettlement>(`/admin/pool/settlements/${payload.settlement_id}/lock`)
  return mapSettlement(data)
}

export async function confirmSettlement(id: number, userId?: number): Promise<SharedPoolSettlementPreview> {
	const url = `/admin/pool/settlements/${id}/confirm`
	const { data } = userId
		? await apiClient.post<RawSettlement>(url, undefined, { params: { user_id: userId } })
		: await apiClient.post<RawSettlement>(url)
  return mapSettlement(data)
}

export async function markSettlementPaid(id: number): Promise<SharedPoolSettlementPreview> {
  const { data } = await apiClient.post<RawSettlement>(`/admin/pool/settlements/${id}/paid`)
  return mapSettlement(data)
}

export async function listSources(params?: PoolPeriodParams): Promise<SharedPoolSourceList> {
  if (!params) return { items: [] }
  const raw = await getRawOverview(params)
  type Accumulator = {
    accountCount: number
    costMinor: number
    valueMinor: number
    eligible7: number
    banned7: number
    eligible30: number
    banned30: number
    eligible90: number
    banned90: number
    refunded: number
    survivalDays: number
  }
  type SourceAccumulator = Accumulator & { name: string; accounts: SharedPoolSourceStat['accounts'] }
  type UploaderAccumulator = Accumulator & {
    uploader_user_id?: number | null
    uploader_name: string
    sources: Map<string, SourceAccumulator>
  }
  const empty = (): Accumulator => ({ accountCount: 0, costMinor: 0, valueMinor: 0, eligible7: 0, banned7: 0, eligible30: 0, banned30: 0, eligible90: 0, banned90: 0, refunded: 0, survivalDays: 0 })
  const uploaders = new Map<string, UploaderAccumulator>()
  const end = new Date(raw.end_at)
  for (const account of raw.accounts || []) {
    const uploaderName = account.uploader_username?.trim() || '-'
    const uploaderKey = account.uploader_user_id ? String(account.uploader_user_id) : `name:${uploaderName}`
    let uploader = uploaders.get(uploaderKey)
    if (!uploader) {
      uploader = { ...empty(), uploader_user_id: account.uploader_user_id, uploader_name: uploaderName, sources: new Map() }
      uploaders.set(uploaderKey, uploader)
    }
    const sourceName = account.purchase_source?.trim() || 'Unspecified'
    let source = uploader.sources.get(sourceName)
    if (!source) {
      source = { ...empty(), name: sourceName, accounts: [] }
      uploader.sources.set(sourceName, source)
    }
    const purchasedAt = account.purchased_at ? new Date(account.purchased_at) : null
    const bannedAt = account.banned_at ? new Date(account.banned_at) : null
    for (const item of [uploader, source]) {
      item.accountCount++
      item.costMinor += account.net_cost_minor || 0
      item.valueMinor += account.value_minor || 0
      for (const days of [7, 30, 90] as const) {
        const eligible = !!purchasedAt && end.getTime() - purchasedAt.getTime() >= days * 86400000
        const banned = eligible && !!bannedAt && bannedAt.getTime() <= purchasedAt!.getTime() + days * 86400000
        item[`eligible${days}`] += eligible ? 1 : 0
        item[`banned${days}`] += banned ? 1 : 0
      }
      item.refunded += account.refunded ? 1 : 0
      item.survivalDays += account.survival_days || 0
    }
    source.accounts.push({
      account_id: account.account_id,
      account_name: account.account_name,
      uploaded_at: account.uploaded_at,
      purchase_cost: minorToAmount(account.net_cost_minor),
      usage_value: minorToAmount(account.value_minor),
      roi_rate: ratioToPercent(account.recovery_rate)
    })
  }
  const rate = (part: number, total: number) => total > 0 ? part / total * 100 : 0
  const finishSource = (item: SourceAccumulator): SharedPoolSourceStat => ({
    name: item.name,
    account_count: item.accountCount,
    sample_size: item.accountCount,
    purchase_cost: minorToAmount(item.costMinor),
    usage_value: minorToAmount(item.valueMinor),
    roi_rate: rate(item.valueMinor, item.costMinor),
    ban_rate_7d: rate(item.banned7, item.eligible7),
    ban_rate_30d: rate(item.banned30, item.eligible30),
    ban_rate_90d: rate(item.banned90, item.eligible90),
    refund_rate: rate(item.refunded, item.accountCount),
    average_survival_days: item.accountCount ? item.survivalDays / item.accountCount : 0,
    accounts: item.accounts.sort((a, b) => b.account_id - a.account_id)
  })
  return { items: [...uploaders.values()].map((item) => ({
    uploader_user_id: item.uploader_user_id,
    uploader_name: item.uploader_name,
    account_count: item.accountCount,
    purchase_cost: minorToAmount(item.costMinor),
    usage_value: minorToAmount(item.valueMinor),
    roi_rate: rate(item.valueMinor, item.costMinor),
    ban_rate_30d: rate(item.banned30, item.eligible30),
    sources: [...item.sources.values()].map(finishSource).sort((a, b) => a.name.localeCompare(b.name))
  })).sort((a, b) => a.uploader_name.localeCompare(b.uploader_name)) }
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
  listCostSummaries,
  listLedgerEntries,
  listPurchaseSources,
  createPurchaseSource,
  createAccountIntake,
  createCost,
  createBatchCosts,
  recordLifecycleEvent,
  getLatestFXRate,
  saveFXRate,
  previewSettlement,
  lockSettlement,
  confirmSettlement,
  markSettlementPaid,
  listSources,
  createApproval,
  listApprovals,
  approveApproval,
  rejectApproval,
  revealApproval
}
