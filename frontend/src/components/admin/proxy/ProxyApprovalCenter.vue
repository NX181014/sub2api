<template>
  <BaseDialog :show="show" title="代理与 Mihomo 审批" width="wide" @close="$emit('close')">
    <div class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="flex overflow-x-auto rounded-lg bg-gray-100 p-1 dark:bg-dark-800" role="tablist">
          <button v-for="item in scopes" :key="item.value" type="button" class="min-h-11 shrink-0 rounded-md px-3 text-sm" :class="scope === item.value ? 'bg-white font-medium text-primary-600 shadow-sm dark:bg-dark-700' : 'text-gray-500'" @click="changeScope(item.value)">{{ item.label }}</button>
        </div>
        <button type="button" class="btn btn-secondary min-h-11" :disabled="loading" @click="load">刷新</button>
      </div>

      <div v-if="loading" class="py-10 text-center text-sm text-gray-500">加载中…</div>
      <div v-else-if="!approvals.length" class="py-10 text-center text-sm text-gray-500">暂无代理或 Mihomo 审批</div>
      <div v-else class="grid gap-3">
        <article v-for="approval in approvals" :key="approval.id" class="min-w-0 rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <h3 class="font-semibold text-gray-900 dark:text-white">{{ actionLabel(approval.action_type) }}</h3>
                <span :class="['badge', statusClass(approval.status)]">{{ statusLabel(approval.status) }}</span>
              </div>
              <p class="mt-1 truncate text-sm text-gray-600 dark:text-gray-300" :title="objectName(approval)">{{ objectName(approval) }}</p>
              <p class="mt-1 text-xs text-gray-500">申请人 {{ approval.requested_by_email }} · {{ formatTime(approval.requested_at) }}</p>
            </div>
            <div class="flex flex-wrap gap-2">
              <button v-if="scope === 'reviewable' && approval.status === 'pending'" type="button" class="btn btn-danger min-h-11 px-3" :disabled="busy === approval.id" @click="reject(approval)">驳回</button>
              <button v-if="scope === 'reviewable' && approval.status === 'pending'" type="button" class="btn btn-primary min-h-11 px-3" :disabled="busy === approval.id" @click="approve(approval)">通过</button>
              <button v-if="scope === 'mine' && isPrimaryAdmin && approval.status === 'pending'" type="button" class="btn btn-primary min-h-11 px-3" title="首位管理员可直接执行" :disabled="busy === approval.id" @click="approve(approval, true)">立即应用</button>
              <button v-if="scope === 'mine' && approval.status === 'approved' && revealable(approval)" type="button" class="btn btn-primary min-h-11 px-3" :disabled="busy === approval.id" @click="reveal(approval)">一次性领取</button>
            </div>
          </div>

          <div class="mt-3 rounded-lg bg-gray-50 p-3 dark:bg-dark-900/40">
            <p class="text-sm font-medium text-gray-800 dark:text-gray-100">{{ actionLabel(approval.action_type) }} · {{ objectName(approval) }}</p>
            <p class="mt-1 break-words text-xs text-gray-500">申请原因：{{ approval.reason }}</p>
          </div>

          <div v-if="approval.changes?.business?.groups?.length" class="mt-3 grid gap-2 lg:grid-cols-2">
            <section v-for="group in approval.changes.business.groups" :key="group.key" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
              <h4 class="text-xs font-semibold uppercase tracking-wide text-gray-500">{{ groupLabel(group.key) }}</h4>
              <dl class="mt-2 space-y-2">
                <div v-for="change in group.items" :key="change.key" class="text-sm">
                  <dt class="font-medium text-gray-800 dark:text-gray-100">{{ fieldLabel(change.key) }}</dt>
                  <dd class="mt-0.5 break-words text-gray-500">{{ changeValue(change) }}</dd>
                </div>
              </dl>
            </section>
          </div>

          <div v-if="approval.changes?.business?.impacts?.length" class="mt-3 flex flex-wrap gap-2">
            <span v-for="impact in approval.changes.business.impacts" :key="impact.key" class="badge badge-warning">{{ impactLabel(impact.key) }} {{ impact.count }}</span>
          </div>
        </article>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { adminAPI } from '@/api/admin'
import type { PoolApproval, PoolApprovalAction, PoolApprovalScope, PoolApprovalStatus } from '@/api/admin/sharedPool'
import type { RevealedProxy } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  close: []
  applied: [approval: PoolApproval]
  'proxy-revealed': [proxy: RevealedProxy]
  'export-revealed': [proxies: RevealedProxy[]]
}>()
const appStore = useAppStore()
const authStore = useAuthStore()
const isPrimaryAdmin = computed(() => authStore.user?.is_primary_admin === true)
const scope = ref<PoolApprovalScope>('reviewable')
const approvals = ref<PoolApproval[]>([])
const loading = ref(false)
const busy = ref<number | null>(null)
const scopes: Array<{ value: PoolApprovalScope; label: string }> = [
  { value: 'reviewable', label: '待我审核' },
  { value: 'mine', label: '我提交的' },
  { value: 'processed', label: '已处理' }
]

const load = async () => {
  if (!props.show) return
  loading.value = true
  try {
    const results = await Promise.all(['proxy', 'proxy_export', 'mihomo'].map(object_type =>
      adminAPI.sharedPool.listApprovals({ scope: scope.value, object_type: object_type as 'proxy' | 'proxy_export' | 'mihomo', page: 1, page_size: 50 })
    ))
    approvals.value = results.flatMap(result => result.items).sort((a, b) => b.requested_at.localeCompare(a.requested_at))
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || '加载审批失败')
  } finally { loading.value = false }
}
const changeScope = (value: PoolApprovalScope) => { scope.value = value; load() }
const approve = async (approval: PoolApproval, immediate = false) => {
  busy.value = approval.id
  try {
    const applied = await adminAPI.sharedPool.approveApproval(approval.id)
    appStore.showSuccess(immediate ? '变更已立即应用' : '审批已通过')
    emit('applied', applied)
    await load()
  }
  catch (error: any) { appStore.showError(error.response?.data?.detail || (immediate ? '立即应用失败' : '审批失败')) }
  finally { busy.value = null }
}
const reject = async (approval: PoolApproval) => {
  const reason = window.prompt('请输入驳回原因')?.trim()
  if (!reason) return
  busy.value = approval.id
  try { await adminAPI.sharedPool.rejectApproval(approval.id, reason); appStore.showSuccess('已驳回'); await load() }
  catch (error: any) { appStore.showError(error.response?.data?.detail || '驳回失败') }
  finally { busy.value = null }
}
const reveal = async (approval: PoolApproval) => {
  busy.value = approval.id
  try {
    if (approval.action_type === 'VIEW_PROXY_CREDENTIAL') {
      const result = await adminAPI.sharedPool.revealProxyApproval(approval.id)
      emit('proxy-revealed', result.proxy)
    } else {
      const result = await adminAPI.sharedPool.revealProxyExportApproval(approval.id)
      emit('export-revealed', result.proxies)
    }
    await load()
  } catch (error: any) { appStore.showError(error.response?.data?.detail || '领取失败') }
  finally { busy.value = null }
}
const revealable = (approval: PoolApproval) => ['VIEW_PROXY_CREDENTIAL', 'EXPORT_PROXY_CREDENTIALS'].includes(approval.action_type)
const objectName = (approval: PoolApproval) => approval.proxy_name || approval.changes?.business?.object?.name || approval.resource_key || `#${approval.id}`
const actionLabel = (action: PoolApprovalAction) => ({
  UPDATE_PROXY: '修改代理', VIEW_PROXY_CREDENTIAL: '查看代理连接信息', EXPORT_PROXY_CREDENTIALS: '导出代理连接信息', UPDATE_MIHOMO: '修改 Mihomo',
  UPDATE_ACCOUNT: '修改账号', VIEW_CREDENTIAL: '查看账号凭证', DELETE_ACCOUNT: '删除账号'
})[action]
const statusLabel = (status: PoolApprovalStatus) => ({ pending: '待审核', approved: '已通过', rejected: '已驳回', expired: '已过期', consumed: '已领取' })[status]
const statusClass = (status: PoolApprovalStatus) => ({ pending: 'badge-warning', approved: 'badge-success', rejected: 'badge-danger', expired: 'badge-gray', consumed: 'badge-gray' })[status]
const formatTime = (value: string) => new Date(value).toLocaleString()
const groupLabel = (key: string) => ({
  connection: '连接路线', credentials: '认证信息', runtime: '状态与回退', mihomo: 'Mihomo 策略'
})[key] || key
const fieldLabel = (key: string) => ({
  name: '代理名称', protocol: '代理协议', proxy_endpoint: '服务器与端口', proxy_credentials: '连接认证',
  username: '用户名', password: '密码', status: '可用状态', expires_at: '有效期', fallback_mode: '失败回退',
  backup_proxy_id: '备用代理', expiry_warn_days: '到期预警', proxy_export: '导出范围',
  subscription: '订阅来源', mode: '运行模式', refresh: '刷新节点'
})[key] || key
const impactLabel = (key: string) => ({ bound_accounts: '绑定账号', proxies: '代理数量', nodes: '节点' })[key] || key
const changeValue = (change: { key?: string; before?: unknown; after?: unknown; sensitive?: boolean }) => {
  if (change.key === 'proxy_endpoint' && change.sensitive) return '通过后可一次性查看服务器与端口'
  if (change.key === 'proxy_credentials') return '通过后可一次性查看用户名与密码'
  if (change.key === 'proxy_export') return `通过后可一次性导出 ${String(change.after ?? 0)} 个代理`
  if (change.key === 'subscription') return '更新订阅来源，完整地址不在审批详情中展示'
  if (change.key === 'refresh') return '重新拉取订阅节点并同步托管代理'
  if (change.key === 'password') return '密码将更新，具体值不展示'
  if (change.key === 'username') return `${change.before ? '已配置' : '未配置'} → ${change.after ? '已配置' : '未配置'}`
  if (change.key === 'status') return `${change.before === 'active' ? '正常' : '停用'} → ${change.after === 'active' ? '正常' : '停用'}`
  if (change.key === 'fallback_mode') {
    const label = (value: unknown) => ({ none: '不回退', proxy: '切换备用代理', direct: '直连' })[String(value)] || String(value ?? '-')
    return `${label(change.before)} → ${label(change.after)}`
  }
  if (change.key === 'mode') {
    const label = ({ AUTO: '最低延迟', FALLBACK: '容灾优先', DYNAMIC: '动态轮换', REJECT: '关闭 / 未选择' })[String(change.after)] || String(change.after ?? '-')
    return `切换为${label}`
  }
  return change.sensitive ? '敏感值已更新' : `${String(change.before ?? '-')} → ${String(change.after ?? '-')}`
}

watch(() => props.show, open => { if (open) load() })
onMounted(load)
</script>
