<template>
  <section v-if="status?.enabled" class="min-w-0 rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="font-semibold text-gray-900 dark:text-white">Mihomo 节点池</h2>
          <span :class="['badge', status.subscription_configured ? 'badge-success' : 'badge-danger']">
            {{ status.subscription_configured ? '订阅已配置' : '订阅未配置' }}
          </span>
          <span class="badge badge-gray">{{ status.alive_count }}/{{ status.node_count }} 可用</span>
        </div>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          自动分流、定向节点和动态轮换共用同一订阅；{{ changePolicyHint }}
        </p>
        <dl class="mt-3 grid min-w-0 grid-cols-2 gap-x-4 gap-y-2 text-xs xl:grid-cols-4">
          <div class="min-w-0">
            <dt class="text-gray-500 dark:text-gray-400">订阅来源</dt>
            <dd class="mt-0.5 truncate font-medium text-gray-800 dark:text-gray-100" :title="status.subscription_host || '-'">{{ status.subscription_host || '-' }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-gray-500 dark:text-gray-400">Mihomo 版本</dt>
            <dd class="mt-0.5 truncate font-medium text-gray-800 dark:text-gray-100" :title="status.version || '-'">{{ status.version || '-' }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-gray-500 dark:text-gray-400">最近刷新</dt>
            <dd class="mt-0.5 truncate font-medium text-gray-800 dark:text-gray-100" :title="status.updated_at ? formatTime(status.updated_at) : '-'">{{ status.updated_at ? formatTime(status.updated_at) : '-' }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-gray-500 dark:text-gray-400">托管代理 ID</dt>
            <dd class="mt-0.5 truncate font-medium text-gray-800 dark:text-gray-100" :title="proxyIDSummary">{{ proxyIDSummary }}</dd>
          </div>
        </dl>
      </div>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-secondary min-h-11" @click="openAction('refresh')">
          <Icon name="refresh" size="sm" class="mr-1.5" />刷新订阅
        </button>
        <button type="button" class="btn btn-primary min-h-11" @click="openAction('subscription')">更新订阅</button>
      </div>
    </div>

    <div class="mt-4 grid min-w-0 gap-3 xl:grid-cols-3">
      <article class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">自动</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">容灾优先会先保证可用；最低延迟会选择当前响应最快的节点。</p>
        <p class="mt-3 text-xs text-gray-500">当前：{{ automaticLabel }}</p>
        <div class="mt-2 grid grid-cols-2 gap-2">
          <button type="button" class="btn min-h-11 px-2" :class="modeClass('automatic', 'FALLBACK')" @click="openMode('automatic', 'FALLBACK')">容灾优先</button>
          <button type="button" class="btn min-h-11 px-2" :class="modeClass('automatic', 'AUTO')" @click="openMode('automatic', 'AUTO')">最低延迟</button>
        </div>
      </article>

      <article class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">定向</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">将此出口固定到指定订阅节点；未选择时拒绝流量，避免误走其他线路。</p>
        <p class="mt-3 truncate text-xs text-gray-500" :title="directionalLabel">当前：{{ directionalLabel }}</p>
        <div class="mt-2 flex min-w-0 gap-2">
          <input v-model.trim="directionalSelection" list="mihomo-directional-nodes" class="input min-w-0 flex-1" placeholder="搜索或选择节点" />
          <datalist id="mihomo-directional-nodes">
            <option value="REJECT">未选择</option>
            <option v-for="node in status.nodes" :key="node.name" :value="node.name">{{ node.alive ? '可用' : '异常' }} · {{ node.delay || '-' }}ms</option>
          </datalist>
          <button type="button" class="btn btn-secondary min-h-11 shrink-0 px-3" :disabled="!directionalSelection" @click="openMode('directional', directionalSelection)">{{ changeActionLabel }}</button>
        </div>
      </article>

      <article class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">动态</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">动态轮换按 Mihomo 策略切换节点；关闭后此出口直接拒绝流量。</p>
        <p class="mt-3 text-xs text-gray-500">当前：{{ dynamicLabel }}</p>
        <div class="mt-2 grid grid-cols-2 gap-2">
          <button type="button" class="btn min-h-11 px-2" :class="modeClass('dynamic', 'DYNAMIC')" @click="openMode('dynamic', 'DYNAMIC')">动态轮换</button>
          <button type="button" class="btn min-h-11 px-2" :class="modeClass('dynamic', 'REJECT')" @click="openMode('dynamic', 'REJECT')">关闭</button>
        </div>
      </article>
    </div>

    <div class="mt-4 min-w-0 border-t border-gray-200 pt-4 dark:border-dark-700">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">订阅节点</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">查看订阅返回的节点状态和最近探测延迟。</p>
        </div>
        <button type="button" class="btn btn-secondary min-h-11" :aria-expanded="nodesExpanded" @click="nodesExpanded = !nodesExpanded">
          {{ nodesExpanded ? '收起节点' : `查看节点（${status.node_count}）` }}
        </button>
      </div>

      <div v-if="nodesExpanded" class="mt-3 min-w-0 space-y-3">
        <div class="grid min-w-0 gap-2 md:grid-cols-[minmax(0,1fr)_auto]">
          <input v-model.trim="nodeQuery" type="search" class="input min-w-0" placeholder="搜索节点名称" aria-label="搜索 Mihomo 节点" />
          <div class="grid grid-cols-3 gap-1 rounded-lg bg-gray-100 p-1 dark:bg-dark-900/60" role="group" aria-label="节点状态筛选">
            <button v-for="item in nodeFilters" :key="item.value" type="button" class="min-h-11 rounded-md px-3 text-sm" :class="nodeFilter === item.value ? 'bg-white font-medium text-primary-600 shadow-sm dark:bg-dark-700' : 'text-gray-500 dark:text-gray-400'" @click="nodeFilter = item.value">
              {{ item.label }}
            </button>
          </div>
        </div>

        <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
          <div v-if="filteredNodes.length" class="max-h-72 divide-y divide-gray-100 overflow-y-auto dark:divide-dark-700">
            <div v-for="node in filteredNodes" :key="node.name" class="flex min-w-0 items-center gap-3 px-3 py-2.5 text-sm">
              <span class="min-w-0 flex-1 truncate font-medium text-gray-800 dark:text-gray-100" :title="node.name">{{ node.name }}</span>
              <span :class="['badge shrink-0 whitespace-nowrap', node.alive ? 'badge-success' : 'badge-danger']">{{ node.alive ? '可用' : '异常' }}</span>
              <span class="w-14 shrink-0 text-right text-xs tabular-nums text-gray-500 dark:text-gray-400">{{ formatDelay(node.delay) }}</span>
            </div>
          </div>
          <p v-else class="px-3 py-8 text-center text-sm text-gray-500 dark:text-gray-400">没有符合条件的节点</p>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400">显示 {{ filteredNodes.length }} / {{ status.nodes.length }} 个节点</p>
      </div>
    </div>
  </section>

  <BaseDialog :show="dialogOpen" :title="dialogTitle" width="normal" @close="closeDialog">
    <form id="mihomo-approval-form" class="space-y-4" @submit.prevent="submit">
      <div v-if="action === 'subscription'">
        <label class="input-label" for="mihomo-subscription-url">订阅地址</label>
        <input id="mihomo-subscription-url" v-model.trim="subscriptionUrl" type="url" required autocomplete="off" class="input" placeholder="https://…" />
        <p class="input-hint mt-1">{{ subscriptionSecurityHint }}</p>
      </div>
      <div v-else-if="action === 'mode'" class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-700 dark:bg-dark-900/40">
        {{ modeName }}：{{ selectionLabel }}
      </div>
      <div>
        <label class="input-label" for="mihomo-approval-reason">变更原因</label>
        <textarea id="mihomo-approval-reason" v-model.trim="reason" required rows="3" class="input" :placeholder="reasonPlaceholder"></textarea>
        <p class="input-hint mt-1">{{ dialogPolicyHint }}</p>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary min-h-11" @click="closeDialog">取消</button>
        <button type="submit" form="mihomo-approval-form" class="btn btn-primary min-h-11" :disabled="submitting || !reason">{{ submitting ? '处理中…' : changeActionLabel }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { MihomoNode, MihomoStatus } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const emit = defineEmits<{ 'approval-submitted': [] }>()
const appStore = useAppStore()
const authStore = useAuthStore()
const status = ref<MihomoStatus | null>(null)
const action = ref<'subscription' | 'refresh' | 'mode' | null>(null)
const mode = ref('')
const selection = ref('')
const directionalSelection = ref('')
const subscriptionUrl = ref('')
const reason = ref('')
const submitting = ref(false)
const nodesExpanded = ref(false)
const nodeQuery = ref('')
const nodeFilter = ref<'all' | 'alive' | 'down'>('all')
const nodeFilters = [
  { value: 'all' as const, label: '全部' },
  { value: 'alive' as const, label: '可用' },
  { value: 'down' as const, label: '异常' }
]
const isPrimaryAdmin = computed(() => authStore.user?.is_primary_admin === true)
const dialogOpen = computed(() => action.value !== null)
const changeActionLabel = computed(() => isPrimaryAdmin.value ? '直接应用' : '提交审核')
const changePolicyHint = computed(() => isPrimaryAdmin.value ? '首位管理员的变更会直接应用。' : '变更需由另一位管理员审核。')
const dialogPolicyHint = computed(() => isPrimaryAdmin.value ? '提交后直接应用，变更原因会写入审计记录。' : '提交后由另一位管理员审核，变更原因会进入审批与审计记录。')
const subscriptionSecurityHint = '地址始终加密保存，完整值不在页面、审批详情或审计日志中显示。'
const reasonPlaceholder = '说明本次变更用途和影响范围'
const currentSelection = (key: 'automatic' | 'directional' | 'dynamic') => status.value?.modes?.[key]?.selection || 'REJECT'
const automaticLabel = computed(() => currentSelection('automatic') === 'AUTO' ? '最低延迟' : '容灾优先')
const directionalLabel = computed(() => currentSelection('directional') === 'REJECT' ? '未选择' : currentSelection('directional'))
const dynamicLabel = computed(() => currentSelection('dynamic') === 'DYNAMIC' ? '动态轮换' : '关闭')
const dialogTitle = computed(() => action.value === 'subscription' ? '更新 Mihomo 订阅' : action.value === 'refresh' ? '刷新 Mihomo 订阅' : '修改 Mihomo 模式')
const modeName = computed(() => ({ automatic: '自动', directional: '定向', dynamic: '动态' })[mode.value] || mode.value)
const selectionLabel = computed(() => {
  if (selection.value === 'FALLBACK') return '容灾优先'
  if (selection.value === 'AUTO') return '最低延迟'
  if (selection.value === 'DYNAMIC') return '动态轮换'
  if (selection.value === 'REJECT') return '关闭 / 未选择'
  return selection.value
})
const proxyIDSummary = computed(() => {
  if (!status.value) return '-'
  const labels = { automatic: '自动', directional: '定向', dynamic: '动态' } as const
  const items = (Object.keys(labels) as Array<keyof typeof labels>)
    .filter(key => status.value?.proxy_ids?.[key])
    .map(key => `${labels[key]} #${status.value?.proxy_ids?.[key]}`)
  return items.join(' · ') || '-'
})
const filteredNodes = computed(() => {
  const query = nodeQuery.value.toLocaleLowerCase()
  return (status.value?.nodes || []).filter(node => {
    if (nodeFilter.value === 'alive' && !node.alive) return false
    if (nodeFilter.value === 'down' && node.alive) return false
    return !query || node.name.toLocaleLowerCase().includes(query)
  })
})

const load = async () => {
  try { status.value = await adminAPI.mihomo.getStatus() } catch { status.value = null }
}
const formatTime = (value: string) => new Date(value).toLocaleString()
const formatDelay = (delay: MihomoNode['delay']) => typeof delay === 'number' && delay > 0 ? `${delay}ms` : '-'
const modeClass = (key: 'automatic' | 'dynamic', value: string) => currentSelection(key) === value ? 'btn-primary' : 'btn-secondary'
const openAction = (next: 'subscription' | 'refresh') => { action.value = next }
const openMode = (nextMode: string, nextSelection: string) => {
  mode.value = nextMode
  selection.value = nextSelection
  action.value = 'mode'
}
const closeDialog = () => {
  action.value = null
  reason.value = ''
  subscriptionUrl.value = ''
}
const submit = async () => {
  if (!action.value || !reason.value) return
  submitting.value = true
  try {
    const result = action.value === 'subscription'
      ? await adminAPI.mihomo.updateSubscription(subscriptionUrl.value, reason.value)
      : action.value === 'refresh'
        ? await adminAPI.mihomo.refresh(reason.value)
        : await adminAPI.mihomo.updateMode(mode.value, selection.value, reason.value)
    closeDialog()
    if (result.approval_required) {
      appStore.showSuccess('已提交给其他管理员审核')
      emit('approval-submitted')
    } else {
      appStore.showSuccess('Mihomo 变更已直接应用')
      await load()
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || '处理变更失败')
  } finally { submitting.value = false }
}

onMounted(load)
</script>
